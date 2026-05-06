package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

const (
	viewDirectory  = "directory"
	viewFilters    = "filters"
	viewRecords    = "records"
	viewPreview    = "preview"
	viewOptions    = "options"    // bottom bar: context keybindings (like lazygit)
	viewInput      = "input"      // shared editable prompt view, shown on demand
	viewHelp       = "help"       // ? popup overlay, shown on demand
	viewCopyMenu   = "copymenu"   // copy-options popup, shown on demand
	viewInfoPopup  = "infopopup"  // info popup, shown on demand (i key)
	viewServerMenu = "servermenu" // server selection popup, shown on demand (c key)
	viewAuthPopup  = "authpopup"  // OIDC auth popup, shown during device flow
)

// roundedFrame is a 6-rune set that gives every panel rounded corners: ╭─╮╰─╯
var roundedFrame = []rune{'─', '│', '╭', '╮', '╰', '╯'}

// listViews are the panels that show a highlighted cursor row.
var listViews = []string{viewDirectory, viewFilters, viewRecords}

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

	splitRatio := g.cfg.SplitRatio
	if splitRatio <= 0 || splitRatio >= 1 {
		splitRatio = 0.33
	}
	leftW := int(float64(maxX) * splitRatio)
	if leftW < 10 {
		leftW = 10
	}
	rightX0 := leftW

	optionsX1 := maxX - 1

	// [1] Connections shows two lines (Directory, OASF).
	// Height = frame(2) + 2 content lines.
	dirH := 4

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
		v.SelBgColor = g.theme.SelectedRowBg
		v.SelFgColor = gocui.ColorDefault
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
		v.SelBgColor = g.theme.SelectedRowBg
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
		v.SelBgColor = g.theme.SelectedRowBg
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

	// Server selection popup — centered over the connections panel.
	{
		smW := 40
		if leftW-4 > smW {
			smW = leftW - 4
		}
		smH := 6
		if g.state.serverMenuVisible {
			smH = len(g.state.serverMenuItems) + 2
		}
		smX0 := (leftW - smW) / 2
		smY0 := dirY0 + 1
		smX1 := smX0 + smW
		smY1 := smY0 + smH
		if smY1 > panelBottom {
			smY1 = panelBottom
		}
		if v, err := gui.SetView(viewServerMenu, smX0, smY0, smX1, smY1, 0); err != nil {
			if !gocui.IsUnknownView(err) {
				return err
			}
			v.Title = " Select server "
			v.Frame = true
			v.FrameRunes = roundedFrame
			v.Wrap = false
			v.Visible = false
			v.Highlight = true
			v.SelBgColor = g.theme.SelectedRowBg
			v.SelFgColor = gocui.ColorDefault
		} else {
			_, _ = gui.SetView(viewServerMenu, smX0, smY0, smX1, smY1, 0)
			v.Visible = g.state.serverMenuVisible
			if g.state.serverMenuVisible {
				v.Clear()
				for _, item := range g.state.serverMenuItems {
					fmt.Fprintln(v, " "+item)
				}
				_ = v.SetCursor(0, g.state.serverMenuCursor)
			}
		}
	}

	// OIDC auth popup — centered on screen.
	{
		authW := 50
		if leftW-4 < authW {
			authW = leftW - 4
		}
		authH := g.state.authPopupLines + 2
		if authH < 3 {
			authH = 3
		}
		authX0 := 2
		authY0 := dirY1 + 1
		if authY0+authH-1 > panelBottom {
			authY0 = panelBottom - authH + 1
		}
		if authY0 < 0 {
			authY0 = 0
		}
		authX1 := authX0 + authW
		if authX1 > leftW-1 {
			authX1 = leftW - 1
		}
		authY1 := authY0 + authH - 1
		if authY1 > panelBottom {
			authY1 = panelBottom
		}
		if v, err := gui.SetView(viewAuthPopup, authX0, authY0, authX1, authY1, 0); err != nil {
			if !gocui.IsUnknownView(err) {
				return err
			}
			v.Title = " OIDC Login "
			v.Frame = true
			v.FrameRunes = roundedFrame
			v.Wrap = true
			v.Visible = false
		} else {
			_, _ = gui.SetView(viewAuthPopup, authX0, authY0, authX1, authY1, 0)
		}
	}

	// Info popup — positioned under the selected item in the source panel,
	// sized dynamically to fit the content.
	infoW := leftW - 4
	if infoW < 30 {
		infoW = 30
	}
	infoH := g.infoPopupHeight(panelBottom, infoW-2)
	ipX0, ipY0, ipX1, ipY1 := 0, -(infoH + 1), infoW, -1
	if ipv, _ := gui.View(viewInfoPopup); ipv != nil && ipv.Visible {
		sourceView := viewRecords
		sourceY0 := recordY0
		switch g.state.infoPopupPanel {
		case viewDirectory:
			sourceView = viewDirectory
			sourceY0 = dirY0
		case viewFilters:
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
		if _, err := gui.SetCurrentView(viewDirectory); err != nil {
			return err
		}
		g.syncHighlight(gui, viewDirectory)
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
// connIcon returns the colored status indicator for a connection.
func (g *Gui) connIcon(status connStatus) string {
	switch status {
	case connOK:
		return g.theme.Color4 + "●" + g.theme.Reset
	case connFailed:
		return g.theme.Color6 + "●" + g.theme.Reset
	default:
		return g.theme.Color6 + "○" + g.theme.Reset
	}
}

// connSync returns " ↻" when the connection is in the trying state.
func connSync(status connStatus) string {
	if status == connTrying {
		return " ↻"
	}
	return ""
}

func (g *Gui) renderDirectory(gui *gocui.Gui) {
	v, err := gui.View(viewDirectory)
	if err != nil {
		return
	}
	v.Clear()

	dirSync := connSync(g.state.dirStatus)
	fmt.Fprintf(v, " %s Directory: %s%s\n", g.connIcon(g.state.dirStatus), g.state.serverAddr, dirSync)

	oasfAddr := g.state.oasfAddr
	if oasfAddr == "" {
		oasfAddr = "(not configured)"
	}
	fmt.Fprintf(v, " %s OASF:      %s%s", g.connIcon(g.state.oasfStatus), oasfAddr, connSync(g.state.oasfStatus))

	_ = v.SetCursor(0, g.state.connCursor)
}

// infoPopupHeight computes the popup frame height based on the current info
// content. Returns frame(2) + content lines, clamped to the available space.
func (g *Gui) infoPopupHeight(panelBottom, innerWidth int) int {
	maxH := panelBottom / 2
	if g.state.infoPopupPanel == viewRecords {
		maxH = panelBottom * 3 / 4
	}
	if maxH < 3 {
		maxH = 3
	}

	contentLines := g.infoPopupContentLines(innerWidth)

	h := contentLines + 2 // +2 for the frame top/bottom
	if h > maxH {
		h = maxH
	}
	return h
}

// infoPopupContentLines returns how many visual lines the info popup content
// will use, accounting for word-wrap at the given inner width.
func (g *Gui) infoPopupContentLines(wrapWidth int) int {
	if wrapWidth < 10 {
		wrapWidth = 10
	}

	var text string
	switch g.state.infoPopupPanel {
	case viewDirectory:
		text, _ = g.connInfoText()
	case viewFilters:
		if g.state.filters.inlineDescLoading {
			return 1
		}
		text = g.state.filters.inlineDescText
	case viewRecords:
		if g.state.recordInfoLoading {
			return 1
		}
		text = g.state.recordInfoText
	}

	if text == "" {
		return 1
	}
	return wrappedLineCount(text, wrapWidth)
}

// wrappedLineCount counts visual lines a string occupies at a given width,
// adding 1 line of margin so gocui never shows a scrollbar.
func wrappedLineCount(text string, width int) int {
	total := 0
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if len(line) == 0 {
			total++
			continue
		}
		total += (len(line)-1)/width + 1
	}
	return total + 1
}
