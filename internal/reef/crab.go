package reef

// Crabs, ported from engine/crab.js.
//
// Crabs walk sideways, so we see them from the front and a little from
// above. Shells are laid out in plan view in units of carapace width — u
// across the body, w from front (+) to back (−) — and then foreshortened.
// The arrow crab is the exception: its long rostrum only shows from the
// side, so it is drawn side-on and walks forward (see paintArrow).

import "math"

const squash = .45

type crabClaw struct {
	hip, palm pt
	size      float64
	kind      string
}

type crabLook struct {
	plan        []pt
	stand, face float64
	hips        []pt
	span        []float64
	upper       float64
	lower       float64
	claw        crabClaw
	colors      map[string]string
}

var crabLooks = map[string]*crabLook{
	"emerald": {
		plan:  []pt{{0, .4}, {.12, .41}, {.25, .38}, {.37, .3}, {.45, .16}, {.48, 0}, {.45, -.16}, {.37, -.3}, {.23, -.4}, {0, -.43}},
		stand: .22, face: .1, hips: []pt{{.42, .2}, {.47, .04}, {.45, -.12}, {.39, -.27}}, span: []float64{.74, .84, .82, .7}, upper: .3, lower: .36,
		claw:   crabClaw{hip: pt{.3, .36}, palm: pt{.2, .5}, size: .15, kind: "spoon"},
		colors: map[string]string{"shell": "#2f8a52", "side": "#1f5f39", "leg": "#3d8a57", "claw": "#2f7f4c", "tip": "#ece6d2", "mark": "#5fb27c"},
	},
	"porcelain": {
		plan:  []pt{{0, .42}, {.15, .43}, {.3, .38}, {.42, .26}, {.48, .08}, {.47, -.1}, {.41, -.26}, {.28, -.38}, {.12, -.43}, {0, -.44}},
		stand: .17, face: .07, hips: []pt{{.43, .12}, {.46, -.06}, {.41, -.22}}, span: []float64{.7, .76, .66}, upper: .24, lower: .28,
		claw:   crabClaw{hip: pt{.36, .3}, palm: pt{.44, .62}, size: .2, kind: "flat"},
		colors: map[string]string{"shell": "#f3eee6", "side": "#b9ad9e", "leg": "#ece4d8", "claw": "#efe8dd", "tip": "#d9cbb9", "mark": "#a5442f"},
	},
	"swimmer": {
		plan:  []pt{{0, .3}, {.1, .31}, {.2, .28}, {.32, .22}, {.44, .13}, {.62, .07}, {.47, .02}, {.38, -.1}, {.26, -.22}, {.12, -.28}, {0, -.3}},
		stand: .2, face: .08, hips: []pt{{.33, .16}, {.36, .04}, {.33, -.08}, {.26, -.2}}, span: []float64{.62, .7, .68, .6}, upper: .27, lower: .33,
		claw:   crabClaw{hip: pt{.26, .25}, palm: pt{.5, .5}, size: .17, kind: "long"},
		colors: map[string]string{"shell": "#5b5a49", "side": "#3b3a2f", "leg": "#6e6a62", "claw": "#5e6688", "tip": "#e9e3d6", "mark": "#d9d6c8", "paddle": "#7d7a6a"},
	},
}

// at maps a plan point to the screen: the front edge sits lowest.
func (look *crabLook) at(u, w float64) pt {
	return pt{u, -(look.stand + look.face) - (.42-w)*squash}
}

type crabPose struct {
	feet                         []pt
	work, alarm, fan, stroke, lk float64
}

// paintCrab draws a crab facing us for a pose: feet in unit coordinates,
// the claws' work (0…1, alternating sides), raised claws when alarmed, and
// the paddles' stroke for a swimming crab.
func paintCrab(c *ctx, id string, pose *crabPose, haze, L float64) {
	look := crabLooks[id]
	C, hair, col := look.colors, 1/L, shade(haze)
	outline := append([]pt(nil), look.plan...)
	for i := len(look.plan) - 2; i >= 1; i-- {
		outline = append(outline, pt{-look.plan[i][0], look.plan[i][1]})
	}
	// Legs come out from under the shell's rim, back pairs first. Each has a
	// thick upper segment, a bend, and a pointed tip on the sand.
	thick := math.Max(.055, hair*1.4)
	for i := len(look.hips) - 1; i >= 0; i-- {
		for _, side := range []float64{-1, 1} {
			hu, hw := look.hips[i][0], look.hips[i][1]
			hp := look.at(hu*side, hw)
			fi := i * 2
			if side > 0 {
				fi++
			}
			foot := pose.feet[fi]
			paddle := id == "swimmer" && i == len(look.hips)-1
			ankle := pt{foot[0] - side*.03, foot[1] - .07}
			k := knee(hp[0], hp[1]+look.face*.6, ankle[0], ankle[1], look.upper, look.lower-.06)
			sh := (hw+.3)*.3 - .2
			end := foot
			if paddle {
				end = ankle
			}
			limb(c, []pt{{hp[0], hp[1] + look.face*.6}, k, ankle, end}, thick, thick*.45, col(C["leg"], 1, sh))
			c.setFill(col(C["leg"], 1, sh-.15))
			dot(c, k[0], k[1], thick*.55)
			if paddle {
				c.save()
				c.translate(ankle[0], ankle[1])
				c.rotate(side * (.5 + math.Sin(pose.stroke)*.7))
				c.setFill(col(C["paddle"], .95, .05))
				oval(c, side*.08, .01, .1, .045, 0)
				c.restore()
			}
		}
	}
	// Shell: its side shows as a darker band below the top.
	placed := func(dy float64) []pt {
		out := make([]pt, len(outline))
		for i, p := range outline {
			q := look.at(p[0], p[1])
			out[i] = pt{q[0], q[1] + dy}
		}
		return out
	}
	rim := curvePath(wind(placed(look.face)), true)
	c.setFill(col(C["side"]))
	c.fillPath(rim)
	top := curvePath(wind(placed(0)), true)
	y0, y1 := look.at(0, -.45)[1], look.at(0, .45)[1]
	g := c.createLinearGradient(0, y0, 0, y1)
	g.addColorStop(0, col(C["shell"], 1, .14))
	g.addColorStop(1, col(C["shell"], 1, -.1))
	c.setFill(g)
	c.fillPath(top)
	c.save()
	c.clipPath(top)
	if id == "emerald" {
		c.setFill(col(C["mark"], .55))
		for i := 0; i < 26; i++ {
			u, w := float64((i*47)%89)/89-.5, float64((i*31)%67)/67-.5
			p := look.at(u*.85, w*.8)
			dot(c, p[0], p[1], .016)
		}
	}
	if id == "porcelain" {
		c.setFill(col(C["mark"], .9))
		for i := 0; i < 22; i++ {
			u, w := float64((i*53)%97)/97-.5, float64((i*29)%61)/61-.5
			p := look.at(u*.85, w*.85)
			oval(c, p[0], p[1], .03, .022, 0)
		}
	}
	if id == "swimmer" {
		c.setFill(col(C["mark"], .55))
		for i := 0; i < 20; i++ {
			u, w := float64((i*41)%83)/83-.5, float64((i*23)%59)/59-.5
			p := look.at(u, w*.7)
			oval(c, p[0], p[1], .025, .015, 0)
		}
		c.setStroke(col(C["side"], .6))
		c.setLineWidth(hair)
		c.beginPath()
		curve(c, []pt{look.at(-.25, .1), look.at(0, -.05), look.at(.25, .1)}, false)
		c.stroke()
	}
	l0, l1 := look.at(-.1, -.1), look.at(0, 0)
	light := c.createRadialGradient(l0[0], l0[1], .02, l1[0], l1[1], .5)
	light.addColorStop(0, rgbaC(255, 255, 255, .28*(1-haze)))
	light.addColorStop(1, rgbaC(255, 255, 255, 0))
	c.setFill(light)
	c.fillRect(-.7, y0-.1, 1.4, y1-y0+.2)
	c.restore()
	// Mouthparts; the porcelain crab sweeps the water with two feathery fans.
	front := look.at(0, .42)[1]
	c.setFill(col(C["side"], 1, -.25))
	oval(c, 0, front+look.face*.55, .09, look.face*.35, 0)
	if id == "porcelain" {
		for _, side := range []float64{-1, 1} {
			ph := 0.0
			if side > 0 {
				ph = math.Pi
			}
			sweep := math.Max(0, math.Sin(pose.fan+ph))
			c.save()
			c.translate(side*.03, front+look.face*.4)
			c.rotate(side * (.3 + sweep*.9))
			c.setStroke(col("#efe6d8", .75))
			c.setLineWidth(hair * .8)
			for k := -3; k <= 3; k++ {
				c.beginPath()
				c.moveTo(0, 0)
				c.lineTo(side*.05+float64(k)*.015, -.14-sweep*.04+math.Abs(float64(k))*.01)
				c.stroke()
			}
			c.restore()
		}
	}
	// Eyes on short stalks at the front edge.
	for _, side := range []float64{-1, 1} {
		e := look.at(side*.1, .41)
		ex, ey := e[0], e[1]
		tilt := pose.lk * .1 * side
		c.setStroke(col(C["shell"], 1, -.1))
		c.setLineWidth(hair * 2.6)
		c.roundCaps()
		c.beginPath()
		c.moveTo(ex, ey+.01)
		c.lineTo(ex+side*.025+tilt, ey-.06)
		c.stroke()
		c.setFill(col("#0d0b0a"))
		dot(c, ex+side*.028+tilt, ey-.07, .028)
		c.setFill(rgbaC(255, 255, 255, .7*(1-haze)))
		dot(c, ex+side*.028+tilt+.008, ey-.078, .009)
	}
	// Claws in front: each in turn reaches down and brings food to the mouth;
	// alarmed, both rise.
	for _, side := range []float64{-1, 1} {
		cl := look.claw
		hp := look.at(cl.hip[0]*side, cl.hip[1])
		hx, hy := hp[0], hp[1]
		work := math.Max(0, -math.Sin(pose.work))
		if side < 0 {
			work = math.Max(0, math.Sin(pose.work))
		}
		raise, arm := pose.alarm, math.Max(.07, hair*2)
		px, py := cl.palm[0]*side, front+cl.palm[1]*.4
		px = lerp(px, side*.06, work*.7)
		py = lerp(py, front+look.face*.5, work*.6) - work*.02
		px += side * raise * .12
		py -= raise * .3
		kx := lerp(hx, px, .5) + side*.1*(1-work)
		ky := lerp(hy, py, .5) + .05 - raise*.1
		limb(c, []pt{{hx, hy + look.face*.5}, {kx, ky}, {px, py}}, arm, arm*.85, col(C["claw"], 1, -.08))
		c.save()
		c.translate(px, py)
		turn := .5
		if cl.kind == "long" {
			turn = 1.1
		}
		c.rotate(side*turn - raise*side*.8)
		c.scale(-side, 1)
		// The claw points inward (+x here): a palm, then a fixed and a moving finger.
		s, long, flat := cl.size, cl.kind == "long", cl.kind == "flat"
		pl, pw := s*.8, s*.45
		if long {
			pl, pw = s*1.6, s*.32
		} else if flat {
			pl, pw = s*1, s*.52
		}
		gc := c.createLinearGradient(0, -pw, 0, pw)
		gc.addColorStop(0, col(C["claw"], 1, .2))
		gc.addColorStop(1, col(C["claw"], 1, -.15))
		c.setFill(gc)
		oval(c, pl*.45, 0, pl*.55, pw, 0)
		if flat {
			c.setFill(col(C["mark"], .85))
			for _, d := range []pt{{.25, -.3}, {.5, .2}, {.7, -.15}, {.4, .45}} {
				dot(c, pl*d[0], pw*d[1], s*.1)
			}
		}
		if long {
			c.setFill(col(C["tip"], .8))
			for k := 0; k < 4; k++ {
				fk := float64(k)
				c.beginPath()
				c.moveTo(pl*(.15+fk*.2), -pw*.9)
				c.lineTo(pl*(.2+fk*.2), -pw*1.5)
				c.lineTo(pl*(.25+fk*.2), -pw*.9)
				c.fill()
			}
		}
		fx, gap := pl*.95, .25+work*.2
		c.setFill(col(C["claw"], 1, -.18))
		c.beginPath()
		c.moveTo(fx, -pw*.5)
		c.quadraticCurveTo(fx+s*.55, -pw*(.6+gap), fx+s*.75, -pw*.05)
		c.quadraticCurveTo(fx+s*.4, -pw*.15, fx, pw*.05)
		c.fill()
		c.beginPath()
		c.moveTo(fx, pw*.55)
		c.quadraticCurveTo(fx+s*.5, pw*(.6+gap*.5), fx+s*.7, pw*.1)
		c.quadraticCurveTo(fx+s*.35, pw*.2, fx, pw*.1)
		c.fill()
		c.setFill(col(C["tip"], .95))
		if cl.kind == "spoon" {
			oval(c, fx+s*.7, -pw*.08, s*.13, s*.09, 0)
			oval(c, fx+s*.65, pw*.14, s*.12, s*.08, 0)
		} else {
			dot(c, fx+s*.72, -pw*.08, s*.05)
			dot(c, fx+s*.66, pw*.12, s*.045)
		}
		c.restore()
	}
}

type arrowPose struct {
	near, far   []pt
	work, alarm float64
}

// paintArrow draws the arrow crab side-on, facing right: a small striped
// triangle of a body slung between long, thin legs, a long pointed rostrum
// and slender claws with violet tips. Units are leg span.
func paintArrow(c *ctx, pose *arrowPose, haze, L float64) {
	col, hair := shade(haze), 1/L
	body := []pt{{.12, -.42}, {-.12, -.49}, {-.17, -.42}, {-.12, -.36}, {.06, -.35}}
	hips := []pt{{.06, -.37}, {.02, -.37}, {-.03, -.38}, {-.08, -.39}}
	leg := func(feet []pt, color rgba, width float64) {
		for i, f := range feet {
			h := hips[i]
			k := knee(h[0], h[1], f[0], f[1], .3, .36)
			limb(c, []pt{h, k, f}, width, width*.55, color)
			c.setFill(color)
			dot(c, k[0], k[1], width*.7)
		}
	}
	leg(pose.far, col("#c9a46d", .75, -.35), math.Max(.012, hair*.9))
	c.setFill(col("#c39a5f"))
	c.fillPath(curvePath(body, true))
	c.save()
	c.clipPath(curvePath(body, true))
	c.setStroke(col("#3c2a1b", .8))
	c.setLineWidth(hair * 1.1)
	for k := 0; k < 5; k++ {
		fk := float64(k)
		c.beginPath()
		c.moveTo(.12, -.425+fk*.012)
		c.lineTo(-.18, -.5+fk*.03)
		c.stroke()
	}
	c.setStroke(col("#f0d9a0", .7))
	for k := 0; k < 4; k++ {
		fk := float64(k)
		c.beginPath()
		c.moveTo(.12, -.418+fk*.012)
		c.lineTo(-.18, -.49+fk*.03)
		c.stroke()
	}
	c.restore()
	// The rostrum points forward and a little up, like an arrow.
	c.setFill(col("#b98c52"))
	c.beginPath()
	c.moveTo(.1, -.44)
	c.lineTo(.5, -.53)
	c.lineTo(.1, -.415)
	c.fill()
	c.setStroke(col("#3c2a1b", .6))
	c.setLineWidth(hair * .8)
	c.beginPath()
	c.moveTo(.12, -.43)
	c.lineTo(.48, -.525)
	c.stroke()
	c.setFill(col("#120e0b"))
	dot(c, .09, -.455, .016)
	// Slender claws reach forward and down; they pick at the bottom while eating.
	reach := pose.work
	for _, u := range [][2]float64{{0, -.3}, {.01, 0}} {
		dy, light := u[0], u[1]
		tip := pt{.36 - reach*.06 + pose.alarm*.05, -.18 + dy + reach*.14 - pose.alarm*.2}
		limb(c, []pt{{.08, -.37 + dy}, {.22, -.33 + dy}, tip}, math.Max(.018, hair), math.Max(.014, hair*.9), col("#c9a46d", 1, light))
		c.setFill(col("#7d5fb8", .95, light))
		oval(c, tip[0]+.03, tip[1]+.01, .035, .012, .5)
	}
	leg(pose.near, col("#d8b47c", .95), math.Max(.016, hair*1.1))
}

// crab: crabs sidestep on alternating legs, pick at the bottom with their
// claws, raise them when a fish comes close, and (the swimming crab) paddle
// off the sand for a short swim or dig in to rest.
type crab struct {
	crawler
	arrow                                        bool
	work, alarm, fan, stroke, swim, rise, buried float64
	lk                                           float64
}

func newCrab(spec *Spec, w *world, enter bool) *crab {
	k := &crab{}
	k.crawler.init(k, spec, w, enter)
	k.arrow = spec.ID == "arrow"
	if k.arrow {
		rests := []float64{.42, .22, -.08, -.32}
		var legs []legSpec
		for i, dx := range rests {
			legs = append(legs, legSpec{dx: dx, group: i % 2})
		}
		for i, dx := range rests {
			legs = append(legs, legSpec{dx: dx + .05, group: 1 - i%2, dz: .05})
		}
		k.legs = newGait(legs, .2, .1, .3)
	} else {
		look := crabLooks[spec.ID]
		var legs []legSpec
		for i, h := range look.hips {
			for _, side := range []float64{-1, 1} {
				g := i % 2
				if side > 0 {
					g = (i + 1) % 2
				}
				legs = append(legs, legSpec{dx: side * look.span[i], group: g, dz: (.2 - h[1]) * .08})
			}
		}
		k.legs = newGait(legs, .2, .1, .2)
	}
	if k.resting() && spec.ID == "swimmer" {
		k.buried = 1
	}
	return k
}

func (k *crab) resting() bool      { return k.state == "in" && !k.hides }
func (k *crab) settle(w *world)    { k.settleBase(w) }
func (k *crab) under(*ctx, *world) {}

func (k *crab) setActive(on bool, w *world, instant bool) {
	k.setActiveBase(on, w, instant)
	if instant {
		k.buried = 0
		if k.resting() && k.spec.ID == "swimmer" {
			k.buried = 1
		}
	}
}

func (k *crab) startle(w *world) {
	k.alarm = 1
	k.goal = nil
	k.until = w.time + 3
}

func (k *crab) wander(w *world) *goal {
	// Now and then the swimming crab paddles off instead of walking.
	if k.spec.ID == "swimmer" && k.rand.next() < .35 {
		k.swim = 1.6 + k.rand.next()
	}
	return k.wanderBase(w)
}

func (k *crab) step(dt float64, w *world) {
	L := k.size(w)
	if k.swim > 0 {
		k.swim -= dt
	}
	swimming := k.swim > 0 && k.state == "out"
	f := 1.0
	if swimming {
		f = 2.2
	}
	k.routine(dt*f, w)
	rise := 0.0
	if swimming {
		rise = L * .7
	}
	k.rise = follow(k.rise, rise, 3, dt)
	if k.rise > L*.05 {
		foot := k.footing(w)
		k.legs.place(k.x, k.legScale(w), func(x float64, l *leg) float64 { return foot(x, l) - k.rise })
	} else {
		k.legs.update(dt, k.x, k.legScale(w), k.footing(w), k.vx)
	}
	k.alarm = follow(k.alarm, 0, .7, dt)
	if k.eating > 0 || (k.state == "out" && k.goal == nil && k.spec.ID != "porcelain") {
		r := 2.5
		if k.eating > 0 {
			r = 7
		}
		k.work += dt * r
	}
	k.fan += dt * 6
	if swimming {
		k.stroke += dt * 14
	}
	k.lk = noise(k.seed+2, k.clock*.5)
	b := 0.0
	if k.resting() && k.spec.ID == "swimmer" {
		b = 1
	}
	k.buried = follow(k.buried, b, 1, dt)
}

func (k *crab) draw(c *ctx, w *world) {
	if k.hidden() {
		return
	}
	L, haze := k.size(w), k.z*.5
	c.save()
	c.setAlpha(k.shown)
	sr, sa := .55, .32
	if k.arrow {
		sr, sa = .35, .2
	}
	shadow(c, k.x, k.y+1, L*sr, L*.1, sa*(1-haze)/(1+k.rise/L*2))
	lift, sink := k.rise+k.legs.airborne()*L*.02, k.buried*L*.18
	c.translate(k.x, k.y-lift+sink)
	if sink > 0 {
		c.beginPath()
		c.rect(-L*2, -L*2, L*4, L*2-sink+1)
		c.clip()
	}
	if k.arrow {
		c.scale(k.flipX()*L, L)
		feet := k.localFeet(L, lift)
		work := 0.0
		if k.eating > 0 {
			work = .5 + math.Sin(k.clock*8)*.5
		}
		paintArrow(c, &arrowPose{near: feet[:4], far: feet[4:], work: work, alarm: k.alarm}, haze, L)
	} else {
		c.scale(L, L)
		feet := k.localFeet(L, lift-sink)
		paintCrab(c, k.spec.ID, &crabPose{feet: feet, work: k.work, alarm: k.alarm, fan: k.fan, stroke: k.stroke, lk: k.lk}, haze, L)
	}
	c.restore()
	if k.buried > .01 {
		c.setFill(tone("#718371", haze, k.buried, 0))
		c.beginPath()
		c.ellipse(k.x, k.y+1, L*.7, L*.1, 0, 0, TAU, false)
		c.fill()
	}
}

func crabPortrait(c *ctx, spec *Spec, w, h float64) {
	if spec.ID == "arrow" {
		L := math.Min(w*.9/1.15, h*.9/.6)
		c.save()
		c.translate(w/2-L*.08, h*.92)
		c.scale(L, L)
		var near, far []pt
		for _, x := range []float64{.42, .22, -.08, -.32} {
			near = append(near, pt{x, 0})
			far = append(far, pt{x + .05, -.02})
		}
		paintArrow(c, &arrowPose{near: near, far: far}, 0, L)
		c.restore()
		return
	}
	look := crabLooks[spec.ID]
	L := math.Min(w*.9/1.9, h*.88/.85)
	c.save()
	c.translate(w/2, h*.88)
	c.scale(L, L)
	var feet []pt
	for i, h := range look.hips {
		for _, side := range []float64{-1, 1} {
			feet = append(feet, pt{side * look.span[i], -(.2 - h[1]) * .08})
		}
	}
	paintCrab(c, spec.ID, &crabPose{feet: feet, fan: 1}, 0, L)
	c.restore()
}
