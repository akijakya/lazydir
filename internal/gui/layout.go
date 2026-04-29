package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

const (
	viewDirectory = "directory"
	viewFilters   = "filters"
	viewRecords   = "records"
	viewPreview   = "preview"
	viewOptions   = "options" // bottom bar: context keybindings (like lazygit)
	viewInput     = "input"   // shared editable prompt view, shown on demand
	viewHelp      = "help"    // ? popup overlay, shown on demand
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
		fmt.Fprintf(v, "\033[34m%s\033[0m", optionsBarText(focused, v.InnerWidth()))
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

	dirIcon := "\033[31m○\033[0m"
	if g.state.connected {
		dirIcon = "\033[32m●\033[0m"
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
	fmt.Fprintf(v, " \033[32m●\033[0m OASF:      %s\n", oasfAddr)
}
