package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("231")).
			Bold(true)
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("245"))
	wsStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("214")).
		Bold(true)
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")).
			Bold(true)
	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

func (m model) View() string {
	switch m.mode {
	case modeRename:
		return m.overlay("Rename tab", m.input.View(), "enter confirm · esc cancel")
	case modeNewWorkspace:
		return m.overlay("New workspace", m.input.View(), "enter create · esc cancel")
	case modeSwitchWorkspace:
		return m.pickerView()
	case modeConfirmDelete:
		title := "this note"
		if len(m.tabs) > 0 {
			title = m.tabs[m.active].title
		}
		return m.overlay("Delete note", "“"+title+"” — permanently?", "y delete · any other key cancel")
	case modeSearch:
		return m.searchView()
	case modeHelp:
		return m.helpView()
	case modePreview:
		return m.preview.View()
	}

	tabBar := m.tabBar()
	footer := m.footer()
	body := m.ta.View()
	if m.splitPreview {
		h := m.height - 4
		if h < 1 {
			h = 1
		}
		divider := dividerStyle.Render(strings.TrimRight(strings.Repeat("│\n", h), "\n"))
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.ta.View(), divider, m.splitVP.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, body, footer)
}

func (m model) tabBar() string {
	ws := wsStyle.Render("⌂ " + m.ws)

	labels := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		label := t.title
		if t.dirty {
			label += " •"
		}
		if i == m.active {
			labels[i] = activeTabStyle.Render(label)
		} else {
			labels[i] = tabStyle.Render(label)
		}
	}

	// fit a window of tabs (always including the active one) into the width,
	// scrolling and showing ‹N / N› indicators for hidden tabs on each side.
	budget := m.width - lipgloss.Width(ws) - 1
	if budget < 10 {
		budget = 10
	}
	start, end := m.active, m.active+1
	used := lipgloss.Width(labels[m.active])
	for {
		grew := false
		if end < len(labels) && used+lipgloss.Width(labels[end]) <= budget {
			used += lipgloss.Width(labels[end])
			end++
			grew = true
		}
		if start > 0 && used+lipgloss.Width(labels[start-1]) <= budget {
			start--
			used += lipgloss.Width(labels[start])
			grew = true
		}
		if !grew {
			break
		}
	}

	parts := []string{ws, " "}
	if start > 0 {
		parts = append(parts, tabStyle.Render("‹"+strconv.Itoa(start)))
	}
	parts = append(parts, labels[start:end]...)
	if end < len(labels) {
		parts = append(parts, tabStyle.Render(strconv.Itoa(len(labels)-end)+"›"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m model) footer() string {
	help := footerStyle.Render(
		"^t new · ^d del · ^p/^n prev/next · ⌥H/⌥L move · ^f find · ^o preview · ^b split · ^e ws · ^/ help · ^q quit",
	)
	if m.status != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, statusStyle.Render(m.status), "  ", help)
	}
	return help
}

func (m model) searchView() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Find in workspace"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	if len(m.searchResults) == 0 {
		b.WriteString(footerStyle.Render("(no matches)"))
	}
	for i, idx := range m.searchResults {
		cursor := "  "
		line := tabStyle.Render(m.tabs[idx].title)
		if i == m.searchPick {
			cursor = "› "
			line = activeTabStyle.Render(m.tabs[idx].title)
		}
		b.WriteString(cursor + line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("type to filter · ↑/↓ select · enter open · esc cancel"))
	box := overlayStyle.Width(48).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) helpView() string {
	rows := [][2]string{
		{"^t", "new tab"},
		{"^d / ^w", "delete note (confirm)"},
		{"^p / ^n", "previous / next tab"},
		{"alt+1..9", "jump to tab N"},
		{"alt+H / alt+L", "move tab left / right"},
		{"^f", "find in workspace"},
		{"^o", "markdown preview (full)"},
		{"^b", "live split preview"},
		{"^r", "rename tab"},
		{"^e", "switch / new workspace"},
		{"^/ / F1", "this help"},
		{"^s", "save now"},
		{"^q / ^c", "save & quit"},
	}
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("pad — keys"))
	b.WriteString("\n\n")
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("%s  %s\n", keyStyle.Render(fmt.Sprintf("%-14s", r[0])), r[1]))
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("fallback switch: alt+h/l, ^←/^→, shift+←/→ · any key closes"))
	box := overlayStyle.Width(48).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) overlay(title, body, hint string) string {
	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		lipgloss.NewStyle().Bold(true).Render(title),
		body,
		footerStyle.Render(hint),
	)
	box := overlayStyle.Width(40).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) pickerView() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Switch workspace"))
	b.WriteString("\n\n")
	if len(m.wsList) == 0 {
		b.WriteString(footerStyle.Render("(none yet)"))
	}
	for i, w := range m.wsList {
		cursor := "  "
		line := tabStyle.Render(w)
		if i == m.wsPick {
			cursor = "› "
			line = activeTabStyle.Render(w)
		}
		b.WriteString(cursor + line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("↑/↓ move · enter open · n new · esc cancel"))
	box := overlayStyle.Width(40).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
