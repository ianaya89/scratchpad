package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	// non-TUI actions
	print      bool
	tab        string
	newTitle   string
	appendTo   string
	content    string
	contentSet bool
	list       bool
	wsExplicit bool // a workspace was named via flag/positional
}

// defaultDataDir is the fallback used when nothing overrides storage,
// including the PAD_DIR env (used by store.go when config never ran, e.g. tests).
func defaultDataDir() string {
	if d := os.Getenv("PAD_DIR"); d != "" {
		return expandHome(d)
	}
	return defaultDataDirBase()
}

// defaultDataDirBase is the platform default location, ignoring PAD_DIR.
func defaultDataDirBase() string {
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

// loadConfig resolves config with precedence flag > env > config file > default.
// Returns ok=false when the program should exit early (e.g. --help, --version).
func loadConfig(args []string) (config, bool) {
	fs := flag.NewFlagSet("pad", flag.ContinueOnError)
	fs.Usage = usage

	dir := fs.String("dir", "", "data directory")
	ws := fs.String("workspace", "", "workspace to open")
	autosave := fs.Int("autosave", -1, "autosave interval in seconds (0 disables)")
	cfgPath := fs.String("config", "", "path to config file")
	doPrint := fs.Bool("print", false, "print note(s) to stdout and exit (no TUI)")
	tab := fs.String("tab", "", "with --print: only this tab (title substring)")
	newTitle := fs.String("new", "", "create a note with this title and exit (no TUI)")
	appendTo := fs.String("append", "", "append to the note matching this title (creates it if absent) and exit")
	content := fs.String("content", "", "content for --new/--append (default: stdin)")
	doList := fs.Bool("list", false, "list workspaces (or tabs of a named workspace) and exit")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return config{}, false
	}
	if *showVersion {
		fmt.Println("pad", version)
		return config{}, false
	}

	// base defaults
	c := config{
		dataDir:   defaultDataDirBase(),
		workspace: defaultWorkspace,
		autosave:  defaultAutosave,
	}

	// config-file layer
	fc := readConfigFile(resolveConfigPath(*cfgPath))
	if fc.dir != "" {
		c.dataDir = expandHome(fc.dir)
	}
	if fc.workspace != "" {
		c.workspace = fc.workspace
	}
	if fc.autosave >= 0 {
		c.autosave = time.Duration(fc.autosave) * time.Second
	}

	// env layer
	if v := os.Getenv("PAD_DIR"); v != "" {
		c.dataDir = expandHome(v)
	}
	if v := os.Getenv("PAD_WORKSPACE"); v != "" {
		c.workspace = v
	}
	if os.Getenv("PAD_AUTOSAVE") != "" {
		c.autosave = envDuration("PAD_AUTOSAVE", c.autosave)
	}

	// flag layer
	if *dir != "" {
		c.dataDir = expandHome(*dir)
	}
	if *ws != "" {
		c.workspace = *ws
		c.wsExplicit = true
	}
	if *autosave >= 0 {
		c.autosave = time.Duration(*autosave) * time.Second
	}
	if positional := fs.Arg(0); positional != "" {
		c.workspace = positional
		c.wsExplicit = true
	}

	c.list = *doList
	c.print = *doPrint
	c.tab = *tab
	c.newTitle = *newTitle
	c.appendTo = *appendTo
	c.content = *content
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "content" {
			c.contentSet = true
		}
	})
	c.workspace = slug(c.workspace)
	return c, true
}

type fileCfg struct {
	dir       string
	workspace string
	autosave  int
}

// readConfigFile parses a minimal TOML-ish file: `key = value` lines,
// `#` comments, optional quotes. Recognized keys: dir, workspace, autosave.
func readConfigFile(path string) fileCfg {
	fc := fileCfg{autosave: -1}
	if path == "" {
		return fc
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fc
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), "\"'")
		switch k {
		case "dir":
			fc.dir = v
		case "workspace":
			fc.workspace = v
		case "autosave":
			if n, err := strconv.Atoi(v); err == nil {
				fc.autosave = n
			}
		}
	}
	return fc
}

func resolveConfigPath(flagPath string) string {
	if flagPath != "" {
		return expandHome(flagPath)
	}
	if e := os.Getenv("PAD_CONFIG"); e != "" {
		return expandHome(e)
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "pad", "config.toml")
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
  pad                  open the default workspace
  pad ideas            open/create the "ideas" workspace
  pad --dir ~/notes    store notes under ~/notes
  pad --print          dump the workspace's notes to stdout
  pad --print --tab todo
  pad --new "meeting" --content "kickoff notes"
  echo "remember this" | pad --append todo

Flags:
  --dir PATH           data directory
                       (default $PAD_DIR, $XDG_DATA_HOME/pad, or ~/.local/share/pad)
  --workspace NAME     workspace to open (default $PAD_WORKSPACE or "default")
  --autosave N         autosave interval in seconds (default $PAD_AUTOSAVE or 3; 0 disables)
  --config PATH        config file (default $PAD_CONFIG or ~/.config/pad/config.toml)
  --list               list workspaces, or tabs of a named workspace, and exit
  --print              print note(s) to stdout and exit (no TUI)
  --tab TITLE          with --print, restrict to tabs matching TITLE
  --new TITLE          create a note titled TITLE and exit
  --append TITLE       append to the note matching TITLE (creates it if absent)
  --content TEXT       content for --new/--append (default: read from stdin)
  --version            print version and exit
  -h, --help           show this help

Config file (TOML-ish, key = value):
  dir = "~/notes"
  workspace = "work"
  autosave = 5

Data layout:
  <dir>/<workspace>/NN-slug.md   one Markdown file per tab
`)
}
