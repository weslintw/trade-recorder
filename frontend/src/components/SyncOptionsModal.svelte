<script>
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  export let show = false;
  
  let syncOption = 'default'; // 'default', 'all', 'date'
  let syncStartDate = new Date().toISOString().split('T')[0];

  function handleSync() {
    let options = {};
    if (syncOption === 'all') {
      options.sync_all = true;
    } else if (syncOption === 'date') {
      options.from_date = syncStartDate;
    }
    dispatch('sync', options);
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={() => dispatch('close')} role="presentation">
    <div class="modal card">
      <h2>cTrader 同步選項</h2>
      <p class="desc">請選擇您想要同步的交易紀錄範圍：</p>
      
      <div class="options-list">
        <label class="option-item" class:active={syncOption === 'default'}>
          <input type="radio" bind:group={syncOption} value="default" />
          <div class="option-content">
            <span class="title">近期同步 (120天)</span>
            <span class="detail">僅同步最近 4 個月的交易資料，速度較快。</span>
          </div>
        </label>

        <label class="option-item" class:active={syncOption === 'all'}>
          <input type="radio" bind:group={syncOption} value="all" />
          <div class="option-content">
            <span class="title">全部同步 (所有資料)</span>
            <span class="detail">同步該帳號所有的歷史交易紀錄。</span>
          </div>
        </label>

        <label class="option-item" class:active={syncOption === 'date'}>
          <input type="radio" bind:group={syncOption} value="date" />
          <div class="option-content">
            <span class="title">自訂日期</span>
            <span class="detail">同步從指定日期至今的交易資料。</span>
            {#if syncOption === 'date'}
              <input type="date" class="form-control date-picker" bind:value={syncStartDate} />
            {/if}
          </div>
        </label>
      </div>

      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={() => dispatch('close')}>取消</button>
        <button class="btn btn-primary" on:click={handleSync}>開始同步</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }
  .modal {
    width: 90%;
    max-width: 450px;
    padding: 2rem;
    border-radius: 16px;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
    background: white;
  }

  :global(body.dark-mode) .modal {
    background: #1e293b;
    border: 1px solid #334155;
  }

  h2 { margin-top: 0; font-size: 1.5rem; color: var(--text-color); }
  .desc { color: var(--text-muted); margin-bottom: 1.5rem; }
  
  .options-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-bottom: 2rem;
  }
  .option-item {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    border: 1px solid var(--border-color);
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s;
    background: var(--card-bg);
  }
  .option-item:hover { background: var(--bg-secondary); }
  .option-item.active {
    border-color: #4338ca;
    background: rgba(67, 56, 202, 0.05);
  }

  :global(body.dark-mode) .option-item.active {
    border-color: #6366f1;
    background: rgba(99, 102, 241, 0.1);
  }

  .option-content { display: flex; flex-direction: column; gap: 0.25rem; flex: 1; }
  .title { font-weight: 600; color: var(--text-color); }
  .detail { font-size: 0.85rem; color: var(--text-muted); }
  
  .date-picker { 
    margin-top: 0.5rem;
    padding: 0.5rem;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--input-bg);
    color: var(--text-color);
    width: 100%;
  }

  .modal-actions { display: flex; justify-content: flex-end; gap: 1rem; }

  input[type="radio"] {
    width: 1.25rem;
    height: 1.25rem;
    margin-top: 0.25rem;
    cursor: pointer;
  }
</style>
