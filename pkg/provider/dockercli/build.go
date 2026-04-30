package dockercli

import (
	"context"
	"fmt"
	"strings"

	"github.com/stenh0use/hind/pkg/provider"
)

func (c *Client) BuildImage(ctx context.Context, opts provider.BuildImageOptions) (string, error) {
	if opts.Name == "" {
		return "", fmt.Errorf("image name is required")
	}
	if opts.Tag == "" {
		return "", fmt.Errorf("image tag is required")
	}
	if opts.ContextDir == "" {
		return "", fmt.Errorf("build context dir is required")
	}

	cmd := baseClientCmd(ctx)
	cmd.Args = append(cmd.Args, "build")
	cmd.Args = append(cmd.Args, "--tag", fmt.Sprintf("%s:%s", opts.Name, opts.Tag))
	for k, v := range opts.BuildArgs {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Args = append(cmd.Args, opts.ContextDir)

	if _, err := cmd.Output(); err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	return "", nil
}

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
