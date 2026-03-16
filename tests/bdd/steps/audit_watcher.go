//go:build bdd

// Package steps provides godog step definitions for BDD tests.
package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// AuditWatcher watches an audit log file on the host filesystem (bind-mounted from
// a Docker container) and incrementally parses new JSON lines into structured events.
// Assertion steps wait on a channel instead of polling via docker exec.
type AuditWatcher struct {
	mu       sync.Mutex
	events   []map[string]interface{}
	filePath string
	offset   int64
	watcher  *fsnotify.Watcher
	newData  chan struct{} // signaled when new data arrives
	done     chan struct{}
	stopOnce sync.Once
	partial  string          // buffer for incomplete last line
	rawLog   strings.Builder // accumulates raw data for LogString()
}

// NewAuditWatcher creates a new AuditWatcher that watches the given file path.
// The file is created if it does not exist.
func NewAuditWatcher(path string) (*AuditWatcher, error) {
	// Create the file if it doesn't exist.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("create audit log file: %w", err)
	}
	f.Close()

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	if err := w.Add(path); err != nil {
		w.Close()
		return nil, fmt.Errorf("watch %s: %w", path, err)
	}

	aw := &AuditWatcher{
		filePath: path,
		watcher:  w,
		newData:  make(chan struct{}, 1),
		done:     make(chan struct{}),
	}

	go aw.watch()
	return aw, nil
}

// watch is the background goroutine that listens for fsnotify events.
func (aw *AuditWatcher) watch() {
	defer close(aw.done)
	for {
		select {
		case event, ok := <-aw.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				aw.ReadNewData()
				// Signal waiters without blocking.
				select {
				case aw.newData <- struct{}{}:
				default:
				}
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// File was rotated/removed — reset offset and re-add watch.
				aw.mu.Lock()
				aw.offset = 0
				aw.partial = ""
				aw.mu.Unlock()
				// Re-add the watch (the file may be recreated).
				_ = aw.watcher.Add(aw.filePath)
			}
		case _, ok := <-aw.watcher.Errors:
			if !ok {
				return
			}
			// Ignore watcher errors — they don't affect functionality.
		}
	}
}

// ReadNewData reads new data from the audit log file starting at the current offset.
// It parses complete JSON lines and appends them to the events slice.
func (aw *AuditWatcher) ReadNewData() {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	f, err := os.Open(aw.filePath)
	if err != nil {
		return
	}
	defer f.Close()

	// Check file size — if smaller than offset, file was truncated.
	info, err := f.Stat()
	if err != nil {
		return
	}
	if info.Size() < aw.offset {
		aw.offset = 0
		aw.partial = ""
	}

	if _, err := f.Seek(aw.offset, io.SeekStart); err != nil {
		return
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return
	}
	if len(data) == 0 {
		return
	}

	aw.offset += int64(len(data))

	// Accumulate raw data for LogString().
	aw.rawLog.Write(data)

	// Prepend any partial line from previous read.
	content := aw.partial + string(data)
	aw.partial = ""

	lines := strings.Split(content, "\n")
	// If the content doesn't end with a newline, the last element is a partial line.
	if !strings.HasSuffix(content, "\n") {
		aw.partial = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip null bytes from sparse file truncation artifacts.
		if idx := strings.IndexByte(line, '{'); idx > 0 {
			line = line[idx:]
		}
		var event map[string]interface{}
		if json.Unmarshal([]byte(line), &event) == nil {
			aw.events = append(aw.events, event)
		}
	}
}

// WaitForMatch waits for an audit event that matches the given function.
// Only scans newly arrived events on each notification (tracks checkedIdx).
// Returns whether a match was found, the best partial match, and the count of matched fields.
func (aw *AuditWatcher) WaitForMatch(ctx context.Context, matchFn func([]map[string]interface{}) (bool, map[string]interface{}, int)) (bool, map[string]interface{}, int) {
	// Do an initial read in case data arrived before the watcher was set up.
	aw.ReadNewData()

	var bestMatch map[string]interface{}
	var bestCount int
	checkedIdx := 0

	checkNew := func() bool {
		aw.mu.Lock()
		defer aw.mu.Unlock()
		if len(aw.events) <= checkedIdx {
			return false
		}
		found, bm, bc := matchFn(aw.events[checkedIdx:])
		checkedIdx = len(aw.events)
		if bm != nil && bc > bestCount {
			bestMatch = bm
			bestCount = bc
		}
		return found
	}

	if checkNew() {
		return true, nil, 0
	}

	for {
		select {
		case <-ctx.Done():
			// Final read before giving up.
			aw.ReadNewData()
			if checkNew() {
				return true, nil, 0
			}
			return false, bestMatch, bestCount
		case <-aw.newData:
			if checkNew() {
				return true, nil, 0
			}
		case <-time.After(200 * time.Millisecond):
			// Periodic poll as safety net in case fsnotify misses an event.
			aw.ReadNewData()
			if checkNew() {
				return true, nil, 0
			}
		}
	}
}

// Events returns a copy of all parsed audit events.
func (aw *AuditWatcher) Events() []map[string]interface{} {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	cp := make([]map[string]interface{}, len(aw.events))
	copy(cp, aw.events)
	return cp
}

// Clear resets the events slice and offset. Call this between scenarios
// after the audit log file has been truncated on the host.
func (aw *AuditWatcher) Clear() {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	aw.events = nil
	aw.offset = 0
	aw.partial = ""
	aw.rawLog.Reset()
}

// Close stops the fsnotify watcher and background goroutine.
func (aw *AuditWatcher) Close() {
	aw.stopOnce.Do(func() {
		aw.watcher.Close()
	})
	<-aw.done
}

// LogString returns the raw audit log content accumulated since the last Clear().
// No disk I/O — data is buffered in memory during incremental reads.
func (aw *AuditWatcher) LogString() (string, error) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.rawLog.String(), nil
}
