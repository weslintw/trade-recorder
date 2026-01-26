<script>
  export let formData = {};

  $: availableTimeframes = [
    { label: '1分', value: 'M1' },
    { label: '5分', value: 'M5' },
    { label: '15分', value: 'M15' },
    { label: '1小時', value: 'H1' },
    { label: '4小時', value: 'H4' },
    { label: '天', value: 'D1' },
    ...(formData.entry_strategy === 'legend' ? [{ label: '超k', value: 'SuperK' }] : []),
  ];

  /* 如果切換到傳奇模式，預設選擇「超k」 */
  /* 監聽策略變更並處理副作用 */
  let lastStrategy = formData.entry_strategy;
  $: if (formData.entry_strategy !== lastStrategy) {
    if (formData.entry_strategy === 'legend') {
      formData.entry_timeframe = 'SuperK';
    } else if (formData.entry_timeframe === 'SuperK') {
      formData.entry_timeframe = '';
    }
    lastStrategy = formData.entry_strategy;
  }
</script>

<div class="form-row timeframe-trend-row">
  <div class="form-group">
    <label>🎯 進場種類</label>
    <div class="strategy-options mini">
      <!-- svelte-ignore a11y-click-events-have-key-events (Radio input handles a11y, but we intercept click for toggle) -->
      <label 
        class="strategy-option" 
        class:active={formData.entry_strategy === 'expert'}
        on:click|preventDefault={() => {
            formData.entry_strategy = formData.entry_strategy === 'expert' ? '' : 'expert';
        }}
      >
        <span class="strategy-name">達人</span>
      </label>
      <label 
        class="strategy-option" 
        class:active={formData.entry_strategy === 'elite'}
        on:click|preventDefault={() => {
            formData.entry_strategy = formData.entry_strategy === 'elite' ? '' : 'elite';
        }}
      >
        <span class="strategy-name">菁英</span>
      </label>
      <label 
        class="strategy-option" 
        class:active={formData.entry_strategy === 'legend'}
        on:click|preventDefault={() => {
            formData.entry_strategy = formData.entry_strategy === 'legend' ? '' : 'legend';
        }}
      >
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
    color: var(--text-main);
    font-size: 0.95rem;
  }

  .timeframe-trend-row {
    background: var(--nav-group-bg);
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid var(--border-color);
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
    border: 2px solid var(--border-color);
    border-radius: 8px;
    background: var(--card-bg);
    transition: all 0.2s ease;
  }

  .strategy-option:hover {
    border-color: var(--primary);
    background: var(--bg-main);
  }

  .strategy-option.active {
    border-color: var(--primary);
    background: var(--card-bg);
    box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
  }

  .strategy-option input[type='radio'] {
    position: absolute;
    opacity: 0;
  }

  .strategy-name {
    font-weight: 600;
    color: var(--text-main);
    font-size: 1rem;
  }

  .strategy-option.active .strategy-name {
    color: #667eea;
  }

  .timeframe-options {
    display: flex;
    gap: 4px;
    background: #f1f5f9;
    padding: 4px;
    border-radius: 10px;
    width: fit-content;
    border: 1px solid #e2e8f0;
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.02);
  }

  :global(body.dark-mode) .timeframe-options {
    background: #1a1a1a;
    border-color: #334155;
    gap: 2px;
    padding: 2px;
  }

  .timeframe-btn {
    padding: 6px 12px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: #64748b;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    white-space: nowrap;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: fit-content;
  }

  :global(body.dark-mode) .timeframe-btn {
    padding: 6px 10px;
    color: #888;
    border-radius: 6px;
  }

  .timeframe-btn:hover {
    color: #1e293b;
    background: rgba(255, 255, 255, 0.5);
  }

  :global(body.dark-mode) .timeframe-btn:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
  }

  .timeframe-btn.active {
    background: white;
    color: #4f46e5;
    box-shadow:
      0 2px 6px rgba(0, 0, 0, 0.08),
      0 1px 2px rgba(0, 0, 0, 0.05);
  }

  :global(body.dark-mode) .timeframe-btn.active {
    background: #333;
    color: #60a5fa;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
  }
</style>
