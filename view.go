package main

import (
	"fmt"
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
	}

	tabBar := m.tabBar()
	footer := m.footer()
	body := m.ta.View()
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, body, footer)
}

func (m model) tabBar() string {
	ws := wsStyle.Render("⌂ " + m.ws)
	var tabs []string
	for i, t := range m.tabs {
		label := t.title
		if t.dirty {
			label += " •"
		}
		if i == m.active {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, tabStyle.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, append([]string{ws, " "}, tabs...)...)
	return row
}

func (m model) footer() string {
	help := footerStyle.Render(
		"^t new · ^d delete · ^{/^} prev/next · ^r rename · ^e workspace · ^s save · ^q quit",
	)
	if m.status != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, statusStyle.Render(m.status), "  ", help)
	}
	return help
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
