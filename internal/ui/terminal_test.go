package ui

import "testing"

func TestBackgroundIsDark(t *testing.T) {
	tests := []struct {
		name     string
		ndTheme  string
		colorFGB string
		want     bool
	}{
		{name: "default is dark", want: true},
		{name: "ND_THEME=light forces light", ndTheme: "light", want: false},
		{name: "ND_THEME=dark forces dark", ndTheme: "dark", want: true},
		{name: "ND_THEME is case-insensitive", ndTheme: "LIGHT", want: false},
		{name: "ND_THEME trims whitespace", ndTheme: "  dark  ", want: true},
		{name: "ND_THEME wins over COLORFGBG", ndTheme: "light", colorFGB: "15;0", want: false},
		{name: "COLORFGBG dark background (bg 0)", colorFGB: "15;0", want: true},
		{name: "COLORFGBG light background (bg 15)", colorFGB: "0;15", want: false},
		{name: "COLORFGBG light background (bg 7)", colorFGB: "0;7", want: false},
		{name: "COLORFGBG three-field form uses last", colorFGB: "1;default;0", want: true},
		{name: "COLORFGBG malformed falls back to dark", colorFGB: "garbage", want: true},
		{name: "COLORFGBG non-numeric bg falls back to dark", colorFGB: "15;white", want: true},
		{name: "unknown ND_THEME falls through to default", ndTheme: "solarized", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ND_THEME", tc.ndTheme)
			t.Setenv("COLORFGBG", tc.colorFGB)
			if got := BackgroundIsDark(); got != tc.want {
				t.Errorf("BackgroundIsDark() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestColorFGBGBackground(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		wantBG int
		wantOK bool
	}{
		{name: "empty", value: "", wantOK: false},
		{name: "no separator", value: "15", wantOK: false},
		{name: "fg;bg", value: "15;0", wantBG: 0, wantOK: true},
		{name: "light bg", value: "0;15", wantBG: 15, wantOK: true},
		{name: "three fields uses last", value: "1;default;7", wantBG: 7, wantOK: true},
		{name: "whitespace around bg", value: "15; 0 ", wantBG: 0, wantOK: true},
		{name: "non-numeric bg", value: "15;white", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bg, ok := colorFGBGBackground(tc.value)
			if ok != tc.wantOK {
				t.Fatalf("colorFGBGBackground(%q) ok = %v, want %v", tc.value, ok, tc.wantOK)
			}
			if ok && bg != tc.wantBG {
				t.Errorf("colorFGBGBackground(%q) bg = %d, want %d", tc.value, bg, tc.wantBG)
			}
		})
	}
}

func TestGlamourStyle(t *testing.T) {
	t.Run("dark background yields dark style", func(t *testing.T) {
		t.Setenv("ND_THEME", "dark")
		if got := glamourStyle(); got != "dark" {
			t.Errorf("glamourStyle() = %q, want %q", got, "dark")
		}
	})
	t.Run("light background yields light style", func(t *testing.T) {
		t.Setenv("ND_THEME", "light")
		if got := glamourStyle(); got != "light" {
			t.Errorf("glamourStyle() = %q, want %q", got, "light")
		}
	})
}

func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		name         string
		noColor      string
		clicolor     string
		clicolorForc string
		want         bool
	}{
		{name: "NO_COLOR disables", noColor: "1", want: false},
		{name: "CLICOLOR=0 disables", clicolor: "0", want: false},
		{name: "CLICOLOR_FORCE enables", clicolorForc: "1", want: true},
		{name: "NO_COLOR wins over CLICOLOR_FORCE", noColor: "1", clicolorForc: "1", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tc.noColor)
			t.Setenv("CLICOLOR", tc.clicolor)
			t.Setenv("CLICOLOR_FORCE", tc.clicolorForc)
			if got := ShouldUseColor(); got != tc.want {
				t.Errorf("ShouldUseColor() = %v, want %v", got, tc.want)
			}
		})
	}
}
