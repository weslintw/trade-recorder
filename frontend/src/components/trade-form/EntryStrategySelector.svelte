<script>
  export let formData = {};

  $: availableTimeframes = [
    { label: '1分', value: 'M1' },
    { label: '5分', value: 'M5' },
    { label: '15分', value: 'M15' },
    { label: '1小時', value: 'H1' },
    { label: '4小時', value: 'H4' },
    { label: '天', value: 'D1' },
    ...(formData.entry_strategy === 'legend' ? [{ label: '超k', value: 'SuperK' }] : [])
  ];

  /* 如果切換離傳奇模式，重置「超k」選擇 */
  $: if (formData.entry_strategy !== 'legend' && formData.entry_timeframe === 'SuperK') {
    formData.entry_timeframe = '';
  }
</script>

<div class="form-row timeframe-trend-row">
  <div class="form-group">
    <label>🎯 進場種類</label>
    <div class="strategy-options mini">
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
  </div>

  <div class="form-group">
    <label>🕒 進場時區</label>
    <div class="timeframe-options">
      {#each availableTimeframes as tf}
        <button
          type="button"
          class="timeframe-btn"
          class:active={formData.entry_timeframe === tf.value}
          on:click={() => (formData.entry_timeframe = tf.value)}
        >
          {tf.label}
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .form-row {
      display: flex;
      gap: 1.5rem;
      margin-bottom: 2rem;
  }
  
  .form-group {
      flex: 1;
      display: flex;
      flex-direction: column;
  }
  
  label {
      display: block;
      margin-bottom: 0.5rem;
      font-weight: 600;
      color: #4a5568;
      font-size: 0.95rem;
  }

  .timeframe-trend-row {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    margin-bottom: 1.5rem;
    align-items: flex-end; /* 讓內容底部對齊 */
  }

  .strategy-options.mini {
    display: flex;
    gap: 0.5rem;
  }

  .strategy-options.mini .strategy-option {
    padding: 0.35rem 0.75rem;
    font-size: 0.9rem;
    min-width: auto;
    flex: 1;
    justify-content: center;
  }

  .strategy-option {
    position: relative;
    display: flex;
    /* flex-direction: column; */ /* 原樣式可能有，但 mini 樣式可能覆蓋 */
    align-items: center;
    cursor: pointer;
    padding: 1rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    transition: all 0.2s ease;
  }

  .strategy-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .strategy-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
  }

  .strategy-option input[type='radio'] {
    position: absolute;
    opacity: 0;
  }

  .strategy-name {
    font-weight: 600;
    color: #2d3748;
    font-size: 1rem;
  }

  .strategy-option.active .strategy-name {
    color: #667eea;
  }

  .timeframe-options {
    display: flex;
    gap: 2px;
    background: #1a1a1a;
    padding: 4px;
    border-radius: 8px;
    width: fit-content;
  }

  .timeframe-btn {
    padding: 6px 10px;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: #888;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: fit-content;
  }

  .timeframe-btn:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
  }

  .timeframe-btn.active {
    background: #333;
    color: #60a5fa; /* 藍色亮顯，符合交易軟體習慣 */
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }
</style>
