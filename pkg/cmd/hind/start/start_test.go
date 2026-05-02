package start

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

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

	if command.Use != "start [cluster-name]" {
		t.Errorf("Expected Use to be 'start [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Start or create a hind cluster" {
		t.Errorf("Expected Short to be 'Start or create a hind cluster', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 5 * time.Minute
	if DefaultStartTimeout != expected {
		t.Errorf("Expected DefaultStartTimeout to be %v, got %v", expected, DefaultStartTimeout)
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

	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}

	if timeoutFlag.DefValue != "5m0s" {
		t.Errorf("Expected timeout default value to be '5m0s', got '%s'", timeoutFlag.DefValue)
	}

	clientsFlag := command.Flags().Lookup("clients")
	if clientsFlag == nil {
		t.Fatal("Expected 'clients' flag to exist")
	}

	if clientsFlag.DefValue != "1" {
		t.Errorf("Expected clients default value to be '1', got '%s'", clientsFlag.DefValue)
	}

	verboseFlag := command.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("Expected 'verbose' flag to exist")
	}

	if command.Flags().Lookup("version") != nil {
		t.Fatal("Expected 'version' flag to be absent")
	}
}

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
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no args",
			args:      []string{},
			wantError: false,
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
