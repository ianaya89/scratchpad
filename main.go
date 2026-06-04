package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeEdit mode = iota
	modeRename
	modeNewWorkspace
	modeSwitchWorkspace
	modeConfirmDelete
)

type tab struct {
	file    string
	title   string
	content string
	dirty   bool
}

type autosaveMsg time.Time

type model struct {
	ws     string
	tabs   []tab
	active int

	ta    textarea.Model
	input textinput.Model

	mode   mode
	wsList []string // for switch picker
	wsPick int

	width, height int
	status        string
}

func newModel(ws string) model {
	ta := textarea.New()
	ta.Placeholder = "scratch here…"
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
		m.tabs = append(m.tabs, tab{file: n.file, title: n.title, content: readNote(n.file)})
	}
	if len(m.tabs) == 0 {
		m.createTab("untitled")
	}
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	}
	m.syncToTextarea()
}

func (m *model) createTab(title string) {
	path := nextPath(m.ws, title)
	writeNote(path, "")
	m.tabs = append(m.tabs, tab{file: path, title: title})
	m.active = len(m.tabs) - 1
}

func (m *model) syncToTextarea() {
	if len(m.tabs) == 0 {
		return
	}
	m.ta.SetValue(m.tabs[m.active].content)
	m.ta.CursorEnd()
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
}

func (m *model) switchTab(delta int) {
	if len(m.tabs) < 2 {
		return
	}
	m.saveActive()
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.syncToTextarea()
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

func autosaveTick() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return autosaveMsg(t) })
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, autosaveTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layoutTextarea()
		return m, nil

	case autosaveMsg:
		if m.mode == modeEdit {
			m.saveActive()
		}
		return m, autosaveTick()

	case tea.KeyMsg:
		switch m.mode {
		case modeEdit:
			return m.updateEdit(msg)
		case modeRename, modeNewWorkspace:
			return m.updateInput(msg)
		case modeSwitchWorkspace:
			return m.updatePicker(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}

	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		m.saveAll()
		return m, tea.Quit
	case "ctrl+t":
		m.saveActive()
		m.createTab("untitled")
		m.syncToTextarea()
		m.status = "new tab"
		return m, nil
	case "ctrl+d", "ctrl+w":
		m.mode = modeConfirmDelete
		return m, nil
	case "ctrl+}", "ctrl+right", "alt+l", "shift+right":
		m.switchTab(1)
		return m, nil
	case "ctrl+{", "ctrl+left", "alt+h", "shift+left":
		m.switchTab(-1)
		return m, nil
	case "ctrl+s":
		m.saveAll()
		m.status = "saved"
		return m, nil
	case "ctrl+r":
		m.mode = modeRename
		m.input.SetValue(m.tabs[m.active].title)
		m.input.CursorEnd()
		m.input.Focus()
		return m, nil
	case "ctrl+e":
		m.saveAll()
		m.wsList = listWorkspaces()
		m.wsPick = 0
		for i, w := range m.wsList {
			if w == m.ws {
				m.wsPick = i
			}
		}
		m.mode = modeSwitchWorkspace
		return m, nil
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.closeTab()
		m.mode = modeEdit
		m.status = "deleted note"
		return m, nil
	default:
		m.mode = modeEdit
		return m, nil
	}
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeEdit
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			switch m.mode {
			case modeRename:
				t := &m.tabs[m.active]
				newPath := renamePath(t.file, val)
				if newPath != t.file {
					os.Rename(t.file, newPath)
					t.file = newPath
				}
				t.title = val
				m.status = "renamed"
			case modeNewWorkspace:
				m.saveAll()
				m.ws = slug(val)
				m.active = 0
				m.reload()
				m.status = "workspace: " + m.ws
			}
		}
		m.mode = modeEdit
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeEdit
		return m, nil
	case "up", "ctrl+p", "k":
		if m.wsPick > 0 {
			m.wsPick--
		}
		return m, nil
	case "down", "ctrl+n", "j":
		if m.wsPick < len(m.wsList)-1 {
			m.wsPick++
		}
		return m, nil
	case "n":
		m.mode = modeNewWorkspace
		m.input.SetValue("")
		m.input.Focus()
		return m, nil
	case "enter":
		if len(m.wsList) > 0 {
			m.ws = m.wsList[m.wsPick]
			m.active = 0
			m.reload()
			m.status = "workspace: " + m.ws
		}
		m.mode = modeEdit
		return m, nil
	}
	return m, nil
}

func (m *model) layoutTextarea() {
	if m.height == 0 {
		return
	}
	m.ta.SetWidth(m.width)
	m.ta.SetHeight(m.height - 4) // tab bar + footer
}

func main() {
	ws := "default"
	if len(os.Args) > 1 {
		ws = slug(os.Args[1])
	}
	p := tea.NewProgram(newModel(ws), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
