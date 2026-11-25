// Desktop owns the native host: the main window, tray panel, settings and
// menus all draw with MyGo's native UI. The timer and archive stay in Aquarium.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/ui"
)

const feedbackURL = "https://github.com/yam8511/little-reef/issues/new"

var errArchiveCanceled = errors.New("已取消")

type Desktop struct {
	a                   *Aquarium
	win                 *mygo.Window
	tray                *mygo.Tray
	quitting            atomic.Bool
	stopSwim            func()
	menu                *mygo.Menu
	appMenu             *mygo.Menu
	compact             atomic.Bool
	fullBounds          mygo.Rectangle
	settings            *nativeSettings
	shortcuts           *shortcutController
	pausedMenuShortcuts []pausedMenuShortcut

	// The main window's presentation state and the ambience it plays.
	ui    *mainUI
	sound *soundscape

	// Mini native panel anchored to the tray icon.
	flyout *mygo.Window
	// In-flight command tracking for the tray panel and menus.
	timerReq          requestState
	uiReq             requestState
	confirmEnd        bool
	confirmEndSession int64

	// Without a window (headless tests, the MCP harness) updates wait here
	// for the next frame of a view to run them on its thread.
	pendingMu sync.Mutex
	pending   []func()
}

func newDesktop(a *Aquarium) *Desktop {
	d := &Desktop{a: a, sound: newSoundscape(soundFiles())}
	d.ui = newMainUI(d)
	d.sound.onStatus = func(message string, isError bool) {
		d.update(func() {
			d.ui.soundStatus = message
			if isError {
				d.ui.showToast(message)
			}
		})
	}
	d.sound.onChanged = d.invalidateFlyout
	return d
}

// update runs fn on the main thread before the next frame of every window.
// Without a window it runs before the next frame a view builds (drain).
func (d *Desktop) update(fn func()) {
	if d.win == nil {
		d.pendingMu.Lock()
		d.pending = append(d.pending, fn)
		d.pendingMu.Unlock()
		return
	}
	d.win.Update(func() {
		if d.quitting.Load() || d.win.IsDestroyed() {
			return
		}
		fn()
		d.invalidateOthers()
	})
}

// drain runs the updates posted while there was no window, on the view's
// thread; views call it first.
func (d *Desktop) drain() {
	d.pendingMu.Lock()
	queued := d.pending
	d.pending = nil
	d.pendingMu.Unlock()
	for _, fn := range queued {
		fn()
	}
}

// invalidateOthers redraws the windows that show the same state as the main one.
func (d *Desktop) invalidateOthers() {
	if d.flyout != nil && !d.flyout.IsDestroyed() {
		d.flyout.Invalidate()
	}
	if s := d.settings; s != nil && !s.win.IsDestroyed() {
		s.win.Invalidate()
	}
}

func (d *Desktop) spaceShortcutEnabled() bool { return d.shortcutSettings().SpaceEnabled }

// exportArchive asks where to save the backup, then writes it. An empty
// path means the user canceled; callers must stay silent then.
func (d *Desktop) exportArchive(ctx context.Context, parent *mygo.Window) (string, error) {
	text, err := d.a.ExportArchive()
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      parent,
		Title:       "匯出存檔",
		ButtonLabel: "匯出",
		DefaultPath: filepath.Join(home, "little-reef-archive.json"),
		Filters:     []mygo.FileFilter{{Name: "小海底存檔", Extensions: []string{"json"}}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// importArchive asks which backup to restore, then validates it. A canceled
// pick means "已取消" and stays quiet; failed validation preserves the archive.
func (d *Desktop) importArchive(ctx context.Context, parent *mygo.Window) (Snapshot, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:      parent,
		Title:       "匯入存檔",
		ButtonLabel: "匯入",
		Filters:     []mygo.FileFilter{{Name: "小海底存檔", Extensions: []string{"json"}}},
	})
	if err != nil {
		return Snapshot{}, err
	}
	if len(paths) == 0 {
		return Snapshot{}, errArchiveCanceled
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return Snapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	return d.a.importArchive(ctx, string(data))
}

// Full window minimums; a restored bounds below either dimension means the
// previous run quit in compact mode (compact writes a 400x350 rect via
// StateKey, which mygo restores verbatim without clamping to MinWidth).
const (
	fullMinWidth  = 760
	fullMinHeight = 680
	fullWidth     = 1100
	fullHeight    = 800
)

// needsStartupFullReset reports whether a restored window rect (after
// mygo's StateKey restore) is too small to be a genuine full-size window,
// meaning the previous run quit while compact. Pure function, unit-tested.
func needsStartupFullReset(w, h int) bool {
	return w < fullMinWidth || h < fullMinHeight
}

// applyCompactOn applies the compact/full geometry to win.
func (d *Desktop) applyCompactOn(win *mygo.Window, compact bool) {
	area := mygo.Screen.DisplayMatching(win.Bounds()).WorkArea
	wasCompact := d.compact.Swap(compact)
	if compact && !wasCompact {
		d.fullBounds = win.Bounds()
	}
	win.SetAlwaysOnTop(compact)
	if compact {
		win.SetMinimumSize(360, 260)
		win.SetSize(400, 350)
		win.SetPosition(area.X+max(0, area.Width-420), area.Y+max(0, area.Height-370))
	} else {
		win.SetMinimumSize(fullMinWidth, fullMinHeight)
		if wasCompact && d.fullBounds.Width >= fullMinWidth && d.fullBounds.Height >= fullMinHeight {
			bounds := restoredFullBounds(d.fullBounds, area)
			win.SetBounds(bounds)
			d.refreshNativeMenus()
			win.Invalidate()
			return
		}
		win.SetSize(fullWidth, fullHeight)
		width, height := win.Size()
		win.SetPosition(area.X+max(0, (area.Width-width)/2), area.Y+max(0, (area.Height-height)/2))
	}
	d.refreshNativeMenus()
	win.Invalidate()
}

func restoredFullBounds(saved, area mygo.Rectangle) mygo.Rectangle {
	saved.Width = min(max(saved.Width, fullMinWidth), area.Width)
	saved.Height = min(max(saved.Height, fullMinHeight), area.Height)
	saved.X = max(area.X, min(saved.X, area.X+area.Width-saved.Width))
	saved.Y = max(area.Y, min(saved.Y, area.Y+area.Height-saved.Height))
	return saved
}

func (d *Desktop) openAquarium() {
	if d.compact.Load() {
		d.requestCompact(false)
	} else {
		d.show()
	}
}

// CheckForUpdates opens the update window, which says when the app is up to
// date or offers the new version from the repository's latest release.
func (*Desktop) CheckForUpdates() { updater.CheckForUpdates() }

func (d *Desktop) show() {
	if d.win != nil && !d.win.IsDestroyed() && !d.quitting.Load() {
		d.win.Show()
		d.win.Focus()
	}
}

// changed may run on a bound-call or timer goroutine; the main window takes
// the snapshot on the main thread.
func (d *Desktop) changed(s Snapshot) {
	if d.quitting.Load() {
		return
	}
	d.update(func() {
		d.ui.apply(s)
		d.refreshNativeMenus()
	})
}

func (d *Desktop) completed(s Snapshot) {
	if d.quitting.Load() {
		return
	}
	go func() {
		n := mygo.NewNotification(mygo.NotificationOptions{
			Title:  "小海底 · " + map[bool]string{true: "專注完成", false: "休息結束"}[s.Session.Kind == "focus"],
			Body:   s.Notice,
			Sender: &mygo.NotificationSender{ID: "reef", Name: "小海底", Avatar: markPNG(128, markInk)},
		})
		n.OnClick(d.show)
		if err := n.Show(); err != nil && !errors.Is(err, mygo.ErrNotificationsDenied) {
			log.Print("系統通知未顯示：", err)
		}
	}()
}

// handleWindowClose is the pure decision logic for the window close event.
// Returns (allow, interceptHide). When quitting is true the close proceeds
// (real quit); otherwise it is intercepted as a hide, recoverable via tray,
// Dock, or global shortcut regardless of tray presence.
func handleWindowClose(quitting bool) (allow bool, interceptHide bool) {
	if quitting {
		return true, false
	}
	return false, true
}

func (d *Desktop) run() error {
	mygo.Use(updater.New(updater.Options{Interval: 4 * time.Hour}))
	mygo.App.WhenReady(func() {
		d.appMenu = mygo.NewMenu(d.applicationMenu())
		mygo.App.SetMenu(d.appMenu)
		d.ui.apply(d.a.Get())
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "小海底 · 桌面水族館", Width: fullWidth, Height: fullHeight, MinWidth: fullMinWidth, MinHeight: fullMinHeight,
			TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: 58,
			BackgroundColor: "#0c1820", StateKey: "aquarium-main",
			Content: ui.View(d.ui.view),
		})
		// mygo restores the StateKey rect verbatim without clamping to
		// MinWidth/MinHeight; if the previous run quit in compact (400x350),
		// snap back to the full geometry. Genuine full-size rects are kept.
		if w, h := d.win.Size(); needsStartupFullReset(w, h) {
			d.applyCompactOn(d.win, false)
		}
		d.win.OnClose(func(e *mygo.CloseEvent) {
			allow, intercept := handleWindowClose(d.quitting.Load())
			if allow {
				return
			}
			if intercept {
				e.PreventDefault()
				d.win.Hide()
			}
		})
		mygo.App.OnActivate(func(bool) { d.show() })
		d.createTray()
		d.a.onChange = d.changed
		d.a.onDone = d.completed
		if err := d.shortcuts.start(); err != nil {
			log.Print("全域快捷鍵不可用：", err)
		}
		go func() { _ = mygo.App.Dock.SetIcon(markBadgePNG(512)) }()
		mygo.Power.OnSuspend(func() { go func() { _, _ = d.a.Pause() }() })
		go d.a.run()
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { d.quitting.Store(true) })
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if d.shortcuts != nil {
			d.shortcuts.stop()
		}
		if d.stopSwim != nil {
			d.stopSwim()
		}
		d.sound.dispose()
		if err := d.a.shutdown(); err != nil {
			log.Print("存檔失敗：", err)
		}
		if d.tray != nil {
			d.tray.Destroy()
		}
	})
	return mygo.App.Run()
}

// invalidateMain redraws the main window, as after the tray panel changed
// state it shows too.
func (d *Desktop) invalidateMain() {
	if d.win != nil && !d.win.IsDestroyed() {
		d.win.Invalidate()
	}
}

// openAquarium loads the archive and the theme choice from dir.
func openAquarium(dir string) (*Aquarium, error) {
	a, err := newAquarium(filepath.Join(dir, "aquarium.json"))
	if err != nil {
		return nil, err
	}
	themeStore, err := newThemeStore(dir)
	if err != nil {
		return nil, err
	}
	a.themes = themeStore
	cfg := themeStore.loadConfig()
	a.theme, a.themeAuto = cfg.Theme, cfg.Auto
	a.themeShuffle, a.nextThemeShuffleMS = cfg.Shuffle, cfg.NextShuffleMS
	return a, nil
}
