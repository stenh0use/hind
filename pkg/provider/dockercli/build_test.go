package dockercli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/provider"
)

// fakeExecutor is a test double for CommandExecutor that records calls and
// returns configured results.
type fakeExecutor struct {
	// outputFn is called when Output is invoked. If nil, returns empty bytes.
	outputFn func(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// runFn is called when Run is invoked. If nil, returns nil.
	runFn func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error
	// capturedRunArgs holds the args passed to the most recent Run call.
	capturedRunArgs []string
}

func (f *fakeExecutor) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if f.outputFn != nil {
		return f.outputFn(ctx, dir, name, args...)
	}
	return []byte("{}"), nil
}

func (f *fakeExecutor) Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	f.capturedRunArgs = args
	if f.runFn != nil {
		return f.runFn(ctx, dir, stdout, stderr, name, args...)
	}
	return nil
}

// dockerInfoWithBuildx returns a JSON blob representing docker system info with buildx present.
func dockerInfoWithBuildx() []byte {
	info := dockerInfo{
		ClientInfo: clientInfo{
			Plugins: []plugin{{Name: "buildx"}},
		},
	}
	raw, _ := json.Marshal(info)
	return raw
}

// dockerInfoWithoutBuildx returns a JSON blob representing docker system info with no plugins.
func dockerInfoWithoutBuildx() []byte {
	info := dockerInfo{
		ClientInfo: clientInfo{
			Plugins: []plugin{},
		},
	}
	raw, _ := json.Marshal(info)
	return raw
}

// newTestLogger returns a logger that discards all output.
func newTestLogger() *log.Logger {
	return &log.Logger{Handler: discard.New()}
}

// writeMetadataFile writes a metadata.json with the given digest to dir.
func writeMetadataFile(t *testing.T, dir, digest string) {
	t.Helper()
	m := buildMetadata{ContainerImageDigest: digest}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("writeMetadataFile: marshal error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, metadataFileName), data, 0o600); err != nil {
		t.Fatalf("writeMetadataFile: write error: %v", err)
	}
}

func TestBuildImage_BuildxAbsent(t *testing.T) {
	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithoutBuildx(), nil
		},
	}
	client := newWithExecutor(newTestLogger(), exec)

	ctx := context.Background()
	tmpDir := t.TempDir()

	_, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       "myimage",
		Tag:        "latest",
		ContextDir: tmpDir,
	})

	if err == nil {
		t.Fatal("expected error when buildx is absent, got nil")
	}
	if !strings.Contains(err.Error(), "buildx") {
		t.Errorf("expected error to contain 'buildx', got: %q", err.Error())
	}
	// Verify no build command was executed (capturedRunArgs stays nil).
	if exec.capturedRunArgs != nil {
		t.Errorf("expected no build command to be run, but Run was called with: %v", exec.capturedRunArgs)
	}
}

func TestBuildImage_Success(t *testing.T) {
	const wantDigest = "sha256:abc123deadbeef"
	const wantName = "myimage"
	const wantTag = "v1.0.0"

	tmpDir := t.TempDir()

	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithBuildx(), nil
		},
		runFn: func(_ context.Context, _ string, _, _ io.Writer, _ string, _ ...string) error {
			writeMetadataFile(t, tmpDir, wantDigest)
			return nil
		},
	}

	client := newWithExecutor(newTestLogger(), exec)
	ctx := context.Background()

	result, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       wantName,
		Tag:        wantTag,
		ContextDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Digest != wantDigest {
		t.Errorf("Digest = %q, want %q", result.Digest, wantDigest)
	}
	wantImageRef := wantName + ":" + wantTag
	if result.ImageRef != wantImageRef {
		t.Errorf("ImageRef = %q, want %q", result.ImageRef, wantImageRef)
	}
	if !strings.HasPrefix(result.Digest, "sha256:") {
		t.Errorf("Digest does not start with 'sha256:': %q", result.Digest)
	}
}

func TestNew_SucceedsWithoutBuildx(t *testing.T) {
	// New must not check for buildx; it must succeed regardless.
	logger := newTestLogger()
	client := New(logger)
	if client == nil {
		t.Error("New returned nil, want non-nil provider.Client")
	}
}

func TestBuildImage_EmptyDigestIsError(t *testing.T) {
	tmpDir := t.TempDir()

	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithBuildx(), nil
		},
		runFn: func(_ context.Context, _ string, _, _ io.Writer, _ string, _ ...string) error {
			// Write metadata with empty digest.
			writeMetadataFile(t, tmpDir, "")
			return nil
		},
	}

	client := newWithExecutor(newTestLogger(), exec)
	ctx := context.Background()

	_, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       "myimage",
		Tag:        "v1",
		ContextDir: tmpDir,
	})

	if err == nil {
		t.Fatal("expected error for empty digest, got nil")
	}
}

func TestBuildImage_LoadFlagPresent(t *testing.T) {
	tmpDir := t.TempDir()
	const wantDigest = "sha256:loadtest"

	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithBuildx(), nil
		},
		runFn: func(_ context.Context, _ string, _, _ io.Writer, _ string, _ ...string) error {
			writeMetadataFile(t, tmpDir, wantDigest)
			return nil
		},
	}

	client := newWithExecutor(newTestLogger(), exec)
	ctx := context.Background()

	if _, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       "myimage",
		Tag:        "v1",
		ContextDir: tmpDir,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, arg := range exec.capturedRunArgs {
		if arg == "--load" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("--load not found in build args: %v", exec.capturedRunArgs)
	}
}

func TestBuildImage_PlatformOmittedWhenEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	const wantDigest = "sha256:platformtest"

	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithBuildx(), nil
		},
		runFn: func(_ context.Context, _ string, _, _ io.Writer, _ string, _ ...string) error {
			writeMetadataFile(t, tmpDir, wantDigest)
			return nil
		},
	}

	client := newWithExecutor(newTestLogger(), exec)
	ctx := context.Background()

	if _, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       "myimage",
		Tag:        "v1",
		ContextDir: tmpDir,
		Platform:   "", // empty — must be omitted
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, arg := range exec.capturedRunArgs {
		if arg == "--platform" {
			t.Errorf("--platform should be absent when Platform is empty, but found in args: %v", exec.capturedRunArgs)
		}
	}
}

func TestBuildImage_NoCacheWhenWithCacheFalse(t *testing.T) {
	tmpDir := t.TempDir()
	const wantDigest = "sha256:nocachetest"

	exec := &fakeExecutor{
		outputFn: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return dockerInfoWithBuildx(), nil
		},
		runFn: func(_ context.Context, _ string, _, _ io.Writer, _ string, _ ...string) error {
			writeMetadataFile(t, tmpDir, wantDigest)
			return nil
		},
	}

	client := newWithExecutor(newTestLogger(), exec)
	ctx := context.Background()

	if _, err := client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       "myimage",
		Tag:        "v1",
		ContextDir: tmpDir,
		WithCache:  false,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, arg := range exec.capturedRunArgs {
		if arg == "--no-cache" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("--no-cache not found in args when WithCache=false: %v", exec.capturedRunArgs)
	}
}
