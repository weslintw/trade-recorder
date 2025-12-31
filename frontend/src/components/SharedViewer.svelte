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
      if (res.data && res.data.type) {
        sharedData = res.data;
      } else {
        error = '後端回傳資料格式不正確';
      }
    } catch (e) {
      console.error('[SharedViewer] Fetch Error:', e);
      error = e.response?.data?.error || '查無此分享內容或連結已過期';
    } finally {
      loading = false;
    }
  });

  function formatDate(dateStr) {
    if (!dateStr) return '';
    try {
      return new Date(dateStr).toLocaleString('zh-TW', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
      });
    } catch (e) {
      return dateStr;
    }
  }

  function parseJSON(str, defaultValue = null) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  // 達人策略檢查項翻譯
  const expertSignals = {
    'item_ma_flow': 'MA 流向',
    'item_ma_space': 'MA 空間',
    'item_signal_confirm': '訊號確認',
    'item_risk_ratio': '風報比合理'
  };

  // 菁英策略檢查項翻譯
  const eliteChecklist = {
    'item_trend': '順勢',
    'item_zone_s_d': 'Zone (S/D)',
    'item_f_b_break': '假突破 或 真突破/回踩',
    'item_space': '空間',
    'item_signal': '訊號'
  };

  // 傳奇策略檢查項翻譯
  const legendChecklist = {
    'item_trend': '順勢',
    'item_zone_s_d': 'Zone (S/D)',
    'item_618_786': '王者出現回調618或786',
    'item_che': '大時區破"測"破',
    'item_de': '達人整理段訊號'
  };
</script>

<div class="shared-view-container">
  {#if loading}
    <div class="status-box card">
      <div class="loader"></div>
      <p>正在載入分享內容...</p>
    </div>
  {:else if error}
    <div class="status-box card error">
      <div class="error-icon">⚠️</div>
      <h2>存取失敗</h2>
      <p>{error}</p>
      <a href="/" class="btn btn-primary">回到首頁</a>
    </div>
  {:else if sharedData}
    <div class="shared-content">
      <div class="public-badge">👁️ 唯讀分享模式</div>
      
      {#if sharedData.type === 'trade' && sharedData.data}
        {@const trade = sharedData.data}
        {@const checklist = parseJSON(trade.entry_checklist, {})}
        {@const signals = parseJSON(trade.entry_signals, [])}
        {@const patterns = parseJSON(trade.entry_pattern, [])}
        
        <div class="trade-detail-view card">
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-tag">{trade.symbol || '---'}</span>
              <span class="side-tag {trade.side || ''}">{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span>
              <h1>交易紀錄詳情</h1>
            </div>
            <div class="pnl-section">
              {#if trade.pnl !== undefined && trade.pnl !== null}
                <div class="pnl-value {trade.pnl >= 0 ? 'profit' : 'loss'}">
                  {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl).toFixed(2)}
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
              <span class="value-highlight">{trade.entry_price || '---'}</span>
            </div>
            <div class="info-item">
              <label>手數</label>
              <span>{trade.lot_size || '---'}</span>
            </div>
            {#if trade.exit_time}
              <div class="info-item">
                <label>平倉時間</label>
                <span>{formatDate(trade.exit_time)}</span>
              </div>
              <div class="info-item">
                <label>平倉價格</label>
                <span class="value-highlight">{trade.exit_price || '---'}</span>
              </div>
            {/if}
            {#if trade.entry_strategy}
              <div class="info-item">
                <label>交易策略</label>
                <span class="strategy-badge {trade.entry_strategy}">{getStrategyLabel(trade.entry_strategy)}</span>
              </div>
            {/if}
          </div>

          <!-- 策略分析詳情 -->
          {#if trade.entry_strategy}
            <div class="section-box strategy-box">
              <h3>🔍 策略分析詳情</h3>
              <div class="strategy-details">
                <!-- 檢查清單 -->
                {#if Object.keys(checklist).length > 0}
                  <div class="detail-group">
                    <label>檢查清單：</label>
                    <div class="tags-row">
                      {#each Object.entries(checklist) as [id, checked]}
                        {#if checked}
                          {@const label = trade.entry_strategy === 'expert' ? expertSignals[id] : 
                                          trade.entry_strategy === 'elite' ? eliteChecklist[id] : 
                                          legendChecklist[id]}
                          <span class="check-tag">✅ {label || id}</span>
                        {/if}
                      {/each}
                    </div>
                  </div>
                {/if}

                <!-- 進場訊號 (JSON Array) -->
                {#if signals.length > 0}
                  <div class="detail-group">
                    <label>進場訊號：</label>
                    <div class="tags-row">
                      {#each signals as signal}
                        <span class="signal-tag-item">✨ {typeof signal === 'object' ? (signal.name || signal.id || JSON.stringify(signal)) : signal}</span>
                      {/each}
                    </div>
                  </div>
                {/if}

                <!-- 策略截圖 -->
                <div class="strategy-images">
                  {#if trade.entry_strategy_image}
                    <div class="img-preview-box">
                      <p>進場觀察圖：</p>
                      <img src={trade.entry_strategy_image} alt="Strategy Observation" />
                    </div>
                  {/if}

                  {#if trade.entry_strategy === 'legend'}
                    {#if trade.legend_king_image}
                      <div class="img-preview-box">
                        <p>👑 王者回調 ({trade.legend_king_htf})：</p>
                        <img src={trade.legend_king_image} alt="King Callback" />
                      </div>
                    {/if}
                    {#if trade.legend_htf_image}
                      <div class="img-preview-box">
                        <p>🌊 大時區破測破 ({trade.legend_htf})：</p>
                        <img src={trade.legend_htf_image} alt="HTF Breakout" />
                      </div>
                    {/if}
                  {/if}

                  {#if trade.entry_strategy === 'elite'}
                    {#each patterns as pattern}
                      {#if pattern.image}
                        <div class="img-preview-box">
                          <p>🎯 {pattern.name} 樣態圖：</p>
                          <img src={pattern.image} alt={pattern.name} />
                        </div>
                      {/if}
                    {/each}
                  {/if}
                </div>
              </div>
            </div>
          {/if}

          {#if trade.notes}
            <div class="section-box">
              <h3>📝 交易復盤筆記</h3>
              <div class="notes-content ql-editor">{@html trade.notes}</div>
            </div>
          {/if}

          {#if trade.exit_reason}
            <div class="section-box">
              <h3>🎯 平倉理由</h3>
              <div class="notes-content ql-editor">{@html trade.exit_reason}</div>
            </div>
          {/if}

          {#if trade.images && trade.images.length > 0}
            <div class="section-box">
              <h3>🖼️ 圖表截圖 (Gallery)</h3>
              <div class="image-gallery">
                {#each trade.images as img}
                  {#if img && img.image_path}
                    <div class="image-card">
                      <img src={imagesAPI.getUrl(img.image_path)} alt="Trade Chart" />
                      {#if img.image_type}
                        <span class="image-type-label">
                          {img.image_type === 'entry' ? '📍 進場' : img.image_type === 'exit' ? '🎯 平倉' : '📷 觀察'}
                        </span>
                      {/if}
                    </div>
                  {/if}
                {/each}
              </div>
            </div>
          {/if}
        </div>

      {:else if sharedData.type === 'plan' && sharedData.data}
        {@const plan = sharedData.data}
        {@const trendAnalysis = parseJSON(plan.trend_analysis, {})}
        
        <div class="plan-detail-view card">
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-tag">{plan.symbol || '---'}</span>
              <h1>盤面規劃分享</h1>
            </div>
            <div class="date-section">
              <span class="plan-date-tag">📅 {plan.plan_date ? plan.plan_date.slice(0, 10) : ''}</span>
            </div>
          </div>

          <div class="section-box">
            <h3>📝 規劃備註</h3>
            <div class="notes-content ql-editor">{@html plan.notes || '尚無備註內容'}</div>
          </div>
          
          {#each ['asian', 'european', 'us'] as session}
            {#if trendAnalysis[session]}
              {@const sessionData = trendAnalysis[session]}
              <div class="session-block {session}">
                <h4>时段：{session === 'asian' ? '🌏 亞盤' : session === 'european' ? '🌍 歐盤' : '🌎 美盤'}</h4>
                {#if sessionData.notes}
                  <p class="session-notes">{sessionData.notes}</p>
                {/if}
              </div>
            {/if}
          {/each}
        </div>
      {:else}
         <div class="status-box card error">
            <p>資料格式不正確或類型未知</p>
         </div>
      {/if}
    </div>
  {:else}
    <div class="status-box card">
      <p>無內容可顯示</p>
    </div>
  {/if}
</div>

<style>
  .shared-view-container {
    max-width: 850px;
    margin: 3rem auto;
    padding: 0 1.25rem;
    min-height: 400px;
  }

  .public-badge {
    background: #f8fafc;
    color: #64748b;
    padding: 0.5rem 1.25rem;
    border-radius: 99px;
    font-size: 0.8rem;
    font-weight: 700;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
    border: 1px solid #e2e8f0;
  }

  .card {
    background: white;
    border-radius: 1.5rem;
    padding: 2.5rem;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05);
    border: 1px solid #f1f5f9;
  }

  .view-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 2.5rem;
    padding-bottom: 2rem;
    border-bottom: 1px solid #f1f5f9;
    flex-wrap: wrap;
    gap: 1.5rem;
  }

  .title-section h1 {
    font-size: 1.75rem;
    font-weight: 800;
    margin: 0.75rem 0 0 0;
    color: #0f172a;
  }

  .symbol-tag {
    background: #4f46e5;
    color: white;
    padding: 0.375rem 0.8125rem;
    border-radius: 8px;
    font-weight: 800;
    font-size: 0.875rem;
  }

  .side-tag {
    padding: 0.375rem 0.8125rem;
    border-radius: 8px;
    font-weight: 700;
  }
  .side-tag.long { background: #fee2e2; color: #991b1b; }
  .side-tag.short { background: #dcfce7; color: #166534; }

  .pnl-value {
    font-size: 2.5rem;
    font-weight: 900;
  }
  .pnl-value.profit { color: #10b981; }
  .pnl-value.loss { color: #ef4444; }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 2rem;
    margin-bottom: 3rem;
  }

  .info-item label {
    display: block;
    font-size: 0.75rem;
    color: #64748b;
    margin-bottom: 0.5rem;
    font-weight: 700;
    text-transform: uppercase;
  }

  .info-item span {
    font-size: 1.125rem;
    font-weight: 700;
    color: #1e293b;
  }

  .value-highlight {
    color: #4f46e5 !important;
  }

  .section-box {
    margin-bottom: 3rem;
  }

  .section-box h3 {
    font-size: 1.25rem;
    font-weight: 800;
    color: #1e293b;
    margin-bottom: 1.25rem;
    border-left: 4px solid #4f46e5;
    padding-left: 0.75rem;
  }

  .strategy-box {
    background: #fcfcfd;
    padding: 1.5rem;
    border-radius: 1rem;
    border: 1px solid #f1f5f9;
  }

  .detail-group {
    margin-bottom: 1.25rem;
  }

  .detail-group label {
    font-size: 0.85rem;
    font-weight: 700;
    color: #64748b;
    margin-bottom: 0.5rem;
    display: block;
  }

  .tags-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .check-tag, .signal-tag-item {
    background: white;
    padding: 0.35rem 0.75rem;
    border-radius: 6px;
    font-size: 0.85rem;
    font-weight: 600;
    border: 1px solid #e2e8f0;
    color: #334155;
  }

  .img-preview-box {
    margin-top: 1.5rem;
    background: white;
    padding: 1rem;
    border-radius: 12px;
    border: 1px solid #e2e8f0;
  }

  .img-preview-box p {
    font-size: 0.85rem;
    font-weight: 700;
    margin-bottom: 0.75rem;
    color: #475569;
  }

  .img-preview-box img {
    width: 100%;
    border-radius: 8px;
    display: block;
  }

  .notes-content {
    background: #f8fafc;
    padding: 1.75rem;
    border-radius: 1rem;
    line-height: 1.7;
    color: #334155;
    border: 1px solid #f1f5f9;
  }

  /* Quill Editor Style Reset for shared view */
  .ql-editor :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 12px;
    margin: 1rem 0;
    box-shadow: 0 4px 12px rgba(0,0,0,0.08);
  }

  .ql-editor :global(p) {
    margin-bottom: 1rem;
  }

  .image-gallery {
    display: grid;
    grid-template-columns: 1fr;
    gap: 2.5rem;
  }

  .image-card {
    position: relative;
    border-radius: 1.25rem;
    overflow: hidden;
    box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.1);
  }

  .image-card img {
    width: 100%;
    display: block;
  }

  .image-type-label {
    position: absolute;
    top: 1rem;
    left: 1rem;
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(4px);
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    font-size: 0.75rem;
    font-weight: 800;
  }

  .status-box {
    text-align: center;
    padding: 5rem 2rem;
  }

  .loader {
    width: 48px;
    height: 48px;
    border: 5px solid #f1f5f9;
    border-top: 5px solid #4f46e5;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto 1.5rem;
  }

  @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

  .strategy-badge {
    padding: 0.25rem 0.75rem;
    border-radius: 6px;
    color: white;
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .strategy-badge.expert { background: #10b981; }
  .strategy-badge.elite { background: #3b82f6; }
  .strategy-badge.legend { background: #f59e0b; }

  .session-block {
    margin-top: 1.5rem;
    padding: 1.25rem;
    border-radius: 12px;
    border-left: 5px solid #e2e8f0;
    background: #f8fafc;
  }
  .session-block.asian { border-left-color: #3b82f6; }
  .session-block.european { border-left-color: #f59e0b; }
  .session-block.us { border-left-color: #ef4444; }
</style>
