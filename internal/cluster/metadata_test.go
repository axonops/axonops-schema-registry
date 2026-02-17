package cluster

import (
	"runtime"
	"testing"
)

func TestNewClusterInfo(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	if ci == nil {
		t.Fatal("expected non-nil ClusterInfo")
	}

	meta := ci.GetMetadata()
	if meta.ClusterID == "" {
		t.Error("expected non-empty cluster ID")
	}
	if meta.NodeID == "" {
		t.Error("expected non-empty node ID")
	}
	if meta.GoVersion != runtime.Version() {
		t.Errorf("expected go version %s, got %s", runtime.Version(), meta.GoVersion)
	}
	if meta.StartTime.IsZero() {
		t.Error("expected start time to be set")
	}
}

func TestGetClusterID(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	id := ci.GetClusterID()
	if id == "" {
		t.Error("expected non-empty cluster ID")
	}
}

func TestSetClusterID(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.SetClusterID("custom-id")
	if ci.GetClusterID() != "custom-id" {
		t.Errorf("expected custom-id, got %s", ci.GetClusterID())
	}
}

func TestGetVersion(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	v := ci.GetVersion()
	if v["version"] == "" {
		t.Error("expected version")
	}
	if v["go_version"] != runtime.Version() {
		t.Errorf("expected %s, got %s", runtime.Version(), v["go_version"])
	}
}

func TestGetSelf(t *testing.T) {
	ci := NewClusterInfo("10.0.0.1", 9090)
	self := ci.GetSelf()
	if self.Address != "10.0.0.1" {
		t.Errorf("expected address 10.0.0.1, got %s", self.Address)
	}
	if self.Port != 9090 {
		t.Errorf("expected port 9090, got %d", self.Port)
	}
	if self.Status != string(NodeStatusHealthy) {
		t.Errorf("expected healthy, got %s", self.Status)
	}
	if self.Role != string(NodeRoleLeader) {
		t.Errorf("expected leader, got %s", self.Role)
	}
}

func TestGetNodes(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	nodes := ci.GetNodes()
	if len(nodes) != 1 {
		t.Errorf("expected 1 node (self), got %d", len(nodes))
	}
}

func TestAddNode(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.AddNode(&Node{
		ID:       "node-2",
		Hostname: "node2.example.com",
		Address:  "10.0.0.2",
		Port:     8081,
		Status:   string(NodeStatusHealthy),
		Role:     string(NodeRoleFollower),
	})

	nodes := ci.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestGetNode(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.AddNode(&Node{ID: "node-2", Address: "10.0.0.2"})

	node, ok := ci.GetNode("node-2")
	if !ok {
		t.Fatal("expected to find node")
	}
	if node.Address != "10.0.0.2" {
		t.Errorf("expected 10.0.0.2, got %s", node.Address)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	_, ok := ci.GetNode("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestRemoveNode(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.AddNode(&Node{ID: "node-2"})
	ci.RemoveNode("node-2")

	_, ok := ci.GetNode("node-2")
	if ok {
		t.Error("expected node to be removed")
	}
}

func TestUpdateNodeStatus(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.AddNode(&Node{ID: "node-2", Status: string(NodeStatusHealthy)})

	ci.UpdateNodeStatus("node-2", NodeStatusUnhealthy)

	node, _ := ci.GetNode("node-2")
	if node.Status != string(NodeStatusUnhealthy) {
		t.Errorf("expected unhealthy, got %s", node.Status)
	}
	if node.LastSeen.IsZero() {
		t.Error("expected LastSeen to be updated")
	}
}

func TestUpdateNodeStatus_Nonexistent(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	// Should not panic
	ci.UpdateNodeStatus("nonexistent", NodeStatusUnhealthy)
}

func TestUpdateSelfStatus(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.UpdateSelfStatus(NodeStatusStopping)

	self := ci.GetSelf()
	if self.Status != string(NodeStatusStopping) {
		t.Errorf("expected stopping, got %s", self.Status)
	}
}

func TestIsHealthy(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	if !ci.IsHealthy() {
		t.Error("expected healthy")
	}

	ci.UpdateSelfStatus(NodeStatusUnhealthy)
	if ci.IsHealthy() {
		t.Error("expected unhealthy")
	}
}

func TestIsLeader(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	if !ci.IsLeader() {
		t.Error("single node should be leader")
	}
}

func TestGetLeader(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	leader := ci.GetLeader()
	if leader == nil {
		t.Fatal("expected leader")
	}
	if leader.Role != string(NodeRoleLeader) {
		t.Errorf("expected leader role, got %s", leader.Role)
	}
}

func TestGetLeader_NoLeader(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	// Change self to follower
	ci.GetSelf().Role = string(NodeRoleFollower)

	leader := ci.GetLeader()
	if leader != nil {
		t.Error("expected no leader")
	}
}

func TestSetSchemaCount(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.SetSchemaCount(42)

	self := ci.GetSelf()
	if self.SchemaCount != 42 {
		t.Errorf("expected 42, got %d", self.SchemaCount)
	}
}

func TestGetHealthStatus_Healthy(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	status := ci.GetHealthStatus()

	if status.Status != "healthy" {
		t.Errorf("expected healthy, got %s", status.Status)
	}
	if status.NodeCount != 1 {
		t.Errorf("expected 1 node, got %d", status.NodeCount)
	}
	if status.HealthyNodes != 1 {
		t.Errorf("expected 1 healthy, got %d", status.HealthyNodes)
	}
	if status.Leader == "" {
		t.Error("expected leader ID")
	}
	if status.Uptime == "" {
		t.Error("expected uptime")
	}
}

func TestGetHealthStatus_Degraded(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.AddNode(&Node{ID: "node-2", Status: string(NodeStatusUnhealthy), Role: string(NodeRoleFollower)})

	status := ci.GetHealthStatus()
	if status.Status != "degraded" {
		t.Errorf("expected degraded, got %s", status.Status)
	}
	if status.NodeCount != 2 {
		t.Errorf("expected 2 nodes, got %d", status.NodeCount)
	}
	if status.HealthyNodes != 1 {
		t.Errorf("expected 1 healthy, got %d", status.HealthyNodes)
	}
}

func TestGetHealthStatus_Unhealthy(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	ci.UpdateSelfStatus(NodeStatusUnhealthy)

	status := ci.GetHealthStatus()
	if status.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", status.Status)
	}
	if status.HealthyNodes != 0 {
		t.Errorf("expected 0 healthy, got %d", status.HealthyNodes)
	}
}

func TestGetHealthStatus_Checks(t *testing.T) {
	ci := NewClusterInfo("localhost", 8081)
	status := ci.GetHealthStatus()

	if !status.Checks["storage"] {
		t.Error("expected storage check to be true")
	}
	if !status.Checks["memory"] {
		t.Error("expected memory check to be true")
	}
}
