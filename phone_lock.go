package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"little-reef/internal/lockscreen"
)

// The round on a phone's lock screen. On Android the lockscreen module shows
// an ongoing notification that counts down by itself, with a button to pause
// or resume, and keeps the app running while the phone is locked. Desktops
// have no implementation: d.lock stays nil and nothing here runs.

// startLockScreen connects the lock screen, where the platform has one.
func (d *Desktop) startLockScreen() {
	t, err := lockscreen.NewTimer()
	if err != nil {
		return // errors.ErrUnsupported on desktops
	}
	d.useLockScreen(t)
}

func (d *Desktop) useLockScreen(t lockscreen.Timer) {
	d.lock = t
	d.lockStyle = d.loadLockStyle()
	d.lockActions = t.OnAction(func(a lockscreen.Action) {
		switch a {
		case lockscreen.ActionPause:
			d.startTimerAction(timerPause, 0)
		case lockscreen.ActionResume:
			d.startTimerAction(timerResume, 0)
		case lockscreen.ActionOpen:
			d.show()
		case lockscreen.ActionEnd:
			// The app comes to the front and asks, as its own button does.
			d.update(func() {
				if sessionActive(d.a.Get()) {
					d.ui.endOpen = true
				}
			})
			d.show()
		case lockscreen.ActionEndNow:
			// The card asked to confirm itself, pressed twice.
			d.update(func() {
				if s := d.a.Get(); sessionActive(s) {
					d.startTimerAction(timerEnd, s.Session.ID)
				}
			})
		}
	})
}

// lockStyleFile keeps the notification's style beside the archive, apart
// from it, as theme.json keeps the theme.
func (d *Desktop) lockStyleFile() string {
	return filepath.Join(filepath.Dir(d.a.path), "lockscreen.json")
}

func (d *Desktop) loadLockStyle() lockscreen.Style {
	var cfg struct {
		Style lockscreen.Style `json:"style"`
	}
	data, err := os.ReadFile(d.lockStyleFile())
	if err == nil && json.Unmarshal(data, &cfg) == nil {
		switch cfg.Style {
		case lockscreen.StyleLarge, lockscreen.StyleMedia:
			return cfg.Style
		}
	}
	return lockscreen.StyleLive
}

// setLockStyle saves the style and shows the round in it at once.
func (d *Desktop) setLockStyle(style lockscreen.Style) error {
	data, err := json.Marshal(map[string]lockscreen.Style{"style": style})
	if err != nil {
		return err
	}
	if err := os.WriteFile(d.lockStyleFile(), data, 0o600); err != nil {
		return err
	}
	d.lockStyle = style
	d.resendLockScreen()
	return nil
}

// lockRound is what the lock screen shows for a snapshot, and whether it
// shows anything: only a running or paused round does.
func lockRound(s Snapshot, now time.Time, style lockscreen.Style) (lockscreen.Round, bool) {
	if !sessionActive(s) {
		return lockscreen.Round{}, false
	}
	remaining := time.Duration(max(0, s.Session.DurationMS-s.Session.ElapsedMS)) * time.Millisecond
	r := lockscreen.Round{
		Running: s.Session.Status == "running", Remaining: remaining,
		Total:  time.Duration(s.Session.DurationMS) * time.Millisecond,
		EndsAt: now.Add(remaining), Pause: "暫停", Resume: "繼續", End: "結束這一輪", ConfirmEnd: "再按一次結束", Style: style,
	}
	switch {
	case !r.Running:
		r.Title, r.Text = "已暫停", "暫停一下，沒關係"
	case s.Session.Kind == "focus":
		r.Title, r.Text = "專注中", "小魚陪著你，慢慢來"
	default:
		r.Title, r.Text = "休息中", "離開螢幕，看看遠處"
	}
	if s.Session.Kind == "focus" && s.Session.Task != "" {
		r.Text = s.Session.Task
	}
	return r, true
}

// syncLockScreen shows the snapshot's round on the lock screen, telling the
// system only when what it shows changes: the countdown runs by itself.
func (d *Desktop) syncLockScreen(s Snapshot) {
	if d.lock == nil {
		return
	}
	// End where the snapshot, as it was taken, says the round ends.
	r, show := lockRound(s, time.UnixMilli(s.NowMS), d.lockStyle)
	if !show {
		if d.lockKey != "" {
			d.lockKey = ""
			if err := d.lock.Hide(); err != nil {
				log.Print("鎖定畫面計時未能移除：", err)
			}
		}
		return
	}
	// A running round's end moves by the clock's jitter only: round it.
	key := fmt.Sprintf("%d|%s|%s|%s|%v|%d|%d|%s", s.Session.ID, s.Session.Kind, s.Session.Status, r.Text,
		r.Running, r.EndsAt.Unix()/2, int64(r.Remaining/time.Second)*b2i(!r.Running), r.Style)
	if key == d.lockKey {
		return
	}
	first := d.lockKey == ""
	d.lockKey = key
	if err := d.lock.Show(r); err != nil {
		log.Print("鎖定畫面計時未能顯示：", err)
		return
	}
	// The first round shown asks once about what would keep the countdown
	// off the lock screen: notifications turned off, which some systems do
	// for apps installed outside a store, then the battery optimization that
	// may stop the app in the background.
	if !first || !r.Running || d.ui.backgroundAsked {
		return
	}
	switch {
	case d.lock.NotificationsBlocked():
		d.ui.backgroundAsked = true
		d.ui.notificationsPrompt = true
	case d.lock.BackgroundRestricted():
		d.ui.backgroundAsked = true
		d.ui.backgroundPrompt = true
	}
}

// resendLockScreen shows the round again as the app comes back to the
// front: notifications turned on in the system's settings meanwhile show it
// at once, rather than at the round's next change.
func (d *Desktop) resendLockScreen() {
	if d.lock == nil {
		return
	}
	d.lockKey = ""
	d.syncLockScreen(d.a.Get())
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// backgroundDialog asks for what the lock screen's countdown needs: the
// app's notifications turned on, or the app let run in the background. It
// looks as the end-of-round dialog does.
func (m *mainUI) backgroundDialog(c *ui.Context, p *palette) {
	switch {
	case m.notificationsPrompt:
		m.lockDialog(c, p, &m.notificationsPrompt, "開啟通知，才看得到倒數", notificationsAdvice, m.d.openNotificationSettings)
	case m.backgroundPrompt:
		m.lockDialog(c, p, &m.backgroundPrompt, "讓計時在鎖定時繼續", backgroundAdvice, m.d.allowBackground)
	}
}

func (m *mainUI) lockDialog(c *ui.Context, p *palette, open *bool, title, advice string, settings func()) {
	winW, winH := c.Size()
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.Hex("#050d14b3"))
		panel.Width(min(430, winW-36)).MaxHeight(min(640, winH-48)).Padding(26).Gap(10).Radius(20).
			Background(p.Panel).TextColor(p.Text).Border(1, p.Line).Shadow(0, 24, 90, 0, ui.Hex("#000000aa")).
			Role(ui.RoleAlertDialog).Label(title)
		ui.Text(c, title).FontSize(18).FontWeight(500)
		ui.Text(c, advice).FontSize(12).LineHeight(1.7).TextColor(p.Muted)
		ui.Row(c).Gap(12).Margin(6, 0, 0, 0).Children(func() {
			if subtleButton(c, p, "稍後再說", false).Clicked() {
				*open = false
			}
			if primaryButton(c, p, "前往設定", "", false).Grow(1).Clicked() {
				*open = false
				settings()
			}
		})
	})
}

const backgroundAdvice = "手機的省電設定可能在鎖定螢幕後讓小海底停止執行，倒數也會跟著停住。" +
	"建議允許小海底在背景執行（關閉電池最佳化）。小米、華為等手機也請在應用程式的「省電策略」選擇「無限制」。"

const notificationsAdvice = "小海底的通知目前是關閉的，鎖定畫面、狀態列和超級島上的倒數都會被系統擋下。" +
	"請在通知設定中開啟「允許通知」，以及「鎖定畫面通知」；小米手機也可以一併開啟「焦點通知」。"

// openNotificationSettings opens the app's notification settings.
func (d *Desktop) openNotificationSettings() {
	if d.lock == nil {
		return
	}
	if err := d.lock.OpenNotificationSettings(); err != nil {
		d.ui.showToast("無法開啟系統設定：" + err.Error())
	}
}

// allowBackground opens the system's dialog or settings for it.
func (d *Desktop) allowBackground() {
	if d.lock == nil {
		return
	}
	if err := d.lock.AllowBackground(); err != nil {
		d.ui.showToast("無法開啟系統設定：" + err.Error())
	}
}
