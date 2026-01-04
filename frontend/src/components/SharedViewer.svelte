<script>
  import { onMount } from 'svelte';
  import { sharesAPI, imagesAPI } from '../lib/api';
  import { getStrategyLabel } from '../lib/utils';
  import Sparkline from './Sparkline.svelte';

  export let token = '';

  let loading = true;
  let error = null;
  let sharedData = null; // { type: 'trade'|'plan', data: ... }

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
    // Inject loading="lazy" into all img tags
    return html.replace(/<img /g, '<img loading="lazy" ');
  }

  // 達人策略檢查項翻譯
  const expertSignals = {
    item_ma_flow: 'MA 流向',
    item_ma_space: 'MA 空間',
    item_signal_confirm: '訊號確認',
    item_risk_ratio: '風報比合理',
  };

  // 菁英策略檢查項翻譯
  const eliteChecklist = {
    trend_line: '破趨勢線了嗎?',
    price_level: '破價位了嗎?',
    impulse_wave: '有驅動浪了嗎?',
    high_low: '不過高低了嗎?',
    sentiment: '情緒轉換了嗎?',
  };

  // 傳奇策略檢查項翻譯
  const legendChecklist = {
    item_618_786: '王者出現回調618或786',
    item_che: '大時區破[測]破',
    item_de: '整理段的ABC[D][E]',
  };

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

      {#if sharedData.type === 'trade' && sharedData.data}
        {@const trade = sharedData.data}
        {@const checklist = parseJSON(trade.entry_checklist, {})}
        {@const signals = parseJSON(trade.entry_signals, [])}
        {@const patterns = parseJSON(trade.entry_pattern, [])}

        <div class="trade-detail-view card">
          <div class="view-header">
            <div class="title-section">
              <span class="symbol-tag">{trade.symbol || '---'}</span>
              <span class="side-tag {trade.side || ''}"
                >{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span
              >
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
              <div class="info-item">
                <label>持單時間</label>
                <span class="duration-badge"
                  >{calculateDuration(trade.entry_time, trade.exit_time) || '---'}</span
                >
              </div>
            {/if}
            {#if trade.entry_strategy}
              <div class="info-item">
                <label>交易策略</label>
                <span class="strategy-badge {trade.entry_strategy}"
                  >{getStrategyLabel(trade.entry_strategy)}</span
                >
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
                          {@const label =
                            trade.entry_strategy === 'expert'
                              ? expertSignals[id]
                              : trade.entry_strategy === 'elite'
                                ? eliteChecklist[id]
                                : legendChecklist[id]}
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
                        <span class="signal-tag-item"
                          >✨ {typeof signal === 'object'
                            ? signal.name || signal.id || JSON.stringify(signal)
                            : signal}</span
                        >
                      {/each}
                    </div>
                  </div>
                {/if}

                <!-- 策略截圖區 -->
                <div class="strategy-images">
                  <!-- 通用的進場觀察圖 -->
                  {#if trade.entry_strategy_image}
                    <div class="img-preview-box">
                      <p>進場觀察圖：</p>
                      <img
                        src={trade.entry_strategy_image}
                        alt="Strategy Observation"
                        class="clickable-image"
                        loading="lazy"
                        on:click={() => openModal(trade.entry_strategy_image, '進場觀察圖')}
                        role="presentation"
                      />
                    </div>
                  {/if}

                  <!-- 達人訊號圖 (每個訊號可以有自己的圖) -->
                  {#if trade.entry_strategy === 'expert'}
                    {#each signals as signal}
                      {#if typeof signal === 'object' && signal.image}
                        <div class="img-preview-box">
                          <p>✨ 訊號：{signal.name} 圖面：</p>
                          <img
                            src={signal.image}
                            alt={signal.name}
                            class="clickable-image"
                            loading="lazy"
                            on:click={() => openModal(signal.image, `訊號圖：${signal.name}`)}
                            role="presentation"
                          />
                        </div>
                      {/if}
                    {/each}
                  {/if}

                  <!-- 傳奇策略專用圖 -->
                  {#if trade.entry_strategy === 'legend'}
                    {#if trade.legend_king_image}
                      <div class="img-preview-box">
                        <p>👑 王者回調 ({trade.legend_king_htf})：</p>
                        <img
                          src={trade.legend_king_image}
                          alt="King Callback"
                          class="clickable-image"
                          loading="lazy"
                          on:click={() =>
                            openModal(
                              trade.legend_king_image,
                              `王者回調 (${trade.legend_king_htf})`
                            )}
                          role="presentation"
                        />
                      </div>
                    {/if}
                    {#if trade.legend_htf_image}
                      <div class="img-preview-box">
                        <p>🌊 大時區破[測]破 ({trade.legend_htf})：</p>
                        <img
                          src={trade.legend_htf_image}
                          alt="HTF Breakout"
                          class="clickable-image"
                          loading="lazy"
                          on:click={() =>
                            openModal(
                              trade.legend_htf_image,
                              `大時區破[測]破 (${trade.legend_htf})`
                            )}
                          role="presentation"
                        />
                      </div>
                    {/if}
                  {/if}

                  <!-- 菁英策略專用樣態圖 -->
                  {#if trade.entry_strategy === 'elite'}
                    {#each patterns as pattern}
                      {#if pattern.image}
                        <div class="img-preview-box">
                          <p>🎯 {pattern.name} 樣態圖：</p>
                          <img
                            src={pattern.image}
                            alt={pattern.name}
                            class="clickable-image"
                            loading="lazy"
                            on:click={() => openModal(pattern.image, `樣態圖：${pattern.name}`)}
                            role="presentation"
                          />
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
              <div class="notes-content ql-editor">{@html lazyLoadHTML(trade.notes)}</div>
            </div>
          {/if}

          {#if trade.exit_reason}
            <div class="section-box">
              <h3>🎯 平倉理由</h3>
              <div class="notes-content ql-editor">{@html lazyLoadHTML(trade.exit_reason)}</div>
            </div>
          {/if}

          {#if trade.images && trade.images.length > 0}
            <div class="section-box">
              <h3>🖼️ 圖表截圖 (Gallery)</h3>
              <div class="image-gallery">
                {#each trade.images as img}
                  {#if img && img.image_path}
                    <div class="image-card">
                      <img
                        src={imagesAPI.getUrl(img.image_path)}
                        alt="Trade Chart"
                        class="clickable-image"
                        loading="lazy"
                        on:click={() =>
                          openModal(imagesAPI.getUrl(img.image_path), img.image_type || '圖表截圖')}
                        on:keypress={() =>
                          openModal(imagesAPI.getUrl(img.image_path), img.image_type || '圖表截圖')}
                        role="button"
                        tabindex="0"
                      />
                      {#if img.image_type}
                        <span class="image-type-label">
                          {img.image_type === 'entry'
                            ? '📍 進場'
                            : img.image_type === 'exit'
                              ? '🎯 平倉'
                              : '📷 觀察'}
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
              <span class="plan-date-tag"
                >📅 {plan.plan_date ? plan.plan_date.slice(0, 10) : ''}</span
              >
            </div>
          </div>

          <div class="section-box">
            <h3>📝 規劃備註</h3>
            <div class="notes-content ql-editor">
              {@html lazyLoadHTML(plan.notes) || '尚無備註內容'}
            </div>
          </div>

          {#each ['asian', 'european', 'us'] as session}
            {#if trendAnalysis[session]}
              {@const sessionData = trendAnalysis[session]}
              <div class="session-block {session}">
                <h4>
                  時段：{session === 'asian'
                    ? '🌏 亞盤'
                    : session === 'european'
                      ? '🌍 歐盤'
                      : '🌎 美盤'}
                </h4>
                {#if sessionData.notes}
                  <p class="session-notes">{sessionData.notes}</p>
                {/if}

                {#if sessionData.trends}
                  <div class="trends-grid">
                    {#each ['M5', 'M15', 'H1', 'H4', 'D1'] as tf}
                      {@const trend = sessionData.trends[tf]}
                      {#if trend && (trend.image || trend.signals_image || trend.wave_image || trend.direction || (trend.signals && trend.signals.length > 0) || (trend.wave_numbers && trend.wave_numbers.length > 0))}
                        <div class="trend-card {trend.direction}">
                          <div class="trend-header">
                            <span class="tf-badge">{tf}</span>
                            {#if trend.direction}
                              <span class="dir-badge {trend.direction}"
                                >{trend.direction === 'long' ? '多' : '空'}</span
                              >
                            {/if}
                          </div>

                          <div class="trend-body">
                            <!-- General Trend Image -->
                            {#if trend.image}
                              <div class="t-img-box">
                                <span class="img-label">趨勢圖</span>
                                <img
                                  src={trend.image}
                                  alt="{tf} Trend"
                                  class="clickable-image"
                                  loading="lazy"
                                  on:click={() => openModal(trend.image, `${tf} 趨勢圖`)}
                                  on:keypress={() => openModal(trend.image, `${tf} 趨勢圖`)}
                                  role="button"
                                  tabindex="0"
                                />
                              </div>
                            {/if}

                            <!-- Expert Signals -->
                            {#if (trend.signals && trend.signals.length > 0) || trend.signals_image}
                              <div class="t-section">
                                <div class="section-title">✨ 達人訊號</div>
                                {#if trend.signals && trend.signals.length > 0}
                                  <div class="t-tags">
                                    {#each trend.signals as s}
                                      <span class="t-tag">{s}</span>
                                    {/each}
                                  </div>
                                {/if}
                                {#if trend.signals_image}
                                  <div class="t-img-box">
                                    <img
                                      src={trend.signals_image}
                                      alt="{tf} Signals"
                                      class="clickable-image"
                                      loading="lazy"
                                      on:click={() =>
                                        openModal(trend.signals_image, `${tf} 達人訊號`)}
                                      on:keypress={() =>
                                        openModal(trend.signals_image, `${tf} 達人訊號`)}
                                      role="button"
                                      tabindex="0"
                                    />
                                  </div>
                                {/if}
                              </div>
                            {/if}

                            <!-- Wave Analysis -->
                            {#if (trend.wave_numbers && trend.wave_numbers.length > 0) || trend.wave_image}
                              <div class="t-section">
                                <div class="section-title">🌊 波浪分析</div>
                                {#if trend.wave_numbers && trend.wave_numbers.length > 0}
                                  <div class="t-wave-nums">
                                    {#each trend.wave_numbers as n, i}
                                      {#if i > 0}<span class="w-arrow">=></span>{/if}
                                      <span
                                        class="w-num {trend.wave_highlight == n ? 'highlight' : ''}"
                                        >{n}</span
                                      >
                                    {/each}
                                  </div>
                                {/if}
                                {#if trend.wave_image}
                                  <div class="t-img-box">
                                    <img
                                      src={trend.wave_image}
                                      alt="{tf} Wave"
                                      class="clickable-image"
                                      loading="lazy"
                                      on:click={() => openModal(trend.wave_image, `${tf} 波浪分析`)}
                                      on:keypress={() =>
                                        openModal(trend.wave_image, `${tf} 波浪分析`)}
                                      role="button"
                                      tabindex="0"
                                    />
                                  </div>
                                {/if}
                              </div>
                            {/if}
                          </div>
                        </div>
                      {/if}
                    {/each}
                  </div>
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

{#if enlargedImage}
  <div
    class="image-modal"
    on:click={closeModal}
    role="button"
    tabindex="0"
    on:keydown={e => e.key === 'Escape' && closeModal()}
  >
    <div
      class="image-modal-content"
      on:click|stopPropagation
      role="button"
      tabindex="0"
      on:keypress|stopPropagation
    >
      <button class="image-modal-close" on:click={closeModal}>×</button>
      <img src={enlargedImage} alt={enlargedImageTitle} />
      {#if enlargedImageTitle}
        <div class="image-modal-caption">{enlargedImageTitle}</div>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Keeps existing styles and adds Modal Styles */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.85);
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem;
    backdrop-filter: blur(5px);
  }

  .image-modal-content {
    position: relative;
    max-width: 95%;
    max-height: 95%;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .image-modal-content img {
    max-width: 100%;
    max-height: 85vh;
    object-fit: contain;
    border-radius: 8px;
    box-shadow:
      0 20px 25px -5px rgba(0, 0, 0, 0.1),
      0 10px 10px -5px rgba(0, 0, 0, 0.04);
  }

  .image-modal-caption {
    color: white;
    margin-top: 1rem;
    font-size: 1.1rem;
    font-weight: 600;
  }

  .image-modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 2.5rem;
    cursor: pointer;
    line-height: 1;
    padding: 0;
    opacity: 0.8;
    transition: opacity 0.2s;
  }

  .image-modal-close:hover {
    opacity: 1;
  }

  .clickable-image {
    cursor: zoom-in;
    transition: transform 0.2s;
  }
  .clickable-image:hover {
    transform: scale(1.02);
  }

  .shared-view-container {
    max-width: 850px;
    margin: 3rem auto;
    padding: 0 1.25rem;
    min-height: 400px;
    font-family:
      'Inter',
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      Roboto,
      sans-serif;
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
  .side-tag.long {
    background: #fee2e2;
    color: #991b1b;
  }
  .side-tag.short {
    background: #dcfce7;
    color: #166534;
  }

  .pnl-value {
    font-size: 2.5rem;
    font-weight: 900;
  }
  .pnl-value.profit {
    color: #10b981;
  }
  .pnl-value.loss {
    color: #ef4444;
  }

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

  .duration-badge {
    color: #2563eb !important;
    background: #eff6ff;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    font-size: 0.95rem !important;
    border: 1px solid #bfdbfe;
  }

  .sparkline-container-shared {
    margin-top: 0.5rem;
    display: flex;
    justify-content: flex-end;
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

  .check-tag,
  .signal-tag-item {
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

  .img-preview-box {
    position: relative;
    background: #f1f5f9;
    border-radius: 8px;
    overflow: hidden;
    min-height: 150px;
    margin-bottom: 1rem;
  }

  .img-preview-box::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(90deg, #f1f5f9 25%, #e2e8f0 50%, #f1f5f9 75%);
    background-size: 200% 100%;
    animation: skeleton-pulse 1.5s infinite;
    z-index: 0;
  }

  .img-preview-box img {
    width: 100%;
    border-radius: 8px;
    display: block;
    position: relative;
    z-index: 1;
  }

  .notes-content {
    background: #f8fafc;
    padding: 1.75rem;
    border-radius: 1rem;
    line-height: 1.7;
    color: #334155;
    border: 1px solid #f1f5f9;
    font-family: inherit !important;
  }

  /* Quill Editor Style Reset for shared view */
  .ql-editor :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 12px;
    margin: 1rem 0;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
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
    background: #f1f5f9;
    min-height: 200px;
  }

  .image-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(90deg, #f1f5f9 25%, #e2e8f0 50%, #f1f5f9 75%);
    background-size: 200% 100%;
    animation: skeleton-pulse 1.5s infinite;
    z-index: 0;
  }

  .image-card img {
    width: 100%;
    display: block;
    position: relative;
    z-index: 1;
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

  @keyframes spin {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }

  .strategy-badge {
    padding: 0.25rem 0.75rem;
    border-radius: 6px;
    color: white;
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .strategy-badge.expert {
    background: #10b981;
  }
  .strategy-badge.elite {
    background: #3b82f6;
  }
  .strategy-badge.legend {
    background: #f59e0b;
  }

  .session-block {
    margin-top: 1.5rem;
    padding: 1.25rem;
    border-radius: 12px;
    border-left: 5px solid #e2e8f0;
    background: #f8fafc;
  }
  .session-block.asian {
    border-left-color: #3b82f6;
  }
  .session-block.european {
    border-left-color: #f59e0b;
  }
  .session-block.us {
    border-left-color: #ef4444;
  }

  .session-notes {
    white-space: pre-wrap;
    font-family: inherit;
    line-height: 1.6;
    color: #475569;
    margin-top: 0.75rem;
    font-size: 0.95rem;
  }

  /* Trends Grid */
  .trends-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.5rem;
    margin-top: 1.5rem;
  }

  @media (max-width: 600px) {
    .trends-grid {
      grid-template-columns: 1fr;
      gap: 1rem;
    }
  }

  .trend-card {
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    overflow: hidden;
    transition: all 0.2s ease;
  }

  .trend-card:hover {
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  }

  .trend-card.long {
    border-left: 5px solid #ef4444;
  }
  .trend-card.short {
    border-left: 5px solid #10b981;
  }

  .trend-header {
    background: #f8fafc;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .tf-badge {
    font-weight: 800;
    color: #475569;
    font-size: 1rem;
  }

  .dir-badge {
    font-size: 0.8rem;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    font-weight: 700;
    color: white;
  }
  .dir-badge.long {
    background: #ef4444;
  }
  .dir-badge.short {
    background: #10b981;
  }

  .trend-body {
    padding: 1rem;
  }

  .t-section {
    margin-top: 1rem;
    padding-top: 0.75rem;
    border-top: 1px dashed #e2e8f0;
  }

  .section-title {
    font-size: 0.8rem;
    font-weight: 700;
    color: #64748b;
    margin-bottom: 0.5rem;
  }

  .t-img-box {
    margin-top: 0.5rem;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #f1f5f9;
    position: relative;
    background: #f1f5f9; /* Placeholder color */
    min-height: 100px; /* Minimum height to prevent collapse */
  }

  /* Skeleton loading animation */
  .t-img-box::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(90deg, #f1f5f9 25%, #e2e8f0 50%, #f1f5f9 75%);
    background-size: 200% 100%;
    animation: skeleton-pulse 1.5s infinite;
    z-index: 0;
  }

  @keyframes skeleton-pulse {
    0% {
      background-position: 200% 0;
    }
    100% {
      background-position: -200% 0;
    }
  }

  .img-label {
    position: absolute;
    top: 5px;
    left: 5px;
    background: rgba(0, 0, 0, 0.6);
    color: white;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    backdrop-filter: blur(2px);
    z-index: 2;
  }

  .t-img-box img {
    width: 100%;
    height: auto;
    display: block;
    position: relative;
    z-index: 1;
    min-height: 1px;
  }

  .t-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0.5rem;
  }

  .t-tag {
    font-size: 0.75rem;
    background: #f1f5f9;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    color: #334155;
    font-weight: 600;
  }

  .t-wave-nums {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .w-num {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f1f5f9;
    border-radius: 50%;
    font-size: 0.8rem;
    font-weight: 700;
    color: #64748b;
  }
  .w-arrow {
    color: #94a3b8;
    font-weight: 800;
    font-size: 0.75rem;
    align-self: center;
  }
  .w-num.highlight {
    background: #fee2e2;
    color: #ef4444;
  }

  /* Responsive Optimizations */
  @media (max-width: 768px) {
    .shared-view-container {
      margin: 1rem auto;
      padding: 0 0.75rem;
    }

    .card {
      padding: 1.25rem;
      border-radius: 1rem;
    }

    .view-header {
      margin-bottom: 1.5rem;
      padding-bottom: 1.25rem;
      flex-direction: column;
      align-items: flex-start;
      gap: 1rem;
    }

    .title-section h1 {
      font-size: 1.4rem;
    }

    .info-grid {
      grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
      gap: 0.75rem;
    }

    .image-modal {
      padding: 1rem;
    }

    .image-modal-close {
      top: -35px;
      right: 5px;
      font-size: 2rem;
    }

    .image-modal-caption {
      font-size: 0.95rem;
      text-align: center;
    }

    .notes-content {
      padding: 1.25rem;
      font-size: 0.95rem;
    }

    .session-block {
      padding: 1rem;
    }
  }

  /* Special optimization for narrow screens (e.g. Fold outer screen) */
  @media (max-width: 400px) {
    .title-section h1 {
      font-size: 1.25rem;
    }

    .symbol-tag,
    .side-tag {
      font-size: 0.75rem;
      padding: 0.25rem 0.6rem;
    }

    .pnl-value {
      font-size: 1.5rem;
    }

    .image-gallery {
      gap: 1.5rem;
    }
  }

  /* Optimization for larger folding screens (e.g. Fold internal screen) */
  @media (min-width: 601px) and (max-width: 1024px) {
    .trends-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .shared-view-container {
      max-width: 95%;
    }
  }
</style>
