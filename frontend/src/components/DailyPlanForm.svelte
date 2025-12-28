<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { dailyPlansAPI } from '../lib/api';
  import { SYMBOLS, TIMEFRAMES, MARKET_SESSIONS } from '../lib/constants';
  import { selectedAccountId } from '../lib/stores';
  import ImageAnnotator from './ImageAnnotator.svelte';

  export let id = null;

  let activeSession = 'asian'; // 當前切換的分頁

  // 使用從 constants 引入的時限
  const timeframes = TIMEFRAMES;

  // 初始化單個時段的結構
  function createInitialSessionData() {
    const trends = {};
    timeframes.forEach(tf => {
      trends[tf] = {
        direction: '',
        signals: [],
        wave_numbers: [],
        wave_highlight: '',
        image: '',
        originalImage: '',
        signals_image: '',
        signals_originalImage: '',
        wave_image: '',
        wave_originalImage: '',
      };
    });
    return {
      notes: '',
      trends: trends,
    };
  }

  const symbols = SYMBOLS;

  let formData = {
    account_id: $selectedAccountId,
    plan_date: new Date().toISOString().slice(0, 10),
    symbol: SYMBOLS[0],
    sessions: {
      asian: createInitialSessionData(),
      european: createInitialSessionData(),
      us: createInitialSessionData(),
    },
  };

  // 快捷獲取當前分頁資料
  $: currentSessionData = formData.sessions[activeSession];
  $: currentTrends = currentSessionData.trends;
  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const dateParam = params.get('date');
    const sessionParam = params.get('session');
    const symbolParam = params.get('symbol');

    if (dateParam) formData.plan_date = dateParam;
    if (sessionParam) activeSession = sessionParam;
    if (symbolParam) formData.symbol = symbolParam;
  });

  // 達人訊號選項 - 根據做多/做空顯示不同訊號
  const expertSignalsLong = ['向下蘇美', '起漲靠山', '雙柱', '倚天', '攻城池上'];

  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '向上蘇美', '雷霆'];

  // 波浪數字選項
  const waveNumbers = ['1', '2', '3', '4', '5'];

  // 根據時區的方向獲取對應的訊號列表
  function getSignalsForTimeframe(timeframe) {
    const direction = currentTrends[timeframe].direction;
    if (direction === 'long') return expertSignalsLong;
    if (direction === 'short') return expertSignalsShort;
    return [];
  }

  // 切換時區的訊號選擇
  function toggleTimeframeSignal(timeframe, signalName) {
    const signals = currentTrends[timeframe].signals || [];
    const index = signals.indexOf(signalName);

    if (index >= 0) {
      // 取消選擇
      currentTrends[timeframe].signals = signals.filter((_, i) => i !== index);
    } else {
      // 新增選擇
      currentTrends[timeframe].signals = [...signals, signalName];
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 檢查時區訊號是否被選中
  function isTimeframeSignalSelected(timeframe, signalName) {
    const signals = currentTrends[timeframe].signals || [];
    return signals.includes(signalName);
  }

  // 點擊波浪數字
  function clickWaveNumber(timeframe, number) {
    const selectedNumbers = currentTrends[timeframe].wave_numbers || [];
    const currentHighlight = currentTrends[timeframe].wave_highlight || '';

    // 如果這個數字已經被選中
    if (selectedNumbers.includes(number)) {
      // 如果是綠色（未高亮），變成紅色（高亮）
      if (currentHighlight !== number) {
        currentTrends[timeframe].wave_highlight = number;
      } else {
        // 如果已經是紅色，變回綠色
        currentTrends[timeframe].wave_highlight = '';
      }
    } else {
      // 數字未被選中，嘗試選中
      if (selectedNumbers.length === 0) {
        currentTrends[timeframe].wave_numbers = [number];
        currentTrends[timeframe].wave_highlight = '';
      } else if (selectedNumbers.length === 1) {
        const existingNum = parseInt(selectedNumbers[0]);
        const newNum = parseInt(number);

        if (Math.abs(existingNum - newNum) === 1) {
          currentTrends[timeframe].wave_numbers = [selectedNumbers[0], number].sort();
          currentTrends[timeframe].wave_highlight = '';
        }
      } else if (selectedNumbers.length === 2) {
        currentTrends[timeframe].wave_numbers = [number];
        currentTrends[timeframe].wave_highlight = '';
      }
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 檢查波浪數字是否被選中（綠色）
  function isWaveNumberSelected(timeframe, number) {
    const selectedNumbers = currentTrends[timeframe]?.wave_numbers || [];
    return (
      selectedNumbers.includes(number.toString()) || selectedNumbers.includes(parseInt(number))
    );
  }

  // 檢查波浪數字是否被高亮（紅色）
  function isWaveNumberHighlighted(timeframe, number) {
    const highlight = currentTrends[timeframe]?.wave_highlight;
    return highlight === number.toString() || highlight === parseInt(number);
  }

  // 圖片放大相關
  let enlargedImage = null;
  let enlargedImageTitle = '';
  let enlargedImageContext = null;
  let enlargedOriginalImage = null;
  let showAnnotator = false;

  // 用於強制重新渲染波浪按鈕的響應式變量
  let waveButtonKey = 0;

  // 載入規劃（如果是編輯模式）
  if (id) {
    loadPlan();
  }

  async function loadPlan() {
    try {
      const response = await dailyPlansAPI.getOne(id);
      const data = response.data;
      const trendAnalysis = data.trend_analysis ? JSON.parse(data.trend_analysis) : null;

      formData.plan_date = new Date(data.plan_date).toLocaleDateString('en-CA');
      formData.symbol = data.symbol || SYMBOLS[0];

      if (trendAnalysis && trendAnalysis.asian) {
        // 新格式：包含各時段
        formData.sessions = trendAnalysis;
      } else if (trendAnalysis) {
        // 舊格式：遷移至當前 market_session
        const session = data.market_session || 'asian';
        formData.sessions[session] = {
          notes: data.notes || '',
          trends: trendAnalysis,
        };
      }
      formData = formData;
    } catch (error) {
      console.error('載入規劃失敗:', error);
      alert('載入規劃資料失敗');
    }
  }

  async function handleSubmit() {
    try {
      const submitData = {
        account_id: $selectedAccountId,
        plan_date: new Date(formData.plan_date).toISOString(),
        symbol: formData.symbol,
        market_session: 'all', // 標記為整合格式
        notes: 'Session-based unified plan',
        trend_analysis: JSON.stringify(formData.sessions),
      };

      if (id) {
        await dailyPlansAPI.update(id, submitData);
        alert('規劃已更新');
      } else {
        await dailyPlansAPI.create(submitData);
        alert('規劃已建立');
      }

      navigate('/plans');
    } catch (error) {
      console.error('保存失敗:', error);
      const errorMessage = error.response?.data?.error || '保存規劃失敗';
      alert(errorMessage);
    }
  }

  // 處理趨勢圖片貼上
  function handleTrendImagePaste(event, timeframe, imageType = 'trend') {
    const items = (event.clipboardData || event.originalEvent.clipboardData).items;

    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = item.getAsFile();
        const reader = new FileReader();

        reader.onload = e => {
          const trends = currentTrends[timeframe];
          // 根據 imageType 設置不同的圖片欄位
          if (imageType === 'signals') {
            trends.signals_image = e.target.result;
            if (!trends.signals_originalImage) {
              trends.signals_originalImage = e.target.result;
            }
          } else if (imageType === 'wave') {
            trends.wave_image = e.target.result;
            if (!trends.wave_originalImage) {
              trends.wave_originalImage = e.target.result;
            }
          } else {
            trends.image = e.target.result;
            if (!trends.originalImage) {
              trends.originalImage = e.target.result;
            }
          }

          // 強制觸發 Svelte 響應式更新
          formData = formData;
          waveButtonKey++;
        };

        reader.readAsDataURL(file);
        break;
      }
    }
  }

  // 移除趨勢圖片
  function removeTrendImage(timeframe, imageType = 'trend') {
    const trends = currentTrends[timeframe];
    if (imageType === 'signals') {
      trends.signals_image = '';
      trends.signals_originalImage = '';
    } else if (imageType === 'wave') {
      trends.wave_image = '';
      trends.wave_originalImage = '';
    } else {
      trends.image = '';
      trends.originalImage = '';
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 放大圖片
  function enlargeImage(imageSrc, title, context = null) {
    if (!imageSrc) return;
    enlargedImage = imageSrc;
    enlargedImageTitle = title;
    enlargedImageContext = context;
    showAnnotator = false;

    // 獲取原始圖片
    if (context) {
      const trends = currentTrends[context.key];
      if (context.type === 'trend') {
        enlargedOriginalImage = trends?.originalImage || imageSrc;
      } else if (context.type === 'signals') {
        enlargedOriginalImage = trends?.signals_originalImage || imageSrc;
      } else if (context.type === 'wave') {
        enlargedOriginalImage = trends?.wave_originalImage || imageSrc;
      } else {
        enlargedOriginalImage = imageSrc;
      }
    } else {
      enlargedOriginalImage = imageSrc;
    }
  }

  // 關閉放大圖片
  function closeEnlargedImage() {
    enlargedImage = null;
    enlargedImageTitle = '';
    enlargedImageContext = null;
    showAnnotator = false;
  }

  // 切換標註工具顯示
  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  // 處理標註後的圖片
  function handleAnnotatedImage(annotatedImageSrc) {
    if (!enlargedImageContext) {
      enlargedImage = annotatedImageSrc;
      return;
    }

    const { type, key } = enlargedImageContext;
    const trends = currentTrends[key];

    if (type === 'trend') {
      trends.image = annotatedImageSrc;
    } else if (type === 'signals') {
      trends.signals_image = annotatedImageSrc;
    } else if (type === 'wave') {
      trends.wave_image = annotatedImageSrc;
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;

    enlargedImage = annotatedImageSrc;
  }
</script>

<div class="card">
  <h2>{id ? '編輯每日盤面規劃' : '新增每日盤面規劃'}</h2>

  <form on:submit|preventDefault={handleSubmit}>
    <!-- 基本資料 -->
    <div class="form-section">
      <h3>📅 基本資料</h3>

      <!-- 規劃日期 -->
      <div class="form-group">
        <label for="plan_date">規劃日期</label>
        <input
          type="date"
          id="plan_date"
          class="form-control"
          bind:value={formData.plan_date}
          required
        />
      </div>

      <!-- 交易品種 -->
      <div class="form-group">
        <label for="symbol">交易品種</label>
        <select id="symbol" class="form-control" bind:value={formData.symbol}>
          {#each symbols as sym}
            <option value={sym}>{sym}</option>
          {/each}
        </select>
      </div>

      <!-- 市場時段 (分頁切換) -->
      <div class="form-group">
        <label>市場時段 (分頁)</label>
        <div class="market-session-tabs">
          <button
            type="button"
            class="session-tab"
            class:active={activeSession === 'asian'}
            on:click={() => (activeSession = 'asian')}
          >
            {MARKET_SESSIONS.find(s => s.value === 'asian')?.icon}
            {MARKET_SESSIONS.find(s => s.value === 'asian')?.label}
          </button>
          <button
            type="button"
            class="session-tab"
            class:active={activeSession === 'european'}
            on:click={() => (activeSession = 'european')}
          >
            {MARKET_SESSIONS.find(s => s.value === 'european')?.icon}
            {MARKET_SESSIONS.find(s => s.value === 'european')?.label}
          </button>
          <button
            type="button"
            class="session-tab"
            class:active={activeSession === 'us'}
            on:click={() => (activeSession = 'us')}
          >
            {MARKET_SESSIONS.find(s => s.value === 'us')?.icon}
            {MARKET_SESSIONS.find(s => s.value === 'us')?.label}
          </button>
        </div>
      </div>

      <!-- 備註 -->
      <div class="form-group">
        <label for="notes">備註</label>
        <textarea
          id="notes"
          class="form-control"
          bind:value={currentSessionData.notes}
          rows="3"
          placeholder="今日盤面重點、注意事項..."
        ></textarea>
      </div>
    </div>

    <!-- 當前各時區趨勢 -->
    <div class="form-group trend-analysis-section">
      <label class="trend-label">📊 當前各時區趨勢</label>
      <div class="trend-grid">
        {#each TIMEFRAMES as timeframe}
          <div
            class="trend-item"
            tabindex="0"
            on:paste={e => handleTrendImagePaste(e, timeframe)}
            on:click={e => {
              if (!e.target.closest('.trend-options')) {
                e.currentTarget.focus();
              }
            }}
          >
            <label class="timeframe-label">{timeframe}</label>

            <!-- 多空選擇 -->
            <div class="trend-options">
              <label
                class="trend-option"
                class:active={currentTrends[timeframe].direction === 'long'}
              >
                <input
                  type="radio"
                  name="trend_{timeframe}_{activeSession}"
                  value="long"
                  bind:group={currentTrends[timeframe].direction}
                />
                <span class="trend-name">多</span>
              </label>
              <label
                class="trend-option"
                class:active={currentTrends[timeframe].direction === 'short'}
              >
                <input
                  type="radio"
                  name="trend_{timeframe}_{activeSession}"
                  value="short"
                  bind:group={currentTrends[timeframe].direction}
                />
                <span class="trend-name">空</span>
              </label>
            </div>

            <!-- 達人訊號選擇 -->
            <div class="timeframe-signals">
              <label class="section-label">達人訊號：</label>
              <div class="signal-chips">
                {#each getSignalsForTimeframe(timeframe) as signal (waveButtonKey + '-' + timeframe + '-signal-' + signal)}
                  <button
                    type="button"
                    class="signal-chip"
                    class:active={isTimeframeSignalSelected(timeframe, signal)}
                    on:click|stopPropagation={() => toggleTimeframeSignal(timeframe, signal)}
                  >
                    {signal}
                  </button>
                {/each}
              </div>

              <!-- 達人訊號圖片 -->
              {#if currentTrends[timeframe].signals_image}
                <div
                  class="trend-image-preview"
                  on:click|stopPropagation={() =>
                    enlargeImage(
                      currentTrends[timeframe].signals_image,
                      `${timeframe} 達人訊號圖`,
                      { type: 'signals', key: timeframe }
                    )}
                >
                  <img
                    src={currentTrends[timeframe].signals_image}
                    alt="{timeframe} 達人訊號"
                    style="pointer-events: none;"
                  />
                  <button
                    type="button"
                    class="remove-image-btn"
                    on:click|stopPropagation={() => removeTrendImage(timeframe, 'signals')}
                    title="移除圖片"
                  >
                    ×
                  </button>
                </div>
              {:else}
                <div
                  class="trend-image-placeholder"
                  tabindex="0"
                  on:paste|preventDefault|stopPropagation={e =>
                    handleTrendImagePaste(e, timeframe, 'signals')}
                  on:click|stopPropagation={e => e.target.focus()}
                  role="textbox"
                >
                  📋 Ctrl+V 貼上達人訊號圖片
                </div>
              {/if}
            </div>

            <!-- 波浪浪數選擇 -->
            <div class="timeframe-wave">
              <label class="section-label">波浪浪數：</label>
              <div class="wave-numbers">
                {#each waveNumbers as num (waveButtonKey + '-' + timeframe + '-' + num)}
                  <button
                    type="button"
                    class="wave-number-btn"
                    class:selected={isWaveNumberSelected(timeframe, num)}
                    class:highlighted={isWaveNumberHighlighted(timeframe, num)}
                    on:click|stopPropagation={() => clickWaveNumber(timeframe, num)}
                  >
                    {num}
                  </button>
                {/each}
              </div>

              <!-- 波浪圖片 -->
              {#if currentTrends[timeframe].wave_image}
                <div
                  class="trend-image-preview"
                  on:click|stopPropagation={() =>
                    enlargeImage(currentTrends[timeframe].wave_image, `${timeframe} 波浪圖`, {
                      type: 'wave',
                      key: timeframe,
                    })}
                >
                  <img
                    src={currentTrends[timeframe].wave_image}
                    alt="{timeframe} 波浪"
                    style="pointer-events: none;"
                  />
                  <button
                    type="button"
                    class="remove-image-btn"
                    on:click|stopPropagation={() => removeTrendImage(timeframe, 'wave')}
                    title="移除圖片"
                  >
                    ×
                  </button>
                </div>
              {:else}
                <div
                  class="trend-image-placeholder"
                  tabindex="0"
                  on:paste|preventDefault|stopPropagation={e =>
                    handleTrendImagePaste(e, timeframe, 'wave')}
                  on:click|stopPropagation={e => e.target.focus()}
                  role="textbox"
                >
                  📋 Ctrl+V 貼上波浪圖片
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- 操作按鈕 -->
    <div class="form-actions">
      <button type="submit" class="btn btn-primary">
        {id ? '💾 更新規劃' : '✅ 建立規劃'}
      </button>
      <button type="button" class="btn btn-secondary" on:click={() => navigate('/plans')}>
        ❌ 取消
      </button>
    </div>
  </form>
</div>

<!-- 圖片放大模態框 -->
{#if enlargedImage}
  <div class="image-modal" on:click={closeEnlargedImage}>
    <div class="image-modal-content" on:click|stopPropagation>
      <div class="image-modal-header">
        <h3>{enlargedImageTitle}</h3>
        <div class="image-modal-actions">
          <button class="modal-action-btn" on:click={toggleAnnotator}>
            {showAnnotator ? '👁️ 查看' : '✏️ 標註'}
          </button>
          <button class="image-modal-close" on:click={closeEnlargedImage}>×</button>
        </div>
      </div>

      {#if showAnnotator}
        <ImageAnnotator
          imageSrc={enlargedImage}
          originalImageSrc={enlargedOriginalImage}
          onSave={handleAnnotatedImage}
        />
      {:else}
        <img src={enlargedImage} alt={enlargedImageTitle} class="image-modal-img" />
      {/if}
    </div>
  </div>
{/if}

<style>
  h2 {
    margin-bottom: 2rem;
    color: #2d3748;
  }

  h3 {
    font-size: 1.2rem;
    color: #4a5568;
    margin-bottom: 1rem;
  }

  .form-section {
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
  }

  /* 市場時段分頁 */
  .market-session-tabs {
    display: flex;
    gap: 0.5rem;
    background: #edf2f7;
    padding: 0.4rem;
    border-radius: 12px;
  }

  .session-tab {
    flex: 1;
    padding: 0.75rem;
    border: none;
    background: transparent;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    color: #4a5568;
    transition: all 0.2s;
  }

  .session-tab.active {
    background: white;
    color: #667eea;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .session-tab:hover:not(.active) {
    background: #f7fafc;
  }

  /* 趨勢分析 */
  .trend-analysis-section {
    margin-top: 2rem;
  }

  .trend-label {
    display: block;
    font-size: 1.1rem;
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
    padding: 1rem;
    background: white;
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .trend-item:hover {
    border-color: #cbd5e0;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }

  .trend-item:focus {
    outline: none;
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
    margin-bottom: 0.75rem;
  }

  .trend-option {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
  }

  .trend-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .trend-option.active {
    border-color: #667eea;
    background: #667eea;
  }

  .trend-option input[type='radio'] {
    display: none;
  }

  .trend-name {
    font-size: 0.9rem;
    font-weight: 500;
    color: #2d3748;
  }

  .trend-option.active .trend-name {
    color: white;
  }

  /* 時區訊號選擇 */
  .timeframe-signals {
    margin-top: 0.75rem;
  }

  .section-label {
    display: block;
    font-size: 0.8rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.5rem;
  }

  .signal-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .signal-chip {
    display: inline-flex;
    align-items: center;
    padding: 0.3rem 0.6rem;
    border: 1.5px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    font-size: 0.75rem;
    user-select: none;
  }

  .signal-chip:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .signal-chip.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  /* 波浪浪數選擇 */
  .timeframe-wave {
    margin-top: 0.75rem;
  }

  .wave-numbers {
    display: flex;
    gap: 0.4rem;
  }

  .wave-number-btn {
    flex: 1;
    padding: 0.4rem;
    border: 1.5px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    font-size: 0.8rem;
    font-weight: 600;
    user-select: none;
    color: #2d3748;
  }

  .wave-number-btn:hover {
    border-color: #48bb78;
    background: #f7fafc;
  }

  .wave-number-btn.selected {
    border-color: #48bb78 !important;
    background: #48bb78 !important;
    color: white !important;
  }

  .wave-number-btn.highlighted {
    border-color: #e53e3e !important;
    background: #e53e3e !important;
    color: white !important;
  }

  .trend-image-preview {
    position: relative;
    margin-top: 0.5rem;
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    border: 2px solid #e2e8f0;
  }

  .trend-image-preview:hover {
    border-color: #667eea;
  }

  .trend-image-preview img {
    width: 100%;
    height: auto;
    display: block;
  }

  .remove-image-btn {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 50%;
    background: rgba(239, 68, 68, 0.9);
    color: white;
    font-size: 1.5rem;
    line-height: 1;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }

  .remove-image-btn:hover {
    background: rgb(239, 68, 68);
    transform: scale(1.1);
  }

  .trend-image-placeholder {
    margin-top: 0.5rem;
    padding: 1.5rem;
    border: 2px dashed #cbd5e0;
    border-radius: 8px;
    text-align: center;
    color: #a0aec0;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
  }

  .trend-image-placeholder:hover {
    border-color: #667eea;
    background: #f7fafc;
    color: #667eea;
  }

  .trend-image-placeholder:focus {
    border-color: #667eea;
    background: #edf2f7;
    color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  /* 操作按鈕 */
  .form-actions {
    display: flex;
    gap: 1rem;
    margin-top: 2rem;
    padding-top: 2rem;
    border-top: 2px solid #e2e8f0;
  }

  .btn-secondary {
    background: #e2e8f0;
    color: #2d3748;
  }

  .btn-secondary:hover {
    background: #cbd5e0;
  }

  /* 圖片放大模態框 */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 2rem;
  }

  .image-modal-content {
    background: white;
    border-radius: 12px;
    max-width: 90vw;
    max-height: 90vh;
    overflow: auto;
    position: relative;
  }

  .image-modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 2px solid #e2e8f0;
    position: sticky;
    top: 0;
    background: white;
    z-index: 10;
  }

  .image-modal-header h3 {
    margin: 0;
    font-size: 1.2rem;
    color: #2d3748;
  }

  .image-modal-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .modal-action-btn {
    padding: 0.5rem 1rem;
    border: 2px solid #667eea;
    border-radius: 6px;
    background: white;
    color: #667eea;
    cursor: pointer;
    font-weight: 600;
    transition: all 0.2s ease;
  }

  .modal-action-btn:hover {
    background: #667eea;
    color: white;
  }

  .image-modal-close {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 50%;
    background: #f56565;
    color: white;
    font-size: 1.5rem;
    line-height: 1;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }

  .image-modal-close:hover {
    background: #e53e3e;
    transform: rotate(90deg);
  }

  .image-modal-img {
    display: block;
    max-width: 100%;
    height: auto;
    padding: 1rem;
  }

  textarea.form-control {
    resize: vertical;
    font-family: inherit;
  }
</style>
