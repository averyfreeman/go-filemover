package mover

import (
	"bytes"
	"os"
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
	info MockFileInfo
}

func (m *MockFile) Close() error               { return nil }
func (m *MockFile) Stat() (os.FileInfo, error) { return m.info, nil }

// MockFileSystem implements fs.FileSystem
type MockFileSystem struct {
	Files map[string]*MockFile
}

func (m *MockFileSystem) Open(name string) (fs.File, error) {
	if f, ok := m.Files[name]; ok {
		// Return a new reader from the same buffer for multiple reads if needed,
		// but for simplicity we'll just return the MockFile.
		return f, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Create(name string) (fs.File, error) {
	f := &MockFile{Buffer: new(bytes.Buffer), info: MockFileInfo{name: name, isDir: false}}
	m.Files[name] = f
	return f, nil
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if f, ok := m.Files[name]; ok {
		return f.info, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	if f, ok := m.Files[oldpath]; ok {
		m.Files[newpath] = f
		delete(m.Files, oldpath)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFileSystem) Remove(name string) error {
	delete(m.Files, name)
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func TestMatchGlob(t *testing.T) {
	m := NewMover(nil)
	patterns := []string{"*.txt", "*.go"}

	tests := []struct {
		filename string
		want     bool
	}{
		{"test.txt", true},
		{"main.go", true},
		{"README.md", false},
	}

	for _, tt := range tests {
		got, err := m.MatchGlob(patterns, tt.filename)
		if err != nil {
			t.Errorf("MatchGlob(%v, %s) error = %v", patterns, tt.filename, err)
		}
		if got != tt.want {
			t.Errorf("MatchGlob(%v, %s) = %v; want %v", patterns, tt.filename, got, tt.want)
		}
	}
}

func TestGenerateUniquePath(t *testing.T) {
	mockFS := &MockFileSystem{Files: make(map[string]*MockFile)}
	m := NewMover(mockFS)

	original := "test.txt"
	// No collision
	if got := m.GenerateUniquePath(original); got != original {
		t.Errorf("Expected %s, got %s", original, got)
	}

	// Single collision
	mockFS.Create("test.txt")
	expected1 := "test (1).txt"
	if got := m.GenerateUniquePath(original); got != expected1 {
		t.Errorf("Expected %s, got %s", expected1, got)
	}

	// Multiple collisions
	mockFS.Create("test (1).txt")
	expected2 := "test (2).txt"
	if got := m.GenerateUniquePath(original); got != expected2 {
		t.Errorf("Expected %s, got %s", expected2, got)
	}
}
