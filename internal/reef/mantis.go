package reef

// The peacock mantis shrimp, ported from engine/mantis.js.
//
// It is drawn facing right in units of body length, low on the ground at
// y = 0: a long flattened body, two big eyes on stalks that turn on their
// own, clubs folded under the head and a bright tail fan.

import "math"

var mantisC = map[string]string{"body": "#5d7f3e", "dark": "#3a5226", "head": "#8bab62", "spot": "#33461f", "leg": "#e8792f", "club": "#e0602f", "scale": "#e46a35", "edge": "#b0302a", "tail": "#4565c8", "rim": "#d24a32", "eye": "#43b4a4"}
var mantisRests = []float64{.2, .12, .04}

type mantisPose struct {
	near, far    []pt
	eyes         [2]float64
	strike, sway float64
}

// paintMantis draws the mantis shrimp for a pose: feet in unit coordinates,
// the eyes' angles, and strike (0…1) when its club snaps out for food.
func paintMantis(c *ctx, pose *mantisPose, haze, L float64) {
	C := mantisC
	col, hair := shade(haze), 1/L
	thin := math.Max(.014, hair*.9)
	for _, f := range pose.far {
		limb(c, []pt{{.1, -.05}, knee(.1, -.05, f[0], f[1], .045, .05), f}, thin, thin*.7, col(C["leg"], .7, -.35))
	}
	// Tail fan: blue blades edged in red.
	for _, u := range [][2]float64{{-.45, .12}, {.45, .12}, {-.2, .13}, {.2, .13}} {
		a, ln := u[0], u[1]
		c.save()
		c.translate(-.4, -.07)
		c.rotate(math.Pi + a*.6)
		c.setFill(col(C["rim"]))
		oval(c, ln*.5, 0, ln*.55, .032, 0)
		c.setFill(col(C["tail"]))
		oval(c, ln*.48, 0, ln*.46, .022, 0)
		c.restore()
	}
	c.setFill(col(C["body"], 1, .1))
	oval(c, -.43, -.07, .05, .035, 0)
	// Body: one long, flattened body crossed by segment lines, red spots along
	// its sides, then the spotted head shield.
	body := curvePath([]pt{{.22, -.125}, {.05, -.13}, {-.15, -.14}, {-.33, -.13}, {-.42, -.105}, {-.44, -.06}, {-.36, -.03}, {-.15, -.02}, {.05, -.022}, {.22, -.03}}, true)
	g := c.createLinearGradient(0, -.14, 0, -.02)
	g.addColorStop(0, col(C["body"], 1, .15))
	g.addColorStop(.6, col(C["body"]))
	g.addColorStop(1, col(C["dark"]))
	c.setFill(g)
	c.fillPath(body)
	c.save()
	c.clipPath(body)
	c.setStroke(col(C["dark"], .85))
	c.setLineWidth(math.Max(.008, hair))
	for i := 0; i < 8; i++ {
		x := -.36 + float64(i)*.075
		c.beginPath()
		c.moveTo(x+.012, -.15)
		c.quadraticCurveTo(x-.012, -.08, x+.01, -.01)
		c.stroke()
	}
	c.setFill(col(C["edge"], .75))
	for i := 0; i < 7; i++ {
		dot(c, -.33+float64(i)*.075, -.035, .009)
	}
	c.setFill(rgbaC(255, 255, 255, .18*(1-haze)))
	c.fillRect(-.45, -.15, .7, .03)
	c.restore()
	head := []pt{{.47, -.1}, {.42, -.15}, {.3, -.16}, {.2, -.14}, {.18, -.06}, {.3, -.04}, {.42, -.05}}
	hp := curvePath(head, true)
	c.setFill(col(C["head"]))
	c.fillPath(hp)
	c.save()
	c.clipPath(hp)
	c.setFill(col(C["spot"], .8))
	for _, d := range [][3]float64{{.24, -.12, .012}, {.29, -.09, .01}, {.34, -.13, .011}, {.38, -.08, .009}, {.27, -.06, .008}, {.41, -.12, .008}, {.21, -.09, .009}} {
		dot(c, d[0], d[1], d[2])
	}
	c.restore()
	// Antennal scales flare at the sides of the head.
	for _, tilt := range []float64{-.5, -.2} {
		c.save()
		c.translate(.42, -.09)
		c.rotate(tilt)
		c.setFill(col(C["edge"]))
		oval(c, .05, 0, .055, .022, 0)
		c.setFill(col(C["scale"]))
		oval(c, .048, -.002, .045, .015, 0)
		c.restore()
	}
	// The club folds under the head; it snaps forward to strike.
	strike := pose.strike
	reach := .06 + strike*.14
	limb(c, []pt{{.36, -.06}, {.42 + strike*.05, -.02}, {.36 + reach, -.035 + strike*.02}}, math.Max(.03, hair*1.2), math.Max(.026, hair), col(C["club"], 1, -.1))
	c.setFill(col(C["club"], 1, .08))
	oval(c, .37+reach, -.035+strike*.02, .03, .022, -.3+strike)
	c.setFill(col("#f1d9cf", .8))
	dot(c, .38+reach, -.04+strike*.02, .007)
	for _, f := range pose.near {
		limb(c, []pt{{.12, -.045}, knee(.12, -.045, f[0], f[1], .045, .05), f}, thin*1.2, thin*.8, col(C["leg"]))
	}
	// Antennules: three thin whips.
	c.setStroke(col(C["leg"], .8))
	c.setLineWidth(thin * .7)
	for _, a := range []float64{-.55, -.35, -.15} {
		c.beginPath()
		c.moveTo(.46, -.13)
		c.quadraticCurveTo(.56, -.18+a*.1, .7+a*.1, -.24+a*.25+pose.sway*.02)
		c.stroke()
	}
	// Eyes on stalks, each turning on its own; a band divides every eye.
	for i, turn := range pose.eyes {
		bx, by := .44-float64(i)*.03, -.15
		c.setStroke(col(C["head"], 1, -.1))
		c.setLineWidth(math.Max(.018, hair))
		c.beginPath()
		c.moveTo(bx, by)
		c.lineTo(bx+.02, by-.05)
		c.stroke()
		c.save()
		c.translate(bx+.022, by-.07)
		c.rotate(turn)
		light := 0.0
		if i != 0 {
			light = -.25
		}
		c.setFill(col(C["eye"], 1, light))
		oval(c, 0, 0, .03, .045, 0)
		c.setFill(col("#1e3b3a", .85))
		c.fillRect(-.03, -.006, .06, .012)
		c.setFill(rgbaC(255, 255, 255, .6*(1-haze)))
		dot(c, -.006, -.018, .006)
		c.restore()
	}
}

// mantis lives in a burrow: resting, it watches from the opening with only
// its eyes showing; out, it patrols nearby and strikes at food.
type mantis struct {
	crawler
	eyes                     [2]float64
	strike, peek, sway, rise float64
}

func newMantis(spec *Spec, w *world, enter bool) *mantis {
	m := &mantis{}
	m.crawler.init(m, spec, w, enter)
	var legs []legSpec
	for i, dx := range mantisRests {
		legs = append(legs, legSpec{dx: dx, group: i % 2})
	}
	for i, dx := range mantisRests {
		legs = append(legs, legSpec{dx: dx - .03, group: 1 - i%2, dz: .04})
	}
	m.legs = newGait(legs, .12, .03, .16)
	if m.state == "in" {
		m.peek = 1
	} else {
		m.rise = 1
	}
	return m
}

func (m *mantis) settle(w *world) {
	b := w.burrows["mantis"]
	m.burrow = b
	m.home = spot{b.x, b.z}
	m.den = m.home
	m.depths = [2]float64{math.Max(0, b.z-.08), b.z + .08}
	m.hides = true
}

func (m *mantis) wander(w *world) *goal { return m.wanderBase(w) }

func (m *mantis) startle(*world) { m.hideAway(10 + m.rand.next()*10) }

func (m *mantis) setActive(on bool, w *world, instant bool) {
	m.setActiveBase(on, w, instant)
	if instant {
		if on {
			m.rise, m.peek = 1, 0
		} else {
			m.rise, m.peek = 0, 1
		}
	}
}

func (m *mantis) step(dt float64, w *world) {
	// Sink before peeking; tuck the eyes before rising, and wait to walk until mostly out.
	m.busy = m.state == "out" && m.rise < .8
	if m.busy {
		m.vx = 0
	}
	m.routine(dt, w)
	near := 1 - clamp(math.Abs(m.x-m.burrow.x)/(m.size(w)*.5), 0, 1)
	target := 1.0
	switch {
	case m.state == "in" || m.peek >= .05:
		target = 0
	case m.state == "leaving":
		target = 1 - near
	}
	m.rise = follow(m.rise, target, 4, dt)
	m.legs.update(dt, m.x, m.legScale(w), m.footing(w), m.vx)
	t := m.clock
	m.eyes = [2]float64{noise(m.seed, t*.7) * 1.2, noise(m.seed+1, t*.6+3) * 1.2}
	if m.eating > 0 {
		m.strike = math.Pow(math.Max(0, math.Sin(t*5)), 6)
	} else {
		m.strike = follow(m.strike, 0, 10, dt)
	}
	if m.state == "in" && m.rise <= .02 {
		p := 0.0
		if noise(m.seed+5, t*.08) > -.35 {
			p = 1
		}
		m.peek = follow(m.peek, p, 1.5, dt)
	} else {
		m.peek = follow(m.peek, 0, 6, dt)
	}
	m.sway = math.Sin(t * 2)
}

func (m *mantis) under(c *ctx, w *world) {
	b := m.burrow
	hole(c, b.x, w.ground(b.x, b.z), m.spec.Length*w.px*.2)
}

func (m *mantis) draw(c *ctx, w *world) {
	L, haze, b := m.size(w), m.z*.5, m.burrow
	by, r := w.ground(b.x, b.z), m.spec.Length*w.px*.2
	pose := &mantisPose{eyes: m.eyes, strike: m.strike, sway: m.sway}
	if m.rise > .02 {
		sink := 1 - m.rise
		shadow(c, m.x, m.y+1, L*.5, L*.07, .3*(1-haze)*(1-sink))
		c.save()
		if sink > 0 {
			c.beginPath()
			c.rect(m.x-L, m.y-L, L*2, L+1)
			c.clip()
		}
		c.translate(m.x, m.y+sink*L*.5)
		c.scale(m.flipX()*L, L)
		feet := m.localFeet(L, 0)
		pose.near, pose.far = feet[:3], feet[3:]
		paintMantis(c, pose, haze, L)
		c.restore()
	} else {
		// Watching from the burrow: head up, the rest hidden below the rim.
		if m.peek < .05 {
			lip(c, b.x, by, r)
			return
		}
		c.save()
		c.beginPath()
		c.rect(b.x-L, by-L, L*2, L-r*.15)
		c.clip()
		c.translate(b.x-m.facing*L*.1, by+L*(.62-m.peek*.28))
		c.scale(m.facing*L, L)
		c.rotate(-1.3)
		paintMantis(c, pose, haze, L)
		c.restore()
	}
	lip(c, b.x, by, r)
}

func mantisPortrait(c *ctx, spec *Spec, w, h float64) {
	L := math.Min(w*.9/1.15, h*.85/.38)
	c.save()
	c.translate(w/2-L*.1, h*.75)
	c.scale(L, L)
	var near, far []pt
	for _, x := range mantisRests {
		near = append(near, pt{x, 0})
		far = append(far, pt{x - .03, -.01})
	}
	paintMantis(c, &mantisPose{near: near, far: far, eyes: [2]float64{.3, -.4}}, 0, L)
	c.restore()
}
