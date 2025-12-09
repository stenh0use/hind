package file

import (
	"errors"
	"fmt"
	"io"
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

// Creates a new file manager with a path relative from the user home dir
func NewFromHomeDir(paths ...string) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("filed to get user home dir: %w", err)
	}

	appendPath := JoinPath(paths...)
	rootDir := JoinPath(homeDir, appendPath)

	return New(rootDir)
}

// New creates a new file manager for the specified root directory
func New(rootDir string) (*Manager, error) {
	// Validate rootDir
	if err := validatePath(rootDir); err != nil {
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

	fullPath := f.resolvePath(path)
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

	fullPath := f.resolvePath(path)
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

	fullPath := f.resolvePath(path)
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

	fullPath := f.resolvePath(path)
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

	fullPath := f.resolvePath(path)

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

	fullPath := f.resolvePath(path)
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

	fullPath := f.resolvePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CopyFile copies a file from src to dst (both relative to root)
func (f *Manager) CopyFile(src, dst string) error {
	if err := validatePath(src); err != nil {
		return fmt.Errorf("invalid source path for CopyFile: %w", err)
	}
	if err := validatePath(dst); err != nil {
		return fmt.Errorf("invalid destination path for CopyFile: %w", err)
	}

	srcPath := f.resolvePath(src)
	dstPath := f.resolvePath(dst)

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create destination directory for %s: %w", dstPath, err)
	}

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy from %s to %s: %w", srcPath, dstPath, err)
	}

	// Set proper permissions
	if err := os.Chmod(dstPath, filePermissions); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dstPath, err)
	}

	return nil
}

// RemoveFile removes a file
func (f *Manager) RemoveFile(path string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("invalid path for RemoveFile: %w", err)
	}

	fullPath := f.resolvePath(path)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
	}
	return nil
}

// Path Operations

// GetPath returns the full path for a relative path within the root directory
func (f *Manager) GetPath(path string) string {
	if err := validatePath(path); err != nil {
		return ""
	}
	return f.resolvePath(path)
}

// GetRootDir returns the root directory
func (f *Manager) GetRootDir() string {
	return f.rootDir
}

// Exists checks if a path exists (file or directory)
func (f *Manager) Exists(path string) bool {
	if err := validatePath(path); err != nil {
		return false
	}

	fullPath := f.resolvePath(path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// resolvePath resolves a path relative to the root directory
func (f *Manager) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return JoinPath(f.rootDir, path)
}

func JoinPath(paths ...string) string {
	return filepath.Clean(filepath.Join(paths...))
}

// validatePath validates that a path is not empty and is relative
func validatePath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	// Trim whitespace and check again
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path cannot be empty or whitespace")
	}

	return nil
}
