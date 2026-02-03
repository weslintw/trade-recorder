---
trigger: always_on
---

開發應遵守的規則 (Core Rules)
1. 圖片儲存規則 🖼️
必須使用 MinIO：所有上傳的圖片「絕對禁止」存放在本地檔案系統或直接存入資料庫，必須透過後端 internal/minio 模組上傳至 MinIO 物件儲存。
命名與目錄：圖片儲存應符合專案定義的 Bucket (trade-journal) 與路徑結構。
2. 資料庫與相容性 🗄️
SQLite 為核心：目前使用 SQLite 作為主要資料庫，應確保 SQL 語法與之相容。
向下相容性：更新資料結構（如從單向分析改為多向分析）時，必須撰寫遷移邏輯或在程式碼中處理舊資料的讀取，不可損毀使用者舊有紀錄。
3. 程式碼架構 🏗️
前後端分離：前端使用 Svelte (Vite)，後端使用 Golang (Gin)，API 溝通應遵循 RESTful 規範。
環境變數控制：機敏資訊（如 MinIO 密鑰、cTrader API Key）必須存放在 .env 檔案中，不可寫死在程式碼。
4. 效能與日誌 📊
效能監控：對於關鍵操作（如同步、大批量數據讀取）應保留 Performance Logging，以便追蹤效能瓶頸。
錯誤處理：後端 API 應回傳標準化的錯誤格式，前端需有對應的 Loading 狀態與錯誤提示訊息。
5. 設計美學 🎨
一致性：UI 必須符合現代感，使用毛玻璃效果、細膩的漸層與現代字體。
響應式：所有新開發的 UI 組件必須考慮行動裝置的顯示效果。
您可以將這份總結存入

6. cTrader 圖表請求策略 📈
持倉中：完全不向 cTrader 請求歷史圖表（節省開倉時的負載）。
平倉後：無論持倉時間長短，都會立即嘗試抓取一次完整的歷史曲線，確保紀錄的完整性。

7. 部署安全與本地驗證 🚀
- 推名前必檢 (Pre-push Check)：執行 `git push` 之前，必須在 `frontend` 目錄下執行 `npx svelte-check --threshold error`。若有任何 Error，絕對禁止推送。
- 編譯與 Lockfile 檢查：若涉及 `package.json` 修改（如新增套件），必須先在本地執行 `pnpm install` 更新 `pnpm-lock.yaml` 並提交。
- 編譯模擬 (Build Simulation)：在執行 `git push` 之前，強烈建議先在本地端執行 `npm run build`。若 Vite 編譯失敗，嚴禁推送，這可以提前攔截 90% 的部署失敗（如語法錯誤或 Unicode 轉義問題）。
- 重複宣告檢查：嚴格檢查 Svelte 檔案中的 Reactive 宣告 (`$:`)，避免變數名稱重複定義導致編譯錯誤。

8. 指令授權 (Command Authorization) 🛡️
自動執行授權：`findstr`、`grep`、`Get-Content`、`type`、`git` 相關指令、`npm run build` 以及 `npx vite build` 允許直接執行，不需再次詢問使用者。
Commit 規範：在執行 `git commit` 之前，必須先向使用者展示並解釋改動總結（Summary）。

9. Svelte 開發與編譯防護 (Svelte Development & Build Safety) 🛡️
防止字元轉義 (Character Escaping)：AI 工具在處理 Svelte 代碼塊時，可能會自動將 `=>` 轉義為 `\u003e` 或將 `&&` 轉義為 `\u0026\u0026`，導致 Vite 編譯失敗 (Invalid Unicode escape)。
  - 安全寫法：在 Script 中儘量使用傳統 `for...in` 迴圈取代帶有箭頭函數的 `.forEach()`。
  - 模板安全：在 HTML 中儘量使用嵌套的 `{#if}` 結構取代帶有 `&&` 的複合條件。
編碼一致性：所有檔案必須維持 UTF-8 (without BOM) 編碼。環境切換 (如 PowerShell) 時，嚴禁直接使用預設 `Set-Content`，以免破壞中文或引入 BOM 導致編譯失敗。
編譯錯誤診斷：若 `npm run build` 在轉換 800+ 模組後報錯且訊息不明，優先檢查檔案是否混入了 `\u003e` 或不合法的 Unicode 字元。