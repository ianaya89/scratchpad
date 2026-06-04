package main

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	m = upd(m, key(tea.KeyF1))
	if m.mode != modeHelp {
		t.Fatalf("F1 mode = %d, want help", m.mode)
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

func altRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: true}
}

func TestAltJump(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "jump"
	seedWorkspace(t, ws, map[string]string{
		"1-a.md": "a", "2-b.md": "b", "3-c.md": "c",
	})
	m := newModel(ws)
	m.width, m.height = 80, 24

	m = upd(m, altRunes("3"))
	if m.active != 2 {
		t.Errorf("alt+3 -> active %d, want 2", m.active)
	}
	m = upd(m, altRunes("1"))
	if m.active != 0 {
		t.Errorf("alt+1 -> active %d, want 0", m.active)
	}
	m = upd(m, altRunes("9")) // out of range: no-op
	if m.active != 0 {
		t.Errorf("alt+9 (oob) changed active to %d", m.active)
	}
}

func TestTabBarOverflowFits(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "ovf"
	notes := map[string]string{}
	for i := 1; i <= 20; i++ {
		notes[strconv.Itoa(i)+"-tab-number-"+strconv.Itoa(i)+".md"] = ""
	}
	seedWorkspace(t, ws, notes)
	m := newModel(ws)
	m.width, m.height = 40, 24
	m.active = 15
	m = upd(m, key(tea.KeyCtrlT)) // also exercises a fresh render path
	m.active = 15

	bar := m.tabBar()
	if lipgloss.Width(bar) > m.width+4 { // small slack for indicator rounding
		t.Errorf("tab bar width %d exceeds terminal %d", lipgloss.Width(bar), m.width)
	}
	if !strings.Contains(bar, "›") && !strings.Contains(bar, "‹") {
		t.Error("expected an overflow indicator with 20 tabs in width 40")
	}
}

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

func TestSplitPreviewToggle(t *testing.T) {
	m := newTestModel(t, "split")
	if m.editorWidth() != m.width {
		t.Fatalf("editor not full width initially: %d/%d", m.editorWidth(), m.width)
	}
	m = upd(m, key(tea.KeyCtrlB))
	if !m.splitPreview {
		t.Fatal("^b did not enable split")
	}
	if m.editorWidth() >= m.width {
		t.Errorf("editor not halved in split: %d/%d", m.editorWidth(), m.width)
	}
	if !strings.Contains(m.View(), "│") {
		t.Error("split view missing divider")
	}
	m = upd(m, key(tea.KeyCtrlB))
	if m.splitPreview || m.editorWidth() != m.width {
		t.Error("^b did not restore full-width edit")
	}
}

func TestToggleCheckboxLine(t *testing.T) {
	cases := map[string]string{
		"- [ ] task":  "- [x] task",
		"- [x] task":  "- [ ] task",
		"- [X] task":  "- [ ] task",
		"buy milk":    "- [ ] buy milk",
		"- buy milk":  "- [ ] buy milk",
		"  - [ ] sub": "  - [x] sub",
	}
	for in, want := range cases {
		if got := toggleCheckboxLine(in); got != want {
			t.Errorf("toggleCheckboxLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckboxKey(t *testing.T) {
	m := newTestModel(t, "cb")
	m = upd(m, runes("- [ ] do it"))
	m = upd(m, key(tea.KeyCtrlX))
	if !strings.Contains(m.ta.Value(), "[x]") {
		t.Errorf("checkbox not checked: %q", m.ta.Value())
	}
	m = upd(m, key(tea.KeyCtrlX))
	if !strings.Contains(m.ta.Value(), "[ ]") {
		t.Errorf("checkbox not unchecked: %q", m.ta.Value())
	}
}

func TestRestoreLastTab(t *testing.T) {
	dataRoot = t.TempDir()
	ws := "rest"
	seedWorkspace(t, ws, map[string]string{
		"1-a.md": "a", "2-b.md": "b", "3-c.md": "c",
	})
	m := newModel(ws)
	m.width, m.height = 80, 24
	m = upd(m, altRunes("3")) // jump to tab 3, persists active
	if m.active != 2 {
		t.Fatalf("active = %d", m.active)
	}
	// a fresh model for the same workspace should reopen on tab 3
	m2 := newModel(ws)
	if m2.active != 2 {
		t.Errorf("restored active = %d, want 2", m2.active)
	}
}

func TestUndoRedo(t *testing.T) {
	m := newTestModel(t, "undo")
	m = upd(m, runes("hello"))
	m = upd(m, undoTickMsg(m.undoSeq)) // checkpoint
	m = upd(m, runes(" world"))
	m = upd(m, undoTickMsg(m.undoSeq)) // checkpoint
	if m.ta.Value() != "hello world" {
		t.Fatalf("buffer = %q", m.ta.Value())
	}
	m = upd(m, key(tea.KeyCtrlZ))
	if m.ta.Value() != "hello" {
		t.Errorf("after undo = %q, want hello", m.ta.Value())
	}
	m = upd(m, key(tea.KeyCtrlZ))
	if m.ta.Value() != "" {
		t.Errorf("after 2nd undo = %q, want empty", m.ta.Value())
	}
	m = upd(m, key(tea.KeyCtrlY))
	if m.ta.Value() != "hello" {
		t.Errorf("after redo = %q, want hello", m.ta.Value())
	}
	// content stash + dirty so it persists
	if m.tabs[m.active].content != "hello" {
		t.Errorf("tab content = %q, want hello", m.tabs[m.active].content)
	}
}

func TestUndoPerTab(t *testing.T) {
	m := newTestModel(t, "undo2")
	m = upd(m, runes("first"))
	m = upd(m, undoTickMsg(m.undoSeq))
	m = upd(m, key(tea.KeyCtrlT)) // new tab, switches (commits via createTab path)
	m = upd(m, runes("second"))
	m = upd(m, undoTickMsg(m.undoSeq))
	// undo on tab 2 must not touch tab 1
	m = upd(m, key(tea.KeyCtrlZ))
	if m.ta.Value() != "" {
		t.Errorf("tab2 undo = %q, want empty", m.ta.Value())
	}
	if m.tabs[0].content != "first" {
		t.Errorf("tab1 content disturbed: %q", m.tabs[0].content)
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
