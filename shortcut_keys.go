package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	shortcutPrimary  = "timer-primary"
	shortcutEnd      = "timer-end"
	shortcutShow     = "show-main"
	shortcutCompact  = "compact-mode"
	shortcutSettings = "settings"
	shortcutGlobal   = "global-show"
)

type shortcutDefinition struct {
	ID, Label, Default string
	Global             bool
}

var shortcutDefinitions = []shortcutDefinition{
	{shortcutPrimary, "開始／暫停／繼續", "CmdOrCtrl+Enter", false},
	{shortcutEnd, "結束本輪", "CmdOrCtrl+Shift+Enter", false},
	{shortcutShow, "顯示水族館", "CmdOrCtrl+1", false},
	{shortcutCompact, "切換桌面掛件", "CmdOrCtrl+Shift+M", false},
	{shortcutSettings, "開啟設定", "CmdOrCtrl+,", false},
	{shortcutGlobal, "全域喚起水族館", "CmdOrCtrl+Shift+A", true},
}

type shortcutKey struct {
	Key           ui.Key
	Name, Display string
}

var shortcutKeys = func() []shortcutKey {
	keys := []shortcutKey{
		{ui.KeyEnter, "Enter", "↵"}, {ui.KeySpace, "Space", "空格"}, {ui.KeyTab, "Tab", "⇥"},
		{ui.KeyEscape, "Escape", "⎋"}, {ui.KeyBackspace, "Backspace", "⌫"}, {ui.KeyDelete, "Delete", "⌦"},
		{ui.KeyUp, "Up", "↑"}, {ui.KeyDown, "Down", "↓"}, {ui.KeyLeft, "Left", "←"}, {ui.KeyRight, "Right", "→"},
		{ui.KeyHome, "Home", "Home"}, {ui.KeyEnd, "End", "End"}, {ui.KeyPageUp, "PageUp", "PageUp"}, {ui.KeyPageDown, "PageDown", "PageDown"},
		{ui.KeyComma, ",", ","}, {ui.KeyPeriod, ".", "."}, {ui.KeyMinus, "-", "-"}, {ui.KeyEqual, "=", "="},
		{ui.KeySlash, "/", "/"}, {ui.KeySemicolon, ";", ";"}, {ui.KeyQuote, "'", "'"},
		{ui.KeyBracketLeft, "[", "["}, {ui.KeyBracketRight, "]", "]"}, {ui.KeyBackslash, "\\", "\\"}, {ui.KeyBackquote, "`", "`"},
	}
	for i, key := range []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ} {
		name := string(rune('A' + i))
		keys = append(keys, shortcutKey{key, name, name})
	}
	for i, key := range []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9} {
		name := string(rune('0' + i))
		keys = append(keys, shortcutKey{key, name, name})
	}
	for i, key := range []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12} {
		name := fmt.Sprintf("F%d", i+1)
		keys = append(keys, shortcutKey{key, name, name})
	}
	return keys
}()

func shortcutString(mods ui.Modifiers, key string) string {
	var parts []string
	if mods&ui.Cmd != 0 {
		parts = append(parts, "CmdOrCtrl")
		mods &^= ui.Cmd
	}
	for _, m := range []struct {
		bit  ui.Modifiers
		name string
	}{{ui.Ctrl, "Ctrl"}, {ui.Super, "Super"}, {ui.Alt, "Alt"}, {ui.Shift, "Shift"}} {
		if mods&m.bit != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, key), "+")
}

func normalizeShortcut(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	parts := strings.Split(raw, "+")
	var mods ui.Modifiers
	for _, part := range parts[:len(parts)-1] {
		var bit ui.Modifiers
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "cmdorctrl", "commandorcontrol":
			bit = ui.Cmd
		case "cmd", "command", "super", "meta", "win":
			bit = ui.Super
		case "ctrl", "control":
			bit = ui.Ctrl
		case "alt", "option":
			bit = ui.Alt
		case "shift":
			bit = ui.Shift
		default:
			return "", fmt.Errorf("無法識別修飾鍵 %q", part)
		}
		if mods&bit != 0 {
			return "", fmt.Errorf("修飾鍵重複")
		}
		mods |= bit
	}
	if mods&(ui.Cmd|ui.Ctrl|ui.Alt|ui.Super) == 0 {
		return "", fmt.Errorf("請至少包含 Command、Ctrl 或 Alt 中的一個修飾鍵")
	}
	name := strings.TrimSpace(parts[len(parts)-1])
	if strings.EqualFold(name, "Return") {
		name = "Enter"
	}
	for _, key := range shortcutKeys {
		if strings.EqualFold(name, key.Name) {
			return shortcutString(mods, key.Name), nil
		}
	}
	return "", fmt.Errorf("不支援的按鍵 %q", name)
}

func shortcutFromInput(mods ui.Modifiers, key ui.Key) (string, error) {
	for _, k := range shortcutKeys {
		if k.Key == key {
			return normalizeShortcut(shortcutString(mods, k.Name))
		}
	}
	return "", fmt.Errorf("這個按鍵暫不支援，請換一個組合")
}

func displayShortcut(raw string) string {
	if raw == "" {
		return "未設定"
	}
	normalized, err := normalizeShortcut(raw)
	if err != nil {
		return raw
	}
	parts := strings.Split(normalized, "+")
	if runtime.GOOS != "darwin" {
		return strings.ReplaceAll(normalized, "CmdOrCtrl", "Ctrl")
	}
	for i, part := range parts {
		switch part {
		case "CmdOrCtrl", "Super":
			parts[i] = "⌘"
		case "Ctrl":
			parts[i] = "⌃"
		case "Alt":
			parts[i] = "⌥"
		case "Shift":
			parts[i] = "⇧"
		default:
			for _, key := range shortcutKeys {
				if key.Name == part {
					parts[i] = key.Display
					break
				}
			}
		}
	}
	return strings.Join(parts, "")
}

func reservedShortcut(key string) bool {
	for _, raw := range []string{
		"CmdOrCtrl+Q", "CmdOrCtrl+Shift+Q", "CmdOrCtrl+W", "CmdOrCtrl+Shift+W", "CmdOrCtrl+M", "CmdOrCtrl+Alt+M", "CmdOrCtrl+H", "CmdOrCtrl+Alt+H",
		"CmdOrCtrl+A", "CmdOrCtrl+C", "CmdOrCtrl+V", "CmdOrCtrl+X", "CmdOrCtrl+Z", "CmdOrCtrl+Shift+Z", "CmdOrCtrl+Y",
		"CmdOrCtrl+Alt+Shift+V", "CmdOrCtrl+Alt+I", "Ctrl+Shift+I", "Ctrl+Cmd+F", "Alt+F4",
		"CmdOrCtrl+F", "CmdOrCtrl+R", "CmdOrCtrl+Shift+R", "CmdOrCtrl+0", "CmdOrCtrl+-", "CmdOrCtrl+=", "CmdOrCtrl+Shift+=",
		"CmdOrCtrl+Backspace", "CmdOrCtrl+Delete", "Ctrl+Backspace", "Ctrl+Delete",
		"CmdOrCtrl+`", "CmdOrCtrl+Shift+`",
		"CmdOrCtrl+Tab", "CmdOrCtrl+Shift+Tab", "Alt+Tab", "Alt+Shift+Tab", "CmdOrCtrl+Space", "Ctrl+Space", "Super+Space",
		"CmdOrCtrl+Alt+Escape", "Ctrl+Alt+Delete", "Ctrl+Shift+Escape",
	} {
		normalized, _ := normalizeShortcut(raw)
		if normalized == key {
			return true
		}
	}
	return false
}
