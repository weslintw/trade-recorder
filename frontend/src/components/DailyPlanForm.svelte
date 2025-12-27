<script>
  import { navigate } from 'svelte-routing';
  import { dailyPlansAPI } from '../lib/api';
  import ImageAnnotator from './ImageAnnotator.svelte';

  export let id = null;

  let formData = {
    plan_date: new Date().toISOString().slice(0, 10), // 規劃日期
    market_session: '', // asian=亞盤, european=歐盤, us=美盤
    notes: '', // 備註
    trend_analysis: { // 當前各時區趨勢
      M1: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      M5: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      M15: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      M30: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      H1: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      H4: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' },
      D1: { direction: '', signals: [], wave_numbers: [], wave_highlight: '', image: '', originalImage: '', signals_image: '', signals_originalImage: '', wave_image: '', wave_originalImage: '' }
    }
  };

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

  // 波浪數字選項
  const waveNumbers = ['1', '2', '3', '4', '5'];

  // 根據時區的方向獲取對應的訊號列表
  function getSignalsForTimeframe(timeframe) {
    const direction = formData.trend_analysis[timeframe].direction;
    if (direction === 'long') return expertSignalsLong;
    if (direction === 'short') return expertSignalsShort;
    return [];
  }

  // 切換時區的訊號選擇
  function toggleTimeframeSignal(timeframe, signalName) {
    const signals = formData.trend_analysis[timeframe].signals || [];
    const index = signals.indexOf(signalName);
    
    if (index >= 0) {
      // 取消選擇
      formData.trend_analysis[timeframe].signals = signals.filter((_, i) => i !== index);
    } else {
      // 新增選擇
      formData.trend_analysis[timeframe].signals = [...signals, signalName];
    }
    
    // 強制觸發 Svelte 響應式更新 - 創建完全新的對象
    const newTrendAnalysis = {};
    for (const key in formData.trend_analysis) {
      newTrendAnalysis[key] = { ...formData.trend_analysis[key] };
    }
    
    formData = {
      ...formData,
      trend_analysis: newTrendAnalysis
    };
    
    // 強制重新渲染
    waveButtonKey++;
  }

  // 檢查時區訊號是否被選中
  function isTimeframeSignalSelected(timeframe, signalName) {
    const signals = formData.trend_analysis[timeframe].signals || [];
    return signals.includes(signalName);
  }

  // 點擊波浪數字
  function clickWaveNumber(timeframe, number) {
    console.log('clickWaveNumber called:', timeframe, number);
    const selectedNumbers = formData.trend_analysis[timeframe].wave_numbers || [];
    const currentHighlight = formData.trend_analysis[timeframe].wave_highlight || '';
    
    console.log('Current selected numbers:', selectedNumbers);
    console.log('Current highlight:', currentHighlight);
    
    // 如果這個數字已經被選中
    if (selectedNumbers.includes(number)) {
      // 如果是綠色（未高亮），變成紅色（高亮）
      if (currentHighlight !== number) {
        formData.trend_analysis[timeframe] = {
          ...formData.trend_analysis[timeframe],
          wave_highlight: number
        };
      } else {
        // 如果已經是紅色，變回綠色
        formData.trend_analysis[timeframe] = {
          ...formData.trend_analysis[timeframe],
          wave_highlight: ''
        };
      }
    } else {
      // 數字未被選中，嘗試選中
      if (selectedNumbers.length === 0) {
        // 第一次選擇，直接選中
        formData.trend_analysis[timeframe] = {
          ...formData.trend_analysis[timeframe],
          wave_numbers: [number],
          wave_highlight: ''
        };
      } else if (selectedNumbers.length === 1) {
        // 已有一個數字，檢查是否相鄰
        const existingNum = parseInt(selectedNumbers[0]);
        const newNum = parseInt(number);
        
        console.log('Checking adjacency:', existingNum, newNum, Math.abs(existingNum - newNum));
        
        if (Math.abs(existingNum - newNum) === 1) {
          // 相鄰，可以選中
          formData.trend_analysis[timeframe] = {
            ...formData.trend_analysis[timeframe],
            wave_numbers: [selectedNumbers[0], number].sort(),
            wave_highlight: ''
          };
        } else {
          console.log('Numbers are not adjacent, cannot select');
        }
      } else if (selectedNumbers.length === 2) {
        // 已有兩個數字，重新開始選擇
        formData.trend_analysis[timeframe] = {
          ...formData.trend_analysis[timeframe],
          wave_numbers: [number],
          wave_highlight: ''
        };
      }
    }
    
    console.log('After update:', formData.trend_analysis[timeframe].wave_numbers);
    
    // 強制觸發 Svelte 響應式更新 - 創建完全新的對象
    const newTrendAnalysis = {};
    for (const key in formData.trend_analysis) {
      newTrendAnalysis[key] = { ...formData.trend_analysis[key] };
    }
    
    formData = {
      ...formData,
      trend_analysis: newTrendAnalysis
    };
    
    // 強制重新渲染波浪按鈕
    waveButtonKey++;
  }

  // 檢查波浪數字是否被選中（綠色）
  function isWaveNumberSelected(timeframe, number) {
    const selectedNumbers = formData.trend_analysis[timeframe]?.wave_numbers || [];
    const isSelected = selectedNumbers.includes(number.toString()) || selectedNumbers.includes(parseInt(number));
    console.log(`Checking if ${number} (type: ${typeof number}) is selected in ${timeframe}:`, selectedNumbers, 'Result:', isSelected);
    return isSelected;
  }

  // 檢查波浪數字是否被高亮（紅色）
  function isWaveNumberHighlighted(timeframe, number) {
    const highlight = formData.trend_analysis[timeframe]?.wave_highlight;
    const result = highlight === number.toString() || highlight === parseInt(number);
    console.log(`Checking if ${number} (type: ${typeof number}) is highlighted in ${timeframe}:`, highlight, 'Result:', result);
    return result;
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
      formData = {
        ...response.data,
        trend_analysis: response.data.trend_analysis ? JSON.parse(response.data.trend_analysis) : formData.trend_analysis
      };
    } catch (error) {
      console.error('載入規劃失敗:', error);
      alert('載入規劃資料失敗');
    }
  }

  async function handleSubmit() {
    try {
      const submitData = {
        ...formData,
        trend_analysis: JSON.stringify(formData.trend_analysis)
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
      alert('保存規劃失敗');
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
        
        reader.onload = (e) => {
          // 根據 imageType 設置不同的圖片欄位
          if (imageType === 'signals') {
            formData.trend_analysis[timeframe].signals_image = e.target.result;
            if (!formData.trend_analysis[timeframe].signals_originalImage) {
              formData.trend_analysis[timeframe].signals_originalImage = e.target.result;
            }
          } else if (imageType === 'wave') {
            formData.trend_analysis[timeframe].wave_image = e.target.result;
            if (!formData.trend_analysis[timeframe].wave_originalImage) {
              formData.trend_analysis[timeframe].wave_originalImage = e.target.result;
            }
          } else {
            // 舊的趨勢圖（保留向後兼容）
            formData.trend_analysis[timeframe].image = e.target.result;
            if (!formData.trend_analysis[timeframe].originalImage) {
              formData.trend_analysis[timeframe].originalImage = e.target.result;
            }
          }
          
          // 強制觸發 Svelte 響應式更新
          const newTrendAnalysis = {};
          for (const key in formData.trend_analysis) {
            newTrendAnalysis[key] = { ...formData.trend_analysis[key] };
          }
          formData = {
            ...formData,
            trend_analysis: newTrendAnalysis
          };
          waveButtonKey++;
        };
        
        reader.readAsDataURL(file);
        break;
      }
    }
  }

  // 移除趨勢圖片
  function removeTrendImage(timeframe, imageType = 'trend') {
    if (imageType === 'signals') {
      formData.trend_analysis[timeframe].signals_image = '';
      formData.trend_analysis[timeframe].signals_originalImage = '';
    } else if (imageType === 'wave') {
      formData.trend_analysis[timeframe].wave_image = '';
      formData.trend_analysis[timeframe].wave_originalImage = '';
    } else {
      formData.trend_analysis[timeframe].image = '';
      formData.trend_analysis[timeframe].originalImage = '';
    }
    
    // 強制觸發 Svelte 響應式更新
    const newTrendAnalysis = {};
    for (const key in formData.trend_analysis) {
      newTrendAnalysis[key] = { ...formData.trend_analysis[key] };
    }
    formData = {
      ...formData,
      trend_analysis: newTrendAnalysis
    };
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
      if (context.type === 'trend') {
        enlargedOriginalImage = formData.trend_analysis[context.key]?.originalImage || imageSrc;
      } else if (context.type === 'signals') {
        enlargedOriginalImage = formData.trend_analysis[context.key]?.signals_originalImage || imageSrc;
      } else if (context.type === 'wave') {
        enlargedOriginalImage = formData.trend_analysis[context.key]?.wave_originalImage || imageSrc;
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

    if (type === 'trend') {
      formData.trend_analysis[key] = {
        ...formData.trend_analysis[key],
        image: annotatedImageSrc
      };
    } else if (type === 'signals') {
      formData.trend_analysis[key] = {
        ...formData.trend_analysis[key],
        signals_image: annotatedImageSrc
      };
    } else if (type === 'wave') {
      formData.trend_analysis[key] = {
        ...formData.trend_analysis[key],
        wave_image: annotatedImageSrc
      };
    }
    
    // 強制觸發 Svelte 響應式更新
    const newTrendAnalysis = {};
    for (const key in formData.trend_analysis) {
      newTrendAnalysis[key] = { ...formData.trend_analysis[key] };
    }
    formData = {
      ...formData,
      trend_analysis: newTrendAnalysis
    };
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

      <!-- 市場時段 -->
      <div class="form-group">
        <label>市場時段</label>
        <div class="market-session-options">
          <label class="session-option" class:active={formData.market_session === 'asian'}>
            <input type="radio" bind:group={formData.market_session} value="asian" />
            <span>亞盤</span>
          </label>
          <label class="session-option" class:active={formData.market_session === 'european'}>
            <input type="radio" bind:group={formData.market_session} value="european" />
            <span>歐盤</span>
          </label>
          <label class="session-option" class:active={formData.market_session === 'us'}>
            <input type="radio" bind:group={formData.market_session} value="us" />
            <span>美盤</span>
          </label>
        </div>
      </div>

      <!-- 備註 -->
      <div class="form-group">
        <label for="notes">備註</label>
        <textarea
          id="notes"
          class="form-control"
          bind:value={formData.notes}
          rows="3"
          placeholder="今日盤面重點、注意事項..."
        ></textarea>
      </div>
    </div>

    <!-- 當前各時區趨勢 -->
    <div class="form-group trend-analysis-section">
      <label class="trend-label">📊 當前各時區趨勢</label>
      <div class="trend-grid">
        {#each ['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'] as timeframe}
          <div
            class="trend-item"
            tabindex="0"
            on:paste={(e) => handleTrendImagePaste(e, timeframe)}
            on:click={(e) => {
              if (!e.target.closest('.trend-options')) {
                e.currentTarget.focus();
              }
            }}
          >
            <label class="timeframe-label">{timeframe}</label>
            
            <!-- 多空選擇 -->
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

            <!-- 達人訊號選擇 -->
            {#if formData.trend_analysis[timeframe].direction}
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
                {#if formData.trend_analysis[timeframe].signals_image}
                  <div class="trend-image-preview" on:click|stopPropagation={() => enlargeImage(formData.trend_analysis[timeframe].signals_image, `${timeframe} 達人訊號圖`, { type: 'signals', key: timeframe })}>
                    <img 
                      src={formData.trend_analysis[timeframe].signals_image} 
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
                    on:paste|preventDefault|stopPropagation={(e) => handleTrendImagePaste(e, timeframe, 'signals')}
                    on:click|stopPropagation={(e) => e.target.focus()}
                    role="textbox"
                  >
                    📋 Ctrl+V 貼上達人訊號圖片
                  </div>
                {/if}
              </div>
            {/if}

            <!-- 波浪浪數選擇 -->
            {#if formData.trend_analysis[timeframe].direction}
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
                {#if formData.trend_analysis[timeframe].wave_image}
                  <div class="trend-image-preview" on:click|stopPropagation={() => enlargeImage(formData.trend_analysis[timeframe].wave_image, `${timeframe} 波浪圖`, { type: 'wave', key: timeframe })}>
                    <img 
                      src={formData.trend_analysis[timeframe].wave_image} 
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
                    on:paste|preventDefault|stopPropagation={(e) => handleTrendImagePaste(e, timeframe, 'wave')}
                    on:click|stopPropagation={(e) => e.target.focus()}
                    role="textbox"
                  >
                    📋 Ctrl+V 貼上波浪圖片
                  </div>
                {/if}
              </div>
            {/if}
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

  /* 市場時段選項 */
  .market-session-options {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .session-option {
    display: inline-flex;
    align-items: center;
    padding: 0.75rem 1.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
  }

  .session-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .session-option.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  .session-option input[type="radio"] {
    display: none;
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

  .trend-option input[type="radio"] {
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

