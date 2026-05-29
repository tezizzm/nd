// Package ui provides terminal styling and output helpers for nd CLI.
package ui

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// IsTerminal returns true if stdout is connected to a terminal (TTY).
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// ShouldUseColor determines if ANSI color codes should be used.
// Respects standard conventions:
//   - NO_COLOR: https://no-color.org/ - disables color if set
//   - CLICOLOR=0: disables color
//   - CLICOLOR_FORCE: forces color even in non-TTY
//   - Falls back to TTY detection
func ShouldUseColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}
	return IsTerminal()
}

// BackgroundIsDark reports whether nd should render for a dark terminal
// background.
//
// It deliberately does NOT query the terminal at runtime. The usual way to
// learn the background color is an OSC 11 query ("report background color"),
// but that probe is fragile: some terminals answer asynchronously or emit the
// cursor-position report before the color report, which makes the detection
// library give up and leaves the raw reply (e.g. "\x1b]11;rgb:0a0a/0c0c/1010")
// to be echoed onto the screen. nd resolves the theme deterministically from
// the environment instead, defaulting to dark -- the common case for terminal
// tooling.
//
// Resolution order:
//   - ND_THEME=light|dark : explicit override, wins over everything
//   - COLORFGBG="fg;bg"   : honored when the background field is an ANSI index
//   - default             : dark
func BackgroundIsDark() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ND_THEME"))) {
	case "light":
		return false
	case "dark":
		return true
	}

	if bg, ok := colorFGBGBackground(os.Getenv("COLORFGBG")); ok {
		// ANSI indices 7 (white) and 15 (bright white) denote a light
		// background; every other index is treated as dark.
		return bg != 7 && bg != 15
	}

	return true
}

// colorFGBGBackground extracts the background ANSI color index from a COLORFGBG
// value of the form "fg;bg" (e.g. "15;0"). Some terminals emit a three-field
// "fg;default;bg" form, so the background is always the last field. It returns
// false when the value is missing or malformed.
func colorFGBGBackground(v string) (int, bool) {
	if !strings.Contains(v, ";") {
		return 0, false
	}
	parts := strings.Split(v, ";")
	bg, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil {
		return 0, false
	}
	return bg, true
}
