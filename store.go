package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// root is ~/.nb/scratchpad — its own dir, separate from the nb notebook.
func rootDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nb", "scratchpad")
}

func workspaceDir(ws string) string {
	return filepath.Join(rootDir(), ws)
}

// listWorkspaces returns sorted workspace names (subdirs of root).
func listWorkspaces() []string {
	entries, err := os.ReadDir(rootDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

type note struct {
	file  string // absolute path
	title string // human title derived from filename
}

var prefixRe = regexp.MustCompile(`^(\d+)-(.*)\.md$`)

// loadNotes reads a workspace dir, returns notes ordered by numeric prefix.
func loadNotes(ws string) ([]note, error) {
	dir := workspaceDir(ws)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var notes []note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		m := prefixRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		notes = append(notes, note{
			file:  filepath.Join(dir, e.Name()),
			title: unslug(m[2]),
		})
	}
	sort.Slice(notes, func(i, j int) bool {
		return prefixOf(notes[i].file) < prefixOf(notes[j].file)
	})
	return notes, nil
}

func prefixOf(path string) int {
	m := prefixRe.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func readNote(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func writeNote(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// nextPath builds the next numbered file path for a title in a workspace.
func nextPath(ws, title string) string {
	notes, _ := loadNotes(ws)
	max := 0
	for _, n := range notes {
		if p := prefixOf(n.file); p > max {
			max = p
		}
	}
	name := strconv.Itoa(max+1) + "-" + slug(title) + ".md"
	return filepath.Join(workspaceDir(ws), name)
}

// renamePath keeps the numeric prefix, swaps the slug for a new title.
func renamePath(oldPath, title string) string {
	p := prefixOf(oldPath)
	name := strconv.Itoa(p) + "-" + slug(title) + ".md"
	return filepath.Join(filepath.Dir(oldPath), name)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

func unslug(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}
