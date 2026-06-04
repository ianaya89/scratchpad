package main

import (
	"os"
	"path/filepath"
	"testing"
)

func seedWorkspace(t *testing.T, ws string, notes map[string]string) {
	t.Helper()
	dir := workspaceDir(ws)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range notes {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSetPrefix(t *testing.T) {
	got := setPrefix("/x/3-hello-world.md", 7)
	if got != "/x/7-hello-world.md" {
		t.Errorf("setPrefix = %q", got)
	}
}

func TestMoveTab(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "mv"
	seedWorkspace(t, ws, map[string]string{
		"1-alpha.md": "a",
		"2-beta.md":  "b",
	})
	m := newModel(ws)
	if m.tabs[0].title != "alpha" || m.tabs[1].title != "beta" {
		t.Fatalf("initial order: %s, %s", m.tabs[0].title, m.tabs[1].title)
	}
	m.moveTab(1) // move alpha (active 0) right
	if m.tabs[0].title != "beta" || m.tabs[1].title != "alpha" {
		t.Fatalf("after move: %s, %s", m.tabs[0].title, m.tabs[1].title)
	}
	if m.active != 1 {
		t.Errorf("active = %d, want 1 (follows moved tab)", m.active)
	}
	// order must survive a reload from disk
	m2 := newModel(ws)
	if m2.tabs[0].title != "beta" || m2.tabs[1].title != "alpha" {
		t.Errorf("reload order: %s, %s", m2.tabs[0].title, m2.tabs[1].title)
	}
}

func TestRunSearch(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "find"
	seedWorkspace(t, ws, map[string]string{
		"1-shopping.md": "milk and eggs",
		"2-todo.md":     "ship the release",
		"3-ideas.md":    "rocket milk idea",
	})
	m := newModel(ws)

	m.runSearch("milk") // matches body of shopping + ideas
	if len(m.searchResults) != 2 {
		t.Fatalf("milk: got %d results, want 2", len(m.searchResults))
	}
	m.runSearch("todo") // matches title only
	if len(m.searchResults) != 1 {
		t.Fatalf("todo: got %d results, want 1", len(m.searchResults))
	}
	m.runSearch("") // empty matches everything
	if len(m.searchResults) != 3 {
		t.Fatalf("empty: got %d results, want 3", len(m.searchResults))
	}
	m.runSearch("zzz")
	if len(m.searchResults) != 0 {
		t.Fatalf("zzz: got %d results, want 0", len(m.searchResults))
	}
}

func TestAtomicWriteNoTmpLeak(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "atomic"
	dir := workspaceDir(ws)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeNote(filepath.Join(dir, "1-note.md"), "hello"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestConfigFileLayering(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	os.WriteFile(cfgPath, []byte("dir = \"/tmp/from-file\"\nworkspace = \"filews\"\nautosave = 9\n"), 0o644)

	// clear envs that could interfere
	t.Setenv("PAD_DIR", "")
	t.Setenv("PAD_WORKSPACE", "")
	t.Setenv("PAD_AUTOSAVE", "")
	t.Setenv("PAD_CONFIG", cfgPath)

	// file only
	c, _ := loadConfig(nil)
	if c.dataDir != "/tmp/from-file" || c.workspace != "filews" {
		t.Errorf("file layer: dir=%q ws=%q", c.dataDir, c.workspace)
	}
	if c.autosave.Seconds() != 9 {
		t.Errorf("file autosave = %v", c.autosave)
	}

	// env overrides file
	t.Setenv("PAD_WORKSPACE", "envws")
	c, _ = loadConfig(nil)
	if c.workspace != "envws" {
		t.Errorf("env should override file, got %q", c.workspace)
	}

	// flag overrides env + file
	c, _ = loadConfig([]string{"--workspace", "flagws"})
	if c.workspace != "flagws" {
		t.Errorf("flag should win, got %q", c.workspace)
	}
}
