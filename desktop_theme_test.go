package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPaletteDarkBase(t *testing.T) {
	// reef-realistic has supportsDarkAppearance: true.
	p, err := loadPalette("reef-realistic")
	if err != nil {
		t.Fatal(err)
	}
	th := p.uiTheme()
	if !th.Dark || th.Background != ui.Hex("#0c1820") {
		t.Fatalf("bgBase not mapped onto a dark theme: %+v", th.Background)
	}
	// Should inherit DarkTheme's Selection/Danger defaults.
	if th.Selection == (ui.Color{}) || th.Danger == (ui.Color{}) {
		t.Fatal("dark base must retain Selection and Danger")
	}
	if p.Panel != ui.Hex("#12232b") || p.Line != ui.Hex("#254047") || p.Radius != 20 || p.Coral != ui.Hex("#bd918c") {
		t.Fatalf("tokens not mapped as the page's CSS variables were: %+v", p)
	}
}

func TestPaletteLightThemeInheritsAndOverrides(t *testing.T) {
	p, err := loadPalette("ct-cream")
	if err != nil {
		t.Fatal(err)
	}
	th := p.uiTheme()
	if th.Dark || th.Selection == (ui.Color{}) || th.Danger == (ui.Color{}) {
		t.Fatal("light base must be light and retain Selection and Danger")
	}
	if p.Accent != ui.Hex("#C4798D") || p.Radius != 16 || p.MaxCreatures != 6 || p.FishColor != "#F6C6A0" || p.BubbleDensity != 0.4 {
		t.Fatalf("ct-cream overrides were not applied: %+v", p)
	}
}

func TestEveryThemeLoadsAPalette(t *testing.T) {
	themes, err := installedThemes()
	if err != nil {
		t.Fatal(err)
	}
	for _, th := range themes {
		p, err := loadPalette(th.ID)
		if err != nil {
			t.Errorf("%s: %v", th.ID, err)
			continue
		}
		for name, c := range map[string]ui.Color{"bg": p.Bg, "panel": p.Panel, "text": p.Text, "accent": p.Accent, "sand": p.Sand} {
			if c.A == 0 {
				t.Errorf("%s: %s token is missing", th.ID, name)
			}
		}
	}
}

func TestFontStackFallsBackFromYaHeiToJhengHei(t *testing.T) {
	themes, err := installedThemes()
	if err != nil {
		t.Fatal(err)
	}
	for _, th := range themes {
		p, err := loadPalette(th.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, stack := range []string{p.font(), p.digitsFont()} {
			yh, jh := strings.Index(stack, "Microsoft YaHei"), strings.Index(stack, "Microsoft JhengHei")
			if yh < 0 || jh < yh || strings.Contains(stack[:yh], "JhengHei") {
				t.Errorf("%s: JhengHei must come right after YaHei: %q", th.ID, stack)
			}
			if !strings.Contains(stack, "Microsoft YaHei, Microsoft JhengHei") {
				t.Errorf("%s: JhengHei is not the next choice after YaHei: %q", th.ID, stack)
			}
		}
	}
}
