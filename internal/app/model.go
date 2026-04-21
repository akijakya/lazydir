package app

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/oasf"
	"github.com/akijakya/lazydir/internal/panels/classes"
	"github.com/akijakya/lazydir/internal/panels/directory"
	"github.com/akijakya/lazydir/internal/panels/records"
	"github.com/akijakya/lazydir/internal/preview"
)

// ---- async message types ----

type recordsLoadedMsg struct {
	summaries []*dirclient.RecordSummary
	err       error
}

type recordJSONMsg struct {
	cid  string
	json string
	err  error
}

type oasfFetchedMsg struct {
	info *oasf.ClassInfo
	err  error
}

type connectResultMsg struct {
	client *dirclient.Client
	err    error
}

// ---- Model ----

// Model is the root Bubble Tea model for lazydir.
type Model struct {
	// panels
	dirPanel     directory.Model
	classesPanel classes.Model
	recordsPanel records.Model
	previewPanel preview.Model

	focused focusedPanel

	// data
	client    *dirclient.Client
	initCfg   dirclient.Config
	loading   bool
	loadMsg   string
	errorMsg  string

	// terminal size
	width  int
	height int
}

func New(cfg dirclient.Config) Model {
	m := Model{
		dirPanel:     directory.New(cfg),
		classesPanel: classes.New(),
		recordsPanel: records.New(),
		previewPanel: preview.New(),
		focused:      panelRecords,
		initCfg:      cfg,
		loading:      true,
		loadMsg:      "Connecting to " + cfg.ServerAddress + "…",
	}
	m.syncFocus()

	return m
}

func (m Model) Init() tea.Cmd {
	return connectCmd(m.initCfg)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanels()

	case connectResultMsg:
		return m.handleConnect(msg)

	case recordsLoadedMsg:
		return m.handleRecordsLoaded(msg)

	case recordJSONMsg:
		return m.handleRecordJSON(msg)

	case oasfFetchedMsg:
		return m.handleOASFFetched(msg)

	case directory.ConnectRequestMsg:
		m.loading = true
		m.loadMsg = "Connecting to " + msg.Config.ServerAddress + "…"
		m.errorMsg = ""
		if m.client != nil {
			m.client.Close()
			m.client = nil
		}

		return m, connectCmd(msg.Config)

	case classes.ClassSelectedMsg:
		return m.handleClassSelected(msg)

	case records.RecordSelectedMsg:
		return m.handleRecordSelected(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// ---- handlers ----

func (m Model) handleConnect(msg connectResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.err != nil {
		m.errorMsg = "Connection failed: " + msg.err.Error()
		m.dirPanel.SetConnected(false)

		return m, nil
	}

	m.client = msg.client
	m.dirPanel.SetConnected(true)
	m.dirPanel.SetConfig(msg.client.Config)
	m.loading = true
	m.loadMsg = "Loading records…"

	return m, loadRecordsCmd(msg.client)
}

func (m Model) handleRecordsLoaded(msg recordsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.err != nil {
		m.errorMsg = "Failed to load records: " + msg.err.Error()

		return m, nil
	}

	m.errorMsg = ""
	m.recordsPanel.SetRecords(msg.summaries)
	skills, domains, modules := dirclient.ExtractClasses(msg.summaries)
	m.classesPanel.SetItems(skills, domains, modules)
	m.previewPanel.SetEmpty("Select a record or class to preview.")

	return m, nil
}

func (m Model) handleRecordJSON(msg recordJSONMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.err != nil {
		m.previewPanel.SetText("Error", msg.err.Error())

		return m, nil
	}

	m.previewPanel.SetJSON(msg.cid, msg.json)

	return m, nil
}

func (m Model) handleOASFFetched(msg oasfFetchedMsg) (tea.Model, tea.Cmd) {
	m.loading = false

	if msg.err != nil {
		m.previewPanel.SetText("OASF Error", msg.err.Error())

		return m, nil
	}

	title := fmt.Sprintf("[%s] %s", msg.info.Type, msg.info.Name)
	m.previewPanel.SetText(title, msg.info.Description)

	return m, nil
}

func (m Model) handleClassSelected(msg classes.ClassSelectedMsg) (tea.Model, tea.Cmd) {
	m.recordsPanel.SetClassFilter(msg)

	if msg.Name == "" {
		m.previewPanel.SetEmpty("Select a record or class to preview.")

		return m, nil
	}

	var classType oasf.ClassType
	switch msg.Tab {
	case classes.TabSkills:
		classType = oasf.ClassTypeSkill
	case classes.TabDomains:
		classType = oasf.ClassTypeDomain
	case classes.TabModules:
		classType = oasf.ClassTypeModule
	}

	m.loading = true
	m.loadMsg = "Fetching OASF info…"

	return m, fetchOASFCmd(classType, msg.Name)
}

func (m Model) handleRecordSelected(msg records.RecordSelectedMsg) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}

	m.loading = true
	m.loadMsg = "Loading record…"

	return m, pullRecordCmd(m.client, msg.CID)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleGlobalKeys(msg); handled {
		return m, cmd
	}

	switch msg.String() {
	case "tab":
		m.focused = (m.focused + 1) % panelCount
		m.syncFocus()

		return m, nil
	case "shift+tab":
		m.focused = (m.focused - 1 + panelCount) % panelCount
		m.syncFocus()

		return m, nil
	case "1":
		m.focused = panelDirectory
		m.syncFocus()

		return m, nil
	case "2":
		m.focused = panelClasses
		m.syncFocus()

		return m, nil
	case "3":
		m.focused = panelRecords
		m.syncFocus()

		return m, nil
	case "r":
		if m.client != nil {
			m.loading = true
			m.loadMsg = "Refreshing records…"

			return m, loadRecordsCmd(m.client)
		}
	case "up", "k":
		if m.focused == panelPreview {
			m.previewPanel.ScrollUp(3)
		}
	case "down", "j":
		if m.focused == panelPreview {
			m.previewPanel.ScrollDown(3)
		}
	case "pgup":
		if m.focused == panelPreview {
			m.previewPanel.ScrollUp(m.height / 2)
		}
	case "pgdown":
		if m.focused == panelPreview {
			m.previewPanel.ScrollDown(m.height / 2)
		}
	}

	// Delegate to the focused panel.
	var cmd tea.Cmd

	switch m.focused {
	case panelDirectory:
		m.dirPanel, cmd = m.dirPanel.Update(msg)
	case panelClasses:
		m.classesPanel, cmd = m.classesPanel.Update(msg)
	case panelRecords:
		m.recordsPanel, cmd = m.recordsPanel.Update(msg)
	}

	return m, cmd
}

func (m *Model) syncFocus() {
	m.dirPanel.SetFocused(m.focused == panelDirectory)
	m.classesPanel.SetFocused(m.focused == panelClasses)
	m.recordsPanel.SetFocused(m.focused == panelRecords)
}

func (m *Model) resizePanels() {
	leftW := m.width / 3
	rightW := m.width - leftW - 1 // -1 for divider
	statusH := 1
	contentH := m.height - statusH

	// Left panels: split vertically.
	dirH := 4
	classH := (contentH - dirH) / 2
	recordH := contentH - dirH - classH

	m.dirPanel.SetSize(leftW, dirH)
	m.classesPanel.SetSize(leftW, classH)
	m.recordsPanel.SetSize(leftW, recordH)
	m.previewPanel.SetSize(rightW, contentH)
}

// ---- View ----

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Initializing…")
		v.AltScreen = true

		return v
	}

	leftW := m.width / 3
	rightW := m.width - leftW - 1
	statusH := 1
	contentH := m.height - statusH

	dirH := 4
	classH := (contentH - dirH) / 2
	recordH := contentH - dirH - classH

	// Panel border styles.
	focusedBorderColor := lipgloss.Color("#00CFFF")
	normalBorderColor := lipgloss.Color("#444444")
	titleStyle := func(label string, focused bool) string {
		color := normalBorderColor
		if focused {
			color = focusedBorderColor
		}
		return lipgloss.NewStyle().
			Foreground(color).
			Bold(focused).
			Render(label)
	}

	panelStyle := func(focused bool, w, h int) lipgloss.Style {
		color := normalBorderColor
		if focused {
			color = focusedBorderColor
		}

		return lipgloss.NewStyle().
			Width(w).
			Height(h).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Padding(0, 1)
	}

	// Render left panels.
	dirView := panelStyle(m.focused == panelDirectory, leftW-2, dirH-2).
		Render(titleStyle("[1] Directory", m.focused == panelDirectory) + "\n" + m.dirPanel.View())

	classView := panelStyle(m.focused == panelClasses, leftW-2, classH-2).
		Render(titleStyle("[2] Classes", m.focused == panelClasses) + "\n" + m.classesPanel.View())

	recView := panelStyle(m.focused == panelRecords, leftW-2, recordH-2).
		Render(titleStyle("[3] Records", m.focused == panelRecords) + "\n" + m.recordsPanel.View())

	leftCol := dirView + "\n" + classView + "\n" + recView

	// Render right panel.
	previewTitle := "[Preview]"
	if m.loading {
		previewTitle = "[Preview] " + m.loadMsg
	} else if m.errorMsg != "" {
		previewTitle = "[Preview] ⚠ " + m.errorMsg
	}

	rightView := panelStyle(m.focused == panelPreview, rightW-2, contentH-2).
		Render(titleStyle(previewTitle, m.focused == panelPreview) + "\n" + m.previewPanel.View())

	// Divider.
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#333333")).
		Render(strings.Repeat("│\n", contentH))

	// Join left and right side by side.
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, divider, rightView)

	// Status bar.
	statusBar := m.renderStatusBar()

	v := tea.NewView(content + "\n" + statusBar)
	v.AltScreen = true

	return v
}

func (m Model) renderStatusBar() string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	sep := sepStyle.Render("  ")

	hints := []string{
		keyStyle.Render("q") + ":quit",
		keyStyle.Render("tab") + ":focus",
		keyStyle.Render("↑↓") + ":nav",
		keyStyle.Render("enter") + ":select",
		keyStyle.Render("/") + ":filter",
		keyStyle.Render("c") + ":connect",
		keyStyle.Render("r") + ":refresh",
	}

	if m.focused == panelClasses {
		hints = append(hints, keyStyle.Render("tab")+" (here): switch tab")
	}

	bar := dimStyle.Render(strings.Join(hints, sep))
	// Pad to full width.
	visible := stripANSI(bar)
	if len(visible) < m.width {
		bar += strings.Repeat(" ", m.width-len(visible))
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1A1A2E")).
		Foreground(lipgloss.Color("#AAAAAA")).
		Width(m.width).
		Render(bar)
}

// stripANSI strips ANSI codes for length measurement.
func stripANSI(s string) string {
	var result strings.Builder
	inEsc := false

	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// ---- async commands ----

func connectCmd(cfg dirclient.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		c, err := dirclient.Connect(ctx, cfg)

		return connectResultMsg{client: c, err: err}
	}
}

func loadRecordsCmd(c *dirclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		summaries, err := c.ListAll(ctx)

		return recordsLoadedMsg{summaries: summaries, err: err}
	}
}

func pullRecordCmd(c *dirclient.Client, cid string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		json, err := c.PullJSON(ctx, cid)

		return recordJSONMsg{cid: cid, json: json, err: err}
	}
}

func fetchOASFCmd(classType oasf.ClassType, name string) tea.Cmd {
	return func() tea.Msg {
		info, err := oasf.Fetch(classType, name)

		return oasfFetchedMsg{info: info, err: err}
	}
}
