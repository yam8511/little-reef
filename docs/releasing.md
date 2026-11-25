# Windows 釋出流程

發行平臺為 Windows 10/11（amd64 與 arm64），釋出到 [yam8511/little-reef](https://github.com/yam8511/little-reef) 的 Releases。版本號維護在 `mygo.json`，更新說明來自 `CHANGELOG.md` 對應版本節（標題須為 `## <版本>`）。

## 自動更新的原理

`mygo.json` 的 `updates.github` 指向本倉庫，`updates.publicKey` 是更新簽名的公鑰。已安裝的應用每 4 小時讀取
`https://github.com/yam8511/little-reef/releases/latest/download/update-windows-<架構>.json`，發現新版本就下載簽名過的安裝包或增量包，用內建公鑰驗證 Ed25519 簽名後替換自己。因此：

- 倉庫必須保持公開；只認 Releases 裡的附件，提交進倉庫的檔案不會被讀取。
- Release 必須設為 Latest，且附件齊全後才公開。
- 私鑰（`mygo-update.key`，位於 `%APPDATA%/mygo/update-keys`）不能進倉庫。丟失它，已安裝的應用就再也收不到更新；換鑰匙需要使用者手動重新安裝。

## 一次性設定

1. 在倉庫 Settings → Secrets and variables → Actions 新增 secret `MYGO_UPDATER_PRIVATE_KEY`，內容是 `%APPDATA%/mygo/update-keys/mygo-update.key` 的全文（或 `gh secret set MYGO_UPDATER_PRIVATE_KEY < mygo-update.key`）。
2. 確認倉庫 Settings → Actions → General 的 Workflow permissions 允許寫入（`release.yml` 已宣告 `contents: write`）。

## 釋出一個版本

1. 把 `mygo.json` 的 `version` 調高，例如 `0.5.0`；把 `CHANGELOG.md` 的「## 未釋出」改成「## 0.5.0」。
2. 本地確認 `CGO_ENABLED=0 go vet ./...` 與 `CGO_ENABLED=0 go test ./...` 透過，提交併推送到 `main`。
3. 打標籤並推送：`git tag v0.5.0 && git push origin v0.5.0`。
4. `.github/workflows/release.yml` 會先檢查標籤與 `mygo.json` 版本一致、CHANGELOG 有該版本段落、私鑰 secret 已設定，再建立草稿 Release；接著在 Ubuntu 上裝 NSIS，分別為 amd64 和 arm64 執行 `go tool mygo build -platform windows/<架構> -upload`，把安裝包、更新包、增量包和 `update-windows-<架構>.json` 上傳到草稿。
5. 到 Releases 檢查草稿附件齊全，再公開並設為 Latest。公開後下載 `releases/latest/download/update-windows-amd64.json` 核對版本號，並用上一版本實際安裝一次，確認「檢查更新…」能升級。

失敗時在 Actions 頁面用「Re-run failed jobs」重跑，會上傳到同一個草稿。

## 本地手動構建（備用）

需要 [NSIS](https://nsis.sourceforge.io) 才會產出安裝包：

```sh
export MYGO_UPDATER_PRIVATE_KEY="$(cat "$APPDATA/mygo/update-keys/mygo-update.key")"
export GH_TOKEN=<有 repo 權限的 token>
go tool mygo build -platform windows/amd64,windows/arm64 -upload .
```

沒有安裝程式碼簽名證書時，Windows SmartScreen 會在安裝時提示未知釋出者；這不影響自動更新。
