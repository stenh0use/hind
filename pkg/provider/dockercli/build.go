package dockercli

import "context"

type BuildOpts struct{}

func (c *Client) BuildImage(ctx context.Context, opts BuildOpts) {}
