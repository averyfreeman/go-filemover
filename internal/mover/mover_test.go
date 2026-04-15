package mover

import (
	"bytes"
	"errors"
	"hash/crc32"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/jules/go-filemover/internal/fs"
)

// MockFileInfo implements os.FileInfo
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return 0 }
func (m MockFileInfo) Mode() os.FileMode  { return 0 }
func (m MockFileInfo) ModTime() time.Time { return time.Now() }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() any           { return nil }

// MockFile implements fs.File
type MockFile struct {
	*bytes.Buffer
	info   MockFileInfo
	closed bool
}

func (m *MockFile) Close() error {
	m.closed = true
	return nil
}
func (m *MockFile) Stat() (os.FileInfo, error) { return m.info, nil }

type failingReader struct {
	err error
}

func (f *failingReader) Read(p []byte) (int, error) {
	return 0, f.err
}
func (f *failingReader) Close() error {
	return nil
}
func (f *failingReader) Write(p []byte) (int, error) {
	return 0, errors.New("readonly")
}
func (f *failingReader) Stat() (os.FileInfo, error) { return MockFileInfo{}, nil }

// MockFileSystem implements fs.FileSystem
type MockFileSystem struct {
	Files      map[string][]byte
	RenameErr  error
	CreateErr  error
	OpenErr    error
	RemoveErr  error
	OpenReader fs.File
}

func (m *MockFileSystem) Open(name string) (fs.File, error) {
	if m.OpenErr != nil {
		return nil, m.OpenErr
	}
	if m.OpenReader != nil {
		return m.OpenReader, nil
	}
	if content, ok := m.Files[name]; ok {
		return &MockFile{Buffer: bytes.NewBuffer(content), info: MockFileInfo{name: name, isDir: false}}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Create(name string) (fs.File, error) {
	if m.CreateErr != nil {
		return nil, m.CreateErr
	}
	f := &MockFile{Buffer: new(bytes.Buffer), info: MockFileInfo{name: name, isDir: false}}
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[name] = []byte{}
	return &closeWrapper{MockFile: f, fs: m, name: name}, nil
}

type closeWrapper struct {
	*MockFile
	fs   *MockFileSystem
	name string
}

func (c *closeWrapper) Close() error {
	c.fs.Files[c.name] = c.Buffer.Bytes()
	return c.MockFile.Close()
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if _, ok := m.Files[name]; ok {
		return MockFileInfo{name: name, isDir: false}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	if m.RenameErr != nil {
		return m.RenameErr
	}
	if content, ok := m.Files[oldpath]; ok {
		m.Files[newpath] = content
		delete(m.Files, oldpath)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFileSystem) Remove(name string) error {
	if m.RemoveErr != nil {
		return m.RemoveErr
	}
	if _, ok := m.Files[name]; ok {
		delete(m.Files, name)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func TestMatchGlob(t *testing.T) {
	m := NewMover(nil)

	t.Run("valid patterns", func(t *testing.T) {
		patterns := []string{"*.txt", "*.go"}

		matched, err := m.MatchGlob(patterns, "file.txt")
		if err != nil {
			t.Fatalf("unexpected error for valid patterns: %v", err)
		}
		if !matched {
			t.Errorf("expected match for file.txt")
		}

		matched, err = m.MatchGlob(patterns, "image.png")
		if err != nil {
			t.Fatalf("unexpected error for valid patterns: %v", err)
		}
		if matched {
			t.Errorf("did not expect match for image.png")
		}
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		patterns := []string{"[", "*.txt"}

		matched, err := m.MatchGlob(patterns, "file.txt")
		if err == nil {
			t.Fatalf("expected error for invalid glob pattern, got nil")
		}
		if matched {
			t.Errorf("expected matched to be false when error is returned")
		}
	})
}

func TestGenerateUniquePath(t *testing.T) {
	tests := []struct {
		name     string
		original string
		existing []string
		want     string
	}{
		{
			name:     "no collision simple file",
			original: "test.txt",
			existing: nil,
			want:     "test.txt",
		},
		{
			name:     "single collision simple file",
			original: "test.txt",
			existing: []string{"test.txt"},
			want:     "test (1).txt",
		},
		{
			name:     "directory path preserved",
			original: "dir/sub/test.txt",
			existing: []string{"dir/sub/test.txt"},
			want:     "dir/sub/test (1).txt",
		},
		{
			name:     "no extension no collision",
			original: "test",
			existing: nil,
			want:     "test",
		},
		{
			name:     "no extension with collisions",
			original: "test",
			existing: []string{"test", "test (1)", "test (2)"},
			want:     "test (3)",
		},
		{
			name:     "non-sequential collisions",
			original: "test.txt",
			existing: []string{"test.txt", "test (2).txt", "test (3).txt"},
			want:     "test (1).txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &MockFileSystem{Files: make(map[string][]byte)}
			for _, path := range tt.existing {
				mockFS.Files[path] = []byte{}
			}
			m := NewMover(mockFS)

			if got := m.GenerateUniquePath(tt.original); got != tt.want {
				t.Errorf("GenerateUniquePath(%q) = %q, want %q", tt.original, got, tt.want)
			}
		})
	}
}

func TestMoveFile_RenameSuccess(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("hello world"),
		},
	}
	m := NewMover(fs)

	err := m.MoveFile("/src.txt", "/dst.txt")
	if err != nil {
		t.Fatalf("MoveFile returned error: %v", err)
	}

	if _, ok := fs.Files["/src.txt"]; ok {
		t.Fatalf("expected source file to be removed after MoveFile, but it still exists")
	}

	if got, ok := fs.Files["/dst.txt"]; !ok {
		t.Fatalf("expected destination file to exist after MoveFile")
	} else if string(got) != "hello world" {
		t.Fatalf("destination contents = %q, want %q", string(got), "hello world")
	}
}

func TestMoveFile_EXDEVUsesCopyAndDelete(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("cross-device data"),
		},
		RenameErr: &os.LinkError{
			Op:  "rename",
			Old: "/src.txt",
			New: "/dst.txt",
			Err: syscall.EXDEV,
		},
	}
	m := NewMover(fs)

	err := m.MoveFile("/src.txt", "/dst.txt")
	if err != nil {
		t.Fatalf("MoveFile (EXDEV) returned error: %v", err)
	}

	if _, ok := fs.Files["/src.txt"]; ok {
		t.Fatalf("expected source file to be removed after MoveFile fallback, but it still exists")
	}

	if got, ok := fs.Files["/dst.txt"]; !ok {
		t.Fatalf("expected destination file to exist after MoveFile fallback")
	} else if string(got) != "cross-device data" {
		t.Fatalf("destination contents = %q, want %q", string(got), "cross-device data")
	}
}

func TestMoveFile_OtherErrorReturnsOriginal(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("data"),
		},
		RenameErr: os.ErrPermission,
	}
	m := NewMover(fs)

	err := m.MoveFile("/src.txt", "/dst.txt")
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected os.ErrPermission, got %v", err)
	}
}

func TestCopyAndDelete_CreateFails(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("data"),
		},
		CreateErr: os.ErrPermission,
	}
	m := NewMover(fs)

	err := m.CopyAndDelete("/src.txt", "/dst.txt")
	if err == nil {
		t.Fatalf("expected error from CopyAndDelete when Create fails, got nil")
	}

	if _, ok := fs.Files["/src.txt"]; !ok {
		t.Fatalf("expected source file to remain when CopyAndDelete fails on Create")
	}
}

func TestCopyAndDelete_OpenFails(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("data"),
		},
		OpenErr: os.ErrNotExist,
	}
	m := NewMover(fs)

	err := m.CopyAndDelete("/src.txt", "/dst.txt")
	if err == nil {
		t.Fatalf("expected error from CopyAndDelete when Open fails, got nil")
	}
}

func TestCopyAndDelete_RemoveFails(t *testing.T) {
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/src.txt": []byte("data"),
		},
		RemoveErr: os.ErrPermission,
	}
	m := NewMover(fs)

	err := m.CopyAndDelete("/src.txt", "/dst.txt")
	if err == nil {
		t.Fatalf("expected error from CopyAndDelete when Remove fails, got nil")
	}

	if _, ok := fs.Files["/src.txt"]; !ok {
		t.Fatalf("expected source file to remain when CopyAndDelete fails on Remove")
	}
	if _, ok := fs.Files["/dst.txt"]; !ok {
		t.Fatalf("expected destination file to exist when copy succeeded but Remove failed")
	}
}

func TestCalculateCRC32_Success(t *testing.T) {
	content := []byte("checksum content")
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/file.txt": content,
		},
	}
	m := NewMover(fs)

	got, err := m.CalculateCRC32("/file.txt")
	if err != nil {
		t.Fatalf("CalculateCRC32 returned error: %v", err)
	}

	want := crc32.ChecksumIEEE(content)
	if got != want {
		t.Fatalf("CalculateCRC32(\"/file.txt\") = 0x%x, want 0x%x", got, want)
	}
}

func TestCalculateCRC32_OpenFails(t *testing.T) {
	fs := &MockFileSystem{
		Files:   map[string][]byte{},
		OpenErr: os.ErrNotExist,
	}
	m := NewMover(fs)

	_, err := m.CalculateCRC32("/missing.txt")
	if err == nil {
		t.Fatalf("expected error from CalculateCRC32 when Open fails, got nil")
	}
}

func TestCalculateCRC32_CopyFails(t *testing.T) {
	rErr := errors.New("copy failed")
	fs := &MockFileSystem{
		Files: map[string][]byte{
			"/file.txt": []byte("unused"),
		},
		OpenReader: &failingReader{err: rErr},
	}
	m := NewMover(fs)

	_, err := m.CalculateCRC32("/file.txt")
	if err == nil {
		t.Fatalf("expected error from CalculateCRC32 when io.Copy fails, got nil")
	}
}
