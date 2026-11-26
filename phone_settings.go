package main

import (
	"github.com/egoist/mygo/ui"
)

// On phones, settings is a page over the main window rather than a window
// of its own: a pushed window's system title bar shows the system's colors
// until its content first reports its own, and its text color is the
// system's, not the app's. The page draws its own bar in the app's colors,
// and Android's back closes it, as it closes what Escape closes.

// openSettingsPage shows the settings page, keeping a draft it already has.
func (d *Desktop) openSettingsPage() {
	if d.settings == nil {
		s := newNativeSettings(d)
		s.win = d.win // file dialogs and updates go through the main window
		d.settings = s
	}
	d.ui.settingsOpen = true
	d.invalidateMain()
}

// closeSettingsPage takes the page away, with its unsaved draft.
func (d *Desktop) closeSettingsPage() {
	d.ui.settingsOpen = false
	if s := d.settings; s != nil {
		d.settings = nil
		s.stopShortcutRecording()
		s.cancel()
	}
	d.invalidateMain()
}

// settingsPage is the full-screen settings page of phones.
func (m *mainUI) settingsPage(c *ui.Context, p *palette) {
	s := m.d.settings
	if !m.settingsOpen || s == nil {
		if s != nil && onPhone {
			m.d.closeSettingsPage() // Android's back closed it
		}
		return
	}
	ui.DialogBase(c, &m.settingsOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Bg)
		// Overlays draw under the system's bars: keep the page's bar clear of
		// the status bar, and its buttons of the gesture bar.
		top, right, bottom, left := c.SafeArea()
		panel.Fill().Padding(top, right, bottom, left).Background(p.Bg).Role(ui.RoleDialog).Label("小海底設定")
		ui.Row(c).Height(56).Padding(0, 8).Gap(4).BorderWidth(0, 0, 1, 0).BorderColor(p.Line).Shrink(0).Children(func() {
			back := cssButton(c, false, func(b ui.Element, h float32) {
				b.Size(44, 44).Radius(22).Label("返回")
				look{bg: p.HoverBg.Alpha(0), fg: p.Text, hbg: p.HoverBg, hfg: p.Text, filter: 1}.apply(b, h)
				b.Children(func() {
					ui.Box(c).Size(20, 20).Draw(func(pt *ui.Painter, r ui.Rect) {
						arrow := (&ui.Path{}).MoveTo(r.X+13, r.Y+4).LineTo(r.X+6, r.Y+10).LineTo(r.X+13, r.Y+16)
						pt.StrokePath(arrow, 2, p.Text)
					})
				})
			})
			if back.Clicked() {
				s.close()
			}
			ui.Text(c, "小海底設定").FontSize(17).FontWeight(500).TextColor(p.Text).SingleLine()
		})
		ui.Column(c).Grow(1).FillWidth().Children(func() { s.view(c) })
	})
}
