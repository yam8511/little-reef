package reef

// The species registry, ported from species.ts.

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Spec is one creature's model and habits. Lengths are in centimeters.
type Spec struct {
	ID         string
	Model      string
	Length     float64
	Zone       [2]float64
	Speed      float64
	Hover      float64
	Home       bool
	Wiggle     bool
	Burst      bool
	Upstream   bool
	Propulsion string
	Curious    bool
	Rocky      bool
	Range      float64
	Rock       bool
	Idle       float64
	Burrow     bool
	Depths     *[2]float64
	Sideways   bool
	Colors     map[string]string
}

// species is every creature's model and habits. The ids match the visitor
// list in state.go.
var species = map[string]*Spec{
	"resident": {ID: "resident", Model: "fish", Length: 8.5, Zone: [2]float64{0.16, 0.6}, Speed: 1.1, Hover: 0.35, Burst: true, Upstream: true, Propulsion: "tail",
		Colors: map[string]string{"back": "#df7a3e", "flank": "#f09a54", "belly": "#f8cf99", "fin": "#f4a24f", "iris": "#e9b45a"}},
	"clown": {ID: "clown", Model: "fish", Length: 7, Zone: [2]float64{0.3, 0.85}, Speed: 0.9, Hover: 0.5, Home: true, Wiggle: true, Propulsion: "tail",
		Colors: map[string]string{"back": "#dc6624", "flank": "#ee8538", "belly": "#f6a255", "fin": "#ee8a3c", "iris": "#f0a54a"}},
	"tang": {ID: "tang", Model: "fish", Length: 9.5, Zone: [2]float64{0.12, 0.72}, Speed: 1, Hover: 0.2, Propulsion: "pectoral",
		Colors: map[string]string{"back": "#1e4ead", "flank": "#3367c6", "belly": "#6c95d6", "fin": "#2c5ab6", "iris": "#22283a"}},
	"angel": {ID: "angel", Model: "fish", Length: 10, Zone: [2]float64{0.38, 0.8}, Speed: 0.65, Hover: 0.45, Rocky: true, Propulsion: "tail",
		Colors: map[string]string{"back": "#2b6d80", "flank": "#4f9a8e", "belly": "#a7c77d", "fin": "#3a8fa3", "iris": "#d8b450"}},
	"puffer": {ID: "puffer", Model: "fish", Length: 6.5, Zone: [2]float64{0.48, 0.88}, Speed: 0.5, Hover: 0.6, Curious: true, Propulsion: "fins",
		Colors: map[string]string{"back": "#e8d8be", "flank": "#f2eadb", "belly": "#fbf7ef", "fin": "#f1dfab", "iris": "#7dc6b4"}},
	"cleaner":   {ID: "cleaner", Model: "shrimp", Length: 6.5, Speed: 0.55, Range: 0.12, Rock: true, Idle: 1.4},
	"fire":      {ID: "fire", Model: "shrimp", Length: 6, Speed: 0.5, Range: 0.1, Rock: true, Idle: 2},
	"harlequin": {ID: "harlequin", Model: "shrimp", Length: 5.5, Speed: 0.3, Range: 0.14, Rock: true, Idle: 1.5},
	"pistol":    {ID: "pistol", Model: "shrimp", Length: 5.5, Speed: 0.6, Range: 0.08, Burrow: true, Idle: 0.8},
	"mantis":    {ID: "mantis", Model: "mantis", Length: 11, Speed: 0.7, Range: 0.14, Burrow: true, Idle: 1},
	"slipper":   {ID: "slipper", Model: "shrimp", Length: 9, Speed: 0.25, Range: 0.2, Depths: &[2]float64{0.15, 0.7}, Idle: 2.5},
	"hermit":    {ID: "hermit", Model: "hermit", Length: 4.5, Speed: 0.45, Range: 0.22, Depths: &[2]float64{0.1, 0.75}, Idle: 1.2},
	"emerald":   {ID: "emerald", Model: "crab", Length: 5.5, Speed: 0.45, Range: 0.1, Rock: true, Sideways: true, Idle: 1.6},
	"porcelain": {ID: "porcelain", Model: "crab", Length: 3.6, Speed: 0.4, Range: 0.06, Rock: true, Sideways: true, Idle: 3},
	"arrow":     {ID: "arrow", Model: "crab", Length: 8, Speed: 0.9, Range: 0.14, Rock: true, Idle: 1.2},
	"swimmer":   {ID: "swimmer", Model: "crab", Length: 9, Speed: 0.6, Range: 0.25, Depths: &[2]float64{0.1, 0.7}, Sideways: true, Idle: 1.4},
}

// Species lists the ids SPECIES knows, in the table's order.
var Species = []string{"resident", "clown", "tang", "angel", "puffer", "cleaner", "fire", "harlequin", "pistol", "mantis", "slipper", "hermit", "emerald", "porcelain", "arrow", "swimmer"}

// Known reports whether SPECIES has this id.
func Known(id string) bool {
	_, ok := species[id]
	return ok
}

// lighten mixes a "#rrggbb" hex toward white by amt (0…1).
func lighten(hex string, amt float64) string {
	n := jsParseHex(hex[min(1, len(hex)):])
	i := int64(0)
	if !math.IsNaN(n) {
		i = int64(int32(int64(n))) // JS's >> works on 32 bits
	}
	r := float64(i >> 16)
	g := float64((i >> 8) & 0xff)
	b := float64(i & 0xff)
	r += (255 - r) * amt
	g += (255 - g) * amt
	b += (255 - b) * amt
	return fmt.Sprintf("rgb(%s,%s,%s)", strconv.Itoa(int(toInt(r))), strconv.Itoa(int(toInt(g))), strconv.Itoa(int(toInt(b))))
}

// themedSpec returns the species record with theme overrides applied:
// a creature palette rewrites a fish species' body palette. Themes without
// one keep every species' own colors.
func themedSpec(id string, pal *Palette) *Spec {
	spec, ok := species[id]
	if !ok {
		return nil
	}
	s := *spec
	if spec.Colors != nil && pal != nil && pal.FishColor != "" {
		f := pal.FishColor
		s.Colors = map[string]string{"back": f, "flank": lighten(f, 0.18), "belly": lighten(f, 0.38), "fin": f, "iris": spec.Colors["iris"]}
	}
	return &s
}

// speciesColors exposes the resolved palette for tests.
func speciesColors(id string, pal *Palette) map[string]string {
	if s := themedSpec(id, pal); s != nil {
		return s.Colors
	}
	return nil
}

// maxCreatures caps how many species share the tank at once; the visitor
// book still remembers everyone met. reef-realistic sets no cap.
func maxCreatures(pal *Palette) int {
	if pal == nil || pal.MaxCreatures == 0 {
		return 999
	}
	return pal.MaxCreatures
}

// creature is what the reef holds: a fish or a crawler.
type creature interface {
	isCrawler() bool
	depth() float64
	step(dt float64, w *world)
	draw(c *ctx, w *world)
	under(c *ctx, w *world)
	resize(sx, sy float64, w *world)
}

// crawlerCreature is a creature with the crawlers' routine.
type crawlerCreature interface {
	creature
	base() *crawler
	setActive(on bool, w *world, instant bool)
}

// create returns a new creature, or nil for an unknown species or one whose
// model is missing.
func create(id string, w *world, enter bool, pal *Palette) creature {
	spec := themedSpec(id, pal)
	if spec == nil {
		return nil
	}
	switch spec.Model {
	case "fish":
		return newFish(spec, w, enter)
	case "shrimp":
		return newShrimp(spec, w, enter)
	case "mantis":
		return newMantis(spec, w, enter)
	case "hermit":
		return newHermit(spec, w, enter)
	case "crab":
		return newCrab(spec, w, enter)
	}
	return nil
}

// DrawPortrait paints the still visitor-book portrait of a species into
// rect; unknown species get a plain silhouette.
func DrawPortrait(p *ui.Painter, rect ui.Rect, id string, pal Palette) {
	w, h := float64(rect.W), float64(rect.H)
	if w <= 0 || h <= 0 {
		return
	}
	c := newCtx(mat{1, 0, 0, 1, float64(rect.X), float64(rect.Y)})
	portrait(c, id, w, h, &pal)
	// A canvas shows nothing outside itself.
	p.Clip(rect, 0, func() { c.draw(p) })
}

func portrait(c *ctx, id string, w, h float64, pal *Palette) {
	spec := themedSpec(id, pal)
	if spec != nil {
		switch spec.Model {
		case "fish":
			fishPortrait(c, spec, w, h)
			return
		case "shrimp":
			shrimpPortrait(c, spec, w, h)
			return
		case "mantis":
			mantisPortrait(c, spec, w, h)
			return
		case "hermit":
			hermitPortrait(c, spec, w, h)
			return
		case "crab":
			crabPortrait(c, spec, w, h)
			return
		}
	}
	c.setFill(rgbaC(158, 189, 189, .35))
	c.beginPath()
	c.ellipse(w*0.46, h/2, w*0.28, h*0.2, 0, 0, math.Pi*2, false)
	c.fill()
	c.beginPath()
	c.moveTo(w*0.7, h/2)
	c.lineTo(w*0.86, h*0.3)
	c.lineTo(w*0.86, h*0.7)
	c.fill()
}
