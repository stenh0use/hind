package provider

type BuildImageOptions struct {
	Name       string
	Tag        string
	ContextDir string
	BuildArgs  map[string]string
}
