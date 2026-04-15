// Package mover implements the core file movement and collision resolution logic.
package mover

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jules/go-filemover/internal/fs"
)

// Mover coordinates file operations using an abstracted [fs.FileSystem].
type Mover struct {
	fs fs.FileSystem
}

// NewMover creates a new [Mover] instance with the provided [fs.FileSystem].
func NewMover(f fs.FileSystem) *Mover {
	return &Mover{fs: f}
}

// MatchGlob checks if the given filename matches any of the provided glob patterns.
// It returns true if any pattern matches, otherwise false.
func (m *Mover) MatchGlob(patterns []string, filename string) (bool, error) {
	for _, p := range patterns {
		matched, err := filepath.Match(p, filename)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern '%s': %w", p, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// FileExists returns true if a file exists at the given path and is not a directory.
func (m *Mover) FileExists(path string) bool {
	info, err := m.fs.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

// GenerateUniquePath modifies the provided path to be unique if a file already exists there.
// It appends a counter in the format " (n)" before the file extension.
func (m *Mover) GenerateUniquePath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(filepath.Base(originalPath), ext)

	counter := 1
	newPath := originalPath
	for {
		if !m.FileExists(newPath) {
			return newPath
		}
		newPath = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, counter, ext))
		counter++
	}
}

// CalculateCRC32 calculates the CRC32 IEEE checksum of the file at the given path.
func (m *Mover) CalculateCRC32(path string) (uint32, error) {
	f, err := m.fs.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, f); err != nil {
		return 0, err
	}

	return hasher.Sum32(), nil
}

// MoveFile attempts to move a file from src to dst.
// It first tries [fs.FileSystem.Rename], and falls back to [Mover.CopyAndDelete] only if a
// cross-device link error (EXDEV) is encountered. For all other errors, the original error
// from Rename is returned unchanged.
func (m *Mover) MoveFile(src, dst string) error {
	err := m.fs.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Handle cross-device link error (e.g., WSL to Windows NTFS)
	if linkErr, ok := err.(*os.LinkError); ok {
		if linkErr.Err == syscall.EXDEV {
			return m.CopyAndDelete(src, dst)
		}
	}

	return err
}

// CopyAndDelete performs a manual file move by copying data and then removing the source file.
// If copying fails, it attempts to remove the partially created destination file.
func (m *Mover) CopyAndDelete(src, dst string) error {
	srcFile, err := m.fs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := m.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		m.fs.Remove(dst) // Cleanup partial copy
		return fmt.Errorf("failed during byte copy: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}

	if err := m.fs.Remove(src); err != nil {
		return fmt.Errorf("copied successfully, but failed to remove original: %w", err)
	}

	return nil
}
