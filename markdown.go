package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

func (m *model) openPreview() {
	m.stash()
	body := ""
	if len(m.tabs) > 0 {
		body = m.tabs[m.active].content
	}
	w := m.width - 2
	if w < 20 {
		w = 20
	}
	m.preview = viewport.New(m.width, m.previewHeight())
	m.preview.SetContent(m.renderMarkdown(body, "*(empty note)*", w))
	m.previewReady = true
	m.mode = modePreview
}

func (m model) previewHeight() int {
	h := m.height - 2
	if h < 3 {
		h = 3
	}
	return h
}

// editorWidth is the textarea width — half the screen when the split preview
// is on (leaving one column for the divider), full width otherwise.
func (m model) editorWidth() int {
	if m.splitPreview && m.width > 30 {
		return m.width / 2
	}
	return m.width
}

func (m model) splitPaneWidth() int {
	w := m.width - m.editorWidth() - 1
	if w < 10 {
		w = 10
	}
	return w
}

// renderer returns a Glamour renderer for width w, cached across calls so the
// (expensive) construction happens once per width rather than per keystroke.
func (m *model) renderer(w int) *glamour.TermRenderer {
	if m.splitRender == nil || m.splitRenderW != w {
		// WithStandardStyle avoids WithAutoStyle's synchronous OSC background
		// query, which can stall for hundreds of ms in some terminals.
		if r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(glamourStyle), glamour.WithWordWrap(w)); err == nil {
			m.splitRender = r
			m.splitRenderW = w
		}
	}
	return m.splitRender
}

// refreshSplit re-renders the current buffer as markdown into the split pane.
// Heavy (Glamour) — call only on toggle/resize/tab-switch and debounced idle,
// never directly on a keystroke.
func (m *model) refreshSplit() {
	if !m.splitPreview {
		return
	}
	w := m.splitPaneWidth()
	m.splitVP.Width = w
	m.splitVP.Height = m.height - 4
	m.splitVP.SetContent(m.renderMarkdown(m.ta.Value(), "*(empty — type markdown on the left)*", w))
	m.splitDirty = false
}

// renderMarkdown renders body as styled markdown wrapped to w columns, falling
// back to emptyMsg when the body is blank and to raw text if rendering fails.
func (m *model) renderMarkdown(body, emptyMsg string, w int) string {
	if strings.TrimSpace(body) == "" {
		body = emptyMsg
	}
	if r := m.renderer(w); r != nil {
		if out, err := r.Render(body); err == nil {
			return out
		}
	}
	return body
}

// scheduleSplitRender debounces: a render fires ~120ms after the last keystroke,
// coalescing bursts of typing into a single Glamour pass.
func scheduleSplitRender(seq int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return splitRenderMsg(seq) })
}
