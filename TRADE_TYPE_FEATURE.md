# 交易類型功能 - 有進單 vs 純觀察

**日期**: 2025-12-26  
**功能**: 支援區分實際交易記錄和純觀察記錄

---

## 📋 功能概述

系統現在支援兩種記錄類型：

### 1. 💰 **有進單** (actual)
- 實際執行的交易
- 需要記錄：價格、手數、盈虧等完整交易資訊
- 用於計算績效統計

### 2. 👁️ **沒進單（純觀察）** (observation)
- 純粹的市場觀察記錄
- 不需要記錄：價格、手數、盈虧（自動隱藏這些欄位）
- 只記錄市場分析、策略想法等
- 用於復盤學習和策略驗證

---

## 🔄 變更內容

### 資料庫變更

#### 新增欄位：`trade_type`
```sql
ALTER TABLE trades ADD COLUMN trade_type VARCHAR(20) DEFAULT 'actual';
```

**值**：
- `'actual'` - 實際交易（預設值）
- `'observation'` - 純觀察記錄

### 後端變更

#### 1. **Model 更新** (`backend/internal/models/trade.go`)

**Trade 結構**：
```go
type Trade struct {
    ID          int64      `json:"id"`
    TradeType   string     `json:"trade_type"` // "actual" 或 "observation"
    Symbol      string     `json:"symbol"`
    Side        string     `json:"side"`
    EntryPrice  float64    `json:"entry_price"`
    // ... 其他欄位
}
```

**TradeCreate 結構**：
```go
type TradeCreate struct {
    TradeType   string    `json:"trade_type" binding:"required,oneof=actual observation"`
    Symbol      string    `json:"symbol" binding:"required"`
    Side        string    `json:"side" binding:"required,oneof=long short"`
    EntryPrice  *float64  `json:"entry_price"` // observation 時可為 null
    LotSize     *float64  `json:"lot_size"`    // observation 時可為 null
    // ... 其他欄位
}
```

#### 2. **Handler 更新** (`backend/internal/handlers/trade.go`)

- ✅ 所有 SQL 查詢已更新以包含 `trade_type` 欄位
- ✅ `GetTrades` - 查詢時包含交易類型
- ✅ `GetTrade` - 獲取單筆記錄時包含交易類型
- ✅ `CreateTrade` - 插入時包含交易類型
- ✅ `UpdateTrade` - 更新時包含交易類型

### 前端變更

#### 1. **TradeForm.svelte** - 新增交易類型選擇

**新增的 UI 元素**：
```svelte
<div class="trade-type-section">
  <label class="trade-type-label">紀錄類型</label>
  <div class="trade-type-options">
    <!-- 有進單選項 -->
    <label class="radio-option" class:active={formData.trade_type === 'actual'}>
      <input type="radio" bind:group={formData.trade_type} value="actual" />
      <span class="radio-label">
        <span class="radio-icon">💰</span>
        <span class="radio-text">
          <strong>有進單</strong>
          <small>實際交易記錄</small>
        </span>
      </span>
    </label>
    
    <!-- 純觀察選項 -->
    <label class="radio-option" class:active={formData.trade_type === 'observation'}>
      <input type="radio" bind:group={formData.trade_type} value="observation" />
      <span class="radio-label">
        <span class="radio-icon">👁️</span>
        <span class="radio-text">
          <strong>沒進單</strong>
          <small>純觀察記錄</small>
        </span>
      </span>
    </label>
  </div>
</div>
```

**動態欄位顯示**：
```svelte
{#if isActualTrade}
  <!-- 只在「有進單」時顯示 -->
  <div class="form-group">
    <label for="lot_size">手數</label>
    <input type="number" bind:value={formData.lot_size} required />
  </div>
  
  <div class="form-group">
    <label for="entry_price">進場價格</label>
    <input type="number" bind:value={formData.entry_price} required />
  </div>
  
  <!-- 盈虧相關欄位... -->
{/if}
```

**提交邏輯**：
```javascript
async function handleSubmit() {
  const submitData = { ...formData };
  
  if (isActualTrade) {
    // 有進單：解析數值欄位
    submitData.entry_price = parseFloat(formData.entry_price);
    submitData.lot_size = parseFloat(formData.lot_size);
    // ...
  } else {
    // 純觀察：這些欄位設為 null
    submitData.entry_price = null;
    submitData.exit_price = null;
    submitData.lot_size = null;
    submitData.pnl = null;
    submitData.pnl_points = null;
  }
  
  // 提交...
}
```

#### 2. **TradeList.svelte** - 顯示更新

- ✅ 「進場理由」改為「進場分析」
- 未來可添加交易類型徽章顯示

---

## 🎨 UI/UX 設計

### 交易類型選擇器樣式

1. **卡片式設計**
   - 兩個並排的選項卡
   - 清晰的圖示和說明文字

2. **互動效果**
   - Hover 效果：卡片上升、邊框變色、陰影
   - Active 效果：邊框高亮、背景變色、外圍光暈

3. **視覺層次**
   - 圖示：大而醒目（2rem）
   - 主標題：粗體、清晰
   - 副標題：小字、灰色說明

### 響應式行為

```css
.trade-type-options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}
```

- 桌面：兩個選項並排
- 移動設備：自動堆疊

---

## 📊 使用場景

### 場景 1：實際交易記錄（有進單）

**使用時機**：
- 已經執行的真實交易
- 需要計算盈虧和績效
- 需要完整的交易數據

**必填欄位**：
- ✅ 交易品種
- ✅ 方向（做多/做空）
- ✅ 手數
- ✅ 進場價格
- ✅ 進場時間
- ⚪ 平倉價格（可選）
- ⚪ 盈虧（可選）

**範例**：
```
類型：💰 有進單
品種：XAUUSD
方向：做多
手數：1.5
進場價格：2650.50
進場時間：2025-12-26 11:30
進場分析：突破前高，RSI超過70...
```

### 場景 2：純觀察記錄（沒進單）

**使用時機**：
- 看到機會但沒有進場
- 市場分析和策略驗證
- 學習和復盤用途
- 記錄錯過的機會

**必填欄位**：
- ✅ 交易品種
- ✅ 方向（看多/看空）
- ✅ 觀察時間
- ⚪ 分析筆記（推薦填寫）

**隱藏欄位**（自動設為 null）：
- ❌ 手數
- ❌ 進場價格
- ❌ 平倉價格
- ❌ 盈虧金額
- ❌ 盈虧點數

**範例**：
```
類型：👁️ 沒進單
品種：XAUUSD
方向：看多
觀察時間：2025-12-26 11:30
進場分析：
  突破前高但成交量不足，沒有進場。
  如果下次出現類似形態且成交量配合，可以考慮進場。
  [附上圖表截圖]
```

---

## 🔍 未來統計功能考量

### 有進單記錄
- 計算真實勝率
- 計算真實盈虧
- 生成淨值曲線
- 統計交易習慣

### 純觀察記錄
- 統計觀察到的機會數量
- 分析「為什麼沒進場」的原因
- 驗證策略的「紙上模擬」準確度
- 學習曲線追蹤

### 交叉分析
- 比較「有進單」vs「純觀察」的市場環境差異
- 分析是否錯過了好機會
- 改進進場決策流程

---

## 🧪 測試步驟

### 1. 測試「有進單」
1. 訪問 `http://localhost:5173`
2. 點擊「新增交易」
3. 選擇「💰 有進單」
4. 應該看到：
   - ✅ 手數欄位可見
   - ✅ 進場價格、平倉價格可見
   - ✅ 盈虧欄位可見
5. 填寫完整資料並儲存
6. 確認資料正確儲存

### 2. 測試「沒進單」
1. 點擊「新增交易」
2. 選擇「👁️ 沒進單」
3. 應該看到：
   - ✅ 手數欄位消失
   - ✅ 價格欄位消失
   - ✅ 盈虧欄位消失
   - ✅ 只保留品種、方向、時間、分析欄位
4. 填寫基本資訊和分析筆記
5. 儲存並確認

### 3. 測試切換
1. 在表單中選擇「有進單」
2. 看到所有欄位
3. 切換到「沒進單」
4. 確認欄位即時消失
5. 切換回「有進單」
6. 確認欄位即時出現

### 4. 測試編輯現有記錄
1. 編輯一筆舊記錄（預設 trade_type='actual'）
2. 確認顯示為「有進單」
3. 所有欄位正常顯示
4. 可以切換到「沒進單」並儲存

---

## 📝 額外更新

### 文字變更
- ✅ 「進場理由」→「進場分析」（更專業的術語）
- ✅ Placeholder 更新為更清晰的說明文字

---

## 🎯 總結

這次更新讓系統可以同時處理：
1. **實際交易記錄** - 用於績效追蹤和統計
2. **觀察記錄** - 用於學習、復盤和策略驗證

通過動態表單和清晰的UI設計，讓用戶可以輕鬆選擇合適的記錄類型，提升系統的實用性和靈活性！

---

**所有功能已完成並測試通過！** 🎉✨

