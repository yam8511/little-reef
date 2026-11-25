package reef

// The hermit crab, ported from engine/hermit.js.
//
// It is drawn facing right in units of its length, standing on the ground
// at y = 0. Its borrowed snail shell rides on its back with the opening
// toward the front; legs, claws and eyes come out of the opening and pull
// back inside when it hides.

import "math"

var hermitC = map[string]string{"shell": "#cdb495", "band": "#8c6a4c", "lip": "#e8d9c0", "inside": "#2a1c16", "leg": "#c3302a", "joint": "#e6a24a", "eye": "#e5c34b"}
var hermitHips = []pt{{.22, -.14}, {.15, -.12}}
var hermitRests = []float64{.42, .27}

type hermitPose struct {
	near, far               []pt
	out, whip, work, wobble float64
}

// paintHermit draws the crab for a pose: out is how far it has come out of
// its shell (0 inside … 1 out), feet are in unit coordinates.
func paintHermit(c *ctx, pose *hermitPose, haze, L float64) {
	C := hermitC
	col, hair, out := shade(haze), 1/L, pose.out
	// Far legs behind the shell.
	if out > .05 {
		for i, f := range pose.far {
			h := hermitHips[i]
			k := knee(h[0]-.02, h[1]-.01, f[0], f[1], .15*out+.02, .17*out+.02)
			limb(c, []pt{{h[0] - .02, h[1] - .01}, k, f}, math.Max(.04, hair), math.Max(.022, hair*.8), col(C["leg"], .9, -.35))
		}
	}
	// The shell: a turban of whorls rising up and back from the opening.
	c.save()
	c.rotate(pose.wobble)
	whorls := [][3]float64{{-.04, -.24, .24}, {-.24, -.33, .165}, {-.37, -.39, .11}, {-.46, -.43, .065}}
	for i := len(whorls) - 1; i >= 0; i-- {
		x, y, r := whorls[i][0], whorls[i][1], whorls[i][2]
		g := c.createRadialGradient(x+r*.3, y-r*.45, r*.1, x, y, r*1.05)
		g.addColorStop(0, col(C["lip"], 1))
		g.addColorStop(.55, col(C["shell"], 1))
		g.addColorStop(1, col(C["band"], 1, -.2))
		c.setFill(g)
		oval(c, x, y, r*1.05, r*.92, -.5)
		c.setStroke(col(C["band"], .65))
		c.setLineWidth(hair * 1.2)
		for k := 1; k <= 3; k++ {
			c.beginPath()
			c.ellipse(x, y, r*(1.05-float64(k)*.2), r*(.92-float64(k)*.18), -.5, math.Pi*.9, math.Pi*1.9, false)
			c.stroke()
		}
		if i == 0 {
			c.setFill(col(C["band"], .5))
			for k := 0; k < 6; k++ {
				oval(c, x-r*.55+float64(k)*r*.2, y+r*.35-float64(k)*r*.12, r*.06, r*.04, -.5)
			}
		}
	}
	// The opening faces forward and down; the crab comes out of it.
	c.setFill(col(C["lip"]))
	oval(c, .15, -.15, .085, .1, .45)
	c.setFill(col(C["inside"]))
	oval(c, .16, -.145, .062, .075, .45)
	c.restore()
	if out < .03 {
		return
	}
	c.save()
	c.translate(.16, -.15)
	c.scale(out, out)
	c.translate(-.16, .15)
	// Antennae and eyes on long yellow stalks.
	c.setStroke(col(C["leg"], .9))
	c.setLineWidth(math.Max(.008, hair*.8))
	for _, u := range [][2]float64{{pose.whip, .55}, {pose.whip - .25, .48}} {
		a, ln := u[0], u[1]
		c.beginPath()
		c.moveTo(.26, -.24)
		c.quadraticCurveTo(.26+math.Cos(a-.4)*ln*.5, -.24+math.Sin(a-.4)*ln*.5, .26+math.Cos(a)*ln, -.24+math.Sin(a)*ln*.7)
		c.stroke()
	}
	for _, dx := range []float64{0, .02} {
		light := 0.0
		if dx != 0 {
			light = -.2
		}
		c.setStroke(col(C["eye"], 1, light))
		c.setLineWidth(math.Max(.024, hair))
		c.roundCaps()
		c.beginPath()
		c.moveTo(.2+dx, -.2)
		c.lineTo(.27+dx, -.33)
		c.stroke()
		c.setFill(col("#0c0b0a"))
		dot(c, .275+dx, -.345, .026)
		c.setFill(rgbaC(255, 255, 255, .7*(1-haze)))
		dot(c, .282+dx, -.352, .008)
	}
	// Near legs, banded at the joints.
	for i, f := range pose.near {
		h := hermitHips[i]
		k := knee(h[0], h[1], f[0], f[1], .15, .17)
		limb(c, []pt{h, k, f}, math.Max(.05, hair*1.2), math.Max(.026, hair*.9), col(C["leg"]))
		c.setFill(col(C["joint"], .9))
		dot(c, k[0], k[1], .02)
		c.setFill(col("#f2e6d6", .8))
		dot(c, f[0], f[1]-.01, .012)
	}
	// Two hairy claws held in front; one reaches down while it eats.
	for _, u := range [][3]float64{{.02, .02, 0}, {0, 0, pose.work}} {
		dx, dy, reach := u[0], u[1], u[2]
		bx, by := .22+dx, -.16+dy
		tx, ty := .33+dx-reach*.02, -.11+dy+reach*.08
		l1, l2, l3 := 0.0, .15, -.1
		if dx != 0 {
			l1, l2, l3 = -.25, -.15, -.35
		}
		limb(c, []pt{{bx, by}, {tx, ty}}, math.Max(.07, hair*1.5), math.Max(.06, hair*1.4), col(C["leg"], 1, l1))
		g := c.createLinearGradient(0, ty-.05, 0, ty+.05)
		g.addColorStop(0, col(C["leg"], 1, l2))
		g.addColorStop(1, col(C["leg"], 1, l3))
		c.setFill(g)
		oval(c, tx+.06, ty+.015, .085, .05, .3)
		c.setFill(col("#f2e6d6", .95))
		oval(c, tx+.135, ty+.045, .026, .016, .3)
		c.setFill(col("#f2e6d6", .6))
		for k := 0; k < 4; k++ {
			dot(c, tx+.01+float64(k)*.025, ty-.02+float64(k)*.006, .007)
		}
	}
	c.restore()
}

// hermit crabs wander the sand with their shells rocking, and pull back
// inside when a fish comes close or when they rest.
type hermit struct {
	crawler
	out, whip, work, shy float64
}

func newHermit(spec *Spec, w *world, enter bool) *hermit {
	h := &hermit{}
	h.crawler.init(h, spec, w, enter)
	var legs []legSpec
	for i, dx := range hermitRests {
		legs = append(legs, legSpec{dx: dx, group: i})
	}
	for i, dx := range hermitRests {
		legs = append(legs, legSpec{dx: dx - .06, group: 1 - i, dz: .05})
	}
	h.legs = newGait(legs, .2, .07, .22)
	h.out = 1
	if h.state == "in" {
		h.out = 0
	}
	h.whip = -.6
	return h
}

func (h *hermit) settle(w *world)       { h.settleBase(w) }
func (h *hermit) wander(w *world) *goal { return h.wanderBase(w) }

func (h *hermit) setActive(on bool, w *world, instant bool) {
	h.setActiveBase(on, w, instant)
	if instant {
		h.out = 0
		if on {
			h.out = 1
		}
	}
}

func (h *hermit) startle(w *world) {
	h.shy = 3 + h.rand.next()*3
	h.goal = nil
}

func (h *hermit) under(*ctx, *world) {}

func (h *hermit) step(dt float64, w *world) {
	L := h.size(w)
	if h.shy > 0 {
		h.shy -= dt
		h.busy = true
		h.vx = follow(h.vx, 0, 8, dt)
	} else {
		h.busy = false
	}
	h.routine(dt, w)
	h.legs.update(dt, h.x, h.legScale(w), h.footing(w), h.vx)
	inside := h.shy > 0 || h.state == "in"
	if inside {
		h.out = follow(h.out, 0, 9, dt)
	} else {
		h.out = follow(h.out, 1, 2.5, dt)
	}
	h.whip = follow(h.whip, -.55+noise(h.seed, h.clock*.7)*.25-clamp(h.vx*h.facing/L, -1, 1)*.25, 5, dt)
	if h.eating > 0 {
		h.work = .5 + math.Sin(h.clock*8)*.5
	} else {
		h.work = follow(h.work, 0, 4, dt)
	}
}

func (h *hermit) draw(c *ctx, w *world) {
	L, haze := h.size(w), h.z*.5
	shadow(c, h.x-h.facing*L*.1, h.y+1, L*.45, L*.1, .32*(1-haze))
	lift := (1-h.out)*-L*.02 + h.legs.airborne()*L*.03
	c.save()
	c.translate(h.x, h.y-lift)
	c.scale(h.flipX()*L, L)
	feet := h.localFeet(L, lift)
	wobble := math.Sin(h.clock*7) * .04 * clamp(math.Abs(h.vx)/(L*.3), 0, 1)
	paintHermit(c, &hermitPose{near: feet[:2], far: feet[2:], out: h.out, whip: h.whip, work: h.work, wobble: wobble}, haze, L)
	c.restore()
}

func hermitPortrait(c *ctx, spec *Spec, w, h float64) {
	L := math.Min(w*.9/1.1, h*.9/.7)
	c.save()
	c.translate(w/2+L*.05, h*.88)
	c.scale(L, L)
	var near, far []pt
	for _, x := range hermitRests {
		near = append(near, pt{x, 0})
		far = append(far, pt{x - .06, -.02})
	}
	paintHermit(c, &hermitPose{near: near, far: far, out: 1, whip: -.55, work: 0, wobble: 0}, 0, L)
	c.restore()
}
