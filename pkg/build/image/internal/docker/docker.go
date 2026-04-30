// Package docker provides Docker CLI integration for building and managing images.
// It wraps Docker buildx commands and provides utilities for checking Docker daemon
// capabilities and installed plugins.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
)

const (
	defaultBuilder   string = "buildx"
	metadataFileName string = "metadata.json"
)

// Image holds options for building and running a Docker image using the Docker CLI.
type Image struct {
	Name         string          // Name of the image to build
	Tag          string          // Tag part of Name:tag for the built image
	logger       *log.Logger     // Logger for build output
	BuildOptions *BuildOptions   // Options for building the image (nil if not building)
	metadata     *BuildMetadata  // Cached metadata about built image
	executor     CommandExecutor // Command execution seam for tests
}

// CommandExecutor abstracts command execution for Docker operations.
type CommandExecutor interface {
	Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error
	Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
	CommandString(name string, args ...string) string
}

type osCommandExecutor struct{}

func (osCommandExecutor) Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (osCommandExecutor) Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func (osCommandExecutor) CommandString(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	return cmd.String()
}

var defaultCommandExecutor CommandExecutor = osCommandExecutor{}

func (i *Image) getExecutor() CommandExecutor {
	if i.executor != nil {
		return i.executor
	}
	return defaultCommandExecutor
}

func commandToString(name string, args ...string) string {
	return defaultCommandExecutor.CommandString(name, args...)
}

func outputWithExecutor(ctx context.Context, executor CommandExecutor, dir string, name string, args ...string) ([]byte, error) {
	if executor == nil {
		executor = defaultCommandExecutor
	}
	return executor.Output(ctx, dir, name, args...)
}

func checkDependenciesWithExecutor(ctx context.Context, executor CommandExecutor) error {
	if executor == nil {
		executor = defaultCommandExecutor
	}

	raw, err := outputWithExecutor(ctx, executor, "", "docker", "system", "info", "--format", "{{json .}}")
	if err != nil {
		return fmt.Errorf("failed to get docker system info: %w", err)
	}

	info := DockerInfo{}
	if err := json.Unmarshal(raw, &info); err != nil {
		return fmt.Errorf("failed to parse docker system info: %w", err)
	}

	if !info.HasClientPlugin(defaultBuilder) {
		return fmt.Errorf("%s client plugin is needed but not installed", defaultBuilder)
	}

	return nil
}

func runWithExecutor(ctx context.Context, executor CommandExecutor, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	if executor == nil {
		executor = defaultCommandExecutor
	}
	return executor.Run(ctx, dir, stdout, stderr, name, args...)
}

func runAndCapture(ctx context.Context, executor CommandExecutor, dir string, name string, args ...string) (string, string, error) {
	var stdout, stderr strings.Builder
	err := runWithExecutor(ctx, executor, dir, &stdout, &stderr, name, args...)
	return stdout.String(), stderr.String(), err
}

func (i *Image) buildCommandArgs() []string {
	args := []string{
		"buildx",
		"build",
		"-t", i.imageRef(),
		"--metadata-file", metadataFileName,
	}

	if i.BuildOptions.Dockerfile != "" {
		args = append(args, "-f", i.BuildOptions.Dockerfile)
	}

	if !i.BuildOptions.WithCache {
		args = append(args, "--no-cache")
	}

	if i.BuildOptions.Platform != "" {
		args = append(args, "--platform", i.BuildOptions.Platform)
	}

	args = append(args, i.FormatBuildArgs()...)
	args = append(args, ".")
	return args
}

func (i *Image) buildCommandString() string {
	return commandToString("docker", i.buildCommandArgs()...)
}

func (i *Image) runBuildCommand(ctx context.Context, executor CommandExecutor) (string, string, error) {
	return runAndCapture(ctx, executor, i.BuildOptions.ContextDir, "docker", i.buildCommandArgs()...)
}

func (i *Image) runTagExistsCommand(ctx context.Context, executor CommandExecutor) (string, string, error) {
	return runAndCapture(ctx, executor, "", "docker", "images", "-q", i.imageRef())
}

func (i *Image) checkDependencies(ctx context.Context) error {
	return checkDependenciesWithExecutor(ctx, defaultCommandExecutor)
}

func (i *Image) executeBuild(ctx context.Context, executor CommandExecutor) (string, error) {
	stdout, stderr, err := i.runBuildCommand(ctx, executor)
	if err != nil {
		i.logger.WithFields(log.Fields{"stdout": stdout, "stderr": stderr, "error": err}).Debug("failed to build image")
		return "", fmt.Errorf("failed to build image: %w: %s", err, stderr)
	}
	return stdout, nil
}

func (i *Image) executeTagExists(ctx context.Context, executor CommandExecutor) (bool, error) {
	stdout, stderr, err := i.runTagExistsCommand(ctx, executor)
	if err != nil {
		return false, fmt.Errorf("failed to check if tag exists: %w: %s", err, stderr)
	}
	return strings.TrimSpace(stdout) != "", nil
}

func (i *Image) logBuildStart() {
	i.logger.WithFields(log.Fields{"name": i.Name, "tag": i.Tag}).Info("Building image")
}

func (i *Image) logBuildCommand() {
	i.logger.WithField("command", i.buildCommandString()).Debug("Running Docker build command")
}

func (i *Image) logBuildSuccess() {
	i.logger.WithFields(log.Fields{"name": i.Name, "tag": i.Tag}).Info("Successfully built image")
}

func (i *Image) buildAndResolveDigest(ctx context.Context, executor CommandExecutor) (string, error) {
	if _, err := i.executeBuild(ctx, executor); err != nil {
		return "", err
	}
	return i.getImageDigest(ctx)
}

func (i *Image) verifyBuildPreconditions(ctx context.Context, executor CommandExecutor) error {
	if err := i.checkDependencies(ctx); err != nil {
		return fmt.Errorf("failed to build image %s:%s: %w", i.Name, i.Tag, err)
	}
	if i.BuildOptions == nil {
		return fmt.Errorf("build options not set: cannot build image")
	}
	return nil
}

func (i *Image) buildImageWithExecutor(ctx context.Context, executor CommandExecutor) (string, error) {
	if err := i.verifyBuildPreconditions(ctx, executor); err != nil {
		return "", err
	}
	i.logBuildStart()
	i.logBuildCommand()
	digest, err := i.buildAndResolveDigest(ctx, executor)
	if err != nil {
		return "", err
	}
	i.logBuildSuccess()
	return digest, nil
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

func (i *Image) metadataFilePath() string {
	return filepath.Join(i.BuildOptions.ContextDir, metadataFileName)
}

// RefreshBuildMetadata reads and parses the metadata.json file from disk, updating the cache
func (i *Image) RefreshBuildMetadata(ctx context.Context) (*BuildMetadata, error) {
	if i.BuildOptions == nil {
		return nil, fmt.Errorf("build options not set: cannot read metadata file")
	}

	metadataFile := i.metadataFilePath()
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
	return i.buildImageWithExecutor(ctx, i.getExecutor())
}

// imageRef constructs the full image name
func (i *Image) imageRef() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Tag)
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
	return i.executeTagExists(ctx, i.getExecutor())
}

func checkDependencies(ctx context.Context) error {
	return checkDependenciesWithExecutor(ctx, defaultCommandExecutor)
}
