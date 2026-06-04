package main

import "testing"

func TestRunNewAndAppend(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "cli"

	runNew(ws, "Hello World", "body")
	notes, _ := loadNotes(ws)
	if len(notes) != 1 || readNote(notes[0].file) != "body" {
		t.Fatalf("runNew: notes=%d content=%q", len(notes), readNote(notes[0].file))
	}

	runAppend(ws, "hello", "more") // matches by slug
	if got := readNote(notes[0].file); got != "body\nmore\n" {
		t.Errorf("after append = %q, want \"body\\nmore\\n\"", got)
	}

	runAppend(ws, "does-not-exist", "z") // creates
	if notes, _ := loadNotes(ws); len(notes) != 2 {
		t.Errorf("append-missing should create: notes=%d", len(notes))
	}
}
