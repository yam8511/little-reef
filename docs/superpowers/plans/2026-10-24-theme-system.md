# 主題系統（多主題包）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把「純寫實深海底」的硬編碼視覺，改造成可切換的主題包系統：現有寫實主題作為 `reef-realistic` 納入同一 token 契約，九套新主題（cn-shuimo / cn-qinghua / cn-qinglu / cm-pop / cm-cel / cm-ink / ct-cream / ct-strawberry / ct-jelly）逐套落地，切換入口為選單欄下拉子選單（使用者拍板）。全程守住硬紅線：**主題是純表現層，切換絕不觸碰存檔**。

成功定義：
1. 切換主題 = 重綁 CSS 變數 + 替換生物繪製引用，不產生任何 aquarium.json 讀寫；有測試鎖定「切換 100 次後 aquarium.json 位元組級不變」。
2. 主題選擇持久化（重啟後保留），但**不寫入 aquarium.json**，滿足上條迴歸口徑。
3. 選單欄「主題 ▸」子選單單選列表（勾選態），切換即時生效、無彈窗、無慶祝動效。
4. 現有寫實視覺零迴歸（reef-realistic 抽取後逐畫素不變，由現有截圖/測試兜底）。
5. 各主題動效密度按設計文件分級，系統「減少動態效果」開啟時全部靜止停靠。

**Architecture:**

- **theme 包形態**：`web/themes/<themeId>/theme.json`（meta + tokens + assets 清單），隨 `//go:embed web` 一起內嵌 [現狀：web/ 整目錄 embed]。theme.json 結構嚴格按設計文件第 04 章 a 節：meta（themeId/displayName/styleFamily/version/minAppVersion/supportsDarkAppearance/reduceMotionFallback）+ tokens（背景 4 + 生物 4 + UI 8 + 動效守門 4）+ assets 清單。未提供的 token 回退繼承 reef-realistic 的值（設計文件 04a：「新主題可以只寫差異項」）。Go 側用 struct 定義 schema（設計文件 04c：「theme.json schema 用 Go 側 struct 定義並生成 TypeScript 型別」）。
- **token → CSS 變數對映**：theme.js 讀當前主題 token 表，寫到 `:root` 的一組變數（現有 style.css:1 已有 `:root{--bg...}` 10 個變數 [C1]），新增 `--timer-active/--timer-pause/--control-bg/--control-border/--text-secondary/--water-light/--sand/--bubble` 等，把 style.css 裡硬編碼 hex 全部改為 var()（清單見下文「Token 化改動面」）。
- **生物繪製替換機制**：canvas 繪製函式（reef.js、web/creatures/*）當前硬編碼顏色（清單見下）。改造為：繪製前從 `getComputedStyle(document.documentElement).getPropertyValue(...)` 讀取主題色，或由 theme.js 在切換時把 token 表注入一個全域 `THEME` 物件供模組讀取。**生物層不做 sprite 貼圖替換，按文字規格程式化重繪**（使用者硬約束 6：生物資產按文字規格引數化重繪，禁止把 mood 圖當成品）——即把 fish.js/shrimp.js 裡寫死的顏色引數改成讀 token，造型演算法（輪廓/鰭/尾巴）保留，顏色/描邊寬/透明度走 token。
- **切換入口接線**：main.go:252-255 現有托盤選單 `mygo.NewMenu([]*mygo.MenuItem{...})`。mygo v0.6.0 `MenuItem.Submenu []*MenuItem`（menu.go:105）+ `MenuItemRadio`（menu.go:19）+ `SetChecked` 自動取消同組兄弟勾選（menu.go:247-323）+ `Tray.SetMenu(*Menu)`（modules.go:755）均已存在——**子選單可行，無需降級**。新增「主題 ▸」子選單項，每組一項 radio（勾選當前主題），點選 → RPC `Aquarium.SetTheme(id)` → 前端 apply → Go 側重建托盤選單重新整理勾選態。Tray icon 主題化留作後續（設計文件每主題 menuIcon template 圖示屬資產層，本期不接）。
- **持久化方案**：**不寫 aquarium.json**。在 dataDir 下新增獨立小檔案 `theme.json`（僅存 `{"theme":"cn-shuimo"}`）。啟動時 load：有則用之、無則 reef-realistic；切換時寫該檔案（這是唯一允許的主題持久化寫路徑）。迴歸測試：切換前後對比 aquarium.json 位元組——主題檔案存在與否都不在其中。RPC 方法新增 `Aquarium.SetTheme`，需跑 `mygo generate` 重生成 bindings.ts 並補 browser.go RPC 白名單（現狀 8 個方法：Get/Start/Pause/Resume/End/Feed/SetPreferences/Export/Import，browser.go:70-96 附近）。
- **瀏覽器模式降級**：`-web` 模式無托盤選單 → 主題切換入口不可用，預設 reef-realistic（不做 UI 入口，保持現狀）。

**Tech Stack:** Go 1.27 + MyGo 0.6.0、`//go:embed web`、WKWebView。測試沿用：

```sh
CGO_ENABLED=0 go test ./...
node --test --test-concurrency=1 tests/*.test.mjs
```

**Spec:** 設計文件 `docs/主題視覺設計方案.html`（v1.0，2026-10-24，未提交）——9 套主題各 10 節規格（色彩/構圖/生物/控制元件/字型/動效/落地備註/token 對映）+ 第 04 章落地框架（theme 包結構、切換入口、工程注意、一致性規則）。**mood 圖是 AI 氛圍參考，不是成品資產**；落地以文字規格為準。

> 狀態：待核驗。
> 依據：設計文件全文（節號引用如 §04a/§10.4）+ 程式碼事實 [C1]–[C12]（見文末附錄）+ mygo v0.6.0 原始碼（menu.go / modules.go 行號）。

## 硬約束與紅線（使用者已拍板）

1. 主題 = 自包含 theme 包；9 套代號嚴格用 cn-* / cm-* / ct-*；寫實預設納入契約（reef-realistic）。
2. **硬紅線：切換絕不觸碰 aquarium.json**——「切換 100 次存檔位元組級不變」必須有 Go 測試鎖定。主題選擇持久化到 dataDir 獨立 `theme.json`，不進 aquarium.json。
3. 切換入口：選單欄下拉子選單（主入口）。設定面板主題頁（輔入口）留作後續，本期不做。
4. 相容系統「減少動態效果」：各主題動效密度按設計文件 0.2×/0.4×/0.5×… 分級落地；減動效下全部靜止停靠（設計文件每主題第 9 節給了完整靜態方案）。
5. 切換 = 重綁 CSS 變數 + 替換生物繪製引用；**切換不附帶慶祝性動效**（設計文件 04c：600ms 交叉淡入可留，減動效下硬切）。
6. 生物層 token 支援成長階段（coral/seaweed stages，與珊瑚成長玩法對齊），寫實主題同樣補齊。

## Token 化改動面清單（從現有程式碼 grep 實測，非推斷）

**CSS 層（web/style.css，共 20 行單行壓縮檔案）：**
- [C1] style.css:1 `:root` 現有 10 變數：--bg/--panel/--panel-light/--line/--text/--muted/--accent/--cyan/--coral/--radius + font-family——是天然的 token 落點。
- [C2] style.css:2-9 硬編碼 hex（必須改 var()）：`#0e1d24`(input 底)、`#12303b`/`#294148`/`#20343c`(app-shell 漸變/邊框)、`#194250`/`#102e3a`/`#0e252e`(reef-stage 徑向漸變)、`#a6dbce18`(light-ray)、`#9ebdbd`(reef-caption)、`#10242a`/`#1c3b41`(reef-bottom)、`#223b39`(mode-tabs active)、`#2b4549`(ring-track)、`#173c31`(primary-button 字色)、`#46777a`/`#213c40`(sound selected)、`#466258`/`#263f37`(sound-toggle)、`#3e6555`/`#1a332c`(notice)、`#1b3035`(visitor 卡)、`#10212a`(journal/review 底)、`#3b5759`(dialog 邊框)、`#31444a`/`#557176`(toast) 等約 30 處。

**Canvas 生物層（web/reef.js + web/creatures/）：**
- [C3] reef.js:75-78 礁石色 `#6d8179/#82948a`（4 處）。
- [C4] reef.js:234 沙地描邊 `#bd918c`；reef.js:241-245 水草 `#529a86/#70af91/#79b89b/#568d7c`。
- [C5] reef.js:261 水體 `#b4cfb114`；reef.js:267-270 沙地底 `#53675a/#718371/#899781/#566c5e`（含 25 粒沙點）。
- [C6] reef.js:272 滿級光暈 `#a8deba08`；reef.js:287-288 魚食 `rgba(226,194,148)/rgba(255,240,214)`。
- [C7] web/creatures/fish.js 34 處顏色字面量、shrimp.js 14 處、paint.js 4 處——生物造型顏色的主改造面。
- [C8] web/creatures/species.js:15-35 每物種 speed/length/zone 引數（13 物種）——主題漂移速度（driftSpeed token）應作用於此表的乘係數，不改物種行為語義。

**Go 側：**
- [C9] state.go:18-23 Preferences 結構（focusMinutes/breakMinutes/sound/volume/soundEnabled/chimeEnabled）——**不加 Theme 欄位**（紅線 2）；theme 選擇走獨立 theme.json。
- [C10] main.go:164,189-192 dataDir 解析（`-data-dir` flag → 預設 os.UserConfigDir）——theme.json 落在同一 dataDir。
- [C11] main.go:252-255 托盤選單構建處——插「主題 ▸」子選單的位置。
- [C12] icon.go（261 行）原生選單欄小魚繪製——本期不主題化，保持現狀。

## 檔案結構（Create / Modify）

**Create：**
- `web/themes/reef-realistic/theme.json` —— 從現有硬編碼抽取的 16 token 原值（視覺零迴歸的基準）。
- `web/themes/<9 個主題 id>/theme.json` —— 按設計文件各 §10 token 對映直填。
- `web/theme.js` —— token→CSS 變數寫入 + 全域 THEME 物件供 canvas 讀取 + 切換 apply()。
- `theme_store.go` —— theme.json 載入/儲存/列表/當前主題 RPC。
- `tests/theme_test.go` —— 位元組級不變回歸 + schema 校驗。

**Modify：**
- `web/style.css` —— 硬編碼 hex → var()（清單 [C2]）。
- `web/reef.js`、`web/creatures/*.js` —— 硬編碼顏色 → 讀 THEME token（[C3]-[C7]）。
- `main.go` —— 托盤選單加「主題 ▸」子選單（[C11]）。
- `browser.go` —— RPC 白名單加 GetTheme/SetTheme。
- `bindings.ts` —— `mygo generate` 重生成（注意：產物先寫到根檔案 `1`，核對 diff 後 `mv 1 bindings.ts`）。
- `state.go` —— Snapshot 增加 `Theme` 欄位（前端首屏渲染正確主題用；該欄位只從 theme.json 讀，不寫回 aquarium.json）。

## Task 分解

### T1：theme 包骨架 + reef-realistic 抽取
- [ ] 定義 Go Theme struct（meta/tokens/assets，按設計文件 §04a 結構）。
- [ ] 寫 `web/themes/reef-realistic/theme.json`：把 style.css:1 + reef.js/creatures 現有硬編碼值逐字填進 16 token（不改任何視覺）。
- [ ] theme.js：載入 reef-realistic token → 寫 `:root` 變數；把 style.css 的 var() 改造跑通（先 10 個既有變數，後逐處替換硬編碼）。
- [ ] 測試先行：node 測試鎖定「reef-realistic token 表 == 當前 :root 變數值快照」。
- 驗收：全量現有測試綠；瀏覽器模式截圖與改造前逐屏一致（人工複核 main/review/settings 三屏）。

### T2：切換入口 + 持久化 + 存檔迴歸測試
- [ ] theme_store.go：dataDir/theme.json 讀寫（結構 `{"theme":"<id>"}`，未知 id 回落 reef-realistic）。
- [ ] RPC：GetTheme / SetTheme（掛 Aquarium 服務側）；browser.go 白名單補兩個 case；bindings.ts 重生成。
- [ ] 前端：SetTheme 後 theme.js apply 新 token → 無閃爍重繪。
- [ ] main.go：托盤選單插「主題 ▸」子選單（radio 組 + 當前勾選）；切換後 Tray.SetMenu 重建重新整理勾選。
- [ ] **Go 測試（紅線）**：構造臨時 dataDir，記錄 aquarium.json 初始位元組 → SetTheme 切 N=100 次（含非法 id）→ 斷言 aquarium.json 位元組與初始完全一致；同時斷言 theme.json 內容是最後一次合法切換值。
- 驗收：瀏覽器模式冒煙（SetTheme via curl RPC）；桌面真機托盤子選單出現、勾選隨切換移動（使用者復現步驟）。

### T3：九套主題逐套落地（家族漸進）
- [ ] 先打樣一套（建議 cm-cel——設計文件自稱「與現有寫實氣質最接近」，token 對映在 §10 最完整）。
- [ ] 同族三連（cm-pop / cm-ink），再 cn 家族（cn-shuimo → cn-qinghua → cn-qinglu），再 ct 家族（ct-cream → ct-strawberry → ct-jelly）。
- [ ] 每套合入門禁（設計文件 §04d）：對比度數值照設計文件實測值核對、bubbleDensity/driftSpeed ≤1.0×、ambientAudio.defaultOn=false、schema 校驗透過。
- 驗收：每合一套，瀏覽器模式截圖三屏存證；「減少動態效果」開啟下該主題靜止。

### T4：生物引數化重繪
- [ ] 把 [C3]-[C7] 的 canvas 硬編碼改為讀 THEME token；造型演算法（輪廓/鰭/運動）不動。
- [ ] 生物層 token 支援成長階段：coral/seaweed 的 stages 陣列（設計文件 §04a：與珊瑚成長玩法對齊）。
- [ ] driftSpeed/bubbleDensity 以乘係數作用於 species.js 行為（[C8]）。
- 驗收：node 測試鎖定「換主題後同一物種 drawPortrait 顏色來自 token 而非常量」。

### T5：驗收頁 + 全量測試 + 復現步驟
- [ ] 瀏覽器模式加臨時 `?theme=<id>` 驗收引數（或 headless 截圖指令碼），逐主題出三屏截圖。
- [ ] 全量 Go + node 測試綠。
- [ ] 交付說明：改動檔案清單 + 行號 + 使用者復現步驟（瀏覽器模式看主題、桌面模式托盤切換、減動效驗證、100 次切換回歸）。

## Non-goals

- 設定面板主題頁（設計文件推薦輔入口）——後續迭代。
- 選單欄圖示（menuIcon）隨主題換圖——資產層後續。
- 深色模式 / 系統外觀跟隨——九套均淺色，meta.supportsDarkAppearance:false 顯式宣告（§04c）。
- 環境音（ambientAudio）實際接線——token 先落地，defaultOn 恆 false，音訊合成後續。
- mood 圖當資產使用——禁止；全部按文字規格引數化繪製。
- 切換過渡慶祝動效（印章/波紋/金線）——設計文件標註「需產品評審後排期」，本期只做 600ms 交叉淡入（減動效下硬切）。

## 附錄：程式碼事實清單

- [C1] style.css:1 `:root{--bg:#0c1820;--panel:#12232b;--panel-light:#172c32;--line:#254047;--text:#e3ede9;--muted:#8ca7a7;--accent:#a8deba;--cyan:#83c8cc;--coral:#e6b48b;--radius:20px;font-family:"PingFang SC",...}`
- [C2] style.css:2-9 硬編碼 hex 約 30 處（詳見上文清單）。
- [C3]-[C7] reef.js:75-78,234,241-245,261,267-270,272,287-288；fish.js 34 處、shrimp.js 14 處、paint.js 4 處顏色字面量。
- [C8] species.js:15-35 物種行為參數列。
- [C9] state.go:18-23 Preferences 欄位。
- [C10] main.go:164 flag、189-192 dataDir 解析。
- [C11] main.go:252-255 托盤選單 NewMenu。
- [C12] icon.go 原生托盤圖示（261 行），本期不動。
- mygo v0.6.0：MenuItem.Submenu（menu.go:105）、MenuItemRadio（menu.go:19）、SetChecked 同組互斥（menu.go:247-323）、Tray.SetMenu（modules.go:755）、Tray.SetIcon（modules.go:733）。
