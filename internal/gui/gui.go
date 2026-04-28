package gui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/oasf"
	"github.com/jesseduffield/gocui"
)

// appState holds all mutable application state, protected by a mutex so that
// goroutines spawned for async work can safely call g.Update().
type appState struct {
	mu sync.Mutex

	// Directory connection
	serverAddr string
	authMode   string
	connected  bool
	loading    bool
	client     *dirclient.Client

	// OASF connection
	oasfAddr   string
	oasfClient *oasf.Client

	// full record list
	allRecords []*dirclient.RecordSummary
	// records after applied filters + name query
	filteredRecords []*dirclient.RecordSummary
	recordCursor    int

	// distinct values usable as filter options for each category
	filterValues dirclient.FilterValues

	// [2] Filters panel state
	filters filterState

	// name filter (active query; empty means no filter)
	filterQuery string

	// input prompt state
	inputVisible   bool
	inputTitle     string
	prevView       string       // view to return focus to on dismiss
	onInputConfirm func(string) // called with TextArea content on enter
	onInputCancel  func()       // called on esc

	// help popup state
	helpPrevView string // view to return to when help closes
}

// Config bundles everything needed to start the GUI.
type Config struct {
	Directory dirclient.Config
	OASF      oasf.Config
}

// Gui is the top-level lazydir GUI object.
type Gui struct {
	g     *gocui.Gui
	state appState
	cfg   Config
}

// New creates and starts the lazydir GUI.
func New(cfg Config) error {
	oasfClient, err := oasf.NewClient(cfg.OASF)
	if err != nil {
		return fmt.Errorf("configuring OASF client: %w", err)
	}

	app := &Gui{
		cfg: cfg,
		state: appState{
			serverAddr: cfg.Directory.ServerAddress,
			authMode:   cfg.Directory.AuthMode,
			oasfAddr:   cfg.OASF.ServerAddress,
			oasfClient: oasfClient,
			filters:    newFilterState(),
		},
	}

	g, err := gocui.NewGui(gocui.NewGuiOpts{
		OutputMode:      gocui.OutputTrue,
		SupportOverlaps: false,
	})
	if err != nil {
		return fmt.Errorf("creating gui: %w", err)
	}
	defer g.Close()

	app.g = g
	g.Highlight = true
	g.SelFgColor = gocui.ColorGreen
	g.Mouse = true

	g.SetManagerFunc(app.layout)

	if err := app.bindKeys(g); err != nil {
		return fmt.Errorf("binding keys: %w", err)
	}

	// Kick off the initial directory connection in the background.
	go app.connect(cfg.Directory)

	if err := g.MainLoop(); err != nil && !gocui.IsQuit(err) {
		return fmt.Errorf("main loop: %w", err)
	}

	return nil
}

// connect dials the directory server and loads records.
func (app *Gui) connect(cfg dirclient.Config) {
	ctx := context.Background()
	c, err := dirclient.Connect(ctx, cfg)
	if err != nil {
		app.g.Update(func(g *gocui.Gui) error {
			app.state.connected = false
			app.renderDirectory(g)
			app.renderStatus(g)
			app.renderPreviewText(g, "Connection failed", err.Error())
			return nil
		})
		return
	}

	app.g.Update(func(g *gocui.Gui) error {
		if app.state.client != nil {
			app.state.client.Close()
		}
		app.state.client = c
		app.state.serverAddr = cfg.ServerAddress
		app.state.authMode = cfg.AuthMode
		app.state.connected = true
		app.renderDirectory(g)
		app.renderStatus(g)
		return nil
	})

	app.loadRecords(c)
}

// loadRecords fetches all records from the server and populates the panels.
func (app *Gui) loadRecords(c *dirclient.Client) {
	ctx := context.Background()
	summaries, err := c.ListAll(ctx)
	if err != nil {
		app.g.Update(func(g *gocui.Gui) error {
			app.renderPreviewText(g, "Load failed", err.Error())
			return nil
		})
		return
	}

	app.g.Update(func(g *gocui.Gui) error {
		app.state.allRecords = summaries
		app.state.filterValues = dirclient.ExtractFilterValues(summaries)
		app.state.filters = newFilterState()
		app.state.filterQuery = ""
		app.state.recordCursor = 0
		app.applyFilters()
		app.renderFiltersView(g)
		app.renderRecordsView(g)
		app.autoPreviewRecord(g)
		return nil
	})
}

// applyFilters rebuilds filteredRecords by intersecting the active [2] Filters
// selections with the name query.
// Must be called while holding state.mu OR from a g.Update callback (single-threaded).
func (app *Gui) applyFilters() {
	var base []*dirclient.RecordSummary
	if len(app.state.filters.applied) == 0 {
		base = app.state.allRecords
	} else {
		for _, r := range app.state.allRecords {
			if app.recordMatchesFilters(r) {
				base = append(base, r)
			}
		}
	}

	if app.state.filterQuery == "" {
		app.state.filteredRecords = base
		return
	}

	q := strings.ToLower(app.state.filterQuery)
	var out []*dirclient.RecordSummary
	for _, r := range base {
		if strings.Contains(strings.ToLower(r.Name), q) {
			out = append(out, r)
		}
	}
	app.state.filteredRecords = out
}

// openInput shows the shared input prompt, pre-fills it with initialValue,
// focuses it, and wires confirm/cancel callbacks.
func (app *Gui) openInput(title, initialValue string, onConfirm func(string), onCancel func()) {
	iv, err := app.g.View(viewInput)
	if err != nil {
		return
	}

	// Save the currently focused view so we can restore it on dismiss.
	if cv := app.g.CurrentView(); cv != nil {
		app.state.prevView = cv.Name()
	}

	app.state.inputVisible = true
	app.state.inputTitle = title
	app.state.onInputConfirm = onConfirm
	app.state.onInputCancel = onCancel

	iv.Title = title
	iv.Visible = true
	iv.Clear()
	iv.TextArea.Clear()
	iv.TextArea.TypeString(initialValue)
	iv.RenderTextArea()

	_, _ = app.g.SetCurrentView(viewInput)
}

// closeInput hides the input prompt and restores focus to the previous view.
func (app *Gui) closeInput() {
	iv, err := app.g.View(viewInput)
	if err != nil {
		return
	}
	iv.Visible = false
	app.state.inputVisible = false

	target := app.state.prevView
	if target == "" {
		target = viewRecords
	}
	app.state.prevView = ""
	_ = app.focusTo(app.g, target)
}

