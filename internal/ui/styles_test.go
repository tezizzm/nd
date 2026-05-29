package ui

import (
	"strings"
	"testing"
)

func TestRenderStatusIcon(t *testing.T) {
	for _, status := range []string{"open", "in_progress", "blocked", "closed", "deferred", "bogus"} {
		if got := RenderStatusIcon(status); got == "" {
			t.Errorf("RenderStatusIcon(%q) returned empty string", status)
		}
	}
}

func TestRenderStatus(t *testing.T) {
	for _, status := range []string{"open", "in_progress", "blocked", "closed", "deferred"} {
		if stripANSI(RenderStatus(status)) != status {
			t.Errorf("RenderStatus(%q) text = %q, want %q", status, stripANSI(RenderStatus(status)), status)
		}
	}
	if RenderStatus("open") != "open" {
		t.Errorf("RenderStatus(open) should be unstyled, got %q", RenderStatus("open"))
	}
}

func TestRenderPriority(t *testing.T) {
	for p := 0; p <= 3; p++ {
		got := RenderPriority(p)
		if !strings.Contains(got, "P") {
			t.Errorf("RenderPriority(%d) = %q, missing priority label", p, got)
		}
	}
}

func TestRenderType(t *testing.T) {
	for _, typ := range []string{"bug", "epic", "task"} {
		if got := RenderType(typ); stripANSI(got) != typ {
			t.Errorf("RenderType(%q) text = %q, want %q", typ, stripANSI(got), typ)
		}
	}
}

func TestRenderPassthroughHelpers(t *testing.T) {
	if RenderID("abc-123") != "abc-123" {
		t.Errorf("RenderID changed its input")
	}
	for _, fn := range []func(string) string{RenderMuted, RenderBold, RenderAccent, RenderClosedLine} {
		if stripANSI(fn("hello")) != "hello" {
			t.Errorf("render helper altered text: got %q", stripANSI(fn("hello")))
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	t.Run("no color returns input unchanged", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		in := "# Title\n\nbody"
		if got := RenderMarkdown(in); got != in {
			t.Errorf("RenderMarkdown with NO_COLOR = %q, want unchanged", got)
		}
	})

	t.Run("forced color renders without leaking an OSC response", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("CLICOLOR", "")
		t.Setenv("CLICOLOR_FORCE", "1")
		t.Setenv("ND_THEME", "dark")
		got := RenderMarkdown("# Heading\n\nsome text")
		if !strings.Contains(got, "Heading") {
			t.Errorf("RenderMarkdown dropped content: %q", got)
		}
		// The bug being fixed: an OSC 11 background query response leaking
		// into output. Ensure no OSC 11 sequence appears in rendered markdown.
		if strings.Contains(got, "]11;") {
			t.Errorf("RenderMarkdown leaked an OSC 11 response: %q", got)
		}
	})
}

// stripANSI removes ANSI escape sequences so tests can assert on visible text.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			// CSI/SGR sequences terminate on a letter; OSC on BEL or ST.
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == 0x07 {
				inEsc = false
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
