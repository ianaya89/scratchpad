package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultWorkspace = "default"
	defaultAutosave  = 3 * time.Second
)

type config struct {
	dataDir   string
	workspace string
	autosave  time.Duration
}

// defaultDataDir resolves the storage dir when nothing overrides it:
// $PAD_DIR, else $XDG_DATA_HOME/pad, else ~/.local/share/pad.
func defaultDataDir() string {
	if d := os.Getenv("PAD_DIR"); d != "" {
		return expandHome(d)
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "pad")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "pad")
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// loadConfig resolves config from flags > env > defaults and parses argv.
// Returns ok=false when the program should exit early (e.g. --help, --version).
func loadConfig(args []string) (config, bool) {
	fs := flag.NewFlagSet("pad", flag.ContinueOnError)
	fs.Usage = usage

	dir := fs.String("dir", "", "data directory (default $PAD_DIR, $XDG_DATA_HOME/pad, or ~/.local/share/pad)")
	ws := fs.String("workspace", "", "workspace to open (default $PAD_WORKSPACE or \"default\")")
	autosave := fs.Int("autosave", 0, "autosave interval in seconds (default $PAD_AUTOSAVE or 3; 0 disables)")
	autosaveSet := false
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return config{}, false
	}
	if *showVersion {
		fmt.Println("pad", version)
		return config{}, false
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "autosave" {
			autosaveSet = true
		}
	})

	c := config{
		dataDir:   defaultDataDir(),
		workspace: envOr("PAD_WORKSPACE", defaultWorkspace),
		autosave:  envDuration("PAD_AUTOSAVE", defaultAutosave),
	}
	if *dir != "" {
		c.dataDir = expandHome(*dir)
	}
	if *ws != "" {
		c.workspace = *ws
	}
	if positional := fs.Arg(0); positional != "" {
		c.workspace = positional
	}
	if autosaveSet {
		c.autosave = time.Duration(*autosave) * time.Second
	}
	c.workspace = slug(c.workspace)
	return c, true
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	// accept a bare integer (seconds) or a Go duration like "5s"
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := time.ParseDuration(v + "s"); err == nil {
		return n
	}
	return def
}

func usage() {
	fmt.Fprintf(os.Stderr, `pad — tabbed terminal scratchpad

Usage:
  pad [flags] [workspace]

Examples:
  pad                 open the default workspace
  pad ideas           open/create the "ideas" workspace
  pad --dir ~/notes   store notes under ~/notes

Flags:
  --dir PATH          data directory
                      (default $PAD_DIR, $XDG_DATA_HOME/pad, or ~/.local/share/pad)
  --workspace NAME    workspace to open (default $PAD_WORKSPACE or "default")
  --autosave N        autosave interval in seconds (default $PAD_AUTOSAVE or 3; 0 disables)
  --version           print version and exit
  -h, --help          show this help

Data layout:
  <dir>/<workspace>/NN-slug.md   one Markdown file per tab
`)
}
