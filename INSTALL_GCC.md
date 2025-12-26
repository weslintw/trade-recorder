# 打單紀錄器 - Windows 安裝 GCC 指南

## 為什麼需要 GCC？

SQLite Go 驅動（go-sqlite3）需要 CGO，而 CGO 需要 C 編譯器。

## 方案 A：安裝 TDM-GCC（推薦）

### 步驟 1：下載 TDM-GCC

訪問：https://jmeubank.github.io/tdm-gcc/download/

下載：**tdm64-gcc-10.3.0-2.exe**（約 50MB）

### 步驟 2：安裝

1. 執行下載的安裝程式
2. 選擇 "Create"（建立新安裝）
3. 選擇 "MinGW-w64/TDM64 (32-bit and 64-bit)"
4. 安裝路徑：保持預設 `C:\TDM-GCC-64`
5. 點擊 "Install" 並等待完成

### 步驟 3：驗證安裝

開啟新的 PowerShell 視窗：

```powershell
gcc --version
```

應該看到類似輸出：
```
gcc.exe (tdm64-1) 10.3.0
```

### 步驟 4：重新啟動專案

```powershell
.\start.ps1
```

---

## 方案 B：使用預編譯的 SQLite 驅動（無需 GCC）

如果不想安裝 GCC，可以使用純 Go 的 SQLite 驅動。

### 修改 go.mod

將：
```go
github.com/mattn/go-sqlite3 v1.14.22
```

改為：
```go
modernc.org/sqlite v1.28.0
```

### 修改 database/db.go

將：
```go
import _ "github.com/mattn/go-sqlite3"
```

改為：
```go
import _ "modernc.org/sqlite"
```

---

## 推薦方案

**方案 A（安裝 TDM-GCC）** 是最簡單的方案，只需：
1. 下載並安裝 TDM-GCC（5分鐘）
2. 重新啟動專案

安裝後，SQLite 將以原生速度運行，效能最佳。

