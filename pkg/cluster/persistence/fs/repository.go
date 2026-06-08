package fs

// Context cancellation is intentionally not propagated to filesystem operations in
// this package. Go's standard library os and filepath functions do not accept context,
// and the file.Manager abstraction does not expose context-aware I/O. Callers should
// treat repository operations as best-effort: a cancelled context will not interrupt
// an in-flight write. If context-aware I/O is required in future, the file.Manager
// abstraction must be extended first.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/file"
)

var ErrNotFound = errors.New("store: not found")

type Repository struct{ fm *file.Manager }

var _ persistence.Repository = (*Repository)(nil)
var _ persistence.ActiveRepository = (*Repository)(nil)

func NewRepository() (*Repository, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %w", err)
	}
	rootDir := filepath.Clean(filepath.Join(homeDir, ".config", "hind"))
	fm, err := file.New(rootDir)
	if err != nil {
		return nil, err
	}
	return &Repository{fm: fm}, nil
}

func (r *Repository) Load(_ context.Context, name domain.Name) (domain.Cluster, error) {
	path, err := clusterConfigPath("", name)
	if err != nil {
		return domain.Cluster{}, err
	}
	if !r.fm.FileExists(path) {
		return domain.Cluster{}, fmt.Errorf("cluster '%s' not found: %w", name, ErrNotFound)
	}
	data, err := r.fm.ReadFile(path)
	if err != nil {
		return domain.Cluster{}, err
	}
	return decodeCluster(data)
}
func (r *Repository) Save(_ context.Context, c domain.Cluster) error {
	if c.Name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}
	path, err := clusterConfigPath("", c.Name)
	if err != nil {
		return err
	}
	data, err := encodeCluster(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := r.fm.WriteFile(tmp, data); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}
	absRoot := r.fm.GetRootDir()
	if absRoot != "" {
		_ = syncPath(filepath.Join(absRoot, filepath.Dir(tmp)))
	}
	if err := os.Rename(filepath.Join(absRoot, tmp), filepath.Join(absRoot, path)); err != nil {
		return fmt.Errorf("failed to atomically rename config file: %w", err)
	}
	_ = syncPath(filepath.Join(absRoot, filepath.Dir(path)))
	return nil
}
func (r *Repository) Delete(_ context.Context, name domain.Name) error {
	dir, err := clusterDirPath("", name)
	if err != nil {
		return err
	}
	cfgPath, _ := clusterConfigPath("", name)
	jsonExists := r.fm.FileExists(cfgPath)
	dirExists := r.fm.DirExists(dir)
	if !jsonExists && !dirExists {
		return fmt.Errorf("cluster '%s' not found: %w", name, ErrNotFound)
	}
	// Covers both the normal path (cluster.json present) and the stale-dir path
	// (cluster.json absent but the directory still exists as an empty orphan).
	return r.fm.RemoveDir(dir)
}
func (r *Repository) Exists(_ context.Context, name domain.Name) (bool, error) {
	path, err := clusterConfigPath("", name)
	if err != nil {
		return false, err
	}
	return r.fm.FileExists(path), nil
}
func (r *Repository) List(_ context.Context) ([]domain.Name, error) {
	entries, err := r.fm.ListDir(clusterConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.Name{}, nil
		}
		return nil, err
	}
	out := []domain.Name{}
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, domain.Name(e.Name()))
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Compare(string(out[i]), string(out[j])) < 0 })
	return out, nil
}
func (r *Repository) GetActive(ctx context.Context) (string, error) {
	p := activePath("")
	if !r.fm.FileExists(p) {
		return "", nil
	}
	b, err := r.fm.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("failed to read active cluster file: %w", err)
	}
	name := string(b)
	if name == "" {
		return "", nil
	}
	exists, err := r.Exists(ctx, domain.Name(name))
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return name, nil
}
func (r *Repository) SetActive(ctx context.Context, clusterName string) error {
	if _, err := validateName(domain.Name(clusterName)); err != nil {
		return err
	}
	exists, err := r.Exists(ctx, domain.Name(clusterName))
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("cluster '%s' does not exist: %w", clusterName, ErrNotFound)
	}
	if err := r.fm.EnsureDir(clusterConfigDir); err != nil {
		return fmt.Errorf("failed to ensure cluster directory exists: %w", err)
	}
	if err := r.fm.WriteFile(activePath(""), []byte(clusterName)); err != nil {
		return fmt.Errorf("failed to write active cluster file: %w", err)
	}
	return nil
}
func (r *Repository) ClearActive(_ context.Context) error {
	p := activePath("")
	if !r.fm.FileExists(p) {
		return nil
	}
	if err := r.fm.RemoveFile(p); err != nil {
		return fmt.Errorf("failed to remove active cluster file: %w", err)
	}
	return nil
}

// RootDir returns the absolute path to the configuration root directory.
func (r *Repository) RootDir() string {
	return r.fm.GetRootDir()
}

func syncPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close() //nolint:errcheck
		return err
	}
	return f.Close()
}
