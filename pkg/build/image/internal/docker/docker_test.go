package docker

import (
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
)

func TestFormatBuildArgs(t *testing.T) {
	tests := []struct {
		name string
		args []BuildArg
		want []string
	}{
		{
			name: "no build args",
			args: nil,
			want: []string{},
		},
		{
			name: "empty build args",
			args: []BuildArg{},
			want: []string{},
		},
		{
			name: "single build arg",
			args: []BuildArg{
				{Arg: "VERSION", Value: "1.0"},
			},
			want: []string{"--build-arg", "VERSION=1.0"},
		},
		{
			name: "multiple build args",
			args: []BuildArg{
				{Arg: "VERSION", Value: "1.0"},
				{Arg: "BASE", Value: "alpine"},
			},
			want: []string{
				"--build-arg", "VERSION=1.0",
				"--build-arg", "BASE=alpine",
			},
		},
		{
			name: "build args with special characters",
			args: []BuildArg{
				{Arg: "URL", Value: "https://example.com/path?query=value"},
				{Arg: "MESSAGE", Value: "hello world"},
			},
			want: []string{
				"--build-arg", "URL=https://example.com/path?query=value",
				"--build-arg", "MESSAGE=hello world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &log.Logger{Handler: discard.New()}
			img := NewImage(logger, "test", "latest")
			img.UpdateBuildOptions(&BuildOptions{
				BuildArgs: tt.args,
			})

			got := img.FormatBuildArgs()

			if len(got) != len(tt.want) {
				t.Errorf("FormatBuildArgs() length = %d, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FormatBuildArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUpdateBuildOptions(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}

	t.Run("set options when nil", func(t *testing.T) {
		img := NewImage(logger, "test", "latest")

		if img.BuildOptions != nil {
			t.Fatal("Expected BuildOptions to be nil initially")
		}

		opts := &BuildOptions{
			ContextDir: "/build",
			Dockerfile: "Dockerfile",
			BuildArgs: []BuildArg{
				{Arg: "VERSION", Value: "1.0"},
			},
		}

		img.UpdateBuildOptions(opts)

		if img.BuildOptions == nil {
			t.Fatal("BuildOptions should not be nil after update")
		}

		if img.BuildOptions.ContextDir != "/build" {
			t.Errorf("ContextDir = %q, want %q", img.BuildOptions.ContextDir, "/build")
		}
	})

	t.Run("merge non-empty values", func(t *testing.T) {
		img := NewImage(logger, "test", "latest")
		img.BuildOptions = &BuildOptions{
			ContextDir: "/original",
			Dockerfile: "Dockerfile.original",
		}

		img.UpdateBuildOptions(&BuildOptions{
			ContextDir: "/updated",
			// Dockerfile intentionally empty - should not override
		})

		if img.BuildOptions.ContextDir != "/updated" {
			t.Errorf("ContextDir = %q, want %q", img.BuildOptions.ContextDir, "/updated")
		}

		if img.BuildOptions.Dockerfile != "Dockerfile.original" {
			t.Errorf("Dockerfile = %q, want %q (should not override with empty)",
				img.BuildOptions.Dockerfile, "Dockerfile.original")
		}
	})

	t.Run("update build args", func(t *testing.T) {
		img := NewImage(logger, "test", "latest")
		img.BuildOptions = &BuildOptions{
			BuildArgs: []BuildArg{
				{Arg: "OLD", Value: "value"},
			},
		}

		newArgs := []BuildArg{
			{Arg: "NEW", Value: "value"},
		}

		img.UpdateBuildOptions(&BuildOptions{
			BuildArgs: newArgs,
		})

		if len(img.BuildOptions.BuildArgs) != 1 {
			t.Errorf("BuildArgs length = %d, want 1", len(img.BuildOptions.BuildArgs))
		}

		if img.BuildOptions.BuildArgs[0].Arg != "NEW" {
			t.Errorf("BuildArgs[0].Arg = %q, want %q",
				img.BuildOptions.BuildArgs[0].Arg, "NEW")
		}
	})
}

func TestImageRef(t *testing.T) {
	tests := []struct {
		name    string
		imgName string
		imgTag  string
		want    string
	}{
		{
			name:    "standard image ref",
			imgName: "myapp",
			imgTag:  "v1.0.0",
			want:    "myapp:v1.0.0",
		},
		{
			name:    "latest tag",
			imgName: "myapp",
			imgTag:  "latest",
			want:    "myapp:latest",
		},
		{
			name:    "image with registry",
			imgName: "docker.io/user/myapp",
			imgTag:  "sha256abc",
			want:    "docker.io/user/myapp:sha256abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &log.Logger{Handler: discard.New()}
			img := NewImage(logger, tt.imgName, tt.imgTag)

			got := img.imageRef()

			if got != tt.want {
				t.Errorf("imageRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewImage(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}

	t.Run("creates image with correct fields", func(t *testing.T) {
		name := "test-image"
		tag := "v1.0.0"

		img := NewImage(logger, name, tag)

		if img.Name != name {
			t.Errorf("Name = %q, want %q", img.Name, name)
		}

		if img.Tag != tag {
			t.Errorf("Tag = %q, want %q", img.Tag, tag)
		}

		if img.logger == nil {
			t.Error("Logger should not be nil")
		}

		if img.BuildOptions != nil {
			t.Error("BuildOptions should be nil initially")
		}
	})
}
