package reef

// Fish, ported from engine/fish.js.
//
// Looks are drawn facing right in units of body length: the snout is at
// x = .5 and the base of the tail at x = −.5, and y grows downward. Body
// stations are [x, top, bottom]. A fin lists its outer edge from front to
// back; its base runs along the body under the first and last points. The
// tail fin, pectoral and pelvic fins are drawn from their own base point.

import (
	"math"
	"sync"
)

type fin struct {
	edge      []pt
	at        pt
	angle     float64
	line      string
	lineWidth float64
	color     string
	tip       string
	flow      bool
	ripple    bool
}

type fishPose struct {
	tail, beat, fin, breath, gape, lookX, lookY, paddle float64
	colors                                              map[string]string
}

type fishLook struct {
	body     [][3]float64
	eye      [3]float64
	mouth    pt
	gill     float64
	dorsal   *fin
	anal     *fin
	caudal   *fin
	pectoral *fin
	pelvic   *fin
	pattern  func(c *ctx, col shadeFn, pose *fishPose)
}

var fishLooks = map[string]*fishLook{
	// Lyretail anthias (sea goldfish), female: the resident.
	"resident": {
		body: [][3]float64{{.5, .005, .01}, {.45, -.07, .06}, {.36, -.14, .12}, {.2, -.18, .155}, {0, -.18, .155}, {-.2, -.145, .125}, {-.36, -.095, .08}, {-.5, -.065, .055}},
		eye:  [3]float64{.35, -.035, .062}, mouth: pt{.5, .006}, gill: .21,
		dorsal:   &fin{edge: []pt{{.25, -.16}, {.21, -.25}, {.14, -.27}, {.06, -.265}, {-.02, -.26}, {-.1, -.255}, {-.19, -.26}, {-.28, -.25}, {-.35, -.2}, {-.41, -.1}}},
		anal:     &fin{edge: []pt{{-.07, .14}, {-.12, .23}, {-.2, .29}, {-.28, .25}, {-.35, .16}, {-.4, .085}}},
		caudal:   &fin{edge: []pt{{0, -.065}, {-.1, -.15}, {-.26, -.25}, {-.52, -.37}, {-.33, -.17}, {-.24, -.03}, {-.24, .03}, {-.33, .17}, {-.52, .37}, {-.26, .25}, {-.1, .15}, {0, .065}}},
		pectoral: &fin{at: pt{.2, .045}, edge: []pt{{0, -.025}, {-.08, -.05}, {-.17, -.035}, {-.2, .005}, {-.15, .04}, {-.07, .045}, {0, .025}}, angle: .25},
		pelvic:   &fin{at: pt{.15, .14}, edge: []pt{{0, -.01}, {-.05, .08}, {-.17, .17}, {-.1, .06}, {-.03, .01}}},
		pattern: func(c *ctx, col shadeFn, pose *fishPose) {
			// A violet line runs from under the eye down to the pectoral fin.
			c.setStroke(col("#a98ad8", .9))
			c.setLineWidth(.028)
			c.roundCaps()
			c.beginPath()
			c.moveTo(.33, .03)
			c.quadraticCurveTo(.26, .06, .18, .085)
			c.stroke()
		},
	},
	// Ocellaris clownfish.
	"clown": {
		body: [][3]float64{{.5, .015, .025}, {.47, -.07, .09}, {.38, -.19, .18}, {.22, -.255, .225}, {0, -.26, .23}, {-.2, -.2, .18}, {-.36, -.13, .115}, {-.5, -.1, .085}},
		eye:  [3]float64{.345, -.05, .062}, mouth: pt{.5, .02}, gill: .27,
		dorsal:   &fin{edge: []pt{{.23, -.24}, {.19, -.31}, {.13, -.335}, {.07, -.34}, {.02, -.325}, {-.03, -.29}, {-.07, -.35}, {-.15, -.39}, {-.24, -.37}, {-.31, -.3}, {-.36, -.16}}, line: "#1d1512"},
		anal:     &fin{edge: []pt{{-.1, .21}, {-.14, .3}, {-.21, .34}, {-.29, .3}, {-.35, .2}, {-.37, .11}}, line: "#1d1512"},
		caudal:   &fin{edge: []pt{{0, -.09}, {-.08, -.19}, {-.19, -.24}, {-.29, -.19}, {-.33, -.08}, {-.34, 0}, {-.33, .08}, {-.29, .19}, {-.19, .24}, {-.08, .19}, {0, .09}}, line: "#1d1512"},
		pectoral: &fin{at: pt{.18, .05}, edge: []pt{{0, -.025}, {-.07, -.055}, {-.15, -.045}, {-.19, 0}, {-.16, .045}, {-.08, .055}, {0, .025}}, angle: .2},
		pelvic:   &fin{at: pt{.13, .2}, edge: []pt{{0, -.01}, {-.05, .06}, {-.12, .11}, {-.09, .05}, {-.03, 0}}, line: "#1d1512"},
		pattern: func(c *ctx, col shadeFn, pose *fishPose) {
			// Three white bands, each edged with a thin black line.
			bands := [][]pt{
				{{.29, -.3}, {.245, 0}, {.28, .28}, {.2, .28}, {.17, 0}, {.2, -.3}},
				{{.03, -.31}, {.11, -.03}, {.03, .27}, {-.08, .27}, {-.02, -.03}, {-.08, -.31}},
				{{-.39, -.16}, {-.39, .16}, {-.46, .16}, {-.46, -.16}},
			}
			for _, band := range bands {
				p := curvePath(band, true)
				c.setFill(col("#fbf6ec"))
				c.fillPath(p)
				c.setStroke(col("#1d1512", .9))
				c.setLineWidth(.016)
				c.strokePath(p)
			}
		},
	},
	// Palette surgeonfish (blue tang).
	"tang": {
		body: [][3]float64{{.5, .03, .035}, {.46, -.06, .1}, {.37, -.18, .2}, {.2, -.255, .255}, {0, -.265, .265}, {-.2, -.225, .22}, {-.37, -.13, .12}, {-.5, -.06, .055}},
		eye:  [3]float64{.35, -.07, .05}, mouth: pt{.5, .033}, gill: .25,
		dorsal:   &fin{edge: []pt{{.31, -.2}, {.24, -.28}, {.12, -.33}, {0, -.345}, {-.12, -.34}, {-.24, -.3}, {-.34, -.22}, {-.42, -.1}}, line: "#10141c", lineWidth: 2},
		anal:     &fin{edge: []pt{{.06, .25}, {-.05, .32}, {-.17, .335}, {-.28, .29}, {-.37, .2}, {-.43, .085}}, line: "#10141c", lineWidth: 2},
		caudal:   &fin{edge: []pt{{0, -.055}, {-.08, -.13}, {-.2, -.24}, {-.34, -.33}, {-.27, -.16}, {-.22, 0}, {-.27, .16}, {-.34, .33}, {-.2, .24}, {-.08, .13}, {0, .055}}, color: "#f2c64a", line: "#10141c", lineWidth: 2.4},
		pectoral: &fin{at: pt{.2, .05}, edge: []pt{{0, -.03}, {-.09, -.06}, {-.2, -.05}, {-.24, 0}, {-.19, .045}, {-.09, .05}, {0, .03}}, angle: .15, tip: "#f2c64a"},
		pelvic:   &fin{at: pt{.16, .22}, edge: []pt{{0, -.01}, {-.04, .05}, {-.1, .08}, {-.07, .03}, {-.02, 0}}},
		pattern: func(c *ctx, col shadeFn, pose *fishPose) {
			// The black "palette" runs from the eye along the back to the tail and
			// loops down around an oval of blue.
			c.setFill(col("#10141c"))
			c.fillPath(curvePath([]pt{{.4, -.06}, {.3, -.19}, {.12, -.255}, {-.1, -.25}, {-.3, -.19}, {-.46, -.1}, {-.56, -.03}, {-.56, .05}, {-.44, .07}, {-.3, .1}, {-.14, .15}, {0, .14}, {.1, .06}, {.15, -.03}, {.24, -.05}, {.33, -.02}}, true))
			c.setFill(col("#2f62c2"))
			oval(c, -.14, -.065, .19, .085, -.08)
			c.setFill(col("#3d6fcc", .9))
			oval(c, -.5, .0, .05, .03, 0)
		},
	},
	// Queen angelfish.
	"angel": {
		body: [][3]float64{{.5, .04, .06}, {.46, -.06, .14}, {.38, -.2, .24}, {.24, -.31, .31}, {.05, -.34, .33}, {-.15, -.31, .3}, {-.33, -.2, .2}, {-.5, -.085, .085}},
		eye:  [3]float64{.37, -.04, .05}, mouth: pt{.5, .05}, gill: .29,
		dorsal:   &fin{edge: []pt{{.27, -.3}, {.18, -.4}, {.05, -.45}, {-.1, -.47}, {-.25, -.47}, {-.4, -.45}, {-.55, -.43}, {-.72, -.41}, {-.86, -.38}, {-.62, -.3}, {-.5, -.18}, {-.47, -.09}}, line: "#4fb2f0", lineWidth: 1.6, flow: true},
		anal:     &fin{edge: []pt{{.03, .32}, {-.08, .42}, {-.22, .46}, {-.38, .46}, {-.55, .44}, {-.72, .42}, {-.84, .39}, {-.6, .3}, {-.5, .18}, {-.47, .09}}, line: "#4fb2f0", lineWidth: 1.6, flow: true},
		caudal:   &fin{edge: []pt{{0, -.085}, {-.06, -.18}, {-.18, -.24}, {-.27, -.2}, {-.29, 0}, {-.27, .2}, {-.18, .24}, {-.06, .18}, {0, .085}}, color: "#f4c530"},
		pectoral: &fin{at: pt{.22, .06}, edge: []pt{{0, -.03}, {-.08, -.06}, {-.18, -.05}, {-.22, 0}, {-.18, .045}, {-.08, .055}, {0, .03}}, color: "#f4c530", angle: .2},
		pelvic:   &fin{at: pt{.2, .27}, edge: []pt{{0, -.01}, {-.06, .1}, {-.2, .25}, {-.12, .08}, {-.04, 0}}, color: "#f4c530"},
		pattern: func(c *ctx, col shadeFn, pose *fishPose) {
			// Scales with yellow centers, a blue face, and the crown on the forehead.
			c.setFill(col("#e9c94a", .42))
			row := 0
			for y := -.32; y < .34; y += .045 {
				for x := -.48 + float64(row&1)*.022; x < .32; x += .045 {
					dot(c, x+math.Sin(x*91+y*37)*.006, y+math.Cos(x*53-y*71)*.006, .011)
				}
				row++
			}
			face := c.createRadialGradient(.42, .02, 0, .42, .02, .22)
			face.addColorStop(0, col("#3f74b8", .75))
			face.addColorStop(1, col("#3f74b8", 0))
			c.setFill(face)
			c.fillRect(.15, -.3, .4, .6)
			c.setStroke(col("#4fb2f0", .9))
			c.setLineWidth(.014)
			c.beginPath()
			c.arc(.31, .02, .14, -1.2, 1.3, false)
			c.stroke()
			c.setFill(col("#4fb2f0"))
			dot(c, .25, -.235, .055)
			c.setFill(col("#15245a"))
			dot(c, .25, -.235, .042)
			c.setFill(col("#4fb2f0"))
			dot(c, .25, -.235, .012)
		},
	},
	// Valentini's sharpnose puffer.
	"puffer": {
		body: [][3]float64{{.5, .025, .04}, {.45, -.06, .1}, {.33, -.18, .2}, {.15, -.25, .25}, {-.05, -.245, .25}, {-.25, -.17, .19}, {-.4, -.1, .11}, {-.5, -.075, .075}},
		eye:  [3]float64{.28, -.1, .066}, mouth: pt{.5, .033}, gill: .17,
		dorsal:   &fin{edge: []pt{{-.22, -.18}, {-.25, -.27}, {-.31, -.29}, {-.36, -.21}, {-.37, -.13}}, ripple: true},
		anal:     &fin{edge: []pt{{-.24, .18}, {-.27, .27}, {-.33, .28}, {-.37, .2}, {-.38, .13}}, ripple: true},
		caudal:   &fin{edge: []pt{{0, -.075}, {-.07, -.15}, {-.18, -.19}, {-.26, -.15}, {-.29, 0}, {-.26, .15}, {-.18, .19}, {-.07, .15}, {0, .075}}, color: "#f1d27e", line: "#d98a3a"},
		pectoral: &fin{at: pt{.15, .0}, edge: []pt{{0, -.025}, {-.06, -.045}, {-.12, -.03}, {-.13, .01}, {-.09, .035}, {-.04, .035}, {0, .02}}, angle: .1},
		pattern: func(c *ctx, col shadeFn, pose *fishPose) {
			// Four brown saddles, orange spots low on the body, and fine blue-green
			// lines around the eye.
			c.setFill(col("#3b2a21"))
			for _, s := range [][]pt{{{.36, -.2}, {.2, -.27}, {.18, -.1}, {.3, -.05}}, {{.1, -.3}, {-.1, -.3}, {-.12, .1}, {-.02, .2}, {.08, .08}}, {{-.17, -.25}, {-.31, -.2}, {-.3, .03}, {-.2, .08}}, {{-.39, -.13}, {-.5, -.1}, {-.5, .02}, {-.41, .0}}} {
				c.fillPath(curvePath(s, true))
			}
			c.setFill(col("#d27a33"))
			for _, p := range []pt{{.38, .1}, {.3, .15}, {.22, .19}, {.16, .12}, {.05, .2}, {-.08, .22}, {-.2, .16}, {-.28, .1}, {-.36, .07}, {.42, .03}, {.12, .22}} {
				dot(c, p[0], p[1], .022)
			}
			c.setStroke(col("#47b3a6", .9))
			c.setLineWidth(.012)
			for i := 0; i < 7; i++ {
				a := -.6 + float64(i)*.52
				ex, ey := .28, -.1
				c.beginPath()
				c.moveTo(ex+math.Cos(a)*.08, ey+math.Sin(a)*.08)
				c.lineTo(ex+math.Cos(a)*.12, ey+math.Sin(a)*.12)
				c.stroke()
			}
		},
	},
}

// A generic fish body for species without a look of their own yet.
var plainFish = &fishLook{
	body: [][3]float64{{.5, .01, .02}, {.45, -.08, .08}, {.33, -.17, .16}, {.15, -.21, .2}, {-.05, -.2, .19}, {-.25, -.15, .14}, {-.4, -.09, .08}, {-.5, -.065, .06}},
	eye:  [3]float64{.33, -.05, .055}, mouth: pt{.5, .015}, gill: .21,
	dorsal:   &fin{edge: []pt{{.18, -.2}, {.08, -.28}, {-.08, -.29}, {-.22, -.25}, {-.33, -.13}}},
	anal:     &fin{edge: []pt{{-.12, .18}, {-.18, .26}, {-.28, .24}, {-.35, .1}}},
	caudal:   &fin{edge: []pt{{0, -.06}, {-.1, -.15}, {-.3, -.26}, {-.25, 0}, {-.3, .26}, {-.1, .15}, {0, .06}}},
	pectoral: &fin{at: pt{.18, .04}, edge: []pt{{0, -.02}, {-.08, -.045}, {-.16, -.03}, {-.17, .01}, {-.08, .035}, {0, .02}}, angle: .2},
}

func lookOf(id string) *fishLook {
	if l, ok := fishLooks[id]; ok {
		return l
	}
	return plainFish
}

type builtFin struct {
	*fin
	path, rays, rim *path2D
	from, to        pt
}

type fishShape struct {
	body                                   *path2D
	top, bottom                            float64
	dorsal, anal, caudal, pectoral, pelvic *builtFin
	scales, gill, line, rim                *path2D
	bounds                                 struct{ x0, x1, y0, y1 float64 }
}

var (
	builtMu sync.Mutex
	built   = map[*fishLook]*fishShape{}
)

// shape turns a look into paths, once.
func shapeOf(look *fishLook) *fishShape {
	builtMu.Lock()
	defer builtMu.Unlock()
	if s, ok := built[look]; ok {
		return s
	}
	top := func(x float64) float64 { return station(look.body, x, 1) }
	bottom := func(x float64) float64 { return station(look.body, x, 2) }
	outline := []pt{{look.body[0][0], (look.body[0][1] + look.body[0][2]) / 2}}
	for _, b := range look.body[1:] {
		outline = append(outline, pt{b[0], b[1]})
	}
	for i := len(look.body) - 1; i >= 1; i-- {
		outline = append(outline, pt{look.body[i][0], look.body[i][2]})
	}
	s := &fishShape{body: curvePath(outline, true), top: math.Inf(1), bottom: math.Inf(-1)}
	for _, b := range look.body {
		s.top = math.Min(s.top, b[1])
		s.bottom = math.Max(s.bottom, b[2])
	}
	if look.dorsal != nil {
		s.dorsal = edgeFin(look.dorsal, func(x float64) float64 { return top(x) + .025 })
	}
	if look.anal != nil {
		s.anal = edgeFin(look.anal, func(x float64) float64 { return bottom(x) - .025 })
	}
	s.caudal = baseFin(look.caudal, pt{0, 0}, true)
	s.pectoral = baseFin(look.pectoral, pt{0, 0}, false)
	if look.pelvic != nil {
		s.pelvic = baseFin(look.pelvic, pt{0, 0}, false)
	}
	// Scales overlap from head to tail, so each shows a rim toward the tail.
	s.scales = newPath()
	row := 0
	for y := -.34; y < .36; y += .05 {
		for x := -.47 + float64(row&1)*.025; x < look.gill-.02; x += .05 {
			s.scales.moveTo(x-.03*math.Cos(1), y-.03*math.Sin(1))
			s.scales.arc(x, y, .03, math.Pi+1, math.Pi-1, true)
		}
		row++
	}
	ex := look.eye[0]
	s.gill = newPath()
	s.gill.moveTo(look.gill+.02, top(look.gill)*.7)
	s.gill.quadraticCurveTo(look.gill-.05, 0, look.gill+.01, bottom(look.gill)*.75)
	s.line = newPath()
	s.line.moveTo(look.gill-.02, top(look.gill)*.5)
	s.line.quadraticCurveTo(0, top(0)*.62, -.46, -.01)
	var rim []pt
	for _, b := range look.body[1:] {
		rim = append(rim, pt{b[0], b[1] + .006})
	}
	s.rim = curvePath(rim, false)
	// Bounds of everything, for fitting portraits.
	pts := append([]pt(nil), outline...)
	if look.dorsal != nil {
		pts = append(pts, look.dorsal.edge...)
	}
	if look.anal != nil {
		pts = append(pts, look.anal.edge...)
	}
	for _, p := range look.caudal.edge {
		pts = append(pts, pt{p[0] - .5, p[1]})
	}
	if look.pelvic != nil {
		for _, p := range look.pelvic.edge {
			pts = append(pts, pt{p[0] + look.pelvic.at[0], p[1] + look.pelvic.at[1]})
		}
	}
	b := &s.bounds
	b.x0, b.x1, b.y0, b.y1 = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		b.x0 = math.Min(b.x0, p[0])
		b.x1 = math.Max(b.x1, p[0])
		b.y0 = math.Min(b.y0, p[1])
		b.y1 = math.Max(b.y1, p[1])
	}
	b.x1 = math.Max(b.x1, ex)
	built[look] = s
	return s
}

func station(body [][3]float64, x float64, k int) float64 {
	for i := 1; i < len(body); i++ {
		if x >= body[i][0] {
			a, b := body[i-1], body[i]
			t := (x - b[0]) / or(a[0]-b[0], 1)
			return lerp(b[k], a[k], t)
		}
	}
	return body[len(body)-1][k]
}

// edgeFin builds a fin whose base follows the body edge.
func edgeFin(f *fin, base func(float64) float64) *builtFin {
	e := f.edge
	n := len(e)
	x0, x1 := e[0][0], e[n-1][0]
	path := curvePath(e, false)
	for i := 0; i <= 8; i++ {
		x := lerp(x1, x0, float64(i)/8)
		path.lineTo(x, base(x))
	}
	path.closePath()
	rays := newPath()
	for i := 1; i < n-1; i++ {
		x := lerp(x0, x1, float64(i)/float64(n-1))
		rays.moveTo(x, base(x))
		rays.lineTo(e[i][0], e[i][1])
	}
	far := e[0]
	for _, p := range e[1:] {
		if math.Abs(p[1]-base(p[0])) > math.Abs(far[1]-base(far[0])) {
			far = p
		}
	}
	return &builtFin{fin: f, path: path, rays: rays, rim: curvePath(e, false), from: pt{far[0], base(far[0])}, to: far}
}

// baseFin builds a fin that fans out from one base point.
func baseFin(f *fin, b pt, tail bool) *builtFin {
	e := f.edge
	path := curvePath(e, true)
	rays := newPath()
	for i := 1; i < len(e)-1; i++ {
		from := b
		if tail {
			from = pt{0, lerp(e[0][1], e[len(e)-1][1], float64(i)/float64(len(e)-1))}
		}
		rays.moveTo(from[0], from[1])
		rays.lineTo(e[i][0], e[i][1])
	}
	far := e[0]
	for _, p := range e[1:] {
		if math.Hypot(p[0]-b[0], p[1]-b[1]) > math.Hypot(far[0]-b[0], far[1]-b[1]) {
			far = p
		}
	}
	rim := e
	if tail {
		rim = e[1 : len(e)-1]
	}
	to := pt{far[0], far[1]}
	if tail {
		to[1] = 0
	}
	return &builtFin{fin: f, path: path, rays: rays, rim: curvePath(rim, false), from: b, to: to}
}

// paintFin draws a fin: a see-through membrane, its rays and an optional rim.
func paintFin(c *ctx, f *builtFin, color string, col shadeFn, alpha, hair float64) {
	hue := f.color
	if hue == "" {
		hue = color
	}
	g := c.createLinearGradient(f.from[0], f.from[1], f.to[0], f.to[1])
	g.addColorStop(0, col(hue, alpha))
	g.addColorStop(.6, col(hue, alpha*.82, .06))
	g.addColorStop(1, col(hue, alpha*.5, .14))
	c.setFill(g)
	c.fillPath(f.path)
	c.setStroke(col(hue, alpha*.55, -.3))
	c.setLineWidth(hair * .6)
	c.strokePath(f.rays)
	if f.tip != "" {
		c.save()
		c.clipPath(f.path)
		c.setFill(col(f.tip, alpha+.1))
		c.fillRect(-.3, -.2, .16, .4)
		c.restore()
	}
	if f.line != "" {
		c.setStroke(col(f.line, .85))
		c.setLineWidth(hair * or(f.lineWidth, 1.2))
		c.strokePath(f.rim)
	}
}

// drawFish paints a fish in unit space for the given pose.
func drawFish(c *ctx, look *fishLook, s *fishShape, pose *fishPose, haze, L float64) {
	col, C, hair := shade(haze), pose.colors, 1/L
	// The tail sweeps toward and away from us, so seen from the side it
	// narrows and twists rather than swinging up and down.
	st := math.Sin(pose.tail)
	sweep := 1 - .55*pose.beat*st*st
	c.save()
	c.translate(-.5, 0)
	c.transform(sweep, math.Sin(pose.tail)*.12*pose.beat, 0, 1, 0, 0)
	paintFin(c, s.caudal, C["fin"], col, .72+.22*(1-sweep), hair)
	c.restore()
	for _, fs := range []struct {
		f    *builtFin
		sign float64
	}{{s.dorsal, -1}, {s.anal, 1}} {
		f := fs.f
		if f == nil {
			continue
		}
		c.save()
		if f.flow {
			c.translate(-.4, 0)
			c.rotate(math.Sin(pose.fin*.5) * .025 * fs.sign)
			c.translate(.4, 0)
		}
		if f.ripple {
			x0 := f.edge[0][0]
			c.translate(x0, 0)
			c.transform(1, 0, math.Sin(pose.fin*1.6)*.35, 1, 0, 0)
			c.translate(-x0, 0)
		}
		paintFin(c, f, C["fin"], col, .86, hair)
		c.restore()
	}
	if s.pelvic != nil {
		c.save()
		c.translate(look.pelvic.at[0], look.pelvic.at[1])
		c.rotate(-.1 + math.Sin(pose.fin*.7)*.08)
		c.translate(.05, 0)
		paintFin(c, s.pelvic, C["fin"], col, .55, hair)
		c.restore()
	}
	// Body: lighter below than above, then the species' markings, scales, and
	// the light from the surface.
	g := c.createLinearGradient(0, s.top, 0, s.bottom)
	g.addColorStop(0, col(C["back"]))
	g.addColorStop(.48, col(C["flank"]))
	g.addColorStop(1, col(C["belly"]))
	c.setFill(g)
	c.fillPath(s.body)
	c.save()
	c.clipPath(s.body)
	if look.pattern != nil {
		look.pattern(c, col, pose)
	}
	if L > 24 {
		c.setStroke(col("#0e1c22", .07))
		c.setLineWidth(hair * .7)
		c.strokePath(s.scales)
	}
	c.setStroke(col("#ffffff", .1))
	c.setLineWidth(hair * .8)
	c.strokePath(s.line)
	c.setStroke(col(C["back"], .45, -.4))
	c.setLineWidth(hair * 1.1)
	c.save()
	c.translate(math.Sin(pose.breath)*.006, 0)
	c.strokePath(s.gill)
	c.restore()
	light := c.createLinearGradient(0, s.top, 0, s.bottom)
	light.addColorStop(0, rgbaC(255, 255, 255, .24*(1-haze)))
	light.addColorStop(.32, rgbaC(255, 255, 255, .05*(1-haze)))
	light.addColorStop(.62, rgbaC(255, 255, 255, 0))
	light.addColorStop(1, rgbaC(4, 22, 30, .24))
	c.setFill(light)
	c.fillRect(-.6, s.top, 1.2, s.bottom-s.top)
	c.restore()
	c.setStroke(rgbaC(255, 255, 255, .22*(1-haze)))
	c.setLineWidth(hair * .9)
	c.strokePath(s.rim)
	// Eye, with a glint from above.
	ex, ey, er := look.eye[0], look.eye[1], look.eye[2]
	c.setFill(col("#0d0b0a", .45))
	dot(c, ex, ey, er*1.1)
	iris := c.createRadialGradient(ex, ey-er*.3, er*.2, ex, ey, er)
	iris.addColorStop(0, col(C["iris"], 1, .25))
	iris.addColorStop(1, col(C["iris"], 1, -.35))
	c.setFill(iris)
	dot(c, ex, ey, er)
	c.setFill(hexC("#050505"))
	dot(c, ex+pose.lookX*er*.2, ey+pose.lookY*er*.16, er*.62)
	c.setFill(rgbaC(255, 255, 255, .8*(1-haze*.6)))
	dot(c, ex+er*.28, ey-er*.34, er*.2)
	// Mouth, which opens for a bite.
	mx, my := look.mouth[0], look.mouth[1]
	c.setStroke(col(C["back"], .7, -.5))
	c.setLineWidth(hair)
	c.beginPath()
	c.moveTo(mx-.005, my)
	c.lineTo(mx-.055, my+.012)
	c.stroke()
	if pose.gape > 0 {
		c.setFill(col("#2a1712", .85*pose.gape))
		oval(c, mx-.01, my+.006, .028, .022*pose.gape, 0)
	}
	// The near pectoral fin rows over the body.
	p := look.pectoral
	c.save()
	c.translate(p.at[0], p.at[1])
	c.rotate(p.angle + math.Sin(pose.fin)*.4*pose.paddle)
	c.scale(.6+.4*math.Abs(math.Cos(pose.fin)), 1)
	paintFin(c, s.pectoral, C["fin"], col, .42, hair)
	c.restore()
}

var stillFish = fishPose{tail: 0, beat: .2, fin: 1.2, breath: 0, gape: 0, lookX: .3, lookY: 0, paddle: .4}

// fish swim with their tails, row with their fins while hovering, keep their
// distance from each other and turn around with a visible pivot.
type fish struct {
	spec   *Spec
	look   *fishLook
	shape  *fishShape
	seed   uint32
	rand   *rng
	colors map[string]string

	z, facing, x, homeX, y          float64
	vx, vy, pitch, turn             float64
	turnTo, turnFrom                float64
	tail, fin, breath               float64
	beat, burst, gape, lookX, lookY float64
	paddle                          float64
	mode                            string
	until                           float64
	goal                            *pt
	meal                            int
	appetite                        int
}

func newFish(spec *Spec, w *world, enter bool) *fish {
	f := &fish{spec: spec}
	f.look = lookOf(spec.ID)
	f.shape = shapeOf(f.look)
	f.seed = hash(spec.ID)
	f.rand = random(f.seed ^ 0x9e3779b9)
	f.colors = spec.Colors
	r := random(f.seed)
	f.z = .15 + r.next()*.55
	f.facing = 1
	if r.next() < .5 {
		f.facing = -1
	}
	f.x = w.width * (.15 + r.next()*.7)
	f.homeX = f.x
	f.y = w.height * lerp(spec.Zone[0], spec.Zone[1], .3+r.next()*.4)
	if enter {
		if f.facing > 0 {
			f.x = -w.width * .08
		} else {
			f.x = w.width * 1.08
		}
	}
	f.turnTo, f.turnFrom = f.facing, f.facing
	f.tail = r.next() * TAU
	f.fin = r.next() * TAU
	f.breath = r.next() * TAU
	f.beat, f.lookX, f.paddle = .3, .2, .5
	f.mode = "hover"
	if enter {
		f.mode = "swim"
	}
	return f
}

func (f *fish) isCrawler() bool    { return false }
func (f *fish) depth() float64     { return f.z }
func (f *fish) under(*ctx, *world) {}

func (f *fish) size(w *world) float64 { return f.spec.Length * w.px * (1 - .3*f.z) }

func (f *fish) resize(sx, sy float64, w *world) {
	f.x *= sx
	f.y *= sy
	f.goal = nil
}

// pick chooses the next place to swim to inside the species' zone.
func (f *fish) pick(w *world, L float64) *pt {
	s, W, H := f.spec, w.width, w.height
	if s.Home {
		h := w.home(f.homeX)
		r := math.Max(L*2.2, W*.1)
		x := clamp(h[0]+(f.rand.next()*2-1)*r, L, W-L)
		return &pt{x, clamp(h[1]+(f.rand.next()*2-1)*r*.5, H*.12, w.ground(x, f.z)-L*.45)}
	}
	var rock *rock
	if s.Rocky {
		rock = w.rocks[int(math.Floor(f.rand.next()*float64(len(w.rocks))))]
	}
	var x float64
	if rock != nil {
		x = clamp(rock.x+(f.rand.next()*2-1)*W*.15, L, W-L)
	} else {
		x = L + f.rand.next()*(W-2*L)
	}
	return &pt{x, math.Min(H*lerp(s.Zone[0], s.Zone[1], f.rand.next()), w.ground(x, f.z)-L*.45)}
}

// hunt finds the nearest sinking pellet and bites it when close enough;
// after a few bites a fish loses interest and leaves the rest to sink.
func (f *fish) hunt(w *world, L float64) *pellet {
	if f.meal != w.meal {
		f.meal = w.meal
		f.appetite = 2 + int(math.Floor(f.rand.next()*2))
	}
	if f.appetite <= 0 {
		return nil
	}
	var best *pellet
	bd := math.Inf(1)
	for _, p := range w.pellets {
		if p.landed || p.eaten || p.alpha < 1 {
			continue
		}
		d := math.Hypot(p.x-(f.x+f.facing*L*.5), p.y-f.y)
		if d < bd {
			bd = d
			best = p
		}
	}
	if best != nil && math.Hypot(f.x+f.facing*L*.5-best.x, f.y-best.y) < L*.18 {
		w.eat(best)
		f.gape = 1
		f.appetite--
	}
	if best != nil && !best.eaten {
		return best
	}
	return nil
}

func (f *fish) step(dt float64, w *world) {
	s, L, W, H, t := f.spec, f.size(w), w.width, w.height, w.time
	f.z = follow(f.z, .1+.65*(noise(f.seed, t*.025)*.5+.5), .2, dt)
	cruise := s.Speed * L
	var goal *pt
	speed := cruise
	food := f.hunt(w, L)
	if food != nil {
		goal = &pt{food.x - f.facing*L*.45, food.y}
		speed = cruise * 1.8
		f.mode = "swim"
		f.goal = nil
	} else {
		if f.mode == "hover" && t > f.until {
			f.mode = "swim"
			f.goal = f.pick(w, L)
		}
		if f.mode == "swim" {
			if f.goal == nil {
				f.goal = f.pick(w, L)
			}
			if math.Hypot(f.goal[0]-f.x, f.goal[1]-f.y) < L*.6 {
				if f.rand.next() < s.Hover {
					f.mode = "hover"
					f.until = t + 2 + f.rand.next()*6
					f.goal = nil
				} else {
					f.goal = f.pick(w, L)
				}
			}
			goal = f.goal
		}
	}
	var dvx, dvy float64
	if goal != nil {
		dx, dy := goal[0]-f.x, goal[1]-f.y
		d := or(math.Hypot(dx, dy), 1)
		v := speed * clamp(d/(L*1.6), .3, 1) * (1 + f.burst*.9)
		dvx, dvy = dx/d*v, dy/d*v*.6
	} else {
		dvx = noise(f.seed+3, t*.4) * L * .12
		dvy = noise(f.seed+5, t*.3) * L * .08
	}
	// Keep a polite distance from fish at a similar depth, or pass at another depth.
	for _, o := range w.fish {
		if o == f || math.Abs(o.z-f.z) > .25 {
			continue
		}
		ox, oy := f.x-o.x, f.y-o.y
		d := or(math.Hypot(ox, oy), 1)
		near := (L + o.size(w)) * .75
		if d < near {
			push := (near - d) / near
			dvx += ox / d * push * cruise * 3.5
			dvy += oy / d * push * cruise * 5
			// Crowded at the same depth, one drifts in front and the other behind.
			tie := -1.0
			if f.seed > o.seed {
				tie = 1
			}
			f.z = clamp(f.z+or(sign(f.z-o.z), tie)*push*.8*dt, 0, .9)
		}
	}
	// Stay in the water and inside the glass.
	floor := w.ground(f.x, f.z) - L*.3
	if f.y > floor {
		dvy -= (f.y - floor) * 3
	}
	if f.y < H*.08 {
		dvy += (H*.08 - f.y) * 3
	}
	if f.x < L*.6 {
		dvx += (L*.6 - f.x) * 2
	}
	if f.x > W-L*.6 {
		dvx -= (f.x - W + L*.6) * 2
	}
	accel := cruise * 2.4
	ax := clamp((dvx-f.vx)*2.2, -accel, accel)
	ay := clamp((dvy-f.vy)*2.2, -accel, accel)
	if f.turn > 0 && f.turn < .5 {
		ax = -f.vx * 3
	}
	f.vx += ax * dt
	f.vy += ay * dt
	f.x += f.vx * dt
	f.y += f.vy * dt
	// Face the way it swims; hovering fish may face into the current or look around.
	want := f.facing
	if math.Abs(f.vx) > L*.25 {
		want = sign(f.vx)
	} else if f.mode == "hover" && food == nil {
		if s.Upstream {
			want = -1
		} else if s.Curious {
			if noise(f.seed+9, t*.12) > .1 {
				want = 1
			} else {
				want = -1
			}
		}
	}
	if f.turn == 0 && want != f.facing {
		f.turn = 1e-3
		f.turnTo = want
		f.turnFrom = f.facing
	}
	if f.turn != 0 {
		f.turn += dt / .5
		if f.turn >= .5 {
			f.facing = f.turnTo
		}
		if f.turn >= 1 {
			f.turn = 0
		}
	}
	f.pitch = follow(f.pitch, clamp(math.Atan2(f.vy, math.Abs(f.vx)+L*.3), -.35, .35), 3, dt)
	// Fins: thrust drives the tail; slow fish row with their pectorals.
	pace := math.Hypot(f.vx, f.vy) / cruise
	push := math.Max(0, ax*f.facing) / accel
	beat := clamp(.14+pace*.5+push*.7+f.burst*.5, .12, 1)
	if s.Propulsion == "pectoral" {
		if pace > 1.1 {
			beat *= .9
		} else {
			beat *= .35
		}
	}
	if s.Propulsion == "fins" {
		if pace > 1.2 {
			beat *= .8
		} else {
			beat *= .15
		}
	}
	f.beat = follow(f.beat, beat, 4, dt)
	wig := 1.0
	if s.Wiggle {
		wig = 1.45
	}
	f.tail += TAU * (1 + 2.4*f.beat*wig) * dt
	rowing := pace < .4 || s.Propulsion != "tail"
	pt := .35
	if rowing {
		pt = 1
	}
	f.paddle = follow(f.paddle, pt, 3, dt)
	fr := 1.1
	if rowing {
		fr = 2.2
	}
	f.fin += TAU * fr * dt
	f.breath += TAU * 1.2 * dt
	if s.Burst {
		f.burst = follow(f.burst, 0, 1.4, dt)
		if f.mode == "swim" && f.rand.next() < dt*.45 {
			f.burst = 1
		}
	}
	f.gape = math.Max(0, f.gape-dt*3)
	if s.Curious {
		f.lookX = noise(f.seed+11, t*.5)
		f.lookY = noise(f.seed+13, t*.4)
	} else {
		f.lookX, f.lookY = .2, 0
	}
}

func (f *fish) pose() *fishPose {
	return &fishPose{tail: f.tail, beat: f.beat, fin: f.fin, breath: f.breath, gape: f.gape, lookX: f.lookX, lookY: f.lookY, paddle: f.paddle, colors: f.colors}
}

func (f *fish) draw(c *ctx, w *world) {
	L, haze := f.size(w), f.z*.5
	pose := f.pose()
	c.save()
	c.translate(f.x, f.y)
	if f.turn == 0 {
		c.scale(f.facing, 1)
		wig := 0.0
		if f.spec.Wiggle {
			wig = math.Sin(f.tail) * .05 * f.beat
		}
		c.rotate(f.pitch + wig)
		c.scale(L, L)
		drawFish(c, f.look, f.shape, pose, haze, L)
	} else {
		// Turning, the body bends: the head swings round first and the tail
		// follows, each half foreshortened as it points toward us.
		p := f.turn
		bend := func(from float64) float64 {
			v := math.Cos(math.Pi * clamp(p*1.5-from, 0, 1))
			return sign(or(v, 1)) * math.Max(.16, math.Abs(v))
		}
		c.scale(f.turnFrom, 1)
		c.rotate(f.pitch)
		for _, h := range []struct{ half, scale float64 }{{-1, bend(.5)}, {1, bend(0)}} {
			c.save()
			c.scale(h.scale*L, L)
			c.beginPath()
			x := 0.0
			if h.half < 0 {
				x = -2
			}
			c.rect(x, -2, 2, 4)
			c.clip()
			drawFish(c, f.look, f.shape, pose, haze, L)
			c.restore()
		}
	}
	c.restore()
}

// fishPortrait paints a still fish filling a w × h box.
func fishPortrait(c *ctx, spec *Spec, w, h float64) {
	look := lookOf(spec.ID)
	s := shapeOf(look)
	b := s.bounds
	L := math.Min(w*.92/(b.x1-b.x0), h*.9/(b.y1-b.y0))
	c.save()
	c.translate(w/2-(b.x0+b.x1)/2*L, h/2-(b.y0+b.y1)/2*L)
	c.scale(L, L)
	pose := stillFish
	pose.colors = spec.Colors
	drawFish(c, look, s, &pose, 0, L)
	c.restore()
}
