package reef

// Colors and outline helpers shared by the creature models. Ported from
// engine/paint.js.

import (
	"math"
	"strconv"
	"strings"
	"sync"
)

var (
	parsedMu sync.Mutex
	parsed   = map[string][3]float64{}
)

// jsParseHex is parseInt(s, 16): the leading hex digits, or NaN when there are none.
func jsParseHex(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r")
	neg := false
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	n := 0
	for n < len(s) && strings.ContainsRune("0123456789abcdefABCDEF", rune(s[n])) {
		n++
	}
	if n == 0 {
		return math.NaN()
	}
	v, _ := strconv.ParseUint(s[:n], 16, 64)
	if neg {
		return -float64(v)
	}
	return float64(v)
}

// rgb parses a "#rrggbb" color once, as the JS does (slices 1–3, 3–5, 5–7
// through parseInt, so anything else parses to NaN parts).
func rgb(hex string) [3]float64 {
	parsedMu.Lock()
	defer parsedMu.Unlock()
	c, ok := parsed[hex]
	if !ok {
		for k, i := range []int{1, 3, 5} {
			lo, hi := min(i, len(hex)), min(i+2, len(hex))
			c[k] = jsParseHex(hex[lo:hi])
		}
		parsed[hex] = c
	}
	return c
}

// Distance fades every color toward the water, the stage's middle tone.
var water = rgb("#173f4c")

// toInt is JS's v | 0 for the values here (NaN becomes 0).
func toInt(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return float64(int32(int64(math.Trunc(v))))
}

// tone returns a color: light (−1…1) darkens or lightens it first, then
// haze (0…1) fades it toward the water the way distance does.
func tone(hex string, haze, alpha, light float64) rgba {
	c := rgb(hex)
	r, g, b := c[0], c[1], c[2]
	if light > 0 {
		r += (255 - r) * light
		g += (255 - g) * light
		b += (255 - b) * light
	} else if light < 0 {
		r *= 1 + light
		g *= 1 + light
		b *= 1 + light
	}
	r += (water[0] - r) * haze
	g += (water[1] - g) * haze
	b += (water[2] - b) * haze
	return rgba{toInt(r), toInt(g), toInt(b), alpha}
}

// shadeFn is tone() bound to one creature's haze: col(hex), col(hex, alpha), col(hex, alpha, light).
type shadeFn func(hex string, args ...float64) rgba

func shade(haze float64) shadeFn {
	return func(hex string, args ...float64) rgba {
		alpha, light := 1.0, 0.0
		if len(args) > 0 {
			alpha = args[0]
		}
		if len(args) > 1 {
			light = args[1]
		}
		return tone(hex, haze, alpha, light)
	}
}

type pt = [2]float64

// curve adds a smooth line through points (Catmull-Rom as cubic Béziers) to a
// path or a context; closed curves wrap around to the first point.
func curve(path pathSink, points []pt, closed bool) {
	n := len(points)
	at := func(i int) pt {
		if closed {
			return points[((i%n)+n)%n]
		}
		return points[max(0, min(n-1, i))]
	}
	path.moveTo(points[0][0], points[0][1])
	segs := n - 1
	if closed {
		segs = n
	}
	for i := 0; i < segs; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		path.bezierCurveTo(p1[0]+(p2[0]-p0[0])/6, p1[1]+(p2[1]-p0[1])/6, p2[0]-(p3[0]-p1[0])/6, p2[1]-(p3[1]-p1[1])/6, p2[0], p2[1])
	}
	if closed {
		path.closePath()
	}
}

// curvePath is curve(new Path2D(), points, closed).
func curvePath(points []pt, closed bool) *path2D {
	p := newPath()
	curve(p, points, closed)
	return p
}

// limb strokes a jointed leg through points, thinning from base to tip.
func limb(c *ctx, points []pt, base, tip float64, color rgba) {
	c.setStroke(color)
	c.roundCaps()
	c.roundJoin()
	for i := 1; i < len(points); i++ {
		c.setLineWidth(base + (tip-base)*float64(i-1)/math.Max(1, float64(len(points)-2)))
		c.beginPath()
		c.moveTo(points[i-1][0], points[i-1][1])
		c.lineTo(points[i][0], points[i][1])
		c.stroke()
	}
}

// shadow lays the soft contact shadow under something resting on the sand.
func shadow(c *ctx, x, y, rx, ry, alpha float64) {
	c.save()
	c.translate(x, y)
	c.scale(1, ry/rx)
	g := c.createRadialGradient(0, 0, 0, 0, 0, rx)
	g.addColorStop(0, rgbaC(8, 22, 20, alpha))
	g.addColorStop(1, rgbaC(8, 22, 20, 0))
	c.setFill(g)
	c.beginPath()
	c.arc(0, 0, rx, 0, math.Pi*2, false)
	c.fill()
	c.restore()
}

// dot fills a circle.
func dot(c *ctx, x, y, r float64) {
	c.beginPath()
	c.arc(x, y, r, 0, math.Pi*2, false)
	c.fill()
}

// oval fills an ellipse.
func oval(c *ctx, x, y, rx, ry, rotation float64) {
	c.beginPath()
	c.ellipse(x, y, math.Abs(rx), math.Abs(ry), rotation, 0, math.Pi*2, false)
	c.fill()
}

// wind returns polygon points in clockwise order, so overlapping parts of
// one body add up when filled or clipped together.
func wind(points []pt) []pt {
	area := 0.0
	for i := range points {
		p0, p1 := points[i], points[(i+1)%len(points)]
		area += p0[0]*p1[1] - p1[0]*p0[1]
	}
	if area < 0 {
		out := make([]pt, len(points))
		for i, p := range points {
			out[len(points)-1-i] = p
		}
		return out
	}
	return points
}

// hole draws a burrow opening with its ring of dug-out sand; lip redraws the
// front of the ring over whatever is still down in the burrow.
func hole(c *ctx, x, y, r float64) {
	c.setFill(hexC("#7f8e78"))
	oval(c, x, y, r*1.6, r*.55, 0)
	c.setFill(hexC("#16201d"))
	oval(c, x, y-r*.06, r*.85, r*.3, 0)
}

func lip(c *ctx, x, y, r float64) {
	c.setFill(hexC("#7f8e78"))
	c.beginPath()
	c.ellipse(x, y, r*1.6, r*.55, 0, 0, math.Pi, false)
	c.fill()
	c.setFill(rgbaC(160, 172, 150, .5))
	c.beginPath()
	c.ellipse(x, y+r*.05, r*1.2, r*.32, 0, .2, math.Pi-.2, false)
	c.fill()
}
