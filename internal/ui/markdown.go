package ui

import (
	"os"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

// glamourStyle returns the glamour style name ("dark" or "light") for the
// current terminal, resolved without probing the terminal. Using a fixed
// style instead of glamour.WithAutoStyle() avoids the OSC 11 background query,
// whose raw response can otherwise leak onto the screen.
func glamourStyle() string {
	if BackgroundIsDark() {
		return "dark"
	}
	return "light"
}

// RenderMarkdown renders markdown text using glamour with a style chosen from
// the resolved terminal background (see BackgroundIsDark).
// Returns the rendered markdown or the original text if rendering fails.
// Word wraps at terminal width (capped at 100 chars for readability).
func RenderMarkdown(markdown string) string {
	if !ShouldUseColor() {
		return markdown
	}

	const maxReadableWidth = 100
	wrapWidth := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		wrapWidth = w
	}
	if wrapWidth > maxReadableWidth {
		wrapWidth = maxReadableWidth
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return markdown
	}

	rendered, err := renderer.Render(markdown)
	if err != nil {
		return markdown
	}

	return rendered
}
