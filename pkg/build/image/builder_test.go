package image

import (
	"context"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

// newTestLogger returns a logger that discards all output.
func newTestLogger() *log.Logger {
	return &log.Logger{Handler: discard.New()}
}

// newStubClient returns a provider.Client test stub.
func newStubClient() *mock.ClientStub {
	return &mock.ClientStub{
		BuildImageFn: func(_ context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
			return provider.BuildImageResult{Digest: "sha256:stub", ImageRef: opts.Name + ":" + opts.Tag}, nil
		},
		TagExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
}

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name    string
		kind    ImageKind
		wantErr bool
	}{
		{name: "valid consul image", kind: Consul, wantErr: false},
		{name: "valid nomad image", kind: Nomad, wantErr: false},
		{name: "valid nomad-client image", kind: NomadClient, wantErr: false},
		{name: "valid vault image", kind: Vault, wantErr: false},
		{name: "invalid image kind", kind: ImageKind("invalid"), wantErr: true},
		{name: "empty image kind", kind: ImageKind(""), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLogger()
			got, err := NewBuilder(logger, newStubClient(), tt.kind)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.logger)
			assert.Equal(t, tt.kind, got.image.Kind)
		})
	}
}

func TestConstructName(t *testing.T) {
	tests := []struct {
		name       string
		imageKind  ImageKind
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "consul image name",
			imageKind:  Consul,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "consul",
		},
		{
			name:       "nomad image name",
			imageKind:  Nomad,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "nomad",
		},
		{
			name:       "nomad-client image name",
			imageKind:  NomadClient,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "nomad-client",
		},
		{
			name:       "vault image name",
			imageKind:  Vault,
			wantPrefix: "docker.io/stenh0use/hind.",
			wantSuffix: "vault",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.imageKind.ImageName()

			assert.True(t, strings.HasPrefix(got, tt.wantPrefix), "constructName(%v) = %q, want prefix %q", tt.imageKind, got, tt.wantPrefix)
			assert.True(t, strings.HasSuffix(got, tt.wantSuffix), "constructName(%v) = %q, want suffix %q", tt.imageKind, got, tt.wantSuffix)

			expectedFormat := "docker.io/stenh0use/hind." + string(tt.imageKind)
			assert.Equal(t, expectedFormat, got)
		})
	}
}

func TestBuilder_ImageConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		kind              ImageKind
		wantImageName     string
		wantBaseImagePull bool
	}{
		{
			name:              "consul uses debian base",
			kind:              Consul,
			wantImageName:     "consul",
			wantBaseImagePull: true,
		},
		{
			name:              "nomad depends on consul",
			kind:              Nomad,
			wantImageName:     "nomad",
			wantBaseImagePull: false,
		},
		{
			name:              "nomad-client depends on nomad",
			kind:              NomadClient,
			wantImageName:     "nomad-client",
			wantBaseImagePull: false,
		},
		{
			name:              "vault depends on consul",
			kind:              Vault,
			wantImageName:     "vault",
			wantBaseImagePull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewBuilder(newTestLogger(), newStubClient(), tt.kind)
			require.NoError(t, err)

			assert.Equal(t, tt.wantImageName, builder.image.Name)
			assert.Equal(t, tt.wantBaseImagePull, builder.image.BaseImage.Pull)
			assert.NotEmpty(t, builder.image.Packages)
		})
	}
}

func TestBuilder_CheckDependencies_CallsProviderTagExists(t *testing.T) {
	// nomad has BaseImage.Pull=false, so checkDependencies should call TagExists.
	stub := &mock.ClientStub{
		TagExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil // simulate missing base image
		},
		BuildImageFn: func(_ context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
			return provider.BuildImageResult{Digest: "sha256:stub", ImageRef: opts.Name + ":" + opts.Tag}, nil
		},
	}

	builder, err := NewBuilder(newTestLogger(), stub, Nomad)
	require.NoError(t, err)

	err = builder.checkDependencies(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base image dependency not met")
	assert.Contains(t, err.Error(), "Resolution: Run 'hind build")
}

func TestBuilder_BuildImage_CallsProviderBuildImage(t *testing.T) {
	var capturedOpts provider.BuildImageOptions

	stub := &mock.ClientStub{
		BuildImageFn: func(_ context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
			capturedOpts = opts
			return provider.BuildImageResult{Digest: "sha256:abc", ImageRef: "name:tag"}, nil
		},
	}

	// Use consul: BaseImage.Pull=true so checkDependencies skips TagExists.
	builder, err := NewBuilder(newTestLogger(), stub, Consul)
	require.NoError(t, err)

	err = builder.BuildImage(context.Background())
	require.NoError(t, err)

	expectedName := Consul.ImageName()
	assert.Equal(t, expectedName, capturedOpts.Name)
	assert.NotEmpty(t, capturedOpts.Tag)
	assert.NotEmpty(t, capturedOpts.ContextDir)
	assert.NotEmpty(t, capturedOpts.BuildArgs)
}

func TestBuilder_CheckDependencies_SkipsWhenPull(t *testing.T) {
	// consul has BaseImage.Pull=true — TagExists must never be called.
	stub := &mock.ClientStub{
		BuildImageFn: func(_ context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
			return provider.BuildImageResult{Digest: "sha256:stub", ImageRef: opts.Name + ":" + opts.Tag}, nil
		},
		TagExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			panic("TagExists must not be called when BaseImage.Pull is true")
		},
	}

	builder, err := NewBuilder(newTestLogger(), stub, Consul)
	require.NoError(t, err)

	err = builder.checkDependencies(context.Background())
	assert.NoError(t, err)
}
