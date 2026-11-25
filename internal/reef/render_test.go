package reef

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Set REEF_RENDER_DIR to write snapshots of the stage and the portraits there.

var realistic = Palette{
	Sand: ui.Hex("#718371"), Coral: ui.Hex("#bd918c"), Seaweed: ui.Hex("#529a86"),
	Bubble: ui.Hex("#b4cfb1"), Accent: ui.Hex("#a8deba"),
}

func stage(p *ui.Painter, r ui.Rect) {
	p.FillRadialGradient(r, ui.RadialGradient{Stops: []ui.GradientStop{{Color: ui.Hex("#194250"), At: 0}, {Color: ui.Hex("#102e3a"), At: .55}, {Color: ui.Hex("#0c1820"), At: 1}}}, 0)
}

func everyone() []Visitor {
	var vs []Visitor
	for i, id := range Species[1:] {
		vs = append(vs, Visitor{Species: id, Date: time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)})
	}
	return vs
}

func writePNG(t *testing.T, name string, view func(c *ui.Context), w, h int, scale float32) {
	dir := os.Getenv("REEF_RENDER_DIR")
	img := ui.Render(view, w, h, scale)
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestRenderStage(t *testing.T) {
	for _, still := range []bool{false, true} {
		r := New()
		r.SetPalette(realistic)
		r.Update(State{Completed: 30, Visitors: everyone()})
		start := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
		now := start
		draw := func(p *ui.Painter, rc ui.Rect) { stage(p, rc); r.Draw(p, rc, now, still) }
		view := func(c *ui.Context) { ui.Box(c).Fill().Draw(draw) }
		ui.Render(view, 720, 330, 1)
		// The same clock as the browser driver (js/reef.html): a step every 50 ms.
		for i := 0; i < 240; i++ {
			now = now.Add(50 * time.Millisecond)
			if !r.world.still {
				r.advance(now)
			}
		}
		if !still {
			r.Update(State{Completed: 30, Visitors: everyone(), FedSession: 7})
			for i := 0; i < 120; i++ {
				now = now.Add(50 * time.Millisecond)
				r.advance(now)
			}
		}
		for _, id := range r.order {
			switch c := r.creature[id].(type) {
			case *fish:
				t.Logf("%s x=%.6f y=%.6f z=%.6f active=false state=%s", id, c.x, c.y, c.z, c.mode)
			case crawlerCreature:
				b := c.base()
				t.Logf("%s x=%.6f y=%.6f z=%.6f active=%v state=%s", id, b.x, b.y, b.z, b.active, b.state)
			}
		}
		t.Logf("pellets=%d time=%.6f", len(r.world.pellets), r.time)
		name := "stage.png"
		if still {
			name = "stage-still.png"
		}
		writePNG(t, name, view, 720, 330, 2)
	}
}

func TestRenderPortraits(t *testing.T) {
	ids := append(append([]string(nil), Species...), "unknown")
	for _, size := range []struct {
		name string
		w, h float32
	}{{"small", 46, 32}, {"large", 230, 160}} {
		cols := 6
		rows := (len(ids) + cols - 1) / cols
		view := func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(r, ui.Hex("#102e3a"), 0)
				for i, id := range ids {
					x, y := float32(i%cols)*(size.w+4)+2, float32(i/cols)*(size.h+4)+2
					cell := ui.Rect{X: r.X + x, Y: r.Y + y, W: size.w, H: size.h}
					p.Fill(cell, ui.Hex("#173f4c"), 0)
					DrawPortrait(p, cell, id, realistic)
				}
			})
		}
		w, h := cols*int(size.w+4), rows*int(size.h+4)
		writePNG(t, "portraits-"+size.name+".png", view, w, h, 2)
		writePNG(t, "portraits-"+size.name+"-1x.png", view, w, h, 1)
	}
}
