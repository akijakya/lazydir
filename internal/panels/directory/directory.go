package directory

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/akijakya/lazydir/internal/dirclient"
)

// ConnectRequestMsg is sent when the user confirms a new connection.
type ConnectRequestMsg struct {
	Config dirclient.Config
}

type inputField int

const (
	fieldAddr inputField = iota
	fieldAuth
	fieldCount
)

// Model is the directory panel model.
type Model struct {
	config    dirclient.Config
	connected bool
	width     int
	height    int
	focused   bool

	// connect dialog state
	dialogOpen    bool
	activeField   inputField
	inputAddr     string
	inputAuth     string
	cursorAddr    int
	cursorAuth    int
}

func New(cfg dirclient.Config) Model {
	return Model{
		config:    cfg,
		connected: true,
		inputAddr: cfg.ServerAddress,
		inputAuth: cfg.AuthMode,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.dialogOpen {
			return m.handleDialogKey(msg)
		}
		if m.focused {
			return m.handlePanelKey(msg)
		}
	}

	return m, nil
}

func (m Model) handlePanelKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		m.dialogOpen = true
		m.inputAddr = m.config.ServerAddress
		m.inputAuth = m.config.AuthMode
		m.cursorAddr = len(m.inputAddr)
		m.cursorAuth = len(m.inputAuth)
		m.activeField = fieldAddr
	}

	return m, nil
}

func (m Model) handleDialogKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dialogOpen = false
	case "tab":
		m.activeField = (m.activeField + 1) % fieldCount
	case "shift+tab":
		m.activeField = (m.activeField - 1 + fieldCount) % fieldCount
	case "enter":
		m.dialogOpen = false
		cfg := dirclient.Config{
			ServerAddress: m.inputAddr,
			AuthMode:      m.inputAuth,
		}
		m.config = cfg
		m.connected = false

		return m, func() tea.Msg {
			return ConnectRequestMsg{Config: cfg}
		}
	case "backspace":
		switch m.activeField {
		case fieldAddr:
			if m.cursorAddr > 0 {
				m.inputAddr = m.inputAddr[:m.cursorAddr-1] + m.inputAddr[m.cursorAddr:]
				m.cursorAddr--
			}
		case fieldAuth:
			if m.cursorAuth > 0 {
				m.inputAuth = m.inputAuth[:m.cursorAuth-1] + m.inputAuth[m.cursorAuth:]
				m.cursorAuth--
			}
		}
	case "left":
		switch m.activeField {
		case fieldAddr:
			if m.cursorAddr > 0 {
				m.cursorAddr--
			}
		case fieldAuth:
			if m.cursorAuth > 0 {
				m.cursorAuth--
			}
		}
	case "right":
		switch m.activeField {
		case fieldAddr:
			if m.cursorAddr < len(m.inputAddr) {
				m.cursorAddr++
			}
		case fieldAuth:
			if m.cursorAuth < len(m.inputAuth) {
				m.cursorAuth++
			}
		}
	default:
		ch := msg.String()
		if len(ch) == 1 {
			switch m.activeField {
			case fieldAddr:
				m.inputAddr = m.inputAddr[:m.cursorAddr] + ch + m.inputAddr[m.cursorAddr:]
				m.cursorAddr++
			case fieldAuth:
				m.inputAuth = m.inputAuth[:m.cursorAuth] + ch + m.inputAuth[m.cursorAuth:]
				m.cursorAuth++
			}
		}
	}

	return m, nil
}

func (m *Model) SetConnected(ok bool) {
	m.connected = ok
}

func (m *Model) SetConfig(cfg dirclient.Config) {
	m.config = cfg
	m.connected = true
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m Model) View() string {
	if m.dialogOpen {
		return m.renderDialog()
	}

	return m.renderPanel()
}

func (m Model) renderPanel() string {
	statusIcon := "●"
	statusColor := lipgloss.Color("#00FF7F")
	if !m.connected {
		statusIcon = "○"
		statusColor = lipgloss.Color("#FF6B6B")
	}

	iconStyle := lipgloss.NewStyle().Foreground(statusColor)
	addrStyle := lipgloss.NewStyle().Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)

	addr := m.config.ServerAddress
	if len(addr) > m.width-4 {
		addr = addr[:m.width-4]
	}

	authInfo := ""
	if m.config.AuthMode != "" {
		authInfo = "\n  auth: " + m.config.AuthMode
	}

	content := iconStyle.Render(statusIcon) + " " + addrStyle.Render(addr) +
		authInfo +
		"\n\n" + hintStyle.Render("c: connect to server")

	return content
}

func (m Model) renderDialog() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00CFFF")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)

	renderInput := func(label, value string, cursor int, active bool) string {
		style := inactiveStyle
		if active {
			style = activeStyle
		}
		// Insert cursor character.
		display := value[:cursor] + "|" + value[cursor:]
		return labelStyle.Render(label+": ") + style.Render(display)
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Connect to Directory") + "\n\n")
	sb.WriteString(renderInput("Server Address", m.inputAddr, m.cursorAddr, m.activeField == fieldAddr) + "\n")
	sb.WriteString(renderInput("Auth Mode     ", m.inputAuth, m.cursorAuth, m.activeField == fieldAuth) + "\n\n")
	sb.WriteString(hintStyle.Render("tab: next field  enter: connect  esc: cancel"))

	return sb.String()
}
