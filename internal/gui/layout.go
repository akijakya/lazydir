package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

const (
	viewDirectory = "directory"
	viewClasses   = "classes"
	viewRecords   = "records"
	viewPreview   = "preview"
	viewOptions   = "options" // bottom-left: context keybindings (like lazygit)
	viewInfo      = "info"    // bottom-right: connection/version info
	viewInput     = "input"   // shared editable prompt view, shown on demand
	viewHelp      = "help"    // ? popup overlay, shown on demand
)

// roundedFrame is a 6-rune set that gives every panel rounded corners: ╭─╮╰─╯
var roundedFrame = []rune{'─', '│', '╭', '╮', '╰', '╯'}

// listViews are the panels that show a highlighted cursor row.
var listViews = []string{viewClasses, viewRecords}

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

	// Bottom bar split: options (left, flexible) | info (right, fixed)
	infoText := g.infoText()
	infoW := len(infoText) + 2
	if infoW > maxX/2 {
		infoW = maxX / 2
	}
	optionsX1 := maxX - infoW - 2

	dirH := 5
	classH := (panelBottom - dirH) / 2
	recY0 := dirH + classH

	// [1] Directory panel
	if v, err := gui.SetView(viewDirectory, 0, 0, leftW-1, dirH-1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[1] Directory"
		v.Frame = true
		v.Wrap = false
		v.Highlight = false
		v.FrameRunes = roundedFrame
		g.renderDirectory(gui)
	}

	// [2] Classes panel
	if v, err := gui.SetView(viewClasses, 0, dirH, leftW-1, recY0-1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[2] Classes"
		v.Frame = true
		v.Highlight = false
		v.SelBgColor = gocui.Get256Color(8)
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = roundedFrame
	}

	// [3] Records panel — extends to panelBottom
	if v, err := gui.SetView(viewRecords, 0, recY0, leftW-1, panelBottom, 0); err != nil {
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

	// Bottom-right: info bar — properties set every layout call (no frame)
	if _, err := gui.SetView(viewInfo, optionsX1+1, bottomY0, maxX-1, bottomY1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
	}
	if v, _ := gui.View(viewInfo); v != nil {
		v.Frame = false
		v.FgColor = gocui.ColorDefault | gocui.AttrBold
	}

	// Shared input prompt — always exists, shown/hidden on demand.
	inputY0 := dirH - 3
	inputY1 := dirH - 1
	if v, err := gui.SetView(viewInput, 1, inputY0, leftW-2, inputY1, 0); err != nil {
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
// and disables it on all others — giving a clear visual focus cue.
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
		fmt.Fprint(v, optionsBarText(focused, v.InnerWidth()))
	}

	if v, err := gui.View(viewInfo); err == nil {
		v.Clear()
		fmt.Fprint(v, g.infoText())
	}
}

// infoText returns the short string shown in the right info bar.
func (g *Gui) infoText() string {
	if g.state.connected {
		return "● " + g.state.serverAddr
	}
	return "○ not connected"
}

func (g *Gui) renderDirectory(gui *gocui.Gui) {
	v, err := gui.View(viewDirectory)
	if err != nil {
		return
	}
	v.Clear()

	icon := "●"
	if !g.state.connected {
		icon = "○"
	}

	fmt.Fprintf(v, " %s %s\n", icon, g.state.serverAddr)
	if g.state.authMode != "" {
		fmt.Fprintf(v, " auth: %s\n", g.state.authMode)
	}
	fmt.Fprintf(v, "\n c: connect to a server")
}
