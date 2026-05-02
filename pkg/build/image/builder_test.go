package image

import (
	"context"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

// providerStub is a minimal stub implementing provider.Client for use in builder tests.
// Only BuildImage and TagExists need real behaviour; all others are no-ops.
type providerStub struct {
	buildImageFn func(ctx context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error)
	tagExistsFn  func(ctx context.Context, name, tag string) (bool, error)
}

func (s *providerStub) BuildImage(ctx context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
	if s.buildImageFn != nil {
		return s.buildImageFn(ctx, opts)
	}
	return provider.BuildImageResult{Digest: "sha256:stub", ImageRef: opts.Name + ":" + opts.Tag}, nil
}

func (s *providerStub) TagExists(ctx context.Context, name, tag string) (bool, error) {
	if s.tagExistsFn != nil {
		return s.tagExistsFn(ctx, name, tag)
	}
	return true, nil
}

// Remaining provider.Client methods — no-op stubs.
func (s *providerStub) CreateContainer(ctx context.Context, cfg provider.ContainerSpec) (string, error) {
	return "", nil
}
func (s *providerStub) StartContainer(ctx context.Context, name string) error  { return nil }
func (s *providerStub) StopContainer(ctx context.Context, name string) error   { return nil }
func (s *providerStub) KillContainer(ctx context.Context, name string) error   { return nil }
func (s *providerStub) DeleteContainer(ctx context.Context, name string) error { return nil }
func (s *providerStub) InspectContainer(ctx context.Context, name string) (*provider.ContainerInfo, error) {
	return nil, nil
}
func (s *providerStub) ListContainers(ctx context.Context, filters []string) ([]provider.ContainerInfo, error) {
	return nil, nil
}
func (s *providerStub) PullImage(ctx context.Context, name, tag string) error { return nil }
func (s *providerStub) CreateNetwork(ctx context.Context, cfg config.Network) (string, error) {
	return "", nil
}
func (s *providerStub) DeleteNetwork(ctx context.Context, name string) error { return nil }
func (s *providerStub) ListNetworks(ctx context.Context, filters []string) ([]provider.NetworkInfo, error) {
	return nil, nil
}
func (s *providerStub) InspectNetwork(ctx context.Context, name string) (*provider.NetworkInfo, error) {
	return nil, nil
}

// newTestLogger returns a logger that discards all output.
func newTestLogger() *log.Logger {
	return &log.Logger{Handler: discard.New()}
}

// newStubClient returns a providerStub that satisfies provider.Client.
func newStubClient() *providerStub {
	return &providerStub{}
}

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name    string
		kind    release.ImageKind
		wantErr bool
	}{
		{name: "valid consul image", kind: release.Consul, wantErr: false},
		{name: "valid nomad image", kind: release.Nomad, wantErr: false},
		{name: "valid nomad-client image", kind: release.NomadClient, wantErr: false},
		{name: "valid vault image", kind: release.Vault, wantErr: false},
		{name: "invalid image kind", kind: release.ImageKind("invalid"), wantErr: true},
		{name: "empty image kind", kind: release.ImageKind(""), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLogger()
			got, err := NewBuilder(logger, newStubClient(), tt.kind)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBuilder(%v) = %v, want error", tt.kind, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewBuilder(%v) unexpected error: %v", tt.kind, err)
			}
			if got == nil {
				t.Errorf("NewBuilder(%v) = nil, want non-nil Builder", tt.kind)
			}
			if got.logger == nil {
				t.Errorf("NewBuilder(%v).logger = nil, want non-nil logger", tt.kind)
			}
			if got.image.Kind != tt.kind {
				t.Errorf("NewBuilder(%v).image.Kind = %v, want %v", tt.kind, got.image.Kind, tt.kind)
			}
		})
	}
}

func TestConstructName(t *testing.T) {
	tests := []struct {
		name       string
		imageKind  release.ImageKind
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "consul image name",
			imageKind:  release.Consul,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "consul",
		},
		{
			name:       "nomad image name",
			imageKind:  release.Nomad,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "nomad",
		},
		{
			name:       "nomad-client image name",
			imageKind:  release.NomadClient,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "nomad-client",
		},
		{
			name:       "vault image name",
			imageKind:  release.Vault,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "vault",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.imageKind.ImageName()

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("constructName(%v) = %q, want prefix %q", tt.imageKind, got, tt.wantPrefix)
			}
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("constructName(%v) = %q, want suffix %q", tt.imageKind, got, tt.wantSuffix)
			}

			expectedFormat := "docker.io/stenh0use/hind." + string(tt.imageKind)
			if got != expectedFormat {
				t.Errorf("constructName(%v) = %q, want %q", tt.imageKind, got, expectedFormat)
			}
		})
	}
}

func TestBuilder_ImageConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		kind              release.ImageKind
		wantImageName     string
		wantBaseImagePull bool
	}{
		{
			name:              "consul uses debian base",
			kind:              release.Consul,
			wantImageName:     "consul",
			wantBaseImagePull: true,
		},
		{
			name:              "nomad depends on consul",
			kind:              release.Nomad,
			wantImageName:     "nomad",
			wantBaseImagePull: false,
		},
		{
			name:              "nomad-client depends on nomad",
			kind:              release.NomadClient,
			wantImageName:     "nomad-client",
			wantBaseImagePull: false,
		},
		{
			name:              "vault depends on consul",
			kind:              release.Vault,
			wantImageName:     "vault",
			wantBaseImagePull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewBuilder(newTestLogger(), newStubClient(), tt.kind)
			if err != nil {
				t.Fatalf("NewBuilder(%v) unexpected error: %v", tt.kind, err)
			}

			if builder.image.Name != tt.wantImageName {
				t.Errorf("Builder.image.Name = %q, want %q", builder.image.Name, tt.wantImageName)
			}
			if builder.image.BaseImage.Pull != tt.wantBaseImagePull {
				t.Errorf("Builder.image.BaseImage.Pull = %v, want %v", builder.image.BaseImage.Pull, tt.wantBaseImagePull)
			}
			if len(builder.image.Packages) == 0 {
				t.Errorf("Builder.image.Packages is empty, want non-empty package list")
			}
		})
	}
}

func TestBuilder_CheckDependencies_CallsProviderTagExists(t *testing.T) {
	// nomad has BaseImage.Pull=false, so checkDependencies should call TagExists.
	stub := &providerStub{
		tagExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil // simulate missing base image
		},
	}

	builder, err := NewBuilder(newTestLogger(), stub, release.Nomad)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	err = builder.checkDependencies(context.Background())
	if err == nil {
		t.Fatal("expected error when base image is absent, got nil")
	}
	if !strings.Contains(err.Error(), "base image dependency not met") {
		t.Errorf("error should contain 'base image dependency not met', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Resolution: Run 'hind build") {
		t.Errorf("error should contain resolution hint, got: %q", err.Error())
	}
}

func TestBuilder_CheckDependencies_SkipsWhenPull(t *testing.T) {
	// consul has BaseImage.Pull=true — TagExists must never be called.
	stub := &providerStub{
		tagExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			panic("TagExists must not be called when BaseImage.Pull is true")
		},
	}

	builder, err := NewBuilder(newTestLogger(), stub, release.Consul)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	if err := builder.checkDependencies(context.Background()); err != nil {
		t.Errorf("checkDependencies should return nil for pull=true image, got: %v", err)
	}
}
