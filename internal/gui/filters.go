package gui

import (
	"sort"

	"github.com/akijakya/lazydir/internal/dirclient"
)

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
}

func newFilterState() filterState {
	return filterState{
		mode:    filterModeList,
		applied: map[filterCategory]map[string]bool{},
	}
}

// optionsFor returns the option labels available for a given category, given
// the current record set. The returned slice is alphabetically sorted (or in
// natural order for booleans).
func (app *Gui) optionsFor(c filterCategory) []string {
	switch c {
	case filterSkills:
		return app.state.filterValues.Skills
	case filterDomains:
		return app.state.filterValues.Domains
	case filterModules:
		return app.state.filterValues.Modules
	case filterOASFVersion:
		return app.state.filterValues.OASFVersions
	case filterVersion:
		return app.state.filterValues.Versions
	case filterAuthor:
		return app.state.filterValues.Authors
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

// recordMatchesFilters returns true if the supplied record satisfies every
// active filter category. Each non-empty category contributes an OR over its
// selected options, and categories AND together.
func (app *Gui) recordMatchesFilters(r *dirclient.RecordSummary) bool {
	for c, set := range app.state.filters.applied {
		if len(set) == 0 {
			continue
		}
		if !recordMatchesCategory(r, c, set) {
			return false
		}
	}
	return true
}

func recordMatchesCategory(r *dirclient.RecordSummary, c filterCategory, set map[string]bool) bool {
	switch c {
	case filterSkills:
		return anyIn(r.Skills, set)
	case filterDomains:
		return anyIn(r.Domains, set)
	case filterModules:
		return anyIn(r.Modules, set)
	case filterOASFVersion:
		return set[r.SchemaVersion]
	case filterVersion:
		return set[r.Version]
	case filterAuthor:
		return anyIn(r.Authors, set)
	case filterTrusted, filterVerified:
		// For lack of a server-side trust attestation cached locally, both
		// options use the same "has signature" proxy. They behave identically
		// today; keeping them distinct preserves the intent so a future server
		// integration can refine each independently without UX changes.
		want := ""
		switch {
		case set["yes"] && !set["no"]:
			want = "yes"
		case set["no"] && !set["yes"]:
			want = "no"
		default:
			return true // both selected (or neither — treat as no filter)
		}
		if want == "yes" {
			return r.Signed
		}
		return !r.Signed
	}
	return true
}

func anyIn(values []string, set map[string]bool) bool {
	for _, v := range values {
		if set[v] {
			return true
		}
	}
	return false
}
