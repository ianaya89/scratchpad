package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetPrefix(t *testing.T) {
	got := setPrefix("/x/3-hello-world.md", 7)
	if got != "/x/7-hello-world.md" {
		t.Errorf("setPrefix = %q", got)
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

func TestStoreRoundtrip(t *testing.T) {
	ws := "__smoke_test"
	defer os.RemoveAll(workspaceDir(ws))

	p1 := nextPath(ws, "My First Note")
	if err := writeNote(p1, "hello"); err != nil {
		t.Fatal(err)
	}
	p2 := nextPath(ws, "Second")
	writeNote(p2, "world")

	notes, err := loadNotes(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("want 2 notes, got %d", len(notes))
	}
	if notes[0].title != "my first note" {
		t.Errorf("title0 = %q", notes[0].title)
	}
	if readNote(notes[1].file) != "world" {
		t.Errorf("body1 mismatch")
	}
	rn := renamePath(notes[0].file, "Renamed!!")
	if rn == notes[0].file {
		t.Errorf("rename did not change path")
	}
	if prefixOf(rn) != prefixOf(notes[0].file) {
		t.Errorf("rename lost numeric prefix")
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Hello World":  "hello-world",
		"  spaced  ":   "spaced",
		"a/b:c":        "a-b-c",
		"":             "untitled",
		"UPPER_case99": "upper-case99",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}
