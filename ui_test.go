package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func upd(m model, msg tea.Msg) model {
	nm, _ := m.Update(msg)
	return nm.(model)
}

func key(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
func runes(s string) tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func newTestModel(t *testing.T, ws string) model {
	t.Helper()
	dataRoot = t.TempDir()
	m := newModel(ws)
	m.width, m.height = 80, 24
	m.layoutTextarea()
	return m
}

func TestKeyNewTab(t *testing.T) {
	m := newTestModel(t, "nt")
	if len(m.tabs) != 1 {
		t.Fatalf("start tabs = %d", len(m.tabs))
	}
	m = upd(m, key(tea.KeyCtrlT))
	if len(m.tabs) != 2 {
		t.Fatalf("after ^t tabs = %d", len(m.tabs))
	}
	if m.mode != modeEdit {
		t.Errorf("mode = %d, want edit", m.mode)
	}
}

func TestKeyDeleteConfirmFlow(t *testing.T) {
	m := newTestModel(t, "del")
	m = upd(m, key(tea.KeyCtrlT)) // 2 tabs
	if len(m.tabs) != 2 {
		t.Fatalf("setup tabs = %d", len(m.tabs))
	}
	m = upd(m, key(tea.KeyCtrlD))
	if m.mode != modeConfirmDelete {
		t.Fatalf("^d mode = %d, want confirmDelete", m.mode)
	}
	m = upd(m, runes("y"))
	if m.mode != modeEdit {
		t.Errorf("after y mode = %d, want edit", m.mode)
	}
	if len(m.tabs) != 1 {
		t.Errorf("after delete tabs = %d, want 1", len(m.tabs))
	}
}

func TestKeyDeleteCancel(t *testing.T) {
	m := newTestModel(t, "delc")
	m = upd(m, key(tea.KeyCtrlT))
	m = upd(m, key(tea.KeyCtrlD))
	m = upd(m, runes("n")) // anything but y cancels
	if m.mode != modeEdit {
		t.Errorf("mode = %d, want edit", m.mode)
	}
	if len(m.tabs) != 2 {
		t.Errorf("tabs = %d, want 2 (no delete)", len(m.tabs))
	}
}

func TestKeySearchFlow(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "srch"
	seedWorkspace(t, ws, map[string]string{
		"1-a.md": "milk and eggs",
		"2-b.md": "water only",
	})
	m := newModel(ws)
	m.width, m.height = 80, 24

	m = upd(m, key(tea.KeyCtrlF))
	if m.mode != modeSearch {
		t.Fatalf("^f mode = %d, want search", m.mode)
	}
	if len(m.searchResults) != 2 {
		t.Errorf("empty query results = %d, want 2", len(m.searchResults))
	}
	m = upd(m, runes("milk"))
	if len(m.searchResults) != 1 {
		t.Fatalf("query results = %d, want 1", len(m.searchResults))
	}
	want := m.searchResults[0]
	m = upd(m, key(tea.KeyEnter))
	if m.mode != modeEdit {
		t.Errorf("after enter mode = %d, want edit", m.mode)
	}
	if m.active != want {
		t.Errorf("active = %d, want %d (jumped to match)", m.active, want)
	}
}

func TestKeyPreviewFlow(t *testing.T) {
	m := newTestModel(t, "prev")
	m = upd(m, key(tea.KeyCtrlO))
	if m.mode != modePreview {
		t.Fatalf("^o mode = %d, want preview", m.mode)
	}
	m = upd(m, key(tea.KeyEsc))
	if m.mode != modeEdit {
		t.Errorf("after esc mode = %d, want edit", m.mode)
	}
}

func TestKeyRenameFlow(t *testing.T) {
	m := newTestModel(t, "ren")
	m = upd(m, key(tea.KeyCtrlR))
	if m.mode != modeRename {
		t.Fatalf("^r mode = %d, want rename", m.mode)
	}
	if m.input.Value() != m.tabs[m.active].title {
		t.Errorf("input prefilled %q, want current title", m.input.Value())
	}
	m.input.SetValue("renamed")
	m = upd(m, key(tea.KeyEnter))
	if m.mode != modeEdit {
		t.Errorf("after enter mode = %d", m.mode)
	}
	if m.tabs[m.active].title != "renamed" {
		t.Errorf("title = %q, want renamed", m.tabs[m.active].title)
	}
}

func TestKeyHelpToggle(t *testing.T) {
	m := newTestModel(t, "help")
	m = upd(m, key(tea.KeyCtrlG))
	if m.mode != modeHelp {
		t.Fatalf("^g mode = %d, want help", m.mode)
	}
	m = upd(m, runes("x")) // any key closes
	if m.mode != modeEdit {
		t.Errorf("mode = %d, want edit", m.mode)
	}
}

func TestKeyQuitReturnsQuitCmd(t *testing.T) {
	m := newTestModel(t, "quit")
	_, cmd := m.Update(key(tea.KeyCtrlQ))
	if cmd == nil {
		t.Fatal("^q returned nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("^q should return tea.Quit")
	}
}

func TestStatusAutoClear(t *testing.T) {
	m := newTestModel(t, "stat")

	m.setStatus("saved")
	// not yet expired
	m = upd(m, statusTickMsg(time.Now()))
	if m.status != "saved" {
		t.Errorf("status cleared too early: %q", m.status)
	}
	// force expiry
	m.statusAt = time.Now().Add(-statusTTL - time.Second)
	m = upd(m, statusTickMsg(time.Now()))
	if m.status != "" {
		t.Errorf("status = %q, want cleared", m.status)
	}
}
