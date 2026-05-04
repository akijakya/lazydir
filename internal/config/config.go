package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config mirrors the YAML structure of ~/.config/lazydir/config.yml.
type Config struct {
	GUI GUIConfig `yaml:"gui"`
}

// GUIConfig groups all visual/TUI settings.
type GUIConfig struct {
	Theme ThemeConfig `yaml:"theme"`
}

// ThemeConfig lets users override the base16 color palette used throughout the
// TUI. Each field accepts a color name ("red", "brightCyan", …), a 256-color
// index ("42"), or a hex true-color value ("#ff8800").
type ThemeConfig struct {
	Color1  string `yaml:"color1"`
	Color2  string `yaml:"color2"`
	Color3  string `yaml:"color3"`
	Color4  string `yaml:"color4"`
	Color5  string `yaml:"color5"`
	Color6  string `yaml:"color6"`
	Color7  string `yaml:"color7"`
	Color8  string `yaml:"color8"`
	Color9  string `yaml:"color9"`
	Color10 string `yaml:"color10"`
}

// Dir returns the lazydir configuration directory, respecting XDG_CONFIG_HOME.
func Dir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lazydir")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "lazydir")
}

// Load reads the config file from the standard location. If the file does not
// exist or cannot be parsed, a zero-value Config is returned (all defaults).
func Load() Config {
	var cfg Config
	dir := Dir()
	if dir == "" {
		return cfg
	}
	for _, name := range []string{"config.yml", "config.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		_ = yaml.Unmarshal(data, &cfg)
		return cfg
	}
	return cfg
}

// colorNames maps base16 color names to ANSI SGR foreground escape codes.
var colorNames = map[string]string{
	"black":         "\033[30m",
	"red":           "\033[31m",
	"green":         "\033[32m",
	"yellow":        "\033[33m",
	"blue":          "\033[34m",
	"magenta":       "\033[35m",
	"cyan":          "\033[36m",
	"white":         "\033[37m",
	"brightBlack":   "\033[90m",
	"brightRed":     "\033[91m",
	"brightGreen":   "\033[92m",
	"brightYellow":  "\033[93m",
	"brightBlue":    "\033[94m",
	"brightMagenta": "\033[95m",
	"brightCyan":    "\033[96m",
	"brightWhite":   "\033[97m",
}

// ResolveColor converts a user-supplied color value to an ANSI escape code.
// Accepted formats:
//   - Color name: "red", "brightCyan", …
//   - 256-color index: "42"
//   - Hex true-color: "#ff8800"
//
// If the value is empty or unrecognised, fallback is returned unchanged.
func ResolveColor(value, fallback string) string {
	if value == "" {
		return fallback
	}
	value = strings.TrimSpace(value)
	if code, ok := colorNames[value]; ok {
		return code
	}
	if n, err := strconv.Atoi(value); err == nil && n >= 0 && n <= 255 {
		return fmt.Sprintf("\033[38;5;%dm", n)
	}
	if strings.HasPrefix(value, "#") && len(value) == 7 {
		r, err1 := strconv.ParseUint(value[1:3], 16, 8)
		g, err2 := strconv.ParseUint(value[3:5], 16, 8)
		b, err3 := strconv.ParseUint(value[5:7], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
		}
	}
	return fallback
}
