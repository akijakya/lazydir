package gui

import (
	"context"
	"fmt"
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
	if err := g.SetKeybinding(viewFilters, gocui.KeyTab, gocui.ModNone, app.filterTab); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFilters, gocui.KeyEsc, gocui.ModNone, app.filterEsc); err != nil {
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
	if err := app.focusTo(g, viewFilters); err != nil {
		return err
	}
	_, cy := v.Cursor()
	_, oy := v.Origin()
	idx := oy + cy

	switch app.state.filters.mode {
	case filterModeList:
		rows := app.listRows()
		if idx < 0 || idx >= len(rows) {
			return nil
		}
		app.state.filters.listCursor = idx
		app.renderFiltersView(g)
		row := rows[idx]
		app.state.filters.editing = row.category
		app.state.filters.optionsCursor = 0
		app.clearInlineDesc()
		app.state.filters.mode = filterModeOptions
		app.renderFiltersView(g)
	case filterModeOptions:
		options := app.optionsFor(app.state.filters.editing)
		if idx < 0 || idx >= len(options) {
			return nil
		}
		app.state.filters.optionsCursor = idx
		app.toggleOptionUnderCursor(g)
	}
	return nil
}

func (app *Gui) filterCursorUp(g *gocui.Gui, v *gocui.View) error {
	switch app.state.filters.mode {
	case filterModeList:
		if app.state.filters.listCursor > 0 {
			app.state.filters.listCursor--
		}
	case filterModeOptions:
		if app.state.filters.optionsCursor > 0 {
			app.state.filters.optionsCursor--
		}
	}
	app.renderFiltersView(g)
	return nil
}

func (app *Gui) filterCursorDown(g *gocui.Gui, v *gocui.View) error {
	switch app.state.filters.mode {
	case filterModeList:
		rows := app.listRows()
		if app.state.filters.listCursor < len(rows)-1 {
			app.state.filters.listCursor++
		}
	case filterModeOptions:
		options := app.optionsFor(app.state.filters.editing)
		if app.state.filters.optionsCursor < len(options)-1 {
			app.state.filters.optionsCursor++
		}
	}
	app.renderFiltersView(g)
	return nil
}

// filterEnter dispatches by mode:
// - list mode: open the options view for the category under the cursor
// - options mode: toggle the selection under the cursor (same as tab)
func (app *Gui) filterEnter(g *gocui.Gui, v *gocui.View) error {
	switch app.state.filters.mode {
	case filterModeList:
		rows := app.listRows()
		if app.state.filters.listCursor >= len(rows) {
			return nil
		}
		row := rows[app.state.filters.listCursor]
		app.state.filters.editing = row.category
		app.state.filters.optionsCursor = 0
		app.clearInlineDesc()
		app.state.filters.mode = filterModeOptions
		app.renderFiltersView(g)
	case filterModeOptions:
		app.toggleOptionUnderCursor(g)
	}
	return nil
}

// filterTab toggles the option under the cursor when in options mode; in
// list mode it falls through to global focus cycling so the panel still
// behaves like every other left-column panel.
func (app *Gui) filterTab(g *gocui.Gui, v *gocui.View) error {
	if app.state.filters.mode == filterModeOptions {
		app.toggleOptionUnderCursor(g)
		return nil
	}
	return app.cycleFocusForward(g, v)
}

// filterEsc returns from options mode back to the filter list. In list mode
// it does nothing — there is no "clear all filters" gesture (intentional;
// users remove filters by toggling them off in the options view).
func (app *Gui) filterEsc(g *gocui.Gui, v *gocui.View) error {
	if app.state.filters.mode == filterModeOptions {
		app.clearInlineDesc()
		app.state.filters.mode = filterModeList
		app.state.filters.listCursor = app.listCursorForCategory(app.state.filters.editing)
		app.renderFiltersView(g)
	}
	return nil
}

// toggleOptionUnderCursor flips selection of the option highlighted in the
// options view, then re-issues the SearchRecords stream with the updated
// filter set so the [3] Records pane reflects it.
func (app *Gui) toggleOptionUnderCursor(g *gocui.Gui) {
	cat := app.state.filters.editing
	options := app.optionsFor(cat)
	if app.state.filters.optionsCursor >= len(options) {
		return
	}
	app.toggleApplied(cat, options[app.state.filters.optionsCursor])
	app.startRecordsStream()
	app.renderFiltersView(g)
}

// listCursorForCategory returns the row index of the supplied category in
// the expanded list view.
func (app *Gui) listCursorForCategory(c filterCategory) int {
	rows := app.listRows()
	for i, r := range rows {
		if r.category == c && r.option == "" {
			return i
		}
	}
	return 0
}

// ── Records panel handlers ────────────────────────────────────────────────────

func (app *Gui) recordMouseClick(g *gocui.Gui, v *gocui.View) error {
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
		nil,
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

// recordToggleInfo toggles the inline description for the currently highlighted
// record, fetching it via the directory's PullInfo RPC.
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
		app.clearRecordInlineDesc()
		app.renderRecordsView(g)
		return nil
	}

	app.state.recordInfoCID = cid
	app.state.recordInfoText = ""
	app.state.recordInfoLoading = true
	app.renderRecordsView(g)

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
			app.state.recordInfoText = formatRecordInfo(info)
		}
		app.renderRecordsView(g)
		return nil
	})
}

// formatRecordInfo renders a RecordInfo as colored, human-readable lines.
func formatRecordInfo(info *dirclient.RecordInfo) string {
	const (
		cyan    = "\033[36m"
		yellow  = "\033[33m"
		green   = "\033[32m"
		magenta = "\033[35m"
		reset   = "\033[0m"
	)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%sCID:%s %s", cyan, reset, info.CID))

	if len(info.Annotations) > 0 {
		sb.WriteString(fmt.Sprintf("\n%sAnnotations:%s", yellow, reset))
		keys := make([]string, 0, len(info.Annotations))
		for k := range info.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("\n    %s%s:%s %s", yellow, k, reset, info.Annotations[k]))
		}
	}

	if info.SchemaVersion != "" {
		sb.WriteString(fmt.Sprintf("\n%sSchema version:%s %s", green, reset, info.SchemaVersion))
	}
	if info.CreatedAt != "" {
		sb.WriteString(fmt.Sprintf("\n%sCreated at:%s %s", magenta, reset, info.CreatedAt))
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
		nil,
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
		nil,
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

func (app *Gui) fetchOASF(ct oasf.ClassType, name string) {
	client := app.state.oasfClient
	if client == nil {
		app.g.Update(func(g *gocui.Gui) error {
			app.renderPreviewText(g, "OASF not configured",
				"No OASF server is configured. Press 'o' on the Connections panel to set one.")
			return nil
		})
		return
	}

	info, err := client.Fetch(context.Background(), ct, name)
	app.g.Update(func(g *gocui.Gui) error {
		if err != nil {
			app.renderPreviewText(g, "Error", err.Error())
			return nil
		}
		title := fmt.Sprintf("%s %s", info.Type, info.Name)
		if info.Caption != "" {
			title = fmt.Sprintf("%s %s (%s)", info.Type, info.Name, info.Caption)
		}
		app.renderPreviewText(g, title, info.Description)
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

// filterToggleInfo toggles the inline OASF description for the currently
// highlighted skill/domain/module in the filter options view.
func (app *Gui) filterToggleInfo(g *gocui.Gui, v *gocui.View) error {
	if app.state.filters.mode != filterModeOptions {
		return nil
	}
	cat := app.state.filters.editing
	var ct oasf.ClassType
	switch cat {
	case filterSkills:
		ct = oasf.ClassTypeSkill
	case filterDomains:
		ct = oasf.ClassTypeDomain
	case filterModules:
		ct = oasf.ClassTypeModule
	default:
		return nil
	}

	options := app.optionsFor(cat)
	fs := &app.state.filters
	if fs.optionsCursor >= len(options) {
		return nil
	}
	name := options[fs.optionsCursor]

	if fs.inlineDesc == name {
		app.clearInlineDesc()
		app.renderFiltersView(g)
		return nil
	}

	fs.inlineDesc = name
	fs.inlineDescText = ""
	fs.inlineDescLoading = true
	app.renderFiltersView(g)

	go app.fetchInlineDesc(ct, name)
	return nil
}

// fetchInlineDesc fetches the OASF description for name and stores it in
// the inline description state, triggering a re-render on completion.
func (app *Gui) fetchInlineDesc(ct oasf.ClassType, name string) {
	client := app.state.oasfClient
	if client == nil {
		app.g.Update(func(g *gocui.Gui) error {
			if app.state.filters.inlineDesc != name {
				return nil
			}
			app.state.filters.inlineDescLoading = false
			app.state.filters.inlineDescText = "OASF not configured"
			app.renderFiltersView(g)
			return nil
		})
		return
	}

	info, err := client.Fetch(context.Background(), ct, name)
	app.g.Update(func(g *gocui.Gui) error {
		if app.state.filters.inlineDesc != name {
			return nil
		}
		app.state.filters.inlineDescLoading = false
		if err != nil {
			app.state.filters.inlineDescText = err.Error()
		} else {
			app.state.filters.inlineDescText = info.Description
		}
		app.renderFiltersView(g)
		return nil
	})
}

// clearInlineDesc resets the inline description toggle state.
func (app *Gui) clearInlineDesc() {
	app.state.filters.inlineDesc = ""
	app.state.filters.inlineDescText = ""
	app.state.filters.inlineDescLoading = false
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
