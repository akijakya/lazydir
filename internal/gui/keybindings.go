package gui

import (
	"context"
	"fmt"
	"strings"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/oasf"
	"github.com/jesseduffield/gocui"
)

func (app *Gui) bindKeys(g *gocui.Gui) error {
	// ── Global ───────────────────────────────────────────────────────────────
	for _, key := range []interface{}{gocui.KeyCtrlC, 'q'} {
		if err := g.SetKeybinding("", key, gocui.ModNone, quit); err != nil {
			return err
		}
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, app.cycleFocusForward); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyBacktab, gocui.ModNone, app.cycleFocusBackward); err != nil {
		return err
	}
	if err := g.SetKeybinding("", '1', gocui.ModNone, app.focusView(viewDirectory)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", '2', gocui.ModNone, app.focusView(viewClasses)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", '3', gocui.ModNone, app.focusView(viewRecords)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'r', gocui.ModNone, app.refresh); err != nil {
		return err
	}

	// ── Input prompt (shared) ────────────────────────────────────────────────
	// enter and esc are static; the actual work is done via the onConfirm/onCancel
	// callbacks set at open time — no dynamic key binding ever needed.
	if err := g.SetKeybinding(viewInput, gocui.KeyEnter, gocui.ModNone, app.inputConfirm); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewInput, gocui.KeyEsc, gocui.ModNone, app.inputCancel); err != nil {
		return err
	}
	// Let ctrl+c quit even from the input view.
	if err := g.SetKeybinding(viewInput, gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	// ── Directory panel ──────────────────────────────────────────────────────
	if err := g.SetKeybinding(viewDirectory, 'c', gocui.ModNone, app.openConnectDialog); err != nil {
		return err
	}

	// ── Classes panel ────────────────────────────────────────────────────────
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewClasses, key, gocui.ModNone, app.classCursorUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewClasses, key, gocui.ModNone, app.classCursorDown); err != nil {
			return err
		}
	}
	if err := g.SetKeybinding(viewClasses, gocui.KeyEnter, gocui.ModNone, app.classSelect); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewClasses, gocui.KeyEsc, gocui.ModNone, app.classClearFilter); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewClasses, 'l', gocui.ModNone, app.classNextTab); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewClasses, 'h', gocui.ModNone, app.classPrevTab); err != nil {
		return err
	}

	// ── Records panel ────────────────────────────────────────────────────────
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewRecords, key, gocui.ModNone, app.recordCursorUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewRecords, key, gocui.ModNone, app.recordCursorDown); err != nil {
			return err
		}
	}
	if err := g.SetKeybinding(viewRecords, gocui.KeyEnter, gocui.ModNone, app.recordSelect); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewRecords, '/', gocui.ModNone, app.openFilterDialog); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewRecords, gocui.KeyEsc, gocui.ModNone, app.clearFilter); err != nil {
		return err
	}

	// ── Preview panel ────────────────────────────────────────────────────────
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewPreview, key, gocui.ModNone, app.previewScrollUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewPreview, key, gocui.ModNone, app.previewScrollDown); err != nil {
			return err
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// ── Input prompt handlers ─────────────────────────────────────────────────────

func (app *Gui) inputConfirm(g *gocui.Gui, v *gocui.View) error {
	value := strings.TrimSpace(v.TextArea.GetContent())
	cb := app.state.onInputConfirm
	app.closeInput()
	if cb != nil {
		cb(value)
	}
	return nil
}

func (app *Gui) inputCancel(g *gocui.Gui, v *gocui.View) error {
	cb := app.state.onInputCancel
	app.closeInput()
	if cb != nil {
		cb()
	}
	return nil
}

// ── Focus helpers ─────────────────────────────────────────────────────────────

var focusOrder = []string{viewDirectory, viewClasses, viewRecords, viewPreview}

// focusTo sets the current view and updates highlight state on list panels.
func (app *Gui) focusTo(g *gocui.Gui, name string) error {
	_, err := g.SetCurrentView(name)
	if err != nil {
		return err
	}
	app.syncHighlight(g, name)
	return nil
}

func (app *Gui) focusView(name string) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		return app.focusTo(g, name)
	}
}

func (app *Gui) cycleFocusForward(g *gocui.Gui, v *gocui.View) error {
	return app.cycleFocus(g, 1)
}

func (app *Gui) cycleFocusBackward(g *gocui.Gui, v *gocui.View) error {
	return app.cycleFocus(g, -1)
}

func (app *Gui) cycleFocus(g *gocui.Gui, dir int) error {
	cur := g.CurrentView()
	curName := ""
	if cur != nil {
		curName = cur.Name()
	}
	idx := 0
	for i, name := range focusOrder {
		if name == curName {
			idx = i
			break
		}
	}
	next := (idx + dir + len(focusOrder)) % len(focusOrder)
	return app.focusTo(g, focusOrder[next])
}

// ── Classes panel handlers ────────────────────────────────────────────────────

func (app *Gui) classCursorUp(g *gocui.Gui, v *gocui.View) error {
	if app.state.classCursor > 0 {
		app.state.classCursor--
		app.renderClassesView(g)
	}
	return nil
}

func (app *Gui) classCursorDown(g *gocui.Gui, v *gocui.View) error {
	items := app.currentClassItems()
	if app.state.classCursor < len(items) {
		app.state.classCursor++
		app.renderClassesView(g)
	}
	return nil
}

func (app *Gui) classSelect(g *gocui.Gui, v *gocui.View) error {
	items := app.currentClassItems()
	cursor := app.state.classCursor

	if cursor == 0 {
		app.state.selectedClass = ""
	} else if cursor-1 < len(items) {
		app.state.selectedClass = items[cursor-1]
		var ct oasf.ClassType
		switch app.state.activeTab {
		case tabSkills:
			ct = oasf.ClassTypeSkill
		case tabDomains:
			ct = oasf.ClassTypeDomain
		case tabModules:
			ct = oasf.ClassTypeModule
		}
		app.state.selectedClassType = ct
		go app.fetchOASF(ct, app.state.selectedClass)
	}

	app.state.recordCursor = 0
	app.applyFilters()
	app.renderRecordsView(g)
	return nil
}

func (app *Gui) classClearFilter(g *gocui.Gui, v *gocui.View) error {
	app.state.selectedClass = ""
	app.state.classCursor = 0
	app.state.recordCursor = 0
	app.applyFilters()
	app.renderClassesView(g)
	app.renderRecordsView(g)
	return nil
}

func (app *Gui) classNextTab(g *gocui.Gui, v *gocui.View) error {
	app.state.activeTab = classTab((int(app.state.activeTab) + 1) % 3)
	app.state.classCursor = 0
	app.renderClassesView(g)
	return nil
}

func (app *Gui) classPrevTab(g *gocui.Gui, v *gocui.View) error {
	app.state.activeTab = classTab((int(app.state.activeTab) + 2) % 3)
	app.state.classCursor = 0
	app.renderClassesView(g)
	return nil
}

// ── Records panel handlers ────────────────────────────────────────────────────

func (app *Gui) recordCursorUp(g *gocui.Gui, v *gocui.View) error {
	if app.state.recordCursor > 0 {
		app.state.recordCursor--
		app.renderRecordsView(g)
	}
	return nil
}

func (app *Gui) recordCursorDown(g *gocui.Gui, v *gocui.View) error {
	if app.state.recordCursor < len(app.state.filteredRecords)-1 {
		app.state.recordCursor++
		app.renderRecordsView(g)
	}
	return nil
}

func (app *Gui) recordSelect(g *gocui.Gui, v *gocui.View) error {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return nil
	}
	cid := records[app.state.recordCursor].CID
	if cid == "" {
		return nil
	}
	go app.pullRecord(cid)
	return nil
}

func (app *Gui) openFilterDialog(g *gocui.Gui, v *gocui.View) error {
	app.openInput("Filter records (/)", app.state.filterQuery,
		func(value string) {
			app.state.filterQuery = value
			app.state.recordCursor = 0
			app.applyFilters()
			app.g.Update(func(g *gocui.Gui) error {
				app.renderRecordsView(g)
				return nil
			})
		},
		nil,
	)
	return nil
}

func (app *Gui) clearFilter(g *gocui.Gui, v *gocui.View) error {
	app.state.filterQuery = ""
	app.state.recordCursor = 0
	app.applyFilters()
	app.renderRecordsView(g)
	return nil
}

// ── Preview panel handlers ────────────────────────────────────────────────────

func (app *Gui) previewScrollUp(g *gocui.Gui, v *gocui.View) error {
	pv, err := g.View(viewPreview)
	if err != nil {
		return nil
	}
	_, oy := pv.Origin()
	if oy > 0 {
		_ = pv.SetOrigin(0, oy-3)
	}
	return nil
}

func (app *Gui) previewScrollDown(g *gocui.Gui, v *gocui.View) error {
	pv, err := g.View(viewPreview)
	if err != nil {
		return nil
	}
	_, oy := pv.Origin()
	_ = pv.SetOrigin(0, oy+3)
	return nil
}

// ── Directory / connect dialog ────────────────────────────────────────────────

func (app *Gui) openConnectDialog(g *gocui.Gui, v *gocui.View) error {
	app.openInput("Connect to server (enter addr)", app.state.serverAddr,
		func(addr string) {
			if addr == "" {
				return
			}
			cfg := dirclient.Config{
				ServerAddress: addr,
				AuthMode:      app.state.authMode,
				TLSSkipVerify: app.cfg.TLSSkipVerify,
				TLSCAFile:     app.cfg.TLSCAFile,
				TLSCertFile:   app.cfg.TLSCertFile,
				TLSKeyFile:    app.cfg.TLSKeyFile,
				AuthToken:     app.cfg.AuthToken,
			}
			go app.connect(cfg)
		},
		nil,
	)
	return nil
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func (app *Gui) refresh(g *gocui.Gui, v *gocui.View) error {
	if app.state.client == nil {
		return nil
	}
	go app.loadRecords(app.state.client)
	return nil
}

// ── Async actions ─────────────────────────────────────────────────────────────

func (app *Gui) pullRecord(cid string) {
	app.g.Update(func(g *gocui.Gui) error {
		app.setStatus("Loading record " + cid[:min(16, len(cid))] + "…")
		return nil
	})

	ctx := context.Background()
	jsonStr, err := app.state.client.PullJSON(ctx, cid)
	app.g.Update(func(g *gocui.Gui) error {
		if err != nil {
			app.renderPreviewText(g, "Error", err.Error())
			app.setStatus("Failed to load record: " + err.Error())
			return nil
		}
		app.renderPreviewJSON(g, cid, jsonStr)
		app.setStatus(fmt.Sprintf("Showing record %s", cid[:min(20, len(cid))]))
		return nil
	})
}

func (app *Gui) fetchOASF(ct oasf.ClassType, name string) {
	app.g.Update(func(g *gocui.Gui) error {
		app.setStatus("Fetching OASF info for " + name + "…")
		return nil
	})

	info, err := oasf.Fetch(ct, name)
	app.g.Update(func(g *gocui.Gui) error {
		if err != nil {
			app.renderPreviewText(g, "OASF Error", err.Error())
			app.setStatus("OASF fetch failed: " + err.Error())
			return nil
		}
		title := fmt.Sprintf("[%s] %s", info.Type, info.Name)
		app.renderPreviewText(g, title, info.Description)
		app.setStatus("Showing OASF description for " + name)
		return nil
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
