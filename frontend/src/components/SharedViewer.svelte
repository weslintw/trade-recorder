<script>
  import { onMount } from 'svelte';
  import { sharesAPI, imagesAPI } from '../lib/api';
  import { getStrategyLabel, determineMarketSession, getMarketSessionLabel, calculateDuration } from '../lib/utils';
  import 'quill/dist/quill.snow.css';
import Sparkline from './Sparkline.svelte';
import SharedTradeDetail from './SharedTradeDetail.svelte';
import SharedPlanDetail from './SharedPlanDetail.svelte';

  export let token = '';

  let loading = true;
  let error = null;
  let sharedData = null; // { type: 'trade'|'plan'|'account'|'batch', data: ... }

  // Detail View State
  let selectedItem = null; // { type: 'trade'|'plan', data: ... }

  // Image Modal State
  let enlargedImage = null;
  let enlargedImageTitle = '';

  function openModal(src, title) {
    if (!src) return;
    enlargedImage = src;
    enlargedImageTitle = title || '圖片';
  }

  function closeModal() {
    enlargedImage = null;
    enlargedImageTitle = '';
    zoom = 1;
    offsetX = 0;
    offsetY = 0;
  }

  // Image Modal Advanced State
  let zoom = 1;
  let offsetX = 0;
  let offsetY = 0;
  let isDragging = false;
  let startX = 0;
  let startY = 0;

  function handleWheel(e) {
    if (!enlargedImage) return;
    e.preventDefault();
    const delta = e.deltaY;
    const zoomStep = 0.1;
    const prevZoom = zoom;
    
    if (delta < 0) {
      zoom = Math.min(zoom + zoomStep, 5);
    } else {
      zoom = Math.max(zoom - zoomStep, 0.5);
    }
    
    // When zooming out to <= 1, reset offsets
    if (zoom <= 1) {
      offsetX = 0;
      offsetY = 0;
    }
  }

  function handleMouseDown(e) {
    if (zoom <= 1) return;
    isDragging = true;
    startX = e.clientX - offsetX;
    startY = e.clientY - offsetY;
    e.preventDefault();
  }

  function handleMouseMove(e) {
    if (!isDragging) return;
    offsetX = e.clientX - startX;
    offsetY = e.clientY - startY;
  }

  function handleMouseUp() {
    isDragging = false;
  }


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
        hour12: false,
      });
    } catch (e) {
      return dateStr;
    }
  }

  function formatTime(dateStr) {
    if (!dateStr) return '';
    try {
      const d = new Date(dateStr);
      return d.toLocaleTimeString('zh-TW', { hour: '2-digit', minute: '2-digit', hour12: false });
    } catch (e) {
      return '';
    }
  }

  function formatDay(dateStr) {
    if (!dateStr || dateStr === 'unknown') return '日期不詳';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-TW', { month: 'long', day: 'numeric', weekday: 'short' });
  }


  function parseJSON(str, defaultValue = null) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  function lazyLoadHTML(html) {
    if (!html) return '';
    return html.replace(/<img /g, '<img loading="lazy" ');
  }

  function getTradingDate(isoString) {
    if (!isoString) return 'unknown';
    try {
      const date = new Date(isoString);
      // Format to parts in NY time to handle DST correctly
      const options = {
        timeZone: 'America/New_York',
        year: 'numeric',
        month: 'numeric',
        day: 'numeric',
        hour: 'numeric',
        hour12: false
      };
      const formatter = new Intl.DateTimeFormat('en-US', options);
      const parts = formatter.formatToParts(date);
      
      const p = {};
      parts.forEach(({type, value}) => p[type] = value);
      
      let nyYear = parseInt(p.year);
      let nyMonth = parseInt(p.month); 
      let nyDay = parseInt(p.day);
      let nyHour = parseInt(p.hour);
      if (isNaN(nyHour)) nyHour = 0;

      // Logic: If Hour >= 17 (5 PM), belongs to next trading day
      if (nyHour >= 17) {
         const d = new Date(nyYear, nyMonth - 1, nyDay);
         d.setDate(d.getDate() + 1);
         nyYear = d.getFullYear();
         nyMonth = d.getMonth() + 1;
         nyDay = d.getDate();
      }
      
      return `${nyYear}-${String(nyMonth).padStart(2, '0')}-${String(nyDay).padStart(2, '0')}`;
    } catch(e) {
      console.warn('Date parse error', e);
      return isoString.slice(0, 10);
    }
  }

  function groupDataByDate(trades, plans) {
    const groups = {};
    
    trades.forEach(t => {
      // Use trading date logic
      const date = t.entry_time ? getTradingDate(t.entry_time) : 'unknown';
      if (!groups[date]) groups[date] = { date, trades: [], plans: [] };
      groups[date].trades.push(t);
    });
    
    plans.forEach(p => {
      // Plans use their intrinsic date
      const date = p.plan_date ? p.plan_date.slice(0, 10) : 'unknown';
      if (!groups[date]) groups[date] = { date, trades: [], plans: [] };
      groups[date].plans.push(p);
    });
    
    // Sort days DESC (newest first)
    return Object.values(groups).sort((a, b) => b.date.localeCompare(a.date)).map(g => {
      // Sort trades DESC (newest first)
      g.trades.sort((a, b) => {
        const timeA = a.entry_time ? new Date(a.entry_time).getTime() : 0;
        const timeB = b.entry_time ? new Date(b.entry_time).getTime() : 0;
        return timeB - timeA;
      });
      // Sort plans DESC (newest first)
      g.plans.sort((a, b) => {
        const timeA = a.plan_date ? new Date(a.plan_date).getTime() : 0;
        const timeB = b.plan_date ? new Date(b.plan_date).getTime() : 0;
        // If dates are same, sort by ID
        if (timeB === timeA) return b.id - a.id;
        return timeB - timeA;
      });
      return g;
    });
  }


  function getTimeRange(trades) {
    if (!trades || trades.length === 0) return '';
    const times = trades.filter(t => t.entry_time).map(t => new Date(t.entry_time).getTime());
    if (times.length === 0) return '';
    const min = new Date(Math.min(...times));
    const max = new Date(Math.max(...times));
    
    const fmt = (d) => d.toLocaleDateString('zh-TW', { year: 'numeric', month: '2-digit', day: '2-digit' });
    
    if (fmt(min) === fmt(max)) return fmt(min);
    return `${fmt(min)} ~ ${fmt(max)}`;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && enlargedImage) {
      closeModal();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="shared-view-container">
  {#if loading}
    <div class="loading-overlay">
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

      {#if selectedItem}
        <div class="detail-overlay-header">
            <button class="back-btn" on:click={() => selectedItem = null}>
                <span class="icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 14 4 9l5-5"/><path d="M4 9h12a4 4 0 0 1 4 4v2"/></svg>
                </span> 返回列表
            </button>
        </div>
        {#if selectedItem.type === 'trade'}
            <SharedTradeDetail trade={selectedItem.data} {openModal} />

        {:else if selectedItem.type === 'plan'}
            <SharedPlanDetail plan={selectedItem.data} {openModal} />
        {/if}

      {:else if sharedData.type === 'trade' && sharedData.data}
        <SharedTradeDetail trade={sharedData.data} {openModal} />
      {:else if sharedData.type === 'plan' && sharedData.data}
        <SharedPlanDetail plan={sharedData.data} {openModal} />
      {:else if (sharedData.type === 'account' || sharedData.type === 'batch') && sharedData.data}
        {@const data = sharedData.data}
        {@const grouped = groupDataByDate(data.trades || [], data.plans || [])}
        <div class="batch-viewer">
          <div class="view-header-main">
            <h1>{sharedData.type === 'account' ? `${sharedData.username} 的 ${data.account.name}` : `${sharedData.username} 的精選分享`}</h1>
            
            <div class="header-info-line">
              {#if data.account}
                <div class="account-badges">
                  <span class="acc-badge type">{data.account.type === 'ctrader' ? 'cTrader' : 'Local'}</span>
                  {#if data.account.ctrader_env}
                    <span class="acc-badge env {data.account.ctrader_env}">{data.account.ctrader_env.toUpperCase()}</span>
                  {/if}
                  {#if data.account.ctrader_account_id}
                    <span class="acc-badge id">ID: {data.account.ctrader_account_id}</span>
                  {/if}
                </div>
              {/if}
              <div class="time-range-badge">
                📅 {getTimeRange(data.trades)}
              </div>
            </div>

            <p class="batch-meta">包含 {data.trades ? data.trades.length : 0} 筆交易與 {data.plans ? data.plans.length : 0} 筆規劃</p>
          </div>
          <div class="timeline">
            {#each grouped as group}
              <div class="day-group">
                <div class="date-header"><span class="date-tag">{formatDay(group.date)}</span></div>
                <div class="day-card-container">
                  <!-- 左側規劃 -->
                  <div class="plan-column">
                    {#if group.plans.length > 0}
                      {#each group.plans as plan}
                        {@const trendData = parseJSON(plan.trend_analysis, {})}
                        <div class="plan-item-card clickable" on:click={() => selectedItem = { type: 'plan', data: plan }}>
                          <div class="item-header">
                            <span class="item-type">📌 盤面規劃</span>
                            <span class="symbol-inline-tag">{plan.symbol}</span>
                          </div>
                          
                          <div class="mini-progression">
                            {#each ['M5', 'M15', 'H1', 'H4'] as tf}
                              {@const asianTrend = trendData.asian?.trends?.[tf]}
                              {@const europeanTrend = trendData.european?.trends?.[tf]}
                              {@const usTrend = trendData.us?.trends?.[tf]}
                              {#if asianTrend?.direction || europeanTrend?.direction || usTrend?.direction}
                                <div class="tf-row">
                                  <span class="tf-name">{tf}:</span>
                                  <div class="tf-steps">
                                    {#each [{v:'asian', l:'亞'}, {v:'european', l:'歐'}, {v:'us', l:'美'}] as session, i}
                                      {@const trend = trendData[session.v]?.trends?.[tf]}
                                      <span class="mini-step {trend?.direction || 'na'}">
                                        {session.l}
                                        {trend?.direction === 'long' ? '多' : trend?.direction === 'short' ? '空' : 'NA'}
                                      </span>
                                      {#if i < 2}<span class="step-arrow">=></span>{/if}
                                    {/each}
                                  </div>
                                </div>
                              {/if}
                            {/each}
                          </div>

                          {#if trendData.asian?.notes || trendData.european?.notes || trendData.us?.notes}
                            <div class="mini-notes">
                              {#if trendData.asian?.notes}<div class="mini-note-item"><span class="note-session asian">亞</span>{trendData.asian.notes.slice(0, 30)}{trendData.asian.notes.length > 30 ? '...' : ''}</div>{/if}
                              {#if trendData.european?.notes}<div class="mini-note-item"><span class="note-session european">歐</span>{trendData.european.notes.slice(0, 30)}{trendData.european.notes.length > 30 ? '...' : ''}</div>{/if}
                              {#if trendData.us?.notes}<div class="mini-note-item"><span class="note-session us">美</span>{trendData.us.notes.slice(0, 30)}{trendData.us.notes.length > 30 ? '...' : ''}</div>{/if}
                            </div>
                          {:else if plan.notes && plan.notes !== 'Session-based unified plan'}
                            <p class="simple-notes">{plan.notes}</p>
                          {/if}
                        </div>
                      {/each}
                    {:else}
                      <div class="empty-placeholder-shared">無規劃紀錄</div>
                    {/if}
                  </div>

                  <!-- 右側交易 -->
                  <div class="trade-column">
                    {#if group.trades.length > 0}
                      <div class="trades-stack">
                        {#each group.trades as trade}
                          <div class="trade-item-card clickable {trade.color_tag ? `tag-${trade.color_tag}` : ''}" on:click={() => selectedItem = { type: 'trade', data: trade }}>
                            <div class="item-header">
                              <div class="trade-meta">
                                <span class="symbol-inline-tag">{trade.symbol}</span>
                                <span class="session-tag {determineMarketSession(trade.entry_time)}">{getMarketSessionLabel(trade)}</span>
                                {#if trade.entry_strategy}<span class="strategy-tag {trade.entry_strategy}">{getStrategyLabel(trade.entry_strategy)}</span>{/if}
                                <span class="side-tag {trade.side}">{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span>
                              </div>
                              <div class="trade-right">
                                <div class="color-tags-static">
                                  <span class="color-dot green {trade.color_tag === 'green' ? 'active' : ''}"></span>
                                  <span class="color-dot yellow {trade.color_tag === 'yellow' ? 'active' : ''}"></span>
                                  <span class="color-dot red {trade.color_tag === 'red' ? 'active' : ''}"></span>
                                </div>
                                {#if trade.pnl_series}
                                  <div class="header-sparkline">
                                    <Sparkline data={trade.pnl_series} width={100} height={32} isOpen={!trade.exit_time} />
                                  </div>
                                {/if}
                                <span class="pnl-tag {trade.pnl >= 0 ? 'profit' : 'loss'}">
                                  {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl || 0).toFixed(2)}
                                </span>
                              </div>
                            </div>

                            <div class="trade-details-shared">
                                <div class="detail-row">
                                    <div class="info-group">
                                        <span class="label">進場</span>
                                        <strong>{trade.entry_price}</strong>
                                        <span class="arrow">→</span>
                                        <span class="label">平倉</span>
                                        <strong>{trade.exit_price || '-'}</strong>
                                    </div>
                                    <div class="info-group">
                                        <span class="label">初始ＳＬ</span>
                                        <strong>{trade.initial_sl || '-'}</strong>
                                        {#if trade.exit_sl}
                                            <span class="label">平倉ＳＬ</span>
                                            <strong>{trade.exit_sl}</strong>
                                        {/if}
                                    </div>
                                    <div class="info-group">
                                        <span class="label">子彈</span>
                                        <strong class="bullet">{trade.bullet_size || 'NA'}</strong>
                                        {#if trade.rr_ratio}
                                            <span class="label">風報比</span>
                                            <strong class="rr {trade.rr_ratio >= 0 ? 'profit' : 'loss'}">{trade.rr_ratio.toFixed(2)}</strong>
                                        {/if}
                                        <span class="label">手數</span>
                                        <strong>{trade.lot_size}</strong>
                                    </div>
                                </div>
                                <div class="trade-time-shared">
                                    {formatTime(trade.entry_time)} - {trade.exit_time ? formatTime(trade.exit_time) : '進行中'}
                                    {#if trade.exit_time}
                                        <span class="duration-text">({calculateDuration(trade.entry_time, trade.exit_time)})</span>
                                    {/if}
                                </div>
                            </div>

                            {#if trade.images && trade.images.length > 0}
                              <div class="mini-gallery-shared">
                                {#each trade.images.slice(0, 3) as img}
                                  <div class="mini-img" on:click|stopPropagation={() => openModal(imagesAPI.getUrl(img.image_path), trade.symbol + ' 交易圖表')}>
                                    <img src={imagesAPI.getUrl(img.image_path)} alt="trade" />
                                  </div>
                                {/each}
                                {#if trade.images.length > 3}<div class="more-imgs">+{trade.images.length - 3}</div>{/if}
                              </div>
                            {/if}
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-placeholder-shared">無交易紀錄</div>
                    {/if}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <div class="status-box card error">
          <p>找不到該類型資料</p>
        </div>
      {/if}
    </div>
  {:else}
    <div class="status-box card">
      <p>無內容可顯示</p>
    </div>
  {/if}
</div>

{#if enlargedImage}
  <div class="image-modal" on:click={closeModal} role="button" tabindex="0" on:keydown={e => e.key === 'Escape' && closeModal()} on:wheel={handleWheel}>
    <div class="image-modal-content" on:click|stopPropagation role="button" tabindex="0" on:keypress|stopPropagation on:mousedown={handleMouseDown} on:mousemove={handleMouseMove} on:mouseup={handleMouseUp} on:mouseleave={handleMouseUp}>
      <button class="image-modal-close" on:click={closeModal}>×</button>
      <div class="zoom-container" style="transform: scale({zoom}) translate({offsetX / zoom}px, {offsetY / zoom}px); cursor: {zoom > 1 ? (isDragging ? 'grabbing' : 'grab') : 'default'}">
        <img src={enlargedImage} alt={enlargedImageTitle} class="modal-img" />
      </div>
      {#if enlargedImageTitle}<div class="image-modal-caption">{enlargedImageTitle} (滾輪可縮放，放大的圖片可拖動)</div>{/if}
    </div>
  </div>
{/if}

<style>
  .shared-view-container { max-width: 1200px; margin: 3rem auto; padding: 0 1.25rem; font-family: 'Inter', sans-serif; }
  .detail-overlay-header { margin-bottom: 1.5rem; }
  .back-btn { 
    display: inline-flex; 
    align-items: center; 
    gap: 0.6rem; 
    padding: 0.6rem 1.25rem; 
    background: white; 
    border: 1px solid #e2e8f0; 
    border-radius: 12px; 
    color: #64748b; 
    font-weight: 700; 
    font-size: 0.85rem; 
    cursor: pointer; 
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }
  .back-btn:hover { 
    background: #f8fafc; 
    border-color: #cbd5e1; 
    color: #334155; 
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }
  .back-btn .icon { font-size: 1.1rem; }

  .card { background: var(--card-bg); border-radius: 1.5rem; padding: 2.5rem; box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05); border: 1px solid var(--border-color); margin-bottom: 2rem; }
  .public-badge { background: var(--bg-main); color: var(--text-muted); padding: 0.5rem 1.25rem; border-radius: 99px; font-size: 0.8rem; font-weight: 700; margin-bottom: 1.5rem; border: 1px solid var(--border-color); display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; line-height: 1; }
  .view-header { display: flex; justify-content: space-between; margin-bottom: 2rem; border-bottom: 1px solid var(--border-color); padding-bottom: 1.5rem; }
  .symbol-tag { display: inline-flex; align-items: center; justify-content: center; background: #4f46e5; color: white; padding: 0.25rem 0.75rem; border-radius: 6px; font-weight: 800; font-size: 0.875rem; line-height: 1; }
  
  .pnl-value { font-size: 2.5rem; font-weight: 900; }
  .pnl-value.profit { color: #10b981; }
  .pnl-value.loss { color: #ef4444; }
  
  .info-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 1.5rem; margin-bottom: 2rem; }
  .info-item label { display: block; font-size: 0.75rem; color: var(--text-muted); margin-bottom: 0.25rem; font-weight: 700; }
  .info-item span { font-size: 1rem; font-weight: 700; color: var(--text-main); }
  .notes-content { padding: 1.5rem; background: var(--bg-main); border-radius: 1rem; line-height: 1.6; }
  
  .batch-viewer { margin-top: 1rem; }
  .view-header-main h1 { font-size: 2rem; font-weight: 800; color: var(--text-main); text-align: center; margin-bottom: 0.5rem; letter-spacing: -0.02em; }
  .header-info-line { display: flex; justify-content: center; align-items: center; gap: 1.5rem; margin-bottom: 1rem; }
  .account-badges { display: flex; gap: 0.5rem; }
  .acc-badge { display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; padding: 2px 8px; border-radius: 4px; }
  .acc-badge.type { background: #f1f5f9; color: #475569; border: 1px solid #e2e8f0; }
  .acc-badge.env.live { background: #fef2f2; color: #ef4444; border: 1px solid #fee2e2; }
  .acc-badge.env.demo { background: #f0fdf4; color: #22c55e; border: 1px solid #dcfce7; }
  .acc-badge.id { background: #fafafa; color: #94a3b8; border: 1px solid #f1f5f9; }
  .time-range-badge { font-size: 0.85rem; color: var(--text-muted); font-weight: 600; background: var(--bg-main); padding: 4px 12px; border-radius: 99px; border: 1px solid var(--border-color); }
  .batch-meta { text-align: center; color: #94a3b8; margin-bottom: 3rem; font-weight: 500; }
  
  /* Timeline */
  .timeline { border-left: 2px dashed #e2e8f0; padding-left: 1.5rem; position: relative; margin-left: 2rem; }
  .day-group { margin-bottom: 4rem; position: relative; }
  .date-header { position: absolute; left: -1.5rem; top: 0; transform: translateX(-50%); z-index: 10; }
  .date-tag { display: inline-flex; align-items: center; justify-content: center; background: #6366f1; color: white; padding: 0.4rem 1rem; border-radius: 20px; font-weight: 700; font-size: 0.85rem; box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3); white-space: nowrap; line-height: 1; }
  
  /* Day Card Container */
  .day-card-container {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 1.5rem;
    background: var(--card-bg);
    padding: 1.5rem;
    border-radius: 20px;
    border: 1px solid var(--border-color);
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.03);
  }

  .plan-column, .trade-column { display: flex; flex-direction: column; gap: 1rem; }
  .plan-column { border-right: 1px dashed var(--border-color); padding-right: 1.5rem; }

  /* Cards */
  .plan-item-card, .trade-item-card {
    background: var(--card-bg);
    border-radius: 12px;
    padding: 1.25rem;
    border: 1px solid var(--border-color);
    box-shadow: 0 2px 4px rgba(0,0,0,0.02);
    position: relative;
    overflow: hidden;
  }

  .trade-item-card.tag-green { border-left: 5px solid #22c55e; }
  .trade-item-card.tag-yellow { border-left: 5px solid #eab308; }
  .trade-item-card.tag-red { border-left: 5px solid #ef4444; }

  .item-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
  .item-type { font-size: 0.75rem; font-weight: 700; color: #64748b; text-transform: uppercase; }
  .symbol-inline-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 800; color: var(--text-main); padding: 2px 6px; background: var(--nav-group-bg); border: 1px solid var(--border-color); border-radius: 4px; line-height: 1; }

  /* Plan Mini */
  .mini-progression { display: flex; flex-direction: column; gap: 0.4rem; margin-bottom: 0.75rem; }
  .tf-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; }
  .tf-name { font-weight: 700; color: var(--text-muted); width: 30px; }
  .tf-steps { display: flex; gap: 3px; align-items: center; }
  .mini-step { display: inline-flex; align-items: center; justify-content: center; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: 600; line-height: 1; }
  .mini-step.long { background: #fef2f2; color: #991b1b; }
  .mini-step.short { background: #f0fdf4; color: #166534; }
  .mini-step.na { background: #f8fafc; color: #94a3b8; }
  .step-arrow { color: #cbd5e1; font-weight: 800; font-size: 0.7rem; }

  .mini-notes { margin-top: 0.75rem; padding-top: 0.75rem; border-top: 1px solid var(--border-color); }
  .mini-note-item { font-size: 0.8rem; color: var(--text-main); line-height: 1.4; display: flex; align-items: flex-start; gap: 0.4rem; margin-bottom: 0.3rem; }
  .note-session { display: inline-flex; align-items: center; justify-content: center; font-size: 0.7rem; font-weight: 800; padding: 2px 4px; border-radius: 3px; color: white; min-width: 1.2rem; text-align: center; flex-shrink: 0; line-height: 1; }
  .note-session.asian { background: #3b82f6; }
  .note-session.european { background: #d97706; }
  .note-session.us { background: #dc2626; }
  .simple-notes { font-size: 0.8rem; color: #64748b; margin-top: 0.5rem; font-style: italic; white-space: pre-wrap; }

  /* Trade Mini */
  .trade-meta { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
  .session-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 0.7rem; padding: 2px 6px; border-radius: 4px; font-weight: 700; background: #e2e8f0; color: #475569; line-height: 1; }
  .session-tag.asian { background: #dbeafe; color: #1e40af; }
  .session-tag.european { background: #fef9c3; color: #854d0e; }
  .session-tag.us { background: #fee2e2; color: #991b1b; }
  
  .side-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 0.7rem; padding: 2px 6px; border-radius: 4px; font-weight: 700; line-height: 1; white-space: nowrap; }
  .side-tag.long { background: #fee2e2; color: #991b1b; }
  .side-tag.short { background: #dcfce7; color: #166534; }

  .strategy-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 0.7rem; padding: 2px 6px; border-radius: 4px; font-weight: 700; color: white; line-height: 1; }
  .strategy-tag.expert { background: #059669; }
  .strategy-tag.elite { background: #1e3a8a; }
  .strategy-tag.legend { background: #78350f; }

  .trade-right { display: flex; align-items: center; gap: 0.75rem; }
  .color-tags-static { display: flex; gap: 4px; }
  .color-dot { width: 8px; height: 8px; border-radius: 50%; border: 1px solid #eee; background: #fff; }
  .color-dot.active.green { background: #22c55e; border-color: #16a34a; }
  .color-dot.active.yellow { background: #eab308; border-color: #ca8a04; }
  .color-dot.active.red { background: #ef4444; border-color: #dc2626; }

  .pnl-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 1rem; font-weight: 900; padding: 4px 10px; border-radius: 8px; line-height: 1; }
  .pnl-tag.profit { background: #f0fdf4; color: #16a34a; }
  .pnl-tag.loss { background: #fef2f2; color: #dc2626; }

  .trade-details-shared { margin-top: 1rem; }
  .detail-row { display: flex; flex-wrap: wrap; gap: 1.5rem; align-items: center; margin-bottom: 0.5rem; }
  .info-group { display: flex; align-items: center; gap: 0.5rem; font-size: 0.85rem; color: #64748b; }
  .info-group strong { color: #1e293b; color: #334155; }
  .info-group .bullet { color: #6366f1; font-weight: 800; }
  .info-group .rr.profit { color: #10b981; }
  .info-group .rr.loss { color: #ef4444; }
  .arrow { color: #cbd5e1; }
  .trade-time-shared { font-size: 0.75rem; color: #94a3b8; }
  .duration-text { font-weight: 600; color: #64748b; margin-left: 0.5rem; }

  /* Gallery */
  .mini-gallery-shared { display: flex; gap: 0.5rem; margin-top: 1rem; padding-top: 1rem; border-top: 1px solid #f1f5f9; }
  .mini-img { width: 40px; height: 40px; border-radius: 4px; overflow: hidden; border: 1px solid #e2e8f0; cursor: pointer; }
  .mini-img img { width: 100%; height: 100%; object-fit: cover; }
  .more-imgs { background: #f1f5f9; color: #64748b; font-size: 0.7rem; font-weight: 700; width: 40px; height: 40px; border-radius: 4px; display: flex; align-items: center; justify-content: center; border: 1px solid #e2e8f0; }

  .empty-placeholder-shared { text-align: center; padding: 2rem; background: #fafafa; border: 1px dashed #eee; border-radius: 12px; color: #999; font-size: 0.85rem; }

  @media (max-width: 900px) {
    .day-card-container { grid-template-columns: 1fr; }
    .plan-column { border-right: none; border-bottom: 1px dashed #e2e8f0; padding-right: 0; padding-bottom: 1.5rem; }
    .timeline { margin-left: 1rem; }
  }

  .plan-item-card.clickable, .trade-item-card.clickable { cursor: pointer; transition: all 0.2s; }
  .plan-item-card.clickable:hover, .trade-item-card.clickable:hover {
    border-color: #6366f1;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }


  .image-modal { position: fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.85); z-index:10000; display:flex; align-items:center; justify-content:center; backdrop-filter: blur(8px); overflow: hidden; }
  .image-modal-content { position: relative; width: 100vw; height: 100vh; display: flex; align-items: center; justify-content: center; overflow: hidden; }
  .zoom-container { transition: transform 0.05s ease-out; display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; }
  .modal-img { max-width: 90vw; max-height: 90vh; object-fit: contain; box-shadow: 0 20px 50px rgba(0,0,0,0.5); border-radius: 4px; pointer-events: none; }
  .image-modal-close { position: absolute; top: 20px; right: 20px; color: white; font-size: 2rem; background: rgba(0,0,0,0.5); border: none; cursor: pointer; width: 50px; height: 50px; border-radius: 50%; display: flex; align-items: center; justify-content: center; z-index: 10; transition: all 0.2s; }
  .image-modal-close:hover { background: rgba(255,255,255,0.2); transform: rotate(90deg); }
  .image-modal-caption { position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%); color: white; background: rgba(0,0,0,0.6); padding: 8px 20px; border-radius: 99px; font-size: 0.9rem; font-weight: 500; pointer-events: none; white-space: nowrap; backdrop-filter: blur(4px); }
  .status-box { text-align: center; padding: 4rem 2rem; }
</style>
