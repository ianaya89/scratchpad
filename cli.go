package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

// glamourStyle is the preview/split markdown theme. Override with PAD_THEME
// (glamour standard styles: dark, light, dracula, tokyo-night, pink, notty…).
var glamourStyle = "dark"

// runPrint dumps a workspace's notes to stdout (no TUI). With tabFilter set,
// only notes whose title contains the filter (case-insensitive) are printed.
func runPrint(ws, tabFilter string) {
	notes, err := loadNotes(ws)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	filter := strings.ToLower(strings.TrimSpace(tabFilter))
	for _, n := range notes {
		if filter != "" && !strings.Contains(strings.ToLower(n.title), filter) {
			continue
		}
		fmt.Printf("## %s\n\n%s\n\n", n.title, readNote(n.file))
	}
}

// resolveContent returns the content for --new/--append: the --content flag if
// set, otherwise stdin when it's piped/redirected, otherwise empty.
func resolveContent(cfg config) string {
	if cfg.contentSet {
		return cfg.content
	}
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		b, _ := io.ReadAll(os.Stdin)
		return strings.TrimRight(string(b), "\n")
	}
	return ""
}

// runNew creates a note titled title with content, then prints its path.
func runNew(ws, title, content string) {
	path := nextPath(ws, title)
	if err := writeNote(path, content); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

// runAppend appends content to the first note matching title (by slug, then
// substring); if none matches it creates the note. Prints the path.
func runAppend(ws, title, content string) {
	notes, _ := loadNotes(ws)
	want := slug(title)
	var match string
	for _, n := range notes {
		if slug(n.title) == want {
			match = n.file
			break
		}
	}
	if match == "" {
		lt := strings.ToLower(strings.TrimSpace(title))
		for _, n := range notes {
			if strings.Contains(strings.ToLower(n.title), lt) {
				match = n.file
				break
			}
		}
	}
	if match == "" {
		runNew(ws, title, content)
		return
	}
	existing := readNote(match)
	sep := ""
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		sep = "\n"
	}
	if err := writeNote(match, existing+sep+content+"\n"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(match)
}

// runList prints workspace names, or the tab titles of a named workspace.
func runList(cfg config) {
	if cfg.wsExplicit {
		notes, _ := loadNotes(cfg.workspace)
		for _, n := range notes {
			fmt.Println(n.title)
		}
		return
	}
	for _, w := range listWorkspaces() {
		fmt.Println(w)
	}
}

func main() {
	cfg, ok := loadConfig(os.Args[1:])
	if !ok {
		return
	}
	dataRoot = cfg.dataDir
	autosaveInterval = cfg.autosave
	if t := os.Getenv("PAD_THEME"); t != "" {
		glamourStyle = t
	}

	switch {
	case cfg.list:
		runList(cfg)
		return
	case cfg.print:
		runPrint(cfg.workspace, cfg.tab)
		return
	case cfg.newTitle != "":
		runNew(cfg.workspace, cfg.newTitle, resolveContent(cfg))
		return
	case cfg.appendTo != "":
		runAppend(cfg.workspace, cfg.appendTo, resolveContent(cfg))
		return
	}

	p := tea.NewProgram(newModel(cfg.workspace), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
