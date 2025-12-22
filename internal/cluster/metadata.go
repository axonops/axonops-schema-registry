// Package cluster provides cluster metadata and topology information.
package cluster

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Version information - set at build time
var (
	Version   = "1.0.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Metadata holds cluster metadata information.
type Metadata struct {
	ClusterID string    `json:"cluster_id"`
	Version   string    `json:"version"`
	GitCommit string    `json:"commit,omitempty"`
	BuildTime string    `json:"build_time,omitempty"`
	GoVersion string    `json:"go_version"`
	StartTime time.Time `json:"start_time"`
	NodeID    string    `json:"node_id"`
	Hostname  string    `json:"hostname"`
}

// Node represents a node in the cluster.
type Node struct {
	ID          string    `json:"id"`
	Hostname    string    `json:"hostname"`
	Address     string    `json:"address"`
	Port        int       `json:"port"`
	Status      string    `json:"status"`
	Role        string    `json:"role"`
	StartTime   time.Time `json:"start_time"`
	LastSeen    time.Time `json:"last_seen"`
	Version     string    `json:"version"`
	SchemaCount int       `json:"schema_count,omitempty"`
}

// NodeStatus represents the status of a node.
type NodeStatus string

const (
	NodeStatusHealthy   NodeStatus = "healthy"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusStarting  NodeStatus = "starting"
	NodeStatusStopping  NodeStatus = "stopping"
)

// NodeRole represents the role of a node.
type NodeRole string

const (
	NodeRoleLeader   NodeRole = "leader"
	NodeRoleFollower NodeRole = "follower"
	NodeRoleStandby  NodeRole = "standby"
)

// ClusterInfo provides cluster topology information.
type ClusterInfo struct {
	mu       sync.RWMutex
	metadata *Metadata
	nodes    map[string]*Node
	self     *Node
}

// NewClusterInfo creates a new cluster info instance.
func NewClusterInfo(address string, port int) *ClusterInfo {
	hostname, _ := os.Hostname()
	nodeID := uuid.New().String()
	clusterID := uuid.New().String()

	metadata := &Metadata{
		ClusterID: clusterID,
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		StartTime: time.Now(),
		NodeID:    nodeID,
		Hostname:  hostname,
	}

	self := &Node{
		ID:        nodeID,
		Hostname:  hostname,
		Address:   address,
		Port:      port,
		Status:    string(NodeStatusHealthy),
		Role:      string(NodeRoleLeader), // Single node is always leader
		StartTime: time.Now(),
		LastSeen:  time.Now(),
		Version:   Version,
	}

	ci := &ClusterInfo{
		metadata: metadata,
		nodes:    make(map[string]*Node),
		self:     self,
	}
	ci.nodes[nodeID] = self

	return ci
}

// GetMetadata returns cluster metadata.
func (ci *ClusterInfo) GetMetadata() *Metadata {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.metadata
}

// GetClusterID returns the cluster ID.
func (ci *ClusterInfo) GetClusterID() string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.metadata.ClusterID
}

// SetClusterID sets the cluster ID (for cluster join scenarios).
func (ci *ClusterInfo) SetClusterID(id string) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.metadata.ClusterID = id
}

// GetVersion returns the version information.
func (ci *ClusterInfo) GetVersion() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_time": BuildTime,
		"go_version": runtime.Version(),
	}
}

// GetSelf returns this node's information.
func (ci *ClusterInfo) GetSelf() *Node {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.self
}

// GetNodes returns all nodes in the cluster.
func (ci *ClusterInfo) GetNodes() []*Node {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	nodes := make([]*Node, 0, len(ci.nodes))
	for _, node := range ci.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNode returns a specific node by ID.
func (ci *ClusterInfo) GetNode(id string) (*Node, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	node, ok := ci.nodes[id]
	return node, ok
}

// AddNode adds a node to the cluster.
func (ci *ClusterInfo) AddNode(node *Node) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.nodes[node.ID] = node
}

// RemoveNode removes a node from the cluster.
func (ci *ClusterInfo) RemoveNode(id string) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	delete(ci.nodes, id)
}

// UpdateNodeStatus updates a node's status.
func (ci *ClusterInfo) UpdateNodeStatus(id string, status NodeStatus) {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	if node, ok := ci.nodes[id]; ok {
		node.Status = string(status)
		node.LastSeen = time.Now()
	}
}

// UpdateSelfStatus updates this node's status.
func (ci *ClusterInfo) UpdateSelfStatus(status NodeStatus) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.self.Status = string(status)
	ci.self.LastSeen = time.Now()
}

// IsHealthy returns whether this node is healthy.
func (ci *ClusterInfo) IsHealthy() bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.self.Status == string(NodeStatusHealthy)
}

// GetLeader returns the current leader node.
func (ci *ClusterInfo) GetLeader() *Node {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	for _, node := range ci.nodes {
		if node.Role == string(NodeRoleLeader) {
			return node
		}
	}
	return nil
}

// IsLeader returns whether this node is the leader.
func (ci *ClusterInfo) IsLeader() bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.self.Role == string(NodeRoleLeader)
}

// SetSchemaCount updates the schema count for this node.
func (ci *ClusterInfo) SetSchemaCount(count int) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.self.SchemaCount = count
}

// HealthStatus returns a summary of cluster health.
type HealthStatus struct {
	Status       string          `json:"status"`
	NodeCount    int             `json:"node_count"`
	HealthyNodes int             `json:"healthy_nodes"`
	Leader       string          `json:"leader,omitempty"`
	Uptime       string          `json:"uptime"`
	Checks       map[string]bool `json:"checks"`
}

// GetHealthStatus returns the cluster health status.
func (ci *ClusterInfo) GetHealthStatus() *HealthStatus {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	healthyCount := 0
	var leaderID string
	for _, node := range ci.nodes {
		if node.Status == string(NodeStatusHealthy) {
			healthyCount++
		}
		if node.Role == string(NodeRoleLeader) {
			leaderID = node.ID
		}
	}

	status := "healthy"
	if healthyCount < len(ci.nodes) {
		status = "degraded"
	}
	if healthyCount == 0 {
		status = "unhealthy"
	}

	uptime := time.Since(ci.metadata.StartTime)

	return &HealthStatus{
		Status:       status,
		NodeCount:    len(ci.nodes),
		HealthyNodes: healthyCount,
		Leader:       leaderID,
		Uptime:       uptime.String(),
		Checks: map[string]bool{
			"storage": true, // Would check storage health
			"memory":  true, // Would check memory usage
		},
	}
}
