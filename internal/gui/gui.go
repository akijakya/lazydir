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

// classTab represents which taxonomy tab is active in the classes panel.
type classTab int

const (
	tabSkills  classTab = 0
	tabDomains classTab = 1
	tabModules classTab = 2
)

func (t classTab) String() string {
	switch t {
	case tabSkills:
		return "Skills"
	case tabDomains:
		return "Domains"
	case tabModules:
		return "Modules"
	}
	return ""
}

// appState holds all mutable application state, protected by a mutex so that
// goroutines spawned for async work can safely call g.Update().
type appState struct {
	mu sync.Mutex

	serverAddr string
	authMode   string
	connected  bool
	loading    bool
	statusMsg  string

	client *dirclient.Client

	// full record list
	allRecords []*dirclient.RecordSummary
	// records after class + name filter
	filteredRecords []*dirclient.RecordSummary
	recordCursor    int

	// taxonomy
	skills  []string
	domains []string
	modules []string

	activeTab   classTab
	classCursor int
	// name of the currently selected class (empty = all)
	selectedClass     string
	selectedClassType oasf.ClassType

	// name filter (active query; empty means no filter)
	filterQuery string

	// input prompt state
	inputVisible    bool
	inputTitle      string
	prevView        string            // view to return focus to on dismiss
	onInputConfirm  func(string)      // called with TextArea content on enter
	onInputCancel   func()            // called on esc
}

// Gui is the top-level lazydir GUI object.
type Gui struct {
	g     *gocui.Gui
	state appState
	cfg   dirclient.Config
}

// New creates and starts the lazydir GUI.
func New(cfg dirclient.Config) error {
	app := &Gui{
		cfg: cfg,
		state: appState{
			serverAddr: cfg.ServerAddress,
			authMode:   cfg.AuthMode,
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
	g.Mouse = false

	g.SetManagerFunc(app.layout)

	if err := app.bindKeys(g); err != nil {
		return fmt.Errorf("binding keys: %w", err)
	}

	// Kick off the initial connection in the background.
	go app.connect(cfg)

	if err := g.MainLoop(); err != nil && !gocui.IsQuit(err) {
		return fmt.Errorf("main loop: %w", err)
	}

	return nil
}

// connect dials the directory server and loads records.
func (app *Gui) connect(cfg dirclient.Config) {
	app.g.Update(func(g *gocui.Gui) error {
		app.setStatus("Connecting to " + cfg.ServerAddress + "…")
		return nil
	})

	ctx := context.Background()
	c, err := dirclient.Connect(ctx, cfg)
	if err != nil {
		app.g.Update(func(g *gocui.Gui) error {
			app.state.connected = false
			app.setStatus("Connection failed: " + err.Error())
			app.renderDirectory(g)
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
		app.setStatus("Connected. Loading records…")
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
			app.setStatus("Failed to load records: " + err.Error())
			return nil
		})
		return
	}

	app.g.Update(func(g *gocui.Gui) error {
		app.state.allRecords = summaries
		skills, domains, modules := dirclient.ExtractClasses(summaries)
		app.state.skills = skills
		app.state.domains = domains
		app.state.modules = modules
		app.state.selectedClass = ""
		app.state.filterQuery = ""
		app.state.recordCursor = 0
		app.state.classCursor = 0
		app.applyFilters()
		app.renderClassesView(g)
		app.renderRecordsView(g)
		app.setStatus(fmt.Sprintf("Loaded %d records.", len(summaries)))
		return nil
	})
}

// applyFilters rebuilds filteredRecords based on selected class and name query.
// Must be called while holding state.mu OR from a g.Update callback (single-threaded).
func (app *Gui) applyFilters() {
	var base []*dirclient.RecordSummary
	if app.state.selectedClass == "" {
		base = app.state.allRecords
	} else {
		for _, r := range app.state.allRecords {
			if recordMatchesClass(r, app.state.selectedClassType, app.state.selectedClass) {
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

func recordMatchesClass(r *dirclient.RecordSummary, ct oasf.ClassType, name string) bool {
	switch ct {
	case oasf.ClassTypeSkill:
		for _, s := range r.Skills {
			if s == name {
				return true
			}
		}
	case oasf.ClassTypeDomain:
		for _, d := range r.Domains {
			if d == name {
				return true
			}
		}
	case oasf.ClassTypeModule:
		for _, m := range r.Modules {
			if m == name {
				return true
			}
		}
	}
	return false
}

func (app *Gui) setStatus(msg string) {
	app.state.statusMsg = msg
	v, err := app.g.View(viewStatus)
	if err != nil {
		return
	}
	v.Clear()
	fmt.Fprint(v, msg)
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

// currentClassItems returns the items for the active taxonomy tab.
func (app *Gui) currentClassItems() []string {
	switch app.state.activeTab {
	case tabSkills:
		return app.state.skills
	case tabDomains:
		return app.state.domains
	case tabModules:
		return app.state.modules
	}
	return nil
}
