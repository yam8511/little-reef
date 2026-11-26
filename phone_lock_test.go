package main

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"little-reef/internal/lockscreen"
)

// fakeLock stands in for the Android notification in tests.
type fakeLock struct {
	shows      []lockscreen.Round
	hides      int
	restricted bool
	allowed    int
	blocked    bool
	opened     int
	action     func(lockscreen.Action)
}

func (f *fakeLock) Show(r lockscreen.Round) error { f.shows = append(f.shows, r); return nil }
func (f *fakeLock) Hide() error                   { f.hides++; return nil }
func (f *fakeLock) OnAction(fn func(lockscreen.Action)) io.Closer {
	f.action = fn
	return io.NopCloser(nil)
}
func (f *fakeLock) BackgroundRestricted() bool { return f.restricted }
func (f *fakeLock) AllowBackground() error     { f.allowed++; return nil }
func (f *fakeLock) NotificationsBlocked() bool { return f.blocked }
func (f *fakeLock) OpenNotificationSettings() error {
	f.opened++
	return nil
}

func TestDesktopsHaveNoLockScreen(t *testing.T) {
	if onPhone {
		t.Skip("runs on desktops")
	}
	a, _ := testAquarium(t)
	d := newDesktop(a)
	d.startLockScreen()
	if d.lock != nil {
		t.Fatal("a desktop build got a lock screen")
	}
	if _, err := lockscreen.NewTimer(); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("NewTimer on a desktop: %v", err)
	}
	d.syncLockScreen(a.Get()) // must do nothing without one
}

func TestLockRoundSaysWhatTheRoundIs(t *testing.T) {
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.Local)
	s := Snapshot{Session: Session{Kind: "focus", Status: "idle", DurationMS: 1500000}}
	if _, show := lockRound(s, now, lockscreen.StyleLive); show {
		t.Fatal("an idle timer showed on the lock screen")
	}
	s.Session = Session{ID: 1, Kind: "focus", Status: "running", Task: "寫週報", DurationMS: 1500000, ElapsedMS: 300000}
	r, show := lockRound(s, now, lockscreen.StyleLive)
	if !show || !r.Running || r.Title != "專注中" || r.Text != "寫週報" || !r.EndsAt.Equal(now.Add(20*time.Minute)) || r.Pause != "暫停" || r.Total != 25*time.Minute {
		t.Fatalf("running focus: %+v", r)
	}
	s.Session.Status = "paused"
	r, _ = lockRound(s, now, lockscreen.StyleLive)
	if r.Running || r.Title != "已暫停" || r.Remaining != 20*time.Minute || r.Resume != "繼續" {
		t.Fatalf("paused: %+v", r)
	}
	s.Session = Session{ID: 2, Kind: "break", Status: "running", DurationMS: 300000}
	if r, _ = lockRound(s, now, lockscreen.StyleLive); r.Title != "休息中" || r.Text != "離開螢幕，看看遠處" {
		t.Fatalf("break: %+v", r)
	}
}

func TestLockScreenFollowsTheRoundAndItsButtons(t *testing.T) {
	a, now := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	lock := &fakeLock{restricted: true}
	d.useLockScreen(lock)
	tt := ui.NewTester(d.ui.view, 1100, 800)
	if _, err := a.Start("focus", "讀書"); err != nil {
		t.Fatal(err)
	}
	advance(a, now, 5)
	tt.Frame() // runs the updates the clock posted
	if len(lock.shows) != 1 || !lock.shows[0].Running || lock.shows[0].Text != "讀書" {
		t.Fatalf("a running round was shown %d times: %+v", len(lock.shows), lock.shows)
	}
	if !d.ui.backgroundPrompt || !tt.HasText("讓計時在鎖定時繼續") {
		t.Fatalf("no background prompt while restricted: %v", tt.Texts())
	}
	if err := tt.Click("前往設定"); err != nil {
		t.Fatal(err)
	}
	if lock.allowed != 1 || d.ui.backgroundPrompt {
		t.Fatal("前往設定 did not open the system's settings")
	}
	// The notification's own button pauses the round.
	lock.action(lockscreen.ActionPause)
	settle(t, tt, func() bool { return a.Get().Session.Status == "paused" })
	tt.Frame()
	if last := lock.shows[len(lock.shows)-1]; last.Running || last.Title != "已暫停" {
		t.Fatalf("pausing did not update the lock screen: %+v", last)
	}
	lock.action(lockscreen.ActionResume)
	settle(t, tt, func() bool { return a.Get().Session.Status == "running" })
	if _, err := a.End(); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if lock.hides != 1 {
		t.Fatalf("ending the round did not clear the lock screen: %d", lock.hides)
	}
	if d.ui.backgroundPrompt {
		t.Fatal("the background prompt came back")
	}
}

func TestPhonesKeepCountingWhileTheAppIsFrozen(t *testing.T) {
	for _, wall := range []bool{false, true} {
		a, now := testAquarium(t)
		a.wallClock = wall
		if _, err := a.Start("focus", ""); err != nil {
			t.Fatal(err)
		}
		advance(a, now, 2)
		*now = now.Add(20 * time.Second) // frozen in the background
		a.tick()
		s := a.Get().Session
		if wall && (s.Status != "running" || s.ElapsedMS < 22000) {
			t.Fatalf("phone: the frozen time was lost: %+v", s)
		}
		if !wall && s.Status != "paused" {
			t.Fatalf("desktop: a long gap no longer pauses as sleep: %+v", s)
		}
	}
}

func TestBlockedNotificationsAreAskedForAndResentOnReturn(t *testing.T) {
	a, now := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	lock := &fakeLock{blocked: true, restricted: true}
	d.useLockScreen(lock)
	tt := ui.NewTester(d.ui.view, 1100, 800)
	if _, err := a.Start("focus", ""); err != nil {
		t.Fatal(err)
	}
	advance(a, now, 1)
	tt.Frame()
	// Turned-off notifications come first: nothing shows until they are on.
	if !d.ui.notificationsPrompt || d.ui.backgroundPrompt || !tt.HasText("開啟通知，才看得到倒數") {
		t.Fatalf("no notifications prompt: %v", tt.Texts())
	}
	if err := tt.Click("前往設定"); err != nil {
		t.Fatal(err)
	}
	if lock.opened != 1 || d.ui.notificationsPrompt {
		t.Fatal("前往設定 did not open the notification settings")
	}
	// Back from the settings, the round shows again at once.
	shown := len(lock.shows)
	lock.blocked = false
	d.resendLockScreen()
	if len(lock.shows) != shown+1 || !lock.shows[len(lock.shows)-1].Running {
		t.Fatalf("coming back did not show the round again: %d -> %d", shown, len(lock.shows))
	}
	if d.ui.notificationsPrompt || d.ui.backgroundPrompt {
		t.Fatal("a prompt came back after it was answered")
	}
}

func TestPhoneSettingsShowTheLockScreenStatus(t *testing.T) {
	setPhone(t, true)
	for _, blocked := range []bool{true, false} {
		_, s := settingsFixture(t)
		lock := &fakeLock{blocked: blocked}
		s.desktop.lock = lock
		tt := ui.NewTester(s.view, 400, 1000)
		if got := tt.HasText("開啟通知設定"); got != blocked {
			t.Fatalf("blocked=%v: notification settings button shown=%v: %v", blocked, got, tt.Texts())
		}
		if blocked {
			if err := tt.Click("開啟通知設定"); err != nil {
				t.Fatal(err)
			}
			if lock.opened != 1 {
				t.Fatal("the button did not open the notification settings")
			}
		}
	}
}

func TestLockScreenStyleIsSavedAndShownAtOnce(t *testing.T) {
	a, now := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	lock := &fakeLock{}
	d.useLockScreen(lock)
	if d.lockStyle != lockscreen.StyleLive {
		t.Fatalf("default style: %q", d.lockStyle)
	}
	tt := ui.NewTester(d.ui.view, 1100, 800)
	if _, err := a.Start("focus", ""); err != nil {
		t.Fatal(err)
	}
	advance(a, now, 1)
	tt.Frame()
	if err := d.setLockStyle(lockscreen.StyleLarge); err != nil {
		t.Fatal(err)
	}
	last := lock.shows[len(lock.shows)-1]
	if last.Style != lockscreen.StyleLarge || last.End != "結束這一輪" {
		t.Fatalf("the round was not shown in the new style: %+v", last)
	}
	// A later launch keeps the choice.
	again := newDesktop(a)
	again.useLockScreen(&fakeLock{})
	if again.lockStyle != lockscreen.StyleLarge {
		t.Fatalf("the style was not kept: %q", again.lockStyle)
	}
}

func TestEndFromTheNotificationAsksInTheApp(t *testing.T) {
	a, now := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	lock := &fakeLock{}
	d.useLockScreen(lock)
	tt := ui.NewTester(d.ui.view, 1100, 800)
	if _, err := a.Start("focus", ""); err != nil {
		t.Fatal(err)
	}
	advance(a, now, 1)
	tt.Frame()
	lock.action(lockscreen.ActionEnd)
	tt.Frame()
	if !tt.HasText("這一段時間，也算數。") || a.Get().Session.Status != "running" {
		t.Fatalf("ending from the notification did not ask first: %v", tt.Texts())
	}
}

func TestEndConfirmedOnTheMediaCardEndsAtOnce(t *testing.T) {
	a, now := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	lock := &fakeLock{}
	d.useLockScreen(lock)
	tt := ui.NewTester(d.ui.view, 1100, 800)
	if _, err := a.Start("focus", ""); err != nil {
		t.Fatal(err)
	}
	advance(a, now, 1)
	tt.Frame()
	if r, _ := lockRound(a.Get(), *now, lockscreen.StyleMedia); r.ConfirmEnd == "" {
		t.Fatal("the card's end button has no label to confirm with")
	}
	lock.action(lockscreen.ActionEndNow)
	for deadline := time.Now().Add(5 * time.Second); sessionActive(a.Get()); {
		if time.Now().After(deadline) {
			t.Fatal("ending confirmed on the card did not end the round")
		}
		tt.Frame()
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()
	if tt.HasText("這一段時間，也算數。") {
		t.Fatalf("ending confirmed on the card asked again: %v", tt.Texts())
	}
}

func TestPhoneSettingsSwitchTheLockScreenStyle(t *testing.T) {
	setPhone(t, true)
	_, s := settingsFixture(t)
	s.desktop.useLockScreen(&fakeLock{})
	tt := ui.NewTester(s.view, 400, 1100)
	if err := tt.Click("大字倒數：大大的倒數與圖示按鈕，只在鎖定畫面與通知欄"); err != nil {
		t.Fatal(err)
	}
	if s.desktop.lockStyle != lockscreen.StyleLarge {
		t.Fatalf("choosing the large style did not switch: %q", s.desktop.lockStyle)
	}
}

func TestTheMediaCardStyleIsKept(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	d.useLockScreen(&fakeLock{})
	if err := d.setLockStyle(lockscreen.StyleMedia); err != nil {
		t.Fatal(err)
	}
	again := newDesktop(a)
	again.useLockScreen(&fakeLock{})
	if again.lockStyle != lockscreen.StyleMedia {
		t.Fatalf("the media card style was not kept: %q", again.lockStyle)
	}
}
