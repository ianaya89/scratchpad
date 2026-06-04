package main

import "testing"

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
