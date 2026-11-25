package main

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

func (d *Desktop) panelView(c *ui.Context) {
	d.drain()
	snap := d.a.Get()
	requested, _ := d.ui.audioState()
	c.SetTheme(d.ui.palette().uiTheme())
	busy := d.timerReq.isPending()
	primary, secondary := panelPrimary(snap), panelSecondary(snap)
	confirmationWasOpen := d.confirmEnd
	if d.confirmEnd && (d.confirmEndSession != snap.Session.ID || !sessionActive(snap)) {
		d.confirmEnd = false
	}
	ui.Column(c).Fill().Padding(16).Gap(10).Children(func() {
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Row(c).Children(func() {
					ui.Text(c, "小海底").Bold().Grow(1)
					ui.Text(c, fmt.Sprintf("今天 %d 輪", snap.TodayRounds)).FontSize(11).Opacity(.7)
				})
				if task := []rune(snap.Session.Task); len(task) > 0 {
					if len(task) > 22 {
						task = append(task[:22], '…')
					}
					ui.Text(c, string(task)).Label("當前任務").FontSize(12).Opacity(.8)
				}
				ui.Column(c).AlignItems(ui.Center).Gap(4).Children(func() {
					ui.Text(c, panelTime(snap)).Label("剩餘時間").FontSize(42).Bold()
					ui.Text(c, d.statusLine(snap)).Label("狀態").FontSize(12).Opacity(.7)
				})
				ui.Row(c).Gap(8).Children(func() {
					label := primary.Label
					if busy {
						label = "正在處理…"
					}
					if ui.PrimaryButton(c, label).Disabled(busy).Grow(1).Clicked() {
						d.startTimerAction(primary.Command, 0)
					}
					if secondary.Command != "" {
						if ui.Button(c, secondary.Label).Disabled(busy).Clicked() {
							d.startTimerAction(secondary.Command, 0)
						}
					} else if ui.Button(c, "結束本輪").Disabled(busy).Clicked() {
						d.confirmEnd, d.confirmEndSession = true, snap.Session.ID
					}
				})
				ui.Divider(c)
				label := "開啟聲音"
				if requested {
					label = "關閉聲音"
				}
				if ui.Button(c, label).Label(label).Disabled(d.ui.busy).FillWidth().Clicked() {
					d.requestAudio("toggle", 0)
				}
				volume := d.ui.volumeShown()
				ui.Row(c).Gap(8).Children(func() {
					ui.Text(c, "音量").FontSize(12)
					slider := ui.Slider(c, &volume, 0, 1).Label("音量").Grow(1)
					if slider.Changed() {
						d.ui.volumeDrag, d.ui.volumeActive = volume, true
						d.invalidateMain()
					}
					if d.ui.volumeActive && !slider.Pressed() && !d.ui.busy {
						d.requestAudio("set-volume", d.ui.volumeDrag)
					}
					ui.Text(c, fmt.Sprintf("%d%%", int(math.Round(volume*100)))).Width(34).FontSize(12)
				})

			})
		})
		message := d.timerReq.lastErr()
		if message == "" {
			message = d.uiReq.lastErr()
		}
		if message != "" {
			ui.Text(c, message).Label("錯誤").FontSize(11).TextColor(c.Theme().Danger)
		}
		ui.Divider(c)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, "開啟水族館").Clicked() {
				d.openAquarium()
			}
			mode := "掛件模式"
			if d.compact.Load() {
				mode = "普通視窗"
			}
			if ui.Button(c, mode).Disabled(d.uiReq.isPending()).Clicked() {
				d.requestCompact(!d.compact.Load())
			}
			if ui.Button(c, "設定…").Clicked() {
				d.OpenSettings()
			}
		})

	})
	if d.confirmEnd {
		if ui.AlertDialog(c, &d.confirmEnd, "結束本輪？", "已投入的專注會保留，不計為完成一輪。", "繼續這一輪", "結束本輪") == 1 {
			d.startTimerAction(timerEnd, d.confirmEndSession)
		}
	}
	if confirmationWasOpen != d.confirmEnd {
		d.invalidateNativeMenus()
	}
}
