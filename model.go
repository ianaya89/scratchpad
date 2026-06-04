package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type mode int

const (
	modeEdit mode = iota
	modeRename
	modeNewWorkspace
	modeSwitchWorkspace
	modeConfirmDelete
	modeSearch
	modeHelp
	modePreview
)

type tab struct {
	file    string
	title   string
	content string
	dirty   bool

	undo     []string // prior buffer states (oldest first)
	redo     []string // states undone, available to redo
	undoBase string   // buffer content at the last checkpoint
}

const maxUndo = 200

type autosaveMsg time.Time
type statusTickMsg time.Time
type splitRenderMsg int // carries the seq it was scheduled for, to coalesce
type undoTickMsg int    // debounced checkpoint for undo history

// statusTTL is how long a transient status message stays visible.
const statusTTL = 3 * time.Second

type model struct {
	ws     string
	tabs   []tab
	active int

	ta    textarea.Model
	input textinput.Model

	mode   mode
	wsList []string // for switch picker
	wsPick int

	searchResults []int // tab indices matching the search query
	searchPick    int

	preview      viewport.Model
	previewReady bool

	splitPreview bool
	splitVP      viewport.Model
	splitRender  *glamour.TermRenderer
	splitRenderW int
	splitDirty   bool
	splitSeq     int
	undoSeq      int
	undoDirty    bool

	width, height int
	status        string
	statusAt      time.Time
}

func newModel(ws string) model {
	ta := textarea.New()
	ta.Placeholder = "Start typing…  (^/ help · ^t new tab · ^f find · ^o preview · ^q quit)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Focus()

	in := textinput.New()
	in.Prompt = "› "

	m := model{ws: ws, ta: ta, input: in, mode: modeEdit}
	m.reload()
	return m
}

func (m *model) reload() {
	notes, _ := loadNotes(m.ws)
	m.tabs = nil
	for _, n := range notes {
		body := readNote(n.file)
		m.tabs = append(m.tabs, tab{file: n.file, title: n.title, content: body, undoBase: body})
	}
	if len(m.tabs) == 0 {
		m.createTab("untitled")
	}
	// restore the last active tab for this workspace, if recorded
	if base := readActiveNote(m.ws); base != "" {
		for i, t := range m.tabs {
			if filepath.Base(t.file) == base {
				m.active = i
				break
			}
		}
	}
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	}
	m.syncToTextarea()
}

// persistActive records the current tab so the workspace reopens on it.
func (m *model) persistActive() {
	if len(m.tabs) > 0 {
		writeActiveNote(m.ws, filepath.Base(m.tabs[m.active].file))
	}
}

func (m *model) createTab(title string) {
	path := nextPath(m.ws, title)
	writeNote(path, "")
	m.tabs = append(m.tabs, tab{file: path, title: title})
	m.active = len(m.tabs) - 1
	m.persistActive()
}

func (m *model) syncToTextarea() {
	if len(m.tabs) == 0 {
		return
	}
	m.ta.SetValue(m.tabs[m.active].content)
	m.ta.CursorEnd()
	m.refreshSplit()
}

// stash writes the textarea buffer back into the active tab struct.
func (m *model) stash() {
	if len(m.tabs) == 0 {
		return
	}
	v := m.ta.Value()
	if v != m.tabs[m.active].content {
		m.tabs[m.active].content = v
		m.tabs[m.active].dirty = true
	}
}

func (m *model) saveActive() {
	if len(m.tabs) == 0 {
		return
	}
	m.stash()
	t := &m.tabs[m.active]
	if t.dirty {
		writeNote(t.file, t.content)
		t.dirty = false
	}
}

func (m *model) saveAll() {
	m.stash()
	for i := range m.tabs {
		if m.tabs[i].dirty {
			writeNote(m.tabs[i].file, m.tabs[i].content)
			m.tabs[i].dirty = false
		}
	}
	m.persistActive()
}

func (m *model) switchTab(delta int) {
	if len(m.tabs) < 2 {
		return
	}
	m.checkpoint()
	m.saveActive()
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.syncToTextarea()
	m.persistActive()
}

func (m *model) closeTab() {
	if len(m.tabs) == 0 {
		return
	}
	os.Remove(m.tabs[m.active].file)
	m.tabs = append(m.tabs[:m.active], m.tabs[m.active+1:]...)
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	}
	if len(m.tabs) == 0 {
		m.createTab("untitled")
	}
	m.syncToTextarea()
}

// toggleCheckboxLine toggles a markdown task checkbox on a single line:
func toggleCheckboxLine(s string) string {
	if strings.Contains(s, "[ ]") {
		return strings.Replace(s, "[ ]", "[x]", 1)
	}
	low := strings.ToLower(s)
	if i := strings.Index(low, "[x]"); i >= 0 {
		return s[:i] + "[ ]" + s[i+3:]
	}
	trimmed := strings.TrimLeft(s, " \t")
	indent := s[:len(s)-len(trimmed)]
	trimmed = strings.TrimPrefix(trimmed, "- ")
	return indent + "- [ ] " + trimmed
}

// toggleCheckbox toggles the checkbox on the cursor's current line.
func (m *model) toggleCheckbox() {
	val := m.ta.Value()
	lines := strings.Split(val, "\n")
	li := m.ta.Line()
	if li < 0 || li >= len(lines) {
		return
	}
	lines[li] = toggleCheckboxLine(lines[li])
	m.ta.SetValue(strings.Join(lines, "\n"))
	// SetValue parks the cursor at the buffer end; walk back up to line li
	for i := 0; i < len(lines)-1-li; i++ {
		m.ta.CursorUp()
	}
	m.ta.CursorEnd()
}
func (m *model) checkpoint() {
	if len(m.tabs) == 0 {
		return
	}
	t := &m.tabs[m.active]
	cur := m.ta.Value()
	if cur == t.undoBase {
		return
	}
	t.undo = append(t.undo, t.undoBase)
	if len(t.undo) > maxUndo {
		t.undo = t.undo[len(t.undo)-maxUndo:]
	}
	t.redo = t.redo[:0]
	t.undoBase = cur
	m.undoDirty = false
}

func (m *model) doUndo() {
	m.checkpoint()
	if len(m.tabs) == 0 {
		return
	}
	t := &m.tabs[m.active]
	if len(t.undo) == 0 {
		m.setStatus("nothing to undo")
		return
	}
	t.redo = append(t.redo, m.ta.Value())
	prev := t.undo[len(t.undo)-1]
	t.undo = t.undo[:len(t.undo)-1]
	m.applyHistory(prev)
	m.setStatus("undo")
}

func (m *model) doRedo() {
	if len(m.tabs) == 0 {
		return
	}
	t := &m.tabs[m.active]
	if len(t.redo) == 0 {
		m.setStatus("nothing to redo")
		return
	}
	t.undo = append(t.undo, m.ta.Value())
	next := t.redo[len(t.redo)-1]
	t.redo = t.redo[:len(t.redo)-1]
	m.applyHistory(next)
	m.setStatus("redo")
}

// applyHistory swaps the buffer to a historical state and syncs the tab.
func (m *model) applyHistory(s string) {
	t := &m.tabs[m.active]
	m.ta.SetValue(s)
	m.ta.CursorEnd()
	t.undoBase = s
	t.content = s
	t.dirty = true
	m.refreshSplit()
}

func scheduleCheckpoint(seq int) tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return undoTickMsg(seq) })
}

func (m *model) moveTab(delta int) {
	j := m.active + delta
	if j < 0 || j >= len(m.tabs) || len(m.tabs) < 2 {
		return
	}
	m.saveActive()
	a, b := &m.tabs[m.active], &m.tabs[j]
	pa, pb := prefixOf(a.file), prefixOf(b.file)
	na, nb := setPrefix(a.file, pb), setPrefix(b.file, pa)
	if err := os.Rename(a.file, na); err != nil {
		return
	}
	if err := os.Rename(b.file, nb); err != nil {
		os.Rename(na, a.file) // best-effort rollback
		return
	}
	a.file, b.file = na, nb
	m.tabs[m.active], m.tabs[j] = m.tabs[j], m.tabs[m.active]
	m.active = j
	m.persistActive()
}

func (m *model) runSearch(query string) {
	m.stash()
	q := strings.ToLower(strings.TrimSpace(query))
	m.searchResults = m.searchResults[:0]
	for i, t := range m.tabs {
		if q == "" ||
			strings.Contains(strings.ToLower(t.title), q) ||
			strings.Contains(strings.ToLower(t.content), q) {
			m.searchResults = append(m.searchResults, i)
		}
	}
	if m.searchPick >= len(m.searchResults) {
		m.searchPick = 0
	}
}

var autosaveInterval = defaultAutosave

func autosaveTick() tea.Cmd {
	if autosaveInterval <= 0 {
		return nil
	}
	return tea.Tick(autosaveInterval, func(t time.Time) tea.Msg { return autosaveMsg(t) })
}

// setStatus shows a transient message that auto-clears after statusTTL.
func (m *model) setStatus(s string) {
	m.status = s
	m.statusAt = time.Now()
}

func statusTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return statusTickMsg(t) })
}
func (m *model) layoutTextarea() {
	if m.height == 0 {
		return
	}
	m.ta.SetWidth(m.editorWidth())
	m.ta.SetHeight(m.height - 4) // tab bar + footer
	if m.previewReady {
		m.preview.Width = m.width
		m.preview.Height = m.previewHeight()
	}
}
