package main

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func reviewBase() Snapshot {
	return Snapshot{
		Completed: 5, TotalSecs: 3900, WeekSecs: 900, WeekRounds: 2, WeekDays: 1,
		MonthSecs: 3900, MonthRounds: 3, TotalDays: 4, NextTier: 7,
		Records: []Record{{ID: 1, Task: "a", Seconds: 600, Completed: true, Date: "2026-10-26T09:00:00+08:00"}},
		Visitors: []Visitor{
			{Species: "clownfish", Name: "小丑魚", Visits: 2, Date: "2026-10-26T09:00:00+08:00"},
			{Species: "turtle", Name: "海龜", Visits: 1, Date: "2026-10-20T09:00:00+08:00"},
		},
	}
}

func reviewTexts(r review) []string {
	var out []string
	for _, l := range r.lines {
		if l.text != "" {
			out = append(out, l.text)
		}
	}
	return out
}

func TestReviewCopyMatchesThePage(t *testing.T) {
	if minutesText(59*60) != "59 分鐘" || minutesText(90*60) != "1 小時 30 分鐘" || minutesText(3900) != "1 小時 5 分鐘" {
		t.Fatal("minutesText")
	}
	for n, want := range map[int]string{0: "初生", 2: "初生", 3: "萌芽", 11: "萌芽", 12: "繁盛", 24: "小小綠洲"} {
		if tierName(n) != want {
			t.Errorf("tierName(%d) = %q", n, tierName(n))
		}
	}
	if r := reviewModel(Snapshot{}); !r.empty || r.emptyText != "完成第一輪，這裡開始有故事。" || r.footnote != "這些數字只儲存在你的電腦上。" {
		t.Fatalf("empty review: %+v", r)
	}
	got := reviewTexts(reviewModel(reviewBase()))
	want := []string{"本週 15 分鐘 · 2 輪", "本週 1 天有你的專注", "本月 1 小時 5 分鐘 · 3 輪", "已經一起度過 1 小時 5 分鐘，完成 5 輪專注。",
		"一共 4 天", "圖鑑 2 / 15", "還有 13 位朋友沒來過，慢慢來", "最近到訪：小丑魚 · 來過 2 次", "當前：萌芽", "再 7 輪就是繁盛", "海底已有 2 株植物，長滿後不再擁擠"}
	if !slices.Equal(got, want) {
		t.Fatalf("review lines\n got %q\nwant %q", got, want)
	}
	s := reviewBase()
	s.WeekSecs, s.WeekRounds, s.WeekDays, s.MonthSecs, s.MonthRounds, s.TotalDays, s.NextTier = 0, 0, 0, 0, 0, 0, 0
	got = reviewTexts(reviewModel(s))
	for _, line := range []string{"本週還沒有，不打緊，隨時開始。", "本月 0 分鐘 · 0 輪", "一共 0 天", "繼續積累專注，認識更多海底朋友"} {
		if !slices.Contains(got, line) {
			t.Errorf("zeroed review lacks %q: %q", line, got)
		}
	}
	if plantCount(0) != 0 || plantCount(1) != 1 || plantCount(25) != 8 {
		t.Fatal("plantCount")
	}
}

func TestDeriveTimerFollowsTheSession(t *testing.T) {
	s := Snapshot{Preferences: Preferences{FocusMinutes: 25, BreakMinutes: 5}, Session: Session{Kind: "focus", Status: "idle"}}
	if v := deriveTimer(s, "focus"); v.Time != "25:00" || v.Primary != "開始專注" || v.PrimaryIcon != "play" || !v.EndHidden || v.TaskDisabled {
		t.Fatalf("idle focus: %+v", v)
	}
	if v := deriveTimer(s, "break"); v.Time != "05:00" || v.Primary != "開始休息" || !v.TaskDisabled || v.Placeholder != "去喝杯水，讓眼睛也休息一下" {
		t.Fatalf("idle break tab: %+v", v)
	}
	s.Session = Session{ID: 3, Kind: "focus", Status: "running", DurationMS: 1500000, ElapsedMS: 600400}
	if v := deriveTimer(s, "focus"); v.Time != "15:00" || v.Primary != "暫停一下" || v.PrimaryIcon != "pause" || v.EndHidden || v.Progress < 0.4 || v.Status != "小魚陪著你，慢慢來" {
		t.Fatalf("running: %+v", v)
	}
	s.Session.Status = "paused"
	if v := deriveTimer(s, "focus"); v.Primary != "繼續這一輪" || !v.Paused || v.Status != "暫停一下，沒關係" {
		t.Fatalf("paused: %+v", v)
	}
	s.Session = Session{ID: 4, Kind: "focus", Status: "complete", DurationMS: 1500000, ElapsedMS: 1500000}
	s.LastVisitor = "小丑魚"
	if v := deriveTimer(s, "focus"); v.Primary != "開始休息" || v.PrimaryIcon != "coffee" || v.Caption != "小丑魚來做客了，已留在圖鑑裡。" {
		t.Fatalf("complete: %+v", v)
	}
	s.Session = Session{ID: 5, Kind: "break", Status: "running", DurationMS: 300000}
	if v := deriveTimer(s, "break"); v.FeedDisabled || v.FeedLabel != "撒一點魚食" {
		t.Fatalf("break feeding: %+v", v)
	}
	s.FedSession = 5
	if v := deriveTimer(s, "break"); !v.FeedDisabled || v.FeedLabel != "魚兒吃飽啦" {
		t.Fatalf("fed: %+v", v)
	}
}

func TestApplyDropsStaleSnapshotsAndNeverAutoplaysOnLoad(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	running := a.Get()
	running.Revision = 3
	running.Preferences.SoundEnabled = true
	running.Session = Session{ID: 7, Kind: "focus", Status: "running", DurationMS: 1500000, ElapsedMS: 600000}
	d.ui.apply(running)
	if d.ui.soundPlaying {
		t.Fatal("loading an already-running session autoplayed the ambience")
	}
	stale := running
	stale.Revision = 2
	stale.Session.Status = "paused"
	d.ui.apply(stale)
	if d.ui.snap.Session.Status != "running" {
		t.Fatal("a stale snapshot replaced a newer one")
	}
}

// settle lets the goroutines started by the view finish and hands their
// results to the next frame.
func settle(t *testing.T, tt *ui.Tester, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !done() && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
		tt.Frame()
	}
	if !done() {
		t.Fatalf("view did not settle: %v", tt.Texts())
	}
}

func TestMainViewDrawsThePageAndRunsARound(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	d.ui.apply(a.Get())
	tt := ui.NewTester(d.ui.view, 1100, 800)
	for _, text := range []string{"小海底", "屬於你的安靜角落", "第一片小海底", "專注", "休息", "開始專注", "海底的聲音", "海浪", "粉噪", "開啟聲音"} {
		if !tt.HasText(text) {
			t.Fatalf("page lacks %q: %v", text, tt.Texts())
		}
	}
	if timer, ok := tt.AccessByRole(ui.RoleTimer, "剩餘時間"); !ok || timer.Bounds.W == 0 {
		t.Fatalf("timer face: %+v", timer)
	}
	if err := tt.Click("開始專注"); err != nil {
		t.Fatal(err)
	}
	settle(t, tt, func() bool { return tt.HasText("暫停一下") && tt.HasText("結束本輪") })
	if err := tt.Click("結束本輪"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("這一段時間，也算數。") {
		t.Fatalf("end dialog did not open: %v", tt.Texts())
	}
	if err := tt.Click("結束並記錄"); err != nil {
		t.Fatal(err)
	}
	settle(t, tt, func() bool { return a.Get().Session.Status != "running" && tt.HasText("開始專注") })
	if len(a.Get().Records) != 1 {
		t.Fatalf("ending early did not keep the record: %+v", a.Get().Records)
	}
}

func TestMainViewSpaceStartsTheTimerButNotWhileTyping(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	a.onChange = d.changed
	d.ui.apply(a.Get())
	tt := ui.NewTester(d.ui.view, 1100, 800)
	field, ok := tt.AccessByRole(ui.RoleTextField, "這一輪，想做什麼？")
	if !ok || field.Placeholder != "比如讀幾頁書，寫一小段程式碼" {
		t.Fatalf("task field: %+v %v", field, ok)
	}
	r := field.Bounds
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Type("讀 書")
	if a.Get().Session.Status != "idle" || d.ui.task != "讀 書" {
		t.Fatalf("space in the task field started the timer: %q %+v", d.ui.task, a.Get().Session)
	}
	tt.ClickAt(5, 400) // the page, away from any control
	tt.Key(0, ui.KeySpace)
	settle(t, tt, func() bool { return a.Get().Session.Status == "running" })
	if got := a.Get().Session.Task; got != "讀 書" {
		t.Fatalf("task not passed to the round: %q", got)
	}
}

func TestCompactViewIsTheDesktopWidget(t *testing.T) {
	a, _ := testAquarium(t)
	d := newDesktop(a)
	d.compact.Store(true)
	d.ui.apply(a.Get())
	tt := ui.NewTester(d.ui.view, 400, 350)
	if !tt.HasText("01:00") || tt.HasText("海底的聲音") || tt.HasText("第一片小海底") {
		t.Fatalf("compact layout: %v", tt.Texts())
	}
	for _, label := range []string{"開始專注", "開啟聲音", "展開水族館"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("compact control %q missing", label)
		}
	}
	if err := tt.Click("展開水族館"); err != nil {
		t.Fatal(err)
	}
	if d.compact.Load() {
		t.Fatal("expand did not leave compact mode")
	}
}
