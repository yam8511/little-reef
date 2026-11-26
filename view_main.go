package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
	"little-reef/internal/reef"
)

var soundOptions = []struct{ id, label string }{
	{"waves", "海浪"}, {"rain", "雨聲"}, {"thunder", "雷雨"}, {"stream", "溪流"}, {"bubbles", "氣泡"},
	{"boil", "煮沸"}, {"deepsea", "深海"}, {"wind", "風聲"}, {"fire", "篝火"}, {"forest", "森林"},
	{"summer", "夏日"}, {"night", "夏夜"}, {"desert", "沙漠"}, {"morning", "晨間"}, {"clock", "時鐘"},
	{"knock", "木魚"}, {"cafe", "咖啡館"}, {"musicbox", "卡農"}, {"traffic", "車流"}, {"pink", "粉噪"},
}

var brandMark = func() *ui.Bitmap {
	b, err := ui.DecodeBitmap(markPNG(128, markInk))
	if err != nil {
		panic(err)
	}
	return b
}()

func formatRecord(seconds int64) string {
	return fmt.Sprintf("%d分%d秒", seconds/60, seconds%60)
}

// localDate is the page's toLocaleDateString(…, {month: "numeric", day: "numeric"}): M/D.
func localDate(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return ""
	}
	t = t.Local()
	return fmt.Sprintf("%d/%d", int(t.Month()), t.Day())
}

// view is the main window: the full aquarium page, or the compact desktop widget.
func (m *mainUI) view(c *ui.Context) {
	m.d.drain()
	p := m.palette()
	c.SetTheme(m.themeUI(p))
	c.Root().Background(p.Bg)
	if m.snap == nil {
		return
	}
	s := *m.snap
	tv := deriveTimer(s, m.kind)
	compact := m.d.compact.Load()

	// Space starts or pauses unless typing, a focused control or a dialog takes it.
	if m.d.spaceShortcutEnabled() && !c.FocusTakesText() && c.Shortcut(0, ui.KeySpace) && !c.ShortcutRepeated() {
		m.primary()
	}

	if compact {
		m.compactView(c, p, s, tv)
	} else {
		m.fullView(c, p, s, tv)
	}
	m.endDialog(c, p)
	m.settingsPage(c, p)
	m.backgroundDialog(c, p)
	m.toastView(c, p)
}

func (m *mainUI) titleBar(c *ui.Context, p *palette, compact bool, narrow bool) {
	bar := c.TitleBar()
	pad := float32(24)
	h := max(58, bar.Height)
	if compact {
		pad, h = 16, max(40, bar.Height)
	}
	ui.Row(c).Height(h).Padding(0, pad+bar.Right, 0, pad+bar.Left).Justify(ui.SpaceBetween).
		BorderWidth(0, 0, 1, 0).BorderColor(p.Line).DragWindow().Shrink(0).Children(func() {
		ui.Row(c).Gap(10).Children(func() {
			size := float32(26)
			if compact {
				size = 21
			}
			ui.Image(c, brandMark).Size(size, size).Shrink(0)
			fs := float32(14)
			if compact {
				fs = 11
			}
			ui.Text(c, "小海底").FontSize(fs).FontWeight(600).LetterSpacing(1).SingleLine()
			if !compact && !narrow {
				ui.Text(c, "專注，讓一片海慢慢長大").FontSize(11).LetterSpacing(0.2).TextColor(p.Muted).MarginStart(8).SingleLine()
			}
		})
		ui.Row(c).Gap(8).Children(func() {
			if compact {
				return
			}
			if cssButton(c, false, func(b ui.Element, h float32) {
				b.Size(30, 30).Radius(7).Label("切換桌面掛件").Tooltip("切換桌面掛件")
				look{bg: p.HoverBg.Alpha(0), fg: p.Muted, hbg: p.HoverBg, hfg: p.Muted, filter: 1.12}.apply(b, h)
				b.Children(func() { ui.Icon(c, lucide["compact"]).Size(16, 16) })
			}).Clicked() {
				m.d.requestCompact(true)
			}
		})
	})
}

func (m *mainUI) fullView(c *ui.Context, p *palette, s Snapshot, tv timerView) {
	winW, winH := c.Size()
	mid := winW <= 950
	narrow := winW <= 700
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MinHeight(winH).Draw(func(pt *ui.Painter, r ui.Rect) {
			ellipseGradient(pt, r, 0.16, 0.2, []ui.GradientStop{{Color: p.BgGlow, At: 0}, {Color: p.BgGlow.Alpha(0), At: 0.48}})
		}).Children(func() {
			m.titleBar(c, p, false, narrow)
			pad := []float32{26, 36, 20, 36}
			if mid {
				pad = []float32{24, 24, 24, 24}
			}
			if narrow {
				pad = []float32{18, 18, 18, 18}
			}
			ui.Column(c).FillWidth().MaxWidth(1180).Margin(0, ui.Auto).Padding(pad...).Children(func() {
				m.intro(c, p, narrow)
				grid := ui.Grid(c).Gap(20)
				if narrow {
					grid.Columns(1)
				} else {
					grid.ColumnTracks(ui.Fr(1.55), ui.Fr(1).Min(270))
				}
				grid.Children(func() {
					m.reefPanel(c, p, s, tv, false)
					m.timerPanel(c, p, s, tv, narrow)
				})
				m.soundPanel(c, p, s, mid)
				if s.Notice != "" {
					ui.Text(c, s.Notice).Role(ui.RoleStatus).FontSize(11).LineHeight(1.7).TextColor(p.Text).
						Padding(13, 18).Radius(11).Margin(16, 0, 0, 0).Background(p.ActiveBg).Border(1, p.ActiveBorder)
				}
				bottom := ui.Grid(c).Gap(20).Margin(20, 0, 0, 0)
				if narrow {
					bottom.Columns(1)
				} else {
					bottom.Columns(2)
				}
				bottom.Children(func() {
					m.statsPanel(c, p, s)
					m.visitorsPanel(c, p, s)
				})
				m.reviewSection(c, p, s)
				m.journalSection(c, p, s)
				ui.Row(c).Justify(ui.SpaceBetween).Gap(14).Margin(18, 0, 0, 0).Children(func() {
					ui.Text(c, "讓專注留下一片海，也記得抬頭看看窗外。").FontSize(10).TextColor(p.Faint)
					if cssButton(c, false, func(b ui.Element, h float32) {
						b.Padding(4)
						look{bg: p.Bg.Alpha(0), fg: p.Muted, hbg: p.Bg.Alpha(0), hfg: p.Accent, filter: 1.12}.apply(b, h)
						b.Children(func() { ui.Text(c, "計時設定").FontSize(10).SingleLine() })
					}).Clicked() {
						m.d.OpenSettings()
					}
				})
			})
		})
	})
}

func (m *mainUI) intro(c *ui.Context, p *palette, narrow bool) {
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.End).Margin(0, 0, 22, 0).Children(func() {
		ui.Column(c).Children(func() {
			ui.Text(c, "屬於你的安靜角落").FontSize(11).LetterSpacing(2).TextColor(p.Muted).Margin(0, 0, 10, 0)
			size := float32(27)
			if narrow {
				size = 23
			}
			ui.RichText(c, ui.Span{Text: "你在專注，"}, ui.Span{Text: "海底在生長。", Color: p.Accent}).
				FontSize(size).FontWeight(500).LetterSpacing(-0.7).Role(ui.RoleHeading)
		})
		ui.Row(c).Gap(7).Padding(0, 0, 5, 0).Children(func() {
			dot(c, p.Accent)
			ui.Text(c, "安靜陪伴中").FontSize(11).TextColor(p.Muted).SingleLine()
		})
	})
}

// reefStage is the animated tank: water, two light rays, the reef and its caption.
func (m *mainUI) reefStage(c *ui.Context, p *palette, tv timerView, compact bool) ui.Element {
	stage := ui.Box(c).Grow(1).Clip().Label("魚兒在水草之間緩慢游動")
	if !compact {
		stage.MinHeight(275)
	} else {
		stage.MinHeight(0)
	}
	stage.Draw(func(pt *ui.Painter, r ui.Rect) {
		ellipseGradient(pt, r, 0.5, 0, []ui.GradientStop{{Color: p.WaterLight, At: 0}, {Color: p.BgGradient, At: 0.55}, {Color: p.Bg, At: 1}})
	})
	still := c.Preferences().ReduceMotion
	stage.Children(func() {
		if !still {
			ray := ui.RGBA(0xa6, 0xdb, 0xce, float32(0x18)/255)
			for _, r := range []struct{ left, width, opacity float32 }{{12, 70, 1}, {53, 110, 0.5}} {
				ui.Box(c).Absolute().Top(-70).LeftPercent(r.left).Size(r.width, 420).
					LinearGradient(ui.LinearGradient{From: ray, To: ray.Alpha(0), Angle: 180}).
					Rotate(-27).Blur(9).Opacity(r.opacity).PassThrough()
			}
		}
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Draw(func(pt *ui.Painter, r ui.Rect) {
			m.reef.Draw(pt, r, pt.Now(), still)
			if !still {
				pt.AnimationFrame()
			}
		})
		bottom, size := float32(20), float32(11)
		if compact {
			bottom, size = 10, 10
		}
		ui.Row(c).Absolute().Left(20).Right(20).Bottom(bottom).Gap(8).PassThrough().Children(func() {
			dot(c, p.Accent)
			ui.Text(c, tv.Caption).FontSize(size).TextColor(p.Muted)
		})
	})
	return stage
}

func (m *mainUI) reefPanel(c *ui.Context, p *palette, s Snapshot, tv timerView, compact bool) {
	panel := ui.Column(c).Clip().Background(p.Panel).Role(ui.RoleRegion).Label("你的水族館")
	if compact {
		panel.Grow(1)
	} else {
		panel.Radius(p.Radius).Border(1, p.Line)
	}
	panel.Children(func() {
		if !compact {
			ui.Row(c).Padding(18, 22).Gap(10).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, "第一片小海底").FontSize(11).LetterSpacing(0.4).TextColor(p.Text).SingleLine()
				ui.Text(c, tv.GrowthLabel).FontSize(10).TextColor(p.Accent).SingleLine()
			})
		}
		m.reefStage(c, p, tv, compact)
		if !compact {
			ui.Row(c).Padding(18, 22).Gap(12).Justify(ui.SpaceBetween).Background(p.PanelDeep).
				BorderWidth(1, 0, 0, 0).BorderColor(p.Line).Children(func() {
				ui.Column(c).Children(func() {
					ui.Text(c, "下一次成長").FontSize(10).TextColor(p.Muted)
					ui.Text(c, tv.NextGrowth).FontSize(11).TextColor(p.Accent).Margin(7, 0, 0, 0)
				})
				if subtleButton(c, p, tv.FeedLabel, tv.FeedDisabled || m.busy).Clicked() {
					m.feed()
				}
			})
		}
	})
}

// timerRing is the 186 DIP SVG ring: a track and the remaining time drawn
// clockwise from the top, easing over .6 s, its color over .4 s.
func (m *mainUI) timerRing(c *ui.Context, p *palette, tv timerView, size float32) {
	ring := ui.Box(c).Size(size, size).Margin(6, ui.Auto, 14, ui.Auto)
	progress := ring.AnimateWith("progress", tv.Progress, 600*time.Millisecond, ui.EaseInOut)
	paused := ring.AnimateWith("paused", b2f(tv.Paused), 400*time.Millisecond, ui.EaseInOut)
	color := lerpColor(p.TimerActive, p.TimerPause, paused)
	ring.Draw(func(pt *ui.Painter, r ui.Rect) {
		scale := r.W / 200
		cx, cy, radius, width := r.X+r.W/2, r.Y+r.H/2, 89*scale, 2*scale
		pt.StrokePath((&ui.Path{}).Circle(cx, cy, radius), width, p.RingTrack)
		left := 1 - progress
		if left <= 0 {
			return
		}
		arc := &ui.Path{}
		const steps = 180
		n := max(1, int(math.Ceil(float64(left*steps))))
		for i := 0; i <= n; i++ {
			a := -math.Pi/2 + 2*math.Pi*float64(left)*float64(i)/float64(n)
			x, y := cx+radius*float32(math.Cos(a)), cy+radius*float32(math.Sin(a))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		pt.StrokePath(arc, width, color)
	}).Children(func() {
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(12).Children(func() {
			ui.Text(c, tv.Time).Font(p.digitsFont()).FontSize(43).FontWeight(300).LetterSpacing(-1).
				FontFeatures("tnum").Role(ui.RoleTimer).Label("剩餘時間").SingleLine()
			ui.Text(c, tv.Status).FontSize(11).TextColor(p.Muted).SingleLine()
		})
	})
}

func (m *mainUI) timerPanel(c *ui.Context, p *palette, s Snapshot, tv timerView, narrow bool) {
	active := sessionActive(s)
	ui.Column(c).Background(p.Panel).Border(1, p.Line).Radius(p.Radius).Padding(20, 24).
		Role(ui.RoleRegion).Label("專注計時").Children(func() {
		ui.Row(c).Justify(ui.Center).Gap(4).Margin(0, 0, 4, 0).Role(ui.RoleTabList).Label("計時模式").Children(func() {
			for _, k := range []struct{ kind, label string }{{"focus", "專注"}, {"break", "休息"}} {
				selected := m.kind == k.kind
				disabled := active || m.busy
				tab := ui.ButtonBase(c).Disabled(disabled).DimDisabled(false).Role(ui.RoleTab).Selected(selected)
				hover := tab.Animate("hover", b2f(tab.Hovered() && !disabled), cssTransition)
				bg, fg := p.HoverBg.Alpha(0), p.Muted
				if selected {
					bg, fg = p.ActiveBg, p.Text
				} else {
					bg, fg = lerpColor(bg, p.HoverBg, hover), lerpColor(fg, p.Text, hover)
				}
				if disabled {
					tab.Opacity(0.6)
				} else {
					tab.Cursor(ui.CursorPointer)
				}
				tab.Padding(7, 22).Radius(8).Background(bg).TextColor(fg).Children(func() {
					ui.Text(c, k.label).FontSize(12).SingleLine()
				})
				if tab.Clicked() {
					m.kind = k.kind
				}
			}
		})
		size := float32(186)
		if narrow {
			size = 180
		}
		m.timerRing(c, p, tv, size)
		ui.Row(c).Justify(ui.SpaceBetween).Margin(0, 0, 8, 0).Children(func() {
			ui.Text(c, "這一輪，想做什麼？").FontSize(11).TextColor(p.Muted)
			ui.Text(c, "可選").FontSize(10).TextColor(p.Faint)
		})
		disabled := tv.TaskDisabled || m.busy
		field := ui.Row(c).Radius(11).Padding(11, 12).Background(p.FieldBg).Cursor(ui.CursorText)
		if disabled {
			field.Opacity(0.45)
		}
		field.Children(func() {
			in := ui.TextInputBase(c, &m.task).Grow(1).FontSize(12).TextColor(p.Text).Placeholder(tv.Placeholder).
				PlaceholderColor(p.Faint).MaxLength(80).Disabled(disabled).DimDisabled(false).Label("這一輪，想做什麼？").FocusRing(false)
			focus := field.Animate("focus", b2f(in.Focused()), cssTransition)
			field.Border(1, lerpColor(p.Line, p.Accent, focus))
			if in.Submitted() && !disabled {
				m.primary()
			}
		})
		ui.Row(c).Gap(8).Margin(14, 0, 0, 0).Children(func() {
			if primaryButton(c, p, tv.Primary, tv.PrimaryIcon, m.busy).Grow(1).Clicked() {
				m.primary()
			}
			if !tv.EndHidden {
				if subtleButton(c, p, "結束本輪", m.busy).Clicked() {
					m.endOpen = true
				}
			}
		})
		ui.Text(c, tv.Footnote).FontSize(10).LineHeight(1.6).TextColor(p.Faint).TextAlign(ui.Center).Margin(12, 0, 0, 0)
	})
}

func (m *mainUI) soundPanel(c *ui.Context, p *palette, s Snapshot, mid bool) {
	heading := func() {
		ui.Row(c).Gap(10).MinWidth(170).Children(func() {
			ui.Icon(c, lucide["waves"]).Size(26, 26).TextColor(p.Accent)
			ui.Column(c).Children(func() {
				ui.Text(c, "海底的聲音").FontSize(13).FontWeight(600).Role(ui.RoleHeading)
				ui.Text(c, m.soundStatus).FontSize(10).TextColor(p.Muted).Margin(5, 0, 0, 0).Role(ui.RoleStatus)
			})
		})
	}
	options := func() ui.Element {
		return ui.Row(c).Wrap().Gap(6).Role(ui.RoleGroup).Label("選擇背景音").Children(func() {
			for _, o := range soundOptions {
				selected := s.Preferences.Sound == o.id
				b := cssButton(c, m.busy, func(b ui.Element, h float32) {
					b.Gap(7).Radius(9).Padding(8, 12).Selected(selected)
					l := look{bg: p.HoverBg.Alpha(0), fg: p.Muted, border: p.Line, hbg: p.HoverBg, hfg: p.Muted, hborder: p.Faint, filter: 1.12}
					if selected {
						l = look{bg: p.ActiveBg, fg: p.Text, border: p.ActiveBorder, hbg: p.HoverBg, hfg: p.Text, hborder: p.Faint, filter: 1.12}
					}
					l.apply(b, h)
					b.Children(func() {
						// The icon sat in a span of font-size 15: one line of it tall.
						ui.Box(c).Height(cssLine(15)).Justify(ui.Center).Children(func() { ui.Icon(c, lucide[o.id]).Size(15, 15) })
						ui.Text(c, o.label).FontSize(11).SingleLine()
					})
				})
				if b.Clicked() {
					m.pickSound(o.id)
				}
			}
		})
	}
	volume := func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "音量").FontSize(10).TextColor(p.Muted).SingleLine()
			v := m.volumeShown() * 100
			sl := ui.SliderBase(c, &v, 0, 100).Step(1).Size(88, 18).Label("音量").FocusRing(false).Cursor(ui.CursorPointer)
			if sl.Changed() {
				// The ambience follows the drag at once; the choice is saved
				// as the pointer or finger lets go, or at each arrow key.
				m.volumeDrag, m.volumeActive = math.Round(v)/100, true
				m.d.sound.preview(m.volumeDrag)
			}
			if m.volumeActive && !sl.Pressed() && !m.busy {
				m.setVolume(m.volumeDrag)
			}
			focused := sl.Focused() && sl.FocusVisible()
			frac := float32(m.volumeShown())
			sl.Draw(func(pt *ui.Painter, r ui.Rect) {
				track := ui.Rect{X: r.X, Y: r.Y + r.H/2 - 2, W: r.W, H: 4}
				pt.Fill(track, p.RingTrack, 2)
				pt.Fill(ui.Rect{X: track.X, Y: track.Y, W: track.W * frac, H: 4}, p.Accent, 2)
				// Radix laid its thumb out as a 23×10 pill and kept it inside
				// the track by half its width, scaled from the middle out.
				const tw, th = 23, 10
				x := r.X + r.W*frac + tw/2*(1-2*frac)
				thumb := ui.Rect{X: x - tw/2, Y: r.Y + r.H/2 - th/2, W: tw, H: th}
				pt.Shadow(thumb, th/2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0x55/255.0))
				cy := thumb.Y + th/2
				if focused { // outline: 2px solid active-border, 2px out
					pt.FillPath(ellipse(x, cy, tw/2+4, th/2+4), p.ActiveBorder)
					pt.FillPath(ellipse(x, cy, tw/2+2, th/2+2), p.Panel)
				}
				pt.FillPath(ellipse(x, cy, tw/2, th/2), p.Panel)
				pt.FillPath(ellipse(x, cy, tw/2-2, th/2-2), p.Accent)
			})
			ui.Text(c, fmt.Sprintf("%d%%", int(math.Round(m.volumeShown()*100)))).FontSize(10).TextColor(p.Muted).
				MinWidth(27).FontFeatures("tnum").SingleLine()
		})
	}
	toggle := func() {
		on := m.soundPlaying
		label := "開啟聲音"
		if on {
			label = "關閉聲音"
		}
		if on && m.soundStatus == soundStatusLoading {
			label = "載入中…"
		}
		b := cssButton(c, m.busy, func(b ui.Element, h float32) {
			b.Radius(9).Padding(10, 13).Selected(on)
			l := look{bg: p.ActiveBg, fg: p.Text, border: p.ActiveBorder, hbg: p.ActiveBg, hfg: p.Text, hborder: p.ActiveBorder, filter: 1.05}
			if on {
				l = look{bg: p.ActiveBg.Alpha(0), fg: p.Muted, border: p.Line, hbg: p.ActiveBg.Alpha(0), hfg: p.Muted, hborder: p.Line, filter: 1.05}
			}
			l.apply(b, h)
			b.Children(func() { ui.Text(c, label).FontSize(11).SingleLine() })
		})
		if b.Clicked() {
			m.toggleSound()
		}
	}
	panel := ui.Column(c).Margin(20, 0, 0, 0).Background(p.Panel).Border(1, p.Line).Radius(16).Padding(18, 22).
		Role(ui.RoleRegion).Label("背景音")
	panel.Children(func() {
		if !mid {
			ui.Row(c).Gap(18).Children(func() {
				heading()
				options().Grow(1).Basis(0)
				volume()
				toggle()
			})
			return
		}
		ui.Column(c).Gap(14).Children(func() {
			heading()
			options()
			ui.Row(c).Gap(14).Justify(ui.End).Children(func() {
				volume()
				toggle()
			})
		})
	})
}

func (m *mainUI) statsPanel(c *ui.Context, p *palette, s Snapshot) {
	ui.Column(c).Padding(20, 22).Background(p.Panel).Border(1, p.Line).Radius(16).Role(ui.RoleRegion).Label("專注積累").Children(func() {
		ui.Text(c, "今天的小積累").FontSize(12).FontWeight(600).Role(ui.RoleHeading)
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(15).Margin(17, 0, 0, 0).Children(func() {
			for _, st := range []struct {
				n          string
				unit, note string
			}{
				{fmt.Sprint(s.TodaySecs / 60), " 分鐘", "認真投入的時間"},
				{fmt.Sprint(s.TodayRounds), " 輪", "海底的成長"},
				{fmt.Sprint(len(s.Visitors)), " 位", "認識的新朋友"},
			} {
				ui.Column(c).Children(func() {
					ui.RichText(c, ui.Span{Text: st.n, Size: 25, Color: p.Accent}, ui.Span{Text: st.unit, Size: 11, Color: p.Muted}).
						FontFeatures("tnum").SingleLine()
					ui.Text(c, st.note).FontSize(10).TextColor(p.Muted).Margin(6, 0, 0, 0)
				})
			}
		})
		foot := "你的海底，正在等待第一個故事。"
		if s.TotalSecs > 0 {
			foot = fmt.Sprintf("已經一起度過 %d 分鐘，完成 %d 輪專注。", s.TotalSecs/60, s.Completed)
		}
		ui.Text(c, foot).FontSize(10).TextColor(p.Faint).Margin(15, 0, 0, 0)
	})
}

func (m *mainUI) visitorsPanel(c *ui.Context, p *palette, s Snapshot) {
	ui.Column(c).Padding(20, 22).Background(p.Panel).Border(1, p.Line).Radius(16).Role(ui.RoleRegion).Label("訪客圖鑑").Children(func() {
		ui.Row(c).Justify(ui.SpaceBetween).Gap(10).Children(func() {
			ui.Text(c, "來過的朋友").FontSize(12).FontWeight(600).Role(ui.RoleHeading)
			ui.Text(c, "偶遇，也值得記住").FontSize(10).TextColor(p.Faint).SingleLine()
		})
		ui.Row(c).Wrap().Gap(10).Margin(15, 0, 0, 0).AlignItems(ui.Start).Children(func() {
			if len(s.Visitors) == 0 {
				ui.Row(c).Gap(12).Padding(8, 0).Children(func() {
					// filter: saturate(.6); opacity: .5
					ui.Text(c, "🐠").FontSize(32).AccessHidden(true).Effect(muted.Effect(shader.Options{}))
					ui.Text(c, "第一輪專注後，會有一位朋友來拜訪。").FontSize(11).TextColor(p.Muted).MaxWidth(210).LineHeight(1.8)
				})
				return
			}
			for i, v := range s.Visitors {
				card := ui.Row(c.Key(fmt.Sprintf("%s-%d", v.Species, i))).Gap(9).Padding(9, 11).Radius(11).Border(1, p.Line)
				hover := card.Animate("hover", b2f(card.Hovered()), cssTransition)
				card.Background(lerpColor(p.PanelRaised, p.HoverBg, hover))
				pal := p.reef()
				species := v.Species
				card.Children(func() {
					ui.Box(c).Size(46, 32).Shrink(0).AccessHidden(true).Draw(func(pt *ui.Painter, r ui.Rect) {
						reef.DrawPortrait(pt, r, species, pal)
					})
					ui.Column(c).Children(func() {
						ui.Text(c, v.Name).FontSize(11).TextColor(p.Text).SingleLine()
						ui.Text(c, fmt.Sprintf("%s · 來過 %d 次", localDate(v.Date), v.Visits)).FontSize(9).TextColor(p.Muted).Margin(4, 0, 0, 0).SingleLine()
					})
				})
			}
		})
	})
}

// collapsible is .review / .journal: a trigger row and a panel that reveals
// with the page's .3 s fade and 4 DIP rise.
func (m *mainUI) collapsible(c *ui.Context, p *palette, open *bool, title, aside string, body func()) {
	ui.Column(c).Margin(18, 0, 0, 0).Padding(14, 20).Radius(13).Border(1, p.Line).Background(p.PanelDeep).Children(func() {
		trigger := ui.ButtonBase(c).Gap(10).Padding(2, 0).Justify(ui.Start).Cursor(ui.CursorPointer).Expanded(*open)
		hover := trigger.Animate("hover", b2f(trigger.Hovered()), cssTransition)
		trigger.TextColor(lerpColor(p.Muted, p.Text, hover)).Children(func() {
			ui.Text(c, title).FontSize(11).SingleLine()
			if aside != "" {
				ui.Text(c, aside).FontSize(10).TextColor(p.Faint).MarginStart(ui.Auto).SingleLine()
			}
			chevron(c, p.Faint, *open)
		})
		if trigger.Clicked() {
			*open = !*open
		}
		if *open {
			panel := ui.Column(c.Key("panel"))
			enter(c, panel, ui.Pose{Y: -4, Fade: 1}, ui.Animation{Duration: 300 * time.Millisecond, Ease: cssEase})
			panel.Children(body)
		}
	})
}

func (m *mainUI) reviewSection(c *ui.Context, p *palette, s Snapshot) {
	m.collapsible(c, p, &m.reviewOpen, "這段時間", "", func() {
		r := reviewModel(s)
		if r.empty {
			ui.Text(c, r.emptyText).FontSize(11).TextColor(p.Muted)
		} else {
			for _, line := range r.lines {
				if line.text == "" {
					continue
				}
				color := p.Text
				if line.secondary {
					color = p.Muted
				}
				ui.Text(c, line.text).FontSize(11).LineHeight(1.9).TextColor(color)
			}
		}
		ui.Text(c, r.footnote).FontSize(10).TextColor(p.Faint).Margin(12, 0, 0, 0)
	})
}

func (m *mainUI) journalSection(c *ui.Context, p *palette, s Snapshot) {
	m.collapsible(c, p, &m.journalOpen, "專注日記", fmt.Sprintf("%d 條記錄", len(s.Records)), func() {
		ui.Column(c).Margin(14, 0, 0, 0).Children(func() {
			if len(s.Records) == 0 {
				ui.Text(c, "開始一輪專注，給今天留下一點痕跡。").FontSize(11).TextColor(p.Muted)
				return
			}
			for i, r := range s.Records[:min(30, len(s.Records))] {
				row := ui.Row(c.Key(i)).Justify(ui.SpaceBetween).Gap(14).Padding(11, 12)
				if i > 0 {
					row.BorderWidth(1, 0, 0, 0).BorderColor(p.Line)
				}
				hover := row.Animate("hover", b2f(row.Hovered()), cssTransition)
				if hover > 0 {
					row.Background(p.HoverBg.Alpha(float32(p.HoverBg.A) / 255 * hover)).Radius(8)
				}
				task := r.Task
				if task == "" {
					task = "一段安靜的專注"
				}
				state := "已記錄"
				if r.Completed {
					state = "完成"
				}
				row.Children(func() {
					ui.Text(c, task).FontSize(11).TextColor(p.Text).SingleLine().MaxWidthPercent(60)
					ui.Text(c, fmt.Sprintf("%s · %s · %s", state, formatRecord(r.Seconds), localDate(r.Date))).FontSize(10).TextColor(p.Muted).SingleLine()
				})
			}
		})
	})
}

func (m *mainUI) compactView(c *ui.Context, p *palette, s Snapshot, tv timerView) {
	ui.Column(c).Fill().Draw(func(pt *ui.Painter, r ui.Rect) {
		ellipseGradient(pt, r, 0.16, 0.2, []ui.GradientStop{{Color: p.BgGlow, At: 0}, {Color: p.BgGlow.Alpha(0), At: 0.48}})
	}).Children(func() {
		m.titleBar(c, p, true, false)
		ui.Column(c).Grow(1).Children(func() { m.reefPanel(c, p, s, tv, true) })
		ui.Row(c).Height(44).Padding(0, 16).Gap(10).Background(p.PanelDeep).BorderWidth(1, 0, 0, 0).BorderColor(p.Line).Shrink(0).Children(func() {
			ui.Text(c, tv.Time).Grow(1).Font(p.digitsFont()).FontSize(20).FontWeight(300).FontFeatures("tnum").TextColor(p.Accent).SingleLine()
			mini := func(icon, label string, opacity float32, disabled bool) bool {
				return cssButton(c, disabled, func(b ui.Element, h float32) {
					b.Size(34, 30).Radius(8).Label(label)
					look{bg: p.HoverBg, fg: p.Accent, border: p.Line, hbg: p.ActiveBg, hfg: p.Accent, hborder: p.ActiveBorder, filter: 1.12}.apply(b, h)
					if !disabled && opacity < 1 {
						b.Opacity(opacity)
					}
					b.Children(func() { ui.Icon(c, lucide[icon]).Size(16, 16) })
				}).Clicked()
			}
			if mini(tv.PrimaryIcon, tv.Primary, 1, m.busy) {
				m.primary()
			}
			soundIcon, soundLabel, op := "muted", "開啟聲音", float32(0.5)
			if m.soundPlaying {
				soundIcon, soundLabel, op = "volume", "關閉聲音", 1
			}
			if mini(soundIcon, soundLabel, op, false) {
				m.toggleSound()
			}
			if mini("expand", "展開水族館", 1, false) {
				m.d.requestCompact(false)
			}
		})
	})
}

// endDialog is the AlertDialog asking to end the round early.
func (m *mainUI) endDialog(c *ui.Context, p *palette) {
	if !m.endOpen {
		return
	}
	winW, winH := c.Size()
	ui.DialogBase(c, &m.endOpen, func(backdrop, panel ui.Element) {
		// background: #050d14b3 over backdrop-filter: blur(4px).
		backdrop.Draw(func(pt *ui.Painter, r ui.Rect) {
			frost.Draw(pt, r, shader.Options{BackdropBlur: 4, Uniforms: map[string]any{"Tint": ui.Hex("#050d14b3")}})
		})
		enter(c, backdrop, ui.Pose{Fade: 1}, ui.Animation{Duration: 280 * time.Millisecond, Ease: cssEase})
		panel.Width(min(430, winW-36)).MaxHeight(min(640, winH-48)).Padding(26).Radius(20).
			Background(p.Panel).TextColor(p.Text).Border(1, p.Line).Shadow(0, 24, 90, 0, ui.Hex("#000000aa")).
			Role(ui.RoleAlertDialog).Label("這一段時間，也算數。")
		enter(c, panel, ui.Pose{Y: 14, Scale: 0.98, Fade: 1}, ui.Animation{Duration: 280 * time.Millisecond, Ease: ui.CubicBezier(0.2, 0.9, 0.3, 1.1)})
		ui.Text(c, "這一段時間，也算數。").FontSize(18).FontWeight(500)
		ui.Text(c, "結束後會保留實際專注時間。已有的魚兒和水草不變，本輪不結算新的成長。").FontSize(12).LineHeight(1.7).TextColor(p.Muted)
		ui.Row(c).Gap(12).Children(func() {
			cancel := subtleButton(c, p, "繼續這輪", false).AutoFocus()
			if cancel.Clicked() {
				m.endOpen = false
			}
			if primaryButton(c, p, "結束並記錄", "", false).Grow(1).Clicked() {
				m.endOpen = false
				m.endSession()
			}
		})
	})
}

// toastView is the page's toast: bottom center, rising in over .3 s.
func (m *mainUI) toastView(c *ui.Context, p *palette) {
	if m.toast == "" {
		return
	}
	winW, _ := c.Size()
	ui.Row(c.Key(fmt.Sprint("toast", m.toastSeq))).Absolute().Left(0).Right(0).Bottom(25).Justify(ui.Center).PassThrough().ZIndex(50).Children(func() {
		t := ui.Box(c).Padding(13, 20).Background(p.PopBg).Border(1, p.Line).Radius(12).MaxWidth(winW*0.9).
			Shadow(0, 8, 30, 0, ui.Hex("#000000aa")).Role(ui.RoleStatus)
		enter(c, t, ui.Pose{Y: 10, Fade: 1}, ui.Animation{Duration: 300 * time.Millisecond, Ease: cssEase})
		t.Children(func() {
			ui.Text(c, m.toast).FontSize(12).TextColor(p.Text)
		})
	})
}

// reviewLine is one row of "這段時間".
type reviewLine struct {
	text      string
	secondary bool
}

type review struct {
	empty               bool
	emptyText, footnote string
	lines               []reviewLine
}

func minutesText(secs int64) string {
	minutes := secs / 60
	if minutes < 60 {
		return fmt.Sprintf("%d 分鐘", minutes)
	}
	return fmt.Sprintf("%d 小時 %d 分鐘", minutes/60, minutes%60)
}

func plantCount(completed int) int {
	if completed <= 0 {
		return 0
	}
	return min(8, 1+(completed-1)/3)
}

func reviewModel(s Snapshot) review {
	const footnote = "這些數字只儲存在你的電腦上。"
	if len(s.Records) == 0 {
		return review{empty: true, emptyText: "完成第一輪，這裡開始有故事。", footnote: footnote}
	}
	next := map[bool]string{true: "萌芽", false: ""}
	_ = next
	nextTierName := "小小綠洲"
	switch {
	case s.Completed < 3:
		nextTierName = "萌芽"
	case s.Completed < 12:
		nextTierName = "繁盛"
	}
	var last *Visitor
	for i := range s.Visitors {
		if last == nil || s.Visitors[i].Date > last.Date {
			last = &s.Visitors[i]
		}
	}
	remaining := 15 - len(s.Visitors)
	r := review{footnote: footnote}
	add := func(text string, secondary bool) { r.lines = append(r.lines, reviewLine{text, secondary}) }
	if s.WeekSecs > 0 {
		add(fmt.Sprintf("本週 %s · %d 輪", minutesText(s.WeekSecs), s.WeekRounds), false)
		add(fmt.Sprintf("本週 %d 天有你的專注", s.WeekDays), true)
	} else {
		add("", false)
		add("本週還沒有，不打緊，隨時開始。", true)
	}
	add(fmt.Sprintf("本月 %s · %d 輪", minutesText(s.MonthSecs), s.MonthRounds), false)
	add(fmt.Sprintf("已經一起度過 %s，完成 %d 輪專注。", minutesText(s.TotalSecs), s.Completed), false)
	add(fmt.Sprintf("一共 %d 天", s.TotalDays), true)
	add(fmt.Sprintf("圖鑑 %d / 15", len(s.Visitors)), false)
	if remaining <= 0 {
		add("15 位朋友都來過了", true)
	} else {
		add(fmt.Sprintf("還有 %d 位朋友沒來過，慢慢來", remaining), true)
	}
	if last != nil {
		add(fmt.Sprintf("最近到訪：%s · 來過 %d 次", last.Name, last.Visits), true)
	}
	add("當前："+tierName(s.Completed), false)
	if s.NextTier == 0 {
		add("繼續積累專注，認識更多海底朋友", true)
	} else {
		add(fmt.Sprintf("再 %d 輪就是%s", s.NextTier, nextTierName), true)
	}
	if n := plantCount(s.Completed); n > 0 {
		add(fmt.Sprintf("海底已有 %d 株植物，長滿後不再擁擠", n), true)
	}
	return r
}

var _ = strings.TrimSpace

// frost is the dialog overlay: the window behind, blurred, under a tint.
var frost = shader.MustCompile(`package main

var Tint vec4

func Fragment(dst vec4, pos vec2) vec4 {
	return mix(backdropSample(pos), vec4(Tint.rgb, 1), Tint.a)
}
`)

// muted is the CSS filter saturate(.6) with opacity .5, as the empty
// visitor book drew its fish.
var muted = shader.MustCompile(`package main

func LayerEffect(pos vec2) vec4 {
	c := layerAt(pos)
	gray := dot(c.rgb, vec3(0.2126, 0.7152, 0.0722))
	return vec4(mix(vec3(gray), c.rgb, 0.6), c.a) * 0.5
}
`)
