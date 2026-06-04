package main

import (
	"fmt"
	"io"
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
}

type autosaveMsg time.Time
type statusTickMsg time.Time
type splitRenderMsg int // carries the seq it was scheduled for, to coalesce

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
		m.tabs = append(m.tabs, tab{file: n.file, title: n.title, content: readNote(n.file)})
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
// "[ ]" <-> "[x]", or turns a plain line into "- [ ] ...".
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

// moveTab reorders the active tab by delta, swapping numeric prefixes on disk.
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

// runSearch recomputes which tabs match the current query (title + body).
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

// openPreview renders the active note as styled markdown into the viewport.
func (m *model) openPreview() {
	m.stash()
	body := ""
	if len(m.tabs) > 0 {
		body = m.tabs[m.active].content
	}
	if strings.TrimSpace(body) == "" {
		body = "*(empty note)*"
	}
	w := m.width - 2
	if w < 20 {
		w = 20
	}
	out := body
	if r := m.renderer(w); r != nil {
		if rendered, err := r.Render(body); err == nil {
			out = rendered
		}
	}
	m.preview = viewport.New(m.width, m.previewHeight())
	m.preview.SetContent(out)
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
	body := m.ta.Value()
	if strings.TrimSpace(body) == "" {
		body = "*(empty — type markdown on the left)*"
	}
	out := body
	if r := m.renderer(w); r != nil {
		if s, err := r.Render(body); err == nil {
			out = s
		}
	}
	m.splitVP.Width = w
	m.splitVP.Height = m.height - 4
	m.splitVP.SetContent(out)
	m.splitDirty = false
}

// scheduleSplitRender debounces: a render fires ~120ms after the last keystroke,
// coalescing bursts of typing into a single Glamour pass.
func scheduleSplitRender(seq int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return splitRenderMsg(seq) })
}

// autosaveInterval is set from config; 0 disables periodic autosave.
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
		m.toggleCheckbox()
		if m.splitPreview {
			m.refreshSplit()
		}
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
	if m.splitPreview {
		// debounce: mark dirty and schedule a single render after a pause
		m.splitDirty = true
		m.splitSeq++
		return m, tea.Batch(cmd, scheduleSplitRender(m.splitSeq))
	}
	return m, cmd
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

var version = "dev"

// glamourStyle is the preview/split markdown theme. Override with PAD_THEME
// (glamour standard styles: dark, light, dracula, tokyo-night, pink, notty…).
var glamourStyle = "dark"

// runPrint dumps a workspace's notes to stdout (no TUI). With tabFilter set,
// only notes whose title contains the filter (case-insensitive) are printed.
func runPrint(ws, tabFilter string) {
	notes, err := loadNotes(ws)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	filter := strings.ToLower(strings.TrimSpace(tabFilter))
	for _, n := range notes {
		if filter != "" && !strings.Contains(strings.ToLower(n.title), filter) {
			continue
		}
		fmt.Printf("## %s\n\n%s\n\n", n.title, readNote(n.file))
	}
}

// resolveContent returns the content for --new/--append: the --content flag if
// set, otherwise stdin when it's piped/redirected, otherwise empty.
func resolveContent(cfg config) string {
	if cfg.contentSet {
		return cfg.content
	}
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		b, _ := io.ReadAll(os.Stdin)
		return strings.TrimRight(string(b), "\n")
	}
	return ""
}

// runNew creates a note titled title with content, then prints its path.
func runNew(ws, title, content string) {
	path := nextPath(ws, title)
	if err := writeNote(path, content); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

// runAppend appends content to the first note matching title (by slug, then
// substring); if none matches it creates the note. Prints the path.
func runAppend(ws, title, content string) {
	notes, _ := loadNotes(ws)
	want := slug(title)
	var match string
	for _, n := range notes {
		if slug(n.title) == want {
			match = n.file
			break
		}
	}
	if match == "" {
		lt := strings.ToLower(strings.TrimSpace(title))
		for _, n := range notes {
			if strings.Contains(strings.ToLower(n.title), lt) {
				match = n.file
				break
			}
		}
	}
	if match == "" {
		runNew(ws, title, content)
		return
	}
	existing := readNote(match)
	sep := ""
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		sep = "\n"
	}
	if err := writeNote(match, existing+sep+content+"\n"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(match)
}

// runList prints workspace names, or the tab titles of a named workspace.
func runList(cfg config) {
	if cfg.wsExplicit {
		notes, _ := loadNotes(cfg.workspace)
		for _, n := range notes {
			fmt.Println(n.title)
		}
		return
	}
	for _, w := range listWorkspaces() {
		fmt.Println(w)
	}
}

func main() {
	cfg, ok := loadConfig(os.Args[1:])
	if !ok {
		return
	}
	dataRoot = cfg.dataDir
	autosaveInterval = cfg.autosave
	if t := os.Getenv("PAD_THEME"); t != "" {
		glamourStyle = t
	}

	switch {
	case cfg.list:
		runList(cfg)
		return
	case cfg.print:
		runPrint(cfg.workspace, cfg.tab)
		return
	case cfg.newTitle != "":
		runNew(cfg.workspace, cfg.newTitle, resolveContent(cfg))
		return
	case cfg.appendTo != "":
		runAppend(cfg.workspace, cfg.appendTo, resolveContent(cfg))
		return
	}

	p := tea.NewProgram(newModel(cfg.workspace), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
