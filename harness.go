//go:build harness

// The MCP harness serves the app's native views headlessly, so an MCP client
// (Claude Code reads .mcp.json) can inspect, click, type into and screenshot
// them without a desktop window:
//
//	go run -tags harness . [-listen 127.0.0.1:8765]
//
// Each fixture runs on its own throwaway data directory, seeded like a reef a
// few weeks in, and the ambience is muted. Nothing touches the real archive.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	mygomcp "github.com/auron-labs/mygo-mcp"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8765", "MCP 端點監聽的迴送位址")
	flag.Parse()
	audio.SetVolume(0)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fixtures := []mygomcp.Fixture{
		harnessFixture("Aquarium", fullWidth, fullHeight, true, false, func(d *Desktop) func(*ui.Context) { return d.ui.view }),
		harnessFixture("Aquarium (narrow)", fullMinWidth, fullMinHeight, true, false, func(d *Desktop) func(*ui.Context) { return d.ui.view }),
		harnessFixture("Aquarium (first run)", fullWidth, fullHeight, false, false, func(d *Desktop) func(*ui.Context) { return d.ui.view }),
		harnessFixture("Compact widget", 400, 350, true, true, func(d *Desktop) func(*ui.Context) { return d.ui.view }),
		harnessFixture("Tray panel", 320, 420, true, false, func(d *Desktop) func(*ui.Context) { return d.panelView }),
	}
	log.Printf("MCP harness on http://%s/mcp", *listen)
	if err := mygomcp.Run(ctx, *listen, fixtures); err != nil {
		log.Fatal(err)
	}
}

// harnessFixture builds an isolated desktop state for one view.
func harnessFixture(name string, w, h int, seeded, compact bool, view func(*Desktop) func(*ui.Context)) mygomcp.Fixture {
	return mygomcp.Fixture{Name: name, Width: w, Height: h, Factory: func() (func(*ui.Context), func(), error) {
		dir, err := os.MkdirTemp("", "little-reef-harness-")
		if err != nil {
			return nil, nil, err
		}
		cleanup := func() { _ = os.RemoveAll(dir) }
		if seeded {
			if err := seedArchive(dir); err != nil {
				cleanup()
				return nil, nil, err
			}
		}
		a, err := openAquarium(dir)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		d := newDesktop(a)
		d.compact.Store(compact)
		a.onChange = d.changed
		d.ui.apply(a.Get())
		go a.run()
		return view(d), func() { d.sound.dispose(); _ = a.shutdown(); cleanup() }, nil
	}}
}

// seedArchive writes a reef a few weeks in: 14 rounds, five visitors, a
// short journal.
func seedArchive(dir string) error {
	now := time.Now()
	day := func(back int, hour int) string {
		return time.Date(now.Year(), now.Month(), now.Day()-back, hour, 30, 0, 0, time.Local).Format(time.RFC3339)
	}
	visitors := []Visitor{}
	for i, s := range []string{"clown", "tang", "cleaner", "hermit", "emerald"} {
		for _, sp := range species {
			if sp.Species == s {
				visitors = append(visitors, Visitor{Species: s, Name: sp.Name, Date: day(5-i, 10), Visits: max(1, 3-i)})
			}
		}
	}
	snap := Snapshot{
		Revision:    5,
		Preferences: Preferences{FocusMinutes: 25, BreakMinutes: 5, Sound: "rain", Volume: .35, ChimeEnabled: true},
		Session:     Session{ID: 40, Kind: "focus", Status: "idle", DurationMS: 25 * 60 * 1000},
		Completed:   14, TotalSecs: 21600, TodaySecs: 3000, TodayRounds: 2,
		Records: []Record{
			{ID: 40, Task: "寫週報", Seconds: 1500, Completed: true, Date: day(0, 9)},
			{ID: 39, Task: "讀幾頁書", Seconds: 1500, Completed: true, Date: day(0, 8)},
			{ID: 38, Seconds: 620, Date: day(1, 20)},
		},
		Visitors: visitors, LastVisitor: visitors[len(visitors)-1].Name,
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "aquarium.json"), data, 0o600)
}
