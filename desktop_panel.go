package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Mini native panel anchored to the tray icon.

// --- request tracker ---

type requestState struct {
	mu        sync.Mutex
	current   string // requestId of in-flight command, "" if idle
	err       string
	timer     *time.Timer
	onTimeout func() // called when the timer fires (e.g. clear draft + redraw)
}

func (r *requestState) setOnTimeout(fn func()) {
	r.mu.Lock()
	r.onTimeout = fn
	r.mu.Unlock()
}

func (r *requestState) begin(id string, timeout time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != "" {
		return false
	}
	if r.timer != nil {
		r.timer.Stop()
	}
	r.current = id
	r.err = ""
	r.timer = time.AfterFunc(timeout, func() {
		r.mu.Lock()
		if r.current == id {
			r.current = ""
			r.err = "操作超時"
			cb := r.onTimeout
			r.mu.Unlock()
			if cb != nil {
				cb()
			}
			return
		}
		r.mu.Unlock()
	})
	return true
}

// done marks the request complete, but only if it matches the current id
// (stale ACKs from old requests must not clear new pending state).
func (r *requestState) done(id string, ok bool, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != id {
		return
	}
	if r.timer != nil {
		r.timer.Stop()
	}
	r.current = ""
	if !ok {
		r.err = errMsg
	} else {
		r.err = ""
	}
}

func (r *requestState) isPending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current != ""
}
func (r *requestState) isCurrent(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current == id
}
func (r *requestState) lastErr() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// --- volume draft (survives 1s Snapshot pushes while dragging) ---

// --- panel ---

func (d *Desktop) toggleFlyout() {
	mygo.RunOnMain(func() {
		if d.quitting.Load() {
			return
		}
		if d.flyout != nil && !d.flyout.IsDestroyed() {
			d.flyout.Close()
			d.flyout = nil
			return
		}
		d.openFlyout()
	})
}

// openFlyout is called on the UI thread and never toggles an existing panel closed.
func (d *Desktop) openFlyout() {
	if d.quitting.Load() || d.tray == nil {
		return
	}
	if d.flyout != nil && !d.flyout.IsDestroyed() {
		d.flyout.Show()
		d.flyout.Focus()
		d.flyout.Invalidate()
		return
	}
	f := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      d.tray,
		Placement: mygo.PlacementBottom,
		Focusable: true,
		Popover:   true,
		Width:     320,
		Height:    420,
		Content:   ui.View(d.panelView),
	})
	d.flyout = f
	f.SetTitle("小海底快捷面板")
	f.OnClose(func(*mygo.CloseEvent) {
		// OnClose fires on outside-press/ESC/programmatic Close; clear
		// the field before anything else so subsequent toggles don't
		// race against a still-visible window.
		if d.flyout == f {
			d.flyout = nil
			d.confirmEnd = false
			d.invalidateNativeMenus()
		}
	})
}

func (d *Desktop) invalidateFlyout() {
	if d.win == nil {
		return
	}
	d.win.Update(func() {
		if !d.quitting.Load() && d.flyout != nil && !d.flyout.IsDestroyed() {
			d.flyout.Invalidate()
		}
	})
}

// statusLine derives a human-readable status from the snapshot.
func (d *Desktop) statusLine(snap Snapshot) string {
	switch snap.Session.Status {
	case "running":
		if snap.Session.Kind == "focus" {
			return "專注進行中"
		}
		return "休息進行中"
	case "paused":
		return "已暫停"
	case "complete":
		if snap.Session.Kind == "focus" {
			return "專注完成，建議休息"
		}
		return "休息結束，開始下一輪"
	default: // idle
		if snap.Notice != "" {
			return snap.Notice
		}
		if snap.TodayRounds > 0 {
			return "已完成今日 " + fmt.Sprintf("%d", snap.TodayRounds) + " 輪"
		}
		return "空閒"
	}
}

// --- commands from the tray panel and menus ---

// requestAudio toggles the ambience or sets its volume, as the main window's
// own controls do.
func (d *Desktop) requestAudio(command string, volume float64) {
	d.update(func() {
		switch command {
		case "toggle":
			d.ui.toggleSound()
		case "set-volume":
			d.ui.volumeDrag, d.ui.volumeActive = volume, true
			d.ui.setVolume(volume)
		}
	})
}

// requestCompact switches the main window between the full aquarium and the
// small always-on-top widget.
func (d *Desktop) requestCompact(compact bool) {
	apply := func() {
		if d.win != nil && !d.win.IsDestroyed() && !d.quitting.Load() {
			d.applyCompactOn(d.win, compact)
			d.show()
		} else {
			d.compact.Store(compact)
		}
		d.refreshNativeMenus()
	}
	if d.win != nil {
		d.win.Update(apply)
	} else {
		apply()
	}
}
