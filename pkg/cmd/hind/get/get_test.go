package get

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/provider"
)

func TestNewCommand(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	if command == nil {
		t.Fatal("NewCommand() returned nil")
	}

	if command.Use != "get [cluster-name]" {
		t.Errorf("Expected Use to be 'get [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Get a hind cluster details" {
		t.Errorf("Expected Short to be 'Get a hind cluster details', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 2 * time.Minute
	if DefaultGetTimeout != expected {
		t.Errorf("Expected DefaultGetTimeout to be %v, got %v", expected, DefaultGetTimeout)
	}
}

func TestCommandFlags(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	// Check if timeout flag exists
	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}

	if timeoutFlag.DefValue != "2m0s" {
		t.Errorf("Expected timeout default value to be '2m0s', got '%s'", timeoutFlag.DefValue)
	}
}

func TestCommandArgs(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Test with valid number of args (exactly 1)
	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no args",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "one arg",
			args:      []string{"test-cluster"},
			wantError: false,
		},
		{
			name:      "too many args",
			args:      []string{"cluster1", "cluster2"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCommand(logger, streams)
			command.SetArgs(tt.args)
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

type stubClusterManager struct {
	state *provider.ClusterInfo
	err   error
}

func (s *stubClusterManager) Get(ctx context.Context) (*provider.ClusterInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.state, nil
}

func TestRunE_FormatsStatusAndPortsFromRuntimeState(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}

	var out bytes.Buffer
	streams := cmd.IOStreams{Out: &out, ErrOut: io.Discard}

	originalFactory := newClusterManager
	newClusterManager = func(logger *log.Logger, name string) (clusterManager, error) {
		return &stubClusterManager{state: &provider.ClusterInfo{
			Network: provider.NetworkInfo{Name: "hind.test"},
			Containers: []provider.ContainerInfo{
				{
					HostName: "hind.demo.server.01",
					Image:    "nomad:latest",
					Status:   "running",
					Ports:    []string{"127.0.0.1:4646->4646/tcp", "127.0.0.1:4647->4647/tcp"},
				},
			},
		}}, nil
	}
	defer func() { newClusterManager = originalFactory }()

	err := runE(context.Background(), logger, streams, time.Second, []string{"demo"})
	if err != nil {
		t.Fatalf("runE returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Status: running") {
		t.Fatalf("expected running status in output, got: %s", output)
	}
	if strings.Contains(output, "%!s(") {
		t.Fatalf("expected no fmt artifact in output, got: %s", output)
	}
	if !strings.Contains(output, "127.0.0.1:4646->4646/tcp, 127.0.0.1:4647->4647/tcp") {
		t.Fatalf("expected joined ports in output, got: %s", output)
	}
}

func TestAggregateStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []provider.ContainerInfo
		expected   string
	}{
		{
			name:     "no containers",
			expected: provider.NA.String(),
		},
		{
			name:       "all running",
			containers: []provider.ContainerInfo{{Status: "running"}, {Status: "running"}},
			expected:   provider.Running.String(),
		},
		{
			name:       "all exited treated as stopped",
			containers: []provider.ContainerInfo{{Status: "exited"}, {Status: "exited"}},
			expected:   provider.Stopped.String(),
		},
		{
			name:       "mixed running and stopped reports error",
			containers: []provider.ContainerInfo{{Status: "running"}, {Status: "stopped"}},
			expected:   provider.Error.String(),
		},
		{
			name:       "unknown state reports error",
			containers: []provider.ContainerInfo{{Status: "restarting"}},
			expected:   provider.Error.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := aggregateStatus(&provider.ClusterInfo{Containers: tt.containers})
			if status != tt.expected {
				t.Fatalf("expected status %q, got %q", tt.expected, status)
			}
		})
	}
}

func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []string
		expected string
	}{
		{name: "empty ports", ports: nil, expected: "-"},
		{name: "single port", ports: []string{"127.0.0.1:4646->4646/tcp"}, expected: "127.0.0.1:4646->4646/tcp"},
		{name: "multiple ports", ports: []string{"4646/tcp", "4647/tcp"}, expected: "4646/tcp, 4647/tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatPorts(tt.ports)
			if actual != tt.expected {
				t.Fatalf("expected ports %q, got %q", tt.expected, actual)
			}
		})
	}
}
