package provider

// BuildImageOptions holds the parameters for building a Docker image.
type BuildImageOptions struct {
	Name       string
	Tag        string
	ContextDir string
	BuildArgs  map[string]string
	Dockerfile string // optional; empty means default Dockerfile
	WithCache  bool   // pass --no-cache when false
	Platform   string // optional; empty means omit --platform — do not supply a default
}

// BuildImageResult is the structured result of a successful image build.
// Both fields are non-empty on success; an empty Digest or ImageRef is an error condition.
type BuildImageResult struct {
	Digest   string // sha256 digest, e.g. "sha256:abc123..."
	ImageRef string // fully qualified ref, e.g. "hind/consul:0.1.0"
}
