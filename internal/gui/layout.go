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
	viewStatus    = "status"
	viewInput     = "input" // shared editable prompt view, shown on demand
)

// roundedFrame is a 6-rune set that gives every panel rounded corners: ╭─╮╰─╯
var roundedFrame = []rune{'─', '│', '╭', '╮', '╰', '╯'}

// listViews are the panels that show a highlighted cursor row.
var listViews = []string{viewClasses, viewRecords}

// layout is the gocui Manager — called on every redraw/resize.
func (g *Gui) layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	leftW := maxX / 3
	rightX0 := leftW
	rightX1 := maxX - 1
	statusY := maxY - 2

	dirH := 5
	classH := (statusY - dirH) / 2
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
		v.Highlight = false // toggled by syncHighlight
		v.SelBgColor = gocui.Get256Color(8) // base16 color 8: bright black / dark grey
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = roundedFrame
	}

	// [3] Records panel
	if v, err := gui.SetView(viewRecords, 0, recY0, leftW-1, statusY-1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[3] Records"
		v.Frame = true
		v.Highlight = false // toggled by syncHighlight
		v.SelBgColor = gocui.Get256Color(8) // base16 color 8: bright black / dark grey
		v.SelFgColor = gocui.ColorDefault
		v.FrameRunes = roundedFrame
	}

	// Preview panel (right 2/3)
	if v, err := gui.SetView(viewPreview, rightX0, 0, rightX1, statusY-1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = "[0] Preview"
		v.Frame = true
		v.Wrap = true
		v.FrameRunes = roundedFrame
		v.CanScrollPastBottom = true
	}

	// Status bar — no frame, no rounding
	if v, err := gui.SetView(viewStatus, 0, statusY, maxX-1, maxY-1, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Frame = false
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

	// First-time init: write status bar and set focus.
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
	v, err := gui.View(viewStatus)
	if err != nil {
		return
	}
	v.Clear()
	fmt.Fprint(v, "q:quit  tab:focus  ↑↓/jk:nav  enter:select  /:filter  h/l:tab  c:connect  r:refresh")
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
