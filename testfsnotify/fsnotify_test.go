package testfsnotify

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TestFsnotifyDockerBindMount proves that fsnotify receives Write events
// when a Docker container appends to a file in a bind-mounted directory.
func TestFsnotifyDockerBindMount(t *testing.T) {
	// Create a temp dir for the bind mount.
	dataDir, err := os.MkdirTemp("", "fsnotify-poc-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	logFile := filepath.Join(dataDir, "events.log")

	// Create the file before starting the watcher.
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	f.Close()
	t.Logf("Log file: %s", logFile)

	// Start fsnotify watcher on the file.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(logFile); err != nil {
		t.Fatalf("watch file: %v", err)
	}
	t.Log("Watcher started on file")

	// Count events in a goroutine.
	var writeEvents int64
	var otherEvents int64
	var watcherErrors int64
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					atomic.AddInt64(&writeEvents, 1)
				} else {
					atomic.AddInt64(&otherEvents, 1)
					t.Logf("Non-write event: %s", event)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				atomic.AddInt64(&watcherErrors, 1)
				t.Logf("Watcher error: %v", err)
			}
		}
	}()

	// Start Docker container that writes to the bind-mounted file.
	composeDir, _ := os.Getwd()
	cmd := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"up", "--remove-orphans", "--force-recreate")
	cmd.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	t.Logf("Starting container with DATA_DIR=%s", dataDir)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start compose: %v", err)
	}

	// Wait for container to finish (it writes 50 lines then exits).
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	select {
	case err := <-cmdDone:
		if err != nil {
			t.Logf("compose exited with: %v", err)
		} else {
			t.Log("compose exited cleanly")
		}
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		t.Fatal("compose timed out after 30s")
	}

	// Give a moment for any final events to arrive.
	time.Sleep(500 * time.Millisecond)
	watcher.Close()
	<-done

	writes := atomic.LoadInt64(&writeEvents)
	others := atomic.LoadInt64(&otherEvents)
	errors := atomic.LoadInt64(&watcherErrors)

	t.Logf("Results: write_events=%d other_events=%d errors=%d", writes, others, errors)

	// Verify the file has content.
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	t.Logf("File has %d lines, %d bytes", len(lines), len(data))

	if writes == 0 {
		t.Errorf("FAIL: fsnotify received 0 Write events despite %d lines in file", len(lines))
		t.Log("This would mean inotify does NOT work across Docker bind-mounts on this host")
	} else {
		t.Logf("SUCCESS: fsnotify received %d Write events for %d lines written by container", writes, len(lines))
	}

	// Cleanup compose.
	cleanup := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"down", "--remove-orphans")
	cleanup.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cleanup.Run()
}

// TestFsnotifyWatchDirectory tests watching the directory instead of the file.
// This is an alternative approach if file-level watching doesn't work.
func TestFsnotifyWatchDirectory(t *testing.T) {
	dataDir, err := os.MkdirTemp("", "fsnotify-dir-poc-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	logFile := filepath.Join(dataDir, "events.log")
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	f.Close()

	// Watch the DIRECTORY instead of the file.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(dataDir); err != nil {
		t.Fatalf("watch directory: %v", err)
	}
	t.Logf("Watcher started on directory: %s", dataDir)

	var writeEvents int64
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					atomic.AddInt64(&writeEvents, 1)
				}
				t.Logf("Dir event: %s %s", event.Op, event.Name)
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	composeDir, _ := os.Getwd()
	cmd := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"up", "--remove-orphans", "--force-recreate")
	cmd.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start compose: %v", err)
	}

	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	select {
	case err := <-cmdDone:
		if err != nil {
			t.Logf("compose exited with: %v", err)
		}
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		t.Fatal("compose timed out")
	}

	time.Sleep(500 * time.Millisecond)
	watcher.Close()
	<-done

	writes := atomic.LoadInt64(&writeEvents)
	t.Logf("Directory watcher: write_events=%d", writes)

	if writes == 0 {
		t.Error("FAIL: directory watcher received 0 Write events")
	} else {
		t.Logf("SUCCESS: directory watcher received %d Write events", writes)
	}

	cleanup := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"down", "--remove-orphans")
	cleanup.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cleanup.Run()
}

// TestFsnotifyTruncateAndResume tests that fsnotify continues to deliver
// events after the file is truncated on the host (simulating BDD scenario clear).
func TestFsnotifyTruncateAndResume(t *testing.T) {
	dataDir, err := os.MkdirTemp("", "fsnotify-trunc-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	logFile := filepath.Join(dataDir, "events.log")
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	f.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(logFile); err != nil {
		t.Fatalf("watch file: %v", err)
	}

	var mu sync.Mutex
	var phases []string // "pre" or "post" for each Write event
	var currentPhase string = "pre"
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					mu.Lock()
					phases = append(phases, currentPhase)
					mu.Unlock()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	// Use a long-running container that writes continuously.
	// We'll truncate in the middle.
	composeDir, _ := os.Getwd()
	cmd := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"up", "--remove-orphans", "--force-recreate")
	cmd.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start compose: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cleanup := exec.Command("docker", "compose", "-f",
			filepath.Join(composeDir, "docker-compose.yml"),
			"down", "--remove-orphans")
		cleanup.Env = append(os.Environ(), "DATA_DIR="+dataDir)
		cleanup.Run()
	}()

	// Wait for some writes.
	time.Sleep(3 * time.Second)

	mu.Lock()
	preCount := len(phases)
	mu.Unlock()
	t.Logf("Pre-truncate: %d write events", preCount)

	// Truncate the file on the host (simulating BDD scenario clear).
	t.Log("Truncating file...")
	if err := os.Truncate(logFile, 0); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	mu.Lock()
	currentPhase = "post"
	mu.Unlock()

	// Wait for more writes after truncation.
	time.Sleep(3 * time.Second)

	watcher.Close()
	<-done

	mu.Lock()
	var preEvents, postEvents int
	for _, p := range phases {
		if p == "pre" {
			preEvents++
		} else {
			postEvents++
		}
	}
	mu.Unlock()

	t.Logf("Pre-truncate events: %d, Post-truncate events: %d", preEvents, postEvents)

	if preEvents == 0 {
		t.Error("FAIL: no write events before truncation")
	}
	if postEvents == 0 {
		t.Error("FAIL: no write events after truncation — watch may be broken by truncate")
	} else {
		t.Logf("SUCCESS: fsnotify continues after truncation (%d + %d events)", preEvents, postEvents)
	}
}

// TestFsnotifyIncrementalRead simulates the full AuditWatcher pattern:
// fsnotify + incremental tail-read + JSON parsing.
func TestFsnotifyIncrementalRead(t *testing.T) {
	dataDir, err := os.MkdirTemp("", "fsnotify-read-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	logFile := filepath.Join(dataDir, "events.log")
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	f.Close()

	// Set up fsnotify.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(logFile); err != nil {
		t.Fatalf("watch file: %v", err)
	}

	// Incremental reader state.
	var mu sync.Mutex
	var events []map[string]interface{}
	var offset int64
	var partial string
	newData := make(chan struct{}, 1)

	readNewData := func() {
		mu.Lock()
		defer mu.Unlock()

		f, err := os.Open(logFile)
		if err != nil {
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return
		}
		if info.Size() < offset {
			offset = 0
			partial = ""
		}

		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return
		}

		data, err := io.ReadAll(f)
		if err != nil || len(data) == 0 {
			return
		}

		offset += int64(len(data))
		content := partial + string(data)
		partial = ""

		lines := strings.Split(content, "\n")
		if !strings.HasSuffix(content, "\n") {
			partial = lines[len(lines)-1]
			lines = lines[:len(lines)-1]
		}

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "DONE" {
				continue
			}
			var event map[string]interface{}
			if json.Unmarshal([]byte(line), &event) == nil {
				events = append(events, event)
			}
		}
	}

	// Background goroutine: fsnotify → readNewData → signal.
	var notifyCount int64
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					atomic.AddInt64(&notifyCount, 1)
					readNewData()
					select {
					case newData <- struct{}{}:
					default:
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	// Start container.
	composeDir, _ := os.Getwd()
	cmd := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"up", "--remove-orphans", "--force-recreate")
	cmd.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start compose: %v", err)
	}

	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	// Wait for specific events using channel (simulating WaitForMatch).
	waitForSeq := func(seq int, timeout time.Duration) bool {
		deadline := time.After(timeout)
		// Check existing events first.
		mu.Lock()
		for _, e := range events {
			if s, ok := e["seq"].(float64); ok && int(s) == seq {
				mu.Unlock()
				return true
			}
		}
		mu.Unlock()

		for {
			select {
			case <-deadline:
				// Final read.
				readNewData()
				mu.Lock()
				defer mu.Unlock()
				for _, e := range events {
					if s, ok := e["seq"].(float64); ok && int(s) == seq {
						return true
					}
				}
				return false
			case <-newData:
				mu.Lock()
				for _, e := range events {
					if s, ok := e["seq"].(float64); ok && int(s) == seq {
						mu.Unlock()
						return true
					}
				}
				mu.Unlock()
			case <-time.After(200 * time.Millisecond):
				// Safety-net poll.
				readNewData()
				mu.Lock()
				for _, e := range events {
					if s, ok := e["seq"].(float64); ok && int(s) == seq {
						mu.Unlock()
						return true
					}
				}
				mu.Unlock()
			}
		}
	}

	// Wait for seq 5, 25, 49 with 10s timeout.
	for _, seq := range []int{5, 25, 49} {
		start := time.Now()
		found := waitForSeq(seq, 15*time.Second)
		elapsed := time.Since(start)
		if found {
			t.Logf("Found seq=%d in %dms", seq, elapsed.Milliseconds())
		} else {
			t.Errorf("FAIL: seq=%d not found within timeout", seq)
		}
	}

	select {
	case <-cmdDone:
	case <-time.After(15 * time.Second):
		cmd.Process.Kill()
	}

	time.Sleep(200 * time.Millisecond)
	watcher.Close()
	<-watchDone

	notify := atomic.LoadInt64(&notifyCount)
	mu.Lock()
	eventCount := len(events)
	mu.Unlock()

	t.Logf("Final: notify_events=%d parsed_events=%d", notify, eventCount)

	if notify == 0 {
		t.Error("FAIL: zero fsnotify notifications — same problem as BDD watcher")
	}
	if eventCount < 50 {
		t.Errorf("Expected 50 parsed events, got %d", eventCount)
	} else {
		t.Logf("SUCCESS: parsed all %d events via fsnotify channel pattern", eventCount)
	}

	// Unique seq values.
	mu.Lock()
	seqs := make(map[int]bool)
	for _, e := range events {
		if s, ok := e["seq"].(float64); ok {
			seqs[int(s)] = true
		}
	}
	mu.Unlock()
	t.Logf("Unique sequences: %d (expected 50)", len(seqs))

	// Print timing distribution.
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("fsnotify Write events: %d\n", notify)
	fmt.Printf("Parsed JSON events: %d\n", eventCount)
	fmt.Printf("Unique sequences: %d\n", len(seqs))

	cleanup := exec.Command("docker", "compose", "-f",
		filepath.Join(composeDir, "docker-compose.yml"),
		"down", "--remove-orphans")
	cleanup.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	cleanup.Run()
}
