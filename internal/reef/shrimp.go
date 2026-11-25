package reef

// Shrimp, ported from engine/shrimp.js.
//
// Shrimp are drawn facing right in units of body length, standing on the
// ground at y = 0 with the carapace raised on their legs. The abdomen is a
// chain of six segments that arches over the back and curls under for an
// escape flip; the tail fan opens at its end.

import "math"

type shrimpShape struct {
	carapace, rostrum []pt
	waist             pt
	segments          [][2]float64
	arch              []float64
	fan               float64
	hips              []pt
	rests             []float64
	upper, lower      float64
	eye               [3]float64
	antenna           *pt
	mouth             pt
}

var shrimpShapes = map[string]*shrimpShape{
	// Lysmata and its relatives: a slim body with a hump in the abdomen.
	"slim": {
		carapace: []pt{{.42, -.215}, {.34, -.275}, {.2, -.3}, {.06, -.29}, {-.01, -.255}, {-.02, -.17}, {.06, -.135}, {.2, -.128}, {.33, -.142}, {.41, -.18}},
		rostrum:  []pt{{.36, -.258}, {.53, -.282}, {.4, -.225}},
		waist:    pt{0, -.215}, segments: [][2]float64{{.085, .16}, {.08, .165}, {.075, .15}, {.07, .13}, {.065, .105}, {.075, .08}},
		arch: []float64{.3, .1, -.3, -.34, -.24, -.12}, fan: .12,
		hips: []pt{{.34, -.14}, {.28, -.135}, {.21, -.13}, {.14, -.13}, {.07, -.135}}, rests: []float64{.44, .32, .2, .09, -.02}, upper: .085, lower: .1,
		eye: [3]float64{.38, -.25, .034}, antenna: &pt{.39, -.22}, mouth: pt{.43, -.15},
	},
	// The harlequin shrimp is short and deep, carrying two flat claws.
	"stocky": {
		carapace: []pt{{.38, -.2}, {.32, -.29}, {.18, -.32}, {.04, -.3}, {-.02, -.26}, {-.03, -.15}, {.06, -.12}, {.2, -.115}, {.32, -.13}, {.38, -.16}},
		rostrum:  []pt{{.33, -.25}, {.43, -.27}, {.36, -.22}},
		waist:    pt{-.01, -.215}, segments: [][2]float64{{.08, .19}, {.075, .19}, {.07, .17}, {.065, .145}, {.06, .115}, {.065, .09}},
		arch: []float64{.24, .08, -.26, -.3, -.24, -.12}, fan: .11,
		hips: []pt{{.31, -.125}, {.25, -.12}, {.18, -.115}, {.11, -.115}, {.04, -.12}}, rests: []float64{.4, .29, .18, .07, -.04}, upper: .075, lower: .085,
		eye: [3]float64{.35, -.24, .032}, antenna: &pt{.36, -.21}, mouth: pt{.4, -.14},
	},
	// The pistol shrimp is stockier, with its eyes under a hood.
	"stout": {
		carapace: []pt{{.4, -.2}, {.33, -.26}, {.2, -.28}, {.06, -.27}, {-.01, -.24}, {-.02, -.15}, {.06, -.12}, {.2, -.115}, {.32, -.13}, {.4, -.165}},
		rostrum:  []pt{{.36, -.235}, {.44, -.24}, {.38, -.205}},
		waist:    pt{0, -.2}, segments: [][2]float64{{.08, .15}, {.08, .155}, {.075, .15}, {.07, .135}, {.065, .115}, {.07, .09}},
		arch: []float64{.22, .06, -.22, -.26, -.2, -.1}, fan: .11,
		hips: []pt{{.32, -.125}, {.26, -.12}, {.2, -.115}, {.13, -.115}, {.06, -.12}}, rests: []float64{.42, .31, .19, .08, -.03}, upper: .08, lower: .09,
		eye: [3]float64{.37, -.215, .026}, antenna: &pt{.38, -.2}, mouth: pt{.42, -.14},
	},
	// The slipper lobster is flat and drawn a little from above (see paintSlipper);
	// only its legs use this side-view layout.
	"flat": {
		hips: []pt{{.3, -.04}, {.22, -.04}, {.14, -.04}, {.06, -.04}, {-.02, -.04}}, rests: []float64{.34, .25, .16, .07, -.02}, upper: .04, lower: .045,
		mouth: pt{.4, -.06},
	},
}

type shrimpLook struct {
	shape  string
	whip   float64
	colors map[string]string
}

var shrimpLooks = map[string]*shrimpLook{
	"cleaner":   {shape: "slim", whip: 2.3, colors: map[string]string{"body": "#f1b744", "belly": "#e7a53b", "band": "#c7302a", "stripe": "#f7f2e8", "leg": "#efe6d8", "fan": "#c7302a", "whip": "#f4efe6"}},
	"fire":      {shape: "slim", whip: 2, colors: map[string]string{"body": "#b81e25", "belly": "#a11a20", "band": "#b81e25", "leg": "#f1e9de", "fan": "#b81e25", "whip": "#f4efe6", "spots": "#f7efe5"}},
	"harlequin": {shape: "stocky", whip: .3, colors: map[string]string{"body": "#f3ebdc", "belly": "#e8dccb", "leg": "#efe3d6", "fan": "#f1e6d6", "whip": "#ead9c9", "patch": "#c64c86", "edge": "#6d2f62"}},
	"pistol":    {shape: "stout", whip: .9, colors: map[string]string{"body": "#ecdfc6", "belly": "#dcc9aa", "leg": "#e6c7a0", "fan": "#e3cfb1", "whip": "#d58a52", "patch": "#b4642c"}},
	"slipper":   {shape: "flat", whip: 0, colors: map[string]string{"body": "#a58260", "belly": "#8a6a4c", "leg": "#8a6a4c", "fan": "#9b7a58", "patch": "#5f4734", "plate": "#9a7856"}},
}

type chainSeg struct{ x, y, nx, ny, a, h, ux, uy float64 }

// chain lays out the abdomen chain for a given curl (0 relaxed … 1 tucked).
func chain(shape *shrimpShape, curl float64) []chainSeg {
	out := make([]chainSeg, 0, len(shape.segments))
	x, y, a := shape.waist[0], shape.waist[1], math.Pi
	for i, sg := range shape.segments {
		ln, h := sg[0], sg[1]
		a += shape.arch[i] - curl*(.22+float64(i)*.07)
		nx, ny := x+math.Cos(a)*ln, y+math.Sin(a)*ln
		out = append(out, chainSeg{x, y, nx, ny, a, h, -math.Sin(a), math.Cos(a)})
		x, y = nx, ny
	}
	return out
}

type shrimpPose struct {
	near, far                                   []pt
	curl, whip, whip2, paddle, work, open, haze float64
}

// paintShrimp draws a shrimp in unit space. pose has the feet in unit
// coordinates, the curl, antenna angles and the claws' work.
func paintShrimp(c *ctx, id string, pose *shrimpPose, haze, L float64) {
	look := shrimpLooks[id]
	shape := shrimpShapes[look.shape]
	C, hair, col := look.colors, 1/L, shade(haze)
	if id == "slipper" {
		paintSlipper(c, pose, col, C, hair)
		return
	}
	segs := chain(shape, pose.curl)
	tail := segs[len(segs)-1]
	// Far side first: the far antenna and legs, darker and a little higher.
	if look.whip != 0 {
		whip(c, *shape.antenna, pose.whip2, look.whip*.95, col(C["whip"], .55, -.25), hair*.7)
	}
	shrimpLegs(c, shape, pose.far, col(C["leg"], .75, -.35), hair*1.2)
	// Swimmerets beat under the abdomen.
	c.setStroke(col(C["leg"], .7, -.1))
	c.roundCaps()
	c.setLineWidth(hair * 1.4)
	for i, s := range segs[:5] {
		mx, my := (s.x+s.nx)/2-s.ux*s.h*.45, (s.y+s.ny)/2-s.uy*s.h*.45
		swing := math.Sin(pose.paddle-float64(i)*.8) * .5
		c.beginPath()
		c.moveTo(mx, my)
		c.lineTo(mx-s.ux*.07+math.Cos(s.a+swing)*.03, my-s.uy*.07+math.Sin(s.a+swing)*.03)
		c.stroke()
	}
	// Tail fan: the telson with a uropod on each side.
	fanA, open := tail.a+.05, shape.fan*(1+pose.curl*.3)
	for _, u := range [][3]float64{{.32 + pose.curl*.3, 1, .38}, {-.3 - pose.curl*.3, 1, .38}, {0, 1.05, .28}} {
		da, ln, wd := u[0], u[1], u[2]
		light := 0.0
		if da == 0 {
			light = -.1
		}
		c.setFill(col(C["fan"], .82, light))
		oval(c, tail.nx+math.Cos(fanA+da)*open*ln*.5, tail.ny+math.Sin(fanA+da)*open*ln*.5, open*ln*.55, open*wd*.5, fanA+da)
	}
	// Body: carapace and abdomen as one shape, so markings run across them.
	body := newPath()
	curve(body, wind(shape.carapace), true)
	if shape.rostrum != nil {
		curve(body, wind(shape.rostrum), true)
	}
	var top, under []pt
	for _, s := range segs {
		top = append(top, pt{s.x + s.ux*s.h/2, s.y + s.uy*s.h/2})
	}
	for _, s := range segs {
		under = append(under, pt{s.x - s.ux*s.h/2, s.y - s.uy*s.h/2}, pt{(s.x+s.nx)/2 - s.ux*s.h*.58, (s.y+s.ny)/2 - s.uy*s.h*.58})
	}
	top = append(top, pt{tail.nx + tail.ux*tail.h*.42, tail.ny + tail.uy*tail.h*.42})
	under = append(under, pt{tail.nx - tail.ux*tail.h*.42, tail.ny - tail.uy*tail.h*.42})
	abdomen := append([]pt(nil), top...)
	for i := len(under) - 1; i >= 0; i-- {
		abdomen = append(abdomen, under[i])
	}
	curve(body, wind(abdomen), true)
	g := c.createLinearGradient(0, -.33, 0, -.1)
	g.addColorStop(0, col(C["body"], .96, .06))
	g.addColorStop(1, col(C["belly"], .92, -.05))
	c.setFill(g)
	c.fillPath(body)
	c.save()
	c.clipPath(body)
	back := []pt{{.4, -.27}, {.2, -.31}, {.04, -.29}}
	for _, s := range segs {
		back = append(back, pt{s.nx + s.ux*s.h*.38, s.ny + s.uy*s.h*.38})
	}
	if C["band"] != C["body"] && C["band"] != "" {
		shifted := func(dy float64) []pt {
			out := make([]pt, len(back))
			for i, p := range back {
				out[i] = pt{p[0], p[1] + dy}
			}
			return out
		}
		c.setStroke(col(C["band"]))
		c.setLineWidth(.1)
		c.roundJoin()
		c.beginPath()
		curve(c, shifted(.02), false)
		c.stroke()
		c.setStroke(col(C["stripe"]))
		c.setLineWidth(.02)
		c.beginPath()
		curve(c, shifted(.012), false)
		c.stroke()
	}
	if C["spots"] != "" {
		c.setFill(col(C["spots"], .9))
		for _, d := range [][3]float64{{.3, -.23, .014}, {.22, -.2, .01}, {.14, -.24, .012}, {.06, -.19, .011}, {.25, -.16, .009}} {
			dot(c, d[0], d[1], d[2])
		}
		for _, s := range segs {
			dot(c, (s.x+s.nx)/2+s.ux*.02, (s.y+s.ny)/2+s.uy*.02, .012)
		}
	}
	if id == "harlequin" {
		spots := [][3]float64{{.3, -.22, .045}, {.13, -.24, .05}}
		for i, s := range segs {
			if i%2 == 0 {
				spots = append(spots, [3]float64{(s.x + s.nx) / 2, (s.y + s.ny) / 2, .045})
			}
		}
		for _, d := range spots {
			c.setFill(col(C["edge"], .9))
			dot(c, d[0], d[1], d[2]+.012)
			c.setFill(col(C["patch"]))
			dot(c, d[0], d[1], d[2])
		}
	}
	if id == "pistol" {
		c.setFill(col(C["patch"], .85))
		for _, s := range segs {
			oval(c, (s.x+s.nx)/2+s.ux*.02, (s.y+s.ny)/2+s.uy*.02, .03, .05, s.a)
		}
		oval(c, .2, -.22, .07, .04, -.1)
		oval(c, .06, -.2, .04, .05, 0)
	}
	// Joints between segments and the light from above.
	c.setStroke(col(C["body"], .4, -.45))
	c.setLineWidth(hair * .8)
	for _, s := range segs {
		c.beginPath()
		c.moveTo(s.x+s.ux*s.h*.55, s.y+s.uy*s.h*.55)
		c.quadraticCurveTo(s.x+math.Cos(s.a)*.015, s.y+math.Sin(s.a)*.015, s.x-s.ux*s.h*.6, s.y-s.uy*s.h*.6)
		c.stroke()
	}
	light := c.createLinearGradient(0, -.33, 0, -.08)
	light.addColorStop(0, rgbaC(255, 255, 255, .3*(1-haze)))
	light.addColorStop(.45, rgbaC(255, 255, 255, 0))
	light.addColorStop(1, rgbaC(0, 18, 24, .25))
	c.setFill(light)
	c.fillRect(-.6, -.4, 1.2, .4)
	c.restore()
	// Near legs, claws, antennae and the eye.
	shrimpLegs(c, shape, pose.near, col(C["leg"], .78), hair*1.35)
	if id == "harlequin" {
		petals(c, pose, col, C)
	}
	if id == "pistol" {
		snapper(c, pose, col, C, hair)
	}
	if shape.antenna != nil {
		ax, ay := shape.antenna[0], shape.antenna[1]
		c.setStroke(col(C["whip"], .9))
		c.setLineWidth(hair * .9)
		for _, lift := range []float64{-.5, -.75} {
			c.beginPath()
			c.moveTo(ax+.01, ay-.02)
			c.quadraticCurveTo(ax+.1, ay-.08+lift*.05, ax+.17, ay-.07+lift*.12+math.Sin(pose.paddle*.7+lift)*.02)
			c.stroke()
		}
		if look.whip != 0 {
			whip(c, *shape.antenna, pose.whip, look.whip, col(C["whip"], .9), hair*.9)
		}
	}
	ex, ey, er := shape.eye[0], shape.eye[1], shape.eye[2]
	c.setStroke(col(C["body"], .9, -.2))
	c.setLineWidth(er * .9)
	c.beginPath()
	c.moveTo(ex-.04, ey+.01)
	c.lineTo(ex, ey)
	c.stroke()
	c.setFill(col("#0d0a09", .95))
	dot(c, ex, ey, er)
	c.setFill(rgbaC(255, 255, 255, .75*(1-haze)))
	dot(c, ex+er*.3, ey-er*.35, er*.3)
	if id == "pistol" {
		c.setFill(col(C["body"], .45, .15))
		oval(c, ex-.004, ey-.012, er*1.5, er*1.15, 0)
	}
}

// whip draws a long antenna that arcs forward and droops toward its tip.
func whip(c *ctx, a pt, angle, length float64, color rgba, width float64) {
	ax, ay := a[0], a[1]
	tx, ty := ax+math.Cos(angle)*length, ay+math.Sin(angle)*length*.55+length*.12
	c.setStroke(color)
	c.setLineWidth(width)
	c.roundCaps()
	c.beginPath()
	c.moveTo(ax, ay)
	c.bezierCurveTo(ax+math.Cos(angle-.35)*length*.35, ay+math.Sin(angle-.35)*length*.35, ax+math.Cos(angle-.1)*length*.7, ay+math.Sin(angle-.1)*length*.6, tx, ty)
	c.stroke()
}

// shrimpLegs draws walking legs from their hips to the feet, knees raised.
func shrimpLegs(c *ctx, shape *shrimpShape, feet []pt, color rgba, width float64) {
	for i, f := range feet {
		h := shape.hips[i]
		k := knee(h[0], h[1], f[0], f[1], shape.upper, shape.lower)
		limb(c, []pt{h, k, f}, width, width*.7, color)
	}
}

// petals are the harlequin shrimp's flat, spotted claws held up in front.
func petals(c *ctx, pose *shrimpPose, col shadeFn, C map[string]string) {
	for _, u := range [][2]float64{{0, 0}, {.03, .05}} {
		dx, lift := u[0], u[1]
		c.save()
		c.translate(.38+dx, -.16)
		c.rotate(-.9 - lift + math.Sin(pose.paddle*.3)*.05 - pose.work*.3)
		c.setFill(col(C["body"], .95, .05))
		oval(c, .14, 0, .14, .06, 0)
		c.setFill(col(C["edge"], .9))
		dot(c, .12, 0, .035)
		c.setFill(col(C["patch"]))
		dot(c, .12, 0, .026)
		c.setFill(col(C["edge"], .9))
		dot(c, .22, .01, .02)
		c.setFill(col(C["patch"]))
		dot(c, .22, .01, .014)
		c.restore()
	}
}

// snapper is the pistol shrimp's big claw; its finger snaps shut.
func snapper(c *ctx, pose *shrimpPose, col shadeFn, C map[string]string, hair float64) {
	c.save()
	c.translate(.4, -.13)
	c.rotate(-.15 - pose.work*.25)
	limb(c, []pt{{-.06, -.01}, {.04, .02}}, hair*3, hair*3, col(C["leg"]))
	c.setFill(col("#c9733a"))
	oval(c, .14, .01, .11, .05, -.05)
	c.setFill(col(C["patch"], .9, -.15))
	oval(c, .12, .02, .05, .02, 0)
	c.save()
	c.translate(.24, -.01)
	c.rotate(-.5 * pose.open)
	c.setFill(col("#c9733a", 1, -.08))
	oval(c, .05, 0, .055, .018, 0)
	c.restore()
	c.setFill(rgbaC(255, 255, 255, .25*(1-pose.haze)))
	oval(c, .13, -.015, .06, .012, -.05)
	c.restore()
}

// paintSlipper draws the slipper lobster from a little above: a broad flat
// shell shaped like a pipa, two shovel plates in front and a wide tail fan.
// The outline is laid out as seen from straight above (u along the body, v
// across it), squashed to the viewing angle, with its side showing below.
func paintSlipper(c *ctx, pose *shrimpPose, col shadeFn, C map[string]string, hair float64) {
	squash, high := .55, -.1
	plan := []pt{{.4, -.04}, {.4, -.2}, {.36, -.255}, {.28, -.27}, {.14, -.255}, {.02, -.225}, {-.08, -.18}, {-.2, -.155}, {-.31, -.13}, {-.39, -.11}, {-.43, -.14}, {-.5, -.2}, {-.56, -.12}, {-.58, -.04}, {-.56, 0}}
	outline := append([]pt(nil), plan...)
	for i := len(plan) - 2; i >= 0; i-- {
		outline = append(outline, pt{plan[i][0], -plan[i][1]})
	}
	place := func(pts []pt, dy float64) []pt {
		out := make([]pt, len(pts))
		for i, p := range pts {
			out[i] = pt{p[0], high + p[1]*squash + dy}
		}
		return out
	}
	flat := shrimpShapes["flat"]
	// Short legs poke out from under the shell.
	shrimpLegs(c, flat, pose.far, col(C["leg"], .7, -.3), hair*1.3)
	// The side of the shell, then its top.
	c.setFill(col(C["belly"], 1, -.25))
	c.fillPath(curvePath(place(outline, .035), true))
	top := curvePath(place(outline, 0), true)
	g := c.createLinearGradient(0, high-.09, 0, high+.09)
	g.addColorStop(0, col(C["body"], 1, .1))
	g.addColorStop(1, col(C["body"], 1, -.12))
	c.setFill(g)
	c.fillPath(top)
	c.save()
	c.clipPath(top)
	c.setFill(col(C["patch"], .5))
	for _, d := range [][3]float64{{.25, -.13, .035}, {.2, .1, .03}, {.06, -.05, .04}, {.3, .16, .028}, {-.1, .09, .03}, {-.2, -.07, .03}, {-.3, .05, .025}, {.12, .17, .022}, {.34, -.03, .025}} {
		oval(c, d[0], high+d[1]*squash, d[2], d[2]*squash*1.3, 0)
	}
	c.setFill(col("#e3cfae", .5))
	for i := 0; i < 18; i++ {
		dot(c, .36-math.Mod(float64(i)*.029, .42), high+(float64((i*37)%23)/23-.5)*.36*squash, .007)
	}
	// The carapace's rim and the abdomen's segment lines.
	c.setStroke(col(C["patch"], .5))
	c.setLineWidth(hair)
	c.beginPath()
	curve(c, place([]pt{{.36, -.25}, {.14, -.2}, {.0, -.14}, {-.03, 0}, {.0, .14}, {.14, .2}, {.36, .25}}, 0), false)
	c.stroke()
	for _, u := range []float64{-.1, -.18, -.26, -.34} {
		c.beginPath()
		curve(c, place([]pt{{u + .015, -.15}, {u - .015, 0}, {u + .015, .15}}, 0), false)
		c.stroke()
	}
	c.setStroke(col("#ead8b8", .4))
	c.setLineWidth(hair * 1.4)
	c.beginPath()
	curve(c, place([]pt{{.36, -.02}, {.1, -.03}, {-.36, -.015}}, 0), false)
	c.stroke()
	c.restore()
	// The two shovel plates in front, with toothed edges.
	for _, side := range []float64{-1, 1} {
		plate := []pt{{.39, side * .015}, {.48, side * .02}, {.56, side * .06}, {.53, side * .1}, {.57, side * .15}, {.52, side * .22}, {.45, side * .26}, {.39, side * .24}}
		c.setFill(col(C["belly"], 1, -.25))
		c.fillPath(curvePath(place(plate, .03), true))
		light := -.04
		if side < 0 {
			light = .08
		}
		c.setFill(col(C["plate"], 1, light))
		c.fillPath(curvePath(place(plate, 0), true))
		c.setStroke(col(C["patch"], .55))
		c.setLineWidth(hair)
		c.beginPath()
		curve(c, place([]pt{{.41, side * .06}, {.5, side * .12}, {.52, side * .2}}, 0), false)
		c.stroke()
	}
	// Eyes sit at the front corners of the shell.
	for _, side := range []float64{-1, 1} {
		p := place([]pt{{.37, side * .245}}, 0)[0]
		x, y := p[0], p[1]
		c.setFill(col("#120d0a", .95))
		dot(c, x, y-.012, .02)
		c.setFill(rgbaC(255, 255, 255, .6))
		dot(c, x+.006, y-.018, .007)
	}
	shrimpLegs(c, flat, pose.near, col(C["leg"], .9, -.1), hair*1.5)
}

type pile struct{ x, z, life float64 }

// shrimp walk on legs that keep their footing, sway their antennae, beat
// their swimmerets, and flip backward with their tails when startled.
type shrimp struct {
	crawler
	curl, dart, hop, whip, paddle, work, open, snap, buried float64
	carry                                                   float64
	piles                                                   []pile
}

func newShrimp(spec *Spec, w *world, enter bool) *shrimp {
	s := &shrimp{}
	s.crawler.init(s, spec, w, enter)
	shape := shrimpShapes[shrimpLooks[spec.ID].shape]
	var legs []legSpec
	for i, dx := range shape.rests {
		legs = append(legs, legSpec{dx: dx, group: i % 2})
	}
	for i, dx := range shape.rests {
		legs = append(legs, legSpec{dx: dx - .05, group: 1 - i%2, dz: .06})
	}
	s.legs = newGait(legs, .22, .06, .16)
	s.whip = -.75
	if s.resting() {
		s.buried = 1
	}
	return s
}

func (s *shrimp) settle(w *world) {
	s.settleBase(w)
	if s.spec.Burrow {
		b := w.burrows[s.spec.ID]
		s.home = spot{b.x, b.z}
		s.den = s.home
		s.depths = [2]float64{math.Max(0, b.z-.1), b.z + .1}
		s.hides = true
		s.burrow = b
	}
}

func (s *shrimp) wander(w *world) *goal { return s.wanderBase(w) }

// The slipper lobster digs in where it rests instead of hiding.
func (s *shrimp) resting() bool { return s.state == "in" && !s.hides }

func (s *shrimp) setActive(on bool, w *world, instant bool) {
	s.setActiveBase(on, w, instant)
	if instant {
		s.buried = 0
		if s.resting() {
			s.buried = 1
		}
	}
}

func (s *shrimp) startle(w *world) {
	// Burrowers and the shy fire shrimp hide for a while; the others flip away.
	if s.spec.Burrow || s.spec.ID == "fire" {
		s.hideAway(8 + s.rand.next()*10)
		return
	}
	s.dart = 1
	s.busy = true
	s.turn = 0
	s.goal = nil
	s.until = w.time + 3
	s.vx = -s.facing * s.size(w) * 7
}

func (s *shrimp) step(dt float64, w *world) {
	L := s.size(w)
	if s.dart > 0 {
		// Escape flip: the tail snaps under and the body darts backward.
		s.dart = math.Max(0, s.dart-dt*2.2)
		s.busy = s.dart > 0
		if !s.busy {
			s.vx = 0
		}
		s.curl = math.Sin(math.Pi * math.Min(1, (1-s.dart)*1.6))
		s.hop = math.Sin(math.Pi*(1-s.dart)) * L * .35
		s.vx = follow(s.vx, 0, 3, dt)
		s.x = clamp(s.x, w.width*.02, w.width*.98)
	} else {
		s.busy = false
		s.curl = follow(s.curl, 0, 6, dt)
		s.hop = 0
	}
	s.routine(dt, w)
	if s.dart > 0 {
		s.legs.place(s.x, s.legScale(w), s.footing(w))
	} else {
		s.legs.update(dt, s.x, s.legScale(w), s.footing(w), s.vx)
	}
	// Antennae trail the body a little and wave on their own; cleaners wave more.
	wave := .1
	if s.spec.ID == "cleaner" {
		wave = .22
	}
	s.whip = follow(s.whip, -.7+noise(s.seed, s.clock*.6)*wave-clamp(s.vx*s.facing/L, -1, 1)*.25, 5, dt)
	pr := 2.4
	if s.state == "in" {
		pr = 1.2
	}
	s.paddle += TAU * pr * dt
	if s.eating > 0 {
		s.work = .5 + math.Sin(s.clock*9)*.5
	} else {
		s.work = follow(s.work, 0, 4, dt)
	}
	rest := 0.0
	if s.resting() {
		rest = 1
	}
	s.buried = follow(s.buried, rest, 1.2, dt)
	if s.spec.ID == "pistol" {
		// It bulldozes sand out of its burrow: carried in front of the claws on
		// the way out, dropped in a little heap where it stops.
		b := s.burrow
		outward := s.state == "out" && s.goal != nil && s.goal.food == nil && math.Abs(s.goal.x-b.x) > math.Abs(s.x-b.x)
		if s.carry > .6 && !outward && s.state == "out" {
			s.piles = append(s.piles, pile{x: s.x + s.facing*L*.6, z: s.z, life: 1})
			s.carry = 0
		}
		ct := 0.0
		if outward {
			ct = 1
		}
		s.carry = follow(s.carry, ct, 2, dt)
		kept := s.piles[:0]
		for _, p := range s.piles {
			p.life -= dt / 12
			if p.life > 0 {
				kept = append(kept, p)
			}
		}
		s.piles = kept
		if s.state == "out" && s.goal == nil && s.rand.next() < dt*.08 {
			s.snap = 1
		}
		s.snap = math.Max(0, s.snap-dt*2.5)
		if s.snap > .85 {
			s.open = 1
		} else {
			to := .35
			if s.snap > 0 {
				to = 0
			}
			s.open = follow(s.open, to, 12, dt)
		}
	}
}

func (s *shrimp) under(c *ctx, w *world) {
	if s.burrow == nil {
		return
	}
	b := s.burrow
	r := s.spec.Length * w.px * .55
	hole(c, b.x, w.ground(b.x, b.z), r)
	for _, p := range s.piles {
		c.setFill(tone("#8f9d86", p.z*.5, p.life, 0))
		oval(c, p.x, w.ground(p.x, p.z), r*.5, r*.2, 0)
	}
}

func (s *shrimp) draw(c *ctx, w *world) {
	if s.hidden() && s.burrow == nil {
		return
	}
	L, haze := s.size(w), s.z*.5
	lift := s.hop + s.legs.airborne()*L*.015
	sink := 0.0
	if s.burrow != nil {
		// Going home means backing down into the burrow, a little at a time.
		b := s.burrow
		near := 1 - clamp(math.Abs(s.x-b.x)/(L*.6), 0, 1)
		switch s.state {
		case "in":
			sink = 1 - .35*math.Max(0, noise(s.seed+4, s.clock*.2))
		case "leaving":
			sink = near
		}
		if sink > .98 {
			return
		}
	}
	if s.buried > .01 {
		sink = math.Max(sink, s.buried*.5)
	}
	shadow(c, s.x, s.y+1, L*.45, L*.1, .3*(1-sink)*s.shown*(1-haze))
	c.save()
	c.setAlpha(s.shown)
	c.translate(s.x, s.y-lift+sink*L*.32)
	if sink > 0 {
		c.beginPath()
		c.rect(-L*3, -L*3, L*6, L*3-sink*L*.32+1)
		c.clip()
	}
	c.scale(s.flipX()*L, L)
	feet := s.localFeet(L, lift)
	if s.eating > 0 || s.work > .05 {
		m := shrimpShapes[shrimpLooks[s.spec.ID].shape].mouth
		feet[0] = pt{lerp(feet[0][0], m[0]+.04, s.work), lerp(feet[0][1], m[1]+.02, s.work)}
	}
	paintShrimp(c, s.spec.ID, &shrimpPose{near: feet[:5], far: feet[5:], curl: s.curl, whip: s.whip, whip2: s.whip - .12, paddle: s.paddle, work: s.work, open: s.open, haze: haze}, haze, L)
	c.restore()
	if s.carry > .05 {
		c.setFill(tone("#8f9d86", haze, s.shown, 0))
		oval(c, s.x+s.facing*L*.62, s.y-L*.035, L*.1*s.carry, L*.055*s.carry, 0)
	}
	if s.snap > .6 {
		c.setFill(rgbaC(235, 250, 255, (s.snap-.6)*2))
		c.beginPath()
		c.arc(s.x+s.facing*L*.7, s.y-L*.16, L*.08+(1-s.snap)*L*.3, 0, TAU, false)
		c.fill()
	}
	if s.burrow != nil {
		b := s.burrow
		r := s.spec.Length * w.px * .55
		lip(c, b.x, w.ground(b.x, b.z), r)
	}
	if s.buried > .01 {
		c.setFill(tone("#718371", haze, s.buried, 0))
		c.beginPath()
		c.ellipse(s.x, s.y+1, L*.6, L*.08, 0, 0, TAU, false)
		c.fill()
	}
}

func shrimpPortrait(c *ctx, spec *Spec, w, h float64) {
	look := shrimpLooks[spec.ID]
	shape := shrimpShapes[look.shape]
	reach := .55
	if look.whip != 0 {
		reach = .55 + look.whip*.15
	}
	L := math.Min(w*.92/(.55+reach), h*.82/.42)
	c.save()
	c.translate(w/2-(reach-.55)/2*L, h*.82)
	c.scale(L, L)
	var near, far []pt
	for _, dx := range shape.rests {
		near = append(near, pt{dx, 0})
		far = append(far, pt{dx - .05, 0 - .015})
	}
	paintShrimp(c, spec.ID, &shrimpPose{near: near, far: far, curl: 0, whip: -.62, whip2: -.75, paddle: 1, work: 0, open: .3, haze: 0}, 0, L)
	c.restore()
}
