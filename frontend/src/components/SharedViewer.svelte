<script>
  import { onMount } from 'svelte';
  import { sharesAPI, imagesAPI } from '../lib/api';
  import { getStrategyLabel } from '../lib/utils';
  
  export let token = '';

  let loading = true;
  let error = null;
  let sharedData = null; // { type: 'trade'|'plan', data: ... }

  onMount(async () => {
    try {
      const res = await sharesAPI.getPublic(token);
      sharedData = res.data;
    } catch (e) {
      console.error(e);
      error = e.response?.data?.error || '查無此分享內容或連結已過期';
    } finally {
      loading = false;
    }
  });

  function formatDate(dateStr) {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    });
  }
</script>

<div class="shared-view-container">
  {#if loading}
    <div class="loading-box card">
      <div class="loader"></div>
      <p>正在載入分享內容...</p>
    </div>
  {:else if error}
    <div class="error-box card">
      <div class="error-icon">⚠️</div>
      <h2>存取失敗</h2>
      <p>{error}</p>
      <a href="/" class="btn btn-primary">回到首頁</a>
    </div>
  {:else if sharedData}
    <div class="shared-content">
      <div class="public-badge">👁️ 唯讀分享模式</div>
      
      {#if sharedData.type === 'trade'}
        {@const trade = sharedData.data}
        <div class="trade-detail-view card">
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-badge">{trade.symbol}</span>
              <span class="side-badge {trade.side}">{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span>
              <h1>交易紀錄詳情</h1>
            </div>
            <div class="pnl-section">
              {#if trade.pnl !== null}
                <div class="pnl-value {trade.pnl >= 0 ? 'profit' : 'loss'}">
                  {trade.pnl >= 0 ? '+' : ''}{trade.pnl.toFixed(2)}
                </div>
              {/if}
            </div>
          </div>

          <div class="info-grid">
            <div class="info-item">
              <label>進場時間</label>
              <span>{formatDate(trade.entry_time)}</span>
            </div>
            <div class="info-item">
              <label>進場價格</label>
              <span>{trade.entry_price}</span>
            </div>
            <div class="info-item">
              <label>手數</label>
              <span>{trade.lot_size}</span>
            </div>
            {#if trade.exit_time}
              <div class="info-item">
                <label>平倉時間</label>
                <span>{formatDate(trade.exit_time)}</span>
              </div>
              <div class="info-item">
                <label>平倉價格</label>
                <span>{trade.exit_price}</span>
              </div>
            {/if}
            {#if trade.entry_strategy}
              <div class="info-item">
                <label>使用策略</label>
                <span class="strategy-label {trade.entry_strategy}">{getStrategyLabel(trade.entry_strategy)}</span>
              </div>
            {/if}
          </div>

          {#if trade.notes}
            <div class="section-box">
              <h3>📝 交易筆記</h3>
              <div class="notes-content">{trade.notes}</div>
            </div>
          {/if}

          {#if trade.images && trade.images.length > 0}
            <div class="section-box">
              <h3>🖼️ 圖表截圖</h3>
              <div class="image-gallery">
                {#each trade.images as img}
                  <div class="image-item">
                    <img src={imagesAPI.getUrl(img.image_path)} alt="Trade Chart" />
                    {#if img.description}<p class="img-desc">{img.description}</p>{/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>

      {:else if sharedData.type === 'plan'}
        {@const plan = sharedData.data}
        <div class="plan-detail-view card">
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-badge">{plan.symbol}</span>
              <h1>盤面規劃分享</h1>
            </div>
            <div class="date-section">
              <span class="plan-date">📅 {plan.plan_date.slice(0, 10)}</span>
            </div>
          </div>

          <div class="section-box">
            <h3>📝 規劃內容</h3>
            <div class="notes-content">{plan.notes || '無備註內容'}</div>
          </div>
          
          <!-- 這裡可以根據需要顯示更多 Plan 的細節 (如 trend_analysis) -->
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .shared-view-container {
    max-width: 900px;
    margin: 2rem auto;
    padding: 0 1rem;
  }

  .public-badge {
    background: #f1f5f9;
    color: #64748b;
    padding: 0.5rem 1rem;
    border-radius: 50px;
    font-size: 0.85rem;
    font-weight: 700;
    display: inline-block;
    margin-bottom: 1rem;
    border: 1px solid #e2e8f0;
  }

  .view-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 2rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid #f1f5f9;
  }

  .title-section h1 {
    font-size: 1.5rem;
    margin: 0.5rem 0 0 0;
    color: #1e293b;
  }

  .symbol-badge {
    background: #4f46e5;
    color: white;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    font-weight: 800;
    font-size: 0.9rem;
  }

  .side-badge {
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    font-weight: 700;
    font-size: 0.9rem;
    margin-left: 0.5rem;
  }
  .side-badge.long { background: #fee2e2; color: #991b1b; }
  .side-badge.short { background: #dcfce7; color: #166534; }

  .pnl-value {
    font-size: 2rem;
    font-weight: 800;
  }
  .pnl-value.profit { color: #10b981; }
  .pnl-value.loss { color: #ef4444; }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2.5rem;
  }

  .info-item label {
    display: block;
    font-size: 0.85rem;
    color: #64748b;
    margin-bottom: 0.25rem;
    font-weight: 600;
  }

  .info-item span {
    font-size: 1.1rem;
    font-weight: 700;
    color: #1e293b;
  }

  .section-box {
    margin-bottom: 2.5rem;
  }

  .section-box h3 {
    font-size: 1.1rem;
    color: #475569;
    margin-bottom: 1rem;
  }

  .notes-content {
    background: #f8fafc;
    padding: 1.5rem;
    border-radius: 12px;
    white-space: pre-wrap;
    line-height: 1.6;
    color: #334155;
  }

  .image-gallery {
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .image-item img {
    width: 100%;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);
  }

  .img-desc {
    margin-top: 0.5rem;
    font-size: 0.9rem;
    color: #64748b;
    text-align: center;
  }

  .loading-box, .error-box {
    text-align: center;
    padding: 4rem 2rem;
  }

  .error-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .loader {
    width: 40px;
    height: 40px;
    border: 4px solid #f3f3f3;
    border-top: 4px solid #4f46e5;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 1.5rem;
  }

  @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

  .strategy-label {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    color: white;
    font-size: 0.85rem;
  }
  .strategy-label.expert { background: #059669; }
  .strategy-label.elite { background: #1e3a8a; }
  .strategy-label.legend { background: #78350f; }
</style>
