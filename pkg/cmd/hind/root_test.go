package hind

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cmd"
)

func TestRootCommandUnknownSubcommand(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	streams := cmd.IOStreams{In: strings.NewReader(""), Out: stdout, ErrOut: stderr}

	root := NewCommand(logger, streams)
	root.SetArgs([]string{"releases"})

	err := root.Execute()
	require.Error(t, err, "expected error for unknown subcommand releases")
	assert.Contains(t, err.Error(), "unknown command")
	assert.Contains(t, err.Error(), "releases")
}
