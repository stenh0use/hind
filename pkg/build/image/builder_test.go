package image

import (
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/build/release"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name    string
		kind    release.ImageKind
		wantErr bool
	}{
		{
			name:    "valid consul image",
			kind:    release.Consul,
			wantErr: false,
		},
		{
			name:    "valid nomad image",
			kind:    release.Nomad,
			wantErr: false,
		},
		{
			name:    "valid nomad-client image",
			kind:    release.NomadClient,
			wantErr: false,
		},
		{
			name:    "valid vault image",
			kind:    release.Vault,
			wantErr: false,
		},
		{
			name:    "invalid image kind",
			kind:    release.ImageKind("invalid"),
			wantErr: true,
		},
		{
			name:    "empty image kind",
			kind:    release.ImageKind(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &log.Logger{Handler: discard.New()}
			got, err := NewBuilder(logger, tt.kind)

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

			// Verify full format: registry/repo/prefix.kind
			expectedFormat := "docker.io/stenh0use/hind." + string(tt.imageKind)
			if got != expectedFormat {
				t.Errorf("constructName(%v) = %q, want %q", tt.imageKind, got, expectedFormat)
			}
		})
	}
}

func TestBuilder_ImageConfiguration(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}

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
			wantBaseImagePull: true, // Pulls from registry
		},
		{
			name:              "nomad depends on consul",
			kind:              release.Nomad,
			wantImageName:     "nomad",
			wantBaseImagePull: false, // Uses local consul image
		},
		{
			name:              "nomad-client depends on nomad",
			kind:              release.NomadClient,
			wantImageName:     "nomad-client",
			wantBaseImagePull: false, // Uses local nomad image
		},
		{
			name:              "vault depends on consul",
			kind:              release.Vault,
			wantImageName:     "vault",
			wantBaseImagePull: false, // Uses local consul image
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewBuilder(logger, tt.kind)
			if err != nil {
				t.Fatalf("NewBuilder(%v) unexpected error: %v", tt.kind, err)
			}

			if builder.image.Name != tt.wantImageName {
				t.Errorf("Builder.image.Name = %q, want %q", builder.image.Name, tt.wantImageName)
			}

			if builder.image.BaseImage.Pull != tt.wantBaseImagePull {
				t.Errorf("Builder.image.BaseImage.Pull = %v, want %v", builder.image.BaseImage.Pull, tt.wantBaseImagePull)
			}

			// Verify packages are set
			if len(builder.image.Packages) == 0 {
				t.Errorf("Builder.image.Packages is empty, want non-empty package list")
			}
		})
	}
}
