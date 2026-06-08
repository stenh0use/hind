package fs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

func TestRepositoryRejectsTraversalNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	invalid := domain.Name("../escape")
	_, loadErr := r.Load(context.Background(), invalid)
	assert.True(t, loadErr != nil && strings.Contains(loadErr.Error(), "invalid cluster name"), "Load traversal error = %v, want invalid cluster name", loadErr)
	deleteErr := r.Delete(context.Background(), invalid)
	assert.True(t, deleteErr != nil && strings.Contains(deleteErr.Error(), "invalid cluster name"), "Delete traversal error = %v, want invalid cluster name", deleteErr)
}

func TestRepositorySaveIsAtomic(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	c := domain.Cluster{Name: "demo", Network: domain.Network{Name: "hind.demo"}}
	err = r.Save(context.Background(), c)
	require.NoError(t, err)

	root := r.fm.GetRootDir()
	final := filepath.Join(root, "cluster", "demo", "cluster.json")
	tmp := final + ".tmp"

	_, statErr := os.Stat(final)
	require.NoError(t, statErr, "final config missing")
	_, tmpStatErr := os.Stat(tmp)
	assert.True(t, errors.Is(tmpStatErr, os.ErrNotExist), "temp file should not exist after atomic rename")
}

func TestRepositorySaveWritesEnvelopeFormat(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	c := domain.Cluster{Name: "demo", Network: domain.Network{Name: "hind.demo"}}
	err = r.Save(context.Background(), c)
	require.NoError(t, err)

	path := filepath.Join(r.fm.GetRootDir(), "cluster", "demo", "cluster.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	contents := string(data)
	for _, token := range []string{"\"apiVersion\"", "\"kind\"", "\"metadata\"", "\"spec\""} {
		assert.Contains(t, contents, token, "saved format missing %s", token)
	}
}

func TestRepositoryLoadLegacyConfigCompatibility(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	legacy := []byte("{\n  \"name\": \"legacy\",\n  \"network\": {\n    \"name\": \"hind.legacy\"\n  }\n}")
	path := filepath.Join(r.fm.GetRootDir(), "cluster", "legacy", "cluster.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(path, legacy, 0o644)
	require.NoError(t, err)

	c, err := r.Load(context.Background(), domain.Name("legacy"))
	require.NoError(t, err)
	assert.Equal(t, domain.Name("legacy"), c.Name)
}

func TestRepositoryListReturnsSortedNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	for _, name := range []string{"zeta", "alpha", "mu"} {
		c := domain.Cluster{Name: domain.Name(name), Network: domain.Network{Name: domain.NetworkName("hind." + name)}}
		err = r.Save(context.Background(), c)
		require.NoError(t, err)
	}

	got, err := r.List(context.Background())
	require.NoError(t, err)
	want := []domain.Name{"alpha", "mu", "zeta"}
	assert.Equal(t, want, got)
}

func TestRepositoryGetActiveValidatesExistence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	active := filepath.Join(r.fm.GetRootDir(), "cluster", "active")
	err = os.MkdirAll(filepath.Dir(active), 0o755)
	require.NoError(t, err)

	t.Run("missing active file returns empty", func(t *testing.T) {
		got, err := r.GetActive(context.Background())
		require.NoError(t, err)
		assert.Empty(t, got, "GetActive() should return empty when active file missing")
	})

	t.Run("stale active cluster returns empty", func(t *testing.T) {
		err = os.WriteFile(active, []byte("missing"), 0o644)
		require.NoError(t, err)

		got, err := r.GetActive(context.Background())
		require.NoError(t, err)
		assert.Empty(t, got, "GetActive() should return empty for stale active")
	})
}

func TestRepositorySetActiveRejectsNonexistentCluster(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	err = r.SetActive(context.Background(), "missing")
	require.Error(t, err, "SetActive() error = nil, want not found")
	require.ErrorIs(t, err, ErrNotFound)

	active := filepath.Join(r.fm.GetRootDir(), "cluster", "active")
	_, statErr := os.Stat(active)
	assert.True(t, errors.Is(statErr, os.ErrNotExist), "active file should not exist after failed SetActive(), stat err = %v", statErr)
}

func TestRepositoryClearActiveBehaviorParity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	active := filepath.Join(r.fm.GetRootDir(), "cluster", "active")

	t.Run("clearing missing active file succeeds", func(t *testing.T) {
		err := r.ClearActive(context.Background())
		require.NoError(t, err)
	})

	t.Run("clearing existing active file removes it", func(t *testing.T) {
		c := domain.Cluster{Name: "demo", Network: domain.Network{Name: "hind.demo"}}
		err := r.Save(context.Background(), c)
		require.NoError(t, err)
		err = r.SetActive(context.Background(), "demo")
		require.NoError(t, err)

		err = r.ClearActive(context.Background())
		require.NoError(t, err)

		_, statErr := os.Stat(active)
		assert.True(t, errors.Is(statErr, os.ErrNotExist), "active file should be removed, stat err = %v", statErr)

		got, err := r.GetActive(context.Background())
		require.NoError(t, err)
		assert.Empty(t, got, "GetActive() should return empty after ClearActive")
	})
}

func TestRepositoryDeleteStaleDir(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    bool // create the cluster dir
		setupJSON   bool // create cluster.json inside it
		wantErr     bool
		wantErrIs   error // checked with errors.Is when wantErr is true
		wantDirGone bool  // dir must not exist after Delete
	}{
		{
			name:        "stale dir without cluster.json is removed silently",
			setupDir:    true,
			setupJSON:   false,
			wantErr:     false,
			wantDirGone: true,
		},
		{
			name:      "no dir and no json returns ErrNotFound",
			setupDir:  false,
			setupJSON: false,
			wantErr:   true,
			wantErrIs: ErrNotFound,
		},
		{
			name:        "dir with cluster.json is removed silently",
			setupDir:    true,
			setupJSON:   true,
			wantErr:     false,
			wantDirGone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			r, err := NewRepository()
			require.NoError(t, err)

			clusterDir := filepath.Join(r.fm.GetRootDir(), "cluster", "test")

			if tc.setupDir {
				err = os.MkdirAll(clusterDir, 0o755)
				require.NoError(t, err)
			}
			if tc.setupJSON {
				c := domain.Cluster{Name: "test", Network: domain.Network{Name: "hind.test"}}
				err = r.Save(context.Background(), c)
				require.NoError(t, err)
			}

			deleteErr := r.Delete(context.Background(), domain.Name("test"))

			if tc.wantErr {
				require.Error(t, deleteErr)
				if tc.wantErrIs != nil {
					assert.ErrorIs(t, deleteErr, tc.wantErrIs)
				}
				return
			}
			require.NoError(t, deleteErr)
			if tc.wantDirGone {
				_, statErr := os.Stat(clusterDir)
				assert.True(t, errors.Is(statErr, os.ErrNotExist), "cluster dir still exists after Delete, stat err = %v", statErr)
			}
		})
	}
}

func TestRepositoryDeleteConfinesToClusterRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r, err := NewRepository()
	require.NoError(t, err)

	c := domain.Cluster{Name: "demo", Network: domain.Network{Name: "hind.demo"}}
	err = r.Save(context.Background(), c)
	require.NoError(t, err)

	sentinel := filepath.Join(r.fm.GetRootDir(), "cluster", "safe-marker")
	err = os.WriteFile(sentinel, []byte("keep"), 0o644)
	require.NoError(t, err)

	err = r.Delete(context.Background(), domain.Name("demo"))
	require.NoError(t, err)

	_, statErr := os.Stat(sentinel)
	assert.NoError(t, statErr, "sentinel unexpectedly removed")
}
