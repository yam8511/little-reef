# v0.4 釋出物料（docs/v0.4-release/）

本目錄是 v0.4（回顧面板 + 存檔備份）的釋出物料定稿落盤。文案來源：飛書 docx/OQ8Qd9SaHokTnUxW7zCcSSIbncf（revision 5，2026-10-24 定稿）。

## 檔案清單

| 檔案 | 用途 | 釋出動作（均待使用者確認） |
|-|-|-|
| `website-copy.md` | 官網 reef.echovic.com 中英對照全文（含新增 005 屏） | 官網倉庫位置待使用者告知，本倉庫不部署 |
| `channel-posts.md` | HN / Product Hunt / V2EX / 即刻 / 少數派 / 小紅書 六渠道成稿 | 按 Q7 節奏逐渠道釋出，逐條等確認 |
| `issue-template.yml` | GitHub Issue 回饋模板 YAML（原文保真） | 已同步到 `.github/ISSUE_TEMPLATE/feedback.yml`，隨程式碼進倉庫即生效 |
| `release-notes.md` | CHANGELOG 0.4.0 條目 + GitHub Release 正文 + 六標籤清單 + 護欄審查記錄 | CHANGELOG 條目本輪寫入倉庫；Release tag 與正文釋出待確認 |
| `video-storyboard.md` | 30–45s 演示影片 6 鏡分鏡定稿 + 真機錄製指引 | 初剪版見倉庫外交付物；公開發布待確認 |
| `release-tracker.sh` | 周表指令碼（下載量 / update json 拉取量 / star / issue 標籤計數） | **使用者自行執行，不自動執行** |

## 釋出邊界

- 本目錄只落物料；任何對外動作（打 tag、發 Release、官網部署、發帖、投稿、影片釋出）**逐項等使用者確認**。
- GitHub 六標籤（bug/sound/growth/stats/backup/other，顏色見 release-notes.md）需倉庫管理權限，手動建立指引見 release-notes.md。
- 周表口徑：每週一次拉 GitHub 公開計數（releases 資產 download_count、issues 標籤計數、star 數），基線（2026-10-24）：DMG 合計 5、update json 6、star 2、issue 0。不入庫任何遙測性質資料。
- 官網部署需要官網倉庫位置（使用者待提供）。
