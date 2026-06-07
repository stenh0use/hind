package version

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHindVersion_NonEmpty(t *testing.T) {
	require.NotEmpty(t, HindVersion)
}

func TestHindVersion_SemverPrefix(t *testing.T) {
	// Prefix match (not full match) so future suffixes like "-dev" or "+meta" are allowed.
	re := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+`)
	require.True(t, re.MatchString(HindVersion), "HindVersion %q does not match semver prefix MAJOR.MINOR.PATCH", HindVersion)
}
