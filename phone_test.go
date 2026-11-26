package main

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// setPhone runs the rest of a test as Android or iOS would.
func setPhone(t *testing.T, phone bool) {
	t.Helper()
	was := onPhone
	onPhone = phone
	t.Cleanup(func() { onPhone = was })
}

func TestSettingsTabsHideShortcutsOnlyOnPhones(t *testing.T) {
	for _, phone := range []bool{false, true} {
		setPhone(t, phone)
		_, s := settingsFixture(t)
		tt := ui.NewTester(s.view, 560, 680)
		if got := tt.HasText("快捷鍵"); got == phone {
			t.Fatalf("phone=%v: shortcuts tab shown=%v: %v", phone, got, tt.Texts())
		}
		for _, tab := range []string{"通用", "外觀", "備份與恢復"} {
			if !tt.HasText(tab) {
				t.Fatalf("phone=%v: tab %s missing", phone, tab)
			}
		}
		// The last tab is still the backup page, whichever tabs come before.
		if err := tt.Click("備份與恢復"); err != nil {
			t.Fatal(err)
		}
		if s.tab != settingsArchive || !tt.HasText("匯出存檔…") {
			t.Fatalf("phone=%v: backup tab opened tab %d: %v", phone, s.tab, tt.Texts())
		}
	}
}

// dragVolume drags the main window's volume slider from its left end to
// fraction x of its width, with a mouse or a finger.
func dragVolume(t *testing.T, tt *ui.Tester, x float32, finger bool) (ui.Rect, func()) {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleSlider, "音量")
	if !ok {
		t.Fatalf("volume slider missing: %v", tt.Texts())
	}
	r := info.Bounds
	y := r.Y + r.H/2
	if finger {
		tt.FingerDown(r.X+1, y)
		tt.FingerMove(r.X+r.W*x, y)
		return r, func() { tt.FingerUp(r.X+r.W*x, y) }
	}
	tt.Press(r.X+1, y)
	tt.Move(r.X+r.W*x, y)
	return r, func() { tt.Release(r.X+r.W*x, y) }
}

func TestVolumeSavesWhenTheDragEndsWithAMouseOrAFinger(t *testing.T) {
	for _, finger := range []bool{false, true} {
		a, _ := testAquarium(t)
		d := newDesktop(a)
		a.onChange = d.changed
		d.ui.apply(a.Get())
		tt := ui.NewTester(d.ui.view, 1100, 800)
		_, release := dragVolume(t, tt, 0.8, finger)
		if !d.ui.volumeActive || d.ui.volumeDrag < 0.7 {
			t.Fatalf("finger=%v: the drag did not move the volume: %v", finger, d.ui.volumeDrag)
		}
		if got := a.Get().Preferences.Volume; got != 0.35 {
			t.Fatalf("finger=%v: volume saved mid-drag: %v", finger, got)
		}
		release()
		settle(t, tt, func() bool { return a.Get().Preferences.Volume >= 0.7 && !d.ui.volumeActive })
	}
}

func TestSettingsCloseButtonOnlyOnPhones(t *testing.T) {
	for _, phone := range []bool{false, true} {
		setPhone(t, phone)
		_, s := settingsFixture(t)
		tt := ui.NewTester(s.view, 400, 680)
		for _, tab := range []string{"通用", "外觀", "備份與恢復"} {
			if err := tt.Click(tab); err != nil {
				t.Fatal(err)
			}
			if got := tt.HasText("關閉"); got != phone {
				t.Fatalf("phone=%v, tab %s: close button shown=%v", phone, tab, got)
			}
		}
		if phone {
			if err := tt.Click("關閉"); err != nil {
				t.Fatal(err)
			}
			if !s.closed {
				t.Fatal("the close button did not close settings")
			}
		}
	}
}

func TestSoundShowsItIsLoadingUntilTheRecordingStarts(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	var mu sync.Mutex
	var statuses []string
	d.sound.onStatus = func(message string, _ bool) {
		mu.Lock()
		defer mu.Unlock()
		statuses = append(statuses, message)
	}
	d.sound.request(true, "waves", 0) // muted; the first time it decodes
	mu.Lock()
	first := append([]string(nil), statuses...)
	mu.Unlock()
	if len(first) == 0 || first[0] != soundStatusLoading {
		t.Fatalf("no loading status before decoding: %q", first)
	}
	t.Cleanup(d.sound.dispose)
}

func TestPhoneSettingsIsAPageWithItsOwnBar(t *testing.T) {
	setPhone(t, true)
	for _, how := range []string{"back arrow", "back key", "close button"} {
		a, _ := testAquarium(t)
		d := newDesktop(a)
		d.ui.apply(a.Get())
		tt := ui.NewTester(d.ui.view, 400, 800)
		d.openSettingsPage()
		tt.Frame()
		if !tt.HasText("小海底設定") || !tt.HasText("通用") || tt.HasText("快捷鍵") {
			t.Fatalf("%s: settings page: %v", how, tt.Texts())
		}
		switch how {
		case "back arrow":
			if err := tt.Click("返回"); err != nil {
				t.Fatal(err)
			}
		case "back key": // Android's back closes what Escape closes
			tt.Key(0, ui.KeyEscape)
		case "close button":
			if err := tt.Click("關閉"); err != nil {
				t.Fatal(err)
			}
		}
		tt.Frame()
		if d.ui.settingsOpen || d.settings != nil || tt.HasText("小海底設定") {
			t.Fatalf("%s did not close the settings page: %v", how, tt.Texts())
		}
		if !tt.HasText("海底的聲音") {
			t.Fatalf("%s: the aquarium is not back: %v", how, tt.Texts())
		}
	}
}

func TestPhoneSettingsPageStaysClearOfTheSystemBars(t *testing.T) {
	setPhone(t, true)
	a, _ := testAquarium(t)
	d := newDesktop(a)
	d.ui.apply(a.Get())
	tt := ui.NewTester(d.ui.view, 400, 800)
	tt.SetInsets(36, 0, 24, 0) // a status bar and a gesture bar
	d.openSettingsPage()
	tt.Frame()
	back, ok := tt.Find("返回")
	if !ok || back.Y < 36 {
		t.Fatalf("the page's bar sits under the status bar: %+v", back)
	}
	closeButton, ok := tt.Find("關閉")
	if !ok || closeButton.Y+closeButton.H > 800-24 {
		t.Fatalf("the close button sits under the gesture bar: %+v", closeButton)
	}
}
