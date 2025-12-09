// Package files manages embedded filesystem operations for Docker image build contexts.
// It handles extracting embedded Dockerfiles and rootfs content to temporary build directories.
package files

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/stenh0use/hind/pkg/file"
)

const (
	buildBaseDir = ".cache"
	buildSubDir  = "hind"
)

// EmbeddedFS contains all Dockerfiles and build context files for all images.
//
//go:embed nodes/*/Dockerfile nodes/*/rootfs/**/*
var ImageFS embed.FS

type Image struct {
	name     string
	buildDir string
	files    fs.FS
	manager  *file.Manager
}

func New(name string) (Image, error) {
	i := Image{}

	i.name = name

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return i, fmt.Errorf("failed to get home directory for image %s build files: %w", name, err)
	}

	i.buildDir = file.JoinPath(homeDir, buildBaseDir, buildSubDir, i.name)
	fileManager, err := file.New(i.buildDir)
	if err != nil {
		return i, fmt.Errorf("failed to create new build files cache: %w", err)
	}

	i.manager = fileManager
	i.files, err = imageFS(name)
	if err != nil {
		return i, fmt.Errorf("failed to initialize build files for image %s: %w", name, err)
	}
	return i, nil
}

func imageFS(i string) (fs.FS, error) {
	const nodesDir = "nodes"
	// fs.Sub returns an FS corresponding to the subtree rooted at dir.
	subFS, err := fs.Sub(ImageFS, filepath.Join(nodesDir, i))
	if err != nil {
		return nil, fmt.Errorf("failed to get subFS '%s': %w", i, err)
	}

	return subFS, nil
}

func (i *Image) WriteFiles() error {
	if err := i.manager.EnsureDir(i.buildDir); err != nil {
		return fmt.Errorf("failed to create build dir: %w", err)
	}

	// Walk through the image-specific build context FS.
	// Paths here will be relative to the image's context root (e.g., "Dockerfile", "rootfs/somefile")
	err := fs.WalkDir(i.files, ".", func(pathInSubFS string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error during walk of build context for %s at %s: %w", i.name, pathInSubFS, walkErr)
		}

		if d.IsDir() {
			// Create corresponding directory in the cache if it's a directory
			// fs.WalkDir visits directories too, ensure they are created.
			if err := i.manager.EnsureDir(pathInSubFS); err != nil {
				return fmt.Errorf("failed to create image build dir %s: %w", pathInSubFS, err)
			}
			return nil
		}

		// Ensure the parent directory for the destination file exists
		// This is somewhat redundant if directories are created as above, but good for safety.
		parentDirOfDest := filepath.Dir(pathInSubFS)
		if err := i.manager.EnsureDir(parentDirOfDest); err != nil {
			return fmt.Errorf("failed to create parent directory %s for file %s: %w", parentDirOfDest, pathInSubFS, err)
		}

		// Read the file content from the subFS
		fileContent, err := fs.ReadFile(i.files, pathInSubFS)
		if err != nil {
			return fmt.Errorf("failed to read file %s from build context for %s: %w", pathInSubFS, i.name, err)
		}

		// Write the file content to the destination path
		if err := i.manager.WriteFile(pathInSubFS, fileContent); err != nil {
			return fmt.Errorf("failed to write file %s to %s: %w", pathInSubFS, pathInSubFS, err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("image %s: %w", i.name, err)
	}

	return nil
}

// BuildDir returns the build directory path
func (i *Image) BuildDir() string {
	return i.buildDir
}
