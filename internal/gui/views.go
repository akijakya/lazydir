package gui

import (
	"bytes"
	"fmt"

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

// renderFiltersView redraws the [2] Filters panel in either list or options
// mode. It also updates the panel title to reflect the current mode.
func (app *Gui) renderFiltersView(g *gocui.Gui) {
	v, err := g.View(viewFilters)
	if err != nil {
		return
	}
	v.Clear()

	if app.state.filters.mode == filterModeOptions {
		app.renderFiltersOptions(g, v)
		return
	}
	app.renderFiltersList(g, v)
}

// renderFiltersList draws the default mode: each filter category as a row,
// with any applied selections rendered as indented child rows.
func (app *Gui) renderFiltersList(g *gocui.Gui, v *gocui.View) {
	v.Title = "[2] Filters"

	rows := app.listRows()
	for _, r := range rows {
		if r.option == "" {
			fmt.Fprintln(v, " "+r.category.title())
			continue
		}
		fmt.Fprintln(v, "    "+r.option)
	}

	// Clamp cursor to valid range and render position.
	if app.state.filters.listCursor < 0 {
		app.state.filters.listCursor = 0
	}
	if max := len(rows) - 1; max >= 0 && app.state.filters.listCursor > max {
		app.state.filters.listCursor = max
	}

	_ = v.SetOrigin(0, 0)
	targetLine := app.state.filters.listCursor
	_, viewH := v.Size()
	if targetLine >= viewH {
		_ = v.SetOrigin(0, targetLine-viewH+1)
		_ = v.SetCursor(0, viewH-1)
	} else {
		_ = v.SetCursor(0, targetLine)
	}
}

// renderFiltersOptions draws the options sub-view for the category currently
// being edited, with checkmarks next to selected items.
func (app *Gui) renderFiltersOptions(g *gocui.Gui, v *gocui.View) {
	cat := app.state.filters.editing
	title := "[2] Filters — " + cat.title()
	// While the records stream is still in flight, the option list grows;
	// surface that so the user doesn't think the missing values are absent.
	if !cat.boolean() && (app.state.stream == streamLoading || app.state.stream == streamStreaming) {
		title += " (still loading…)"
	}
	v.Title = title

	options := app.optionsFor(cat)
	applied := app.state.filters.applied[cat]

	if len(options) == 0 {
		fmt.Fprintln(v, " (no options available)")
	}

	for _, opt := range options {
		mark := "[ ]"
		if applied[opt] {
			mark = "[x]"
		}
		fmt.Fprintf(v, " %s %s\n", mark, opt)
	}

	if app.state.filters.optionsCursor < 0 {
		app.state.filters.optionsCursor = 0
	}
	if max := len(options) - 1; max >= 0 && app.state.filters.optionsCursor > max {
		app.state.filters.optionsCursor = max
	}

	_ = v.SetOrigin(0, 0)
	targetLine := app.state.filters.optionsCursor
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
	total := len(app.state.records)

	// Header line: count + stream state + name filter indicator
	count := fmt.Sprintf("%d/%d", len(records), total)
	if app.state.filterQuery == "" {
		// No name query active: collapse the redundant N/N to a single number.
		count = fmt.Sprintf("%d", total)
	}
	state := ""
	switch app.state.stream {
	case streamLoading:
		state = " loading…"
	case streamStreaming:
		state = " streaming…"
	case streamErrored:
		state = " error: " + app.state.streamErr
	}
	filterInfo := ""
	if app.state.filterQuery != "" {
		filterInfo = fmt.Sprintf("  filter: %s", app.state.filterQuery)
	}
	fmt.Fprintf(v, " (%s)%s%s\n", count, state, filterInfo)

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
func (app *Gui) renderPreviewText(g *gocui.Gui, subtitle, content string) {
	v, err := g.View(viewPreview)
	if err != nil {
		return
	}
	v.Title = previewTitle(subtitle)
	v.Clear()
	_ = v.SetOrigin(0, 0)
	fmt.Fprint(v, content)
}

// renderPreviewJSON sets syntax-highlighted JSON in the preview panel.
func (app *Gui) renderPreviewJSON(g *gocui.Gui, subtitle, jsonStr string) {
	v, err := g.View(viewPreview)
	if err != nil {
		return
	}
	v.Title = previewTitle(subtitle)
	v.Clear()
	_ = v.SetOrigin(0, 0)
	fmt.Fprint(v, highlightJSON(jsonStr))
}

// previewTitle formats the preview panel title, always keeping the [0] Preview
// prefix and appending the current item name when one is provided.
func previewTitle(subtitle string) string {
	if subtitle == "" {
		return "[0] Preview"
	}
	return "[0] Preview — " + subtitle
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
