// Package file provides a path-confined filesystem Manager.
// All methods resolve paths relative to the root directory supplied at construction
// and reject traversal segments and absolute paths, preventing any operation from
// escaping the configured root. Callers must supply an absolute root path to [New].
package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirPermissions  = 0755
	filePermissions = 0644
)

// Manager handles file and directory operations for a specific root directory
type Manager struct {
	// The directory that will be prepended to all file path operations
	rootDir string
}

// New creates a new file manager for the specified root directory
func New(rootDir string) (*Manager, error) {
	// Validate rootDir
	if err := validateRootPath(rootDir); err != nil {
		return nil, fmt.Errorf("invalid path for rootDir: %w", err)
	}

	// Clean the path and resolve any relative components
	cleanPath := filepath.Clean(rootDir)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", rootDir, err)
	}

	return &Manager{rootDir: absPath}, nil
}

// Directory Operations

// EnsureDir creates a directory and all necessary parent directories
func (f *Manager) EnsureDir(path string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path for EnsureDir: %w", err)
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path for EnsureDir: %w", err)
	}

	if err := os.MkdirAll(fullPath, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
	}
	return nil
}

// RemoveDir removes a directory and all its contents
func (f *Manager) RemoveDir(path string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path for RemoveDir: %w", err)
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path for RemoveDir: %w", err)
	}

	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
	}
	return nil
}

// DirExists checks if a directory exists
func (f *Manager) DirExists(path string) bool {
	if err := validatePath(path); err != nil {
		return false
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return false
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListDir returns the contents of a directory
func (f *Manager) ListDir(path string) ([]os.DirEntry, error) {
	if err := validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid path for ListDir: %w", err)
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path for ListDir: %w", err)
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", fullPath, err)
	}
	return entries, nil
}

// File Operations

// WriteFile writes data to a file, creating parent directories if necessary
func (f *Manager) WriteFile(path string, data []byte) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path for WriteFile: %w", err)
	}

	if data == nil {
		return errors.New("data cannot be nil")
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path for WriteFile: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create parent directory for file %s: %w", fullPath, err)
	}

	if err := os.WriteFile(fullPath, data, filePermissions); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}
	return nil
}

// ReadFile reads data from a file
func (f *Manager) ReadFile(path string) ([]byte, error) {
	if err := validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid path for ReadFile: %w", err)
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path for ReadFile: %w", err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	return data, nil
}

// FileExists checks if a file exists
func (f *Manager) FileExists(path string) bool {
	if err := validatePath(path); err != nil {
		return false
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return false
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// RemoveFile removes a file
func (f *Manager) RemoveFile(path string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path for RemoveFile: %w", err)
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path for RemoveFile: %w", err)
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
	}
	return nil
}

// Path Operations

// GetRootDir returns the root directory
func (f *Manager) GetRootDir() string {
	return f.rootDir
}

// resolvePath resolves a path relative to the root directory and ensures confinement.
func (f *Manager) resolvePath(path string) (string, error) {
	fullPath := filepath.Clean(filepath.Join(f.rootDir, path))

	relPath, err := filepath.Rel(f.rootDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate relative path: %w", err)
	}

	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes root directory")
	}

	return fullPath, nil
}

// validatePath validates that a path is not empty, not absolute, and does not include traversal segments.
func validatePath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path cannot be empty or whitespace")
	}

	if filepath.IsAbs(trimmed) {
		return errors.New("path must be relative")
	}

	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, segment := range segments {
		if segment == ".." {
			return errors.New("path cannot contain traversal segments")
		}
	}

	return nil
}

// validateRootPath validates root path input for manager creation.
func validateRootPath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path cannot be empty or whitespace")
	}

	return nil
}
