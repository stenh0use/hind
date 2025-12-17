package list

import (
	"testing"
	"time"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

func TestAggregateClusterStatus_AllRunning(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node2", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node3", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", result.Status)
	}
	if result.RunningNodes != 3 {
		t.Errorf("Expected 3 running nodes, got %d", result.RunningNodes)
	}
	if result.TotalNodes != 3 {
		t.Errorf("Expected 3 total nodes, got %d", result.TotalNodes)
	}
}

func TestAggregateClusterStatus_AllStopped(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Stopped.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node2", Status: provider.Stopped.String(), Created: time.Now().Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", result.Status)
	}
	if result.RunningNodes != 0 {
		t.Errorf("Expected 0 running nodes, got %d", result.RunningNodes)
	}
}

func TestAggregateClusterStatus_Mixed(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node2", Status: provider.Stopped.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node3", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "partial" {
		t.Errorf("Expected status 'partial', got '%s'", result.Status)
	}
	if result.RunningNodes != 2 {
		t.Errorf("Expected 2 running nodes, got %d", result.RunningNodes)
	}
}

func TestAggregateClusterStatus_WithErrors(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node2", Status: provider.Error.String(), Created: time.Now().Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", result.Status)
	}
}

func TestAggregateClusterStatus_NoContainers(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "not-found" {
		t.Errorf("Expected status 'not-found', got '%s'", result.Status)
	}
}

func TestAggregateClusterStatus_PartialRunning(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
			{Name: "node2", Status: provider.Running.String(), Created: time.Now().Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}, {}}, // 3 expected but only 2 containers
	}

	result := aggregateClusterStatus(info, cfg)

	if result.Status != "partial" {
		t.Errorf("Expected status 'partial', got '%s'", result.Status)
	}
	if result.RunningNodes != 2 {
		t.Errorf("Expected 2 running nodes, got %d", result.RunningNodes)
	}
	if result.TotalNodes != 3 {
		t.Errorf("Expected 3 total nodes, got %d", result.TotalNodes)
	}
}

func TestParseCreatedTime_RFC3339(t *testing.T) {
	now := time.Now()
	timeStr := now.Format(time.RFC3339)

	parsed, err := parseCreatedTime(timeStr)
	if err != nil {
		t.Errorf("Failed to parse RFC3339 time: %v", err)
	}

	// Allow for small differences due to formatting precision
	if parsed.Unix() != now.Unix() {
		t.Errorf("Parsed time doesn't match. Expected %v, got %v", now.Unix(), parsed.Unix())
	}
}

func TestParseCreatedTime_RFC3339Nano(t *testing.T) {
	now := time.Now()
	timeStr := now.Format(time.RFC3339Nano)

	parsed, err := parseCreatedTime(timeStr)
	if err != nil {
		t.Errorf("Failed to parse RFC3339Nano time: %v", err)
	}

	if parsed.Unix() != now.Unix() {
		t.Errorf("Parsed time doesn't match. Expected %v, got %v", now.Unix(), parsed.Unix())
	}
}

func TestParseCreatedTime_InvalidFormat(t *testing.T) {
	_, err := parseCreatedTime("invalid-time-string")
	if err == nil {
		t.Error("Expected error for invalid time format, got nil")
	}
}

func TestFormatCreatedTime_JustNow(t *testing.T) {
	now := time.Now().Add(-30 * time.Second)
	result := formatCreatedTime(now)

	if result != "just now" {
		t.Errorf("Expected 'just now', got '%s'", result)
	}
}

func TestFormatCreatedTime_Minutes(t *testing.T) {
	past := time.Now().Add(-5 * time.Minute)
	result := formatCreatedTime(past)

	if result != "5m ago" {
		t.Errorf("Expected '5m ago', got '%s'", result)
	}
}

func TestFormatCreatedTime_Hours(t *testing.T) {
	past := time.Now().Add(-3 * time.Hour)
	result := formatCreatedTime(past)

	if result != "3h ago" {
		t.Errorf("Expected '3h ago', got '%s'", result)
	}
}

func TestFormatCreatedTime_Days(t *testing.T) {
	past := time.Now().Add(-2 * 24 * time.Hour)
	result := formatCreatedTime(past)

	if result != "2d ago" {
		t.Errorf("Expected '2d ago', got '%s'", result)
	}
}

func TestFormatCreatedTime_AbsoluteDate(t *testing.T) {
	past := time.Now().Add(-10 * 24 * time.Hour)
	result := formatCreatedTime(past)

	expected := past.Format("2006-01-02")
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatCreatedTime_ZeroTime(t *testing.T) {
	zeroTime := time.Time{}
	result := formatCreatedTime(zeroTime)

	if result != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", result)
	}
}

func TestAggregateClusterStatus_OldestCreationTime(t *testing.T) {
	oldest := time.Now().Add(-48 * time.Hour)
	middle := time.Now().Add(-24 * time.Hour)
	newest := time.Now().Add(-1 * time.Hour)

	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: newest.Format(time.RFC3339)},
			{Name: "node2", Status: provider.Running.String(), Created: oldest.Format(time.RFC3339)},
			{Name: "node3", Status: provider.Running.String(), Created: middle.Format(time.RFC3339)},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}, {}, {}},
	}

	result := aggregateClusterStatus(info, cfg)

	// Should use the oldest time
	if result.Created.Unix() != oldest.Unix() {
		t.Errorf("Expected oldest creation time %v, got %v", oldest, result.Created)
	}
}

func TestAggregateClusterStatus_InvalidCreationTime(t *testing.T) {
	info := &provider.ClusterInfo{
		Containers: []provider.ContainerInfo{
			{Name: "node1", Status: provider.Running.String(), Created: "invalid-time"},
		},
	}

	cfg := &config.Cluster{
		Nodes: []config.Node{{}},
	}

	result := aggregateClusterStatus(info, cfg)

	// Should still return valid status even with invalid time
	if result.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", result.Status)
	}
}
