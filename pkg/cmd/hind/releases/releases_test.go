package releases

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/cmd"
)

// testStreams returns a logger and IOStreams whose stdout is captured in the
// returned buffer.
func testStreams() (*log.Logger, cmd.IOStreams, *bytes.Buffer) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var buf bytes.Buffer
	streams := cmd.IOStreams{
		In:     strings.NewReader(""),
		Out:    &buf,
		ErrOut: io.Discard,
	}
	return logger, streams, &buf
}

// TestRunE_HeaderRow asserts that the first output line contains the four
// required column labels and that they appear in alphabetical order after HIND.
func TestRunE_HeaderRow(t *testing.T) {
	t.Parallel()

	logger, streams, buf := testStreams()

	if err := runE(context.Background(), logger, streams); err != nil {
		t.Fatalf("runE() returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("runE() produced no output")
	}

	header := lines[0]
	for _, col := range []string{"HIND", "CONSUL", "NOMAD", "VAULT"} {
		if !strings.Contains(header, col) {
			t.Errorf("header row missing column %q; got: %q", col, header)
		}
	}

	// Confirm alphabetical column order after HIND:
	// CONSUL must precede NOMAD, NOMAD must precede VAULT.
	idxConsul := strings.Index(header, "CONSUL")
	idxNomad := strings.Index(header, "NOMAD")
	idxVault := strings.Index(header, "VAULT")

	if idxConsul >= idxNomad {
		t.Errorf("expected CONSUL before NOMAD in header; got: %q", header)
	}
	if idxNomad >= idxVault {
		t.Errorf("expected NOMAD before VAULT in header; got: %q", header)
	}
}

// TestRunE_AlphabeticalColumnOrder verifies that HIND is the leftmost column
// (first field in the header row) and that CONSUL < NOMAD < VAULT follow it.
func TestRunE_AlphabeticalColumnOrder(t *testing.T) {
	t.Parallel()

	logger, streams, buf := testStreams()

	if err := runE(context.Background(), logger, streams); err != nil {
		t.Fatalf("runE() returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("runE() produced no output")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 4 {
		t.Fatalf("expected at least 4 header fields, got %d: %v", len(fields), fields)
	}

	if fields[0] != "HIND" {
		t.Errorf("first column must be HIND; got %q", fields[0])
	}
	if fields[1] != "CONSUL" {
		t.Errorf("second column must be CONSUL; got %q", fields[1])
	}
	if fields[2] != "NOMAD" {
		t.Errorf("third column must be NOMAD; got %q", fields[2])
	}
	if fields[3] != "VAULT" {
		t.Errorf("fourth column must be VAULT; got %q", fields[3])
	}
}

// TestRunE_LatestVersionFirstRow asserts that the first data row (line after
// the header) starts with the latest hind version from release.Latest().
func TestRunE_LatestVersionFirstRow(t *testing.T) {
	t.Parallel()

	logger, streams, buf := testStreams()

	if err := runE(context.Background(), logger, streams); err != nil {
		t.Fatalf("runE() returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// lines[0] is the header; lines[1] is the first data row.
	if len(lines) < 2 {
		t.Fatalf("expected at least a header and one data row, got %d line(s)", len(lines))
	}

	latest := release.Latest().Hind
	firstDataRow := lines[1]
	fields := strings.Fields(firstDataRow)
	if len(fields) < 1 {
		t.Fatalf("first data row is empty")
	}

	if fields[0] != latest {
		t.Errorf("expected first data row to start with latest version %q; got %q", latest, fields[0])
	}
}

// TestNewCommand_Structure asserts that the command is wired correctly so that
// it is reachable as "hind releases".
func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	logger, streams, _ := testStreams()
	c := NewCommand(logger, streams)

	if c == nil {
		t.Fatal("NewCommand() returned nil")
	}
	if c.Use != "releases" {
		t.Errorf("expected Use=%q, got %q", "releases", c.Use)
	}
	if c.Args == nil {
		t.Error("expected Args validator to be set (NoArgs)")
	}
	if c.RunE == nil {
		t.Error("expected RunE to be set")
	}
}
