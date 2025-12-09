// Package docker provides Docker CLI integration for building and managing images.
// It wraps Docker buildx commands and provides utilities for checking Docker daemon
// capabilities and installed plugins.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
)

const defaultBuilder string = "buildx"

// Image holds options for building and running a Docker image using the Docker CLI.
type Image struct {
	Name         string         // Name of the image to build
	Tag          string         // Tag part of Name:tag for the built image
	logger       *log.Logger    // Logger for build output
	BuildOptions *BuildOptions  // Options for building the image (nil if not building)
	metadata     *BuildMetadata // Cached metadata about built image
}

type BuildOptions struct {
	ContextDir string
	Dockerfile string
	BuildArgs  []BuildArg
	WithCache  bool   // Whether to use the build cache
	Platform   string // Optional platform to build for
}

// BuildMetadata is extracted from the docker buildx metadata.json
type BuildMetadata struct {
	ContainerImageDigest string `json:"containerimage.config.digest"`
	ImageName            string `json:"image.name"`
}

type BuildArg struct {
	Arg   string
	Value string
}

func NewImage(logger *log.Logger, name, tag string) Image {
	return Image{
		logger: logger,
		Name:   name,
		Tag:    tag,
	}
}

func (i *Image) UpdateBuildOptions(opts *BuildOptions) {
	if i.BuildOptions == nil {
		i.BuildOptions = opts
		return
	}

	if opts.ContextDir != "" {
		i.BuildOptions.ContextDir = opts.ContextDir
	}
	if opts.Dockerfile != "" {
		i.BuildOptions.Dockerfile = opts.Dockerfile
	}
	i.BuildOptions.WithCache = opts.WithCache
	if opts.Platform != "" {
		i.BuildOptions.Platform = opts.Platform
	}
	if opts.BuildArgs != nil {
		i.BuildOptions.BuildArgs = opts.BuildArgs
	}
}

func (i *Image) FormatBuildArgs() []string {
	if i.BuildOptions == nil || i.BuildOptions.BuildArgs == nil {
		return []string{}
	}

	args := make([]string, 0, len(i.BuildOptions.BuildArgs))
	for _, v := range i.BuildOptions.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", v.Arg, v.Value))
	}

	return args
}

// RefreshBuildMetadata reads and parses the metadata.json file from disk, updating the cache
func (i *Image) RefreshBuildMetadata(ctx context.Context) (*BuildMetadata, error) {
	if i.BuildOptions == nil {
		return nil, fmt.Errorf("build options not set: cannot read metadata file")
	}

	metadataFile := i.BuildOptions.ContextDir + "/metadata.json"
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file %s: %w", metadataFile, err)
	}

	var metadata BuildMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from %s: %w", metadataFile, err)
	}

	// Cache the metadata for future calls
	i.metadata = &metadata
	return i.metadata, nil
}

// GetBuildMetadata returns cached metadata, loading from file if not already cached
func (i *Image) GetBuildMetadata(ctx context.Context) (*BuildMetadata, error) {
	// Return cached metadata if available
	if i.metadata != nil {
		return i.metadata, nil
	}

	// Load from file and cache
	return i.RefreshBuildMetadata(ctx)
}

func (i *Image) BuildImage(ctx context.Context) (string, error) {
	if err := checkDependencies(ctx); err != nil {
		return "", fmt.Errorf("failed to build image %s:%s: %w", i.Name, i.Tag, err)
	}

	if i.BuildOptions == nil {
		return "", fmt.Errorf("build options not set: cannot build image")
	}

	i.logger.WithFields(log.Fields{"name": i.Name, "tag": i.Tag}).Info("Building image")

	cmd := i.buildCommand(ctx)

	i.logger.WithField("command", cmd.String()).Debug("Running Docker build command")

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		i.logger.WithFields(log.Fields{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
			"error":  err,
		}).Debug("failed to build image")
		return "", fmt.Errorf("failed to build image: %w: %s", err, stderr.String())
	}

	i.logger.WithFields(log.Fields{"name": i.Name, "tag": i.Tag}).Info("Successfully built image")

	return i.getImageDigest(ctx)
}

// imageRef constructs the full image name
func (i *Image) imageRef() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Tag)
}

// buildCommand creates the docker buildx command with all options
func (i *Image) buildCommand(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"buildx",
		"build",
		"-t", i.imageRef(),
		"--metadata-file", "metadata.json",
	)

	cmd.Dir = i.BuildOptions.ContextDir

	if i.BuildOptions.Dockerfile != "" {
		cmd.Args = append(cmd.Args, "-f", i.BuildOptions.Dockerfile)
	}

	if !i.BuildOptions.WithCache {
		cmd.Args = append(cmd.Args, "--no-cache")
	}

	if i.BuildOptions.Platform != "" {
		cmd.Args = append(cmd.Args, "--platform", i.BuildOptions.Platform)
	}

	cmd.Args = append(cmd.Args, i.FormatBuildArgs()...)
	cmd.Args = append(cmd.Args, ".")

	return cmd
}

// getImageDigest retrieves and logs the built image digest
func (i *Image) getImageDigest(ctx context.Context) (string, error) {
	imageMeta, err := i.GetBuildMetadata(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read image ID from metadata: %w", err)
	}

	i.logger.WithField("imageMeta", imageMeta).Info("Image metadata")
	return imageMeta.ContainerImageDigest, nil
}

func (i *Image) TagExists(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", i.imageRef())
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check if tag exists: %w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()) != "", nil
}

func checkDependencies(ctx context.Context) error {
	info := DockerInfo{}
	if err := info.Get(ctx); err != nil {
		return fmt.Errorf("failed to get docker system info: %w", err)
	}

	if !info.HasClientPlugin(defaultBuilder) {
		return fmt.Errorf("%s client plugin is needed but not installed", defaultBuilder)
	}

	// This is only required for multi platform builds
	// const snapshotter = "io.containerd.snapshotter.v1"
	// if !info.HasDriverType(snapshotter) {
	// 	return fmt.Errorf("'%s' driver is needed but not configured", snapshotter)
	// }

	return nil
}
