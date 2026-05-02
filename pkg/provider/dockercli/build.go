package dockercli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stenh0use/hind/pkg/provider"
)

const metadataFileName = "metadata.json"

// buildMetadata holds the parsed content of the docker buildx metadata.json file.
type buildMetadata struct {
	ContainerImageDigest string `json:"containerimage.config.digest"`
	ImageName            string `json:"image.name"`
}

// BuildImage builds a Docker image using buildx. It checks for buildx availability
// at call time and returns a structured result with the image digest and ref.
func (c *Client) BuildImage(ctx context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
	if opts.Name == "" {
		return provider.BuildImageResult{}, fmt.Errorf("image name is required")
	}
	if opts.Tag == "" {
		return provider.BuildImageResult{}, fmt.Errorf("image tag is required")
	}
	if opts.ContextDir == "" {
		return provider.BuildImageResult{}, fmt.Errorf("build context dir is required")
	}

	// Check buildx availability at call time, not at construction.
	if err := checkBuildxAvailable(ctx, c.executor); err != nil {
		return provider.BuildImageResult{}, err
	}

	args := c.buildxArgs(opts)

	var stdout, stderr strings.Builder
	if err := c.executor.Run(ctx, opts.ContextDir, &stdout, &stderr, "docker", args...); err != nil {
		return provider.BuildImageResult{}, fmt.Errorf("failed to build image: %w: %s", err, stderr.String())
	}

	digest, err := readDigestFromMetadata(filepath.Join(opts.ContextDir, metadataFileName))
	if err != nil {
		return provider.BuildImageResult{}, err
	}

	imageRef := fmt.Sprintf("%s:%s", opts.Name, opts.Tag)
	if imageRef == "" {
		return provider.BuildImageResult{}, fmt.Errorf("imageRef is empty")
	}

	return provider.BuildImageResult{
		Digest:   digest,
		ImageRef: imageRef,
	}, nil
}

// buildxArgs constructs the argument list for `docker buildx build ...`.
// --load is always included so the image is loaded into the local Docker image store.
func (c *Client) buildxArgs(opts provider.BuildImageOptions) []string {
	args := []string{
		"buildx",
		"build",
		"-t", fmt.Sprintf("%s:%s", opts.Name, opts.Tag),
		"--load",
		"--metadata-file", metadataFileName,
	}

	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}

	if !opts.WithCache {
		args = append(args, "--no-cache")
	}

	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}

	for k, v := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, ".")
	return args
}

// readDigestFromMetadata reads and parses metadata.json written by docker buildx.
// Returns an error if the digest is absent or does not begin with "sha256:".
func readDigestFromMetadata(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata file %s: %w", path, err)
	}

	var m buildMetadata
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("failed to parse metadata file %s: %w", path, err)
	}

	if m.ContainerImageDigest == "" {
		return "", fmt.Errorf("metadata file %s contains an empty digest", path)
	}
	if !strings.HasPrefix(m.ContainerImageDigest, "sha256:") {
		return "", fmt.Errorf("unexpected digest format in %s: %q", path, m.ContainerImageDigest)
	}

	return m.ContainerImageDigest, nil
}

// TagExists reports whether the given image name:tag exists in the local Docker image store.
func (c *Client) TagExists(ctx context.Context, name string, tag string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("image name is required")
	}
	if tag == "" {
		return false, fmt.Errorf("image tag is required")
	}

	cmd := baseClientCmd(ctx)
	cmd.Args = append(cmd.Args, "image", "ls", fmt.Sprintf("%s:%s", name, tag), "--format", "{{ .ID }}")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to inspect image tags: %w", err)
	}

	return strings.TrimSpace(string(out)) != "", nil
}

// PullImage pulls an image from a registry.
func (c *Client) PullImage(ctx context.Context, name string, tag string) error {
	if name == "" {
		return fmt.Errorf("image name is required")
	}
	if tag == "" {
		return fmt.Errorf("image tag is required")
	}

	cmd := baseClientCmd(ctx)
	cmd.Args = append(cmd.Args, "pull", fmt.Sprintf("%s:%s", name, tag))
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return nil
}
