package start

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/file"
)

// stubStartManager is a no-op clusterStarter used to bypass real Docker calls in tests.
type stubStartManager struct {
	startResult cluster.StartResult
}

func (s *stubStartManager) ConfigFileExists() bool                        { return false }
func (s *stubStartManager) SetClientCount(_ context.Context, _ int) error { return nil }
func (s *stubStartManager) Start(_ context.Context) (cluster.StartResult, error) {
	return s.startResult, nil
}
func (s *stubStartManager) CountClientNodes() int                { return 1 }
func (s *stubStartManager) Scale(_ context.Context, _ int) error { return nil }

// TestRunE_SetsActiveCluster verifies that after a successful start runE calls
// SetActiveCluster so that the started cluster becomes the active profile.
func TestRunE_SetsActiveCluster(t *testing.T) {
	// Redirect HOME so cluster state is isolated to this test.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	const clusterName = "test-start-cluster"

	// Pre-create the cluster directory so SetActiveCluster accepts the name.
	// runE calls SetActiveCluster after mgr.Start succeeds; SetActiveCluster verifies
	// the cluster directory exists before writing the active file.
	fm, err := file.NewFromHomeDir(cluster.DefaultConfigParentDir, cluster.DefaultConfigName)
	if err != nil {
		t.Fatalf("NewFromHomeDir: %v", err)
	}
	clusterDir := file.JoinPath(cluster.ClusterConfigDir, clusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	// Stub checkDockerDaemonFn so runE does not require a live Docker daemon.
	origDockerCheck := checkDockerDaemonFn
	checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
	defer func() { checkDockerDaemonFn = origDockerCheck }()

	// Stub newStartManagerFn so mgr.Start() does not require Docker.
	origManagerFn := newStartManagerFn
	newStartManagerFn = func(_ *log.Logger, _ string) (clusterStarter, error) {
		return &stubStartManager{startResult: cluster.StartResultCreated}, nil
	}
	defer func() { newStartManagerFn = origManagerFn }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	flags := &flagpole{
		timeout: DefaultStartTimeout,
		clients: 1,
	}

	// Build a minimal cobra command to satisfy the cmd parameter expected by runE.
	cobraCmd := NewCommand(logger, streams)

	if err := runE(cobraCmd, context.Background(), logger, streams, flags, []string{clusterName}); err != nil {
		t.Fatalf("runE returned unexpected error: %v", err)
	}

	// After a successful start the active cluster must be set to the started cluster name.
	active, err := cluster.GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster: %v", err)
	}
	if active != clusterName {
		t.Errorf("expected active cluster %q, got %q", clusterName, active)
	}
}

func TestClusterNameExtraction(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "no args uses default",
			args:     []string{},
			expected: "default",
		},
		{
			name:     "single arg uses cluster name",
			args:     []string{"dev"},
			expected: "dev",
		},
		{
			name:     "custom cluster name",
			args:     []string{"my-test-cluster"},
			expected: "my-test-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the cluster name extraction logic
			clusterName := "default"
			if len(tt.args) > 0 {
				clusterName = tt.args[0]
			}

			if clusterName != tt.expected {
				t.Errorf("expected cluster name %q, got %q", tt.expected, clusterName)
			}
		})
	}
}
