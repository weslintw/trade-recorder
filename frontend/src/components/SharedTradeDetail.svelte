<script>
  import { getStrategyLabel, determineMarketSession } from '../lib/utils';
  import Sparkline from './Sparkline.svelte';
  import { imagesAPI } from '../lib/api';

  export let trade;
  export let openModal = () => {};

  function parseJSON(str, defaultValue = null) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

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

  function lazyLoadHTML(html) {
    if (!html) return '';
    return html.replace(/<img /g, '<img loading="lazy" ');
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
    M1: '1分',
    M5: '5分',
    M15: '15分',
    M30: '30分',
    H1: '1小時',
    H4: '4小時',
    D1: '天',
    SuperK: '超k',
  };

  const legendChecklist = {
    item_618_786: '王者出現回調618或786',
    item_che: '大時區破[測]破',
    item_de: '整理段的ABC[D][E]',
  };

  $: signals = parseJSON(trade.entry_signals, []);
  $: checklist = parseJSON(trade.entry_checklist, {});
  $: patterns = parseJSON(trade.entry_pattern, []);
  $: legendImages = parseJSON(trade.legend_images, []);
  $: expertImages = parseJSON(trade.expert_images, []);
  $: eliteImages = parseJSON(trade.elite_images, []);

  const colorTagMeanings = {
    green: '有照標準進單',
    yellow: '有討論空間',
    red: '衝動，沒有照標準',
  };

  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) return src;
    return imagesAPI.getUrl(src);
  }
</script>

<div class="trade-detail-view card">
  <div class="view-header">
    <div class="title-section">
      <span class="symbol-tag">{trade.symbol || '---'}</span>
      <span class="side-tag {trade.side || ''}"
        >{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span
      >
      {#if trade.color_tag}<span
          class="color-dot {trade.color_tag}"
          title={colorTagMeanings[trade.color_tag] || '顏色標記'}
        ></span>{/if}
      <h1>交易紀錄詳情</h1>
    </div>
    <div class="pnl-section">
      {#if trade.trade_type === 'actual'}
        {#if trade.pnl !== undefined && trade.pnl !== null}
          <div class="pnl-value {trade.pnl >= 0 ? 'profit' : 'loss'}">
            {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl).toFixed(2)}
          </div>
        {/if}
        {#if trade.pnl_series}
          <div class="sparkline-container-shared">
            <Sparkline
              data={trade.pnl_series}
              isOpen={trade.trade_type === 'actual' && !trade.exit_time}
              width={120}
              height={40}
            />
          </div>
        {/if}
      {:else if trade.trade_type === 'observation'}
        <span class="journal-detail-badge">📓 紀錄 / 記事</span>
      {/if}
    </div>
  </div>

  <div class="info-grid extended">
    <div class="info-row-group">
      <div class="info-item">
        <label>交易品種</label><span class="symbol-inline-tag">{trade.symbol}</span>
      </div>
      <div class="info-item">
        <label>做多或做空</label><span
          >{trade.side === 'long'
            ? '做多 (Long)'
            : trade.side === 'short'
              ? '做空 (Short)'
              : 'NA'}</span
        >
      </div>
      <div class="info-item"><label>手數</label><span>{trade.lot_size || '0.00'}</span></div>
      <div class="info-item">
        <label>TICKET</label><span class="ticket-val">{trade.ticket || 'NA'}</span>
      </div>
    </div>

    {#if trade.trade_type === 'actual'}
      <div class="info-row-divider"></div>

      <div class="info-row-group">
        <div class="info-item">
          <label>進場價格</label><span class="value-highlight">{trade.entry_price || '0.00'}</span>
        </div>
        <div class="info-item"><label>初始 S L</label><span>{trade.initial_sl || 'NA'}</span></div>
        <div class="info-item">
          <label>平倉價格</label><span class="value-highlight">{trade.exit_price || 'NA'}</span>
        </div>
        <div class="info-item"><label>平倉 S L</label><span>{trade.exit_sl || 'NA'}</span></div>
      </div>

      <div class="info-row-divider"></div>

      <div class="info-row-group">
        <div class="info-item">
          <label>盈虧金額</label><span class="rr-value {trade.pnl >= 0 ? 'profit' : 'loss'}"
            >{trade.pnl !== undefined && trade.pnl !== null ? trade.pnl.toFixed(2) : '--'}</span
          >
        </div>
        <div class="info-item">
          <label>盈虧點數</label><span
            >{trade.pnl_points != null ? trade.pnl_points.toFixed(1) : 'NA'}</span
          >
        </div>
        <div class="info-item">
          <label>子彈大小 (BULLET)</label><span
            >{trade.bullet_size != null ? trade.bullet_size : '自動計算'}</span
          >
        </div>
        <div class="info-item">
          <label>風報比 (R:R)</label><span class="rr-value-pills"
            >{trade.rr_ratio != null ? trade.rr_ratio.toFixed(2) : '自動計算'}</span
          >
        </div>
      </div>

      <div class="info-row-divider"></div>
    {/if}

    <div class="info-row-group">
      <div class="info-item">
        <label>開倉時間 (UTC+8)</label><span>{formatDate(trade.entry_time)}</span>
      </div>
      <div class="info-item">
        <label>市場時段與規劃</label>
        <div class="mock-session-display">
          <span class="session-label-btn {determineMarketSession(trade.entry_time)}"
            >{getMarketSessionLabel(trade).replace(/^[🌏🌍🌎]\s/, '')}</span
          >
          <span class="session-time-text">
            {#if determineMarketSession(trade.entry_time) === 'asian'}08:00 - 15:00{:else if determineMarketSession(trade.entry_time) === 'european'}16:00
              - 00:00{:else}21:00 - 05:00{/if}
            · 冬令時間
          </span>
        </div>
      </div>
    </div>

    <div class="info-row-divider"></div>

    <div class="info-row-group">
      <div class="info-item">
        <label>平倉時間 (UTC+8)</label><span
          >{trade.exit_time ? formatDate(trade.exit_time) : '--'}</span
        >
      </div>
      <div class="info-item">
        <label>持單時間</label><span class="duration-badge-pill"
          >{calculateDuration(trade.entry_time, trade.exit_time) || 'NA'}</span
        >
      </div>
    </div>

    <div class="info-row-divider"></div>

    <div class="info-row-group">
      <div class="info-item full-width-item">
        <label class="rocket-header">🚀 進場分析</label>
        <div class="analysis-sub-flex horizontal-layout">
          <div class="analysis-sub-group">
            <label class="sub-label">🎯 進場種類</label>
            <div class="mock-strategy-btns">
              <span class="strat-btn {trade.entry_strategy === 'expert' ? 'active' : ''}">達人</span
              >
              <span class="strat-btn {trade.entry_strategy === 'elite' ? 'active' : ''}">菁英</span>
              <span class="strat-btn {trade.entry_strategy === 'legend' ? 'active' : ''}">傳奇</span
              >
            </div>
          </div>
          <div class="analysis-sub-group">
            <label class="sub-label">🕒 進場時區</label>
            <div class="mock-tf-pills">
              {#each ['M1', 'M5', 'M15', 'H1', 'H4', 'D1', ...(trade.entry_strategy === 'legend' ? ['SuperK'] : [])] as tf}
                <span
                  class="tf-pill {trade.entry_timeframe === tf ||
                  trade.entry_timeframe === tf.toLowerCase()
                    ? 'active'
                    : ''}">{timeframeLabels[tf] || tf}</span
                >
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
      {#if trade.entry_strategy === 'legend'}
        <div class="legend-specifics">
          <label class="legend-section-title">👑 傳奇策略圖面</label>
          <div class="legend-images-grid">
            {#if trade.legend_king_image}
              <div class="legend-card">
                <div class="legend-header">
                  <span>王者 (King)</span>
                  {#if trade.legend_king_htf}<span class="htf-tag">{trade.legend_king_htf}</span
                    >{/if}
                </div>
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <div
                  class="legend-img-container"
                  on:click={() =>
                    openModal(getImageUrl(trade.legend_king_image), '王者 (King) 圖面')}
                >
                  <img
                    src={getImageUrl(trade.legend_king_image)}
                    alt="Legend King"
                    loading="lazy"
                  />
                </div>
              </div>
            {/if}

            {#if trade.legend_htf_image}
              <div class="legend-card">
                <div class="legend-header">
                  <span>傳奇觀察</span>
                  {#if trade.legend_htf}<span class="htf-tag">{trade.legend_htf}</span>{/if}
                </div>
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <div
                  class="legend-img-container"
                  on:click={() => openModal(getImageUrl(trade.legend_htf_image), '傳奇觀察圖面')}
                >
                  <img src={getImageUrl(trade.legend_htf_image)} alt="Legend HTF" loading="lazy" />
                </div>
              </div>
            {/if}

            <!-- Support both new legend_images array and old entry_strategy_image -->
            {#if legendImages && legendImages.length > 0}
              {#each legendImages as legendImg, idx}
                {#if legendImg?.image}
                  <div class="legend-card">
                    <div class="legend-header">
                      <span>傳奇觀察圖 {idx + 1}</span>
                    </div>
                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                    <div
                      class="legend-img-container"
                      on:click={() =>
                        openModal(getImageUrl(legendImg.image), `傳奇觀察圖 ${idx + 1}`)}
                    >
                      <img
                        src={getImageUrl(legendImg.image)}
                        alt={`傳奇觀察圖 ${idx + 1}`}
                        loading="lazy"
                      />
                    </div>
                  </div>
                {/if}
              {/each}
            {:else if trade.entry_strategy_image}
              <!-- Fallback to old single image format -->
              <div class="legend-card">
                <div class="legend-header">
                  <span>傳奇全局圖</span>
                </div>
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <div
                  class="legend-img-container"
                  on:click={() => openModal(getImageUrl(trade.entry_strategy_image), '傳奇全局圖')}
                >
                  <img
                    src={getImageUrl(trade.entry_strategy_image)}
                    alt="Legend General"
                    loading="lazy"
                  />
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      {#if trade.entry_strategy === 'expert' && expertImages && expertImages.length > 0}
        <div class="legend-specifics">
          <label class="legend-section-title">⚡ 達人觀察圖面</label>
          <div class="legend-images-grid">
            {#each expertImages as expertImg, idx}
              {#if expertImg?.image}
                <div class="legend-card">
                  <div class="legend-header">
                    <span>達人觀察圖 {idx + 1}</span>
                  </div>
                  <!-- svelte-ignore a11y-click-events-have-key-events -->
                  <div
                    class="legend-img-container"
                    on:click={() => openModal(getImageUrl(expertImg.image), `達人觀察圖 ${idx + 1}`)}
                  >
                    <img
                      src={getImageUrl(expertImg.image)}
                      alt={`達人觀察圖 ${idx + 1}`}
                      loading="lazy"
                    />
                  </div>
                </div>
              {/if}
            {/each}
          </div>
        </div>
      {/if}

      {#if trade.entry_strategy === 'elite' && eliteImages && eliteImages.length > 0}
        <div class="legend-specifics">
          <label class="legend-section-title">💎 菁英觀察圖面</label>
          <div class="legend-images-grid">
            {#each eliteImages as eliteImg, idx}
              {#if eliteImg?.image}
                <div class="legend-card">
                  <div class="legend-header">
                    <span>菁英觀察圖 {idx + 1}</span>
                  </div>
                  <!-- svelte-ignore a11y-click-events-have-key-events -->
                  <div
                    class="legend-img-container"
                    on:click={() => openModal(getImageUrl(eliteImg.image), `菁英觀察圖 ${idx + 1}`)}
                  >
                    <img
                      src={getImageUrl(eliteImg.image)}
                      alt={`菁英觀察圖 ${idx + 1}`}
                      loading="lazy"
                    />
                  </div>
                </div>
              {/if}
            {/each}
          </div>
        </div>
      {/if}

      <div class="analysis-grid">
        {#if signals && signals.length > 0}
          <div class="analysis-item">
            <label>選用訊號</label>
            <div class="tags-container">
              {#each signals as sig}
                {@const sigName = typeof sig === 'string' ? sig : sig.name}
                {@const primaryImg = typeof sig === 'object' ? sig.image || sig.originalImage : null}
                {@const allImages = (sig.images && Array.isArray(sig.images)) ? sig.images : (primaryImg ? [{image: primaryImg}] : [])}
                
                {#if allImages.length > 0}
                  {#each allImages as img, imgIdx}
                    <span
                      class="analysis-tag has-img"
                      on:click={() => openModal(getImageUrl(img.image), `${expertSignals[sigName] || sigName} - 圖 ${imgIdx + 1}`)}
                    >
                      <img src={getImageUrl(img.image)} alt={sigName} class="tag-icon" />
                      {expertSignals[sigName] || sigName} {#if allImages.length > 1}({imgIdx + 1}){/if}
                    </span>
                  {/each}
                {:else}
                  <span class="analysis-tag">
                    {expertSignals[sigName] || sigName}
                  </span>
                {/if}
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
                {@const primaryImg = typeof pat === 'object' ? pat.image || pat.originalImage : null}
                {@const allImages = (pat.images && Array.isArray(pat.images)) ? pat.images : (primaryImg ? [{image: primaryImg}] : [])}
                
                {#if allImages.length > 0}
                  {#each allImages as img, imgIdx}
                    <span
                      class="analysis-tag pattern has-img"
                      on:click={() => openModal(getImageUrl(img.image), `${patName} - 圖 ${imgIdx + 1}`)}
                    >
                      <img src={getImageUrl(img.image)} alt={patName} class="tag-icon" />
                      {patName} {#if allImages.length > 1}({imgIdx + 1}){/if}
                    </span>
                  {/each}
                {:else}
                  <span class="analysis-tag pattern">
                    {patName}
                  </span>
                {/if}
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
                <div class="check-chip">
                  ✅ {eliteChecklist[key] || legendChecklist[key] || key}
                </div>
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
    <h3>📓 紀事</h3>
    {#if trade.journal}
      <div class="notes-content ql-editor">{@html lazyLoadHTML(trade.journal)}</div>
    {:else}
      <p class="empty-placeholder">無紀事紀錄</p>
    {/if}
  </div>

  {#if trade.trade_type === 'actual' || !trade.trade_type}
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
  {/if}
  {#if trade.images && trade.images.length > 0}
    <div class="section-box">
      <h3>🖼️ 圖表截圖</h3>
      <div class="image-gallery">
        {#each trade.images as img}
          {#if img && img.image_path}
            <div class="image-card">
              <img
                src={imagesAPI.getUrl(img.image_path)}
                alt="Trade Chart"
                class="clickable-image"
                loading="lazy"
                on:click={() => openModal(imagesAPI.getUrl(img.image_path), img.image_type)}
              />
            </div>
          {/if}
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .card {
    background: white;
    border-radius: 1.5rem;
    padding: 2.5rem;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05);
    border: 1px solid #f1f5f9;
    margin-bottom: 2rem;
  }
  .view-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 2rem;
    border-bottom: 1px solid #f1f5f9;
    padding-bottom: 1.5rem;
  }
  .symbol-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #4f46e5;
    color: white;
    padding: 0.25rem 0.75rem;
    border-radius: 6px;
    font-weight: 800;
    font-size: 0.875rem;
    line-height: 1;
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
  .info-item label {
    color: #64748b;
    font-weight: 700;
    text-transform: uppercase;
    font-size: 0.75rem;
    margin-bottom: 0.6rem;
    display: block;
  }
  .info-item .value-highlight {
    font-size: 1.1rem;
    font-weight: 800;
    color: #1e293b;
    background: #f8fafc;
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    display: inline-block;
    min-width: 100px;
  }
  .full-width-item {
    grid-column: 1 / -1;
  }

  .symbol-inline-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 800;
    color: #1e293b;
    padding: 2px 6px;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 4px;
    line-height: 1;
  }
  .ticket-val {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.8rem;
    color: #94a3b8;
  }
  .rr-value {
    font-weight: 700;
  }
  .rr-value.profit {
    color: #10b981;
  }
  .rr-value.loss {
    color: #ef4444;
  }
  .rr-value-pills {
    background: #f5f3ff;
    color: #5b21b6;
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 800;
  }
  .duration-badge-pill {
    background: #eff6ff;
    color: #2563eb;
    padding: 6px 16px;
    border-radius: 8px;
    font-weight: 700;
    border: 1px solid #dbeafe;
    display: inline-block;
  }

  .side-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .side-tag.long {
    background: #fee2e2;
    color: #991b1b;
  }
  .side-tag.short {
    background: #dcfce7;
    color: #166534;
  }

  .color-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    display: inline-block;
    margin-right: 8px;
  }
  .color-dot.green {
    background: #22c55e;
  }
  .color-dot.yellow {
    background: #eab308;
  }
  .color-dot.red {
    background: #ef4444;
  }

  .journal-detail-badge {
    background: #f3f4f6;
    color: #374151;
    padding: 0.5rem 1rem;
    border-radius: 8px;
    font-weight: 700;
    border: 1px solid #d1d5db;
  }

  .section-box {
    margin: 2.5rem 0;
  }
  .section-box h3 {
    font-size: 1.25rem;
    font-weight: 800;
    color: #1e293b;
    margin-bottom: 1.25rem;
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }
  .notes-content {
    padding: 1.5rem;
    background: #f8fafc;
    border-radius: 1rem;
    line-height: 1.6;
  }

  .image-gallery {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
  }
  .image-card {
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
  }
  .clickable-image {
    width: 100%;
    height: 150px;
    object-fit: cover;
    cursor: pointer;
    transition: transform 0.2s;
  }
  .clickable-image:hover {
    transform: scale(1.05);
  }

  .mock-session-display {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    background: transparent;
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    border: 2px solid #e0f2fe;
    width: fit-content;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .session-label-btn {
    padding: 4px 12px;
    border-radius: 6px;
    font-weight: 800;
    font-size: 0.8rem;
    color: white;
    line-height: 1.2;
    text-align: center;
    display: inline-block;
  }
  .session-label-btn.asian {
    background: #3b82f6;
  }
  .session-label-btn.european {
    background: #d97706;
  }
  .session-label-btn.us {
    background: #00b4ff;
  }
  .session-time-text {
    font-size: 0.85rem;
    color: #475569;
    font-weight: 600;
    white-space: nowrap;
  }

  .rocket-header {
    font-size: 1.25rem !important;
    font-weight: 800;
    color: #2d3748;
    margin-bottom: 1.5rem !important;
  }
  .analysis-sub-flex.horizontal-layout {
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 3rem;
    margin-top: 1rem;
    align-items: flex-end;
  }
  .analysis-sub-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .mock-strategy-btns {
    display: flex;
    gap: 0.5rem;
  }
  .strat-btn {
    padding: 0.6rem 1.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    color: #4a5568;
    font-size: 1rem;
    font-weight: 600;
    background: white;
    text-align: center;
    min-width: 80px;
  }
  .strat-btn.active {
    border-color: #6366f1;
    background: #e0e7ff;
    color: #4338ca;
    box-shadow: none;
  }

  .mock-tf-pills {
    display: flex;
    background: #1a1a1a;
    border-radius: 8px;
    padding: 4px;
    gap: 2px;
    width: fit-content;
    align-items: center;
  }
  .tf-pill {
    color: #888;
    padding: 6px 12px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: default;
    transition: all 0.2s;
    white-space: nowrap;
    border-radius: 6px;
  }
  .tf-pill.active {
    background: #333;
    color: #60a5fa !important;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }

  .analysis-section {
    background: #fcfdfe;
    border: 1px solid #edf2f7;
    margin-top: 1.5rem;
    padding: 1.5rem;
    border-radius: 12px;
    box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.02);
  }
  .analysis-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    margin-top: 1rem;
  }
  .analysis-item label {
    font-size: 0.75rem;
    font-weight: 800;
    color: #4a5568;
    margin-bottom: 0.75rem;
    display: block;
  }
  .tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
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
  .analysis-tag.has-img {
    cursor: pointer;
    transition: transform 0.2s;
  }
  .analysis-tag.has-img:hover {
    transform: translateY(-1px);
    background: #dbeafe;
  }
  .tag-icon {
    width: 20px;
    height: 20px;
    border-radius: 3px;
    object-fit: cover;
    border: 1px solid rgba(0, 0, 0, 0.05);
  }
  .analysis-tag.pattern {
    background: #faf5ff;
    color: #6b46c1;
  }

  .checklist-display {
    margin-top: 1.5rem;
    padding-top: 1.5rem;
    border-top: 1px dashed #e2e8f0;
  }
  .checklist-display label {
    font-size: 0.75rem;
    font-weight: 800;
    color: #4a5568;
    margin-bottom: 0.75rem;
    display: block;
  }
  .check-items {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }
  .check-chip {
    background: #f0fdf4;
    color: #166534;
    padding: 6px 14px;
    border-radius: 99px;
    font-size: 0.85rem;
    font-weight: 600;
    border: 1px solid #dcfce7;
  }

  .empty-placeholder {
    color: #94a3b8;
    font-style: italic;
  }

  /* Legend Specific Styles */
  .legend-specifics {
    margin-bottom: 2rem;
  }
  .legend-section-title {
    font-size: 0.9rem;
    font-weight: 800;
    color: #475569;
    margin-bottom: 1rem;
    display: block;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .legend-images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.5rem;
  }
  .legend-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  .legend-header {
    padding: 0.75rem 1rem;
    background: #f1f5f9;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 700;
    font-size: 0.85rem;
    color: #64748b;
  }
  .htf-tag {
    background: #6366f1;
    color: white;
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.7rem;
  }
  .legend-img-container {
    cursor: pointer;
    overflow: hidden;
    height: 200px;
    background: #0f172a;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .legend-img-container img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.3s ease;
  }
  .legend-img-container:hover img {
    transform: scale(1.05);
  }

  /* Ensure images in rich text don't overflow */
  .notes-content :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 1rem 0;
  }

  /* Handle very wide images by allowing them to be viewed properly */
  .ql-editor :global(img) {
    max-width: 100%;
    height: auto;
  }

  @media (max-width: 768px) {
    .info-row-group {
      grid-template-columns: 1fr 1fr;
    }
    .analysis-sub-flex.horizontal-layout {
      gap: 1.5rem;
    }
    .analysis-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
