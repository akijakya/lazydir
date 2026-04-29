package gui

import (
	"sort"

	"github.com/akijakya/lazydir/internal/dirclient"
)

// filterValueAggregator collects the unique values seen in the streamed
// records for each filter category. Values become available in the [2]
// Filters options view as soon as a record carrying them arrives.
type filterValueAggregator struct {
	skills        map[string]bool
	domains       map[string]bool
	modules       map[string]bool
	versions      map[string]bool
	schemaVersion map[string]bool
	authors       map[string]bool
}

func newFilterValueAggregator() *filterValueAggregator {
	return &filterValueAggregator{
		skills:        map[string]bool{},
		domains:       map[string]bool{},
		modules:       map[string]bool{},
		versions:      map[string]bool{},
		schemaVersion: map[string]bool{},
		authors:       map[string]bool{},
	}
}

// add folds one record's filterable fields into the aggregator.
func (a *filterValueAggregator) add(r *dirclient.RecordSummary) {
	for _, v := range r.Skills {
		a.skills[v] = true
	}
	for _, v := range r.Domains {
		a.domains[v] = true
	}
	for _, v := range r.Modules {
		a.modules[v] = true
	}
	for _, v := range r.Authors {
		if v != "" {
			a.authors[v] = true
		}
	}
	if r.SchemaVersion != "" {
		a.schemaVersion[r.SchemaVersion] = true
	}
	if r.Version != "" {
		a.versions[r.Version] = true
	}
}

// sorted returns the alphabetically sorted values of m.
func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// filterCategory identifies a filterable record field shown in the [2] Filters
// panel. The list is presented in this order.
type filterCategory int

const (
	filterSkills filterCategory = iota
	filterDomains
	filterModules
	filterOASFVersion
	filterVersion
	filterAuthor
	filterTrusted
	filterVerified
)

// allFilterCategories is the canonical ordered list of filter categories.
var allFilterCategories = []filterCategory{
	filterSkills,
	filterDomains,
	filterModules,
	filterOASFVersion,
	filterVersion,
	filterAuthor,
	filterTrusted,
	filterVerified,
}

// title returns the human-readable label used as the row text in the filter
// list view and as the title in the options view.
func (c filterCategory) title() string {
	switch c {
	case filterSkills:
		return "Skills"
	case filterDomains:
		return "Domains"
	case filterModules:
		return "Modules"
	case filterOASFVersion:
		return "OASF version"
	case filterVersion:
		return "Version"
	case filterAuthor:
		return "Author"
	case filterTrusted:
		return "Trusted"
	case filterVerified:
		return "Verified"
	}
	return ""
}

// boolean reports whether the category is a yes/no filter (only two options).
func (c filterCategory) boolean() bool {
	return c == filterTrusted || c == filterVerified
}

// filterMode is the current display mode for the [2] Filters panel.
type filterMode int

const (
	filterModeList    filterMode = iota // list of categories with applied selections under each
	filterModeOptions                   // options for a single selected category
)

// filterState owns all mutable state for the [2] Filters panel and the set of
// active filters that the records pane filters against. The map keys are
// option labels (e.g. skill name, version string, "yes"/"no").
type filterState struct {
	mode filterMode

	// list cursor: index over expanded rows (categories interleaved with
	// applied selections under them).
	listCursor int

	// options view: which category is being edited and the cursor within
	// the option list.
	editing       filterCategory
	optionsCursor int

	// applied[category] -> set of selected option labels.
	applied map[filterCategory]map[string]bool

	// inline description toggle (options mode only)
	inlineDesc        string // option name currently expanded, "" if none
	inlineDescText    string // cached description text
	inlineDescLoading bool   // fetch in progress
}

func newFilterState() filterState {
	return filterState{
		mode:    filterModeList,
		applied: map[filterCategory]map[string]bool{},
	}
}

// optionsFor returns the option labels available for a given category, given
// the records seen on the unfiltered stream so far. Booleans (Trusted /
// Verified) always offer the same fixed yes/no choices regardless of what
// has been streamed.
func (app *Gui) optionsFor(c filterCategory) []string {
	a := app.state.filterValues
	if a == nil {
		return nil
	}
	switch c {
	case filterSkills:
		return sortedSet(a.skills)
	case filterDomains:
		return sortedSet(a.domains)
	case filterModules:
		return sortedSet(a.modules)
	case filterOASFVersion:
		return sortedSet(a.schemaVersion)
	case filterVersion:
		return sortedSet(a.versions)
	case filterAuthor:
		return sortedSet(a.authors)
	case filterTrusted, filterVerified:
		return []string{"yes", "no"}
	}
	return nil
}

// appliedFor returns the sorted list of currently selected option labels for
// the given category — used to render the indented rows under each category.
func (app *Gui) appliedFor(c filterCategory) []string {
	set := app.state.filters.applied[c]
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// toggleApplied flips the on/off state of an option for a category.
func (app *Gui) toggleApplied(c filterCategory, value string) {
	set := app.state.filters.applied[c]
	if set == nil {
		set = map[string]bool{}
		app.state.filters.applied[c] = set
	}
	if set[value] {
		delete(set, value)
		if len(set) == 0 {
			delete(app.state.filters.applied, c)
		}
	} else {
		set[value] = true
	}
}

// listRow describes one rendered line in the filter list view. Either it is a
// category header (option == ""), or an indented selected option row.
type listRow struct {
	category filterCategory
	option   string // empty for category headers
}

// listRows builds the visible rows for filterModeList: for each category, the
// header followed by any applied selections on indented rows.
func (app *Gui) listRows() []listRow {
	var rows []listRow
	for _, c := range allFilterCategories {
		rows = append(rows, listRow{category: c})
		for _, opt := range app.appliedFor(c) {
			rows = append(rows, listRow{category: c, option: opt})
		}
	}
	return rows
}

// activeQueries flattens the applied filter selections into the slice of
// server-side queries that the directory understands. Yes/no booleans are
// only emitted when exactly one side is selected — picking both, or neither,
// means "no filter" and we omit the query entirely.
func (app *Gui) activeQueries() []dirclient.Query {
	var qs []dirclient.Query
	for _, c := range allFilterCategories {
		set := app.state.filters.applied[c]
		if len(set) == 0 {
			continue
		}
		if c.boolean() {
			yes := set["yes"]
			no := set["no"]
			if yes == no {
				continue
			}
			qs = append(qs, dirclient.Query{
				Category: categoryToFilter(c),
				Value:    boolValue(yes),
			})
			continue
		}
		for v := range set {
			qs = append(qs, dirclient.Query{
				Category: categoryToFilter(c),
				Value:    v,
			})
		}
	}
	return qs
}

func categoryToFilter(c filterCategory) dirclient.FilterCategory {
	switch c {
	case filterSkills:
		return dirclient.FilterSkill
	case filterDomains:
		return dirclient.FilterDomain
	case filterModules:
		return dirclient.FilterModule
	case filterOASFVersion:
		return dirclient.FilterSchemaVersion
	case filterVersion:
		return dirclient.FilterVersion
	case filterAuthor:
		return dirclient.FilterAuthor
	case filterTrusted:
		return dirclient.FilterTrusted
	case filterVerified:
		return dirclient.FilterVerified
	}
	return dirclient.FilterSkill
}

func boolValue(yes bool) string {
	if yes {
		return "true"
	}
	return "false"
}
