package gui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
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
	if err := g.SetKeybinding("", '2', gocui.ModNone, app.focusView(viewFilters)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", '3', gocui.ModNone, app.focusView(viewRecords)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", '0', gocui.ModNone, app.focusView(viewPreview)); err != nil {
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

	// ── Connections panel ────────────────────────────────────────────────────
	if err := g.SetKeybinding(viewDirectory, 'c', gocui.ModNone, app.openConnectDialog); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewDirectory, 'o', gocui.ModNone, app.openOASFDialog); err != nil {
		return err
	}

	// ── Filters panel ────────────────────────────────────────────────────────
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewFilters, key, gocui.ModNone, app.filterCursorUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewFilters, key, gocui.ModNone, app.filterCursorDown); err != nil {
			return err
		}
	}
	if err := g.SetKeybinding(viewFilters, gocui.KeyEnter, gocui.ModNone, app.filterEnter); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, gocui.KeySpace, gocui.ModNone, app.filterToggleOption); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, gocui.KeyEsc, gocui.ModNone, app.filterEsc); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, '/', gocui.ModNone, app.filterOpenSearch); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, 'i', gocui.ModNone, app.filterToggleInfo); err != nil {
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
	if err := g.SetKeybinding(viewRecords, 'i', gocui.ModNone, app.recordToggleInfo); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewRecords, 'y', gocui.ModNone, app.openCopyMenu); err != nil {
		return err
	}

	// ── Info popup ──────────────────────────────────────────────────────────
	if err := g.SetKeybinding(viewInfoPopup, gocui.KeyEsc, gocui.ModNone, app.closeInfoPopup); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewInfoPopup, 'i', gocui.ModNone, app.closeInfoPopup); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewInfoPopup, gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewInfoPopup, key, gocui.ModNone, app.previewScrollUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewInfoPopup, key, gocui.ModNone, app.previewScrollDown); err != nil {
			return err
		}
	}

	// ── Copy menu popup ─────────────────────────────────────────────────────
	if err := g.SetKeybinding(viewCopyMenu, 'c', gocui.ModNone, app.copyCID); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewCopyMenu, 'a', gocui.ModNone, app.copyRecordJSON); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewCopyMenu, gocui.KeyEsc, gocui.ModNone, app.closeCopyMenu); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewCopyMenu, gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
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
	if err := g.SetKeybinding(viewPreview, gocui.MouseWheelUp, gocui.ModNone, app.previewScrollUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewPreview, gocui.MouseWheelDown, gocui.ModNone, app.previewScrollDown); err != nil {
		return err
	}

	// Mouse wheel scrolling on list panels
	if err := g.SetKeybinding(viewFilters, gocui.MouseWheelUp, gocui.ModNone, app.filterCursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, gocui.MouseWheelDown, gocui.ModNone, app.filterCursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewRecords, gocui.MouseWheelUp, gocui.ModNone, app.recordCursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewRecords, gocui.MouseWheelDown, gocui.ModNone, app.recordCursorDown); err != nil {
		return err
	}

	// Mouse click focuses the clicked panel; records and filters get
	// specialised handlers that also update the cursor / open categories.
	for _, name := range []string{viewDirectory, viewPreview} {
		n := name
		if err := g.SetKeybinding(n, gocui.MouseLeft, gocui.ModNone, app.focusView(n)); err != nil {
			return err
		}
	}
	if err := g.SetKeybinding(viewRecords, gocui.MouseLeft, gocui.ModNone, app.recordMouseClick); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, gocui.MouseLeft, gocui.ModNone, app.filterMouseClick); err != nil {
		return err
	}

	// ? opens help popup for all main panels
	for _, name := range []string{"", viewDirectory, viewFilters, viewRecords, viewPreview} {
		if err := g.SetKeybinding(name, '?', gocui.ModNone, app.openHelp); err != nil {
			return err
		}
	}
	// esc and ? close the help popup
	if err := g.SetKeybinding(viewHelp, gocui.KeyEsc, gocui.ModNone, app.closeHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewHelp, '?', gocui.ModNone, app.closeHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewHelp, gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	// scroll help popup
	for _, key := range []interface{}{gocui.KeyArrowUp, 'k'} {
		if err := g.SetKeybinding(viewHelp, key, gocui.ModNone, app.previewScrollUp); err != nil {
			return err
		}
	}
	for _, key := range []interface{}{gocui.KeyArrowDown, 'j'} {
		if err := g.SetKeybinding(viewHelp, key, gocui.ModNone, app.previewScrollDown); err != nil {
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

var focusOrder = []string{viewDirectory, viewFilters, viewRecords, viewPreview}

// focusTo sets the current view and updates highlight state on list panels.
func (app *Gui) focusTo(g *gocui.Gui, name string) error {
	_, err := g.SetCurrentView(name)
	if err != nil {
		return err
	}
	app.syncHighlight(g, name)
	app.renderStatus(g)
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

// ── Filters panel handlers ────────────────────────────────────────────────────

func (app *Gui) filterMouseClick(g *gocui.Gui, v *gocui.View) error {
	app.hideInfoPopupIfVisible(g)
	if err := app.focusTo(g, viewFilters); err != nil {
		return err
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	idx := oy + cy

	rows := app.filteredListRows()
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	app.state.filters.listCursor = idx

	row := rows[idx]
	if row.option == "" {
		app.state.filters.expanded[row.category] = !app.state.filters.expanded[row.category]
		app.clearInlineDesc()
	} else {
		app.toggleApplied(row.category, row.option)
		app.startRecordsStream()
	}
	app.renderFiltersView(g)
	return nil
}

func (app *Gui) filterCursorUp(g *gocui.Gui, v *gocui.View) error {
	if app.state.filters.listCursor > 0 {
		app.state.filters.listCursor--
	}
	app.renderFiltersView(g)
	return nil
}

func (app *Gui) filterCursorDown(g *gocui.Gui, v *gocui.View) error {
	rows := app.filteredListRows()
	if app.state.filters.listCursor < len(rows)-1 {
		app.state.filters.listCursor++
	}
	app.renderFiltersView(g)
	return nil
}

// filterEnter toggles expand/collapse on category headers and toggles
// filter selection on option rows.
func (app *Gui) filterEnter(g *gocui.Gui, v *gocui.View) error {
	rows := app.filteredListRows()
	if app.state.filters.listCursor >= len(rows) {
		return nil
	}
	row := rows[app.state.filters.listCursor]

	if row.option == "" {
		app.state.filters.expanded[row.category] = !app.state.filters.expanded[row.category]
		app.clearInlineDesc()
	} else {
		app.toggleApplied(row.category, row.option)
		app.startRecordsStream()
	}
	app.renderFiltersView(g)
	return nil
}

// filterToggleOption toggles filter selection on the option under the cursor.
// On category headers it does nothing (use enter to expand/collapse).
func (app *Gui) filterToggleOption(g *gocui.Gui, v *gocui.View) error {
	rows := app.filteredListRows()
	if app.state.filters.listCursor >= len(rows) {
		return nil
	}
	row := rows[app.state.filters.listCursor]
	if row.option == "" {
		return nil
	}
	app.toggleApplied(row.category, row.option)
	app.startRecordsStream()
	app.renderFiltersView(g)
	return nil
}

// filterEsc clears the search query when active. Otherwise it does nothing —
// filters are removed by toggling them off with enter.
func (app *Gui) filterEsc(g *gocui.Gui, v *gocui.View) error {
	if app.state.filters.filterQuery != "" {
		app.state.filters.filterQuery = ""
		app.state.filters.listCursor = 0
		app.renderFiltersView(g)
		return nil
	}
	return nil
}

// filterOpenSearch opens the input prompt to search filter options across all
// non-boolean categories simultaneously.
func (app *Gui) filterOpenSearch(g *gocui.Gui, v *gocui.View) error {
	prevQuery := app.state.filters.filterQuery
	app.openInput("Search filters (/)", app.state.filters.filterQuery,
		func(value string) {
			app.g.Update(func(g *gocui.Gui) error {
				app.state.filters.filterQuery = value
				app.state.filters.listCursor = 0
				app.renderFiltersView(g)
				return nil
			})
		},
		func() {
			app.g.Update(func(g *gocui.Gui) error {
				app.state.filters.filterQuery = prevQuery
				app.state.filters.listCursor = 0
				app.renderFiltersView(g)
				return nil
			})
		},
		func(value string) {
			app.state.filters.filterQuery = value
			app.state.filters.listCursor = 0
			app.renderFiltersView(app.g)
		},
	)
	return nil
}

// ── Records panel handlers ────────────────────────────────────────────────────

func (app *Gui) recordMouseClick(g *gocui.Gui, v *gocui.View) error {
	app.hideInfoPopupIfVisible(g)
	if err := app.focusTo(g, viewRecords); err != nil {
		return err
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	idx := oy + cy
	if idx >= 0 && idx < len(app.state.filteredRecords) {
		app.state.recordCursor = idx
		app.renderRecordsView(g)
		app.autoPreviewRecord(g)
	}
	return nil
}

func (app *Gui) recordCursorUp(g *gocui.Gui, v *gocui.View) error {
	if app.state.recordCursor > 0 {
		app.state.recordCursor--
		app.renderRecordsView(g)
		app.autoPreviewRecord(g)
	}
	return nil
}

func (app *Gui) recordCursorDown(g *gocui.Gui, v *gocui.View) error {
	if app.state.recordCursor < len(app.state.filteredRecords)-1 {
		app.state.recordCursor++
		app.renderRecordsView(g)
		app.autoPreviewRecord(g)
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
	prevQuery := app.state.filterQuery
	app.openInput("Filter records (/)", app.state.filterQuery,
		func(value string) {
			app.g.Update(func(g *gocui.Gui) error {
				app.state.filterQuery = value
				app.state.recordCursor = 0
				app.applyNameFilter()
				app.renderRecordsView(g)
				return nil
			})
		},
		func() {
			app.g.Update(func(g *gocui.Gui) error {
				app.state.filterQuery = prevQuery
				app.state.recordCursor = 0
				app.applyNameFilter()
				app.renderRecordsView(g)
				return nil
			})
		},
		func(value string) {
			app.state.filterQuery = value
			app.state.recordCursor = 0
			app.applyNameFilter()
			app.renderRecordsView(app.g)
		},
	)
	return nil
}

func (app *Gui) clearFilter(g *gocui.Gui, v *gocui.View) error {
	app.state.filterQuery = ""
	app.state.recordCursor = 0
	app.applyNameFilter()
	app.renderRecordsView(g)
	return nil
}

// recordToggleInfo opens/closes the info popup for the currently highlighted
// record, fetching details via the directory's PullInfo RPC.
func (app *Gui) recordToggleInfo(g *gocui.Gui, v *gocui.View) error {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return nil
	}
	cid := records[app.state.recordCursor].CID
	if cid == "" {
		return nil
	}

	if app.state.recordInfoCID == cid {
		return app.closeInfoPopup(g, v)
	}

	app.state.recordInfoCID = cid
	app.state.recordInfoText = ""
	app.state.recordInfoLoading = true
	app.openInfoPopup(g, viewRecords)

	go app.fetchRecordInfo(cid)
	return nil
}

func (app *Gui) fetchRecordInfo(cid string) {
	client := app.state.client
	if client == nil {
		return
	}

	info, err := client.PullInfo(context.Background(), cid)
	app.g.Update(func(g *gocui.Gui) error {
		if app.state.recordInfoCID != cid {
			return nil
		}
		app.state.recordInfoLoading = false
		if err != nil {
			app.state.recordInfoText = err.Error()
		} else {
			app.state.recordInfoText = formatRecordInfo(info, app.theme)
		}
		app.renderInfoPopup(g)
		return nil
	})
}

// formatRecordInfo renders a RecordInfo as colored, human-readable lines.
// The CID is omitted here because it's already shown in the preview panel title.
func formatRecordInfo(info *dirclient.RecordInfo, t Theme) string {
	var sb strings.Builder
	first := true

	if len(info.Annotations) > 0 {
		fmt.Fprintf(&sb, "%sAnnotations:%s", t.Color1, t.Reset)
		first = false
		keys := make([]string, 0, len(info.Annotations))
		for k := range info.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "\n%s%s%s:%s %s", indent1, t.Color1, k, t.Reset, info.Annotations[k])
		}
	}

	if info.SchemaVersion != "" {
		if !first {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%sSchema version:%s %s", t.Color4, t.Reset, info.SchemaVersion)
		first = false
	}
	if info.CreatedAt != "" {
		if !first {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%sCreated at:%s %s", t.Color3, t.Reset, info.CreatedAt)
	}

	return sb.String()
}

func (app *Gui) clearRecordInlineDesc() {
	app.state.recordInfoCID = ""
	app.state.recordInfoText = ""
	app.state.recordInfoLoading = false
}

// ── Preview panel handlers ────────────────────────────────────────────────────

func (app *Gui) previewScrollUp(g *gocui.Gui, v *gocui.View) error {
	return scrollViewUp(g, v)
}

func (app *Gui) previewScrollDown(g *gocui.Gui, v *gocui.View) error {
	return scrollViewDown(g, v)
}

func scrollViewUp(_ *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	_, oy := v.Origin()
	if oy > 0 {
		_ = v.SetOrigin(0, oy-3)
	}
	return nil
}

func scrollViewDown(_ *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	_, oy := v.Origin()
	_ = v.SetOrigin(0, oy+3)
	return nil
}

// ── Connections: directory and OASF server dialogs ────────────────────────────

func (app *Gui) openConnectDialog(g *gocui.Gui, v *gocui.View) error {
	app.openInput("Connect to directory (enter addr)", app.state.serverAddr,
		func(addr string) {
			if addr == "" {
				return
			}
			cfg := dirclient.Config{
				ServerAddress: addr,
				AuthMode:      app.state.authMode,
				TLSSkipVerify: app.cfg.Directory.TLSSkipVerify,
				TLSCAFile:     app.cfg.Directory.TLSCAFile,
				TLSCertFile:   app.cfg.Directory.TLSCertFile,
				TLSKeyFile:    app.cfg.Directory.TLSKeyFile,
				AuthToken:     app.cfg.Directory.AuthToken,
			}
			go app.connect(cfg)
		},
		nil, nil,
	)
	return nil
}

// openOASFDialog prompts the user for a new OASF schema server URL. On confirm
// a fresh oasf.Client is constructed and any cached class info is dropped.
func (app *Gui) openOASFDialog(g *gocui.Gui, v *gocui.View) error {
	app.openInput("Connect to OASF server (enter URL)", app.state.oasfAddr,
		func(addr string) {
			if addr == "" {
				return
			}
			client, err := oasf.NewClient(oasf.Config{ServerAddress: addr})
			if err != nil {
				app.g.Update(func(g *gocui.Gui) error {
					app.renderPreviewText(g, "OASF configuration failed", err.Error())
					return nil
				})
				return
			}
			app.g.Update(func(g *gocui.Gui) error {
				app.state.oasfClient = client
				app.state.oasfAddr = addr
				app.renderDirectory(g)
				return nil
			})
		},
		nil, nil,
	)
	return nil
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func (app *Gui) refresh(g *gocui.Gui, v *gocui.View) error {
	if app.state.client == nil {
		return nil
	}
	app.startRecordsStream()
	return nil
}

// ── Async actions ─────────────────────────────────────────────────────────────

func (app *Gui) pullRecord(cid string) {
	ctx := context.Background()
	jsonStr, err := app.state.client.PullJSON(ctx, cid)
	app.g.Update(func(g *gocui.Gui) error {
		if err != nil {
			app.renderPreviewText(g, "Error", err.Error())
			return nil
		}
		app.renderPreviewJSON(g, cid, jsonStr)
		return nil
	})
}

// autoPreviewRecord fires a background pull for the record currently under the
// cursor, resetting the preview scroll position first.
func (app *Gui) autoPreviewRecord(g *gocui.Gui) {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return
	}
	cid := records[app.state.recordCursor].CID
	if cid == "" {
		return
	}
	if pv, err := g.View(viewPreview); err == nil {
		_ = pv.SetOrigin(0, 0)
	}
	go app.pullRecord(cid)
}

// filterToggleInfo opens/closes the info popup for the currently highlighted
// skill/domain/module option in the filter tree.
func (app *Gui) filterToggleInfo(g *gocui.Gui, v *gocui.View) error {
	rows := app.filteredListRows()
	fs := &app.state.filters
	if fs.listCursor >= len(rows) {
		return nil
	}
	row := rows[fs.listCursor]
	if row.option == "" {
		return nil
	}

	var ct oasf.ClassType
	switch row.category {
	case filterSkills:
		ct = oasf.ClassTypeSkill
	case filterDomains:
		ct = oasf.ClassTypeDomain
	case filterModules:
		ct = oasf.ClassTypeModule
	default:
		return nil
	}

	name := row.option
	if fs.inlineDesc == name {
		return app.closeInfoPopup(g, v)
	}

	fs.inlineDesc = name
	fs.inlineDescText = ""
	fs.inlineDescLoading = true
	app.openInfoPopup(g, viewFilters)

	go app.fetchInlineDesc(ct, name)
	return nil
}

// fetchInlineDesc fetches the OASF description for name and stores it in
// the filter info state, triggering a popup re-render on completion.
func (app *Gui) fetchInlineDesc(ct oasf.ClassType, name string) {
	client := app.state.oasfClient
	if client == nil {
		app.g.Update(func(g *gocui.Gui) error {
			if app.state.filters.inlineDesc != name {
				return nil
			}
			app.state.filters.inlineDescLoading = false
			app.state.filters.inlineDescText = "OASF not configured"
			app.renderInfoPopup(g)
			return nil
		})
		return
	}

	schemaVer := app.state.classEntriesVer
	info, err := client.Fetch(context.Background(), ct, name, schemaVer)
	app.g.Update(func(g *gocui.Gui) error {
		if app.state.filters.inlineDesc != name {
			return nil
		}
		app.state.filters.inlineDescLoading = false
		if err != nil {
			app.state.filters.inlineDescText = err.Error()
		} else {
			ipv, _ := g.View(viewInfoPopup)
			descW := 40
			if ipv != nil {
				w, _ := ipv.Size()
				descW = w - 2
				if descW < 20 {
					descW = 20
				}
			}
			app.state.filters.inlineDescText = formatClassInfo(info, descW, app.theme)
		}
		app.renderInfoPopup(g)
		return nil
	})
}

// formatClassInfo produces a pre-formatted, ANSI-colored text block showing
// the class hierarchy tree and description, similar to record info display.
func formatClassInfo(info *oasf.ClassInfo, descW int, t Theme) string {
	var sb strings.Builder

	// Taxonomy header + hierarchy tree
	fmt.Fprintf(&sb, "%sTaxonomy:%s\n", t.Color5, t.Reset)
	ancestors := info.Ancestors
	for depth, a := range ancestors {
		prefix := indent1 + strings.Repeat("    ", depth)
		connector := "└── "
		if depth == 0 {
			connector = ""
		}
		fmt.Fprintf(&sb, "%s%s%s%s%s %s(%d)%s\n",
			prefix, t.Color2, connector, t.Color1, a.Caption, t.Color10, a.ID, t.Reset)
	}

	selfDepth := len(ancestors)
	selfPrefix := indent1 + strings.Repeat("    ", selfDepth)
	selfConnector := "└── "
	if selfDepth == 0 {
		selfConnector = ""
	}
	caption := info.Caption
	if caption == "" {
		caption = info.Name
	}
	fmt.Fprintf(&sb, "%s%s%s%s%s %s(%d)%s",
		selfPrefix, t.Color2, selfConnector, t.Color1, caption, t.Color10, info.ID, t.Reset)

	// Description
	if info.Description != "" {
		fmt.Fprintf(&sb, "\n%sDescription:%s\n", t.Color5, t.Reset)
		for _, dl := range wrapText(info.Description, descW) {
			fmt.Fprintf(&sb, "%s%s\n", indent1, dl)
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// clearInlineDesc resets the inline description toggle state.
func (app *Gui) clearInlineDesc() {
	app.state.filters.inlineDesc = ""
	app.state.filters.inlineDescText = ""
	app.state.filters.inlineDescLoading = false
}

// ── Info popup ────────────────────────────────────────────────────────────────

// hideInfoPopupIfVisible dismisses the info popup without changing focus,
// useful when focus is being moved by another action (e.g. mouse click).
func (app *Gui) hideInfoPopupIfVisible(g *gocui.Gui) {
	ipv, err := g.View(viewInfoPopup)
	if err != nil || !ipv.Visible {
		return
	}
	ipv.Visible = false
	app.clearRecordInlineDesc()
	app.clearInlineDesc()
	app.state.infoPrevView = ""
	app.state.infoPopupPanel = ""
}

// openInfoPopup shows the info popup anchored under the selection in sourcePanel.
func (app *Gui) openInfoPopup(g *gocui.Gui, sourcePanel string) {
	ipv, err := g.View(viewInfoPopup)
	if err != nil {
		return
	}

	if cv := g.CurrentView(); cv != nil && cv.Name() != viewInfoPopup {
		app.state.infoPrevView = cv.Name()
	}
	app.state.infoPopupPanel = sourcePanel

	ipv.Clear()
	_ = ipv.SetOrigin(0, 0)
	ipv.Visible = true
	app.renderInfoPopup(g)
	_, _ = g.SetCurrentView(viewInfoPopup)
	_, _ = g.SetViewOnTop(viewInfoPopup)
}

// closeInfoPopup hides the info popup and returns focus.
func (app *Gui) closeInfoPopup(g *gocui.Gui, v *gocui.View) error {
	ipv, err := g.View(viewInfoPopup)
	if err != nil {
		return nil
	}
	ipv.Visible = false

	app.clearRecordInlineDesc()
	app.clearInlineDesc()

	target := app.state.infoPrevView
	if target == "" {
		target = viewRecords
	}
	app.state.infoPrevView = ""
	app.state.infoPopupPanel = ""
	return app.focusTo(g, target)
}

// renderInfoPopup updates the content of the info popup based on whichever
// panel triggered it.
func (app *Gui) renderInfoPopup(g *gocui.Gui) {
	ipv, err := g.View(viewInfoPopup)
	if err != nil || !ipv.Visible {
		return
	}
	ipv.Clear()
	_ = ipv.SetOrigin(0, 0)

	switch app.state.infoPopupPanel {
	case viewFilters:
		if app.state.filters.inlineDescLoading {
			fmt.Fprintf(ipv, "%sloading…%s", app.theme.Color4, app.theme.Reset)
		} else if app.state.filters.inlineDescText != "" {
			fmt.Fprint(ipv, app.state.filters.inlineDescText)
		}
	case viewRecords:
		if app.state.recordInfoLoading {
			fmt.Fprintf(ipv, "%sloading…%s", app.theme.Color4, app.theme.Reset)
		} else if app.state.recordInfoText != "" {
			fmt.Fprint(ipv, app.state.recordInfoText)
		}
	}
}

// ── Help popup ────────────────────────────────────────────────────────────────

func (app *Gui) openHelp(g *gocui.Gui, v *gocui.View) error {
	hv, err := g.View(viewHelp)
	if err != nil {
		return nil
	}

	// Remember where we came from.
	if cv := g.CurrentView(); cv != nil && cv.Name() != viewHelp {
		app.state.helpPrevView = cv.Name()
	}

	// Populate content.
	focused := app.state.helpPrevView
	hv.Clear()
	_ = hv.SetOrigin(0, 0)
	for _, line := range helpPopupLines(focused) {
		fmt.Fprintln(hv, line)
	}

	hv.Visible = true
	_, _ = g.SetCurrentView(viewHelp)
	_, _ = g.SetViewOnTop(viewHelp)
	return nil
}

func (app *Gui) closeHelp(g *gocui.Gui, v *gocui.View) error {
	hv, err := g.View(viewHelp)
	if err != nil {
		return nil
	}
	hv.Visible = false

	target := app.state.helpPrevView
	if target == "" {
		target = viewRecords
	}
	return app.focusTo(g, target)
}

// ── Copy menu popup ───────────────────────────────────────────────────────────

func (app *Gui) openCopyMenu(g *gocui.Gui, v *gocui.View) error {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return nil
	}

	cv, err := g.View(viewCopyMenu)
	if err != nil {
		return nil
	}

	if cur := g.CurrentView(); cur != nil && cur.Name() != viewCopyMenu {
		app.state.copyMenuPrevView = cur.Name()
	}

	cv.Clear()
	fmt.Fprintf(cv, "  %sc%s  copy CID\n", app.theme.Color2, app.theme.Reset)
	fmt.Fprintf(cv, "  %sa%s  copy record JSON", app.theme.Color2, app.theme.Reset)

	cv.Visible = true
	_, _ = g.SetCurrentView(viewCopyMenu)
	_, _ = g.SetViewOnTop(viewCopyMenu)
	return nil
}

func (app *Gui) closeCopyMenu(g *gocui.Gui, v *gocui.View) error {
	cv, err := g.View(viewCopyMenu)
	if err != nil {
		return nil
	}
	cv.Visible = false

	target := app.state.copyMenuPrevView
	if target == "" {
		target = viewRecords
	}
	return app.focusTo(g, target)
}

func (app *Gui) copyCID(g *gocui.Gui, v *gocui.View) error {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return app.closeCopyMenu(g, v)
	}
	cid := records[app.state.recordCursor].CID
	if cid == "" {
		return app.closeCopyMenu(g, v)
	}
	_ = writeClipboard(cid)
	return app.closeCopyMenu(g, v)
}

func (app *Gui) copyRecordJSON(g *gocui.Gui, v *gocui.View) error {
	records := app.state.filteredRecords
	if app.state.recordCursor >= len(records) {
		return app.closeCopyMenu(g, v)
	}
	cid := records[app.state.recordCursor].CID
	if cid == "" {
		return app.closeCopyMenu(g, v)
	}
	if err := app.closeCopyMenu(g, v); err != nil {
		return err
	}
	go app.fetchAndCopyJSON(cid)
	return nil
}

func (app *Gui) fetchAndCopyJSON(cid string) {
	ctx := context.Background()
	jsonStr, err := app.state.client.PullJSON(ctx, cid)
	if err != nil {
		app.g.Update(func(g *gocui.Gui) error {
			app.renderPreviewText(g, "Error", "Failed to fetch record: "+err.Error())
			return nil
		})
		return
	}
	_ = writeClipboard(jsonStr)
}

// writeClipboard writes text to the system clipboard.
func writeClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
