package records

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/panels/classes"
)

// RecordSelectedMsg is sent when a record is highlighted (for preview).
type RecordSelectedMsg struct {
	CID string
}

// Model is the records panel model.
type Model struct {
	allRecords     []*dirclient.RecordSummary
	filtered       []*dirclient.RecordSummary
	cursor         int
	focused        bool
	width          int
	height         int
	filterMode     bool
	filterQuery    string
	activeClass    classes.ClassSelectedMsg
}

func New() Model {
	return Model{}
}

func (m *Model) SetRecords(records []*dirclient.RecordSummary) {
	m.allRecords = records
	m.applyFilter()
	m.cursor = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
	if !f {
		m.filterMode = false
	}
}

func (m *Model) SetClassFilter(msg classes.ClassSelectedMsg) {
	m.activeClass = msg
	m.cursor = 0
	m.applyFilter()
}

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
	if m.filterMode {
		return m.handleFilterKey(msg)
	}

	switch msg.String() {
	case "/":
		m.filterMode = true
		m.filterQuery = ""
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.filtered) {
			cid := m.filtered[m.cursor].CID

			return m, func() tea.Msg {
				return RecordSelectedMsg{CID: cid}
			}
		}
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterQuery = ""
		m.applyFilter()
		m.cursor = 0
	case "enter":
		m.filterMode = false
		m.applyFilter()
		m.cursor = 0
	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
			m.applyFilter()
			m.cursor = 0
		}
	default:
		ch := msg.String()
		if len(ch) == 1 {
			m.filterQuery += ch
			m.applyFilter()
			m.cursor = 0
		}
	}

	return m, nil
}

func (m *Model) applyFilter() {
	classFilter := m.activeClass

	var base []*dirclient.RecordSummary

	if classFilter.Name == "" {
		base = m.allRecords
	} else {
		for _, r := range m.allRecords {
			if matchesClass(r, classFilter) {
				base = append(base, r)
			}
		}
	}

	if m.filterQuery == "" {
		m.filtered = base

		return
	}

	query := strings.ToLower(m.filterQuery)
	var out []*dirclient.RecordSummary

	for _, r := range base {
		if strings.Contains(strings.ToLower(r.Name), query) {
			out = append(out, r)
		}
	}

	m.filtered = out
}

func matchesClass(r *dirclient.RecordSummary, sel classes.ClassSelectedMsg) bool {
	switch sel.Tab {
	case classes.TabSkills:
		for _, s := range r.Skills {
			if s == sel.Name {
				return true
			}
		}
	case classes.TabDomains:
		for _, d := range r.Domains {
			if d == sel.Name {
				return true
			}
		}
	case classes.TabModules:
		for _, m := range r.Modules {
			if m == sel.Name {
				return true
			}
		}
	}

	return false
}

func (m Model) View() string {
	visibleHeight := m.height - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	scrollOffset := 0
	if m.cursor >= visibleHeight {
		scrollOffset = m.cursor - visibleHeight + 1
	}

	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00CFFF"))

	var rows []string

	for i, r := range m.filtered {
		if i < scrollOffset || i-scrollOffset >= visibleHeight {
			continue
		}

		name := r.Name
		if name == "" {
			name = r.CID
		}
		name = truncate(name, m.width-14)

		version := r.Version
		if version == "" {
			version = "n/a"
		}
		version = truncate(version, 10)

		line := fmt.Sprintf("%-*s %s", m.width-14, name, dimStyle.Render(version))
		if i == m.cursor {
			rows = append(rows, cursorStyle.Render("> "+line))
		} else {
			rows = append(rows, normalStyle.Render("  "+line))
		}
	}

	if len(rows) == 0 {
		rows = append(rows, dimStyle.Render("  (no records)"))
	}

	filterLine := ""
	if m.filterMode {
		filterLine = "\n" + filterStyle.Render("filter: "+m.filterQuery+"█")
	} else if m.filterQuery != "" {
		filterLine = "\n" + dimStyle.Render("filter: "+m.filterQuery)
	}

	countLine := dimStyle.Render(fmt.Sprintf(" (%d/%d)", len(m.filtered), len(m.allRecords)))

	return countLine + "\n" + strings.Join(rows, "\n") + filterLine
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
