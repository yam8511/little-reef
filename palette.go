package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"little-reef/internal/reef"
)

// Everything the app ships lives in the binary: the theme contracts and the
// public-domain ambience recordings. No network access is ever needed.
//
//go:embed assets/themes assets/sounds
var bundled embed.FS

func soundFiles() fs.FS {
	sub, err := fs.Sub(bundled, "assets/sounds")
	if err != nil {
		panic(err)
	}
	return sub
}

// palette is one theme's design tokens merged over reef-realistic, exactly as
// the former web theme layer rebound its CSS variables.
type palette struct {
	ID   string
	Dark bool

	Bg, BgGradient, BgGlow, WaterLight          ui.Color
	Sand, Coral, Fish, Seaweed, Bubble          ui.Color
	Accent, AccentInk, TimerActive, TimerPause  ui.Color
	Panel, Line, PanelDeep, PanelRaised         ui.Color
	FieldBg, ActiveBg, ActiveBorder, RingTrack  ui.Color
	HoverBg, PopBg, Text, Muted, Faint, MenuIcn ui.Color

	Radius        float32 // --radius, the large panels' corners
	TitleFont     string  // --font-title
	DigitsFont    string  // --font-digits ("" = inherit the title font)
	BubbleDensity float64
	MaxCreatures  int
	FishColor     string
}

type themeFile struct {
	Meta struct {
		ThemeID                string `json:"themeId"`
		SupportsDarkAppearance *bool  `json:"supportsDarkAppearance"`
	} `json:"meta"`
	Tokens   map[string]json.RawMessage `json:"tokens"`
	Motion   map[string]json.RawMessage `json:"motion"`
	Controls map[string]string          `json:"controls"`
	Fonts    map[string]string          `json:"fonts"`
	Behavior *struct {
		MaxCreatures int `json:"maxCreatures"`
	} `json:"behavior"`
	CreaturePalette *struct {
		Fish string `json:"fish"`
	} `json:"creaturePalette"`
}

func readThemeFile(id string) (themeFile, error) {
	var t themeFile
	data, err := fs.ReadFile(bundled, "assets/themes/"+id+"/theme.json")
	if err != nil {
		return t, err
	}
	return t, json.Unmarshal(data, &t)
}

// tokenColor flattens a token that is a hex string or a {color, stages} record.
func tokenColor(raw json.RawMessage) (string, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, s != ""
	}
	var rec struct {
		Color string `json:"color"`
	}
	if json.Unmarshal(raw, &rec) == nil && rec.Color != "" {
		return rec.Color, true
	}
	return "", false
}

func cssFirstFamily(stack string) string {
	if stack == "" || stack == "inherit" {
		return ""
	}
	return stack
}

// loadPalette merges a theme onto reef-realistic so omitted tokens inherit.
func loadPalette(id string) (*palette, error) {
	base, err := readThemeFile(defaultTheme)
	if err != nil {
		return nil, err
	}
	own := base
	if id != "" && id != defaultTheme {
		if own, err = readThemeFile(id); err != nil {
			return nil, err
		}
	} else {
		id = defaultTheme
	}
	tokens := map[string]string{}
	for _, src := range []map[string]json.RawMessage{base.Tokens, own.Tokens} {
		for k, raw := range src {
			if c, ok := tokenColor(raw); ok {
				tokens[k] = c
			}
		}
	}
	col := func(k string) ui.Color { return cssColor(tokens[k]) }
	p := &palette{
		ID: id, Dark: own.Meta.SupportsDarkAppearance == nil || *own.Meta.SupportsDarkAppearance,
		Bg: col("bgBase"), BgGradient: col("bgGradient"), BgGlow: col("bgGlow"), WaterLight: col("waterLight"),
		Sand: col("sand"), Coral: col("coral"), Fish: col("fish"), Seaweed: col("seaweed"), Bubble: col("bubble"),
		Accent: col("accent"), AccentInk: col("accentInk"), TimerActive: col("timerActive"), TimerPause: col("timerPause"),
		Panel: col("controlBg"), Line: col("controlBorder"), PanelDeep: col("panelDeep"), PanelRaised: col("panelRaised"),
		FieldBg: col("fieldBg"), ActiveBg: col("activeBg"), ActiveBorder: col("activeBorder"), RingTrack: col("ringTrack"),
		HoverBg: col("hoverBg"), PopBg: col("popBg"), Text: col("textPrimary"), Muted: col("textSecondary"), Faint: col("textFaint"),
		MenuIcn: col("menuIcon"), Radius: 20, BubbleDensity: 1, MaxCreatures: 999,
	}
	merge := func(dst map[string]string, srcs ...map[string]string) {
		for _, s := range srcs {
			for k, v := range s {
				dst[k] = v
			}
		}
	}
	controls, fonts := map[string]string{}, map[string]string{}
	merge(controls, base.Controls, own.Controls)
	merge(fonts, base.Fonts, own.Fonts)
	if r, err := strconv.ParseFloat(strings.TrimSuffix(controls["radius"], "px"), 32); err == nil {
		p.Radius = float32(r)
	}
	p.TitleFont = cssFirstFamily(fonts["title"])
	p.DigitsFont = cssFirstFamily(fonts["digits"])
	motion := map[string]json.RawMessage{}
	for k, v := range base.Motion {
		motion[k] = v
	}
	for k, v := range own.Motion {
		motion[k] = v
	}
	if raw, ok := motion["bubbleDensity"]; ok {
		_ = json.Unmarshal(raw, &p.BubbleDensity)
	}
	behavior := own.Behavior
	if behavior == nil {
		behavior = base.Behavior
	}
	if behavior != nil && behavior.MaxCreatures > 0 {
		p.MaxCreatures = behavior.MaxCreatures
	}
	cp := own.CreaturePalette
	if cp == nil {
		cp = base.CreaturePalette
	}
	if cp != nil {
		p.FishColor = cp.Fish
	}
	return p, nil
}

func (p *palette) reef() reef.Palette {
	return reef.Palette{Sand: p.Sand, Coral: p.Coral, Seaweed: p.Seaweed, Bubble: p.Bubble, Accent: p.Accent,
		BubbleDensity: p.BubbleDensity, MaxCreatures: p.MaxCreatures, FishColor: p.FishColor}
}

// uiTheme is the MyGo theme every window and the tray panel draw with: the
// light or dark base, rebound to the palette's tokens and fonts.
func (p *palette) uiTheme() *ui.Theme {
	var th ui.Theme
	if p.Dark {
		th = *ui.DarkTheme()
	} else {
		th = *ui.LightTheme()
	}
	th.Background = p.Bg
	th.Surface = p.PanelRaised
	th.Border = p.Line
	th.Text = p.Text
	th.TextMuted = p.Faint
	th.Accent = p.Accent
	th.AccentText = p.AccentInk
	th.Focus = p.ActiveBorder
	th.InactiveAccent = p.Accent
	th.Font = p.font()
	return &th
}

// font is the CSS font stack the page used: the theme's title stack, then
// the default "PingFang SC","Microsoft YaHei",system-ui. Microsoft JhengHei
// follows Microsoft YaHei wherever it is listed, so a Windows without YaHei
// shows the Traditional Chinese text in JhengHei rather than a generic font.
func (p *palette) font() string {
	const fallback = `PingFang SC, Microsoft YaHei, system-ui, sans-serif`
	stack := fallback
	if p.TitleFont != "" {
		stack = strings.ReplaceAll(p.TitleFont, `"`, "") + ", " + fallback
	}
	return strings.ReplaceAll(stack, "Microsoft YaHei", "Microsoft YaHei, Microsoft JhengHei")
}

func (p *palette) digitsFont() string {
	if p.DigitsFont == "" {
		return p.font()
	}
	return p.DigitsFont + ", " + p.font()
}

// paletteCache parses theme.json once per theme id.
type paletteCache struct {
	mu   sync.Mutex
	byID map[string]*palette
}

func (c *paletteCache) get(id string) *palette {
	if id == "" {
		id = defaultTheme
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if p := c.byID[id]; p != nil {
		return p
	}
	p, err := loadPalette(id)
	if err != nil {
		if id == defaultTheme {
			panic(err)
		}
		return c.byIDLocked(defaultTheme)
	}
	if c.byID == nil {
		c.byID = map[string]*palette{}
	}
	c.byID[id] = p
	return p
}

func (c *paletteCache) byIDLocked(id string) *palette {
	if p := c.byID[id]; p != nil {
		return p
	}
	p, err := loadPalette(id)
	if err != nil {
		panic(err)
	}
	if c.byID == nil {
		c.byID = map[string]*palette{}
	}
	c.byID[id] = p
	return p
}

var palettes paletteCache

// cssColor reads the color syntaxes theme tokens use: #rgb, #rrggbb,
// #rrggbbaa and rgb()/rgba(). Anything else is transparent.
func cssColor(s string) ui.Color {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		switch len(s) {
		case 4, 7, 9:
			return ui.Hex(s)
		}
		return ui.Color{}
	}
	open, end := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if open < 0 || end < open || !strings.HasPrefix(s, "rgb") {
		return ui.Color{}
	}
	parts := strings.FieldsFunc(s[open+1:end], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
	if len(parts) < 3 {
		return ui.Color{}
	}
	var ch [3]uint8
	for i := range ch {
		v, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return ui.Color{}
		}
		ch[i] = uint8(min(255, max(0, v)))
	}
	alpha := 1.0
	if len(parts) > 3 {
		a := parts[3]
		pct := strings.HasSuffix(a, "%")
		v, err := strconv.ParseFloat(strings.TrimSuffix(a, "%"), 64)
		if err == nil {
			if pct {
				v /= 100
			}
			alpha = min(1, max(0, v))
		}
	}
	return ui.RGBA(ch[0], ch[1], ch[2], float32(alpha))
}
