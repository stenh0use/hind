package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/build/image/files"
	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/provider"
)

// Builder orchestrates building a single hind Docker image via a provider.Client.
type Builder struct {
	logger *log.Logger
	client provider.Client
	image  Image
}

// NewBuilder constructs a Builder for the given image kind, using the provided
// provider.Client for all runtime Docker interactions.
func NewBuilder(logger *log.Logger, client provider.Client, kind ImageKind) (*Builder, error) {
	return NewBuilderWithRelease(logger, client, kind, release.Versions())
}

// NewBuilderWithRelease constructs a Builder for the given image kind and
// explicit release metadata.
func NewBuilderWithRelease(logger *log.Logger, client provider.Client, kind ImageKind, rel release.Packages) (*Builder, error) {
	image, err := NewImageWithRelease(kind, rel)
	if err != nil {
		return nil, fmt.Errorf("failed to create image definition: %w", err)
	}

	return &Builder{
		logger: logger,
		client: client,
		image:  image,
	}, nil
}

// BuildImage builds the image, writing build files to a temporary directory and
// delegating the Docker build to the provider client.
func (b *Builder) BuildImage(ctx context.Context) error {
	if err := b.checkDependencies(ctx); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	buildFiles, err := files.New(b.image.Kind.String())
	if err != nil {
		return fmt.Errorf("failed to create build files: %w", err)
	}

	if err := buildFiles.WriteFiles(); err != nil {
		return fmt.Errorf("failed to write build files for %s: %w", b.image.Kind, err)
	}

	buildArgs, err := b.image.buildArgs()
	if err != nil {
		return fmt.Errorf("failed to generate build args: %w", err)
	}

	result, err := b.client.BuildImage(ctx, provider.BuildImageOptions{
		Name:       b.image.Kind.ImageName(),
		Tag:        b.image.Release,
		ContextDir: buildFiles.BuildDir(),
		BuildArgs:  buildArgs,
		WithCache:  false,
		Platform:   "",
	})
	if err != nil {
		return fmt.Errorf("failed to build image %s: %w", b.image.Kind, err)
	}

	b.logger.WithField("image", result.ImageRef).Info("Successfully built image")
	return nil
}

// checkDependencies verifies that required base images exist locally before building.
func (b *Builder) checkDependencies(ctx context.Context) error {
	if b.image.BaseImage.Pull {
		// Base image is from registry (e.g., debian:bullseye-slim), no local dependency.
		return nil
	}

	sanitizedName, _ := strings.CutPrefix(b.image.BaseImage.Name, ImageRegistry+"/")

	exists, err := b.client.TagExists(ctx, sanitizedName, b.image.BaseImage.Tag)
	if err != nil {
		return fmt.Errorf("failed to check tag exists: %w", err)
	}

	component, _ := strings.CutPrefix(sanitizedName, ImageRepo+"/"+ImageNamePrefix)

	if !exists {
		return fmt.Errorf("base image dependency not met: %s\n"+
			"Resolution: Run 'hind build %s' to build the required dependency",
			sanitizedName, component)
	}

	return nil
}
