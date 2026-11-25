package main

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// The former page's look, element by element: every size, color and timing
// here is the value its styles.css gave, so the native window looks the same.

const cssTransition = 200 * time.Millisecond // transition: … .2s

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// brightness is CSS's filter: brightness(f) on one color.
func brightness(c ui.Color, f float32) ui.Color {
	ch := func(v uint8) uint8 { return uint8(math.Min(255, math.Round(float64(v)*float64(f)))) }
	return ui.Color{R: ch(c.R), G: ch(c.G), B: ch(c.B), A: c.A}
}

// lerpColor mixes a toward b by t (0…1), as a CSS color transition does.
func lerpColor(a, b ui.Color, t float32) ui.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return a.Mix(b, t)
}

// cssButton is a <button>: it takes the focus, shows the pointer, dims to
// .45 when disabled, eases its hover in over .2 s and sinks 1 DIP while
// pressed. style paints it, given how far the hover has eased in.
func cssButton(c *ui.Context, disabled bool, style func(b ui.Element, hover float32)) ui.Element {
	b := ui.ButtonBase(c).Disabled(disabled).DimDisabled(false)
	hover := b.Animate("hover", b2f(b.Hovered() && !disabled), cssTransition)
	if disabled {
		b.Opacity(0.45)
	} else {
		b.Cursor(ui.CursorPointer)
		if b.Pressed() {
			b.Top(1)
		}
	}
	style(b, hover)
	return b
}

// look is a button's colors at rest and on hover; filter is the hover's
// brightness() over the whole button (1 for none).
type look struct {
	bg, fg, border    ui.Color
	hbg, hfg, hborder ui.Color
	filter            float32
}

func (l look) apply(b ui.Element, hover float32) {
	f := 1 + (l.filter-1)*hover
	pick := func(rest, hovered ui.Color) ui.Color {
		return brightness(lerpColor(rest, hovered, hover), f)
	}
	b.Background(pick(l.bg, l.hbg)).TextColor(pick(l.fg, l.hfg))
	if l.border.A != 0 || l.hborder.A != 0 {
		b.Border(1, pick(l.border, l.hborder))
	}
}

// subtleButton is .subtle-button: transparent, muted, a hairline border.
func subtleButton(c *ui.Context, p *palette, label string, disabled bool) ui.Element {
	return cssButton(c, disabled, func(b ui.Element, h float32) {
		b.Radius(9).Padding(9, 13).NoWrap()
		look{bg: p.Panel.Alpha(0), fg: p.Muted, border: p.Line, hbg: p.HoverBg, hfg: p.Text, hborder: p.Line, filter: 1.12}.apply(b, h)
		b.Children(func() { ui.Text(c, label).FontSize(11).SingleLine() })
	})
}

// primaryButton is .primary-button: the accent, 42 high, 12 px semibold.
func primaryButton(c *ui.Context, p *palette, label, icon string, disabled bool) ui.Element {
	return cssButton(c, disabled, func(b ui.Element, h float32) {
		b.Radius(11).Padding(12, 16).MinHeight(42).Gap(10).Justify(ui.Center)
		look{bg: p.Accent, fg: p.AccentInk, border: p.Accent, hbg: p.Accent, hfg: p.AccentInk, hborder: p.Accent, filter: 0.95}.apply(b, h)
		b.Children(func() {
			if icon != "" {
				ui.Icon(c, lucide[icon]).Size(16, 16)
			}
			ui.Text(c, label).FontSize(12).FontWeight(600).SingleLine()
		})
	})
}

// ellipseGradient paints CSS's radial-gradient(ellipse at cx cy, …) with
// its farthest-corner size: an ellipse of the farthest sides' proportions
// through the farthest corner.
func ellipseGradient(p *ui.Painter, r ui.Rect, cx, cy float32, stops []ui.GradientStop) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	px, py := r.X+r.W*cx, r.Y+r.H*cy
	rx := max(cx, 1-cx) * r.W * math.Sqrt2
	ry := max(cy, 1-cy) * r.H * math.Sqrt2
	if rx <= 0 || ry <= 0 {
		return
	}
	sy := ry / rx
	p.Clip(r, 0, func() {
		p.Transform(ui.Scaling(1, sy), px, py, func() {
			lr := ui.Rect{X: r.X, Y: py - (py-r.Y)/sy, W: r.W, H: r.H / sy}
			p.FillRadialGradient(lr, ui.RadialGradient{
				Center: ui.Point{X: cx, Y: (py - lr.Y) / lr.H},
				Radius: rx, Stops: stops,
			}, 0)
		})
	})
}

// chevron is the 7×7 two-sided border the collapsibles turn: pointing down
// closed (45°), up open (225°), lifted by its -4px top margin while closed
// (2 DIPs, as a centered flex item).
func chevron(c *ui.Context, color ui.Color, open bool) {
	box := ui.Box(c).Size(7, 7)
	turn := box.Animate("turn", b2f(open), 250*time.Millisecond)
	box.Top(-2 * (1 - turn)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		p.Transform(ui.Rotation(45+180*turn), cx, cy, func() {
			edge := (&ui.Path{}).MoveTo(r.X+r.W-0.75, r.Y).LineTo(r.X+r.W-0.75, r.Y+r.H-0.75).LineTo(r.X, r.Y+r.H-0.75)
			p.StrokePath(edge.Cap(ui.ButtCap).Join(ui.MiterJoin), 1.5, color)
		})
	})
}

// dot is the 5×5 accent dot of the online state and the reef caption.
func dot(c *ui.Context, color ui.Color) {
	ui.Box(c).Size(5, 5).Radius(2.5).Background(color).Shrink(0)
}

// enter brings an element in from a pose as it first shows, as the page's
// CSS keyframe animations (pop, fade, reveal, toast-in) did.
func enter(c *ui.Context, e ui.Element, from ui.Pose, a ui.Animation) {
	entered := ui.Local(e, "entered", func() bool { return false })
	pose := ui.Pose{}
	if !*entered {
		pose, *entered = from, true
		c.Invalidate()
	}
	e.AnimatePose(pose, a)
}

// cssEase is CSS's "ease" timing function.
var cssEase = ui.CubicBezier(0.25, 0.1, 0.25, 1)

// cssLine is the height of one line of text at size under CSS's
// line-height: normal for the page's CJK font (Microsoft YaHei, PingFang SC).
func cssLine(size float32) float32 { return size * 1.32 }

// The page drew 咖啡館 with lucide's coffee cup, as it did 開始休息.
func init() { lucide["cafe"] = lucide["coffee"] }

// ellipse is the path of an axis-aligned ellipse, as border-radius: 50%
// rounds a box that is not square.
func ellipse(cx, cy, rx, ry float32) *ui.Path {
	const k = 0.5522848
	return (&ui.Path{}).MoveTo(cx+rx, cy).
		CubeTo(cx+rx, cy+ry*k, cx+rx*k, cy+ry, cx, cy+ry).
		CubeTo(cx-rx*k, cy+ry, cx-rx, cy+ry*k, cx-rx, cy).
		CubeTo(cx-rx, cy-ry*k, cx-rx*k, cy-ry, cx, cy-ry).
		CubeTo(cx+rx*k, cy-ry, cx+rx, cy-ry*k, cx+rx, cy).Close()
}
