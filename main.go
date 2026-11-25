//go:build !harness

// Little Reef is a focus timer whose completed sessions grow a small reef.
// Every window is MyGo native UI; nothing loads a web page or the network.
package main

import (
	"flag"
	"log"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

func main() {
	dataDir := flag.String("data-dir", "", "指定獨立存檔目錄")
	exportIcons := flag.Bool("export-icons", false, "重新生成 assets/icon.png 後退出")
	flag.Parse()
	if *exportIcons {
		if err := exportIconAssets(); err != nil {
			log.Fatal(err)
		}
		return
	}
	mygo.App.SetName("小海底")
	mygo.App.SetLanguages("zh-Hant")
	dir := *dataDir
	if dir == "" {
		var err error
		if dir, err = mygo.App.Path(mygo.PathUserData); err != nil {
			log.Fatal(err)
		}
	}
	a, err := openAquarium(dir)
	if err != nil {
		log.Fatal(err)
	}
	desktop := newDesktop(a)
	desktop.shortcuts = newShortcutController(dir, mygo.GlobalShortcut, desktop.show)
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	if err := desktop.run(); err != nil {
		log.Fatal(err)
	}
}
