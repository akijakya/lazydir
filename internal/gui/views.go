package gui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/jesseduffield/gocui"
)

// lazydirStyle is a chroma style derived from "tango" with punctuation
// remapped to plain white so that { } [ ] ( ) , : ; are readable on dark
// terminals instead of the default bold-black that tango uses.
var lazydirStyle = func() *chroma.Style {
	base := styles.Get("tango")
	if base == nil {
		base = styles.Fallback
	}
	b := base.Builder()
	// "bold" alone inherits the token's foreground but ensures it is bright;
	// "#ffffff bold" forces bright white — readable on any dark background.
	b.Add(chroma.Punctuation, "#ffffff bold")
	s, err := b.Build()
	if err != nil {
		return base
	}
	return s
}()

// renderClassesView redraws the [2] Classes panel.
func (app *Gui) renderClassesView(g *gocui.Gui) {
	v, err := g.View(viewClasses)
	if err != nil {
		return
	}
	v.Clear()

	items := app.currentClassItems()

	// Tab bar
	tabs := []classTab{tabSkills, tabDomains, tabModules}
	var tabLabels []string
	for _, t := range tabs {
		label := t.String()
		if t == app.state.activeTab {
			label = "[" + label + "]"
		}
		tabLabels = append(tabLabels, label)
	}
	fmt.Fprintln(v, " "+strings.Join(tabLabels, "  "))
	fmt.Fprintln(v, strings.Repeat("─", 30))

	// "(All)" entry at cursor index 0
	allLabel := " (All)"
	fmt.Fprintln(v, allLabel)

	for _, item := range items {
		fmt.Fprintln(v, " "+item)
	}

	// Position the view cursor to match state.
	// +2 for the tab bar + separator line, +1 for (All) = offset 0 maps to line 2
	_ = v.SetOrigin(0, 0)
	targetLine := app.state.classCursor + 2 // +2 for tab bar + separator
	_, viewH := v.Size()
	if targetLine >= viewH {
		_ = v.SetOrigin(0, targetLine-viewH+1)
		_ = v.SetCursor(0, viewH-1)
	} else {
		_ = v.SetCursor(0, targetLine)
	}
}

// renderRecordsView redraws the [3] Records panel.
func (app *Gui) renderRecordsView(g *gocui.Gui) {
	v, err := g.View(viewRecords)
	if err != nil {
		return
	}
	v.Clear()

	records := app.state.filteredRecords
	total := len(app.state.allRecords)

	// Header line: count + filter indicator
	filterInfo := ""
	if app.state.filterQuery != "" {
		filterInfo = fmt.Sprintf("  filter: %s", app.state.filterQuery)
	}
	fmt.Fprintf(v, " (%d/%d)%s\n", len(records), total, filterInfo)

	viewW, _ := v.Size()
	nameW := viewW - 14
	if nameW < 8 {
		nameW = 8
	}

	for _, r := range records {
		name := r.Name
		if name == "" {
			name = r.CID
		}
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		version := r.Version
		if version == "" {
			version = "n/a"
		}
		fmt.Fprintf(v, " %-*s  %s\n", nameW, name, version)
	}

	// Position cursor.
	cursor := app.state.recordCursor
	_, viewH := v.Size()
	targetLine := cursor + 1 // +1 for header line
	if targetLine >= viewH {
		_ = v.SetOrigin(0, targetLine-viewH+1)
		_ = v.SetCursor(0, viewH-1)
	} else {
		_ = v.SetOrigin(0, 0)
		_ = v.SetCursor(0, targetLine)
	}
}

// renderPreviewText sets plain text content in the preview panel.
func (app *Gui) renderPreviewText(g *gocui.Gui, title, content string) {
	v, err := g.View(viewPreview)
	if err != nil {
		return
	}
	v.Title = title
	v.Clear()
	_ = v.SetOrigin(0, 0)
	fmt.Fprint(v, content)
}

// renderPreviewJSON sets syntax-highlighted JSON in the preview panel.
func (app *Gui) renderPreviewJSON(g *gocui.Gui, title, jsonStr string) {
	v, err := g.View(viewPreview)
	if err != nil {
		return
	}
	v.Title = title
	v.Clear()
	_ = v.SetOrigin(0, 0)
	fmt.Fprint(v, highlightJSON(jsonStr))
}


// highlightJSON returns ANSI-colored JSON using chroma with the terminal's
// own color palette so the output blends with the user's theme.
func highlightJSON(src string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal16")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iter, err := lexer.Tokenise(nil, src)
	if err != nil {
		return src
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, lazydirStyle, iter); err != nil {
		return src
	}

	return buf.String()
}
