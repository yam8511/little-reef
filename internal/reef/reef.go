// Package reef draws the animated aquarium: bubbles, sand, rocks, plants,
// food pellets and the creatures met so far. It is a port of the browser
// version (web/src/reef/Reef.ts, species.ts and engine/*.js) onto MyGo's
// painter, step for step and color for color.
package reef

import (
	"math"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
)

// How far back each planted spot sits, so fish can pass in front of some
// plants and behind others.
var plantDepth = []float64{0.55, 0.2, 0.7, 0.35, 0.85, 0.15, 0.6, 0.4}

// Visitor is a species met, with the RFC 3339 date it came.
type Visitor struct {
	Species string
	Date    string
}

// State is the part of the app's snapshot the reef shows.
type State struct {
	Completed  int       // completed rounds: they grow the plants, and from 24 tint the water
	Visitors   []Visitor // in snapshot order
	FedSession int64     // a change (non-zero, while not still) scatters food
}

// Palette carries the theme values the reef reads. A zero Color means the
// reef's own default.
type Palette struct {
	Sand, Coral, Seaweed, Bubble, Accent ui.Color
	BubbleDensity                        float64 // theme motion.bubbleDensity, 0 => 1
	MaxCreatures                         int     // theme behavior.maxCreatures, 0 => 999
	FishColor                            string  // theme creaturePalette.fish ("#rrggbb"), "" => species' own colors
}

// themeColor is the theme's color, or the fallback when it sets none.
func themeColor(c ui.Color, fallback string) rgba {
	if c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 {
		return hexC(fallback)
	}
	return fromUI(c, 1)
}

type pellet struct {
	x, y, vy, seed, z, alpha float64
	eaten, landed, claimed   bool
}

type rock struct {
	x, y, rx, ry, angle float64
	color               string
	z                   float64
}

type plant struct {
	x, height float64
	seed      int
	coral     bool
	z         float64
}

type burrow struct{ x, z, y float64 }

// world is what the creatures see of the reef.
type world struct {
	r             *Reef
	width, height float64
	px            float64
	time          float64
	still         bool
	fish          []*fish
	pellets       []*pellet
	meal          int
	rocks         []*rock
	burrows       map[string]*burrow
}

func (w *world) ground(x, z float64) float64  { return w.r.ground(x, z) }
func (w *world) surface(x, z float64) float64 { return w.r.surface(x, z) }
func (w *world) home(x float64) pt            { return w.r.home(x) }
func (w *world) eat(p *pellet)                { p.eaten = true }

// Reef is the aquarium.
type Reef struct {
	state     *State
	pending   []State
	sized     bool
	time      float64
	last      time.Time
	hasLast   bool
	order     []string
	creature  map[string]creature
	populated bool
	newest    string
	feedUntil float64
	nextSwap  float64
	plants    []plant
	rand      *rng
	width     float64
	height    float64
	sand      [][2]float64
	world     *world
	pal       Palette
}

// New returns an empty reef.
func New() *Reef {
	r := &Reef{creature: map[string]creature{}, nextSwap: 150, rand: random(20261015), width: 1, height: 1}
	r.world = &world{r: r, width: 1, height: 1, px: 5, burrows: map[string]*burrow{}}
	return r
}

// Update shows a new snapshot. Before the reef has a size (its first
// Draw), snapshots wait, as the browser's reef measured its canvas before
// any came.
func (r *Reef) Update(s State) {
	s.Visitors = append([]Visitor(nil), s.Visitors...)
	if !r.sized {
		r.pending = append(r.pending, s)
		return
	}
	r.update(s)
}

func (r *Reef) update(s State) {
	var oldFed int64
	had := r.state != nil
	if had {
		oldFed = r.state.FedSession
	}
	r.state = &s
	r.plantBed()
	r.populate()
	if had && s.FedSession != oldFed && s.FedSession != 0 && !r.world.still {
		r.feed()
	}
}

// SetPalette applies a theme: the reef lays itself out again, and draws
// with the new colors. Creatures already in the tank keep theirs; ones
// that come later take the theme's.
func (r *Reef) SetPalette(p Palette) {
	r.pal = p
	if r.sized {
		r.layout()
	}
}

// SetStill switches reduced motion on or off ahead of a Draw, as the
// browser's reef did when the preference changed.
func (r *Reef) SetStill(still bool) {
	was := r.world.still
	r.world.still = still
	if still && !was && r.populated {
		r.world.pellets = r.world.pellets[:0]
		r.order = nil
		r.creature = map[string]creature{}
		r.rand = random(20261015)
		r.populated = false
		r.populate()
	}
	if !still && was {
		r.hasLast = false
	}
}

// Animating reports whether the reef moves, so its Draw should ask for frames.
func (r *Reef) Animating() bool { return !r.world.still }

func (r *Reef) resize(w, h float64) {
	oldWidth, oldHeight := or(r.width, 1), or(r.height, 1)
	r.width = math.Max(1, w)
	r.height = math.Max(1, h)
	r.layout()
	for _, id := range r.order {
		r.creature[id].resize(r.width/oldWidth, r.height/oldHeight, r.world)
	}
	for _, p := range r.world.pellets {
		p.x *= r.width / oldWidth
		p.y *= r.height / oldHeight
		if p.landed {
			p.y = r.ground(p.x, p.z)
		}
	}
	if r.populated {
		r.arrange(false, r.world.still)
	}
}

// layout measures the sand, rocks and burrows for the current size.
func (r *Reef) layout() {
	w, h, wd := r.width, r.height, r.world
	wd.width, wd.height = w, h
	wd.px = clamp(h/48, 4.2, 6.5)
	r.sand = r.sand[:0]
	for i := 0; i <= 64; i++ {
		t := float64(i) / 64
		u := 1 - t
		r.sand = append(r.sand, [2]float64{3*u*u*t*w*0.4 + 3*u*t*t*w*0.62 + t*t*t*w, u*u*u*(h-17) + 3*u*u*t*(h-43) + 3*u*t*t*(h-17) + t*t*t*(h-25)})
	}
	wd.rocks = []*rock{
		{x: w * 0.81, y: h - 25, rx: 28, ry: 14, angle: -0.15, color: "#6d8179", z: 0.4},
		{x: w * 0.85, y: h - 20, rx: 17, ry: 10, angle: 0.2, color: "#82948a", z: 0.3},
		{x: w * 0.45, y: h - 23, rx: 23, ry: 12, angle: 0.1, color: "#6d8179", z: 0.45},
		{x: w * 0.42, y: h - 19, rx: 14, ry: 9, angle: -0.25, color: "#82948a", z: 0.35},
	}
	wd.burrows = map[string]*burrow{
		"mantis": {x: w * 0.6, z: 0.3},
		"pistol": {x: w * 0.7, z: 0.55},
	}
	for _, b := range wd.burrows {
		b.y = r.ground(b.x, b.z)
	}
	r.plantBed()
}

// ground is the sand's surface at x; nearer things (smaller z) stand lower.
func (r *Reef) ground(x, z float64) float64 {
	s := r.sand
	i := 1
	for i < len(s)-1 && s[i][0] < x {
		i++
	}
	x0, y0, x1, y1 := s[i-1][0], s[i-1][1], s[i][0], s[i][1]
	top := y0 + (y1-y0)*clamp((x-x0)/or(x1-x0, 1), 0, 1)
	return top + (1-clamp(z, 0, 1))*clamp((r.height-top)*0.5, 4, 14)
}

// surface is where a crawler at depth z stands: on a rock at its depth, or the sand.
func (r *Reef) surface(x, z float64) float64 {
	y := r.ground(x, z)
	for _, k := range r.world.rocks {
		if math.Abs(k.z-z) > 0.16 {
			continue
		}
		u := (x - k.x) / (k.rx * 0.9)
		if math.Abs(u) < 1 {
			y = math.Min(y, k.y-k.ry*0.92*math.Sqrt(1-u*u)+1)
		}
	}
	return y
}

// plantBed lays out the plants that the completed rounds have grown.
func (r *Reef) plantBed() {
	growth := 0
	if r.state != nil {
		growth = r.state.Completed
	}
	w := r.width
	r.plants = nil
	if growth <= 0 {
		return
	}
	count := min(8, 1+int(math.Floor(float64(growth-1)/3)))
	for i := 0; i < count; i++ {
		progress := math.Min(3, math.Max(1, float64(growth-i*3)))
		r.plants = append(r.plants, plant{x: w * (0.09 + math.Mod(float64(i)*0.137, 0.71)), height: 28 + progress*18, seed: i, coral: i > 0 && i%3 == 0, z: plantDepth[i]})
	}
}

// home is where the clownfish lives: the nearest plant, or a spot beside the nearest rock.
func (r *Reef) home(x float64) pt {
	var best *plant
	for i := range r.plants {
		p := &r.plants[i]
		if best == nil || math.Abs(p.x-x) < math.Abs(best.x-x) {
			best = p
		}
	}
	if best != nil {
		return pt{best.x, r.height - 25 - best.height*0.55}
	}
	var rk *rock
	for _, k := range r.world.rocks {
		if rk == nil || math.Abs(k.x-x) < math.Abs(rk.x-x) {
			rk = k
		}
	}
	return pt{rk.x - 20, r.height - 70}
}

// populate adds a creature for the resident and every visitor met so far.
func (r *Reef) populate() {
	var visitors []Visitor
	if r.state != nil {
		visitors = r.state.Visitors
	}
	if n := maxCreatures(&r.pal) - 1; len(visitors) > n {
		visitors = visitors[:max(0, n)]
	}
	enter := r.populated && !r.world.still
	ids := []string{"resident"}
	for _, v := range visitors {
		ids = append(ids, v.Species)
	}
	for _, id := range ids {
		if _, ok := r.creature[id]; ok {
			continue
		}
		if c := create(id, r.world, enter, &r.pal); c != nil {
			r.creature[id] = c
			r.order = append(r.order, id)
		}
	}
	var newest *Visitor
	for i := range visitors {
		if v := &visitors[i]; newest == nil || v.Date > newest.Date {
			newest = v
		}
	}
	r.newest = ""
	if newest != nil {
		r.newest = newest.Species
	}
	r.arrange(false, !r.populated || r.world.still)
	r.populated = true
}

func (r *Reef) crawlers() []crawlerCreature {
	var out []crawlerCreature
	for _, id := range r.order {
		if c, ok := r.creature[id].(crawlerCreature); ok {
			out = append(out, c)
		}
	}
	return out
}

func without(list []crawlerCreature, c crawlerCreature) []crawlerCreature {
	var out []crawlerCreature
	for _, o := range list {
		if o != c {
			out = append(out, o)
		}
	}
	return out
}

// arrange keeps a few crawlers out and the rest resting at home.
func (r *Reef) arrange(swap, instant bool) {
	crawlers := r.crawlers()
	if len(crawlers) == 0 {
		return
	}
	limit := int(clamp(math.Floor(r.width/120), 3, 5))
	newest, _ := r.creature[r.newest].(crawlerCreature)
	var out []crawlerCreature
	for _, c := range crawlers {
		if c.base().active {
			out = append(out, c)
		}
	}
	var resting crawlerCreature
	if swap && len(crawlers) > limit {
		var leaving []crawlerCreature
		for _, c := range out {
			if c != newest {
				leaving = append(leaving, c)
			}
		}
		if len(leaving) > 0 {
			resting = leaving[int(math.Floor(r.rand.next()*float64(len(leaving))))]
			resting.setActive(false, r.world, instant)
			out = without(out, resting)
		}
	}
	if newest != nil && !newest.base().active {
		newest.setActive(true, r.world, instant)
		out = append(out, newest)
	}
	for len(out) > limit {
		var c crawlerCreature
		for _, o := range out {
			if o != newest {
				c = o
				break
			}
		}
		c.setActive(false, r.world, instant)
		out = without(out, c)
	}
	var waiting []crawlerCreature
	for _, c := range crawlers {
		if !c.base().active && c != resting {
			waiting = append(waiting, c)
		}
	}
	for len(out) < limit && len(waiting) > 0 {
		room := func(c crawlerCreature) float64 {
			m := math.Inf(1)
			for _, o := range out {
				m = math.Min(m, math.Abs(o.base().home.x-c.base().home.x))
			}
			return m + r.rand.next()*r.width*0.15
		}
		v8Sort(waiting, func(a, b crawlerCreature) float64 { rb := room(b); return rb - room(a) })
		c := waiting[0]
		waiting = waiting[1:]
		c.setActive(true, r.world, instant)
		out = append(out, c)
	}
}

// v8Sort sorts as Chromium's Array.prototype.sort does for the short
// arrays here (a binary insertion sort), so a comparator that draws random
// numbers draws them in the same order as the browser did. (Checked against
// Chrome's comparator calls; older V8, as Node 24's, first looked for a
// sorted run and then inserted the rest.)
func v8Sort[T any](a []T, cmp func(x, y T) float64) {
	for start := 1; start < len(a); start++ {
		left, right := 0, start
		pivot := a[start]
		for left < right {
			mid := left + (right-left)>>1
			if cmp(pivot, a[mid]) < 0 {
				right = mid
			} else {
				left = mid + 1
			}
		}
		copy(a[left+1:start+1], a[left:start])
		a[left] = pivot
	}
}

// feed scatters pellets near the surface; they sink slowly and drift.
func (r *Reef) feed() {
	rn := r.rand
	x := r.width * (0.3 + rn.next()*0.4)
	r.world.pellets = r.world.pellets[:0]
	for i := 0; i < 12; i++ {
		p := &pellet{alpha: 1}
		p.x = x + (rn.next()-0.5)*r.width*0.16
		p.y = r.height * (0.03 + rn.next()*0.06)
		p.vy = r.height * (0.06 + rn.next()*0.03)
		p.seed = rn.next() * 9
		p.z = 0.15 + rn.next()*0.5
		r.world.pellets = append(r.world.pellets, p)
	}
	r.feedUntil = r.time + 20
	r.world.meal++
}

func (r *Reef) step(dt float64) {
	w := r.world
	w.time = r.time
	w.fish = w.fish[:0]
	for _, id := range r.order {
		if f, ok := r.creature[id].(*fish); ok {
			w.fish = append(w.fish, f)
		}
	}
	if r.time > r.nextSwap {
		r.nextSwap = r.time + 120 + r.rand.next()*120
		r.arrange(true, false)
	}
	for i := len(w.pellets) - 1; i >= 0; i-- {
		p := w.pellets[i]
		if p.eaten || p.alpha <= 0 {
			w.pellets = append(w.pellets[:i], w.pellets[i+1:]...)
			continue
		}
		if !p.landed {
			p.y += p.vy * dt
			p.x += math.Sin(r.time*1.3+p.seed) * 5 * dt
			floor := r.ground(p.x, p.z)
			if p.y >= floor {
				p.y = floor
				p.landed = true
			}
		}
		if r.time > r.feedUntil {
			p.alpha -= dt
		}
	}
	for _, id := range r.order {
		r.creature[id].step(dt, w)
	}
}

// Draw advances the reef to now and paints it into rect.
//
// The reef steps at most 24 times a second, each step at most 0.15 s, as
// the browser's animation loop did; drawing again at the same time (or
// sooner than a step is due) paints the same picture. With still, it is
// reduced motion: nothing moves and no food falls.
func (r *Reef) Draw(p *ui.Painter, rect ui.Rect, now time.Time, still bool) {
	if still != r.world.still {
		r.SetStill(still)
	}
	w, h := math.Max(1, float64(rect.W)), math.Max(1, float64(rect.H))
	if !r.sized || w != r.width || h != r.height {
		first := !r.sized
		r.sized = true
		r.resize(w, h)
		if first {
			for _, s := range r.pending {
				r.update(s)
			}
			r.pending = nil
		}
	}
	if !r.world.still {
		r.advance(now)
	}
	c := newCtx(mat{1, 0, 0, 1, float64(rect.X), float64(rect.Y)})
	r.paint(c)
	// A canvas shows nothing outside itself.
	p.Clip(rect, 0, func() { c.draw(p) })
}

// advance runs the throttled animation loop up to now.
func (r *Reef) advance(now time.Time) {
	if r.hasLast && now.Sub(r.last) < time.Second/24 {
		return
	}
	delta := 0.0
	if r.hasLast {
		delta = math.Min(now.Sub(r.last).Seconds(), 0.15)
	}
	r.last, r.hasLast = now, true
	r.time += delta
	r.step(delta)
}

func (r *Reef) plant(c *ctx, x, height float64, seed int, coral bool) {
	h := r.height
	sway := math.Sin(r.time*0.5+float64(seed)) * 5
	c.save()
	c.translate(x, h-25)
	c.roundCaps()
	if coral {
		c.setStroke(themeColor(r.pal.Coral, "#bd918c"))
		c.setLineWidth(4)
		for i := 0; i < 5; i++ {
			side := -1.0
			if i%2 != 0 {
				side = 1
			}
			y := -height * (0.3 + float64(i)*0.14)
			c.beginPath()
			c.moveTo(0, 2)
			c.quadraticCurveTo(sway/3, -height/2, sway, -height)
			c.stroke()
			c.beginPath()
			c.moveTo(sway*0.3, y+12)
			c.quadraticCurveTo(side*15, y, side*18, y-18)
			c.stroke()
		}
	} else {
		if seed%2 != 0 {
			c.setStroke(themeColor(r.pal.Seaweed, "#529a86"))
		} else {
			c.setStroke(hexC("#70af91"))
		}
		c.setLineWidth(3)
		c.beginPath()
		c.moveTo(0, 0)
		c.bezierCurveTo(-8, -height/3, 10+sway, -height*0.7, sway, -height)
		c.stroke()
		for i := 0; i < 4; i++ {
			y := -height * (0.2 + float64(i)*0.19)
			side := -1.0
			if i%2 != 0 {
				side = 1
				c.setFill(hexC("#79b89b"))
			} else {
				c.setFill(hexC("#568d7c"))
			}
			c.beginPath()
			c.moveTo(sway*0.2, y)
			c.quadraticCurveTo(side*25, y-3, side*16+sway, y-22)
			c.quadraticCurveTo(side*2, y-17, sway*0.2, y)
			c.fill()
		}
	}
	c.restore()
}

func (r *Reef) rock(c *ctx, k *rock) {
	c.setFill(hexC(k.color))
	c.beginPath()
	c.ellipse(k.x, k.y, k.rx, k.ry, k.angle, 0, math.Pi*2, false)
	c.fill()
}

// paint draws the whole stage, as Reef.draw did.
func (r *Reef) paint(c *ctx) {
	w, h, t := r.width, r.height, r.time
	b := themeColor(r.pal.Bubble, "#b4cfb1")
	b.a = 0x14 / 255.0
	c.setFill(b)
	density := r.pal.BubbleDensity
	if density == 0 {
		density = 1
	}
	bubbles := int(jsRound(18 * density))
	for i := 0; i < bubbles; i++ {
		fi := float64(i)
		y := h - math.Mod(t*(7+float64(i%3))+fi*27, h+30)
		x := math.Mod(fi*43+21, w) + math.Sin(t*0.2+fi)*8
		c.beginPath()
		c.arc(x, y, 1+float64(i%2), 0, math.Pi*2, false)
		c.fill()
	}
	c.setFill(hexC("#53675a"))
	c.beginPath()
	c.moveTo(0, h)
	c.lineTo(0, h-28)
	c.bezierCurveTo(w*0.25, h-53, w*0.7, h-7, w, h-34)
	c.lineTo(w, h)
	c.fill()
	c.setFill(themeColor(r.pal.Sand, "#718371"))
	c.beginPath()
	c.moveTo(0, h)
	c.lineTo(0, h-17)
	c.bezierCurveTo(w*0.4, h-43, w*0.62, h-17, w, h-25)
	c.lineTo(w, h)
	c.fill()
	for i := 0; i < 25; i++ {
		if i%2 != 0 {
			c.setFill(hexC("#899781"))
		} else {
			c.setFill(hexC("#566c5e"))
		}
		c.beginPath()
		c.ellipse(math.Mod(float64(i*71+14), w), h-8-float64(i%19), 1.5, 1, 0, 0, math.Pi*2, false)
		c.fill()
	}
	if r.state != nil && r.state.Completed >= 24 {
		a := themeColor(r.pal.Accent, "#a8deba")
		a.a = 0x08 / 255.0
		c.setFill(a)
		c.fillRect(0, 0, w, h)
	}
	for _, id := range r.order {
		r.creature[id].under(c, r.world)
	}
	type item struct {
		z    float64
		kind int
		i    int
	}
	var scene []item
	for i, k := range r.world.rocks {
		scene = append(scene, item{k.z, 0, i})
	}
	for i, p := range r.plants {
		scene = append(scene, item{p.z, 1, i})
	}
	for i, id := range r.order {
		scene = append(scene, item{r.creature[id].depth(), 2, i})
	}
	sort.SliceStable(scene, func(i, j int) bool { return scene[j].z-scene[i].z < 0 })
	for _, it := range scene {
		switch it.kind {
		case 0:
			r.rock(c, r.world.rocks[it.i])
		case 1:
			p := r.plants[it.i]
			r.plant(c, p.x, p.height, p.seed, p.coral)
		default:
			r.creature[r.order[it.i]].draw(c, r.world)
		}
	}
	for _, p := range r.world.pellets {
		rad := 1.7 * (1 - 0.3*p.z)
		// An alpha below zero made an invalid color, which the canvas ignored.
		if a := jsRound(p.alpha * 255); a >= 0 {
			pc := themeColor(r.pal.Coral, "#e2c294")
			pc.a = math.Min(a, 255) / 255
			c.setFill(pc)
		}
		c.beginPath()
		c.arc(p.x, p.y-rad*0.6, rad, 0, math.Pi*2, false)
		c.fill()
		c.setFill(rgbaC(255, 240, 214, 0.6*p.alpha))
		c.beginPath()
		c.arc(p.x-rad*0.3, p.y-rad, rad*0.4, 0, math.Pi*2, false)
		c.fill()
	}
}

// jsRound is Math.round: halves round up.
func jsRound(v float64) float64 { return math.Floor(v + 0.5) }
