package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestMenuPrimaryFollowsSessionTransitions(t *testing.T) {
	for _, tc := range []struct {
		status, kind, label string
		command             timerCommand
	}{
		{"idle", "focus", "開始專注", timerStartFocus},
		{"running", "focus", "暫停專注", timerPause},
		{"running", "break", "暫停休息", timerPause},
		{"paused", "focus", "繼續專注", timerResume},
		{"paused", "break", "繼續休息", timerResume},
		{"complete", "focus", "開始休息", timerStartBreak},
		{"complete", "break", "開始專注", timerStartFocus},
	} {
		a := panelPrimary(Snapshot{Session: Session{Kind: tc.kind, Status: tc.status}})
		if a.Label != tc.label || a.Command != tc.command {
			t.Errorf("%s/%s: %+v", tc.kind, tc.status, a)
		}
	}
}

func TestQuickPanelCompletedFocusOffersBreak(t *testing.T) {
	a, _ := testAquarium(t)
	a.state.Session.Status = "complete"
	d := newDesktop(a)
	tt := ui.NewTester(d.panelView, 320, 460)
	if err := tt.Click("開始休息"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for a.Get().Session.Kind != "break" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := a.Get().Session; got.Kind != "break" || got.Status != "running" {
		t.Fatalf("next round: %+v", got)
	}
}

func TestQuickPanelNavigationFitsWithTaskAndError(t *testing.T) {
	a, _ := testAquarium(t)
	a.state.Session.Task = "正在讀一本需要慢慢讀完的技術書"
	d := newDesktop(a)
	d.timerReq.begin("failed", time.Second)
	d.timerReq.done("failed", false, "存檔暫時無法寫入，請檢查目錄權限。")
	tt := ui.NewTester(d.panelView, 320, 420)
	for _, label := range []string{"開啟水族館", "掛件模式", "設定…"} {
		r, ok := tt.Find(label)
		if !ok || r.Y < 0 || r.Y+r.H > 420 {
			t.Fatalf("navigation %s is clipped: %+v", label, r)
		}
	}
}

func TestEndingConfirmedRoundCannotEndANewerSession(t *testing.T) {
	a, _ := testAquarium(t)
	first, err := a.Start("focus", "first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.End(); err != nil {
		t.Fatal(err)
	}
	second, err := a.Start("focus", "second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.endSession(first.Session.ID); err == nil {
		t.Fatal("stale confirmation ended a newer round")
	}
	if got := a.Get().Session; got.ID != second.Session.ID || got.Status != "running" {
		t.Fatalf("new round was changed: %+v", got)
	}
}

func TestRestoreFullBoundsPreservesPositionAndFitsCurrentDisplay(t *testing.T) {
	work := mygo.Rectangle{X: 100, Y: 50, Width: 1440, Height: 900}
	for _, tc := range []struct{ saved, want mygo.Rectangle }{
		{mygo.Rectangle{X: 200, Y: 100, Width: 1000, Height: 740}, mygo.Rectangle{X: 200, Y: 100, Width: 1000, Height: 740}},
		{mygo.Rectangle{X: 2000, Y: 1400, Width: 1000, Height: 740}, mygo.Rectangle{X: 540, Y: 210, Width: 1000, Height: 740}},
	} {
		if got := restoredFullBounds(tc.saved, work); got != tc.want {
			t.Errorf("restoredFullBounds(%+v)=%+v, want %+v", tc.saved, got, tc.want)
		}
	}
}

func TestQuickPanelTimerStartsFromTheTray(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	tt := ui.NewTester(d.panelView, 320, 460)
	if err := tt.Click("開始專注"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for a.Get().Session.Status != "running" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.Get().Session.Status != "running" {
		t.Fatal("the tray panel did not start the timer")
	}
}

func TestQuickPanelShowsSoundStateAndPendingVolume(t *testing.T) {
	a, _ := testAquarium(t)
	a.state.Preferences.SoundEnabled = true
	d := newDesktop(a)
	d.ui.apply(a.Get())
	tt := ui.NewTester(d.panelView, 320, 460)
	if !tt.HasText("開啟聲音") || !tt.HasText("35%") {
		t.Fatalf("initial audio state: %v", tt.Texts())
	}
	d.ui.soundPlaying = true
	d.ui.volumeDrag, d.ui.volumeActive = .23, true
	d.ui.busy = true // keeps the drag from committing while the test looks
	tt.Frame()
	if !tt.HasText("關閉聲音") || !tt.HasText("23%") {
		t.Fatalf("live audio state: %v", tt.Texts())
	}
}

func TestCompactRequestFlipsTheLayoutWithoutAWindow(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	d.requestCompact(true)
	if !d.compact.Load() {
		t.Fatal("compact mode did not turn on")
	}
	d.requestCompact(false)
	if d.compact.Load() {
		t.Fatal("compact mode did not turn off")
	}
}
