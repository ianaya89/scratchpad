package main

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, autosaveTick(), statusTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layoutTextarea()
		m.refreshSplit()
		return m, nil

	case autosaveMsg:
		if m.mode == modeEdit {
			m.saveActive()
		}
		return m, autosaveTick()

	case statusTickMsg:
		if m.status != "" && time.Since(m.statusAt) >= statusTTL {
			m.setStatus("")
		}
		return m, statusTick()

	case splitRenderMsg:
		// only render for the latest scheduled seq (coalesce typing bursts)
		if int(msg) == m.splitSeq && m.splitDirty && m.splitPreview {
			m.refreshSplit()
		}
		return m, nil

	case undoTickMsg:
		if int(msg) == m.undoSeq && m.undoDirty {
			m.checkpoint()
		}
		return m, nil

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
		case modeSearch:
			return m.updateSearch(msg)
		case modeHelp:
			m.mode = modeEdit
			return m, nil
		case modePreview:
			return m.updatePreview(msg)
		}
	}

	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// alt+1..9 jumps straight to that tab
	if s := msg.String(); len(s) == 5 && strings.HasPrefix(s, "alt+") && s[4] >= '1' && s[4] <= '9' {
		if idx := int(s[4] - '1'); idx < len(m.tabs) && idx != m.active {
			m.saveActive()
			m.active = idx
			m.syncToTextarea()
			m.persistActive()
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		m.saveAll()
		return m, tea.Quit
	case "ctrl+t":
		m.saveActive()
		m.createTab("untitled")
		m.syncToTextarea()
		m.setStatus("new tab")
		return m, nil
	case "ctrl+d", "ctrl+w":
		m.mode = modeConfirmDelete
		return m, nil
	case "ctrl+x":
		m.checkpoint()
		m.toggleCheckbox()
		m.checkpoint()
		if m.splitPreview {
			m.refreshSplit()
		}
		return m, nil
	case "ctrl+z":
		m.doUndo()
		return m, nil
	case "ctrl+y":
		m.doRedo()
		return m, nil
	case "ctrl+n", "ctrl+>", "alt+l", "shift+right":
		m.switchTab(1)
		return m, nil
	case "ctrl+p", "ctrl+<", "alt+h", "shift+left":
		m.switchTab(-1)
		return m, nil
	case "ctrl+right", "alt+L", "alt+shift+l":
		m.moveTab(1)
		m.setStatus("moved tab")
		return m, nil
	case "ctrl+left", "alt+H", "alt+shift+h":
		m.moveTab(-1)
		m.setStatus("moved tab")
		return m, nil
	case "ctrl+f":
		m.searchPick = 0
		m.input.SetValue("")
		m.input.Focus()
		m.runSearch("")
		m.mode = modeSearch
		return m, nil
	case "ctrl+o":
		m.openPreview()
		return m, nil
	case "ctrl+b":
		m.splitPreview = !m.splitPreview
		m.layoutTextarea()
		m.refreshSplit()
		if m.splitPreview {
			m.setStatus("split preview on")
		} else {
			m.setStatus("split preview off")
		}
		return m, nil
	case "ctrl+/", "ctrl+_", "f1":
		m.mode = modeHelp
		return m, nil
	case "ctrl+s":
		m.saveAll()
		m.setStatus("saved")
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
	cmds := []tea.Cmd{cmd}
	// debounce an undo checkpoint after the typing burst settles
	m.undoDirty = true
	m.undoSeq++
	cmds = append(cmds, scheduleCheckpoint(m.undoSeq))
	if m.splitPreview {
		m.splitDirty = true
		m.splitSeq++
		cmds = append(cmds, scheduleSplitRender(m.splitSeq))
	}
	return m, tea.Batch(cmds...)
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.closeTab()
		m.mode = modeEdit
		m.setStatus("deleted note")
		return m, nil
	default:
		m.mode = modeEdit
		return m, nil
	}
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeEdit
		return m, nil
	case "up", "ctrl+p":
		if m.searchPick > 0 {
			m.searchPick--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.searchPick < len(m.searchResults)-1 {
			m.searchPick++
		}
		return m, nil
	case "enter":
		if len(m.searchResults) > 0 {
			m.saveActive()
			m.active = m.searchResults[m.searchPick]
			m.syncToTextarea()
		}
		m.mode = modeEdit
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.runSearch(m.input.Value())
	return m, cmd
}

func (m model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+o", "q", "ctrl+c":
		m.mode = modeEdit
		return m, nil
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
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
				m.setStatus("renamed")
			case modeNewWorkspace:
				m.saveAll()
				m.ws = slug(val)
				m.active = 0
				m.reload()
				m.setStatus("workspace: " + m.ws)
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
			m.setStatus("workspace: " + m.ws)
		}
		m.mode = modeEdit
		return m, nil
	}
	return m, nil
}
