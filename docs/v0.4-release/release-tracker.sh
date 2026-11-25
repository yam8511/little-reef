#!/bin/sh
# release-tracker.sh — v0.4 釋出周表（使用者自行執行，不自動執行）
# 口徑：GitHub 公開計數（releases 資產下載量 / issues 標籤計數 / star 數），無任何遙測。
# 基線（2026-10-24）：DMG 合計 5、update json 6、star 2、issue 0。
# 用法：sh docs/v0.4-release/release-tracker.sh
# 輸出：追加列印一行周表（日期 | DMG 下載 | update json 拉取 | star | issue 總數 | 分標籤計數）。

REPO="echoVic/little-reef"
API="https://api.github.com/repos/$REPO"

echo "== $(date +%F) =="

# Release assets: DMG 與 update-darwin-arm64.json 的 download_count。
curl -sf "$API/releases" | python3 - <<'PY'
import json, sys
try:
    releases = json.load(sys.stdin)
except Exception as e:
    print("releases 拉取失敗：", e); sys.exit(1)
dmg, updater = 0, 0
for r in releases:
    for a in r.get("assets", []):
        n = a.get("name", "")
        c = a.get("download_count", 0)
        if n.endswith(".dmg"):
            dmg += c
        if "update" in n and n.endswith(".json"):
            updater += c
print(f"DMG 下載合計: {dmg}")
print(f"update json 拉取合計: {updater}")
PY

# Star 數。
curl -sf "$API" | python3 -c "import json,sys; d=json.load(sys.stdin); print('star:', d.get('stargazers_count'))"

# Issues 按標籤計數。
curl -sf "$API/issues?state=all&per_page=100" | python3 - <<'PY'
import json, sys
try:
    issues = json.load(sys.stdin)
except Exception as e:
    print("issues 拉取失敗：", e); sys.exit(1)
counts = {}
real = 0
for i in issues:
    if "pull_request" in i:  # PR 也會出現在 issues 介面裡，排除
        continue
    real += 1
    for l in i.get("labels", []):
        name = l.get("name") if isinstance(l, dict) else l
        counts[name] = counts.get(name, 0) + 1
print("issue 總數:", real)
for tag in ["bug", "sound", "growth", "stats", "backup", "other"]:
    print(f"  {tag}: {counts.get(tag, 0)}")
PY
