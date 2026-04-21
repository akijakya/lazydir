package app

import tea "charm.land/bubbletea/v2"

// focusedPanel tracks which panel has keyboard focus.
type focusedPanel int

const (
	panelDirectory focusedPanel = iota
	panelClasses
	panelRecords
	panelPreview
	panelCount
)

// globalKeyMsg is sent when a globally-handled key is pressed.
type globalKeyMsg struct{ key string }

// handleGlobalKeys processes keys that work regardless of focused panel.
func handleGlobalKeys(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit, true
	}

	return nil, false
}
