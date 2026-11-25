package reef

// A small Canvas2D work-alike over *ui.Painter, so the reef's creature code
// ports from the browser almost line for line.
//
// Drawing calls are recorded into a tree (clips own what is drawn after
// them until the matching restore, as a canvas clip lasts) and replayed
// onto the painter. Geometry is transformed to window DIPs on the CPU, so
// every path is rasterised at the size it shows (MyGo makes a path's
// coverage mask at the path's own size). Strokes under a non-uniform
// transform are drawn through a residual painter transform that only
// shrinks, which keeps their outline exactly what the canvas strokes.
// Linear gradients are mapped to MyGo's CSS-style gradients over the
// path's box; radial (two-circle) gradients are drawn by a small shader
// that evaluates the canvas formula per pixel.

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// mat is a canvas transform [a c e; b d f].
type mat struct{ a, b, c, d, e, f float64 }

var identity = mat{1, 0, 0, 1, 0, 0}

// mul returns m·n: n applies first, as canvas's transform() post-multiplies.
func (m mat) mul(n mat) mat {
	return mat{
		m.a*n.a + m.c*n.b, m.b*n.a + m.d*n.b,
		m.a*n.c + m.c*n.d, m.b*n.c + m.d*n.d,
		m.a*n.e + m.c*n.f + m.e, m.b*n.e + m.d*n.f + m.f,
	}
}

func (m mat) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.e, m.b*x + m.d*y + m.f
}

func (m mat) det() float64 { return m.a*m.d - m.b*m.c }

func (m mat) inv() mat {
	d := m.det()
	return mat{m.d / d, -m.b / d, -m.c / d, m.a / d, (m.c*m.f - m.d*m.e) / d, (m.b*m.e - m.a*m.f) / d}
}

// singular returns the largest and smallest stretch of m's linear part.
func (m mat) singular() (float64, float64) {
	s := m.a*m.a + m.b*m.b + m.c*m.c + m.d*m.d
	dt := math.Abs(m.det())
	q := math.Sqrt(math.Max(0, s*s-4*dt*dt))
	return math.Sqrt((s + q) / 2), math.Sqrt(math.Max(0, (s-q)/2))
}

// rgba is a CSS color: r, g, b in 0…255 (whole numbers), a in 0…1 (clamped when drawn).
type rgba struct{ r, g, b, a float64 }

func rgbaC(r, g, b, a float64) rgba { return rgba{r, g, b, a} }

// hexC parses "#rrggbb" or "#rrggbbaa".
func hexC(s string) rgba {
	v := func(i int) float64 {
		n := 0
		for _, ch := range s[i : i+2] {
			n *= 16
			switch {
			case ch >= '0' && ch <= '9':
				n += int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				n += int(ch-'a') + 10
			case ch >= 'A' && ch <= 'F':
				n += int(ch-'A') + 10
			}
		}
		return float64(n)
	}
	c := rgba{v(1), v(3), v(5), 1}
	if len(s) >= 9 {
		c.a = v(7) / 255
	}
	return c
}

// fromUI converts a theme color; alpha replaces its own, as appending "14" to a hex string does.
func fromUI(c ui.Color, alpha float64) rgba {
	return rgba{float64(c.R), float64(c.G), float64(c.B), alpha}
}

func (c rgba) ui(alpha float64) ui.Color {
	ch := func(v float64) uint8 {
		if !(v > 0) {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return ui.RGBA(ch(c.r), ch(c.g), ch(c.b), float32(clamp(c.a, 0, 1)*clamp(alpha, 0, 1)))
}

type gstop struct {
	at float64
	c  rgba
}

// gradient is a CanvasGradient: linear (x0,y0)→(x1,y1) or radial between two circles.
type gradient struct {
	radial                 bool
	x0, y0, r0, x1, y1, r1 float64
	stops                  []gstop
}

func (g *gradient) addColorStop(at float64, c rgba) {
	if at < 0 || at > 1 || math.IsNaN(at) {
		return
	}
	i := len(g.stops)
	for i > 0 && g.stops[i-1].at > at {
		i--
	}
	g.stops = append(g.stops, gstop{})
	copy(g.stops[i+1:], g.stops[i:])
	g.stops[i] = gstop{at, c}
}

// colorAt interpolates the stops without premultiplying, as the canvas does.
func (g *gradient) colorAt(t float64) rgba {
	s := g.stops
	if t <= s[0].at {
		return s[0].c
	}
	for i := 1; i < len(s); i++ {
		if t < s[i].at {
			u := (t - s[i-1].at) / (s[i].at - s[i-1].at)
			a, b := s[i-1].c, s[i].c
			return rgba{a.r + (b.r-a.r)*u, a.g + (b.g-a.g)*u, a.b + (b.b-a.b)*u, clamp(a.a, 0, 1) + (clamp(b.a, 0, 1)-clamp(a.a, 0, 1))*u}
		}
	}
	return s[len(s)-1].c
}

// Path segments.
type segKind uint8

const (
	segMove segKind = iota
	segLine
	segQuad
	segCubic
	segClose
)

type seg struct {
	k segKind
	p [6]float64
}

// path2D is a Path2D, and the context's current path (whose points are
// transformed when added, as a canvas does).
type path2D struct {
	segs    []seg
	has     bool // a current point exists
	pending bool // the current point starts a subpath not yet emitted
	cx, cy  float64
	sx, sy  float64
	m       mat
	useM    bool
}

func newPath() *path2D { return &path2D{} }

func (p *path2D) reset() {
	p.segs = p.segs[:0]
	p.has, p.pending = false, false
}

func (p *path2D) pt(x, y float64) (float64, float64) {
	if p.useM {
		return p.m.apply(x, y)
	}
	return x, y
}

func (p *path2D) begin() {
	if p.pending {
		p.segs = append(p.segs, seg{k: segMove, p: [6]float64{p.cx, p.cy}})
		p.sx, p.sy = p.cx, p.cy
		p.pending = false
	}
}

func finite(v ...float64) bool {
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
	}
	return true
}

func (p *path2D) moveTo(x, y float64) {
	if !finite(x, y) {
		return
	}
	p.cx, p.cy = p.pt(x, y)
	p.sx, p.sy = p.cx, p.cy
	p.has, p.pending = true, true
}

func (p *path2D) lineTo(x, y float64) {
	if !finite(x, y) {
		return
	}
	if !p.has {
		p.moveTo(x, y)
		return
	}
	p.begin()
	X, Y := p.pt(x, y)
	p.segs = append(p.segs, seg{k: segLine, p: [6]float64{X, Y}})
	p.cx, p.cy = X, Y
}

func (p *path2D) quadraticCurveTo(cpx, cpy, x, y float64) {
	if !finite(cpx, cpy, x, y) {
		return
	}
	if !p.has {
		p.moveTo(cpx, cpy)
	}
	p.begin()
	a, b := p.pt(cpx, cpy)
	X, Y := p.pt(x, y)
	p.segs = append(p.segs, seg{k: segQuad, p: [6]float64{a, b, X, Y}})
	p.cx, p.cy = X, Y
}

func (p *path2D) bezierCurveTo(c1x, c1y, c2x, c2y, x, y float64) {
	if !finite(c1x, c1y, c2x, c2y, x, y) {
		return
	}
	if !p.has {
		p.moveTo(c1x, c1y)
	}
	p.begin()
	a, b := p.pt(c1x, c1y)
	c, d := p.pt(c2x, c2y)
	X, Y := p.pt(x, y)
	p.segs = append(p.segs, seg{k: segCubic, p: [6]float64{a, b, c, d, X, Y}})
	p.cx, p.cy = X, Y
}

func (p *path2D) closePath() {
	if !p.has || p.pending {
		return
	}
	p.segs = append(p.segs, seg{k: segClose})
	p.cx, p.cy = p.sx, p.sy
	p.pending = true
}

func (p *path2D) rect(x, y, w, h float64) {
	if !finite(x, y, w, h) {
		return
	}
	p.moveTo(x, y)
	p.lineTo(x+w, y)
	p.lineTo(x+w, y+h)
	p.lineTo(x, y+h)
	p.closePath()
	p.moveTo(x, y)
}

func (p *path2D) arc(x, y, r, a0, a1 float64, ccw bool) {
	p.ellipse(x, y, r, r, 0, a0, a1, ccw)
}

// ellipse adds an elliptical arc, in cubic Béziers of at most a quarter turn.
func (p *path2D) ellipse(x, y, rx, ry, rot, a0, a1 float64, ccw bool) {
	if !finite(x, y, rx, ry, rot, a0, a1) || rx < 0 || ry < 0 {
		return
	}
	// The sweep as Chromium canonicalises it.
	start := math.Mod(a0, TAU)
	if start < 0 {
		start += TAU
	}
	end := a1 + (start - a0)
	var sweep float64
	switch {
	case !ccw && end-start >= TAU:
		sweep = TAU
	case ccw && start-end >= TAU:
		sweep = -TAU
	case !ccw && start > end:
		sweep = TAU - math.Mod(start-end, TAU)
	case ccw && start < end:
		sweep = -(TAU - math.Mod(end-start, TAU))
	default:
		sweep = end - start
	}
	cr, sr := math.Cos(rot), math.Sin(rot)
	at := func(t float64) (float64, float64, float64, float64) {
		ex, ey := rx*math.Cos(t), ry*math.Sin(t)
		dx, dy := -rx*math.Sin(t), ry*math.Cos(t)
		return x + cr*ex - sr*ey, y + sr*ex + cr*ey, cr*dx - sr*dy, sr*dx + cr*dy
	}
	sx, sy, _, _ := at(start)
	if p.has {
		p.lineTo(sx, sy)
	} else {
		p.moveTo(sx, sy)
	}
	if sweep == 0 {
		return
	}
	n := int(math.Ceil(math.Abs(sweep)/(math.Pi/2) - 1e-9))
	if n < 1 {
		n = 1
	}
	step := sweep / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	t := start
	for i := 0; i < n; i++ {
		x1, y1, dx1, dy1 := at(t)
		x2, y2, dx2, dy2 := at(t + step)
		p.bezierCurveTo(x1+k*dx1, y1+k*dy1, x2-k*dx2, y2-k*dy2, x2, y2)
		t += step
	}
}

// pathSink is what curve() writes to: a Path2D or the context.
type pathSink interface {
	moveTo(x, y float64)
	bezierCurveTo(c1x, c1y, c2x, c2y, x, y float64)
	closePath()
}

// transformed returns a path's segments mapped by m.
func transformed(segs []seg, m mat) []seg {
	out := make([]seg, len(segs))
	for i, s := range segs {
		out[i].k = s.k
		n := 0
		switch s.k {
		case segMove, segLine:
			n = 1
		case segQuad:
			n = 2
		case segCubic:
			n = 3
		}
		for j := 0; j < n; j++ {
			out[i].p[2*j], out[i].p[2*j+1] = m.apply(s.p[2*j], s.p[2*j+1])
		}
	}
	return out
}

// Recording.

type nodeKind uint8

const (
	nodeFill nodeKind = iota
	nodeStroke
	nodeClip
)

type node struct {
	kind  nodeKind
	segs  []seg // window DIPs
	m     mat   // the transform at the time, for widths and gradients
	style any   // rgba or *gradient
	width float64
	cap   ui.LineCap
	join  ui.LineJoin
	alpha float64
	kids  []*node
}

type state struct {
	m         mat
	fill      any
	stroke    any
	lineWidth float64
	cap       ui.LineCap
	join      ui.LineJoin
	alpha     float64
	depth     int
}

// ctx is the CanvasRenderingContext2D work-alike.
type ctx struct {
	st    state
	stack []state
	path  path2D
	root  node
	conts []*node
}

func newCtx(base mat) *ctx {
	c := &ctx{st: state{m: base, fill: rgba{0, 0, 0, 1}, stroke: rgba{0, 0, 0, 1}, lineWidth: 1, cap: ui.ButtCap, join: ui.MiterJoin, alpha: 1}}
	c.path.useM = true
	c.conts = []*node{&c.root}
	return c
}

func (c *ctx) save() {
	s := c.st
	s.depth = len(c.conts)
	c.stack = append(c.stack, s)
}

func (c *ctx) restore() {
	if len(c.stack) == 0 {
		return
	}
	s := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	c.conts = c.conts[:s.depth]
	c.st = s
}

func (c *ctx) setFill(s any)   { c.st.fill = s }
func (c *ctx) setStroke(s any) { c.st.stroke = s }
func (c *ctx) setLineWidth(w float64) {
	if w > 0 && finite(w) {
		c.st.lineWidth = w
	}
}
func (c *ctx) setAlpha(a float64) {
	if a >= 0 && a <= 1 {
		c.st.alpha = a
	}
}
func (c *ctx) roundCaps() { c.st.cap = ui.RoundCap }
func (c *ctx) roundJoin() { c.st.join = ui.RoundJoin }

func (c *ctx) transform(a, b, cc, d, e, f float64) {
	if !finite(a, b, cc, d, e, f) {
		return
	}
	c.st.m = c.st.m.mul(mat{a, b, cc, d, e, f})
}
func (c *ctx) translate(x, y float64) { c.transform(1, 0, 0, 1, x, y) }
func (c *ctx) scale(x, y float64)     { c.transform(x, 0, 0, y, 0, 0) }
func (c *ctx) rotate(a float64) {
	if !finite(a) {
		return
	}
	s, co := math.Sin(a), math.Cos(a)
	c.transform(co, s, -s, co, 0, 0)
}

func (c *ctx) beginPath()          { c.path.reset() }
func (c *ctx) moveTo(x, y float64) { c.path.m = c.st.m; c.path.moveTo(x, y) }
func (c *ctx) lineTo(x, y float64) { c.path.m = c.st.m; c.path.lineTo(x, y) }
func (c *ctx) quadraticCurveTo(a, b, x, y float64) {
	c.path.m = c.st.m
	c.path.quadraticCurveTo(a, b, x, y)
}
func (c *ctx) bezierCurveTo(a, b, cc, d, x, y float64) {
	c.path.m = c.st.m
	c.path.bezierCurveTo(a, b, cc, d, x, y)
}
func (c *ctx) closePath() { c.path.closePath() }
func (c *ctx) rect(x, y, w, h float64) {
	c.path.m = c.st.m
	c.path.rect(x, y, w, h)
}
func (c *ctx) arc(x, y, r, a0, a1 float64, ccw bool) {
	c.path.m = c.st.m
	c.path.arc(x, y, r, a0, a1, ccw)
}
func (c *ctx) ellipse(x, y, rx, ry, rot, a0, a1 float64, ccw bool) {
	c.path.m = c.st.m
	c.path.ellipse(x, y, rx, ry, rot, a0, a1, ccw)
}

func (c *ctx) add(n *node) {
	top := c.conts[len(c.conts)-1]
	top.kids = append(top.kids, n)
}

func (c *ctx) current() []seg { return append([]seg(nil), c.path.segs...) }

// fill fills the current path.
func (c *ctx) fill() { c.fillSegs(c.current()) }

// fillPath fills a Path2D.
func (c *ctx) fillPath(p *path2D) { c.fillSegs(transformed(p.segs, c.st.m)) }

func (c *ctx) fillSegs(segs []seg) {
	if len(segs) == 0 {
		return
	}
	c.add(&node{kind: nodeFill, segs: segs, m: c.st.m, style: c.st.fill, alpha: c.st.alpha})
}

func (c *ctx) stroke()              { c.strokeSegs(c.current()) }
func (c *ctx) strokePath(p *path2D) { c.strokeSegs(transformed(p.segs, c.st.m)) }

func (c *ctx) strokeSegs(segs []seg) {
	if len(segs) == 0 {
		return
	}
	c.add(&node{kind: nodeStroke, segs: segs, m: c.st.m, style: c.st.stroke, width: c.st.lineWidth, cap: c.st.cap, join: c.st.join, alpha: c.st.alpha})
}

func (c *ctx) fillRect(x, y, w, h float64) {
	if w == 0 || h == 0 || !finite(x, y, w, h) {
		return
	}
	var p path2D
	p.rect(x, y, w, h)
	c.fillPath(&p)
}

func (c *ctx) clip() { c.clipSegs(c.current()) }

func (c *ctx) clipPath(p *path2D) { c.clipSegs(transformed(p.segs, c.st.m)) }

func (c *ctx) clipSegs(segs []seg) {
	n := &node{kind: nodeClip, segs: segs, m: c.st.m}
	c.add(n)
	c.conts = append(c.conts, n)
}

func (c *ctx) createLinearGradient(x0, y0, x1, y1 float64) *gradient {
	return &gradient{x0: x0, y0: y0, x1: x1, y1: y1}
}

func (c *ctx) createRadialGradient(x0, y0, r0, x1, y1, r1 float64) *gradient {
	return &gradient{radial: true, x0: x0, y0: y0, r0: r0, x1: x1, y1: y1, r1: r1}
}

// Replay.

func uiPath(segs []seg, m *mat) *ui.Path {
	var p ui.Path
	f := func(i int, s seg) (float32, float32) {
		x, y := s.p[2*i], s.p[2*i+1]
		if m != nil {
			x, y = m.apply(x, y)
		}
		return float32(x), float32(y)
	}
	for _, s := range segs {
		switch s.k {
		case segMove:
			x, y := f(0, s)
			p.MoveTo(x, y)
		case segLine:
			x, y := f(0, s)
			p.LineTo(x, y)
		case segQuad:
			a, b := f(0, s)
			x, y := f(1, s)
			p.QuadTo(a, b, x, y)
		case segCubic:
			a, b := f(0, s)
			cc, d := f(1, s)
			x, y := f(2, s)
			p.CubeTo(a, b, cc, d, x, y)
		case segClose:
			p.Close()
		}
	}
	return &p
}

// draw replays everything recorded onto the painter.
func (c *ctx) draw(p *ui.Painter) { replay(p, c.root.kids) }

func replay(p *ui.Painter, nodes []*node) {
	for _, n := range nodes {
		switch n.kind {
		case nodeClip:
			if len(n.kids) == 0 {
				continue
			}
			kids := n.kids
			p.ClipPath(uiPath(n.segs, nil), func() { replay(p, kids) })
		case nodeFill:
			fillNode(p, n)
		case nodeStroke:
			strokeNode(p, n)
		}
	}
}

func fillNode(p *ui.Painter, n *node) {
	switch s := n.style.(type) {
	case rgba:
		p.FillPath(uiPath(n.segs, nil), s.ui(n.alpha))
	case *gradient:
		if len(s.stops) == 0 || n.m.det() == 0 {
			return
		}
		if s.radial {
			fillRadial(p, n, s)
		} else {
			fillLinear(p, n, s)
		}
	}
}

func strokeNode(p *ui.Painter, n *node) {
	col, ok := n.style.(rgba)
	if !ok {
		return
	}
	hi, lo := n.m.singular()
	if hi == 0 {
		return
	}
	if lo/hi > 0.999 {
		path := uiPath(n.segs, nil).Cap(n.cap).Join(n.join)
		if n.join == ui.MiterJoin {
			path.MiterLimit(10)
		}
		p.StrokePath(path, float32(n.width*math.Sqrt(math.Abs(n.m.det()))), col.ui(n.alpha))
		return
	}
	if lo == 0 {
		return
	}
	// Back to the stroke's own space, scaled up uniformly by the largest
	// stretch; what is left of the transform then only shrinks.
	inv := n.m.inv()
	pre := mat{hi, 0, 0, hi, 0, 0}.mul(inv)
	path := uiPath(n.segs, &pre).Cap(n.cap).Join(n.join)
	if n.join == ui.MiterJoin {
		path.MiterLimit(10)
	}
	rest := mat{n.m.a / hi, n.m.b / hi, n.m.c / hi, n.m.d / hi, n.m.e, n.m.f}
	p.Transform(uiTransform(rest), 0, 0, func() {
		p.StrokePath(path, float32(n.width*hi), col.ui(n.alpha))
	})
}

// uiTransform builds m from MyGo's parts: a skew, a scale, a rotation and a move.
func uiTransform(m mat) ui.Transform {
	theta := math.Atan2(m.b, m.a)
	sx := math.Hypot(m.a, m.b)
	ct, st := math.Cos(theta), math.Sin(theta)
	u12 := ct*m.c + st*m.d
	sy := -st*m.c + ct*m.d
	phi := math.Atan(u12 / sx)
	deg := 180 / math.Pi
	return ui.Skewing(float32(phi*deg), 0).
		Then(ui.Scaling(float32(sx), float32(sy))).
		Then(ui.Rotation(float32(theta * deg))).
		Then(ui.Translation(float32(m.e), float32(m.f)))
}

// uiStops turns canvas stops (interpolated without premultiplying) into
// MyGo's, which premultiply: segments whose color and alpha both change
// get extra stops in between.
func uiStops(g *gradient, at func(t float64) float64, alpha float64) []ui.GradientStop {
	var out []ui.GradientStop
	add := func(t float64, c rgba) {
		out = append(out, ui.GradientStop{Color: c.ui(alpha), At: float32(at(t))})
	}
	s := g.stops
	sub := 1
	if len(s) <= 4 {
		sub = 8
	}
	add(s[0].at, s[0].c)
	for i := 1; i < len(s); i++ {
		a, b := s[i-1], s[i]
		if a.c.a != b.c.a && (a.c.r != b.c.r || a.c.g != b.c.g || a.c.b != b.c.b) && b.at > a.at {
			for k := 1; k < sub; k++ {
				t := a.at + (b.at-a.at)*float64(k)/float64(sub)
				add(t, g.colorAt(t))
			}
		}
		add(b.at, b.c)
	}
	return out
}

func fillLinear(p *ui.Painter, n *node, g *gradient) {
	vx, vy := g.x1-g.x0, g.y1-g.y0
	v2 := vx*vx + vy*vy
	if v2 == 0 {
		return
	}
	path := uiPath(n.segs, nil)
	b := path.Bounds()
	if b.W <= 0 && b.H <= 0 {
		return
	}
	// t(q) = nv·(q − M·p0) in window space.
	m := n.m
	inv := m.inv()
	nx, ny := (inv.a*vx+inv.b*vy)/v2, (inv.c*vx+inv.d*vy)/v2
	nl := math.Hypot(nx, ny)
	if nl == 0 {
		return
	}
	dx, dy := nx/nl, ny/nl
	px, py := m.apply(g.x0, g.y0)
	angle := math.Atan2(dx, -dy)
	W, H := float64(b.W), float64(b.H)
	lg := math.Abs(W*math.Sin(angle)) + math.Abs(H*math.Cos(angle))
	if lg <= 0 {
		return
	}
	cx, cy := float64(b.X)+W/2, float64(b.Y)+H/2
	off := (cx-px)*dx + (cy-py)*dy
	at := func(t float64) float64 { return (t/nl-off)/lg + 0.5 }
	p.FillPathGradient(path, ui.LinearGradient{Angle: float32(angle * 180 / math.Pi), Stops: uiStops(g, at, n.alpha)})
}

var radialShader = shader.MustCompile(`package main

var Origin vec2
var Inv0 vec3
var Inv1 vec3
var Circle0 vec3
var Circle1 vec3
var Colors [4]vec4
var Offsets [4]float
var Count int

func colorAt(t float) vec4 {
	if t <= Offsets[0] {
		return Colors[0]
	}
	res := Colors[0]
	for i := 1; i < 4; i++ {
		if i < Count {
			if t >= Offsets[i] {
				res = Colors[i]
			} else if t > Offsets[i-1] {
				u := (t - Offsets[i-1]) / (Offsets[i] - Offsets[i-1])
				return mix(Colors[i-1], Colors[i], u)
			}
		}
	}
	return res
}

func Fragment(dst vec4, pos vec2) vec4 {
	q := vec3(pos+Origin, 1)
	p := vec2(dot(Inv0, q), dot(Inv1, q))
	cd := Circle1.xy - Circle0.xy
	dr := Circle1.z - Circle0.z
	pd := p - Circle0.xy
	a := dot(cd, cd) - dr*dr
	b := dot(pd, cd) + Circle0.z*dr
	c := dot(pd, pd) - Circle0.z*Circle0.z
	t := 0.0
	if abs(a) < 1e-12 {
		if b == 0 {
			return vec4(0)
		}
		t = c / (2 * b)
		if Circle0.z+t*dr < 0 {
			return vec4(0)
		}
	} else {
		disc := b*b - a*c
		if disc < 0 {
			return vec4(0)
		}
		s := sqrt(disc)
		hi := max((b+s)/a, (b-s)/a)
		lo := min((b+s)/a, (b-s)/a)
		if Circle0.z+hi*dr >= 0 {
			t = hi
		} else if Circle0.z+lo*dr >= 0 {
			t = lo
		} else {
			return vec4(0)
		}
	}
	col := colorAt(t)
	return vec4(col.rgb*col.a, col.a)
}
`)

func fillRadial(p *ui.Painter, n *node, g *gradient) {
	if g.x0 == g.x1 && g.y0 == g.y1 && g.r0 == g.r1 {
		return
	}
	path := uiPath(n.segs, nil)
	b := path.Bounds()
	if b.W <= 0 || b.H <= 0 {
		return
	}
	inv := n.m.inv()
	cnt := len(g.stops)
	if cnt > 4 {
		cnt = 4
	}
	colors := make([]float32, 16)
	offsets := make([]float32, 4)
	for i := 0; i < cnt; i++ {
		s := g.stops[i]
		col := s.c.ui(n.alpha)
		a := clamp(s.c.a, 0, 1) * clamp(n.alpha, 0, 1)
		colors[4*i], colors[4*i+1], colors[4*i+2], colors[4*i+3] = float32(col.R)/255, float32(col.G)/255, float32(col.B)/255, float32(a)
		offsets[i] = float32(s.at)
	}
	radialShader.FillPath(p, path, shader.Options{Uniforms: map[string]any{
		"Origin":  []float32{b.X, b.Y},
		"Inv0":    []float32{float32(inv.a), float32(inv.c), float32(inv.e)},
		"Inv1":    []float32{float32(inv.b), float32(inv.d), float32(inv.f)},
		"Circle0": []float32{float32(g.x0), float32(g.y0), float32(g.r0)},
		"Circle1": []float32{float32(g.x1), float32(g.y1), float32(g.r1)},
		"Colors":  colors,
		"Offsets": offsets,
		"Count":   cnt,
	}})
}
