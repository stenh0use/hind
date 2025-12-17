package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/build/image/files"
	"github.com/stenh0use/hind/pkg/build/image/internal/docker"
	"github.com/stenh0use/hind/pkg/build/release"
)

type Builder struct {
	logger *log.Logger
	image  Image
}

func NewBuilder(logger *log.Logger, kind release.ImageKind) (*Builder, error) {
	image, err := NewImage(kind)
	if err != nil {
		return nil, fmt.Errorf("failed to create image definition: %w", err)
	}

	return &Builder{
		logger: logger,
		image:  image,
	}, nil
}

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

	imageName := b.image.Kind.ImageName()
	dockerImg := docker.NewImage(b.logger, imageName, b.image.Release)

	buildArgs, err := b.image.buildArgs()
	if err != nil {
		return fmt.Errorf("failed to generate build args: %w", err)
	}

	dockerImg.UpdateBuildOptions(
		&docker.BuildOptions{
			ContextDir: buildFiles.BuildDir(),
			BuildArgs:  buildArgs,
		})

	_, err = dockerImg.BuildImage(ctx)
	if err != nil {
		return fmt.Errorf("failed to build image %s: %w", b.image.Kind, err)
	}

	b.logger.WithField("image", fmt.Sprintf("%s:%s", b.image.Name, b.image.Release)).
		Info("Successfully built image")
	return nil
}

// checkDependencies implements feature requirement for dependency validation
func (b *Builder) checkDependencies(ctx context.Context) error {
	if b.image.BaseImage.Pull {
		// Base image is from registry (e.g., debian:bullseye-slim), no local dependency
		return nil
	}

	sanitizedName, _ := strings.CutPrefix(b.image.BaseImage.Name, release.ImageRegistry+"/")

	i := docker.NewImage(b.logger, sanitizedName, b.image.BaseImage.Tag)

	exists, err := i.TagExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check tag exists: %w", err)
	}

	component, _ := strings.CutPrefix(sanitizedName, release.ImageRepo+"/"+release.ImageNamePrefix)

	if !exists {
		return fmt.Errorf("base image dependency not met: %s\n"+
			"Resolution: Run 'hind build %s' to build the required dependency",
			sanitizedName, component)
	}

	return nil
}
