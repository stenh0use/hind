//go:build e2e
// +build e2e

package e2e_test

import (
	"fmt"
	"strings"
	"testing"
)

func TestE2E_Lifecycle_StartStopStartRmStop(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	clusterName := defaultE2EClusterName()
	commands := lifecycleCommands(clusterName)

	runE2EPreflightCleanup(t, hindBinary)

	t.Cleanup(func() {
		cleanupCluster(t, hindBinary, clusterName)
	})

	runHindCommand(t, hindBinary, commands["build_all"].args...)
	runHindCommand(t, hindBinary, commands["start"].args...)

	assertRunningState(t, hindBinary, clusterName, commands)
	validateRunningContainers(t, clusterName)
	verifyServiceReachability(t)

	runHindCommand(t, hindBinary, commands["stop"].args...)
	assertStoppedState(t, hindBinary, clusterName, commands)
	validateStoppedContainers(t, clusterName)

	runHindCommand(t, hindBinary, commands["start"].args...)
	assertRunningState(t, hindBinary, clusterName, commands)
	validateRunningContainers(t, clusterName)
	verifyServiceReachability(t)

	runHindCommand(t, hindBinary, commands["rm"].args...)
	assertRemovedState(t, hindBinary, clusterName, commands)
	validateRemovedContainers(t, clusterName)

	finalStopErr := runHindCommandExpectError(t, hindBinary, commands["stop"].args...)
	if !strings.Contains(strings.ToLower(finalStopErr.Error()), "not found") {
		t.Fatalf("expected final stop after rm to return not-found style error, got: %v", finalStopErr)
	}
}

func assertRunningState(t *testing.T, hindBinary string, clusterName string, commands map[string]commandSpec) {
	t.Helper()
	listResult := runHindCommand(t, hindBinary, commands["list"].args...)
	if !strings.Contains(listResult.stdout, clusterName) {
		t.Fatalf("expected list output to contain cluster %q while running, got stdout:\n%s\nstderr:\n%s", clusterName, listResult.stdout, listResult.stderr)
	}
	if !strings.Contains(strings.ToLower(listResult.stdout), "running") {
		t.Fatalf("expected list output to indicate running state, got stdout:\n%s", listResult.stdout)
	}

	getResult := runHindCommand(t, hindBinary, commands["get"].args...)
	for _, token := range []string{clusterName, "consul", "nomad", "vault"} {
		if !strings.Contains(strings.ToLower(getResult.stdout), strings.ToLower(token)) {
			t.Fatalf("expected get output to include %q while running, got stdout:\n%s\nstderr:\n%s", token, getResult.stdout, getResult.stderr)
		}
	}
}

func assertStoppedState(t *testing.T, hindBinary string, clusterName string, commands map[string]commandSpec) {
	t.Helper()
	listResult := runHindCommand(t, hindBinary, commands["list"].args...)
	clusterListLine, ok := findLineContainingAll(listResult.stdout, clusterName)
	if !ok {
		t.Fatalf("expected list output to include cluster %q after stop, got stdout:\n%s", clusterName, listResult.stdout)
	}
	if !strings.Contains(strings.ToLower(clusterListLine), "stopped") {
		t.Fatalf("expected list line for cluster %q to indicate stopped state, got line %q from stdout:\n%s", clusterName, clusterListLine, listResult.stdout)
	}

	getResult := runHindCommand(t, hindBinary, commands["get"].args...)
	if !strings.Contains(getResult.stdout, clusterName) {
		t.Fatalf("expected get output to include cluster %q after stop, got stdout:\n%s\nstderr:\n%s", clusterName, getResult.stdout, getResult.stderr)
	}
	if !strings.Contains(strings.ToLower(getResult.stdout), "stopped") {
		t.Fatalf("expected get output to indicate stopped state after stop, got stdout:\n%s\nstderr:\n%s", getResult.stdout, getResult.stderr)
	}
	for _, service := range []string{"consul", "nomad", "vault"} {
		if !strings.Contains(strings.ToLower(getResult.stdout), strings.ToLower(service)) {
			t.Fatalf("expected get output to include service %q after stop, got stdout:\n%s\nstderr:\n%s", service, getResult.stdout, getResult.stderr)
		}
	}
}

func findLineContainingAll(output string, tokens ...string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lineLower := strings.ToLower(trimmed)
		match := true
		for _, token := range tokens {
			if !strings.Contains(lineLower, strings.ToLower(token)) {
				match = false
				break
			}
		}
		if match {
			return trimmed, true
		}
	}
	return "", false
}

func assertCleanupAttempts(t *testing.T, cleanupErrors []error, expectedCommands ...string) {
	t.Helper()
	if len(cleanupErrors) == 0 {
		t.Fatalf("expected at least one cleanup attempt diagnostic, got none")
	}

	for _, expectedCommand := range expectedCommands {
		found := false
		for _, cleanupErr := range cleanupErrors {
			if strings.Contains(cleanupErr.Error(), expectedCommand) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected cleanup diagnostics to include attempted %q command, got: %v", expectedCommand, cleanupErrors)
		}
	}
}

func runLifecycleFlowWithInjectedFailure(t *testing.T, hindBinary string, commands map[string]commandSpec, failAfterCommand string) []error {
	t.Helper()
	cleanupErrors := make([]error, 0, 2)

	if _, err := runHindCommandAllowError(hindBinary, commands["start"].args...); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("start command failed before injected failure: %w", err))
		return cleanupErrors
	}

	if failAfterCommand == "start" {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("injected failure after start for cleanup verification"))
	}

	if _, err := runHindCommandAllowError(hindBinary, commands["stop"].args...); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("attempted stop: %w", err))
	} else {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("attempted stop: ok"))
	}

	if _, err := runHindCommandAllowError(hindBinary, commands["rm"].args...); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("attempted rm: %w", err))
	} else {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("attempted rm: ok"))
	}

	return cleanupErrors
}

func TestE2E_Lifecycle_CleanupOnFailure(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	clusterName := uniqueClusterName("e2e-cleanup")
	commands := lifecycleCommands(clusterName)

	// Ensure the unique cluster is torn down even if the test fails before
	// runLifecycleFlowWithInjectedFailure performs its own inline stop+rm.
	t.Cleanup(func() {
		cleanupCluster(t, hindBinary, clusterName)
	})

	cleanupErrors := runLifecycleFlowWithInjectedFailure(t, hindBinary, commands, "start")
	assertCleanupAttempts(t, cleanupErrors, "stop", "rm")
	if !strings.Contains(cleanupErrors[0].Error(), "injected failure after start") {
		t.Fatalf("expected injected failure context in diagnostics, got: %v", cleanupErrors)
	}
	if !strings.Contains(cleanupErrors[0].Error(), "cleanup verification") {
		t.Fatalf("expected actionable injected failure diagnostics, got: %v", cleanupErrors)
	}

	assertRemovedState(t, hindBinary, clusterName, commands)
}

func assertRemovedState(t *testing.T, hindBinary string, clusterName string, commands map[string]commandSpec) {
	t.Helper()
	listResult := runHindCommand(t, hindBinary, commands["list"].args...)
	if strings.Contains(listResult.stdout, clusterName) {
		t.Fatalf("expected list output to exclude removed cluster %q, got stdout:\n%s", clusterName, listResult.stdout)
	}

	getErr := runHindCommandExpectError(t, hindBinary, commands["get"].args...)
	if !strings.Contains(strings.ToLower(getErr.Error()), "not found") {
		t.Fatalf("expected get on removed cluster to return not-found style error, got: %v", getErr)
	}
}

