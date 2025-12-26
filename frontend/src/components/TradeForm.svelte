<script>
  import { navigate } from 'svelte-routing';
  import { tradesAPI } from '../lib/api';
  import RichTextEditor from './RichTextEditor.svelte';

  export let id = null;

  let formData = {
    trade_type: 'actual', // actual=有進單, observation=純觀察
    symbol: 'XAUUSD',
    side: 'long',
    entry_price: '',
    exit_price: '',
    lot_size: '',
    pnl: '',
    pnl_points: '',
    notes: '',
    entry_reason: '',
    exit_reason: '',
    entry_strategy: '', // expert=達人, elite=菁英, legend=傳奇
    entry_strategy_image: '', // 進場種類圖片
    entry_signals: [], // 達人訊號（多選）
    entry_checklist: {}, // 菁英/傳奇檢查清單
    trend_analysis: { // 當前趨勢
      M1: { direction: '', image: '' },
      M5: { direction: '', image: '' },
      M15: { direction: '', image: '' },
      M30: { direction: '', image: '' },
      H1: { direction: '', image: '' },
      H4: { direction: '', image: '' },
      D1: { direction: '', image: '' }
    },
    market_session: '', // asian=亞盤, european=歐盤, us=美盤
    timezone_offset: new Date().getTimezoneOffset() / -60, // 預設系統時區
    entry_time: new Date().toISOString().slice(0, 16),
    exit_time: '',
    tags: []
  };

  // 響應式：根據交易類型判斷是否顯示交易相關欄位
  $: isActualTrade = formData.trade_type === 'actual';
  
  // 達人訊號選項 - 根據做多/做空顯示不同訊號
  const expertSignalsLong = [
    '向下蘇美',
    '起漲靠山',
    '雙柱',
    '倚天',
    '攻城池上'
  ];
  
  const expertSignalsShort = [
    '起跌靠山',
    '君臨城下',
    '雙塔',
    '向上蘇美',
    '雷霆'
  ];
  
  // 根據方向選擇對應的訊號列表
  $: expertSignals = formData.side === 'long' ? expertSignalsLong : expertSignalsShort;
  
  // 菁英/傳奇檢查清單
  const eliteChecklist = [
    { id: 'trend_line', label: '破趨勢線了嗎?' },
    { id: 'price_level', label: '破價位了嗎?' },
    { id: 'impulse_wave', label: '有驅動浪了嗎?' },
    { id: 'high_low', label: '不過高低了嗎?' },
    { id: 'sentiment', label: '情緒轉換了嗎?' }
  ];

  // 時區選項 (UTC-12 到 UTC+14)
  const timezoneOptions = [];
  for (let i = -12; i <= 14; i++) {
    timezoneOptions.push({
      value: i,
      label: i >= 0 ? `UTC+${i}` : `UTC${i}`
    });
  }

  // 市場時段判別函數
  function determineMarketSession(entryTime, timezoneOffset) {
    if (!entryTime) return '';
    
    const date = new Date(entryTime);
    const month = date.getMonth() + 1; // 1-12
    
    // 判斷是否為夏令時間（3月-11月）
    const isDST = month >= 3 && month <= 11;
    
    // 轉換為 UTC 時間
    const utcHour = date.getUTCHours();
    const utcMinute = date.getUTCMinutes();
    
    // 轉換為 GMT+8（台北時間）用於判斷
    const gmt8Hour = (utcHour + 8 + 24) % 24;
    const timeInMinutes = gmt8Hour * 60 + utcMinute;
    
    // 時間範圍定義（以 GMT+8 為基準，單位：分鐘）
    // 亞盤（東京）：08:00 - 15:00（全年不變）
    const asianStart = 8 * 60;   // 08:00
    const asianEnd = 15 * 60;    // 15:00
    
    // 歐盤（倫敦）
    let europeanStart, europeanEnd;
    if (isDST) {
      // 夏令時間：15:00 - 23:00
      europeanStart = 15 * 60;   // 15:00
      europeanEnd = 23 * 60;     // 23:00
    } else {
      // 冬令時間：16:00 - 00:00
      europeanStart = 16 * 60;   // 16:00
      europeanEnd = 24 * 60;     // 00:00 (midnight)
    }
    
    // 美盤（紐約）
    let usStart, usEnd;
    if (isDST) {
      // 夏令時間：20:00 - 04:00（跨日）
      usStart = 20 * 60;         // 20:00
      usEnd = 4 * 60;            // 04:00
    } else {
      // 冬令時間：21:00 - 05:00（跨日）
      usStart = 21 * 60;         // 21:00
      usEnd = 5 * 60;            // 05:00
    }
    
    // 判斷市場時段
    // 亞盤：08:00 - 15:00
    if (timeInMinutes >= asianStart && timeInMinutes < asianEnd) {
      return 'asian';
    }
    
    // 歐盤
    if (isDST) {
      // 夏令時間：15:00 - 23:00
      if (timeInMinutes >= europeanStart && timeInMinutes < europeanEnd) {
        return 'european';
      }
    } else {
      // 冬令時間：16:00 - 00:00（處理跨日）
      if (timeInMinutes >= europeanStart || timeInMinutes < 0) {
        return 'european';
      }
    }
    
    // 美盤（處理跨日情況）
    if (timeInMinutes >= usStart || timeInMinutes < usEnd) {
      return 'us';
    }
    
    // 其他時間（間隙）預設為亞盤
    return 'asian';
  }

  // 監聽進場時間和時區變化，自動更新市場時段
  $: {
    if (formData.entry_time && formData.timezone_offset !== null) {
      formData.market_session = determineMarketSession(formData.entry_time, formData.timezone_offset);
    }
  }

  // 市場時段顯示名稱
  const marketSessionNames = {
    asian: '亞盤',
    european: '歐盤',
    us: '美盤'
  };

  // 取得市場時段時間範圍文字
  function getMarketSessionTime(session) {
    if (!session || !formData.entry_time) return '';
    
    const date = new Date(formData.entry_time);
    const month = date.getMonth() + 1;
    const isDST = month >= 3 && month <= 11;
    
    switch(session) {
      case 'asian':
        return '08:00 - 15:00';
      case 'european':
        return isDST ? '15:00 - 23:00' : '16:00 - 00:00';
      case 'us':
        return isDST ? '20:00 - 04:00' : '21:00 - 05:00';
      default:
        return '';
    }
  }

  // 取得夏/冬令時間標示
  function getSeasonLabel() {
    if (!formData.entry_time) return '';
    const date = new Date(formData.entry_time);
    const month = date.getMonth() + 1;
    const isDST = month >= 3 && month <= 11;
    return isDST ? '夏令時間' : '冬令時間';
  }

  let tagInput = '';
  let saving = false;

  // 富文本編輯器引用
  let entryReasonEditor;
  let exitReasonEditor;
  let notesEditor;

  // 圖片放大查看
  let enlargedImage = null;
  let enlargedImageTitle = '';

  const symbols = ['XAUUSD', 'NAS100', 'US30', 'EURUSD', 'GBPUSD', 'USDJPY'];

  if (id) {
    loadTrade();
  }

  async function loadTrade() {
    try {
      const response = await tradesAPI.getOne(id);
      formData = {
        ...response.data,
        entry_reason: response.data.entry_reason || '',
        exit_reason: response.data.exit_reason || '',
        notes: response.data.notes || '',
        entry_strategy: response.data.entry_strategy || '',
        entry_strategy_image: response.data.entry_strategy_image || '',
        entry_signals: response.data.entry_signals ? JSON.parse(response.data.entry_signals) : [],
        entry_checklist: response.data.entry_checklist ? JSON.parse(response.data.entry_checklist) : {},
        trend_analysis: response.data.trend_analysis ? JSON.parse(response.data.trend_analysis) : {
          M1: { direction: '', image: '' },
          M5: { direction: '', image: '' },
          M15: { direction: '', image: '' },
          M30: { direction: '', image: '' },
          H1: { direction: '', image: '' },
          H4: { direction: '', image: '' },
          D1: { direction: '', image: '' }
        },
        market_session: response.data.market_session || '',
        timezone_offset: response.data.timezone_offset !== null ? response.data.timezone_offset : new Date().getTimezoneOffset() / -60,
        entry_time: new Date(response.data.entry_time).toISOString().slice(0, 16),
        exit_time: response.data.exit_time ? new Date(response.data.exit_time).toISOString().slice(0, 16) : '',
        tags: response.data.tags?.map(t => t.name) || [],
      };
    } catch (error) {
      console.error('載入交易失敗:', error);
      alert('載入交易資料失敗');
    }
  }

  function addTag() {
    if (tagInput.trim() && !formData.tags.includes(tagInput.trim())) {
      formData.tags = [...formData.tags, tagInput.trim()];
      tagInput = '';
    }
  }

  function removeTag(tag) {
    formData.tags = formData.tags.filter(t => t !== tag);
  }

  // 監聽方向變化，清空已選訊號（避免做多訊號和做空訊號混淆）
  let previousSide = formData.side;
  $: {
    if (formData.side !== previousSide && formData.entry_strategy === 'expert') {
      formData.entry_signals = [];
      previousSide = formData.side;
    }
  }

  // 處理趨勢圖片貼上
  function handleTrendImagePaste(event, timeframe) {
    const items = (event.clipboardData || event.originalEvent.clipboardData).items;
    
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = item.getAsFile();
        const reader = new FileReader();
        
        reader.onload = (e) => {
          formData.trend_analysis[timeframe].image = e.target.result;
          formData = formData; // 觸發更新
        };
        
        reader.readAsDataURL(file);
        break;
      }
    }
  }

  // 移除趨勢圖片
  function removeTrendImage(timeframe) {
    formData.trend_analysis[timeframe].image = '';
    formData = formData;
  }

  // 處理進場種類圖片貼上
  function handleStrategyImagePaste(event) {
    const items = (event.clipboardData || event.originalEvent.clipboardData).items;
    
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = item.getAsFile();
        const reader = new FileReader();
        
        reader.onload = (e) => {
          formData.entry_strategy_image = e.target.result;
          formData = formData;
        };
        
        reader.readAsDataURL(file);
        break;
      }
    }
  }

  // 移除進場種類圖片
  function removeStrategyImage() {
    formData.entry_strategy_image = '';
    formData = formData;
  }

  // 放大查看圖片
  function enlargeImage(imageSrc, title) {
    enlargedImage = imageSrc;
    enlargedImageTitle = title;
  }

  // 關閉放大圖片
  function closeEnlargedImage() {
    enlargedImage = null;
    enlargedImageTitle = '';
  }

  async function handleSubmit() {
    try {
      saving = true;

      // 從富文本編輯器取得內容
      const submitData = {
        ...formData,
        entry_reason: entryReasonEditor ? entryReasonEditor.getContent() : formData.entry_reason,
        exit_reason: exitReasonEditor ? exitReasonEditor.getContent() : formData.exit_reason,
        notes: notesEditor ? notesEditor.getContent() : formData.notes,
        entry_signals: JSON.stringify(formData.entry_signals),
        entry_checklist: JSON.stringify(formData.entry_checklist),
        trend_analysis: JSON.stringify(formData.trend_analysis),
        entry_strategy_image: formData.entry_strategy_image,
        entry_time: new Date(formData.entry_time).toISOString(),
        exit_time: formData.exit_time ? new Date(formData.exit_time).toISOString() : null
      };

      // 如果是實際交易，添加交易相關欄位
      if (isActualTrade) {
        submitData.entry_price = formData.entry_price ? parseFloat(formData.entry_price) : null;
        submitData.exit_price = formData.exit_price ? parseFloat(formData.exit_price) : null;
        submitData.lot_size = formData.lot_size ? parseFloat(formData.lot_size) : null;
        submitData.pnl = formData.pnl ? parseFloat(formData.pnl) : null;
        submitData.pnl_points = formData.pnl_points ? parseFloat(formData.pnl_points) : null;
      } else {
        // 純觀察記錄，這些欄位設為 null
        submitData.entry_price = null;
        submitData.exit_price = null;
        submitData.lot_size = null;
        submitData.pnl = null;
        submitData.pnl_points = null;
      }

      if (id) {
        await tradesAPI.update(id, submitData);
        alert('交易紀錄更新成功！');
      } else {
        await tradesAPI.create(submitData);
        alert('交易紀錄建立成功！');
      }

      navigate('/');
    } catch (error) {
      console.error('儲存失敗:', error);
      alert('儲存失敗：' + (error.response?.data?.error || error.message));
    } finally {
      saving = false;
    }
  }
</script>

<div class="card">
  <h2>{id ? '編輯' : '新增'}交易紀錄</h2>

  <form on:submit|preventDefault={handleSubmit}>
    <!-- 交易類型選擇 -->
    <div class="form-group trade-type-section">
      <label class="trade-type-label">紀錄類型</label>
      <div class="trade-type-options">
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

    <!-- 基本資訊 -->
    <div class="form-row">
      <div class="form-group">
        <label for="symbol">交易品種</label>
        <select id="symbol" class="form-control" bind:value={formData.symbol} required>
          {#each symbols as symbol}
            <option value={symbol}>{symbol}</option>
          {/each}
        </select>
      </div>

      <div class="form-group">
        <label for="side">做多或做空</label>
        <select id="side" class="form-control" bind:value={formData.side} required>
          <option value="long">做多 (Long)</option>
          <option value="short">做空 (Short)</option>
        </select>
      </div>

      {#if isActualTrade}
      <div class="form-group">
        <label for="lot_size">手數</label>
        <input type="number" step="0.01" id="lot_size" class="form-control" 
               bind:value={formData.lot_size} required />
      </div>
      {/if}
    </div>

    {#if isActualTrade}
    <div class="form-row">
      <div class="form-group">
        <label for="entry_price">進場價格</label>
        <input type="number" step="0.00001" id="entry_price" class="form-control" 
               bind:value={formData.entry_price} required />
      </div>

      <div class="form-group">
        <label for="exit_price">平倉價格</label>
        <input type="number" step="0.00001" id="exit_price" class="form-control" 
               bind:value={formData.exit_price} />
      </div>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label for="pnl">盈虧金額</label>
        <input type="number" step="0.01" id="pnl" class="form-control" 
               bind:value={formData.pnl} />
      </div>

      <div class="form-group">
        <label for="pnl_points">盈虧點數</label>
        <input type="number" step="0.1" id="pnl_points" class="form-control" 
               bind:value={formData.pnl_points} />
      </div>
    </div>
    {/if}

    <div class="form-row">
      <div class="form-group">
        <label for="entry_time">開倉時間</label>
        <input type="datetime-local" id="entry_time" class="form-control" 
               bind:value={formData.entry_time} required />
      </div>

      <div class="form-group">
        <label for="timezone">UTC</label>
        <select id="timezone" class="form-control" bind:value={formData.timezone_offset}>
          {#each timezoneOptions as tz}
            <option value={tz.value}>{tz.label}</option>
          {/each}
        </select>
      </div>

      {#if formData.market_session}
        <div class="form-group">
          <label>市場時段</label>
          <div class="market-session-display">
            <div class="market-session-info">
              <span class="market-session-badge {formData.market_session}">
                {marketSessionNames[formData.market_session]}
              </span>
              <div class="session-details">
                <span class="session-time">{getMarketSessionTime(formData.market_session)}</span>
                <span class="session-season">{getSeasonLabel()}</span>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <div class="form-row">
      <div class="form-group">
        <label for="exit_time">平倉時間</label>
        <input type="datetime-local" id="exit_time" class="form-control" 
               bind:value={formData.exit_time} />
      </div>
    </div>

    <div class="form-group">
      <label>📍 進場分析</label>
    </div>

    <!-- 進場種類選擇 -->
    <div 
      class="form-group entry-strategy-section"
      tabindex="0"
      on:paste={handleStrategyImagePaste}
      on:click={(e) => {
        // 如果點擊的不是 radio 按鈕或圖片相關元素，聚焦以便貼上
        if (!e.target.closest('.strategy-options') && !e.target.closest('.strategy-image-preview')) {
          e.currentTarget.focus();
        }
      }}
    >
      <label class="strategy-label">🎯 進場種類</label>
      <div class="strategy-options">
        <label class="strategy-option" class:active={formData.entry_strategy === 'expert'}>
          <input type="radio" bind:group={formData.entry_strategy} value="expert" />
          <span class="strategy-name">達人</span>
        </label>
        <label class="strategy-option" class:active={formData.entry_strategy === 'elite'}>
          <input type="radio" bind:group={formData.entry_strategy} value="elite" />
          <span class="strategy-name">菁英</span>
        </label>
        <label class="strategy-option" class:active={formData.entry_strategy === 'legend'}>
          <input type="radio" bind:group={formData.entry_strategy} value="legend" />
          <span class="strategy-name">傳奇</span>
        </label>
      </div>

      <!-- 進場種類圖片預覽 -->
      {#if formData.entry_strategy_image}
        <div class="strategy-image-preview">
          <img 
            src={formData.entry_strategy_image} 
            alt="進場種類圖"
            on:click={(e) => {
              e.stopPropagation();
              enlargeImage(formData.entry_strategy_image, '進場種類圖');
            }}
            style="cursor: zoom-in;"
          />
          <button 
            type="button" 
            class="remove-strategy-image"
            on:click={(e) => {
              e.stopPropagation();
              removeStrategyImage();
            }}
            title="移除圖片"
          >
            ×
          </button>
        </div>
      {/if}

      <!-- 達人訊號（多選） -->
      {#if formData.entry_strategy === 'expert'}
        <div class="signals-section">
          <label class="signals-label">選擇訊號（可多選）：</label>
          <div class="signals-grid">
            {#each expertSignals as signal}
              <label class="checkbox-item">
                <input 
                  type="checkbox" 
                  value={signal}
                  checked={formData.entry_signals.includes(signal)}
                  on:change={(e) => {
                    if (e.target.checked) {
                      formData.entry_signals = [...formData.entry_signals, signal];
                    } else {
                      formData.entry_signals = formData.entry_signals.filter(s => s !== signal);
                    }
                  }}
                />
                <span class="checkbox-label">{signal}</span>
              </label>
            {/each}
          </div>
        </div>
      {/if}

      <!-- 菁英/傳奇檢查清單 -->
      {#if formData.entry_strategy === 'elite' || formData.entry_strategy === 'legend'}
        <div class="checklist-section">
          <label class="checklist-label">檢查清單：</label>
          <div class="checklist-items">
            {#each eliteChecklist as item}
              <label class="checkbox-item">
                <input 
                  type="checkbox" 
                  checked={formData.entry_checklist[item.id] || false}
                  on:change={(e) => {
                    formData.entry_checklist = {
                      ...formData.entry_checklist,
                      [item.id]: e.target.checked
                    };
                  }}
                />
                <span class="checkbox-label">{item.label}</span>
              </label>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <!-- 當前趨勢 -->
    <div class="form-group trend-analysis-section">
      <label class="trend-label">📊 當前趨勢</label>
      <div class="trend-grid">
        {#each ['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'] as timeframe}
          <div 
            class="trend-item"
            tabindex="0"
            on:paste={(e) => handleTrendImagePaste(e, timeframe)}
            on:click={(e) => {
              // 如果點擊的是 radio 按鈕區域，不要聚焦
              if (!e.target.closest('.trend-options')) {
                e.currentTarget.focus();
              }
            }}
          >
            <label class="timeframe-label">{timeframe}</label>
            <div class="trend-options">
              <label class="trend-option" class:active={formData.trend_analysis[timeframe].direction === 'long'}>
                <input 
                  type="radio" 
                  name="trend_{timeframe}"
                  value="long"
                  bind:group={formData.trend_analysis[timeframe].direction}
                />
                <span class="trend-name">多</span>
              </label>
              <label class="trend-option" class:active={formData.trend_analysis[timeframe].direction === 'short'}>
                <input 
                  type="radio" 
                  name="trend_{timeframe}"
                  value="short"
                  bind:group={formData.trend_analysis[timeframe].direction}
                />
                <span class="trend-name">空</span>
              </label>
            </div>
            
            <!-- 顯示已貼上的圖片 -->
            {#if formData.trend_analysis[timeframe].image}
              <div class="trend-image-preview">
                <img 
                  src={formData.trend_analysis[timeframe].image} 
                  alt="{timeframe} 趨勢圖"
                  on:click={(e) => {
                    e.stopPropagation();
                    enlargeImage(formData.trend_analysis[timeframe].image, timeframe + ' 趨勢圖');
                  }}
                  style="cursor: zoom-in;"
                />
                <button 
                  type="button" 
                  class="remove-trend-image"
                  on:click={(e) => {
                    e.stopPropagation();
                    removeTrendImage(timeframe);
                  }}
                  title="移除圖片"
                >
                  ×
                </button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <div class="form-group">
      <label for="exit_reason">
        🎯 平倉理由
        <span class="hint-inline">（支援圖片貼上：Ctrl+V 或點擊工具列圖片按鈕）</span>
      </label>
      <RichTextEditor 
        bind:this={exitReasonEditor}
        bind:value={formData.exit_reason}
        placeholder="為什麼平倉？止盈/止損/訊號反轉？可以貼上圖片說明..."
        height="180px"
      />
    </div>

    <div class="form-group">
      <label for="notes">
        📝 交易復盤
        <span class="hint-inline">（支援圖片貼上：Ctrl+V 或點擊工具列圖片按鈕）</span>
      </label>
      <RichTextEditor 
        bind:this={notesEditor}
        bind:value={formData.notes}
        placeholder="記錄當下的心態、策略、失誤等...可以貼上圖片說明..."
        height="200px"
      />
    </div>

    <div class="form-group">
      <label>標籤</label>
      <div class="tag-input-wrapper">
        <input type="text" class="form-control" bind:value={tagInput} 
               placeholder="輸入標籤（如：突破、回踩、新聞單）" 
               on:keypress={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())} />
        <button type="button" class="btn btn-primary" on:click={addTag}>新增</button>
      </div>
      <div class="tags-container">
        {#each formData.tags as tag}
          <span class="tag">
            #{tag}
            <button type="button" class="tag-remove" on:click={() => removeTag(tag)}>×</button>
          </span>
        {/each}
      </div>
    </div>

    <div class="form-actions">
      <button type="button" class="btn" on:click={() => navigate('/')}>取消</button>
      <button type="submit" class="btn btn-primary" disabled={saving}>
        {#if saving}
          儲存中...
        {:else}
          {id ? '更新' : '建立'}交易
        {/if}
      </button>
    </div>
  </form>
</div>

<!-- 圖片放大查看模態視窗 -->
{#if enlargedImage}
  <div class="image-modal" on:click={closeEnlargedImage}>
    <div class="image-modal-content" on:click={(e) => e.stopPropagation()}>
      <button class="image-modal-close" on:click={closeEnlargedImage}>×</button>
      <h3 class="image-modal-title">{enlargedImageTitle}</h3>
      <img src={enlargedImage} alt={enlargedImageTitle} class="image-modal-img" />
    </div>
  </div>
{/if}

<style>
  h2 {
    margin-bottom: 2rem;
    color: #2d3748;
  }

  /* 交易類型選擇 */
  .trade-type-section {
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
  }

  .trade-type-label {
    display: block;
    font-size: 1.1rem;
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 1rem;
  }

  .trade-type-options {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .radio-option {
    position: relative;
    cursor: pointer;
    border: 2px solid #cbd5e0;
    border-radius: 12px;
    padding: 1.25rem;
    background: white;
    transition: all 0.2s ease;
  }

  .radio-option:hover {
    border-color: #667eea;
    background: #f7fafc;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.1);
  }

  .radio-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .radio-option input[type="radio"] {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .radio-icon {
    font-size: 2rem;
    line-height: 1;
  }

  .radio-text {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .radio-text strong {
    font-size: 1rem;
    color: #2d3748;
  }

  .radio-text small {
    font-size: 0.85rem;
    color: #718096;
  }

  .form-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  /* 進場種類選擇 */
  .entry-strategy-section {
    margin: 1.5rem 0;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
  }

  .entry-strategy-section:hover {
    border-color: #667eea;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .entry-strategy-section:focus {
    border-color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .strategy-label {
    display: block;
    font-size: 1rem;
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 1rem;
  }

  .strategy-options {
    display: flex;
    gap: 1rem;
    margin-bottom: 1.5rem;
    flex-wrap: wrap;
  }

  .strategy-option {
    position: relative;
    cursor: pointer;
    padding: 0.75rem 1.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    transition: all 0.2s ease;
  }

  .strategy-option:hover {
    border-color: #667eea;
    background: #f7fafc;
    transform: translateY(-1px);
  }

  .strategy-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .strategy-option input[type="radio"] {
    position: absolute;
    opacity: 0;
  }

  .strategy-name {
    font-weight: 600;
    color: #2d3748;
  }

  .strategy-option.active .strategy-name {
    color: #667eea;
  }

  /* 進場種類圖片預覽 */
  .strategy-image-preview {
    position: relative;
    margin-top: 1rem;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
  }

  .strategy-image-preview img {
    width: 100%;
    height: auto;
    display: block;
    max-height: 300px;
    object-fit: contain;
    background: white;
  }

  .remove-strategy-image {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 28px;
    height: 28px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.3rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .remove-strategy-image:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  /* 訊號和檢查清單 */
  .signals-section,
  .checklist-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .signals-label,
  .checklist-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.75rem;
  }

  .signals-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 0.75rem;
  }

  .checklist-items {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .checkbox-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 6px;
    transition: background 0.2s ease;
  }

  .checkbox-item:hover {
    background: #f7fafc;
  }

  .checkbox-item input[type="checkbox"] {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #667eea;
  }

  .checkbox-label {
    font-size: 0.9rem;
    color: #2d3748;
    user-select: none;
  }

  /* 市場時段顯示 */
  .market-session-display {
    display: flex;
    align-items: center;
    height: auto;
    padding: 0.5rem 0;
  }

  .market-session-info {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .market-session-badge {
    display: inline-block;
    padding: 0.5rem 1rem;
    border-radius: 20px;
    font-weight: 600;
    font-size: 0.95rem;
    text-align: center;
    width: fit-content;
  }

  .market-session-badge.asian {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
  }

  .market-session-badge.european {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
    color: white;
  }

  .market-session-badge.us {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
    color: white;
  }

  .session-details {
    display: flex;
    gap: 0.75rem;
    font-size: 0.85rem;
    color: #718096;
    padding-left: 0.25rem;
  }

  .session-time {
    font-weight: 600;
    color: #4a5568;
  }

  .session-season {
    color: #a0aec0;
  }

  .session-season::before {
    content: '•';
    margin-right: 0.5rem;
  }

  /* 當前趨勢選擇 */
  .trend-analysis-section {
    margin: 1.5rem 0;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
  }

  .trend-label {
    display: block;
    font-size: 1rem;
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 1rem;
  }

  .trend-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .trend-item {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
  }

  .trend-item:hover {
    border-color: #667eea;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .trend-item:focus {
    border-color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .timeframe-label {
    display: block;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
  }

  .trend-options {
    display: flex;
    gap: 0.5rem;
  }

  .trend-option {
    flex: 1;
    position: relative;
    cursor: pointer;
    padding: 0.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    transition: all 0.2s ease;
    text-align: center;
  }

  .trend-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .trend-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
  }

  .trend-option input[type="radio"] {
    position: absolute;
    opacity: 0;
  }

  .trend-name {
    font-weight: 600;
    color: #2d3748;
    font-size: 0.9rem;
  }

  .trend-option.active .trend-name {
    color: #667eea;
  }

  /* 趨勢圖片預覽 */
  .trend-image-preview {
    position: relative;
    margin-top: 0.5rem;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
  }

  .trend-image-preview img {
    width: 100%;
    height: auto;
    display: block;
    max-height: 200px;
    object-fit: contain;
    background: #f7fafc;
  }

  .remove-trend-image {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 24px;
    height: 24px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.2rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .remove-trend-image:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  /* 圖片放大查看模態視窗 */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    padding: 2rem;
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .image-modal-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
    background: white;
    border-radius: 12px;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from {
      transform: scale(0.9);
      opacity: 0;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }

  .image-modal-close {
    position: absolute;
    top: 1rem;
    right: 1rem;
    width: 36px;
    height: 36px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.5rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
    z-index: 1;
  }

  .image-modal-close:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  .image-modal-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #2d3748;
    margin: 0;
    padding-right: 3rem;
  }

  .image-modal-img {
    max-width: 100%;
    max-height: calc(90vh - 8rem);
    object-fit: contain;
    border-radius: 8px;
  }

  .tag-input-wrapper {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .tag {
    background: #667eea;
    color: white;
    padding: 0.5rem 1rem;
    border-radius: 20px;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
  }

  .tag-remove {
    background: none;
    border: none;
    color: white;
    font-size: 1.5rem;
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
    padding-top: 2rem;
    border-top: 2px solid #e2e8f0;
  }

  textarea.form-control {
    resize: vertical;
    font-family: inherit;
  }

  .hint-inline {
    color: #a0aec0;
    font-size: 0.85rem;
    font-weight: normal;
    margin-left: 0.5rem;
  }

  label {
    display: flex;
    align-items: center;
    margin-bottom: 0.5rem;
  }
</style>
