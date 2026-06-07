//go:build e2e
// +build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain runs a pre-flight cleanup of the default e2e cluster before any
// test executes, ensuring ports and containers from a prior run do not
// interfere. Individual tests that use unique cluster names manage their own
// cleanup independently.
func TestMain(m *testing.M) {
	runPreflightCleanup()
	os.Exit(m.Run())
}

func runPreflightCleanup() {
	clusterName := defaultE2EClusterName()
	log.Printf("preflight cleanup: stopping and removing cluster %q", clusterName)

	hindBinary := os.Getenv("HIND_E2E_BIN")
	if hindBinary == "" {
		repoRoot, err := findRepoRoot()
		if err != nil {
			log.Printf("preflight cleanup: unable to resolve repo root, skipping: %v", err)
			return
		}
		hindBinary = filepath.Join(repoRoot, "bin", "hind")
	}

	if _, err := os.Stat(hindBinary); err != nil {
		log.Printf("preflight cleanup: hind binary not found at %q, skipping cleanup", hindBinary)
		return
	}

	if _, err := runHindCommandAllowError(hindBinary, "stop", clusterName); err != nil {
		log.Printf("preflight cleanup: stop %q: %v (ignored)", clusterName, err)
	} else {
		log.Printf("preflight cleanup: stop %q: ok", clusterName)
	}

	if _, err := runHindCommandAllowError(hindBinary, "rm", clusterName); err != nil {
		log.Printf("preflight cleanup: rm %q: %v (ignored)", clusterName, err)
	} else {
		log.Printf("preflight cleanup: rm %q: ok", clusterName)
	}

	log.Printf("preflight cleanup: complete for cluster %q", clusterName)
}

const (
	commandTimeout = 20 * time.Minute
)

type commandSpec struct {
	name string
	args []string
}

type commandResult struct {
	stdout string
	stderr string
}

func lifecycleCommands(clusterName string) map[string]commandSpec {
	return map[string]commandSpec{
		"build_all": {
			name: "build all",
			args: []string{"build", "all"},
		},
		"start": {
			name: "start",
			args: []string{"start", clusterName},
		},
		"list": {
			name: "list",
			args: []string{"list"},
		},
		"get": {
			name: "get",
			args: []string{"get", clusterName},
		},
		"stop": {
			name: "stop",
			args: []string{"stop", clusterName},
		},
		"rm": {
			name: "rm",
			args: []string{"rm", clusterName},
		},
	}
}

func validateRunningContainers(t *testing.T, clusterName string) {
	t.Helper()
	services := []string{"consul", "nomad", "vault"}
	for _, service := range services {
		name := fmt.Sprintf("hind.%s.%s.01", clusterName, service)
		state := inspectContainerState(t, name)
		if !state.Exists {
			t.Fatalf("expected running container %q to exist", name)
		}
		if !state.Running {
			t.Fatalf("expected container %q to be running", name)
		}
	}
}

func validateStoppedContainers(t *testing.T, clusterName string) {
	t.Helper()
	services := []string{"consul", "nomad", "vault"}
	for _, service := range services {
		name := fmt.Sprintf("hind.%s.%s.01", clusterName, service)
		state := inspectContainerState(t, name)
		if !state.Exists {
			t.Fatalf("expected stopped container %q to still exist", name)
		}
		if state.Running {
			t.Fatalf("expected container %q to be stopped", name)
		}
	}
}

func validateRemovedContainers(t *testing.T, clusterName string) {
	t.Helper()
	services := []string{"consul", "nomad", "vault"}
	for _, service := range services {
		name := fmt.Sprintf("hind.%s.%s.01", clusterName, service)
		state := inspectContainerState(t, name)
		if state.Exists {
			t.Fatalf("expected removed container %q to be absent", name)
		}
	}
}

type containerState struct {
	Exists  bool
	Running bool
}

func inspectContainerState(t *testing.T, containerName string) containerState {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if strings.Contains(strings.ToLower(stderr.String()), "no such object") {
			return containerState{Exists: false}
		}
		t.Fatalf("docker inspect failed for %q: %v stderr=%s", containerName, err, stderr.String())
	}

	return containerState{Exists: true, Running: strings.TrimSpace(stdout.String()) == "true"}
}

func verifyServiceReachability(t *testing.T) {
	t.Helper()
	checkConsul(t)
	checkNomad(t)
	checkVault(t)
}

const (
	serviceReadinessTimeout  = 60 * time.Second
	serviceReadinessInterval = 2 * time.Second
)

func checkConsul(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(serviceReadinessTimeout)
	attempt := 0
	for {
		attempt++
		leader, err := fetchConsulLeader()
		if err == nil && leader != "" {
			t.Logf("checkConsul: attempt %d: leader elected: %s", attempt, leader)
			return
		}
		remaining := time.Until(deadline)
		t.Logf("checkConsul: attempt %d: not ready (leader=%q err=%v), %.0fs remaining", attempt, leader, err, remaining.Seconds())
		if remaining <= 0 {
			t.Fatalf("consul leader not elected within %s (last leader=%q, last err=%v)", serviceReadinessTimeout, leader, err)
		}
		time.Sleep(serviceReadinessInterval)
	}
}

func fetchConsulLeader() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8500/v1/status/leader", nil)
	if err != nil {
		return "", fmt.Errorf("create consul request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("consul request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("consul leader endpoint returned %d", resp.StatusCode)
	}
	var leader string
	if err := json.NewDecoder(resp.Body).Decode(&leader); err != nil {
		return "", fmt.Errorf("decode consul leader: %w", err)
	}
	return leader, nil
}

func checkNomad(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(serviceReadinessTimeout)
	attempt := 0
	for {
		attempt++
		leader, err := fetchNomadLeader()
		if err == nil && leader != "" {
			t.Logf("checkNomad: attempt %d: leader elected: %s", attempt, leader)
			return
		}
		remaining := time.Until(deadline)
		t.Logf("checkNomad: attempt %d: not ready (leader=%q err=%v), %.0fs remaining", attempt, leader, err, remaining.Seconds())
		if remaining <= 0 {
			t.Fatalf("nomad leader not elected within %s (last leader=%q, last err=%v)", serviceReadinessTimeout, leader, err)
		}
		time.Sleep(serviceReadinessInterval)
	}
}

func fetchNomadLeader() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:4646/v1/status/leader", nil)
	if err != nil {
		return "", fmt.Errorf("create nomad request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nomad request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nomad leader endpoint returned %d", resp.StatusCode)
	}
	var leader string
	if err := json.NewDecoder(resp.Body).Decode(&leader); err != nil {
		return "", fmt.Errorf("decode nomad leader: %w", err)
	}
	return leader, nil
}

func checkVault(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(serviceReadinessTimeout)
	attempt := 0
	for {
		attempt++
		sealed, err := fetchVaultSealStatus()
		if err == nil && !sealed {
			t.Logf("checkVault: attempt %d: vault unsealed", attempt)
			return
		}
		remaining := time.Until(deadline)
		t.Logf("checkVault: attempt %d: not ready (sealed=%v err=%v), %.0fs remaining", attempt, sealed, err, remaining.Seconds())
		if remaining <= 0 {
			t.Fatalf("vault not unsealed within %s (last sealed=%v, last err=%v)", serviceReadinessTimeout, sealed, err)
		}
		time.Sleep(serviceReadinessInterval)
	}
}

func fetchVaultSealStatus() (sealed bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8200/v1/sys/health", nil)
	if err != nil {
		return true, fmt.Errorf("create vault request: %w", err)
	}
	req.Header.Set("X-Vault-Token", "root")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return true, fmt.Errorf("vault request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return true, fmt.Errorf("vault health endpoint returned %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return true, fmt.Errorf("decode vault health: %w", err)
	}
	sealedVal, ok := payload["sealed"].(bool)
	if !ok {
		return true, fmt.Errorf("vault health payload missing boolean sealed field")
	}
	return sealedVal, nil
}

func httpGetJSON(t *testing.T, url string, headers map[string]string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create http request %s: %v", url, err)
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("http request failed %s: %v", url, err)
	}
	return response
}

func requireE2EPrerequisites(t *testing.T) string {
	t.Helper()

	hindBinary := os.Getenv("HIND_E2E_BIN")
	if hindBinary == "" {
		repoRoot, err := findRepoRoot()
		if err != nil {
			t.Skipf("skipping e2e test: unable to resolve repo root: %v", err)
		}
		hindBinary = filepath.Join(repoRoot, "bin", "hind")
	}

	if _, err := os.Stat(hindBinary); err != nil {
		t.Skipf("skipping e2e test: hind binary not found at %q (build with `make hind-cli`)", hindBinary)
	}

	dockerCheck := exec.Command("docker", "info")
	if err := dockerCheck.Run(); err != nil {
		t.Skipf("skipping e2e test: docker is not reachable: %v", err)
	}

	return hindBinary
}

func runHindCommand(t *testing.T, binary string, args ...string) commandResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("command timed out: %q %s: %v\nstdout:\n%s\nstderr:\n%s", binary, strings.Join(args, " "), ctx.Err(), stdout.String(), stderr.String())
	}
	if err != nil {
		t.Fatalf("command failed: %q %s: %v\nstdout:\n%s\nstderr:\n%s", binary, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}

	return commandResult{stdout: stdout.String(), stderr: stderr.String()}
}

func uniqueClusterName(prefix string) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%d-%04d", prefix, time.Now().Unix(), rng.Intn(10000))
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	current := cwd
	for {
		if _, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil {
			return current, nil
		}

		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	return "", fmt.Errorf("go.mod not found from %q", cwd)
}

func cleanupCluster(t *testing.T, binary string, clusterName string) {
	t.Helper()
	cleanupClusterWithMode(t, binary, clusterName, "post")
}

func preflightCleanupCluster(t *testing.T, binary string, clusterName string) {
	t.Helper()
	cleanupClusterWithMode(t, binary, clusterName, "preflight")
}

func cleanupClusterWithMode(t *testing.T, binary string, clusterName string, mode string) {
	t.Helper()

	commands := lifecycleCommands(clusterName)
	t.Logf("%s cleanup: ensuring cluster %q starts clean", mode, clusterName)

	cleanupErrs := make([]error, 0, 2)

	if _, err := runHindCommandAllowError(binary, commands["stop"].args...); err != nil {
		if !isNotFoundError(err) {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("stop cleanup attempt: %w", err))
		}
	}

	if _, err := runHindCommandAllowError(binary, commands["rm"].args...); err != nil {
		if !isNotFoundError(err) {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("rm cleanup attempt: %w", err))
		}
	}

	if len(cleanupErrs) > 0 {
		t.Fatalf("%s cleanup failed for cluster %q: %v", mode, clusterName, errors.Join(cleanupErrs...))
	}

	t.Logf("%s cleanup complete for cluster %q", mode, clusterName)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func defaultE2EClusterName() string {
	if name := strings.TrimSpace(os.Getenv("HIND_E2E_CLUSTER_NAME")); name != "" {
		return name
	}
	return "e2e-cli"
}

func runE2EPreflightCleanup(t *testing.T, binary string) {
	t.Helper()
	preflightCleanupCluster(t, binary, defaultE2EClusterName())
}

func verifyE2EPreflightCleanup(t *testing.T, binary string) {
	t.Helper()
	commands := lifecycleCommands(defaultE2EClusterName())
	assertRemovedState(t, binary, defaultE2EClusterName(), commands)
}

func TestE2E_PreflightCleanup_RemovesRunningCluster(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	clusterName := defaultE2EClusterName()
	commands := lifecycleCommands(clusterName)

	preflightCleanupCluster(t, hindBinary, clusterName)
	runHindCommand(t, hindBinary, commands["start"].args...)
	runE2EPreflightCleanup(t, hindBinary)

	verifyE2EPreflightCleanup(t, hindBinary)
}

func TestE2E_PreflightCleanup_HandlesMissingCluster(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	runE2EPreflightCleanup(t, hindBinary)
	verifyE2EPreflightCleanup(t, hindBinary)
}

func TestE2E_PreflightCleanup_IsIdempotent(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	runE2EPreflightCleanup(t, hindBinary)
	runE2EPreflightCleanup(t, hindBinary)
	verifyE2EPreflightCleanup(t, hindBinary)
}

func TestE2E_PreflightCleanup_NotFoundErrorMatcher(t *testing.T) {
	err := fmt.Errorf("cluster not found")
	if !isNotFoundError(err) {
		t.Fatal("expected not found error matcher to return true")
	}
}

func TestE2E_PreflightCleanup_NotFoundErrorMatcherFalse(t *testing.T) {
	err := fmt.Errorf("permission denied")
	if isNotFoundError(err) {
		t.Fatal("expected not-found matcher to return false")
	}
}

func TestE2E_PreflightCleanup_NotFoundErrorNil(t *testing.T) {
	if isNotFoundError(nil) {
		t.Fatal("expected nil error not to be treated as not-found")
	}
}

func TestE2E_PreflightCleanup_VisibleInLogs(t *testing.T) {
	hindBinary := requireE2EPrerequisites(t)
	runE2EPreflightCleanup(t, hindBinary)
	t.Log("pre-flight cleanup ran before lifecycle assertions")
}

func runHindCommandAllowError(binary string, args ...string) (commandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{stdout: stdout.String(), stderr: stderr.String()}
	if ctx.Err() != nil {
		return result, fmt.Errorf("command timed out: %q %s: %w; stdout=%q stderr=%q", binary, strings.Join(args, " "), ctx.Err(), result.stdout, result.stderr)
	}
	if err != nil {
		return result, fmt.Errorf("command failed: %q %s: %w; stdout=%q stderr=%q", binary, strings.Join(args, " "), err, result.stdout, result.stderr)
	}

	return result, nil
}

func runHindCommandExpectError(t *testing.T, binary string, args ...string) error {
	t.Helper()

	_, err := runHindCommandAllowError(binary, args...)
	if err == nil {
		return fmt.Errorf("command unexpectedly succeeded: %q %s", binary, strings.Join(args, " "))
	}

	return err
}
