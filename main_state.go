package main

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"little-reef/internal/reef"
)

// mainUI is the main window's presentation state: the native port of the
// former page's useAquarium hook. Every field is read and written on the
// main thread only (views, Window.Update callbacks).
type mainUI struct {
	d *Desktop

	snap    *Snapshot // last applied snapshot
	busy    bool
	kind    string // selected tab: "focus" | "break"
	task    string // the "這一輪，想做什麼？" draft
	endOpen bool

	soundPlaying bool // the user's intent; playback also needs soundEnabled
	soundSession int64
	audioKey     string
	soundStatus  string

	volumeDrag   float64
	volumeActive bool

	reviewOpen, journalOpen bool

	toast    string
	toastSeq int

	// Phones ask once to let the app run in the background (phone_lock.go).
	backgroundAsked, backgroundPrompt, notificationsPrompt bool
	// settingsOpen shows the settings page over the window, on phones.
	settingsOpen bool

	reef      *reef.Reef
	reefTheme string
}

func newMainUI(d *Desktop) *mainUI {
	return &mainUI{d: d, kind: "focus", soundSession: -1, soundStatus: soundStatusInitial, reef: reef.New()}
}

// formatClock turns milliseconds left into MM:SS, rounding up as the page did.
func formatClock(ms int64) string {
	seconds := max(int64(0), int64(math.Ceil(float64(ms)/1000)))
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

// timerView is everything the timer panel derives from a snapshot and the
// selected tab (deriveTimer in the former page).
type timerView struct {
	Time, Status, Primary, PrimaryIcon, Placeholder, Footnote string
	Progress                                                  float32 // 0…1 of the ring
	Paused, TaskDisabled, EndHidden, FeedDisabled             bool
	FeedLabel, GrowthLabel, NextGrowth, Caption               string
}

func tierName(completed int) string {
	switch {
	case completed < 3:
		return "初生"
	case completed < 12:
		return "萌芽"
	case completed < 24:
		return "繁盛"
	}
	return "小小綠洲"
}

func deriveTimer(s Snapshot, kind string) timerView {
	session, p := s.Session, s.Preferences
	active := sessionActive(s)
	sameKind := session.Kind == kind
	complete := session.Status == "complete" && sameKind
	chosen := int64(p.FocusMinutes) * 60000
	if kind == "break" {
		chosen = int64(p.BreakMinutes) * 60000
	}
	duration, elapsed := chosen, int64(0)
	if active || complete {
		duration, elapsed = session.DurationMS, session.ElapsedMS
	}
	v := timerView{Time: formatClock(duration - elapsed), Status: "準備好，就開始", Paused: session.Status == "paused"}
	if active {
		switch {
		case session.Status == "paused":
			v.Status = "暫停一下，沒關係"
		case session.Kind == "focus":
			v.Status = "小魚陪著你，慢慢來"
		default:
			v.Status = "離開螢幕，看看遠處"
		}
	} else if complete {
		v.Status = "休息結束 · 歡迎回來"
		if session.Kind == "focus" {
			v.Status = "專注完成 · 海底長大了"
		}
	}
	v.Primary, v.PrimaryIcon = "開始專注", "play"
	if kind == "break" {
		v.Primary = "開始休息"
	}
	switch {
	case active && session.Status == "running":
		v.Primary, v.PrimaryIcon = "暫停一下", "pause"
	case active:
		v.Primary = "繼續這一輪"
	case complete && session.Kind == "focus":
		v.Primary, v.PrimaryIcon = "開始休息", "coffee"
	case complete && session.Kind == "break":
		v.Primary = "開始下一輪"
	}
	growth := s.Completed
	v.GrowthLabel = fmt.Sprintf("%s · %d 輪專注", tierName(growth), growth)
	switch {
	case growth == 0:
		v.NextGrowth = "完成一輪，長出第一株水草"
	case growth >= 24:
		v.NextGrowth = "繼續積累專注，認識更多海底朋友"
	case growth%3 == 0:
		v.NextGrowth = "再一輪，海底會長出新植物"
	default:
		v.NextGrowth = "再一輪，水草會長高一些"
	}
	v.Caption = "一條魚，一片海，等你慢慢來。"
	switch {
	case s.LastVisitor != "" && complete && session.Kind == "focus":
		v.Caption = s.LastVisitor + "來做客了，已留在圖鑑裡。"
	case active && session.Kind == "focus":
		v.Caption = "你專注做事，海底安靜陪著你。"
	case active && session.Kind == "break":
		v.Caption = "魚兒不著急，你也可以休息一下。"
	}
	if duration > 0 {
		v.Progress = float32(min(1, float64(elapsed)/float64(duration)))
	}
	v.TaskDisabled = active || kind == "break"
	v.Placeholder = "比如讀幾頁書，寫一小段程式碼"
	if kind == "break" {
		v.Placeholder = "去喝杯水，讓眼睛也休息一下"
	}
	v.EndHidden = !active
	v.Footnote = "不著急，每一點專注都會留下痕跡。"
	if active {
		v.Footnote = "中斷也會記錄時間，已有成長不會消失。"
	}
	v.FeedDisabled = session.Kind != "break" || session.Status != "running" || s.FedSession == session.ID
	v.FeedLabel = "撒一點魚食"
	if s.FedSession == session.ID && session.Kind == "break" {
		v.FeedLabel = "魚兒吃飽啦"
	}
	return v
}

// apply takes a newer snapshot: sound intent, chime, selected tab, reef and
// window title follow it exactly as the page's apply() did.
func (m *mainUI) apply(next Snapshot) {
	previous := m.snap
	if previous != nil && next.Revision < previous.Revision {
		return
	}
	m.snap = &next
	session := next.Session
	if m.soundSession != session.ID {
		// Loading an already-running timer must not autoplay on start.
		m.soundPlaying = previous != nil && session.Status == "running" && session.Kind == "focus" && next.Preferences.SoundEnabled
		m.soundSession = session.ID
	} else if previous != nil && previous.Session.Status != session.Status {
		m.soundPlaying = session.Status == "running" && session.Kind == "focus"
	}
	if !next.Preferences.SoundEnabled || session.Status == "paused" || session.Status == "complete" {
		m.soundPlaying = false
	}
	if previous != nil && session.ID == previous.Session.ID && session.Status == "complete" && previous.Session.Status != "complete" && next.Preferences.ChimeEnabled {
		m.d.sound.chime()
	}
	if session.Status == "running" || session.Status == "paused" || session.Status == "complete" {
		m.kind = session.Kind
	}
	if m.endOpen && !sessionActive(next) {
		m.endOpen = false
	}
	m.reef.Update(reefState(next))
	m.updateAudio()
	m.updateTitle()
}

func reefState(s Snapshot) reef.State {
	visitors := make([]reef.Visitor, len(s.Visitors))
	for i, v := range s.Visitors {
		visitors[i] = reef.Visitor{Species: v.Species, Date: v.Date}
	}
	return reef.State{Completed: s.Completed, Visitors: visitors, FedSession: s.FedSession}
}

func (m *mainUI) updateTitle() {
	if m.d.win == nil || m.snap == nil {
		return
	}
	title := "小海底 · 桌面水族館"
	if sessionActive(*m.snap) {
		title = deriveTimer(*m.snap, m.kind).Time + " · 小海底"
	}
	if m.d.win.Title() != title {
		m.d.win.SetTitle(title)
	}
}

// updateAudio hands the soundscape the live key when it changed.
func (m *mainUI) updateAudio() {
	if m.snap == nil {
		return
	}
	p := m.snap.Preferences
	playing := m.soundPlaying && p.SoundEnabled
	key := fmt.Sprintf("%v:%s:%v", playing, p.Sound, p.Volume)
	if key != m.audioKey {
		m.audioKey = key
		m.d.sound.request(playing, p.Sound, p.Volume)
	}
}

func (m *mainUI) showToast(message string) {
	m.toast = message
	m.toastSeq++
	seq := m.toastSeq
	time.AfterFunc(5500*time.Millisecond, func() {
		m.d.update(func() {
			if m.toastSeq == seq {
				m.toast = ""
			}
		})
	})
}

// action runs a mutating call off the main thread with the busy lock, applies
// its snapshot and surfaces its error as a toast.
func (m *mainUI) action(call func() (Snapshot, error), then func(Snapshot)) {
	if m.busy || m.snap == nil {
		return
	}
	m.busy = true
	go func() {
		next, err := call()
		m.d.update(func() {
			m.busy = false
			if err != nil {
				msg := err.Error()
				if msg == "" {
					msg = "操作未完成，請稍後再試。"
				}
				m.showToast(msg)
				return
			}
			if next.Revision != 0 {
				m.apply(next)
			}
			if then != nil {
				then(next)
			}
		})
	}()
}

func (m *mainUI) primary() {
	s := m.snap
	if s == nil {
		return
	}
	a := m.d.a
	session := s.Session
	m.action(func() (Snapshot, error) {
		switch session.Status {
		case "running":
			return a.Pause()
		case "paused":
			return a.Resume()
		}
		kind := m.kind
		if session.Status == "complete" && session.Kind == kind {
			kind = map[string]string{"focus": "break", "break": "focus"}[session.Kind]
		}
		return a.Start(kind, m.task)
	}, nil)
}

func (m *mainUI) endSession() { m.action(m.d.a.End, nil) }
func (m *mainUI) feed()       { m.action(m.d.a.Feed, nil) }

func (m *mainUI) setPreferences(change func(*Preferences), then func(Snapshot)) {
	if m.snap == nil {
		return
	}
	p := m.snap.Preferences
	change(&p)
	m.action(func() (Snapshot, error) { return m.d.a.SetPreferences(p) }, then)
}

func (m *mainUI) pickSound(kind string) {
	m.setPreferences(func(p *Preferences) { p.Sound = kind }, nil)
}

func (m *mainUI) setVolume(v float64) {
	m.setPreferences(func(p *Preferences) { p.Volume = v }, func(Snapshot) { m.volumeActive = false })
}

// toggleSound flips the ambience; preview while idle or on break is an
// explicit choice, so it plays then too.
func (m *mainUI) toggleSound() {
	before := m.snap
	if m.busy || before == nil {
		return
	}
	wanted := !m.soundPlaying
	m.setPreferences(func(p *Preferences) { p.SoundEnabled = wanted }, func(next Snapshot) {
		latest := m.snap
		if latest.Session.ID == before.Session.ID && latest.Session.Status == before.Session.Status {
			m.soundPlaying = wanted && latest.Preferences.SoundEnabled
		}
		m.audioKey = ""
		m.updateAudio()
	})
}

// volumeShown is the slider's value: the live drag, else the saved volume.
func (m *mainUI) volumeShown() float64 {
	if m.volumeActive {
		return m.volumeDrag
	}
	if m.snap == nil {
		return 0.35
	}
	return m.snap.Preferences.Volume
}

// audioState is what the tray panel shows about the ambience.
func (m *mainUI) audioState() (requested, running bool) {
	if m.snap == nil {
		return false, false
	}
	requested = m.soundPlaying && m.snap.Preferences.SoundEnabled
	return requested, requested && m.d.sound.playing()
}

// palette is the theme the snapshot asks for.
func (m *mainUI) palette() *palette {
	id := defaultTheme
	if m.snap != nil && m.snap.Theme != "" {
		id = m.snap.Theme
	}
	p := palettes.get(id)
	if m.reefTheme != p.ID {
		m.reefTheme = p.ID
		m.reef.SetPalette(p.reef())
	}
	return p
}

// themeUI is the MyGo theme the main window and tray panel draw with.
func (m *mainUI) themeUI(p *palette) *ui.Theme {
	return p.uiTheme()
}
