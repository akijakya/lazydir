package classes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TabType represents which taxonomy class tab is active.
type TabType int

const (
	TabSkills TabType = iota
	TabDomains
	TabModules
	tabCount
)

func (t TabType) String() string {
	switch t {
	case TabSkills:
		return "Skills"
	case TabDomains:
		return "Domains"
	case TabModules:
		return "Modules"
	}

	return ""
}

// ClassSelectedMsg is sent when the user selects a class entry.
type ClassSelectedMsg struct {
	Tab  TabType
	Name string // empty means "all"
}

// Model is the classes panel model.
type Model struct {
	activeTab TabType
	cursor    int
	focused   bool
	width     int
	height    int

	skills  []string
	domains []string
	modules []string
}

func New() Model {
	return Model{}
}

func (m *Model) SetItems(skills, domains, modules []string) {
	m.skills = skills
	m.domains = domains
	m.modules = modules
	m.cursor = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m Model) ActiveTab() TabType { return m.activeTab }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	items := m.currentItems()

	switch msg.String() {
	case "tab":
		m.activeTab = (m.activeTab + 1) % tabCount
		m.cursor = 0
	case "shift+tab":
		m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items) {
			m.cursor++
		}
	case "enter":
		name := ""
		if m.cursor > 0 && m.cursor-1 < len(items) {
			name = items[m.cursor-1]
		}

		return m, func() tea.Msg {
			return ClassSelectedMsg{Tab: m.activeTab, Name: name}
		}
	case "esc":
		m.cursor = 0

		return m, func() tea.Msg {
			return ClassSelectedMsg{Tab: m.activeTab, Name: ""}
		}
	}

	return m, nil
}

func (m Model) currentItems() []string {
	switch m.activeTab {
	case TabSkills:
		return m.skills
	case TabDomains:
		return m.domains
	case TabModules:
		return m.modules
	}

	return nil
}

func (m Model) View() string {
	// Render tabs.
	var tabs []string
	for i := TabType(0); i < tabCount; i++ {
		label := i.String()
		if i == m.activeTab {
			tabs = append(tabs, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00CFFF")).
				Underline(true).
				Render(label))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Render(label))
		}
	}

	tabBar := strings.Join(tabs, lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(" │ "))

	items := m.currentItems()
	visibleHeight := m.height - 3 // subtract tab bar + padding
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Compute scroll offset so cursor is always visible.
	scrollOffset := 0
	totalItems := len(items) + 1 // +1 for "(All)" entry
	if m.cursor >= visibleHeight {
		scrollOffset = m.cursor - visibleHeight + 1
	}

	var rows []string
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	allStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Italic(true)

	// "(All)" entry at index 0.
	allLabel := "(All)"
	if m.cursor == 0 {
		rows = append(rows, cursorStyle.Render("> "+allLabel))
	} else {
		rows = append(rows, allStyle.Render("  "+allLabel))
	}

	for i, item := range items {
		listIdx := i + 1
		if listIdx < scrollOffset || listIdx-scrollOffset >= visibleHeight {
			continue
		}
		label := truncate(item, m.width-4)
		if m.cursor == listIdx {
			rows = append(rows, cursorStyle.Render("> "+label))
		} else {
			rows = append(rows, normalStyle.Render("  "+label))
		}
	}

	countInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render(fmt.Sprintf(" (%d)", totalItems-1))

	return tabBar + countInfo + "\n" + strings.Join(rows, "\n")
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}

	return s[:max-1] + "…"
}
