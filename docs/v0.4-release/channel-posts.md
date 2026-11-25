# 六渠道帖子成稿（v0.4 定稿，可直接貼上）

> 來源：飛書物料定稿 docx/OQ8Qd9SaHokTnUxW7zCcSSIbncf（revision 5，2026-10-24）。
> **節奏（Q7）**：W4（11-16 週一 Release 先行 → 週二 HN/PH → 週三 V2EX/即刻 → 週四少數派 → 週六小紅書）。**所有發帖動作待使用者逐項確認。**

## 2.1 Hacker News（Show HN）

**用途**：英文主渠道，投放節奏見 Q7（北京時間週二晚 = 美東周二上午）。

**標題**（67 字元，符合 HN 上限）：

```text
Show HN: Little Reef – A no-pressure focus timer that grows a reef
```

**正文**（貼上後附 GitHub + 官網連結各一條，HN 正文不支援格式，純文字）：

```text
Hi HN — I made Little Reef, a quiet focus timer for macOS. You start a 25-minute session and a little reef in the window grows as you focus. Fish, shrimp and crabs occasionally drop by and get recorded in a collection log.

A few things I deliberately left out:

- No penalties. Stop halfway or walk away — nothing dies, nothing is lost, and the time you did spend still gets recorded.
- No streaks, no goals, no reminders. The review panel shows your week and month, but it never renders a broken streak or nags you back.
- No account, no telemetry. Records live in a local JSON file, and you can export a backup any time from Settings.
- No sounds out of nowhere — it never makes noise unless you ask it to.

It also has 20 ambient sounds (waves, rain, campfire, café…) that ship inside the app and loop without a seam, and it can shrink to a desktop widget or a menu-bar item with a tiny swimming fish.

Tech: Go with the MyGo framework, the front end embedded via go:embed (no JS build step), rendered by the system WKWebView. MIT licensed; the sounds are public-domain recordings, each credited in the repo.

Apple silicon only for now (macOS 12+). The framework can build for Windows/Linux but I haven't tested those, so I'm not shipping binaries I can't stand behind.

Happy to answer anything. Source: https://github.com/echoVic/little-reef — site: https://reef.echovic.com
```

**FAQ 備答**（評論區常見問題，逐條備好）：

1. **"Why arm64-only?"** — The auto-update chain and the notarized app are built and tested on Apple silicon. The framework supports other platforms and you can build from source, but I don't want to publish binaries I've never run.
2. **"What does it send anywhere?"** — Nothing. No account, no telemetry, no network calls except the update check and links you click. Records are one local JSON file, exportable from Settings.
3. **"How is this different from Forest?"** — Forest is a lovely product, but it punishes: leave the app and the tree dies. Little Reef never punishes — walk away and your partial time is still recorded, nothing dies, nothing is lost. It's also Mac-native (menu bar + widget) and open source.
4. **（備用）"Roadmap?"** — Short and evidence-driven: fix what release feedback surfaces first. Sound mixing and other platforms are candidates, gated on what users actually ask for.

## 2.2 Product Hunt

**用途**：英文第二渠道，週二發起，與 HN 錯開一天。

- **Tagline**（39 字元）：Grow a little reef with your focus time
- **一句話描述**：A no-pressure focus timer for Mac. Sessions grow a reef, visitors drop by and stay in a log, 20 ambient sounds loop seamlessly — no streaks, no nagging, nothing uploaded.
- **Gallery**：演示影片（PH 主視覺位）+ 5 張截圖，順序與說明如下（截圖說明即 alt 文案）：

| # | 截圖內容 | 說明文案 |
|-|-|-|
| 1 | 主視窗專注中（左海底右計時，選單欄小魚游動） | Your reef on the left, the timer on the right — a menu-bar fish swims while you focus |
| 2 | 訪客圖鑑（小丑魚畫像 + 「來過 N 次」） | First session brings a clownfish; 15 species in all, remembered forever |
| 3 | 「這段時間」回顧面板展開態 | This week, this month, the days you focused — records only, never reminders |
| 4 | 聲音面板（20 種背景音） | 20 ambient sounds, all offline, loop without a seam |
| 5 | 桌面掛件模式（角落小窗 + 倒計時） | Shrinks to a corner widget with just the reef and the countdown |

- **Maker comment**（釋出後 30 分鐘內跟評）：

```text
Hi Product Hunt! I'm the maker of Little Reef — a one-person, open-source side project.

Most focus timers felt like they were grading me: streaks to keep, trees that die, red badges when I slipped. I wanted the opposite — something that just keeps me company while I work.

So in Little Reef, nothing ever dies, nothing nags you, and nothing leaves your Mac. Sessions quietly grow a reef; fish and crabs drop by and stay in a log. v0.4 adds a gentle review panel (your week, your days, your collection — no streaks anywhere) and one-click archive export/import for backups.

It's free and MIT-licensed, and every ambient sound is a public-domain recording credited in the repo. Happy to answer anything in the comments!
```

## 2.3 V2EX（分享創造節點）

**用途**：中文主渠道，週三上午發帖。

**標題**：

```text
小海底：用專注時間養出一片海底的 macOS 計時器（MIT 開源，v0.4 新增回顧面板與存檔備份）
```

**正文**（約 460 字）：

```text
大家好，分享一個自己做的 macOS 專注計時器：小海底（little-reef）。

用法很簡單：按空格開始一輪專注（預設 25 分鐘），每完成一輪，左邊的海底就成長一點；每三輪多一株植物，長滿為止。偶爾會有新的魚、蝦或螃蟹來做客，記進圖鑑，一共 15 位朋友，沒見過的先來。

它和常見番茄鍾最大的不同是「無壓力」：中途停下或者直接走開，沒有魚會死、也不會扣掉什麼，投入過的時間照樣記錄；不顯示連續打卡，沒有目標和提醒，不會突然出聲。記錄只儲存在本機，沒有賬號、不上傳——換機時可以在設定裡一鍵匯出存檔，匯入回去一切如初。

20 種背景音（海浪、雨聲、篝火、咖啡館、木魚、卡農……）全部打包在應用裡，迴圈聽不出接縫，播放不聯網。視窗可以收成桌面掛件，或者收進選單欄，專注時選單欄有一條小魚輕輕游動。

v0.4 剛加了「這段時間」回顧面板：本週、本月、一起度過的天數、圖鑑進度，展開就能看，全部是正向記錄——不渲染中斷、不設目標、不打擾。存檔匯出/匯入同版往返無損。

技術棧：Go + MyGo 桌面框架，前端 go:embed 直接內嵌、沒有 JS 構建步驟，WebView 用系統自帶的 WKWebView，MIT 開源，背景音是公共領域錄音，出處都在倉庫裡。

需要 macOS 12+ 和 Apple 晶片，免費。歡迎試試：
官網：https://reef.echovic.com
原始碼：https://github.com/echoVic/little-reef
```

**配圖說明**（3–5 張，帖內依序插入）：

1. 主視窗專注中：左海底右計時（開場圖，體現產品形態）
2. 訪客圖鑑：小丑魚來訪記錄（體現「來過 N 次」）
3. 「這段時間」回顧面板展開態（v0.4 主特性）
4. 桌面掛件 + 選單欄小魚（體現形態差異）
5. 設定裡的「備份與恢復」（體現存檔安全網）

**回覆預案**（按 V2EX 社羣高頻問題各備一條）：

1. **「為什麼只有 macOS / 只有 arm64？」**——如實：自動更新鏈按 macOS 設計、我在 Apple 晶片上開發測試；Win/Linux 可原始碼構建但沒實測過，不想發自己沒跑過的包。等有真實訴求再排期。
2. **「為什麼不做 iOS / 雲同步？」**——隱私是承諾不是短板：無賬號、不上傳是官網寫明的承諾；換機走存檔匯出，成本近零。雲同步會推翻這條產品線，寧可不做。
3. **「和 Forest 有什麼區別？」**——Forest 中途離開樹會枯死（懲罰機制）；小海底不懲罰，提前結束也記錄時間、什麼都不死。另外小海底是 Mac 原生（選單欄/掛件）且開源。

## 2.4 即刻

**用途**：短動態，週三晚，投「獨立開發者」「效率工具」兩個圈子。

**動態正文**：

```text
一個人做的小產品：小海底。按空格開始專注，每完成一輪，桌面上的海底就長大一點；偶爾有魚蝦螃蟹來做客，記進圖鑑。中途走開也不會死魚，不打卡、不催促、不上傳。收進選單欄，專注時有一條小魚陪你遊。

v0.4 新增「這段時間」回顧：本週專注、一起度過的天數、圖鑑進度——只做記錄，從不提醒。

免費開源（MIT），Apple 晶片 Mac 可用：reef.echovic.com
```

**圖組**（4 圖，每圖配一句話）：

1. 專注中：左邊海底生長，右邊計時走動 —— 配文：專注時，它就慢慢長
2. 小丑魚來訪，圖鑑記下第一位朋友 —— 配文：第一位朋友總會來
3. 「這段時間」面板：本週專注與天數 —— 配文：數字只是陪著，不是成績單
4. 收成掛件後的桌面一角 —— 配文：不用的時候，它就安靜待在角落

**圈子 tag**：#獨立開發者 #效率工具

## 2.5 少數派（sspai 投稿）

**用途**：週三投稿（稽覈有延遲，宜早投），800–1500 字，工作流場景切入。

**標題**：小海底：一款從不催你的 Mac 專注計時器

**正文**（約 980 字）：

```text
番茄鍾工具大概分兩類：一類遊戲化很重，任務、勳章、商店，專注本身反而成了支線劇情；另一類極簡到只剩一個倒計時，走不走完全靠自己。小海底（little-reef）想試試第三種：讓專注本身有一點溫柔的回報，但不給你任何壓力。

它是一個 macOS 應用，開源（MIT）、免費。一句話概括：你專注，桌面上的海底就慢慢長大。

專注：按空格開始一輪，預設 25 分鐘。視窗左半邊是一片水下世界：水草會隨完成的輪數長高，每三輪多一株植物，蝦和螃蟹在沙地上爬、會躲進石縫。專注進行時，選單欄圖示會變成一條輕輕游動的小魚——不看倒計時，也知道時間在走。

休息：一輪結束，預設休息 5 分鐘。休息時可以喂一次魚：魚食慢慢下沉，吃剩的落到沙地上，被蝦蟹撿走。等一等，也可以不那麼難熬。

回看：v0.4 新增「這段時間」摺疊面板——本週和本月的專注時長、一起度過的天數、圖鑑進度、當前成長階段，展開就能看。值得注意的是它刻意不顯示「連續 N 天」：中斷不渲染、不標紅、不提醒，只有正向的記錄。數字是陪伴，不是 KPI。

訪客：圖鑑是另一條溫柔的線。第一輪專注必定遇見小丑魚，之後每輪有三分之一的機會來一位新朋友——4 種魚、6 種蝦、5 種蟹，全部按真實物種繪製，沒見過的先來。都見過之後，老朋友會再來做客。

「無壓力」不是口號，是幾條硬規則：中途停下或直接走開，沒有魚會死，投入過的時間照樣記錄；不顯示打卡和目標，不發提醒推送；從不突然出聲，第一次開啟也不會；Mac 休眠的時間不算進專注。

隱私上同樣乾淨：沒有賬號，不上傳任何東西，你的記錄就是本地的一個檔案。想換機或備份，v0.4 可以在設定裡一鍵匯出存檔；匯入前會校驗，失敗不會影響當前資料，匯出再匯入，內容完全一樣。

其它細節：20 種背景音（海浪、雨聲、篝火、咖啡館、木魚、卡農……）全部打包在本機，迴圈聽不出接縫；視窗可以收成桌面掛件，或者收進選單欄；系統開啟「減少動態效果」時會自動停下所有動畫，只留計時。

它需要 macOS 12 或更高、Apple 晶片的 Mac。官網 reef.echovic.com 有完整介紹和下載，原始碼在 GitHub（echoVic/little-reef）。如果你也想要一個不催你的專注計時器，可以試試。
```

**配圖建議**（隨投稿附）：主視窗專注圖、「這段時間」面板、掛件模式、圖鑑頁 4 張（沿用官網新截圖）。

## 2.6 小紅書

**用途**：週六發，生活化口吻，豎版圖組（3:4）。

**筆記標題**（20 字內）：桌面角落，養了一片小海底

**首圖文案**（豎版封面，兩行大字）：

```text
專注的時候
它就悄悄長大
```

**正文**：

```text
最近把 Mac 的角落讓給了一片海。

開始專注它就慢慢長，暫停也不催你，走開了也不會死魚。
一輪結束偶爾有魚蝦來做客，圖鑑會一直記著，現在集了 5 位。

20 種白噪音都在本機：海浪、咖啡館、木魚、卡農，換著聽，迴圈聽不出接縫。
不用的時候收起來，只剩角落一個小掛件，或者躲進選單欄變成一條小魚。

沒有打卡，沒有提醒，記錄只存在自己電腦裡，想備份隨時匯出。

免費開源，Apple 晶片的 Mac 可以試試。
```

**圖組**（5 圖豎版，依序）：

1. 封面：掛件在乾淨桌面的角落（首圖兩行大字）
2. 主視窗：海底生長中 + 計時器
3. 圖鑑：第一位朋友小丑魚
4. 「這段時間」面板：本週專注與一起度過的天數
5. 聲音面板 + 掛件對比圖

**話題標籤**：#macOS #桌面美化 #桌面小元件 #效率工具 #專注 #白噪音 #開源軟體 #獨立開發
