<script>
  import { onMount } from 'svelte';
  import { sharesAPI, imagesAPI } from '../lib/api';
  import { getStrategyLabel, determineMarketSession } from '../lib/utils';
  import Sparkline from './Sparkline.svelte';

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

  function getMarketSessionLabel(trade) {
    const session = trade.market_session || determineMarketSession(trade.entry_time);
    const map = {
      asian: '🌏 亞盤',
      european: '🌍 歐盤',
      us: '🌎 美盤',
    };
    return map[session] || '🕒 未知';
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

  function calculateDuration(start, end) {
    if (!start || !end) return '';
    const s = new Date(start);
    const e = new Date(end);
    if (isNaN(s.getTime()) || isNaN(e.getTime())) return '';
    const diff = e - s;
    if (diff < 0) return '';

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}天 ${hours % 24}小時 ${minutes % 60}分`;
    if (hours > 0) return `${hours}小時 ${minutes % 60}分`;
    if (minutes > 0) return `${minutes}分`;
    return '1分鐘內';
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

  const expertSignals = {
    item_ma_flow: 'MA 流向',
    item_ma_space: 'MA 空間',
    item_signal_confirm: '訊號確認',
    item_risk_ratio: '風報比合理',
  };

  const eliteChecklist = {
    trend_line: '破趨勢線了嗎?',
    price_level: '破價位了嗎?',
    impulse_wave: '有驅動浪了嗎?',
    high_low: '不過高低了嗎?',
    sentiment: '情緒轉換了嗎?',
  };

  const timeframeLabels = {
    'M1': '1分',
    'M5': '5分',
    'M15': '15分',
    'M30': '30分',
    'H1': '1小時',
    'H4': '4小時',
    'D1': '天'
  };

  const legendChecklist = {
    item_618_786: '王者出現回調618或786',
    item_che: '大時區破[測]破',
    item_de: '整理段的ABC[D][E]',
  };

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

      {#if selectedItem}
        <div class="detail-overlay-header">
            <button class="back-btn" on:click={() => selectedItem = null}>
                <span class="icon">↩️</span> 返回列表
            </button>
        </div>
        {#if selectedItem.type === 'trade'}
            {@const trade = selectedItem.data}
            {@const signals = parseJSON(trade.entry_signals, [])}
            {@const checklist = parseJSON(trade.entry_checklist, {})}
            {@const patterns = parseJSON(trade.entry_pattern, [])}

            <div class="trade-detail-view card">
                <div class="view-header">
                  <div class="title-section">
                    <span class="symbol-tag">{trade.symbol || '---'}</span>
                    <span class="side-tag {trade.side || ''}">{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span>
                    {#if trade.color_tag}<span class="color-dot {trade.color_tag}" title="顏色標記"></span>{/if}
                    <h1>交易紀錄詳情</h1>
                  </div>
                  <div class="pnl-section">
                    {#if trade.pnl !== undefined && trade.pnl !== null}
                      <div class="pnl-value {trade.pnl >= 0 ? 'profit' : 'loss'}">
                        {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl).toFixed(2)}
                      </div>
                    {/if}
                    {#if trade.pnl_series}
                      <div class="sparkline-container-shared">
                        <Sparkline data={trade.pnl_series} side={trade.side} width={120} height={40} />
                      </div>
                    {/if}
                  </div>
                </div>

                <div class="info-grid extended">
                  <div class="info-row-group">
                    <div class="info-item"><label>交易品種</label><span class="symbol-inline-tag">{trade.symbol}</span></div>
                    <div class="info-item"><label>做多或做空</label><span>{trade.side === 'long' ? '做多 (Long)' : trade.side === 'short' ? '做空 (Short)' : 'NA'}</span></div>
                    <div class="info-item"><label>手數</label><span>{trade.lot_size || '0.00'}</span></div>
                    <div class="info-item"><label>TICKET</label><span class="ticket-val">{trade.ticket || 'NA'}</span></div>
                  </div>

                  <div class="info-row-divider"></div>

                  <div class="info-row-group">
                    <div class="info-item"><label>進場價格</label><span class="value-highlight">{trade.entry_price || '0.00'}</span></div>
                    <div class="info-item"><label>初始 S L</label><span>{trade.initial_sl || 'NA'}</span></div>
                    <div class="info-item"><label>平倉價格</label><span class="value-highlight">{trade.exit_price || 'NA'}</span></div>
                    <div class="info-item"><label>平倉 S L</label><span>{trade.exit_sl || 'NA'}</span></div>
                  </div>

                  <div class="info-row-divider"></div>

                  <div class="info-row-group">
                    <div class="info-item"><label>盈虧金額</label><span class="rr-value {trade.pnl >= 0 ? 'profit' : 'loss'}">{trade.pnl !== undefined && trade.pnl !== null ? trade.pnl.toFixed(2) : '--'}</span></div>
                    <div class="info-item"><label>盈虧點數</label><span>{trade.pnl_points != null ? trade.pnl_points.toFixed(1) : 'NA'}</span></div>
                    <div class="info-item"><label>子彈大小 (Bullet)</label><span>{trade.bullet_size != null ? trade.bullet_size : '自動計算'}</span></div>
                    <div class="info-item"><label>風報比 (R:R)</label><span class="rr-value-pills">{trade.rr_ratio != null ? trade.rr_ratio.toFixed(2) : '自動計算'}</span></div>
                  </div>

                  <div class="info-row-divider"></div>

                  <div class="info-row-group">
                    <div class="info-item"><label>開倉時間 (UTC+8)</label><span>{formatDate(trade.entry_time)}</span></div>
                    <div class="info-item"><label>市場時段與規劃</label>
                      <div class="mock-session-display">
                        <span class="session-label-btn {determineMarketSession(trade.entry_time)}">{getMarketSessionLabel(trade).replace(/^[🌏🌍🌎]\s/, '')}</span>
                        <span class="session-time-text">
                            {#if determineMarketSession(trade.entry_time) === 'asian'}08:00 - 15:00{:else if determineMarketSession(trade.entry_time) === 'european'}16:00 - 00:00{:else}21:00 - 05:00{/if} 
                            · 冬令時間
                        </span>
                      </div>
                    </div>
                  </div>

                  <div class="info-row-divider"></div>

                  <div class="info-row-group">
                    <div class="info-item"><label>平倉時間 (UTC+8)</label><span>{trade.exit_time ? formatDate(trade.exit_time) : '--'}</span></div>
                    <div class="info-item"><label>持單時間</label><span class="duration-badge-pill">{calculateDuration(trade.entry_time, trade.exit_time) || 'NA'}</span></div>
                  </div>

                  <div class="info-row-divider"></div>

                  <div class="info-row-group">
                    <div class="info-item full-width-item">
                        <label class="rocket-header">🚀 進場分析</label>
                        <div class="analysis-sub-flex horizontal-layout">
                            <div class="analysis-sub-group">
                                <label class="sub-label">🎯 進場種類</label>
                                <div class="mock-strategy-btns">
                                    <span class="strat-btn {trade.entry_strategy === 'expert' ? 'active' : ''}">達人</span>
                                    <span class="strat-btn {trade.entry_strategy === 'elite' ? 'active' : ''}">菁英</span>
                                    <span class="strat-btn {trade.entry_strategy === 'legend' ? 'active' : ''}">傳奇</span>
                                </div>
                            </div>
                            <div class="analysis-sub-group">
                                <label class="sub-label">🕒 進場時區</label>
                                <div class="mock-tf-pills">
                                    {#each ['M1', 'M5', 'M15', 'H1', 'H4', 'D1'] as tf}
                                        <span class="tf-pill {trade.entry_timeframe === tf || trade.entry_timeframe === tf.toLowerCase() ? 'active' : ''}">{timeframeLabels[tf] || tf}</span>
                                    {/each}
                                </div>
                            </div>
                        </div>
                    </div>
                  </div>
                </div>

                <!-- 進場分析區塊 -->
                
                {#if trade.entry_strategy === 'expert' || trade.entry_strategy === 'elite' || trade.entry_strategy === 'legend'}
                <div class="section-box analysis-section">
                  <!-- Removed duplicate header <h3>🔍 進場分析</h3> -->
                  <div class="analysis-grid">
                    {#if signals && signals.length > 0}
                      <div class="analysis-item">
                        <label>選用訊號</label>
                        <div class="tags-container">
                          {#each signals as sig}
                            {@const sigName = typeof sig === 'string' ? sig : sig.name}
                            {@const sigImg = typeof sig === 'object' ? (sig.image || sig.originalImage) : null}
                            <span class="analysis-tag {sigImg ? 'has-img' : ''}" on:click={() => sigImg && openModal(sigImg, expertSignals[sigName] || sigName)}>
                              {#if sigImg}
                                <img src={sigImg} alt={sigName} class="tag-icon" />
                              {/if}
                              {expertSignals[sigName] || sigName}
                            </span>
                          {/each}
                        </div>
                      </div>
                    {:else}
                      <div class="analysis-item">
                        <label>選用訊號</label>
                        <span class="na-txt">無訊號</span>
                      </div>
                    {/if}
                    
                    {#if patterns && patterns.length > 0}
                      <div class="analysis-item">
                        <label>進場樣態</label>
                        <div class="tags-container">
                          {#each patterns as pat}
                            {@const patName = typeof pat === 'string' ? pat : pat.name}
                            {@const patImg = typeof pat === 'object' ? (pat.image || pat.originalImage) : null}
                            <span class="analysis-tag pattern {patImg ? 'has-img' : ''}" on:click={() => patImg && openModal(patImg, patName)}>
                              {#if patImg}
                                <img src={patImg} alt={patName} class="tag-icon" />
                              {/if}
                              {patName}
                            </span>
                          {/each}
                        </div>
                      </div>
                    {:else}
                      <div class="analysis-item">
                        <label>進場樣態</label>
                        <span class="na-txt">無樣態</span>
                      </div>
                    {/if}
                  </div>

                  {#if Object.keys(checklist).length > 0}
                    <div class="checklist-display">
                      <label>檢查清單</label>
                      <div class="check-items">
                        {#each Object.entries(checklist) as [key, val]}
                          {#if val}
                            <div class="check-chip">✅ {eliteChecklist[key] || legendChecklist[key] || key}</div>
                          {/if}
                        {/each}
                      </div>
                    </div>
                  {:else}
                    <div class="checklist-display">
                      <label>檢查清單</label>
                      <span class="na-txt">無檢查項目</span>
                    </div>
                  {/if}
                </div>
                {/if}

                <div class="section-box">
                    <h3>🎯 平倉理由</h3>
                    {#if trade.exit_reason}
                        <div class="notes-content ql-editor">{@html lazyLoadHTML(trade.exit_reason)}</div>
                    {:else}
                        <p class="empty-placeholder">無平倉理由紀錄</p>
                    {/if}
                </div>
                
                <div class="section-box">
                    <h3>📝 交易復盤筆記</h3>
                    {#if trade.notes}
                        <div class="notes-content ql-editor">{@html lazyLoadHTML(trade.notes)}</div>
                    {:else}
                        <p class="empty-placeholder">無交易復盤紀錄</p>
                    {/if}
                </div>
                {#if trade.images && trade.images.length > 0}
                  <div class="section-box">
                    <h3>🖼️ 圖表截圖</h3>
                    <div class="image-gallery">
                      {#each trade.images as img}
                        {#if img && img.image_path}
                          <div class="image-card">
                            <img src={imagesAPI.getUrl(img.image_path)} alt="Trade Chart" class="clickable-image" loading="lazy" on:click={() => openModal(imagesAPI.getUrl(img.image_path), img.image_type)} />
                          </div>
                        {/if}
                      {/each}
                    </div>
                  </div>
                {/if}
            </div>
        {:else if selectedItem.type === 'plan'}
            {@const plan = selectedItem.data}
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
                {#if plan.notes}<div class="section-box"><h3>📝 規劃備註</h3><div class="notes-content ql-editor">{@html lazyLoadHTML(plan.notes)}</div></div>{/if}
                {#each ['asian', 'european', 'us'] as session}
                  {#if trendAnalysis[session]}
                    <div class="session-block {session}">
                      <h4>{session === 'asian' ? '🌏 亞盤' : session === 'european' ? '🌍 歐盤' : '🌎 美盤'}</h4>
                      <p class="session-notes">{trendAnalysis[session].notes || ''}</p>
                    </div>
                  {/if}
                {/each}
            </div>
        {/if}

      {:else if sharedData.type === 'trade' && sharedData.data}
        {@const trade = sharedData.data}
        {@const checklist = parseJSON(trade.entry_checklist, {})}
        {@const signals = parseJSON(trade.entry_signals, [])}
        {@const patterns = parseJSON(trade.entry_pattern, [])}
        <div class="trade-detail-view card">
          <!-- 交易詳情 HTML -->
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-tag">{trade.symbol || '---'}</span>
              <span class="side-tag {trade.side || ''}">{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span>
              {#if trade.color_tag}<span class="color-dot {trade.color_tag}" title="顏色標記"></span>{/if}
              <h1>交易紀錄詳情</h1>
            </div>
            <div class="pnl-section">
              {#if trade.pnl !== undefined && trade.pnl !== null}
                <div class="pnl-value {trade.pnl >= 0 ? 'profit' : 'loss'}">
                  {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl).toFixed(2)}
                </div>
              {/if}
              {#if trade.pnl_series}
                <div class="sparkline-container-shared">
                  <Sparkline data={trade.pnl_series} side={trade.side} width={120} height={40} />
                </div>
              {/if}
            </div>
          </div>
          <!-- (Rest of trade detail items...) -->
          <div class="info-grid">
            <div class="info-item"><label>進場時間</label><span>{formatDate(trade.entry_time)}</span></div>
            <div class="info-item"><label>進場價格</label><span class="value-highlight">{trade.entry_price || '---'}</span></div>
            <div class="info-item"><label>初始 SL</label><span>{trade.initial_sl || 'NA'}</span></div>
            <div class="info-item"><label>平台編號</label><span class="ticket-val">{trade.ticket || 'NA'}</span></div>

            <div class="info-item"><label>平倉時間</label><span>{trade.exit_time ? formatDate(trade.exit_time) : '進行中'}</span></div>
            <div class="info-item"><label>平倉價格</label><span class="value-highlight">{trade.exit_price || 'NA'}</span></div>
            <div class="info-item"><label>平倉 SL</label><span>{trade.exit_sl || 'NA'}</span></div>
            <div class="info-item"><label>持單時間</label><span class="duration-badge">{calculateDuration(trade.entry_time, trade.exit_time) || '---'}</span></div>

            <div class="info-item"><label>盈虧點數</label><span>{trade.pnl_points !== undefined && trade.pnl_points !== null ? trade.pnl_points.toFixed(1) : 'NA'}</span></div>
            <div class="info-item"><label>子彈大小</label><span>{trade.bullet_size || 'NA'}</span></div>
            <div class="info-item"><label>風報比 (R:R)</label><span class="rr-value">{trade.rr_ratio ? trade.rr_ratio.toFixed(2) + ' R' : 'NA'}</span></div>
            <div class="info-item"><label>手數</label><span>{trade.lot_size || '---'}</span></div>

            <div class="info-item"><label>市場時段</label><span>{getMarketSessionLabel(trade)}</span></div>
            <div class="info-item"><label>進場時區</label><span>{trade.entry_timeframe || 'NA'}</span></div>
            <div class="info-item"><label>進場種類</label><span class="strategy-badge {trade.entry_strategy || ''}">{getStrategyLabel(trade.entry_strategy) || 'NA'}</span></div>
          </div>
                  {#if trade.notes}<div class="section-box"><h3>📝 交易復盤筆記</h3><div class="notes-content ql-editor">{@html lazyLoadHTML(trade.notes)}</div></div>{/if}
          <!-- Images... -->
          {#if trade.images && trade.images.length > 0}
            <div class="section-box">
              <h3>🖼️ 圖表截圖</h3>
              <div class="image-gallery">
                {#each trade.images as img}
                  {#if img && img.image_path}
                    <div class="image-card">
                      <img src={imagesAPI.getUrl(img.image_path)} alt="Trade Chart" class="clickable-image" loading="lazy" on:click={() => openModal(imagesAPI.getUrl(img.image_path), img.image_type)} />
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
          <!-- 規劃詳情 HTML -->
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-tag">{plan.symbol || '---'}</span>
              <h1>盤面規劃分享</h1>
            </div>
            <div class="date-section">
              <span class="plan-date-tag">📅 {plan.plan_date ? plan.plan_date.slice(0, 10) : ''}</span>
            </div>
          </div>
          {#if plan.notes}<div class="section-box"><h3>📝 規劃備註</h3><div class="notes-content ql-editor">{@html lazyLoadHTML(plan.notes)}</div></div>{/if}
          <!-- Session blocks... -->
          {#each ['asian', 'european', 'us'] as session}
            {#if trendAnalysis[session]}
              <div class="session-block {session}">
                <h4>{session === 'asian' ? '🌏 亞盤' : session === 'european' ? '🌍 歐盤' : '🌎 美盤'}</h4>
                <p class="session-notes">{trendAnalysis[session].notes || ''}</p>
              </div>
            {/if}
          {/each}
        </div>
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
                            {#each ['15m', '1h', '4h', 'D1'] as tf}
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
                              {#if trendData.asian?.notes}<div class="mini-note-item"><span class="note-session asian">亞</span>{trendData.asian.notes}</div>{/if}
                              {#if trendData.european?.notes}<div class="mini-note-item"><span class="note-session european">歐</span>{trendData.european.notes}</div>{/if}
                              {#if trendData.us?.notes}<div class="mini-note-item"><span class="note-session us">美</span>{trendData.us.notes}</div>{/if}
                            </div>
                          {:else}
                            <p class="simple-notes">{plan.notes || '無備註'}</p>
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
                                    <Sparkline data={trade.pnl_series} width={100} height={32} side={trade.side} />
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

  .card { background: white; border-radius: 1.5rem; padding: 2.5rem; box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05); border: 1px solid #f1f5f9; margin-bottom: 2rem; }
  .public-badge { background: #f8fafc; color: #64748b; padding: 0.5rem 1.25rem; border-radius: 99px; font-size: 0.8rem; font-weight: 700; margin-bottom: 1.5rem; border: 1px solid #e2e8f0; display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; line-height: 1; }
  .view-header { display: flex; justify-content: space-between; margin-bottom: 2rem; border-bottom: 1px solid #f1f5f9; padding-bottom: 1.5rem; }
  .symbol-tag { display: inline-flex; align-items: center; justify-content: center; background: #4f46e5; color: white; padding: 0.25rem 0.75rem; border-radius: 6px; font-weight: 800; font-size: 0.875rem; line-height: 1; }
  
  .pnl-value { font-size: 2.5rem; font-weight: 900; }
  .pnl-value.profit { color: #10b981; }
  .pnl-value.loss { color: #ef4444; }
  
  .info-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 1.5rem; margin-bottom: 2rem; }
  .info-item label { display: block; font-size: 0.75rem; color: #64748b; margin-bottom: 0.25rem; font-weight: 700; }
  .info-item span { font-size: 1rem; font-weight: 700; color: #1e293b; }
  .notes-content { padding: 1.5rem; background: #f8fafc; border-radius: 1rem; line-height: 1.6; }
  
  .batch-viewer { margin-top: 1rem; }
  .view-header-main h1 { font-size: 2rem; font-weight: 800; color: #1e293b; text-align: center; margin-bottom: 0.5rem; letter-spacing: -0.02em; }
  .header-info-line { display: flex; justify-content: center; align-items: center; gap: 1.5rem; margin-bottom: 1rem; }
  .account-badges { display: flex; gap: 0.5rem; }
  .acc-badge { display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; padding: 2px 8px; border-radius: 4px; }
  .acc-badge.type { background: #f1f5f9; color: #475569; border: 1px solid #e2e8f0; }
  .acc-badge.env.live { background: #fef2f2; color: #ef4444; border: 1px solid #fee2e2; }
  .acc-badge.env.demo { background: #f0fdf4; color: #22c55e; border: 1px solid #dcfce7; }
  .acc-badge.id { background: #fafafa; color: #94a3b8; border: 1px solid #f1f5f9; }
  .time-range-badge { font-size: 0.85rem; color: #64748b; font-weight: 600; background: #f8fafc; padding: 4px 12px; border-radius: 99px; border: 1px solid #f1f5f9; }
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
    background: white;
    padding: 1.5rem;
    border-radius: 20px;
    border: 1px solid #f1f5f9;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.03);
  }

  .plan-column, .trade-column { display: flex; flex-direction: column; gap: 1rem; }
  .plan-column { border-right: 1px dashed #e2e8f0; padding-right: 1.5rem; }

  /* Cards */
  .plan-item-card, .trade-item-card {
    background: white;
    border-radius: 12px;
    padding: 1.25rem;
    border: 1px solid #f1f5f9;
    box-shadow: 0 2px 4px rgba(0,0,0,0.02);
    position: relative;
    overflow: hidden;
  }

  .trade-item-card.tag-green { border-left: 5px solid #22c55e; }
  .trade-item-card.tag-yellow { border-left: 5px solid #eab308; }
  .trade-item-card.tag-red { border-left: 5px solid #ef4444; }

  .item-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
  .item-type { font-size: 0.75rem; font-weight: 700; color: #64748b; text-transform: uppercase; }
  .symbol-inline-tag { display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 800; color: #1e293b; padding: 2px 6px; background: #f1f5f9; border: 1px solid #e2e8f0; border-radius: 4px; line-height: 1; }

  /* Plan Mini */
  .mini-progression { display: flex; flex-direction: column; gap: 0.4rem; margin-bottom: 0.75rem; }
  .tf-row { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; }
  .tf-name { font-weight: 700; color: #475569; width: 30px; }
  .tf-steps { display: flex; gap: 3px; align-items: center; }
  .mini-step { display: inline-flex; align-items: center; justify-content: center; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; font-weight: 600; line-height: 1; }
  .mini-step.long { background: #fef2f2; color: #991b1b; }
  .mini-step.short { background: #f0fdf4; color: #166534; }
  .mini-step.na { background: #f8fafc; color: #94a3b8; }
  .step-arrow { color: #cbd5e1; font-weight: 800; font-size: 0.7rem; }

  .mini-notes { margin-top: 0.75rem; padding-top: 0.75rem; border-top: 1px solid #edf2f7; }
  .mini-note-item { font-size: 0.8rem; color: #4a5568; line-height: 1.4; display: flex; align-items: flex-start; gap: 0.4rem; margin-bottom: 0.3rem; }
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

  .info-grid.extended {
    display: flex;
    flex-direction: column;
    gap: 0;
    margin-bottom: 2rem;
  }
  .info-row-group {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.5rem;
    padding: 1.25rem 0;
  }
  .info-row-divider {
    height: 1px;
    background: #f1f5f9;
    width: 100%;
  }
  .info-item label { color: #64748b; font-weight: 700; text-transform: uppercase; font-size: 0.75rem; margin-bottom: 0.6rem; display: block; }
  .info-item .value-highlight { font-size: 1.1rem; font-weight: 800; color: #1e293b; background: #f8fafc; padding: 0.4rem 0.75rem; border-radius: 8px; border: 1px solid #e2e8f0; display: inline-block; min-width: 100px; }
  .full-width-item { grid-column: 1 / -1; }

  /* Sync with Edit Form Styles */
  .mock-session-display { display: flex; align-items: center; gap: 0.75rem; background: transparent; padding: 0.4rem 0.75rem; border-radius: 8px; border: 2px solid #e0f2fe; width: fit-content; white-space: nowrap; flex-shrink: 0; }
  .session-label-btn { padding: 4px 12px; border-radius: 6px; font-weight: 800; font-size: 0.8rem; color: white; line-height: 1.2; text-align: center; display: inline-block; }
  .session-label-btn.asian { background: #3b82f6; }
  .session-label-btn.european { background: #d97706; }
  .session-label-btn.us { background: #00b4ff; }
  .session-time-text { font-size: 0.85rem; color: #475569; font-weight: 600; white-space: nowrap; }

  .duration-badge-pill { background: #eff6ff; color: #2563eb; padding: 6px 16px; border-radius: 8px; font-weight: 700; border: 1px solid #dbeafe; display: inline-block; }

  /* Shared Viewer Analysis Styling Updates */
  .rocket-header { font-size: 1.25rem !important; font-weight: 800; color: #2d3748; margin-bottom: 1.5rem !important; }

  .analysis-sub-flex.horizontal-layout { display: flex; flex-direction: row; flex-wrap: wrap; gap: 3rem; margin-top: 1rem; align-items: flex-end; }
  .analysis-sub-group { display: flex; flex-direction: column; gap: 0.5rem; }
  
  .mock-strategy-btns { display: flex; gap: 0.5rem; }
  .strat-btn { padding: 0.6rem 1.5rem; border: 2px solid #cbd5e0; border-radius: 8px; color: #4a5568; font-size: 1rem; font-weight: 600; background: white; text-align: center; min-width: 80px; }
  .strat-btn.active { border-color: #6366f1; background: #e0e7ff; color: #4338ca; box-shadow: none; }

  .mock-tf-pills { display: flex; background: #1a1a1a; border-radius: 8px; padding: 4px; gap: 2px; width: fit-content; align-items: center; }
  .tf-pill { color: #888; padding: 6px 12px; font-size: 0.85rem; font-weight: 600; cursor: default; transition: all 0.2s; white-space: nowrap; border-radius: 6px; }
  .tf-pill.active { background: #333; color: #60a5fa !important; box-shadow: 0 2px 4px rgba(0,0,0,0.2); }

  .analysis-section { background: #fcfdfe; border: 1px solid #edf2f7; margin-top: 0; padding: 0 1.5rem 1.5rem 1.5rem; border-radius: 12px; border-top: none; border-top-left-radius: 0; border-top-right-radius: 0; }
  /* Make sure internal content isn't stuck to edges */
  .analysis-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; margin-top: 1rem; }
  .analysis-item label { font-size: 0.75rem; font-weight: 800; color: #4a5568; margin-bottom: 0.75rem; display: block; }
  .tags-container { display: flex; flex-wrap: wrap; gap: 0.5rem; }
  .analysis-tag { 
    background: #ebf4ff; 
    color: #2b6cb0; 
    padding: 4px 12px; 
    border-radius: 6px; 
    font-size: 0.85rem; 
    font-weight: 700; 
    display: inline-flex; 
    align-items: center; 
    gap: 0.5rem;
  }
  .analysis-tag.has-img { cursor: pointer; transition: transform 0.2s; }
  .analysis-tag.has-img:hover { transform: translateY(-1px); background: #dbeafe; }
  .tag-icon { width: 20px; height: 20px; border-radius: 3px; object-fit: cover; border: 1px solid rgba(0,0,0,0.05); }
  .analysis-tag.pattern { background: #faf5ff; color: #6b46c1; }
  
  .checklist-display { margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px dashed #e2e8f0; }
  .checklist-display label { font-size: 0.75rem; font-weight: 800; color: #4a5568; margin-bottom: 0.75rem; display: block; }
  .check-items { display: flex; flex-wrap: wrap; gap: 0.75rem; }
  .check-chip { background: #f0fdf4; color: #166534; padding: 6px 14px; border-radius: 99px; font-size: 0.85rem; font-weight: 600; border: 1px solid #dcfce7; }

  .ticket-val { font-family: 'JetBrains Mono', monospace; font-size: 0.8rem; color: #94a3b8; }
  .rr-value-pills { background: #f5f3ff; color: #5b21b6; padding: 2px 8px; border-radius: 4px; font-weight: 800; }

  .image-modal { position: fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.85); z-index:10000; display:flex; align-items:center; justify-content:center; backdrop-filter: blur(8px); overflow: hidden; }
  .image-modal-content { position: relative; width: 100vw; height: 100vh; display: flex; align-items: center; justify-content: center; overflow: hidden; }
  .zoom-container { transition: transform 0.05s ease-out; display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; }
  .modal-img { max-width: 90vw; max-height: 90vh; object-fit: contain; box-shadow: 0 20px 50px rgba(0,0,0,0.5); border-radius: 4px; pointer-events: none; }
  .image-modal-close { position: absolute; top: 20px; right: 20px; color: white; font-size: 2rem; background: rgba(0,0,0,0.5); border: none; cursor: pointer; width: 50px; height: 50px; border-radius: 50%; display: flex; align-items: center; justify-content: center; z-index: 10; transition: all 0.2s; }
  .image-modal-close:hover { background: rgba(255,255,255,0.2); transform: rotate(90deg); }
  .image-modal-caption { position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%); color: white; background: rgba(0,0,0,0.6); padding: 8px 20px; border-radius: 99px; font-size: 0.9rem; font-weight: 500; pointer-events: none; white-space: nowrap; backdrop-filter: blur(4px); }
  .loader { width: 40px; height: 40px; border: 4px solid #f1f5f9; border-top-color: #4f46e5; border-radius: 50%; animation: spin 1s linear infinite; margin: 0 auto 1rem; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .status-box { text-align: center; padding: 4rem 2rem; }
</style>
