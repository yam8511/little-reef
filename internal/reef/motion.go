package reef

// Motion shared by the creature models: seeded randomness, smooth noise,
// limbs, legs that keep their feet planted, and the crawlers' daily routine.
// Ported from engine/motion.js.

import "math"

const TAU = math.Pi * 2

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func lerp(a, b, t float64) float64    { return a + (b-a)*t }
func smooth(t float64) float64        { return t * t * (3 - 2*t) }

// sign is Math.sign.
func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return v // 0, -0 or NaN
}

// or is JS's a || b for numbers: b when a is 0 or NaN.
func or(a, b float64) float64 {
	if a == 0 || math.IsNaN(a) {
		return b
	}
	return a
}

// hash turns a species id into a stable seed, so placement survives redraws.
func hash(text string) uint32 {
	h := uint32(2166136261)
	for _, ch := range text {
		h = (h ^ uint32(ch)) * 16777619
	}
	return h
}

// rng is random(seed): a seeded generator of numbers in [0, 1) (mulberry32).
type rng struct{ s uint32 }

func random(seed uint32) *rng { return &rng{seed} }

func (r *rng) next() float64 {
	r.s += 0x6d2b79f5
	s := r.s
	t := (s ^ (s >> 15)) * (1 | s)
	t = (t + (t^(t>>7))*(61|t)) ^ t
	return float64(t^(t>>14)) / 4294967296
}

func lattice(seed uint32, i int64) float64 {
	h := (seed ^ (uint32(i) * 374761393)) * 668265263
	h = (h ^ (h >> 13)) * 1274126177
	return float64(h^(h>>16))/2147483648 - 1
}

// noise is smooth value noise in −1…1 along t, different for every seed.
func noise(seed uint32, t float64) float64 {
	i := math.Floor(t)
	return lerp(lattice(seed, int64(i)), lattice(seed, int64(i)+1), smooth(t-i))
}

// follow eases a value toward its target; rate is how quickly, per second.
func follow(value, target, rate, dt float64) float64 {
	return target + (value-target)*math.Exp(-rate*dt)
}

// knee finds the joint of a two-segment limb from a hip to a foot, on the
// upper side of the line between them (or the lower side when up is false).
func knee(hx, hy, fx, fy, upper, lower float64) [2]float64 {
	return kneeUp(hx, hy, fx, fy, upper, lower, true)
}

func kneeUp(hx, hy, fx, fy, upper, lower float64, up bool) [2]float64 {
	dx, dy := fx-hx, fy-hy
	d := clamp(math.Hypot(dx, dy), 1e-6, upper+lower-1e-6)
	a := (upper*upper - lower*lower + d*d) / (2 * d)
	h := math.Sqrt(math.Max(0, upper*upper-a*a))
	ux, uy := dx/d, dy/d
	px, py := hx+ux*a, hy+uy*a
	side := 1.0
	if (ux > 0) == up {
		side = -1
	}
	return [2]float64{px - uy*h*side, py + ux*h*side}
}

// leg is one foot of a Gait.
type leg struct {
	dx, dz                  float64
	group                   int
	x, y, fromX, fromY, toX float64
	t                       float64
}

type groundFn func(x float64, l *leg) float64

// Gait keeps each foot planted until the body has moved past it, then
// swings it ahead. Legs belong to two groups that never swing at the same
// time; a waiting group stops the airborne group from restarting steps.
// When both groups are due, the group that did not step last goes first.
// Rest offsets are in body lengths; sx scales them to pixels and mirrors
// them with the facing.
type gait struct {
	legs                   []*leg
	stride, lift, duration float64
	placed                 bool
	last                   int
}

type legSpec struct {
	dx, dz float64
	group  int
}

func newGait(specs []legSpec, stride, lift, duration float64) *gait {
	g := &gait{stride: stride, lift: lift, duration: duration, last: 1}
	for _, s := range specs {
		g.legs = append(g.legs, &leg{dx: s.dx, dz: s.dz, group: s.group, t: 1})
	}
	return g
}

// place puts every foot straight under its rest position.
func (g *gait) place(x, sx float64, ground groundFn) {
	for _, l := range g.legs {
		l.x = x + l.dx*sx
		l.toX = l.x
		l.y = ground(l.x, l)
		l.t = 1
	}
	g.placed = true
}

const noGroup = -1

// update swings stepping feet and starts new steps; vx is the body's speed.
func (g *gait) update(dt, x, sx float64, ground groundFn, vx float64) {
	if !g.placed {
		g.place(x, sx, ground)
		return
	}
	L := math.Abs(sx)
	stride := g.stride * L
	walking := math.Abs(vx) > stride*.25
	for _, l := range g.legs {
		if l.t < 1 {
			l.t = math.Min(1, l.t+dt/g.duration)
			e := smooth(l.t)
			l.x = lerp(l.fromX, l.toX, e)
			l.y = lerp(l.fromY, ground(l.x, l), e) - math.Sin(math.Pi*l.t)*g.lift*L
		} else {
			l.y = ground(l.x, l)
		}
	}
	reach := .15
	if walking {
		reach = .5
	}
	due := func(l *leg) bool { return l.t >= 1 && math.Abs(l.x-(x+l.dx*sx)) >= stride*reach }
	var air *leg
	for _, l := range g.legs {
		if l.t < 1 {
			air = l
			break
		}
	}
	group := noGroup
	if air != nil {
		other := false
		for _, l := range g.legs {
			if l.group != air.group && due(l) {
				other = true
				break
			}
		}
		if !other {
			group = air.group
		}
	} else {
		for _, l := range g.legs {
			if l.group != g.last && due(l) {
				group = l.group
				break
			}
		}
		if group == noGroup {
			for _, l := range g.legs {
				if due(l) {
					group = l.group
					break
				}
			}
		}
	}
	for _, l := range g.legs {
		if l.group != group || !due(l) {
			continue
		}
		rest := x + l.dx*sx
		l.fromX, l.fromY = l.x, l.y
		l.toX = rest
		if walking {
			l.toX += sign(vx) * stride * .45
		}
		l.t = 0
		g.last = l.group
	}
}

// airborne tells how much of the body's weight is off the ground (0…1).
func (g *gait) airborne() float64 {
	up := 0.0
	for _, l := range g.legs {
		if l.t < 1 {
			up += math.Sin(math.Pi * l.t)
		}
	}
	return up / float64(len(g.legs))
}

// spot is a place on the bottom: x and depth.
type spot struct{ x, z float64 }

// goal is where a crawler walks to, maybe to eat.
type goal struct {
	x, z float64
	food *pellet
}

// crawlerModel is what models override of the routine.
type crawlerModel interface {
	settle(w *world)
	startle(w *world)
	wander(w *world) *goal
	setActive(on bool, w *world, instant bool)
}

// Crawler is the routine shared by shrimp and crabs. While out it wanders
// near home, picks up food that reached the sand and flinches when a fish
// comes close; sent in, it walks to its den and rests or hides there.
// Models add the body, legs and drawing.
type crawler struct {
	self   crawlerModel
	spec   *Spec
	seed   uint32
	rand   *rng
	active bool
	state  string
	home   spot
	den    spot
	depths [2]float64
	hides  bool
	burrow *burrow

	x, y, z, facing, vx float64
	goal                *goal
	until               float64
	meal                *pellet
	eating, calm, turn  float64
	shown, clock        float64
	retreat             float64
	busy                bool
	legs                *gait
}

func (c *crawler) init(self crawlerModel, spec *Spec, w *world, enter bool) {
	c.self = self
	c.spec = spec
	c.seed = hash(spec.ID)
	c.rand = random(c.seed ^ 0x51ed27)
	c.active = false
	c.state = "in"
	self.settle(w)
	c.x, c.z = c.den.x, c.den.z
	c.facing = 1
	if c.rand.next() < .5 {
		c.facing = -1
	}
	c.vx, c.goal, c.until = 0, nil, 0
	c.meal, c.eating, c.calm = nil, 0, 0
	c.turn = 0
	c.shown = 1
	if c.hides {
		c.shown = 0
	}
	c.clock = c.rand.next() * 100
	c.retreat = 0
	if enter {
		c.state, c.active, c.shown = "out", true, 1
		if c.rand.next() < .5 {
			c.x = -w.width * .04
		} else {
			c.x = w.width * 1.04
		}
		c.z = c.home.z
		c.goal = &goal{x: c.home.x, z: c.home.z}
	}
	c.y = w.surface(c.x, c.z)
}

func (c *crawler) base() *crawler  { return c }
func (c *crawler) isCrawler() bool { return true }
func (c *crawler) depth() float64  { return c.z }

// size is the body length in pixels at the current depth.
func (c *crawler) size(w *world) float64 { return c.spec.Length * w.px * (1 - .3*c.z) }

// settle chooses home (where it spends its time out) and den (where it
// rests). Rock dwellers live on a rock and hide behind it; others use a
// patch of sand. Models with burrows override this.
func (c *crawler) settleBase(w *world) {
	W, s := w.width, c.spec
	if s.Rock {
		rocks := w.rocks
		r := rocks[[]int{0, 2, 1, 3}[c.seed%4]%len(rocks)]
		x := r.x + (c.rand.next()-.5)*r.rx*.8
		c.home = spot{x, r.z}
		dx := r.x + (c.rand.next()-.5)*r.rx*.4
		c.den = spot{dx, math.Min(1, r.z+.22)}
		c.depths = [2]float64{math.Max(0, r.z-.12), r.z + .04}
		c.hides = true
	} else {
		x := W * (.12 + float64(c.seed%1000)/1000*.76)
		z0, z1 := .15, .7
		if s.Depths != nil {
			z0, z1 = s.Depths[0], s.Depths[1]
		}
		c.home = spot{x, lerp(z0, z1, c.rand.next())}
		c.den = c.home
		c.depths = [2]float64{z0, z1}
		c.hides = false
	}
}

// hidden reports whether nothing of it shows right now.
func (c *crawler) hidden() bool { return c.shown < .02 }

func (c *crawler) setActiveBase(on bool, w *world, instant bool) {
	was := c.state
	c.active = on
	c.drop()
	if on {
		if instant {
			c.x, c.z = c.home.x, c.home.z
			c.goal = nil
			c.until = w.time + c.rand.next()*4
			c.shown = 1
		} else {
			if was == "in" {
				c.x, c.z = c.den.x, c.den.z
			}
			c.goal = &goal{x: c.home.x, z: c.home.z}
		}
		c.state = "out"
	} else if instant {
		c.state = "in"
		c.x, c.z = c.den.x, c.den.z
		c.goal = nil
		c.vx = 0
		c.shown = 1
		if c.hides {
			c.shown = 0
		}
	} else {
		c.state = "leaving"
		c.goal = &goal{x: c.den.x, z: c.den.z}
	}
	c.y = w.surface(c.x, c.z)
	if instant && c.legs != nil {
		c.legs.place(c.x, c.legScale(w), c.footing(w))
	}
}

func (c *crawler) resize(sx, sy float64, w *world) {
	c.self.settle(w)
	c.drop()
	c.x *= sx
	c.goal = nil
	if c.state != "out" {
		if c.active {
			c.state = "out"
			c.x, c.z = c.home.x, c.home.z
		} else {
			c.state = "in"
			c.x, c.z = c.den.x, c.den.z
		}
	}
	c.y = w.surface(c.x, c.z)
	if c.legs != nil {
		c.legs.place(c.x, c.legScale(w), c.footing(w))
	}
}

// flipX is the horizontal scale for drawing: the facing, squashed while it turns around.
func (c *crawler) flipX() float64 {
	if c.turn != 0 {
		return c.facing * math.Max(.2, math.Abs(math.Cos(math.Pi*c.turn)))
	}
	return c.facing
}

// localFeet maps the feet into body units for drawing, with the body raised by lift pixels.
func (c *crawler) localFeet(L, lift float64) [][2]float64 {
	out := make([][2]float64, len(c.legs.legs))
	if !c.legs.placed {
		for i, l := range c.legs.legs {
			out[i] = [2]float64{l.dx, lift/L - l.dz*1.5}
		}
		return out
	}
	sx := c.flipX() * L
	if c.spec.Sideways {
		sx = L
	}
	for i, l := range c.legs.legs {
		out[i] = [2]float64{(l.x - c.x) / sx, (l.y - c.y + lift) / L}
	}
	return out
}

// legScale mirrors leg offsets with the facing; front-on crabs don't.
func (c *crawler) legScale(w *world) float64 {
	if c.spec.Sideways {
		return c.size(w)
	}
	return c.size(w) * c.facing
}

// footing is the ground the legs stand on: the body's own surface, with
// feet on the far side a little higher, as they would look from above.
func (c *crawler) footing(w *world) groundFn {
	L := c.size(w)
	return func(x float64, l *leg) float64 { return w.surface(x, c.z) - l.dz*L*1.5 }
}

// wander picks the next spot to walk to while out.
func (c *crawler) wanderBase(w *world) *goal {
	r := w.width * or(c.spec.Range, .2)
	x := clamp(c.home.x+(c.rand.next()*2-1)*r, w.width*.04, w.width*.96)
	return &goal{x: x, z: lerp(c.depths[0], c.depths[1], c.rand.next())}
}

// routine moves the body; models call it from step before animating.
func (c *crawler) routine(dt float64, w *world) {
	t, L := w.time, c.size(w)
	c.clock += dt
	if c.meal != nil && c.meal.alpha < 1 {
		c.meal = nil
		c.eating = 0
	}
	// After a scare it stays home a while, then comes back out.
	if c.retreat > 0 {
		c.retreat -= dt
		if c.retreat <= 0 && c.active && c.state == "in" {
			c.state = "out"
			c.goal = &goal{x: c.home.x, z: c.home.z}
		}
	}
	if c.eating > 0 {
		c.eating -= dt
		c.vx = follow(c.vx, 0, 8, dt)
		if c.eating <= 0 && c.meal != nil {
			w.eat(c.meal)
			c.meal = nil
		}
	} else if c.state != "in" && !c.busy {
		if c.state == "out" {
			food := c.forage(w)
			if food != nil {
				s := or(sign(food.x-c.x), c.facing)
				c.goal = &goal{x: food.x - c.reach(L)*s, z: food.z, food: food}
			} else if c.goal == nil && t > c.until {
				c.goal = c.self.wander(w)
			}
			if t > c.calm {
				c.watch(w, L)
			}
		}
		if c.goal != nil {
			dx, dz := c.goal.x-c.x, c.goal.z-c.z
			speed := c.spec.Speed * L
			if c.state == "leaving" {
				speed *= 1.2
			}
			c.vx = follow(c.vx, clamp(dx*3, -speed, speed), 4, dt)
			c.z = clamp(c.z+clamp(dz*1.5, -.2, .2)*dt, 0, 1)
			if math.Abs(dx) < L*.1 && math.Abs(dz) < .03 {
				if f := c.goal.food; f != nil && !f.eaten && f.alpha >= 1 {
					c.meal = f
					c.eating = 1.8
				}
				c.goal = nil
				if c.state == "leaving" {
					c.state = "in"
				} else {
					c.until = t + (2+c.rand.next()*6)*or(c.spec.Idle, 1)
				}
			}
		} else {
			c.vx = follow(c.vx, 0, 6, dt)
		}
	} else if !c.busy {
		c.vx = follow(c.vx, 0, 6, dt)
	}
	c.x += c.vx * dt
	if !c.busy && !c.spec.Sideways && c.turn == 0 && math.Abs(c.vx) > L*.08 && sign(c.vx) != c.facing {
		c.turn = 1e-3
	}
	if c.turn != 0 {
		c.turn += dt / .35
		if c.turn >= .5 {
			c.facing = or(sign(c.vx), c.facing)
		}
		if c.turn >= 1 {
			c.turn = 0
		}
	}
	target := 1.0
	if c.state == "in" && c.hides {
		target = 0
	}
	c.shown = follow(c.shown, target, 3, dt)
	c.y = w.surface(c.x, c.z)
}

// forage finds food lying on the sand within reach of home.
func (c *crawler) forage(w *world) *pellet {
	if c.goal != nil && c.goal.food != nil {
		if !c.goal.food.eaten && c.goal.food.alpha >= 1 {
			return c.goal.food
		}
		c.drop()
		c.goal = nil
	}
	var best *pellet
	bd := w.width * or(c.spec.Range, .2) * 1.5
	for _, p := range w.pellets {
		if !p.landed || p.eaten || p.claimed || p.alpha < 1 {
			continue
		}
		d := math.Abs(p.x - c.x)
		if d < bd {
			bd = d
			best = p
		}
	}
	if best != nil {
		best.claimed = true
	}
	return best
}

// reach is how far from the body's center food is picked up.
func (c *crawler) reach(L float64) float64 {
	if c.spec.Sideways {
		return 0
	}
	return L * .4
}

// watch flinches when a fish swims within a body length.
func (c *crawler) watch(w *world, L float64) {
	for _, f := range w.fish {
		if math.Abs(f.z-c.z) > .3 {
			continue
		}
		if math.Hypot(f.x-c.x, f.y-c.y) < L+f.size(w)*.4 {
			c.calm = w.time + 8
			c.drop()
			c.self.startle(w)
			return
		}
	}
}

// startle is the model's flinch; by default it freezes for a moment.
func (c *crawler) startleBase(w *world) {
	c.goal = nil
	c.until = w.time + 2.5
}

// hideAway runs home to the den for a while, still counted as out.
func (c *crawler) hideAway(seconds float64) {
	c.drop()
	c.retreat = seconds
	c.state = "leaving"
	c.goal = &goal{x: c.den.x, z: c.den.z}
}

// drop gives up the pellet it was walking to, so another can have it.
func (c *crawler) drop() {
	if c.goal != nil && c.goal.food != nil {
		c.goal.food.claimed = false
	}
}
