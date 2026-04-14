// Package fs provides a filesystem abstraction layer to facilitate testing.
package fs

import (
	"io"
	"os"
)

// FileSystem defines the interface for filesystem operations.
// This allows for mocking the filesystem in unit tests.
type FileSystem interface {
	// Open opens the named file for reading.
	Open(name string) (File, error)
	// Create creates or truncates the named file.
	Create(name string) (File, error)
	// Stat returns a [os.FileInfo] describing the named file.
	Stat(name string) (os.FileInfo, error)
	// Rename renames (moves) oldpath to newpath.
	Rename(oldpath, newpath string) error
	// Remove removes the named file or (empty) directory.
	Remove(name string) error
	// MkdirAll creates a directory and all necessary parents.
	MkdirAll(path string, perm os.FileMode) error
}

// File defines the interface for file operations, combining [io.Reader], [io.Writer], and [io.Closer].
type File interface {
	io.Reader
	io.Writer
	io.Closer
	// Stat returns the [os.FileInfo] structure describing file.
	Stat() (os.FileInfo, error)
}

// OSFileSystem implements [FileSystem] using the standard [os] package.
type OSFileSystem struct{}

// Open implements [FileSystem.Open].
func (OSFileSystem) Open(name string) (File, error) { return os.Open(name) }

// Create implements [FileSystem.Create].
func (OSFileSystem) Create(name string) (File, error) { return os.Create(name) }

// Stat implements [FileSystem.Stat].
func (OSFileSystem) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

// Rename implements [FileSystem.Rename].
func (OSFileSystem) Rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }

// Remove implements [FileSystem.Remove].
func (OSFileSystem) Remove(name string) error { return os.Remove(name) }

// MkdirAll implements [FileSystem.MkdirAll].
func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
