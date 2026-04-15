package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
[task1]
wd = "/tmp/watch"
td = "/tmp/target"
fg = "*.txt, *.log"
`
	tmpfile, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg) != 1 {
		t.Errorf("Expected 1 task, got %d", len(cfg))
	}

	task, ok := cfg["task1"]
	if !ok {
		t.Fatal("task1 not found")
	}

	if task.Wd != "/tmp/watch" || task.Td != "/tmp/target" || task.Fg != "*.txt, *.log" {
		t.Errorf("Task values mismatch: %+v", task)
	}
}

func TestGetPatterns(t *testing.T) {
	task := TaskConfig{Fg: "*.txt, *.log ,  *.pdf "}
	patterns := task.GetPatterns()
	expected := []string{"*.txt", "*.log", "*.pdf"}

	if len(patterns) != len(expected) {
		t.Fatalf("Expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, p := range patterns {
		if p != expected[i] {
			t.Errorf("Pattern %d: expected %s, got %s", i, expected[i], p)
		}
	}
}
