package main

import (
	"github.com/egoist/mygo/ui"
	"little-reef/internal/lockscreen"
)

func (s *nativeSettings) generalView(c *ui.Context) {
	ui.Fieldset(c, "計時與提示音", func() {
		ui.Form(c, func() {
			ui.Field(c, "專注時長", func() { ui.TextInput(c, &s.focusMinutes).Width(120).Disabled(s.busy) }).Description("1–120 分鐘")
			ui.Field(c, "休息時長", func() { ui.TextInput(c, &s.breakMinutes).Width(120).Disabled(s.busy) }).Description("1–30 分鐘")
		})
		ui.Checkbox(c, &s.chime, "完成時播放輕柔提示音").Disabled(s.busy)
		ui.Text(c, "時長調整從下一輪開始，本輪計時保持不變。").FontSize(12).Opacity(.7)
		if onPhone {
			ui.Text(c, "鎖定螢幕或切到其他應用時，計時會繼續，並顯示在鎖定畫面上。").FontSize(12).Opacity(.7)
		} else {
			ui.Text(c, "電腦休眠和應用退出會暫停計時，回來後手動繼續。").FontSize(12).Opacity(.7)
		}
	})
	if lock := s.desktop.lock; onPhone && lock != nil {
		ui.Fieldset(c, "鎖定畫面倒數", func() {
			if lock.NotificationsBlocked() {
				ui.Text(c, "通知已關閉：鎖定畫面、狀態列和超級島都看不到倒數。").FontSize(12)
				ui.Text(c, notificationsAdvice).FontSize(12).Opacity(.7)
				if ui.Button(c, "開啟通知設定").Clicked() {
					s.desktop.openNotificationSettings()
				}
			} else {
				ui.Text(c, "通知已開啟：倒數會顯示在鎖定畫面與狀態列。").FontSize(12)
			}
			style := s.desktop.lockStyle
			ui.Radio(c, &style, lockscreen.StyleLive, "超級島與狀態列：系統樣式，按鈕為文字")
			ui.Radio(c, &style, lockscreen.StyleLarge, "大字倒數：大大的倒數與圖示按鈕，只在鎖定畫面與通知欄")
			ui.Radio(c, &style, lockscreen.StyleMedia, "媒體控制卡（實驗）：圖示按鈕與進度條，如同播放器")
			if style != s.desktop.lockStyle {
				s.feedback(s.desktop.setLockStyle(style), "鎖定畫面樣式已切換。")
			}
		})
		ui.Fieldset(c, "背景執行", func() {
			status := "已允許：鎖定螢幕時倒數會繼續。"
			if lock.BackgroundRestricted() {
				status = "受電池最佳化限制：鎖定後小海底可能被系統停止，倒數也會停住。"
			}
			ui.Text(c, status).FontSize(12)
			ui.Text(c, backgroundAdvice).FontSize(12).Opacity(.7)
			if ui.Button(c, "前往設定").Clicked() {
				s.desktop.allowBackground()
			}
		})
	}
}

func (s *nativeSettings) appearanceView(c *ui.Context) {
	a := s.desktop.a
	snap := a.Get()
	ui.Fieldset(c, "海底皮膚", func() {
		ui.Form(c, func() {
			ui.Field(c, "當前皮膚", func() {
				name := a.themes.names[snap.Theme]
				ui.MenuButton(c, name, func(m *ui.Menu) {
					for _, theme := range a.themes.list() {
						if m.Item(theme.Name).Radio(theme.ID == snap.Theme).Chosen() {
							_, err := a.SetTheme(theme.ID)
							s.feedback(err, "皮膚已切換，自動模式已關閉。")
						}
					}
				}).Label("海底皮膚").Disabled(s.busy)
			})
			auto, shuffle := snap.ThemeAuto, snap.ThemeShuffle
			var autoSwitch, shuffleSwitch ui.Element
			ui.Field(c, "隨時間自動切換主題", func() {
				autoSwitch = ui.Switch(c, &auto).Disabled(s.busy)
			}).Description("06:00 馬卡龍 · 10:00 賽璐璐\n17:00 青綠江山 · 20:00 深海寫實")
			if autoSwitch.Changed() {
				_, err := a.SetThemeAuto(auto)
				s.feedback(err, "主題模式已儲存。")
			}
			ui.Field(c, "全部主題隨機輪換", func() {
				shuffleSwitch = ui.Switch(c, &shuffle).Disabled(s.busy)
			}).Description("每 30 分鐘隨機切換，不連續重複。")
			if shuffleSwitch.Changed() {
				_, err := a.SetThemeShuffle(shuffle)
				s.feedback(err, "主題模式已儲存。")
			}
		})
		ui.Text(c, "皮膚和主題模式立即生效；兩種自動模式只能開啟一種。").FontSize(12).Opacity(.7)
	})
}

func (s *nativeSettings) archiveView(c *ui.Context) {
	ui.Fieldset(c, "備份與恢復", func() {
		ui.Text(c, "匯出一份存檔；換電腦或重新安裝時可以恢復。").FontSize(12).Opacity(.7)
		ui.Row(c).Gap(10).Children(func() {
			if ui.Button(c, "匯出存檔…").Disabled(s.busy).Clicked() {
				s.exportArchive()
			}
			if ui.Button(c, "匯入存檔…").Disabled(s.busy).Clicked() {
				s.confirmImport = true
			}
		})
	})
}
