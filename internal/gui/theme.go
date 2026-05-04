package gui

import (
	"github.com/akijakya/lazydir/internal/config"
)

// Theme holds all ANSI color escape sequences used throughout the TUI.
// Each slot defaults to a base16 terminal color and can be overridden in
// the config file (~/.config/lazydir/config.yml) under gui.theme.
type Theme struct {
	Color1  string // default: yellow       — skills, annotations
	Color2  string // default: cyan         — domains, class tree, accents
	Color3  string // default: magenta      — modules, timestamps
	Color4  string // default: green        — connected, OASF version, loading
	Color5  string // default: blue         — version, options bar, section headers
	Color6  string // default: red          — disconnected indicator
	Color7  string // default: bright red   — author filter
	Color8  string // default: bright yellow— trusted filter
	Color9  string // default: bright green — verified filter
	Color10 string // default: bright black — dim/muted (IDs, gray)
	Reset   string // ANSI reset sequence
}

var defaultTheme = Theme{
	Color1:  "\033[33m",
	Color2:  "\033[36m",
	Color3:  "\033[35m",
	Color4:  "\033[32m",
	Color5:  "\033[34m",
	Color6:  "\033[31m",
	Color7:  "\033[91m",
	Color8:  "\033[93m",
	Color9:  "\033[92m",
	Color10: "\033[90m",
	Reset:   "\033[0m",
}

func newTheme(cfg config.ThemeConfig) Theme {
	t := defaultTheme
	t.Color1 = config.ResolveColor(cfg.Color1, t.Color1)
	t.Color2 = config.ResolveColor(cfg.Color2, t.Color2)
	t.Color3 = config.ResolveColor(cfg.Color3, t.Color3)
	t.Color4 = config.ResolveColor(cfg.Color4, t.Color4)
	t.Color5 = config.ResolveColor(cfg.Color5, t.Color5)
	t.Color6 = config.ResolveColor(cfg.Color6, t.Color6)
	t.Color7 = config.ResolveColor(cfg.Color7, t.Color7)
	t.Color8 = config.ResolveColor(cfg.Color8, t.Color8)
	t.Color9 = config.ResolveColor(cfg.Color9, t.Color9)
	t.Color10 = config.ResolveColor(cfg.Color10, t.Color10)
	return t
}

// filterColor returns the ANSI color code for a given filter category.
func (t Theme) filterColor(c filterCategory) string {
	switch c {
	case filterSkills:
		return t.Color1
	case filterDomains:
		return t.Color2
	case filterModules:
		return t.Color3
	case filterOASFVersion:
		return t.Color4
	case filterVersion:
		return t.Color5
	case filterAuthor:
		return t.Color7
	case filterTrusted:
		return t.Color8
	case filterVerified:
		return t.Color9
	}
	return ""
}
