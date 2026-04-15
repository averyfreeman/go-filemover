# API Documentation
## Internal/Config
package config // import "github.com/jules/go-filemover/internal/config"

Package config handles loading and parsing of the go-filemover configuration.

TYPES

type Config map[string]TaskConfig
    Config is a map of task IDs to their respective TaskConfig.

func LoadConfig(path string) (Config, error)
    LoadConfig reads and parses the TOML configuration file from the given path.

type TaskConfig struct {
	// Wd is the watched directory path.
	Wd string `toml:"wd"`
	// Td is the target directory path.
	Td string `toml:"td"`
	// Fg is a comma-separated list of glob patterns to match files against.
	Fg string `toml:"fg"`
}
    TaskConfig represents the configuration for a single file watching task.

func (t *TaskConfig) ExpandPaths()
    ExpandPaths expands environment variables within the Wd and Td fields of the
    TaskConfig.

func (t *TaskConfig) GetPatterns() []string
    GetPatterns splits the comma-separated Fg string into a slice of trimmed
    glob patterns.


## Internal/Mover
package mover // import "github.com/jules/go-filemover/internal/mover"

Package mover implements the core file movement and collision resolution logic.

TYPES

type Mover struct {
	// Has unexported fields.
}
    Mover coordinates file operations using an abstracted fs.FileSystem.

func NewMover(f fs.FileSystem) *Mover
    NewMover creates a new Mover instance with the provided fs.FileSystem.

func (m *Mover) CalculateCRC32(path string) (uint32, error)
    CalculateCRC32 calculates the CRC32 IEEE checksum of the file at the given
    path.

func (m *Mover) CopyAndDelete(src, dst string) error
    CopyAndDelete performs a manual file move by copying data and then removing
    the source file.

func (m *Mover) FileExists(path string) bool
    FileExists returns true if a file exists at the given path and is not a
    directory.

func (m *Mover) GenerateUniquePath(originalPath string) string
    GenerateUniquePath modifies the provided path to be unique if a file already
    exists there. It appends a counter in the format " (n)" before the file
    extension.

func (m *Mover) MatchGlob(patterns []string, filename string) (bool, error)
    MatchGlob checks if the given filename matches any of the provided glob
    patterns. It returns true if any pattern matches, otherwise false.

func (m *Mover) MoveFile(src, dst string) error
    MoveFile attempts to move a file from src to dst. It first tries
    fs.FileSystem.Rename, and falls back to Mover.CopyAndDelete if a
    cross-device link error (EXDEV) is encountered.


## Internal/FS
package fs // import "github.com/jules/go-filemover/internal/fs"

Package fs provides a filesystem abstraction layer to facilitate testing.

TYPES

type File interface {
	io.Reader
	io.Writer
	io.Closer
	// Stat returns the [os.FileInfo] structure describing file.
	Stat() (os.FileInfo, error)
}
    File defines the interface for file operations, combining io.Reader,
    io.Writer, and io.Closer.

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
    FileSystem defines the interface for filesystem operations. This allows for
    mocking the filesystem in unit tests.

type OSFileSystem struct{}
    OSFileSystem implements FileSystem using the standard os package.

func (OSFileSystem) Create(name string) (File, error)
    Create implements [FileSystem.Create].

func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error
    MkdirAll implements [FileSystem.MkdirAll].

func (OSFileSystem) Open(name string) (File, error)
    Open implements [FileSystem.Open].

func (OSFileSystem) Remove(name string) error
    Remove implements [FileSystem.Remove].

func (OSFileSystem) Rename(oldpath, newpath string) error
    Rename implements [FileSystem.Rename].

func (OSFileSystem) Stat(name string) (os.FileInfo, error)
    Stat implements [FileSystem.Stat].
