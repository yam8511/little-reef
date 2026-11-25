package reef

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Expected numbers come from running the JS engine (web/src/reef) in Node
// and in Chrome with the same inputs.

func TestRandomHashNoiseMatchTheJS(t *testing.T) {
	r := random(20261015)
	for i, want := range []float64{0.40124592115171254, 0.50874828593805432, 0.54776566359214485, 0.50491111027076840} {
		if got := r.next(); got != want {
			t.Errorf("random #%d = %.17g, want %.17g", i, got, want)
		}
	}
	// The last id is CJK test data, hashed by the JS as written: keep it 鱼.
	for i, id := range []string{"resident", "clown", "mantis", "swimmer", "鱼"} {
		if got, want := hash(id), []uint32{746716513, 281156754, 2237791249, 834618391, 4162238011}[i]; got != want {
			t.Errorf("hash(%q) = %d, want %d", id, got, want)
		}
	}
	cases := []struct {
		seed uint32
		t    float64
		want float64
	}{{hash("tang"), 0.3, 0.073658859141170985}, {hash("tang") + 3, 12.7, 0.93339108029007900}, {7, -2.25, -0.86301159353752155}}
	for _, c := range cases {
		if got := noise(c.seed, c.t); got != c.want {
			t.Errorf("noise(%d, %v) = %.17g, want %.17g", c.seed, c.t, got, c.want)
		}
	}
	if k := knee(0, 0, 1, 0, .6, .6); math.Abs(k[0]-.5) > 1e-15 || math.Abs(k[1]+0.33166247903553997) > 1e-15 {
		t.Errorf("knee = %v", k)
	}
}

func TestSpeciesRoster(t *testing.T) {
	if len(Species) != 16 || len(species) != 16 {
		t.Fatalf("%d species listed, %d in the table", len(Species), len(species))
	}
	models := map[string]int{}
	for _, id := range Species {
		if !Known(id) {
			t.Errorf("%s unknown", id)
		}
		models[species[id].Model]++
	}
	if !maps(models, map[string]int{"fish": 5, "shrimp": 5, "mantis": 1, "hermit": 1, "crab": 4}) {
		t.Errorf("models %v", models)
	}
	if Known("unknown") || Known("") {
		t.Error("unknown species known")
	}
	w := New()
	w.sized = true
	w.resize(720, 330)
	for _, id := range Species {
		c := create(id, w.world, false, &Palette{})
		if c == nil {
			t.Fatalf("no creature for %s", id)
		}
		if c.isCrawler() == (species[id].Model == "fish") {
			t.Errorf("%s crawler=%v", id, c.isCrawler())
		}
	}
	if create("unknown", w.world, false, &Palette{}) != nil {
		t.Error("created an unknown species")
	}
}

func maps(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestSpeciesPaletteAndCap(t *testing.T) {
	if got := speciesColors("resident", &Palette{})["back"]; got != "#df7a3e" {
		t.Errorf("resident back %s", got)
	}
	if got := speciesColors("clown", &Palette{})["back"]; got != "#dc6624" {
		t.Errorf("clown back %s", got)
	}
	themed := speciesColors("clown", &Palette{FishColor: "#2E6BA0"})
	if themed["back"] != "#2E6BA0" || themed["fin"] != "#2E6BA0" || themed["iris"] != "#f0a54a" {
		t.Errorf("themed clown %v", themed)
	}
	// lighten's output, as species.ts builds it.
	if themed["flank"] != "rgb(83,133,177)" || themed["belly"] != "rgb(125,163,196)" {
		t.Errorf("themed flank %s belly %s", themed["flank"], themed["belly"])
	}
	if speciesColors("cleaner", &Palette{FishColor: "#2E6BA0"}) != nil {
		t.Error("a crawler got a fish palette")
	}
	if maxCreatures(&Palette{}) != 999 || maxCreatures(&Palette{MaxCreatures: 3}) != 3 {
		t.Error("maxCreatures")
	}
	r := sized(720, 330)
	r.SetPalette(Palette{MaxCreatures: 3})
	r.Update(State{Visitors: everyone()})
	if !slices.Equal(r.order, []string{"resident", "clown", "tang"}) {
		t.Errorf("capped tank holds %v", r.order)
	}
}

func sized(w, h float64) *Reef {
	r := New()
	r.sized = true
	r.resize(w, h)
	return r
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestGroundAndSurface(t *testing.T) {
	r := sized(720, 330)
	for i, c := range [][3]float64{{0, 0, 321.5}, {100, .3, 314.17744179557570}, {360, 1, 302.14303885229725}, {500, .5, 311.53313089012795}, {719, 0, 317.51375128917402}} {
		if got := r.ground(c[0], c[1]); !near(got, c[2]) {
			t.Errorf("ground #%d = %.17g, want %.17g", i, got, c[2])
		}
	}
	for i, c := range [][3]float64{{583, .4, 293.12040565012688}, {612, .3, 301.80000000000001}, {324, .45, 296.95999999999998}, {100, .4, 312.96032193369695}} {
		if got := r.surface(c[0], c[1]); !near(got, c[2]) {
			t.Errorf("surface #%d = %.17g, want %.17g", i, got, c[2])
		}
	}
	if b := r.world.burrows["mantis"]; b.x != 432 || !near(b.y, r.ground(432, .3)) {
		t.Errorf("mantis burrow %+v", *b)
	}
	if r.world.px != clamp(330.0/48, 4.2, 6.5) {
		t.Errorf("px %v", r.world.px)
	}
}

func TestPlantsAndHome(t *testing.T) {
	r := sized(720, 330)
	r.Update(State{Completed: 30})
	want := []string{"64.8000/82/false", "163.4400/82/false", "262.0800/82/false", "360.7200/82/true", "459.3600/82/false", "558.0000/82/false", "145.4400/82/true", "244.0800/82/false"}
	if len(r.plants) != len(want) {
		t.Fatalf("%d plants", len(r.plants))
	}
	for i, p := range r.plants {
		got := fmt.Sprintf("%.4f/%v/%v", p.x, p.height, p.coral)
		if got != want[i] {
			t.Errorf("plant %d = %s, want %s", i, got, want[i])
		}
	}
	if h := r.home(100); !near(h[0], 64.8) || !near(h[1], 259.9) {
		t.Errorf("home %v", h)
	}
	r.Update(State{Completed: 4})
	if len(r.plants) != 2 || r.plants[0].height != 82 || r.plants[1].height != 46 {
		t.Errorf("4 rounds grew %+v", r.plants)
	}
	r.Update(State{})
	if len(r.plants) != 0 {
		t.Error("plants without rounds")
	}
	if h := r.home(100); !near(h[0], 720*0.42-20) || h[1] != 330-70 {
		t.Errorf("home without plants %v", h)
	}
}

func TestArrangeKeepsAFewCrawlersOut(t *testing.T) {
	// The crawlers out after the first snapshot, as Chrome's reef chose them.
	for _, c := range []struct {
		width float64
		out   []string
	}{
		{300, []string{"harlequin", "arrow", "swimmer"}},
		{720, []string{"pistol", "mantis", "slipper", "arrow", "swimmer"}},
		{1000, []string{"pistol", "mantis", "slipper", "arrow", "swimmer"}},
	} {
		r := sized(c.width, 330)
		r.Update(State{Completed: 5, Visitors: everyone()})
		if got := active(r); !slices.Equal(got, c.out) {
			t.Errorf("width %v: out %v, want %v", c.width, got, c.out)
		}
	}
	// After 150 s one goes home and another comes out.
	r := sized(720, 330)
	r.Update(State{Completed: 5, Visitors: everyone()})
	r.time = 150.5
	r.step(0.05)
	if got := active(r); !slices.Equal(got, []string{"mantis", "slipper", "hermit", "arrow", "swimmer"}) {
		t.Errorf("after the swap %v", got)
	}
	if r.nextSwap != 375.97151175327599 {
		t.Errorf("next swap %.17g", r.nextSwap)
	}
	// The newest visitor always comes out.
	r = sized(300, 330)
	vs := everyone()
	// make the cleaner the newest
	vs[4].Date = "2028-01-01T00:00:00Z"
	r.Update(State{Visitors: vs})
	if !slices.Contains(active(r), "cleaner") || len(active(r)) != 3 {
		t.Errorf("newest cleaner not out: %v", active(r))
	}
}

func active(r *Reef) []string {
	var out []string
	for _, c := range r.crawlers() {
		if c.base().active {
			out = append(out, c.base().spec.ID)
		}
	}
	return out
}

func TestFeed(t *testing.T) {
	r := sized(720, 330)
	vs := everyone()[:3]
	r.Update(State{Completed: 30, Visitors: vs, FedSession: 1})
	if len(r.world.pellets) != 0 {
		t.Fatal("the first snapshot fed")
	}
	r.Update(State{Completed: 30, Visitors: vs, FedSession: 2})
	ps := r.world.pellets
	if len(ps) != 12 || r.feedUntil != 20 || r.world.meal != 1 {
		t.Fatalf("%d pellets until %v, meal %d", len(ps), r.feedUntil, r.world.meal)
	}
	want := [][5]float64{
		{332.56662783175705, 20.745760139124464, 24.798619991680606, 7.9290338340215385, 0.30605522098485383},
		{276.47908684462311, 15.139797966275363, 29.204486343520696, 8.0638773115351796, 0.49434168972074988},
	}
	for i, w := range want {
		p := ps[i]
		if !near(p.x, w[0]) || !near(p.y, w[1]) || !near(p.vy, w[2]) || !near(p.seed, w[3]) || !near(p.z, w[4]) || p.alpha != 1 {
			t.Errorf("pellet %d = %+v, want %v", i, *p, w)
		}
	}
	// The same session again, or none, feeds nothing more.
	r.Update(State{Completed: 30, Visitors: vs, FedSession: 2})
	r.Update(State{Completed: 30, Visitors: vs, FedSession: 0})
	if r.world.meal != 1 {
		t.Errorf("meal %d", r.world.meal)
	}
	// Pellets sink, land and fade after 20 s.
	for i := 0; i < 30*24; i++ {
		r.time += 1.0 / 24
		r.step(1.0 / 24)
	}
	if len(r.world.pellets) != 0 {
		t.Errorf("%d pellets left after 30 s", len(r.world.pellets))
	}
}

// render draws the reef once at now, twice over, as MyGo may.
func render(r *Reef, w, h int, now time.Time, still bool) {
	ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, rc ui.Rect) {
			r.Draw(p, rc, now, still)
			r.Draw(p, rc, now, still)
		})
	}, w, h, 1)
}

func TestDrawSteps24TimesASecondAtMost(t *testing.T) {
	r := New()
	r.Update(State{Completed: 3, Visitors: everyone()})
	if r.sized || len(r.order) != 0 {
		t.Fatal("populated before it had a size")
	}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	render(r, 720, 330, t0, false)
	if !r.sized || len(r.order) != 16 || r.width != 720 || r.height != 330 {
		t.Fatalf("first draw: sized %v, %d creatures, %vx%v", r.sized, len(r.order), r.width, r.height)
	}
	if r.time != 0 {
		t.Fatalf("time %v after the first frame", r.time)
	}
	f := r.creature["tang"].(*fish)
	x := f.x
	render(r, 720, 330, t0.Add(30*time.Millisecond), false) // sooner than 1/24 s
	if r.time != 0 || f.x != x {
		t.Error("stepped before 1/24 s")
	}
	render(r, 720, 330, t0.Add(50*time.Millisecond), false)
	if !near(r.time, 0.05) || f.x == x {
		t.Errorf("time %v after 50 ms", r.time)
	}
	render(r, 720, 330, t0.Add(50*time.Millisecond+5*time.Second), false)
	if !near(r.time, 0.2) {
		t.Errorf("a long gap stepped %v, want 0.15", r.time-0.05)
	}
	if !r.Animating() {
		t.Error("not animating")
	}
}

func TestResizeRescales(t *testing.T) {
	r := New()
	r.Update(State{Visitors: everyone()})
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	render(r, 720, 330, t0, true)
	f := r.creature["tang"].(*fish)
	x, y := f.x, f.y
	render(r, 360, 330, t0, true)
	if !near(f.x, x/2) || !near(f.y, y) {
		t.Errorf("fish at %v,%v after halving the width (was %v,%v)", f.x, f.y, x, y)
	}
	if r.world.rocks[0].x != 360*0.81 || r.world.burrows["mantis"].x != 216 {
		t.Error("layout not redone")
	}
	if len(active(r)) != 3 {
		t.Errorf("%d out at 360 wide", len(active(r)))
	}
}

func TestStillMode(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := New()
	render(r, 400, 350, t0, true)
	r.Update(State{FedSession: 1})
	r.Update(State{FedSession: 2})
	if len(r.world.pellets) != 0 {
		t.Error("still reef fed")
	}
	render(r, 400, 350, t0.Add(time.Second), true)
	if r.time != 0 || r.Animating() {
		t.Error("still reef moved")
	}
	// Motion back: frames resume from the next one, without a jump.
	render(r, 400, 350, t0.Add(2*time.Second), false)
	if r.time != 0 || !r.Animating() {
		t.Errorf("time %v on resuming", r.time)
	}
	r.Update(State{FedSession: 3})
	if len(r.world.pellets) == 0 {
		t.Fatal("no food once moving")
	}
	render(r, 400, 350, t0.Add(2*time.Second+100*time.Millisecond), false)
	if !near(r.time, 0.1) {
		t.Errorf("time %v", r.time)
	}
	// Still again: the food goes and the tank is set up anew, as at the start.
	r.Update(State{FedSession: 3, Visitors: everyone()})
	render(r, 400, 350, t0.Add(3*time.Second), true)
	if len(r.world.pellets) != 0 {
		t.Error("pellets kept when still")
	}
	fresh := New()
	fresh.SetStill(true)
	fresh.Update(State{Visitors: everyone()})
	render(fresh, 400, 350, t0, true)
	for _, id := range fresh.order {
		a, b := r.creature[id], fresh.creature[id]
		if f, ok := a.(*fish); ok {
			if g := b.(*fish); f.x != g.x || f.y != g.y || f.z != g.z {
				t.Errorf("%s at %v,%v, a fresh still reef has it at %v,%v", id, f.x, f.y, g.x, g.y)
			}
			continue
		}
		ca, cb := a.(crawlerCreature).base(), b.(crawlerCreature).base()
		if ca.x != cb.x || ca.active != cb.active || ca.state != cb.state {
			t.Errorf("%s at %v (%v), a fresh still reef has it at %v (%v)", id, ca.x, ca.active, cb.x, cb.active)
		}
	}
}

func TestPortraitsPaint(t *testing.T) {
	for _, id := range append(append([]string(nil), Species...), "unknown") {
		img := ui.Render(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				DrawPortrait(p, ui.Rect{X: r.X + 10, Y: r.Y + 10, W: 46, H: 32}, id, Palette{})
			})
		}, 66, 52, 1)
		inside, outside := 0, 0
		b := img.Bounds()
		bg := img.At(0, 0)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if img.At(x, y) == bg {
					continue
				}
				if x >= 10 && x < 56 && y >= 10 && y < 42 {
					inside++
				} else {
					outside++
				}
			}
		}
		if inside < 40 || outside > 0 {
			t.Errorf("%s: %d pixels painted inside, %d outside", id, inside, outside)
		}
	}
}
