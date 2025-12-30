<script>
  import { fade, scale } from 'svelte/transition';

  export let show = false;
  export let trades = []; // 這些應該是 type='observation' 的單子
  export let currentSymbol = '';
  export let onConfirm = () => {};
  export let onClose = () => {};

  let selectedTradeId = null;

  // 預設選中第一筆
  $: if (trades && trades.length > 0 && !selectedTradeId) {
    selectedTradeId = trades[0].id;
  }

  function handleConfirm() {
    const trade = trades.find(t => t.id === selectedTradeId);
    if (trade) {
      onConfirm(trade);
      onClose();
    }
  }

  function formatDate(dateStr) {
    return new Date(dateStr).toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  // 格式化策略名稱
  function formatStrategy(strategy) {
    const map = {
      expert: '達人',
      elite: '菁英',
      legend: '傳奇'
    };
    return map[strategy] || strategy || '未指定';
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={onClose} transition:fade={{ duration: 200 }}>
    <div class="modal-card" transition:scale={{ start: 0.95, duration: 200 }}>
      <div class="modal-header">
        <h3>📋 從觀察單選擇併入</h3>
        <button class="close-btn" on:click={onClose}>&times;</button>
      </div>

      <div class="modal-body">
        {#if trades.length === 0}
            <div class="empty-state">
                <p>找不到同品種 ({currentSymbol}) 的觀察單。</p>
            </div>
        {:else}
            <p class="description">請選擇要併入的觀察單 ({currentSymbol})：</p>
            <div class="trade-list">
            {#each trades as trade}
                <label class="trade-item" class:selected={selectedTradeId === trade.id}>
                <input
                    type="radio"
                    name="selectedTrade"
                    value={trade.id}
                    bind:group={selectedTradeId}
                />
                <div class="trade-info">
                    <div class="trade-meta">
                        <span class="trade-date">{formatDate(trade.entry_time)}</span>
                        <span class="badge {trade.side}">{trade.side === 'long' ? '多' : '空'}</span>
                        <span class="badge strategy">{formatStrategy(trade.entry_strategy)}</span>
                    </div>
                    {#if trade.entry_reason}
                        <div class="trade-reason" title={trade.entry_reason}>
                           <strong>進場理由:</strong> {trade.entry_reason.replace(/<[^>]*>?/gm, '').slice(0, 50)}...
                        </div>
                    {/if}
                    <div class="trade-tags">
                        {#if trade.tags && trade.tags.length > 0}
                           {#each trade.tags as tag}
                              <span class="tag">#{tag.name}</span>
                           {/each}
                        {/if}
                    </div>
                </div>
                </label>
            {/each}
            </div>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={onClose}>取消</button>
        <button class="btn btn-primary" on:click={handleConfirm} disabled={!selectedTradeId || trades.length === 0}>
          確認併入
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }

  .modal-card {
    background: white;
    width: 90%;
    max-width: 600px;
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    max-height: 85vh;
  }

  .modal-header {
    padding: 1.2rem;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.25rem;
    color: #2d3748;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: #a0aec0;
    padding: 0;
    line-height: 1;
  }

  .close-btn:hover {
    color: #4a5568;
  }

  .modal-body {
    padding: 1.5rem;
    overflow-y: auto;
  }

  .empty-state {
      text-align: center;
      color: #718096;
      padding: 2rem;
  }

  .description {
    margin-bottom: 1rem;
    color: #718096;
    font-size: 0.95rem;
  }

  .trade-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .trade-item {
    display: flex;
    align-items: flex-start;
    padding: 1rem;
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .trade-item:hover {
    border-color: #cbd5e0;
    background: #f7fafc;
  }

  .trade-item.selected {
    border-color: #4299e1;
    background: #ebf8ff;
  }

  .trade-item input[type="radio"] {
    margin-top: 0.3rem;
    margin-right: 0.75rem;
  }

  .trade-info {
    flex: 1;
  }

  .trade-meta {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.5rem;
      flex-wrap: wrap;
  }

  .trade-date {
    font-weight: 700;
    color: #2d3748;
    font-size: 0.9rem;
  }

  .badge {
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    font-weight: 600;
  }

  .badge.long {
      background: #fed7d7;
      color: #c53030;
  }

  .badge.short {
      background: #c6f6d5;
      color: #276749;
  }
  
  .badge.strategy {
      background: #edf2f7;
      color: #4a5568;
  }

  .trade-reason {
    font-size: 0.85rem;
    color: #4a5568;
    line-height: 1.4;
    margin-bottom: 0.4rem;
  }

  .trade-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 0.3rem;
  }
  
  .tag {
      font-size: 0.75rem;
      color: #3182ce;
      background: #ebf8ff;
      padding: 0.1rem 0.3rem;
      border-radius: 3px;
  }

  .modal-footer {
    padding: 1.2rem;
    background: #f7fafc;
    border-top: 1px solid #e2e8f0;
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    border-bottom-left-radius: 12px;
    border-bottom-right-radius: 12px;
  }

  .btn {
    padding: 0.6rem 1.2rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: background 0.2s;
  }

  .btn-primary {
    background: #4299e1;
    color: white;
  }

  .btn-primary:hover {
    background: #3182ce;
  }
  
  .btn-primary:disabled {
    background: #a0aec0;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: #e2e8f0;
    color: #4a5568;
  }

  .btn-secondary:hover {
    background: #cbd5e0;
  }
</style>
