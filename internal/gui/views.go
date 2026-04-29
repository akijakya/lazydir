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

// filterCategoryColor maps each filter category to a distinct ANSI color code
// so that applied selections are visually distinguishable at a glance.
var filterCategoryColor = map[filterCategory]string{
	filterSkills:      "\033[33m", // yellow
	filterDomains:     "\033[36m", // cyan
	filterModules:     "\033[35m", // magenta
	filterOASFVersion: "\033[32m", // green
	filterVersion:     "\033[34m", // blue
	filterAuthor:      "\033[91m", // bright red
	filterTrusted:     "\033[93m", // bright yellow
	filterVerified:    "\033[92m", // bright green
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
		color := filterCategoryColor[r.category]
		fmt.Fprintf(v, "    %s%s\033[0m\n", color, r.option)
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
// being edited, with checkmarks next to selected items. When an option has
// its inline description toggled (via 'i'), the description is rendered as
// indented green lines immediately below the option row.
func (app *Gui) renderFiltersOptions(g *gocui.Gui, v *gocui.View) {
	cat := app.state.filters.editing
	v.Title = "[2] Filters — " + cat.title()

	options := app.optionsFor(cat)
	applied := app.state.filters.applied[cat]
	fs := &app.state.filters

	if fs.optionsCursor < 0 {
		fs.optionsCursor = 0
	}
	if max := len(options) - 1; max >= 0 && fs.optionsCursor > max {
		fs.optionsCursor = max
	}

	if len(options) == 0 {
		fmt.Fprintln(v, " (no options available)")
	}

	const descIndent = "      "
	viewW, _ := v.Size()
	descW := viewW - len(descIndent) - 1
	if descW < 10 {
		descW = 10
	}

	lineNum := 0
	targetLine := 0
	for i, opt := range options {
		if i == fs.optionsCursor {
			targetLine = lineNum
		}

		mark := "[ ]"
		if applied[opt] {
			mark = "[x]"
		}
		fmt.Fprintf(v, " %s %s\n", mark, opt)
		lineNum++

		if opt == fs.inlineDesc {
			var descLines []string
			if fs.inlineDescLoading {
				descLines = []string{"loading…"}
			} else if fs.inlineDescText != "" {
				descLines = wrapText(fs.inlineDescText, descW)
			}
			for _, dl := range descLines {
				fmt.Fprintf(v, "%s\033[32m%s\033[0m\n", descIndent, dl)
				lineNum++
			}
		}
	}

	_ = v.SetOrigin(0, 0)
	_, viewH := v.Size()
	if targetLine >= viewH {
		_ = v.SetOrigin(0, targetLine-viewH+1)
		_ = v.SetCursor(0, viewH-1)
	} else {
		_ = v.SetCursor(0, targetLine)
	}
}

// renderRecordsView redraws the [3] Records panel and updates its title to
// reflect the current record count, stream state, and name filter.
func (app *Gui) renderRecordsView(g *gocui.Gui) {
	v, err := g.View(viewRecords)
	if err != nil {
		return
	}
	v.Clear()

	records := app.state.filteredRecords
	total := len(app.state.records)

	// Build the title: [3] Records (N)  /: foo
	title := "[3] Records"
	if total > 0 || app.state.stream == streamDone {
		if app.state.filterQuery != "" {
			title += fmt.Sprintf(" (%d/%d)", len(records), total)
		} else {
			title += fmt.Sprintf(" (%d)", total)
		}
	}
	if app.state.stream == streamErrored {
		title += " (error)"
	}
	if app.state.filterQuery != "" {
		title += fmt.Sprintf("  /: %s", app.state.filterQuery)
	}
	v.Title = title

	viewW, _ := v.Size()
	nameW := viewW - 14
	if nameW < 8 {
		nameW = 8
	}

	const infoIndent = "      "
	infoW := viewW - len(infoIndent) - 1
	if infoW < 10 {
		infoW = 10
	}

	lineNum := 0
	targetLine := 0
	for i, r := range records {
		if i == app.state.recordCursor {
			targetLine = lineNum
		}

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
		lineNum++

		if r.CID != "" && r.CID == app.state.recordInfoCID {
			var infoLines []string
			if app.state.recordInfoLoading {
				infoLines = []string{"\033[32mloading…\033[0m"}
			} else if app.state.recordInfoText != "" {
				infoLines = strings.Split(app.state.recordInfoText, "\n")
			}
			for _, il := range infoLines {
				fmt.Fprintf(v, "%s%s\n", infoIndent, il)
				lineNum++
			}
		}
	}

	_, viewH := v.Size()
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


// wrapText splits text into lines that fit within maxWidth, breaking on word
// boundaries where possible. Newlines in the input are preserved.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return strings.Split(text, "\n")
	}
	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		for len(paragraph) > maxWidth {
			cut := maxWidth
			for cut > 0 && paragraph[cut] != ' ' {
				cut--
			}
			if cut == 0 {
				cut = maxWidth
			}
			result = append(result, paragraph[:cut])
			paragraph = strings.TrimLeft(paragraph[cut:], " ")
		}
		if paragraph != "" {
			result = append(result, paragraph)
		}
	}
	return result
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
