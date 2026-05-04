package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

const (
	viewDirectory = "directory"
	viewFilters   = "filters"
	viewRecords   = "records"
	viewPreview   = "preview"
	viewOptions   = "options"   // bottom bar: context keybindings (like lazygit)
	viewInput     = "input"     // shared editable prompt view, shown on demand
	viewHelp      = "help"      // ? popup overlay, shown on demand
	viewCopyMenu  = "copymenu"  // copy-options popup, shown on demand
	viewInfoPopup = "infopopup" // info popup, shown on demand (i key)
)

// roundedFrame is a 6-rune set that gives every panel rounded corners: ╭─╮╰─╯
var roundedFrame = []rune{'─', '│', '╭', '╮', '╰', '╯'}

// listViews are the panels that show a highlighted cursor row.
var listViews = []string{viewFilters, viewRecords}

// layout is the gocui Manager — called on every redraw/resize.
func (g *Gui) layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	// The bottom bar occupies the last row (maxY-1).
	// All panels extend down to maxY-2, so they sit flush against it.
	// For frameless views, content cell y=0 is placed at screen row v.y0+1.
	// So to render at terminal row (maxY-1), we need v.y0=maxY-2, v.y1=maxY.
	bottomY0 := maxY - 2 // y0 for bottom bar views
	bottomY1 := maxY     // y1 for bottom bar views (one past screen, allowed)
	panelBottom := maxY - 2

	leftW := maxX / 3
	rightX0 := leftW

	optionsX1 := maxX - 1

	// [1] Connections shows two lines (Directory, OASF) plus an optional
	// auth-mode line. Height = frame(2) + content lines.
	dirH := 4
	if g.state.authMode != "" {
		dirH = 5
	}

	// The input prompt, when visible, steals a 3-row slot on the left column
	// above the panel that requested it (the "host"). Panels below the host
	// are all shifted down by inputSlot rows.
	const inputSlot = 3
	inputHost := g.inputHostView()
	showInput := g.state.inputVisible

	var (
		dirY0, dirY1     = 0, dirH - 1
		filtersY0        int
		filtersY1        int
		recordY0         int
		inputOnLeft      = false
		inputX0, inputY0 = 0, 0
		inputX1, inputY1 = 0, 0
	)

	// Reserve the prompt slot before deciding panel heights.
	slotOffsetDir := 0     // shift applied to Connections (only when host=viewDirectory)
	slotOffsetFilters := 0 // shift applied to Filters
	slotOffsetRecord := 0  // shift applied to Records
	if showInput {
		switch inputHost {
		case viewDirectory:
			slotOffsetDir = inputSlot
			slotOffsetFilters = inputSlot
			slotOffsetRecord = inputSlot
		case viewFilters:
			slotOffsetFilters = inputSlot
			slotOffsetRecord = inputSlot
		case viewRecords, viewPreview, "":
			// Default: above Records (used by `/` filter too).
			slotOffsetRecord = inputSlot
		}
		inputOnLeft = (inputHost != viewPreview)
	}

	// Connections panel placement.
	dirY0 += slotOffsetDir
	dirY1 += slotOffsetDir

	// Filters panel: vertical space left between Connections and Records.
	filtersY0 = dirY1 + 1 + (slotOffsetFilters - slotOffsetDir)
	filtersH := (panelBottom - dirY1 - 1 - slotOffsetFilters + slotOffsetDir) / 2
	if filtersH < 3 {
		filtersH = 3
	}
	filtersY1 = filtersY0 + filtersH - 1

	// Records panel starts right after Filters, plus any extra slot above it.
	recordY0 = filtersY1 + 1 + (slotOffsetRecord - slotOffsetFilters)
	if recordY0 >= panelBottom {
		recordY0 = panelBottom - 3
	}

	// Input prompt coordinates, when shown on the left column.
	if showInput && inputOnLeft {
		inputX0 = 0
		inputX1 = leftW - 1
		switch inputHost {
		case viewDirectory:
			inputY0 = 0
		case viewFilters:
			inputY0 = dirY1 + 1
		default:
			inputY0 = filtersY1 + 1
		}
		inputY1 = inputY0 + inputSlot - 1
	}

	// [1] Connections panel
	if v, err := gui.SetView(viewDirectory, 0, dirY0, leftW-1, dirY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[1] Connections"
		v.Frame = true
		v.Wrap = false
		v.Highlight = false
		v.FrameRunes = roundedFrame
		g.renderDirectory(gui)
	}

	// [2] Filters panel
	if v, err := gui.SetView(viewFilters, 0, filtersY0, leftW-1, filtersY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[2] Filters"
		v.Frame = true
		v.Highlight = false
		v.SelBgColor = gocui.Get256Color(8)
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = roundedFrame
	}

	// [3] Records panel — extends to panelBottom
	if v, err := gui.SetView(viewRecords, 0, recordY0, leftW-1, panelBottom, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[3] Records"
		v.Frame = true
		v.Highlight = false
		v.SelBgColor = gocui.Get256Color(8)
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = roundedFrame
	}

	// Preview panel — extends to panelBottom
	if v, err := gui.SetView(viewPreview, rightX0, 0, maxX-1, panelBottom, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[0] Preview"
		v.Frame = true
		v.Wrap = true
		v.FrameRunes = roundedFrame
		v.CanScrollPastBottom = true
	}

	// Bottom-left: options bar — properties set every layout call (no frame)
	if _, err := gui.SetView(viewOptions, 0, bottomY0, optionsX1, bottomY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
	}
	if v, _ := gui.View(viewOptions); v != nil {
		v.Frame = false
	}

	// Shared input prompt — lives as a regular row above its host panel.
	// We still create the view at startup so focus/keybindings work; when
	// not visible it's parked off-screen above the viewport.
	if !showInput {
		inputX0, inputX1 = 0, leftW-1
		inputY0, inputY1 = -inputSlot-1, -2
	}
	if v, err := gui.SetView(viewInput, inputX0, inputY0, inputX1, inputY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Editable = true
		v.KeybindOnEdit = true
		v.Wrap = false
		v.Frame = true
		v.FrameRunes = roundedFrame
		v.Visible = false
	}
	if v, _ := gui.View(viewInput); v != nil && v.Visible {
		_, _ = gui.SetViewOnTop(viewInput)
	}

	// Help popup overlay — centered, shown/hidden on demand.
	helpW := 54
	helpH := 22
	helpX0 := (maxX - helpW) / 2
	helpY0 := (maxY - helpH) / 2
	if v, err := gui.SetView(viewHelp, helpX0, helpY0, helpX0+helpW, helpY0+helpH, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = " Keybindings  (esc/? to close) "
		v.Frame = true
		v.FrameRunes = roundedFrame
		v.Wrap = false
		v.Visible = false
	}

	// Copy-menu popup — positioned under the selected record when visible.
	copyW := 28
	copyH := 4 // frame(2) + 2 content lines
	cmX0, cmY0, cmX1, cmY1 := 0, -(copyH + 1), copyW, -1
	if cmv, _ := gui.View(viewCopyMenu); cmv != nil && cmv.Visible {
		if rv, rvErr := gui.View(viewRecords); rvErr == nil {
			_, cy := rv.Cursor()
			screenY := recordY0 + 1 + cy
			cmX0 = 2
			cmY0 = screenY + 1
			if cmY0+copyH-1 > panelBottom {
				cmY0 = screenY - copyH
			}
			cmX1 = cmX0 + copyW
			if cmX1 > leftW-1 {
				cmX1 = leftW - 1
			}
			cmY1 = cmY0 + copyH - 1
		}
	}
	if v, err := gui.SetView(viewCopyMenu, cmX0, cmY0, cmX1, cmY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = " Copy options "
		v.Frame = true
		v.FrameRunes = roundedFrame
		v.Wrap = false
		v.Visible = false
	}

	// Info popup — positioned under the selected item in the source panel,
	// sized dynamically to fit the content.
	infoW := leftW - 4
	if infoW < 30 {
		infoW = 30
	}
	infoH := g.infoPopupHeight(panelBottom)
	ipX0, ipY0, ipX1, ipY1 := 0, -(infoH + 1), infoW, -1
	if ipv, _ := gui.View(viewInfoPopup); ipv != nil && ipv.Visible {
		sourceView := viewRecords
		sourceY0 := recordY0
		if g.state.infoPopupPanel == viewFilters {
			sourceView = viewFilters
			sourceY0 = filtersY0
		}
		if sv, svErr := gui.View(sourceView); svErr == nil {
			_, cy := sv.Cursor()
			screenY := sourceY0 + 1 + cy
			ipX0 = 2
			ipY0 = screenY + 1
			if ipY0+infoH-1 > panelBottom {
				ipY0 = screenY - infoH
			}
			if ipY0 < 0 {
				ipY0 = 0
			}
			ipX1 = ipX0 + infoW
			if ipX1 > leftW-1 {
				ipX1 = leftW - 1
			}
			ipY1 = ipY0 + infoH - 1
			if ipY1 > panelBottom {
				ipY1 = panelBottom
			}
		}
	}
	if v, err := gui.SetView(viewInfoPopup, ipX0, ipY0, ipX1, ipY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = " Info "
		v.Frame = true
		v.FrameRunes = roundedFrame
		v.Wrap = true
		v.Visible = false
	}

	// First-time init: populate bottom bar and set focus.
	if gui.CurrentView() == nil {
		g.renderStatus(gui)
		g.renderDirectory(gui)
		if _, err := gui.SetCurrentView(viewRecords); err != nil {
			return err
		}
		g.syncHighlight(gui, viewRecords)
	}

	return nil
}

// syncHighlight enables the row-highlight cursor on the focused list view only,
// and disables it on all others — giving a clear visual focus cue. The focused
// panel's border is painted green via g.SelFrameColor set at init time.
func (g *Gui) syncHighlight(gui *gocui.Gui, focused string) {
	for _, name := range listViews {
		v, err := gui.View(name)
		if err != nil {
			continue
		}
		v.Highlight = (name == focused)
	}
}

func (g *Gui) renderStatus(gui *gocui.Gui) {
	focused := ""
	if cv := gui.CurrentView(); cv != nil {
		focused = cv.Name()
	}

	if v, err := gui.View(viewOptions); err == nil {
		v.Clear()
		fmt.Fprintf(v, "%s%s%s", g.theme.Color5, optionsBarText(focused, v.InnerWidth()), g.theme.Reset)
	}
}

// inputHostView resolves which left-column panel the input prompt should
// attach itself to. The prompt is inserted above the host, shifting the
// panels below it down.
func (g *Gui) inputHostView() string {
	host := g.state.prevView
	switch host {
	case viewDirectory, viewFilters, viewRecords:
		return host
	default:
		return viewRecords
	}
}

// renderDirectory refreshes the [1] Connections panel with both the Directory
// and OASF endpoints the app is currently talking to. A sync indicator is
// appended to the Directory line while the records stream is in flight.
func (g *Gui) renderDirectory(gui *gocui.Gui) {
	v, err := gui.View(viewDirectory)
	if err != nil {
		return
	}
	v.Clear()

	dirIcon := g.theme.Color6 + "○" + g.theme.Reset
	if g.state.connected {
		dirIcon = g.theme.Color4 + "●" + g.theme.Reset
	}

	sync := ""
	switch g.state.stream {
	case streamLoading, streamStreaming:
		sync = " ↻"
	}

	fmt.Fprintf(v, " %s Directory: %s%s\n", dirIcon, g.state.serverAddr, sync)
	if g.state.authMode != "" {
		fmt.Fprintf(v, "   auth: %s\n", g.state.authMode)
	}

	oasfAddr := g.state.oasfAddr
	if oasfAddr == "" {
		oasfAddr = "(not configured)"
	}
	fmt.Fprintf(v, " %s●%s OASF:      %s\n", g.theme.Color4, g.theme.Reset, oasfAddr)
}

// infoPopupHeight computes the popup frame height based on the current info
// content. Returns frame(2) + content lines, clamped between 4 and the
// available vertical space. The records panel uses a more generous max
// (3/4 of the screen) so annotation-heavy records can display fully.
func (g *Gui) infoPopupHeight(panelBottom int) int {
	const minH = 4

	maxH := panelBottom / 2
	if g.state.infoPopupPanel == viewRecords {
		maxH = panelBottom * 3 / 4
	}
	if maxH < minH {
		maxH = minH
	}

	var text string
	var loading bool
	switch g.state.infoPopupPanel {
	case viewFilters:
		text = g.state.filters.inlineDescText
		loading = g.state.filters.inlineDescLoading
	case viewRecords:
		text = g.state.recordInfoText
		loading = g.state.recordInfoLoading
	}

	contentLines := 1
	if loading {
		contentLines = 1
	} else if text != "" {
		contentLines = strings.Count(text, "\n") + 1
	}

	h := contentLines + 2 // +2 for the frame top/bottom
	if h < minH {
		h = minH
	}
	if h > maxH {
		h = maxH
	}
	return h
}
