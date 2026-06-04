package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestLoadConfigPrecedence(t *testing.T) {
	t.Setenv("PAD_DIR", "/tmp/env-dir")
	t.Setenv("PAD_WORKSPACE", "envws")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")

	// flag and positional override env
	c, ok := loadConfig([]string{"--dir", "/tmp/flag-dir", "posws"})
	if !ok {
		t.Fatal("loadConfig returned ok=false")
	}
	if c.dataDir != "/tmp/flag-dir" {
		t.Errorf("dataDir = %q, want flag value", c.dataDir)
	}
	if c.workspace != "posws" {
		t.Errorf("workspace = %q, want positional", c.workspace)
	}
}

func TestLoadConfigEnvFallback(t *testing.T) {
	t.Setenv("PAD_DIR", "/tmp/env-dir")
	t.Setenv("PAD_WORKSPACE", "envws")

	c, _ := loadConfig(nil)
	if c.dataDir != "/tmp/env-dir" {
		t.Errorf("dataDir = %q, want env value", c.dataDir)
	}
	if c.workspace != "envws" {
		t.Errorf("workspace = %q, want env value", c.workspace)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	t.Setenv("PAD_DIR", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("PAD_WORKSPACE", "")

	c, _ := loadConfig(nil)
	if filepath.Base(c.dataDir) != "pad" {
		t.Errorf("default dataDir = %q, want .../pad", c.dataDir)
	}
	if c.workspace != defaultWorkspace {
		t.Errorf("default workspace = %q", c.workspace)
	}
	if c.autosave != defaultAutosave {
		t.Errorf("default autosave = %v", c.autosave)
	}
}

func TestEnvDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"5":  5 * time.Second,
		"2s": 2 * time.Second,
		"":   defaultAutosave,
		"x":  defaultAutosave,
	}
	for in, want := range cases {
		t.Setenv("PAD_AUTOSAVE", in)
		if in == "" {
			t.Setenv("PAD_AUTOSAVE", "")
		}
		if got := envDuration("PAD_AUTOSAVE", defaultAutosave); got != want {
			t.Errorf("envDuration(%q) = %v, want %v", in, got, want)
		}
	}
}
