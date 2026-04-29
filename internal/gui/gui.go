package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/oasf"
	"github.com/jesseduffield/gocui"
)

// streamState describes the lifecycle phase of the records stream.
type streamState int

const (
	streamIdle      streamState = iota // no client yet, or no stream issued
	streamLoading                      // first page hasn't arrived yet
	streamStreaming                    // first page rendered, still receiving the rest
	streamDone                         // stream finished cleanly
	streamErrored                      // stream finished with an error
)

// appState holds all mutable application state. Fields are only mutated on
// the GUI goroutine (inside g.Update callbacks or key handlers).
type appState struct {
	// Directory connection
	serverAddr string
	authMode   string
	connected  bool
	client     *dirclient.Client

	// OASF connection
	oasfAddr   string
	oasfClient *oasf.Client

	// records: server already filtered them; we render this slice directly
	// (after the optional name query in filterQuery is applied).
	records         []*dirclient.RecordSummary
	filteredRecords []*dirclient.RecordSummary
	recordCursor    int

	// records-stream lifecycle
	stream     streamState
	streamErr  string
	cancelLoad context.CancelFunc

	// inline record info toggle (records panel)
	recordInfoCID     string // CID of the record whose info is expanded, "" if none
	recordInfoText    string // cached description text
	recordInfoLoading bool   // fetch in progress

	// distinct values usable as filter options for each category, growing
	// monotonically across every stream (we never forget an option once
	// we've seen it, even if the next filtered stream wouldn't include it).
	filterValues *filterValueAggregator

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
	onInputChange  func(string) // called live (debounced) as the user types; nil disables
	inputDebounce  *time.Timer  // debounce timer for live onChange

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
			serverAddr:   cfg.Directory.ServerAddress,
			authMode:     cfg.Directory.AuthMode,
			oasfAddr:     cfg.OASF.ServerAddress,
			oasfClient:   oasfClient,
			filters:      newFilterState(),
			filterValues: newFilterValueAggregator(),
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
	g.SelFrameColor = gocui.ColorGreen
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

	// Install the client and kick off the initial unfiltered stream in a
	// single Update callback. Splitting them into two would be racy: the
	// scheduler may run the second before the first, and startRecordsStream
	// would then see a nil client and silently no-op.
	app.g.Update(func(g *gocui.Gui) error {
		if app.state.client != nil {
			app.state.client.Close()
		}
		app.state.client = c
		app.state.serverAddr = cfg.ServerAddress
		app.state.authMode = cfg.AuthMode
		app.state.connected = true
		app.state.filters = newFilterState()
		app.state.filterQuery = ""
		app.renderDirectory(g)
		app.renderStatus(g)
		app.startRecordsStream()
		return nil
	})
}

// startRecordsStream cancels any in-flight records stream and issues a fresh
// SearchRecords RPC for the current filter selection. It must run on the GUI
// goroutine (i.e. inside a g.Update callback or a key handler) because it
// touches state without taking state.mu.
func (app *Gui) startRecordsStream() {
	if app.state.client == nil {
		return
	}

	if app.state.cancelLoad != nil {
		app.state.cancelLoad()
		app.state.cancelLoad = nil
	}

	app.state.records = nil
	app.state.recordCursor = 0
	app.state.streamErr = ""
	app.state.stream = streamLoading
	app.state.recordInfoCID = ""
	app.state.recordInfoText = ""
	app.state.recordInfoLoading = false
	app.applyNameFilter()
	app.renderRecordsView(app.g)
	app.renderDirectory(app.g)

	ctx, cancel := context.WithCancel(context.Background())
	app.state.cancelLoad = cancel

	queries := app.activeQueries()
	client := app.state.client

	go client.Stream(ctx, queries, dirclient.StreamCallbacks{
		OnFirstPage: func(summaries []*dirclient.RecordSummary) {
			app.g.Update(func(g *gocui.Gui) error {
				if ctx.Err() != nil {
					return nil
				}
				app.state.records = append(app.state.records, summaries...)
				for _, r := range summaries {
					app.state.filterValues.add(r)
				}
				// Stay in streamLoading until OnDone confirms the stream
				// is fully exhausted — avoids a "streaming…" flash when
				// all records fit in the first page.
				app.state.stream = streamStreaming
				app.applyNameFilter()
				app.renderRecordsView(g)
				app.renderFiltersView(g)
				app.autoPreviewRecord(g)
				return nil
			})
		},
		OnBatch: func(batch []*dirclient.RecordSummary) {
			app.g.Update(func(g *gocui.Gui) error {
				if ctx.Err() != nil {
					return nil
				}
				app.state.records = append(app.state.records, batch...)
				for _, r := range batch {
					app.state.filterValues.add(r)
				}
				app.applyNameFilter()
				app.renderRecordsView(g)
				app.renderFiltersView(g)
				return nil
			})
		},
		OnDone: func(err error) {
			app.g.Update(func(g *gocui.Gui) error {
				if ctx.Err() != nil {
					return nil
				}
				if err != nil {
					app.state.stream = streamErrored
					app.state.streamErr = err.Error()
					app.renderPreviewText(g, "Stream error", err.Error())
				} else {
					app.state.stream = streamDone
				}
				app.renderRecordsView(g)
				app.renderDirectory(g)
				return nil
			})
		},
	})
}

// applyNameFilter recomputes filteredRecords from records by applying only
// the local name query. Server-side filters have already been applied to
// records by the time they reach us; the name query is intentionally local
// so the user can narrow incrementally without restarting the stream.
//
// Must be called from the GUI goroutine (g.Update callback or key handler).
func (app *Gui) applyNameFilter() {
	if app.state.filterQuery == "" {
		app.state.filteredRecords = app.state.records
		return
	}
	q := strings.ToLower(app.state.filterQuery)
	out := make([]*dirclient.RecordSummary, 0, len(app.state.records))
	for _, r := range app.state.records {
		if strings.Contains(strings.ToLower(r.Name), q) {
			out = append(out, r)
		}
	}
	app.state.filteredRecords = out
}

// openInput shows the shared input prompt, pre-fills it with initialValue,
// focuses it, and wires confirm/cancel/change callbacks. When onChange is
// non-nil the filter is applied live (debounced) as the user types.
func (app *Gui) openInput(title, initialValue string, onConfirm func(string), onCancel func(), onChange func(string)) {
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
	app.state.onInputChange = onChange

	if onChange != nil {
		iv.Editor = &liveInputEditor{gui: app}
	} else {
		iv.Editor = nil
	}

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
	if app.state.inputDebounce != nil {
		app.state.inputDebounce.Stop()
		app.state.inputDebounce = nil
	}
	app.state.onInputChange = nil

	iv, err := app.g.View(viewInput)
	if err != nil {
		return
	}
	iv.Visible = false
	iv.Editor = nil
	app.state.inputVisible = false

	target := app.state.prevView
	if target == "" {
		target = viewRecords
	}
	app.state.prevView = ""
	_ = app.focusTo(app.g, target)
}

// liveInputEditor wraps the default text-area editor and schedules a
// debounced onChange callback whenever the content changes.
type liveInputEditor struct {
	gui *Gui
}

func (e *liveInputEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	before := v.TextArea.GetContent()
	ret := gocui.DefaultEditor.Edit(v, key, ch, mod)
	if v.TextArea.GetContent() != before {
		e.gui.scheduleInputChange()
	}
	return ret
}

const inputDebounceDelay = 150 * time.Millisecond

// scheduleInputChange resets the debounce timer so the onChange callback
// fires inputDebounceDelay after the last keystroke.
func (app *Gui) scheduleInputChange() {
	if app.state.inputDebounce != nil {
		app.state.inputDebounce.Stop()
	}
	app.state.inputDebounce = time.AfterFunc(inputDebounceDelay, func() {
		app.g.Update(func(g *gocui.Gui) error {
			if app.state.onInputChange == nil {
				return nil
			}
			iv, err := g.View(viewInput)
			if err != nil {
				return nil
			}
			app.state.onInputChange(strings.TrimSpace(iv.TextArea.GetContent()))
			return nil
		})
	})
}
