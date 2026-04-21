package preview

import (
	"bytes"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// ContentType controls how the preview renders content.
type ContentType int

const (
	ContentEmpty ContentType = iota
	ContentJSON
	ContentText
)

// Model holds the state of the preview panel.
type Model struct {
	content     string
	contentType ContentType
	scrollY     int
	width       int
	height      int
	title       string
}

func New() Model {
	return Model{}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetJSON sets the preview to display syntax-highlighted JSON.
func (m *Model) SetJSON(title, json string) {
	m.title = title
	m.content = highlightJSON(json)
	m.contentType = ContentJSON
	m.scrollY = 0
}

// SetText sets the preview to display plain text (e.g., OASF description).
func (m *Model) SetText(title, text string) {
	m.title = title
	m.content = text
	m.contentType = ContentText
	m.scrollY = 0
}

// SetEmpty clears the preview.
func (m *Model) SetEmpty(hint string) {
	m.title = ""
	m.content = hint
	m.contentType = ContentEmpty
	m.scrollY = 0
}

// ScrollDown scrolls the viewport down by n lines.
func (m *Model) ScrollDown(n int) {
	lines := strings.Split(m.content, "\n")
	max := len(lines) - m.height
	if max < 0 {
		max = 0
	}
	m.scrollY += n
	if m.scrollY > max {
		m.scrollY = max
	}
}

// ScrollUp scrolls the viewport up by n lines.
func (m *Model) ScrollUp(n int) {
	m.scrollY -= n
	if m.scrollY < 0 {
		m.scrollY = 0
	}
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFD700")).
		MaxWidth(m.width)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)

	var header string
	if m.title != "" {
		header = titleStyle.Render(m.title) + "\n"
	}

	bodyHeight := m.height
	if m.title != "" {
		bodyHeight--
	}
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	if m.contentType == ContentEmpty {
		return header + dimStyle.Render(m.content)
	}

	lines := strings.Split(m.content, "\n")
	start := m.scrollY
	if start > len(lines) {
		start = len(lines)
	}
	end := start + bodyHeight
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]

	// Truncate lines wider than panel.
	truncated := make([]string, len(visible))
	for i, line := range visible {
		// Strip ANSI for width calculation approximation; keep ANSI codes for display.
		if len(stripANSI(line)) > m.width {
			// Trim visible characters while preserving color codes is complex;
			// for now just render and let the terminal clip.
		}
		truncated[i] = line
	}

	scrollIndicator := ""
	if len(lines) > bodyHeight {
		pct := 0
		if len(lines)-bodyHeight > 0 {
			pct = (m.scrollY * 100) / (len(lines) - bodyHeight)
		}
		scrollIndicator = dimStyle.Render(strings.Repeat("─", m.width-8)) +
			dimStyle.Render(strings.Repeat(" ", 0)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(
				" ↕ "+itoa(pct)+"%",
			) + "\n"
	}

	return header + strings.Join(truncated, "\n") + "\n" + scrollIndicator
}

// highlightJSON uses chroma to produce ANSI-colored JSON.
func highlightJSON(src string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, src)
	if err != nil {
		return src
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return src
	}

	return buf.String()
}

// stripANSI removes ANSI escape sequences for length calculation.
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[pos:])
}
