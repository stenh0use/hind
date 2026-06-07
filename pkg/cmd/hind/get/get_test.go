package get

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cmd"
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

	require.NotNil(t, command, "NewCommand() returned nil")

	assert.Equal(t, "get [cluster-name]", command.Use)
	assert.Equal(t, "Get a hind cluster details", command.Short)
}

func TestDefaultTimeout(t *testing.T) {
	expected := 2 * time.Minute
	assert.Equal(t, expected, DefaultGetTimeout)
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
	require.NotNil(t, timeoutFlag, "Expected 'timeout' flag to exist")

	assert.Equal(t, "2m0s", timeoutFlag.DefValue)
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
	state orchestration.InspectResult
	err   error
}

func (s *stubClusterManager) Inspect(ctx context.Context) (orchestration.InspectResult, error) {
	if s.err != nil {
		return orchestration.InspectResult{}, s.err
	}
	return s.state, nil
}

func TestRunE_FormatsStatusFromRuntimeState(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}

	var out bytes.Buffer
	streams := cmd.IOStreams{Out: &out, ErrOut: io.Discard}

	originalFactory := newClusterManager
	newClusterManager = func(logger *log.Logger, name string) (clusterManager, error) {
		return &stubClusterManager{state: orchestration.InspectResult{
			NetworkName: "hind.test",
			Containers: []orchestration.ContainerSummary{
				{
					HostName: "hind.demo.server.01",
					Image:    "nomad:latest",
					Status:   domain.ContainerStatusRunning,
				},
			},
		}}, nil
	}
	defer func() { newClusterManager = originalFactory }()

	err := runE(context.Background(), logger, streams, time.Second, []string{"demo"})
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Status: running")
	assert.NotContains(t, output, "%!s(")
}

func TestRunE_NotFoundMapping(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	tests := []struct {
		name      string
		err       error
		wantToken string
	}{
		{name: "typed not found", err: &orchestration.NotFoundError{Operation: "inspect", Cluster: "demo"}, wantToken: "cluster 'demo' not found"},
		{name: "generic wrapped error", err: errors.New("boom"), wantToken: "failed to get cluster"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			originalFactory := newClusterManager
			newClusterManager = func(logger *log.Logger, name string) (clusterManager, error) {
				return &stubClusterManager{err: tt.err}, nil
			}
			defer func() { newClusterManager = originalFactory }()

			err := runE(context.Background(), logger, streams, time.Second, []string{"demo"})
			require.Error(t, err, "runE expected error")
			assert.Contains(t, err.Error(), tt.wantToken)
		})
	}
}

func TestAggregateContainerStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []orchestration.ContainerSummary
		expected   string
	}{
		{
			name:     "no containers",
			expected: string(domain.ClusterStatusNotFound),
		},
		{
			name: "all running",
			containers: []orchestration.ContainerSummary{
				{Status: domain.ContainerStatusRunning},
				{Status: domain.ContainerStatusRunning},
			},
			expected: string(domain.ClusterStatusRunning),
		},
		{
			name: "all stopped",
			containers: []orchestration.ContainerSummary{
				{Status: domain.ContainerStatusStopped},
				{Status: domain.ContainerStatusStopped},
			},
			expected: string(domain.ClusterStatusStopped),
		},
		{
			name: "mixed running and stopped returns partial",
			containers: []orchestration.ContainerSummary{
				{Status: domain.ContainerStatusRunning},
				{Status: domain.ContainerStatusStopped},
			},
			expected: string(domain.ClusterStatusPartial),
		},
		{
			name: "unknown state returns degraded",
			containers: []orchestration.ContainerSummary{
				{Status: domain.ContainerStatusUnknown},
			},
			expected: string(domain.ClusterStatusDegraded),
		},
		{
			name: "unhealthy returns degraded",
			containers: []orchestration.ContainerSummary{
				{Status: domain.ContainerStatusUnhealthy},
			},
			expected: string(domain.ClusterStatusDegraded),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := domain.AggregateContainerStatus(tt.containers)
			assert.Equal(t, tt.expected, string(status))
		})
	}
}
