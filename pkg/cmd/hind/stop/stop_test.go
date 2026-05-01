package stop

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
)

func testLogger() *log.Logger {
	return &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
}

type fakeStopManager struct {
	configExists bool
	result       cluster.StopResult
	err          error
	receivedOpts cluster.StopOptions
}

func (f *fakeStopManager) ConfigFileExists() bool {
	return f.configExists
}

func (f *fakeStopManager) StopWithOptions(_ context.Context, opts cluster.StopOptions) (cluster.StopResult, error) {
	f.receivedOpts = opts
	if f.err != nil {
		return cluster.StopResult{}, f.err
	}
	return f.result, nil
}

func withRunEStubs(t *testing.T, active string, build func() clusterStopper) {
	t.Helper()
	oldActive := getActiveClusterFn
	oldNewMgr := newClusterManagerFn
	getActiveClusterFn = func() (string, error) { return active, nil }
	newClusterManagerFn = func(_ *log.Logger, _ string) (clusterStopper, error) { return build(), nil }
	t.Cleanup(func() {
		getActiveClusterFn = oldActive
		newClusterManagerFn = oldNewMgr
	})
}

func TestRunEMessageContracts(t *testing.T) {
	tests := []struct {
		name            string
		force           bool
		verbose         bool
		result          cluster.StopResult
		wantLines       []string
		wantForceOpt    bool
		wantVerboseOpt  bool
		wantFailContain bool
	}{
		{
			name:      "already stopped",
			result:    cluster.StopResult{AlreadyStoppedCount: 2},
			wantLines: []string{"Cluster 'default' is already stopped"},
		},
		{
			name:         "force stopped",
			force:        true,
			result:       cluster.StopResult{StoppedCount: 2},
			wantLines:    []string{"Cluster 'default' force stopped"},
			wantForceOpt: true,
		},
		{
			name:      "partial failure warnings",
			result:    cluster.StopResult{StoppedCount: 1, FailedCount: 1, Failures: []string{"n2"}},
			wantLines: []string{"Failed to stop container 'n2'", "Cluster 'default' partially stopped"},
		},
		{
			name:      "unhealthy pre-failed",
			result:    cluster.StopResult{AlreadyStoppedCount: 1, FailedPreStopCount: 1},
			wantLines: []string{"Cluster 'default' stopped (some containers were already failed)"},
		},
		{
			name:    "verbose ordering",
			verbose: true,
			result: cluster.StopResult{StoppedCount: 1, VerboseLines: []string{
				"Checking container 'n1' status",
				"Stopping container 'n1'",
			}},
			wantLines: []string{
				"Checking cluster 'default' status",
				"Checking container 'n1' status",
				"Stopping container 'n1'",
				"Cluster 'default' stopped successfully",
			},
			wantVerboseOpt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &fakeStopManager{configExists: true, result: tt.result}
			withRunEStubs(t, "", func() clusterStopper { return mgr })
			errBuf := &bytes.Buffer{}
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: errBuf}

			err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, tt.force, tt.verbose, "")
			if err != nil {
				t.Fatalf("runE() error = %v", err)
			}

			out := strings.TrimSpace(errBuf.String())
			gotLines := []string{}
			if out != "" {
				gotLines = strings.Split(out, "\n")
			}
			if len(gotLines) != len(tt.wantLines) {
				t.Fatalf("line count=%d want=%d output=%q", len(gotLines), len(tt.wantLines), errBuf.String())
			}
			for i := range tt.wantLines {
				if gotLines[i] != tt.wantLines[i] {
					t.Fatalf("line[%d]=%q want %q", i, gotLines[i], tt.wantLines[i])
				}
			}
			if mgr.receivedOpts.Force != tt.wantForceOpt {
				t.Fatalf("force opt=%v want %v", mgr.receivedOpts.Force, tt.wantForceOpt)
			}
			if mgr.receivedOpts.Verbose != tt.wantVerboseOpt {
				t.Fatalf("verbose opt=%v want %v", mgr.receivedOpts.Verbose, tt.wantVerboseOpt)
			}
		})
	}
}

func TestRunEStopError(t *testing.T) {
	mgr := &fakeStopManager{configExists: true, err: errors.New("boom")}
	withRunEStubs(t, "", func() clusterStopper { return mgr })
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to stop cluster") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunEClusterNotFound(t *testing.T) {
	mgr := &fakeStopManager{configExists: false}
	withRunEStubs(t, "", func() clusterStopper { return mgr })
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cluster 'default' not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunEUsesActiveClusterWhenNoArg(t *testing.T) {
	mgr := &fakeStopManager{configExists: true, result: cluster.StopResult{AlreadyStoppedCount: 1}}
	oldActive := getActiveClusterFn
	oldNewMgr := newClusterManagerFn
	getActiveClusterFn = func() (string, error) { return "active", nil }
	var gotName string
	newClusterManagerFn = func(_ *log.Logger, clusterName string) (clusterStopper, error) {
		gotName = clusterName
		return mgr, nil
	}
	t.Cleanup(func() {
		getActiveClusterFn = oldActive
		newClusterManagerFn = oldNewMgr
	})

	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, false, "")
	if err != nil {
		t.Fatalf("runE() error = %v", err)
	}
	if gotName != "active" {
		t.Fatalf("cluster name=%q want active", gotName)
	}
}

func TestNewCommand(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	if command == nil {
		t.Fatal("NewCommand() returned nil")
	}

	if command.Use != "stop [cluster-name]" {
		t.Errorf("Expected Use to be 'stop [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Stop a hind cluster" {
		t.Errorf("Expected Short to be 'Stop a hind cluster', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 30 * time.Second
	if DefaultStopTimeout != expected {
		t.Errorf("Expected DefaultStopTimeout to be %v, got %v", expected, DefaultStopTimeout)
	}
}

func TestCommandFlags(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	// Check flags exist
	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}
	forceFlag := command.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("Expected 'force' flag to exist")
	}
	verboseFlag := command.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("Expected 'verbose' flag to exist")
	}

	if timeoutFlag.DefValue != "30s" {
		t.Errorf("Expected timeout default value to be '30s', got '%s'", timeoutFlag.DefValue)
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("Expected force default value to be 'false', got '%s'", forceFlag.DefValue)
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("Expected verbose default value to be 'false', got '%s'", verboseFlag.DefValue)
	}
}

func TestCommandArgs(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Test with valid number of args (0 or 1)
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

	if command.Use != "stop [cluster-name]" {
		t.Errorf("Expected Use to be 'stop [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Stop a hind cluster" {
		t.Errorf("Expected Short to be 'Stop a hind cluster', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 30 * time.Second
	if DefaultStopTimeout != expected {
		t.Errorf("Expected DefaultStopTimeout to be %v, got %v", expected, DefaultStopTimeout)
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

	// Check flags exist
	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}
	forceFlag := command.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("Expected 'force' flag to exist")
	}
	verboseFlag := command.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("Expected 'verbose' flag to exist")
	}

	if timeoutFlag.DefValue != "30s" {
		t.Errorf("Expected timeout default value to be '30s', got '%s'", timeoutFlag.DefValue)
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("Expected force default value to be 'false', got '%s'", forceFlag.DefValue)
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("Expected verbose default value to be 'false', got '%s'", verboseFlag.DefValue)
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

	// Test with valid number of args (0 or 1)
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
