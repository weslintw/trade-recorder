<script>
  import { onMount, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { Link, navigate } from 'svelte-routing';
  import { tradesAPI, tagsAPI, imagesAPI, dailyPlansAPI } from '../lib/api';
  import { SYMBOLS, MARKET_SESSIONS } from '../lib/constants';
  import { determineMarketSession, getStrategyLabel, parseJSONSafe } from '../lib/utils';
  import ImageAnnotator from './ImageAnnotator.svelte';

  let showAnnotator = false;
  let enlargedOriginalImage = null;
  let enlargedImageContext = null; // { tradeId, imageIndex, type: 'general' | 'expert' | ... }
  import { selectedSymbol, selectedAccountId } from '../lib/stores';

  export let isCompact = false;

  let trades = [];
  let allPlans = [];
  let loading = true;
  let pagination = {
    page: 1,
    page_size: 20,
    total: 0,
  };
  let allTags = [];

  // 篩選條件
  let filters = {
    symbol: '',
    side: '',
    tag: '',
    start_date: '',
    end_date: '',
  };

  onMount(() => {
    loadTags();
  });

  let selectedImage = null;
  let modalTitle = '查看圖片';

  // 當全局品種改變時，更新篩選器並重新載入
  $: if ($selectedSymbol || $selectedAccountId) {
    filters.symbol = $selectedSymbol;
    pagination.page = 1;
    loadTrades();
    loadPlans();
  }

  async function loadPlans() {
    try {
      const response = await dailyPlansAPI.getAll({
        account_id: $selectedAccountId,
        page_size: 100,
      });
      allPlans = response.data.data || [];
    } catch (error) {
      console.error('載入規劃失敗:', error);
    }
  }

  function getMatchedPlan(trade) {
    if (!trade.entry_time || !trade.market_session || allPlans.length === 0) return null;

    try {
      const tradeDate = new Date(trade.entry_time).toISOString().slice(0, 10);
      return allPlans.find(plan => {
        const planDate = new Date(plan.plan_date).toISOString().slice(0, 10);
        if (planDate !== tradeDate) return false;

        // 同時匹配品種 (舊資料預設 XAUUSD)
        const planSymbol = plan.symbol || SYMBOLS[0];
        if (planSymbol !== trade.symbol) return false;

        if (plan.market_session === 'all') {
          // 新格式：檢查該時段在 JSON 中是否有任何趨勢或備註
          try {
            const trendData = JSON.parse(plan.trend_analysis || '{}');
            const sessionData = trendData[trade.market_session];
            // 如果該時段有備註或任何時區有方向，視為匹配
            return (
              sessionData &&
              (sessionData.notes ||
                (sessionData.trends && Object.values(sessionData.trends).some(t => t.direction)))
            );
          } catch (e) {
            return false;
          }
        } else {
          // 舊格式：直接匹配時段
          return plan.market_session === trade.market_session;
        }
      });
    } catch (e) {
      return null;
    }
  }

  function getMarketSessionLabel(trade) {
    let session = trade.market_session;
    // 如果資料庫中沒有時段資料，根據時間即時計算
    if (!session && trade.entry_time) {
      session = determineMarketSession(trade.entry_time);
    }
    const s = MARKET_SESSIONS.find(s => s.value === session);
    return s ? s.label : session || '未設定';
  }

  async function loadTrades() {
    try {
      loading = true;
      const params = {
        account_id: $selectedAccountId,
        page: pagination.page,
        page_size: pagination.page_size,
        ...filters,
      };

      // 移除空值
      Object.keys(params).forEach(key => {
        if (params[key] === '') delete params[key];
      });

      const response = await tradesAPI.getAll(params);
      trades = response.data.data || [];
      pagination = response.data.pagination;
    } catch (error) {
      console.error('載入交易列表失敗:', error);
      alert('載入交易列表失敗');
    } finally {
      loading = false;
    }
  }

  async function loadTags() {
    try {
      const response = await tagsAPI.getAll();
      allTags = response.data || [];
    } catch (error) {
      console.error('載入標籤失敗:', error);
    }
  }

  async function deleteTrade(id) {
    if (!confirm('確定要刪除此交易紀錄嗎？')) return;

    try {
      await tradesAPI.delete(id);
      alert('刪除成功');
      loadTrades();
    } catch (error) {
      console.error('刪除失敗:', error);
      alert('刪除失敗：' + (error.response?.data?.error || error.message));
    }
  }

  async function syncTrade(id) {
    try {
      await tradesAPI.sync(id); // 此處需要確認 lib/api.js 是否有此方法
      alert('已送出手動同步請求，請稍候幾秒後重新載入或查看更新');
      // 延遲一段時間後自動重新加載
      setTimeout(() => loadTrades(), 3000);
    } catch (error) {
      console.error('同步失敗:', error);
      alert('同步失敗：' + (error.response?.data?.error || error.message));
    }
  }

  function applyFilters() {
    pagination.page = 1;
    loadTrades();
  }

  function clearFilters() {
    filters = {
      symbol: '',
      side: '',
      tag: '',
      start_date: '',
      end_date: '',
    };
    pagination.page = 1;
    loadTrades();
  }

  function changePage(newPage) {
    pagination.page = newPage;
    loadTrades();
  }

  function formatDate(dateString) {
    return new Date(dateString).toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  }


  function openImageModal(imagePath, title = '查看圖片', context = null, originalPath = null) {
    if (!imagePath) return;
    modalTitle = title;
    enlargedImageContext = context;
    enlargedOriginalImage = originalPath || imagePath;
    selectedImage = imagePath.startsWith('http') || imagePath.startsWith('data:') || imagePath.startsWith('blob:') 
      ? imagePath 
      : imagesAPI.getUrl(imagePath);
    showAnnotator = false;
  }

  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  async function handleAnnotatedImage(annotatedImageSrc) {
    if (!enlargedImageContext) return;
    try {
      const res = await fetch(annotatedImageSrc);
      const blob = await res.blob();
      const file = new File([blob], 'annotated_list.png', { type: 'image/png' });

      const uploadData = new FormData();
      uploadData.append('image', file);
      uploadData.append('symbol', 'annotated'); 

      const uploadRes = await imagesAPI.upload(uploadData);
      const serverPath = uploadRes.data.path;

      const { tradeId, type, index, name } = enlargedImageContext;
      const fullTradeRes = await tradesAPI.getOne(tradeId);
      const fullTrade = fullTradeRes.data;
      
      const payload = sanitizeTradePayload(fullTrade);
      const originalPath = enlargedOriginalImage;

      // 1. 核心邏輯：更新指定的欄位
      if (type === 'general') {
        if (payload.images && payload.images[index]) {
          payload.images[index].image_path = serverPath;
        }
      } else if (type === 'signal') {
        const signals = parseJSONSafe(payload.entry_signals, []);
        const sIdx = signals.findIndex(s => s.name === name);
        if (sIdx >= 0) {
          const sigImages = signals[sIdx].images || (signals[sIdx].image ? [{image: signals[sIdx].image}] : []);
          if (sigImages[index]) {
            sigImages[index].image = serverPath;
          }
          signals[sIdx].images = sigImages;
          payload.entry_signals = JSON.stringify(signals);
        }
      } else if (type === 'pattern') {
        const patterns = parseJSONSafe(payload.entry_pattern, []);
        const pIdx = patterns.findIndex(p => p.name === name);
        if (pIdx >= 0) {
          const patImages = patterns[pIdx].images || (patterns[pIdx].image ? [{image: patterns[pIdx].image}] : []);
          if (patImages[index]) {
            patImages[index].image = serverPath;
          }
          patterns[pIdx].images = patImages;
          payload.entry_pattern = JSON.stringify(patterns);
        }
      } else if (type === 'strategy') {
        payload.entry_strategy_image = serverPath;
      } else if (type === 'legend_htf') {
        payload.legend_htf_image = serverPath;
      } else if (type === 'legend_king') {
        payload.legend_king_image = serverPath;
      } else if (type === 'legend_images') {
        const lImages = parseJSONSafe(payload.legend_images, []);
        if (lImages[index]) {
          lImages[index].image = serverPath;
        }
        payload.legend_images = JSON.stringify(lImages);
      }

      // 2. 聰明邏輯：全域掃描。如果其他欄位也使用了同一個原始圖片路徑，一併更新為標註後的版本
      // 這能解決使用者在多個地方（如訊號欄位與一般圖片欄位）貼上同一張圖，但只標註其中一個的情況
      if (originalPath) {
        // 更新 images 陣列
        if (payload.images) {
          payload.images.forEach(img => {
            if (img.image_path === originalPath) img.image_path = serverPath;
          });
        }
        // 更新 entry_signals (JSON string)
        if (payload.entry_signals && typeof payload.entry_signals === 'string') {
          const sigs = parseJSONSafe(payload.entry_signals, []);
          sigs.forEach(sig => {
            if (sig.image === originalPath) sig.image = serverPath;
            if (sig.images) {
              sig.images.forEach(img => {
                if (img.image === originalPath) img.image = serverPath;
              });
            }
          });
          payload.entry_signals = JSON.stringify(sigs);
        }
        // 更新 entry_pattern (JSON string)
        if (payload.entry_pattern && typeof payload.entry_pattern === 'string') {
          const pats = parseJSONSafe(payload.entry_pattern, []);
          pats.forEach(pat => {
            if (pat.image === originalPath) pat.image = serverPath;
            if (pat.images) {
              pat.images.forEach(img => {
                if (img.image === originalPath) img.image = serverPath;
              });
            }
          });
          payload.entry_pattern = JSON.stringify(pats);
        }
        // 更新單一欄位
        if (payload.entry_strategy_image === originalPath) payload.entry_strategy_image = serverPath;
        if (payload.legend_htf_image === originalPath) payload.legend_htf_image = serverPath;
        if (payload.legend_king_image === originalPath) payload.legend_king_image = serverPath;
        if (payload.legend_images && typeof payload.legend_images === 'string') {
          const lgs = parseJSONSafe(payload.legend_images, []);
          lgs.forEach(lg => {
            if (lg.image === originalPath) lg.image = serverPath;
          });
          payload.legend_images = JSON.stringify(lgs);
        }
        if (payload.expert_images && typeof payload.expert_images === 'string') {
          const exps = parseJSONSafe(payload.expert_images, []);
          exps.forEach(exp => {
            if (exp.image === originalPath) exp.image = serverPath;
          });
          payload.expert_images = JSON.stringify(exps);
        }
        if (payload.elite_images && typeof payload.elite_images === 'string') {
          const elts = parseJSONSafe(payload.elite_images, []);
          elts.forEach(elt => {
            if (elt.image === originalPath) elt.image = serverPath;
          });
          payload.elite_images = JSON.stringify(elts);
        }
      }
      
      await tradesAPI.update(tradeId, payload);
      
      selectedImage = imagesAPI.getUrl(serverPath);
      showAnnotator = false;
      loadTrades(); 
    } catch (e) {
      console.error('Failed to save annotated image', e);
      alert('儲存標註圖片失敗');
    }
  }

  function sanitizeTradePayload(fullTrade, newColor) {
    const payload = {
      ...fullTrade,
      color_tag: newColor !== undefined ? newColor || '' : fullTrade.color_tag || '',
      account_id: fullTrade.account_id,
      trade_type: fullTrade.trade_type || 'actual',
      symbol: fullTrade.symbol,
      side: fullTrade.side,
      entry_time: fullTrade.entry_time,
      entry_reason: fullTrade.entry_reason || '',
      exit_reason: fullTrade.exit_reason || '',
      entry_strategy: fullTrade.entry_strategy || '',
      entry_signals: fullTrade.entry_signals || '',
      entry_checklist: fullTrade.entry_checklist || '',
      entry_pattern: fullTrade.entry_pattern || '',
      trend_analysis: fullTrade.trend_analysis || '',
      entry_timeframe: fullTrade.entry_timeframe || '',
      trend_type: fullTrade.trend_type || '',
      market_session: fullTrade.market_session || '',
      legend_king_htf: fullTrade.legend_king_htf || '',
      legend_king_image: fullTrade.legend_king_image || '',
      legend_king_image_original: fullTrade.legend_king_image_original || '',
      legend_htf: fullTrade.legend_htf || '',
      legend_htf_image: fullTrade.legend_htf_image || '',
      legend_htf_image_original: fullTrade.legend_htf_image_original || '',
      legend_de_htf: fullTrade.legend_de_htf || '',
      legend_images: fullTrade.legend_images || '',
      expert_images: fullTrade.expert_images || '',
      elite_images: fullTrade.elite_images || '',
      entry_strategy_image: fullTrade.entry_strategy_image || '',
      entry_strategy_image_original: fullTrade.entry_strategy_image_original || '',
      notes: fullTrade.notes || '',
      journal: fullTrade.journal || '',
      timezone_offset:
        fullTrade.timezone_offset !== null && fullTrade.timezone_offset !== undefined
          ? fullTrade.timezone_offset
          : 0,
    };

    if (fullTrade.images) {
      payload.images = fullTrade.images.map(img => ({
        image_type: img.image_type,
        image_path: img.image_path,
      }));
    }

    return payload;
  }

  function closeImageModal() {
    selectedImage = null;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && selectedImage) {
      closeImageModal();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="card">
  {#if !isCompact}
    <div class="header">
      <h2>📋 交易歷史紀錄</h2>
      <button class="btn btn-primary" on:click={() => navigate(`/new?symbol=${$selectedSymbol}`)}
        >➕ 新增交易</button
      >
    </div>

    <!-- 篩選器 -->
    <div class="filters">
      <div class="filter-group">
        <label>品種</label>
        <select bind:value={filters.symbol} class="form-control">
          <option value="">全部品種</option>
          {#each SYMBOLS as sym}
            <option value={sym}>{sym}</option>
          {/each}
        </select>
      </div>

      <div class="filter-group">
        <label>方向</label>
        <select bind:value={filters.side} class="form-control">
          <option value="">全部方向</option>
          <option value="long">做多</option>
          <option value="short">做空</option>
        </select>
      </div>

      <div class="filter-group">
        <label>標籤</label>
        <select bind:value={filters.tag} class="form-control">
          <option value="">全部標籤</option>
          {#each allTags as tag}
            <option value={tag.name}>{tag.name}</option>
          {/each}
        </select>
      </div>

      <div class="filter-group">
        <label>開始日期</label>
        <input type="date" bind:value={filters.start_date} class="form-control" />
      </div>

      <div class="filter-group">
        <label>結束日期</label>
        <input type="date" bind:value={filters.end_date} class="form-control" />
      </div>

      <div class="filter-actions">
        <button class="btn btn-primary" on:click={applyFilters}>套用篩選</button>
        <button class="btn" on:click={clearFilters}>清除</button>
      </div>
    </div>
  {:else}
    <div class="header">
      <h3>📋 最新交易紀錄</h3>
    </div>
  {/if}

  <!-- 交易列表 -->
  {#if loading}
    <div class="loading-overlay">
      <div class="loader"></div>
      <p>正在載入交易紀錄...</p>
    </div>
  {:else if trades.length === 0}
    <div class="empty">
      <p>📭 尚無交易紀錄</p>
      <Link to="/new" class="btn btn-primary">開始記錄第一筆交易</Link>
    </div>
  {:else}
    <div class="trades-grid">
      {#each trades as trade (trade.id)}
        {@const matchedPlan = getMatchedPlan(trade)}
        <div
          class="trade-card {trade.trade_type === 'actual' && !trade.exit_time ? 'is-ongoing' : ''}"
          role="button"
          tabindex="0"
          on:click={() => navigate(`/edit/${trade.id}`)}
          on:keydown={e => (e.key === 'Enter' || e.key === ' ') && navigate(`/edit/${trade.id}`)}
        >
          <!-- 重新整理按鈕 (置於右上角) -->
          <button
            class="sync-btn"
            on:click|stopPropagation={() => syncTrade(trade.id)}
            title="重新從交易所同步此筆資料"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path
                d="M21 3v5h-5"
              /></svg
            >
          </button>

          <!-- 單一行：品種 + 方向 + 所有資訊 + 盈虧 -->
          <div class="trade-header-compact">
            <div class="compact-left">
              <h3>{trade.symbol}</h3>
              <span class="badge {trade.side === 'long' ? 'badge-danger' : 'badge-success'}">
                {trade.side === 'long' ? '📈 做多' : '📉 做空'}
              </span>
              {#if trade.entry_strategy}
                <span class="strategy-badge {trade.entry_strategy}">
                  {getStrategyLabel(trade.entry_strategy)}
                </span>
              {/if}
              {#if trade.trade_type === 'observation'}
                <span class="badge journal-badge">📓 記事</span>
              {/if}
              {#if trade.trade_type === 'actual'}
                <span class="compact-item">
                  <span class="compact-label">進場:</span>
                  <span class="compact-value">{trade.entry_price}</span>
                </span>
                {#if trade.exit_price}
                  <span class="compact-item">
                    <span class="compact-label">平倉:</span>
                    <span class="compact-value">{trade.exit_price}</span>
                  </span>
                {/if}
                {#if trade.initial_sl}
                  <span class="compact-item">
                    <span class="compact-label">停損:</span>
                    <span class="compact-value">{trade.initial_sl}</span>
                  </span>
                  {#if trade.bullet_size}
                    <span class="compact-item">
                      <span class="compact-label">子彈:</span>
                      <span class="compact-value">{trade.bullet_size.toFixed(1)}</span>
                    </span>
                  {/if}
                  {#if trade.rr_ratio}
                    <span class="compact-item">
                      <span class="compact-label">風報:</span>
                      <span class="compact-value">{trade.rr_ratio.toFixed(2)}</span>
                    </span>
                  {/if}
                {/if}
                <span class="compact-item">
                  <span class="compact-label">手數:</span>
                  <span class="compact-value">{trade.lot_size}</span>
                </span>
              {/if}
              <span class="compact-item">
                <span class="compact-label">時間:</span>
                <span class="compact-value">{formatDate(trade.entry_time)}</span>
              </span>
            </div>
            {#if trade.trade_type === 'actual'}
              <span
                class="pnl {trade.pnl >= 0 ? (trade.pnl === null ? '' : 'profit') : 'loss'}"
                style="margin-right: 1.5rem;"
              >
                {trade.pnl === null || trade.pnl === undefined
                  ? 'NA'
                  : (trade.pnl >= 0 ? '+' : '') + trade.pnl.toFixed(2)}
              </span>
            {/if}
          </div>

          <!-- 盤面規劃整合區 -->
          <div class="daily-plan-match-section">
            <span class="session-label-inline">
              時段：<strong>{getMarketSessionLabel(trade)}</strong>
            </span>
            {#if matchedPlan}
              <div
                class="matched-plan-info"
                role="button"
                tabindex="0"
                on:click|stopPropagation={() => navigate(`/plans/edit/${matchedPlan.id}`)}
                on:keydown|stopPropagation={e =>
                  (e.key === 'Enter' || e.key === ' ') && navigate(`/plans/edit/${matchedPlan.id}`)}
              >
                <span class="plan-badge">✅ 已有規劃</span>
                {#if matchedPlan.market_session === 'all'}
                  {@const trendData = JSON.parse(matchedPlan.trend_analysis || '{}')}
                  {@const sessionData = trendData[trade.market_session]}
                  {#if sessionData}
                    {@const sessionTrends = sessionData.trends || {}}
                    {@const longs = Object.entries(sessionTrends)
                      .filter(([_, t]) => t.direction === 'long')
                      .map(([tf, _]) => tf)}
                    {@const shorts = Object.entries(sessionTrends)
                      .filter(([_, t]) => t.direction === 'short')
                      .map(([tf, _]) => tf)}
                    <div class="plan-summary-group">
                      {#if longs.length > 0}
                        <span class="trend-item bullish">{longs.join(', ')}</span>
                      {/if}
                      {#if shorts.length > 0}
                        <span class="trend-item bearish">{shorts.join(', ')}</span>
                      {/if}
                    </div>
                  {/if}
                {:else if matchedPlan.notes && matchedPlan.notes !== 'Session-based unified plan'}
                  <p class="plan-summary-text">
                    {matchedPlan.notes.slice(0, 50)}{matchedPlan.notes.length > 50 ? '...' : ''}
                  </p>
                {/if}
              </div>
            {:else}
              <div class="no-plan-info">
                <span class="plan-badge missing">❌ 尚無規劃</span>
                <button
                  class="btn btn-sm btn-outline-primary"
                  on:click|stopPropagation={() => {
                    const date = new Date(trade.entry_time).toISOString().slice(0, 10);
                    navigate(
                      `/plans/new?date=${date}&session=${trade.market_session}&symbol=${trade.symbol}`
                    );
                  }}
                >
                  ➕ 新增盤勢
                </button>
              </div>
            {/if}
          </div>

          {#if trade.tags && trade.tags.length > 0}
            <div class="trade-tags">
              {#each trade.tags as tag}
                <span class="tag">#{tag.name}</span>
              {/each}
            </div>
          {/if}

          {#if trade.entry_reason || trade.exit_reason}
            <div class="trade-reasons">
              {#if trade.entry_reason}
                <div class="reason-item">
                  <span class="reason-label">📍 進場分析：</span>
                  <div
                    class="reason-content"
                    on:click={e => e.stopPropagation()}
                    role="presentation"
                  >
                    {@html trade.entry_reason}
                  </div>
                </div>
              {/if}
              {#if trade.exit_reason}
                <div class="reason-item">
                  <span class="reason-label">🎯 平倉理由：</span>
                  <div
                    class="reason-content"
                    on:click={e => e.stopPropagation()}
                    role="presentation"
                  >
                    {@html trade.exit_reason}
                  </div>
                </div>
              {/if}
            </div>
          {/if}

          {#if trade.notes}
            <div class="trade-notes">
              <span class="reason-label">📝 交易復盤：</span>
              <div class="notes-content" on:click={e => e.stopPropagation()}>
                {@html trade.notes}
              </div>
            </div>
          {/if}

          {#if trade.images && trade.images.length > 0}
            <div class="trade-images">
              {#each trade.images as image, index}
                <button
                  class="image-thumb"
                  on:click={e => {
                    e.stopPropagation();
                    openImageModal(
                      image.image_path,
                      `${image.image_type === 'entry' ? '進場' : image.image_type === 'exit' ? '平倉' : '圖片'}截圖`,
                      { tradeId: trade.id, type: 'general', index }
                    );
                  }}
                  title="點擊查看圖片"
                >
                  <img
                    src={imagesAPI.getUrl(image.image_path)}
                    alt={image.image_type}
                    on:error={e => {
                      console.error('圖片載入失敗:', image.image_path);
                      e.target.src =
                        'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgZmlsbD0iI2VlZSIvPjx0ZXh0IHg9IjUwJSIgeT0iNTAlIiBmb250LXNpemU9IjE0IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+5ZyW54mH6Yyy5aSx5pWXPC90ZXh0Pjwvc3ZnPg==';
                    }}
                  />
                  <span class="image-label">
                    {#if image.image_type === 'entry'}
                      📈 進場
                    {:else if image.image_type === 'exit'}
                      📉 平倉
                    {:else if image.image_type === 'trailing_stop'}
                      🎯 移動停利
                    {:else if image.image_type === 'observation'}
                      📓 圖面紀錄
                    {:else}
                      📷 圖片
                    {/if}
                  </span>
                </button>
              {/each}

              <!-- 顯示達人訊號圖 -->
              {#if trade.entry_signals}
                {#each parseJSONSafe(trade.entry_signals, []) as signal}
                  {#if signal.image}
                    <button
                      class="image-thumb"
                      on:click={e => {
                        e.stopPropagation();
                        openImageModal(signal.image, `訊號圖: ${signal.name}`, { tradeId: trade.id, type: 'signal_legacy', name: signal.name });
                      }}
                      title="點擊查看 {signal.name} 訊號圖"
                    >
                      <img
                        src={signal.image}
                        alt={signal.name}
                        on:error={e => {
                          console.error('訊號圖片載入失敗:', signal.name);
                          e.target.src =
                            'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgZmlsbD0iI2VlZSIvPjx0ZXh0IHg9IjUwJSIgeT0iNTAlIiBmb250LXNpemU9IjE0IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+5ZyW54mH6Yyy5aSx5pWXPC90ZXh0Pjwvc3ZnPg==';
                        }}
                      />
                      <span class="image-label">⚡ {signal.name}</span>
                    </button>
                  {/if}
                {/each}
              {/if}

              <!-- 顯示進場樣態圖 (JSON 或 Legacy Base64) -->
              {#if trade.entry_pattern}
                {@const parsedPatterns = parseJSONSafe(trade.entry_pattern, [])}
                {#if Array.isArray(parsedPatterns)}
                  {#each parsedPatterns as pattern}
                    {#if pattern.image}
                      <button
                        class="image-thumb"
                        on:click={e => {
                          e.stopPropagation();
                          openImageModal(pattern.image, `樣態圖: ${pattern.name}`, { tradeId: trade.id, type: 'pattern', name: pattern.name });
                        }}
                        title="點擊查看 {pattern.name} 樣態圖"
                      >
                        <img
                          src={pattern.image}
                          alt={pattern.name}
                          on:error={e => {
                            console.error('樣態圖片載入失敗:', pattern.name);
                            e.target.src =
                              'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgZmlsbD0iI2VlZSIvPjx0ZXh0IHg9IjUwJSIgeT0iNTAlIiBmb250LXNpemU9IjE0IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+5ZyW54mH6Yyy5aSx5pWXPC90ZXh0Pjwvc3ZnPg==';
                          }}
                        />
                        <span class="image-label">🧩 {pattern.name}</span>
                      </button>
                    {/if}
                  {/each}
                {:else if typeof trade.entry_pattern === 'string' && trade.entry_strategy_image}
                  <!-- Legacy support -->
                  <button
                    class="image-thumb"
                    on:click={e => {
                      e.stopPropagation();
                      openImageModal(trade.entry_strategy_image, `進場樣態: ${trade.entry_pattern}`, { tradeId: trade.id, type: 'strategy' });
                    }}
                    title="點擊查看進場樣態圖"
                  >
                    <img
                      src={trade.entry_strategy_image}
                      alt="進場樣態"
                      on:error={e => {
                        console.error('樣態圖片載入失敗');
                        e.target.src =
                          'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgZmlsbD0iI2VlZSIvPjx0ZXh0IHg9IjUwJSIgeT0iNTAlIiBmb250LXNpemU9IjE0IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+5ZyW54mH6Yyy5aSx5pWXPC90ZXh0Pjwvc3ZnPg==';
                      }}
                    />
                    <span class="image-label">🧩 {trade.entry_pattern}</span>
                  </button>
                {/if}
              {/if}

              <!-- 顯示訊號圖片(支援多圖) -->
              {#each parseJSONSafe(trade.entry_signals, []) as sig}
                {@const sigImages = sig.images || (sig.image ? [{image: sig.image}] : [])}
                {#each sigImages as img, idx}
                  {#if img.image}
                    <button
                      class="image-thumb"
                      on:click={e => {
                        e.stopPropagation();
                        openImageModal(img.image, `訊號圖: ${sig.name || sig} (${idx + 1})`, { tradeId: trade.id, type: 'signal', name: sig.name || sig, index: idx });
                      }}
                      title="點擊查看訊號圖片 ({sig.name || sig})"
                    >
                      <img src={imagesAPI.getUrl(img.image)} alt="訊號圖" />
                      <span class="image-label">⚡ {sig.name || sig} {#if sigImages.length > 1}({idx + 1}){/if}</span>
                    </button>
                  {/if}
                {/each}
              {/each}

              <!-- 顯示樣態圖片(支援多圖) -->
              {#each parseJSONSafe(trade.entry_pattern, []) as pat}
                {@const patImages = pat.images || (pat.image ? [{image: pat.image}] : [])}
                {#each patImages as img, idx}
                  {#if img.image}
                    <button
                      class="image-thumb"
                      on:click={e => {
                        e.stopPropagation();
                        openImageModal(img.image, `樣態圖: ${pat.name || pat} (${idx + 1})`, { tradeId: trade.id, type: 'pattern', name: pat.name || pat, index: idx });
                      }}
                      title="點擊查看進場樣態圖 ({pat.name || pat})"
                    >
                      <img src={imagesAPI.getUrl(img.image)} alt="樣態圖" />
                      <span class="image-label">🧩 {pat.name || pat} {#if patImages.length > 1}({idx + 1}){/if}</span>
                    </button>
                  {/if}
                {/each}
              {/each}

              <!-- 顯示傳奇觀察圖 -->
              {#if trade.legend_images}
                {#each parseJSONSafe(trade.legend_images, []) as img, idx}
                  {#if img.image}
                    <button
                      class="image-thumb"
                      on:click={e => {
                        e.stopPropagation();
                        openImageModal(img.image, `傳奇圖 (${idx + 1})`, { tradeId: trade.id, type: 'legend_images', index: idx });
                      }}
                      title="點擊查看傳奇觀察圖 {idx + 1}"
                    >
                      <img src={imagesAPI.getUrl(img.image)} alt="傳奇觀察圖" />
                      <span class="image-label">👑 傳奇圖 {idx + 1}</span>
                    </button>
                  {/if}
                {/each}
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- 分頁 -->
    <div class="pagination">
      <button
        class="btn"
        disabled={pagination.page === 1}
        on:click={() => changePage(pagination.page - 1)}
      >
        上一頁
      </button>
      <span
        >第 {pagination.page} 頁，共 {Math.ceil(pagination.total / pagination.page_size)} 頁</span
      >
      <button
        class="btn"
        disabled={pagination.page >= Math.ceil(pagination.total / pagination.page_size)}
        on:click={() => changePage(pagination.page + 1)}
      >
        下一頁
      </button>
    </div>
  {/if}
</div>

<!-- 圖片模態框 -->
{#if selectedImage}
  <div class="image-modal active" on:click={closeImageModal} transition:fade={{ duration: 200 }} role="presentation">
    <div class="image-modal-content" on:click={e => e.stopPropagation()} role="presentation">
      <div class="image-modal-header">
        <h3 class="image-modal-title">{modalTitle}</h3>
        <div class="image-modal-actions">
          {#if enlargedImageContext}
            <button
              class="annotator-toggle-btn"
              class:active={showAnnotator}
              on:click={toggleAnnotator}
              title="標註工具"
            >
              {showAnnotator ? '👁️ 查看' : '✏️ 標註'}
            </button>
          {/if}
          <button class="image-modal-close" on:click={closeImageModal}>&times;</button>
        </div>
      </div>
      <div class="image-modal-body" class:annotator-mode={showAnnotator}>
        {#if showAnnotator}
          <ImageAnnotator
            imageSrc={selectedImage}
            originalImageSrc={imagesAPI.getUrl(enlargedOriginalImage)}
            onSave={handleAnnotatedImage}
          />
        {:else}
          <img src={selectedImage} alt={modalTitle} class="image-modal-img" />
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  h2 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-main);
    letter-spacing: -0.025em;
  }

  .filters {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f1f5f9;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-color);
  }

  .filter-group {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .filter-group label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .filter-actions {
    display: flex;
    gap: 0.5rem;
    align-items: flex-end;
  }

  .empty {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
    background: var(--card-bg);
    border-radius: var(--radius-lg);
    border: 2px dashed var(--border-color);
  }

  .empty p {
    font-size: 1.125rem;
    margin-bottom: 1.5rem;
  }

  .trades-grid {
    display: grid;
    gap: 1rem;
  }

  .trade-card {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    padding: 1.5rem 1.25rem 1.25rem 1.25rem;
    transition: all 0.2s ease;
    cursor: pointer;
    position: relative;
    box-shadow: var(--shadow-sm);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Ongoing Trade Animation */
  @keyframes float {
    0% {
      transform: translateY(0px);
    }
    50% {
      transform: translateY(-6px);
    }
    100% {
      transform: translateY(0px);
    }
  }

  @keyframes pulse-bg {
    0% {
      opacity: 0.6;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: 0.6;
    }
  }

  .trade-card.is-ongoing {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.08) 0%,
      rgba(168, 85, 247, 0.08) 100%
    ) !important;
    border-color: rgba(99, 102, 241, 0.4) !important;
    animation: float 4s ease-in-out infinite;
    box-shadow: 0 15px 35px rgba(99, 102, 241, 0.12);
    z-index: 5;
  }

  .trade-card.is-ongoing::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: radial-gradient(circle at top right, rgba(99, 102, 241, 0.1), transparent 70%);
    pointer-events: none;
    animation: pulse-bg 3s ease-in-out infinite;
  }

  :global(body.dark-mode) .trade-card.is-ongoing {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.15) 0%,
      rgba(168, 85, 247, 0.15) 100%
    ) !important;
    border-color: rgba(99, 102, 241, 0.5) !important;
    box-shadow: 0 15px 40px rgba(0, 0, 0, 0.4);
  }

  .trade-card:hover {
    border-color: var(--primary);
    box-shadow: var(--shadow-md);
    transform: translateY(-2px);
  }

  /* 右上角刪除按鈕 */
  .delete-btn {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    border-radius: 6px;
    font-size: 1.25rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    opacity: 0;
  }

  .trade-card:hover .delete-btn {
    opacity: 1;
  }

  /* 重新整理按鈕 */
  .sync-btn {
    position: absolute;
    top: 0.3rem;
    right: 0.3rem;
    width: 26px;
    height: 26px;
    border: none;
    background: transparent;
    color: #94a3b8;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    opacity: 0;
    z-index: 20;
  }

  .trade-card:hover .sync-btn {
    opacity: 0.8;
  }

  .sync-btn:hover {
    color: var(--primary);
    opacity: 1 !important;
    transform: rotate(30deg);
  }

  .trade-header-compact {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }

  .compact-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .trade-header-compact h3 {
    margin: 0;
    color: var(--text-main);
    font-size: 1.125rem;
    font-weight: 700;
  }

  .compact-item {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    color: var(--text-muted);
    font-size: 0.8125rem;
  }

  .compact-value {
    color: var(--text-main);
    font-weight: 600;
  }

  .pnl {
    font-size: 1.25rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }

  .pnl.profit {
    color: #3b82f6;
  }

  .pnl.loss {
    color: #ef4444;
  }

  .trade-tags {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.375rem;
    margin-top: 0.75rem;
  }

  .tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #f1f5f9;
    color: var(--text-muted);
    padding: 0.125rem 0.5rem;
    border-radius: 6px;
    font-size: 0.75rem;
    font-weight: 500;
    border: 1px solid var(--border-color);
    white-space: nowrap;
    line-height: 1;
  }

  .trade-reasons,
  .trade-notes {
    margin-top: 1rem;
    padding: 1rem;
    background: #fffdf5;
    border: 1px solid #fef3c7;
    border-radius: 8px;
  }

  /* 盤面規劃整合樣式 */
  .daily-plan-match-section {
    margin: 0.75rem 0;
    padding: 0.75rem 1rem;
    background: #f8fafc;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .session-label-inline {
    font-size: 0.9rem;
    color: #64748b;
  }

  .session-label-inline strong {
    color: #334155;
  }

  .matched-plan-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    cursor: pointer;
    transition: all 0.2s;
    padding: 2px 8px;
    border-radius: 6px;
  }

  .matched-plan-info:hover {
    background: #f1f5f9;
  }

  .no-plan-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .plan-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    background: #dcfce7;
    color: #166534;
    white-space: nowrap;
    line-height: 1;
  }

  .plan-badge.missing {
    background: #fee2e2;
    color: #991b1b;
  }

  .strategy-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 6px;
    white-space: nowrap;
    line-height: 1;
  }

  .strategy-badge.expert {
    background: #059669;
    color: white;
    border: none;
  }

  .strategy-badge.elite {
    background: #1e3a8a;
    color: white;
    border: none;
  }

  .strategy-badge.legend {
    background: #78350f;
    color: white;
    border: none;
  }

  .journal-badge {
    background: #f3f4f6 !important;
    color: #374151 !important;
    border: 1px solid #d1d5db !important;
  }
  :global(body.dark-mode) .journal-badge {
    background: #374151 !important;
    color: #f3f4f6 !important;
    border-color: #4b5563 !important;
  }

  .plan-summary-group {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .trend-item {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    white-space: nowrap;
    line-height: 1;
  }

  .trend-item.bullish {
    background: #fee2e2;
    color: #991b1b;
  }

  .trend-item.bearish {
    background: #dcfce7;
    color: #166534;
  }

  .btn-sm {
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
  }

  .btn-outline-primary {
    border: 1px solid #6366f1;
    color: #6366f1;
    background: white;
  }

  .btn-outline-primary:hover {
    background: #f5f3ff;
  }

  .trade-notes {
    background: #f0f9ff;
    border: 1px solid #bae6fd;
  }

  .reason-label {
    color: #0369a1;
    font-weight: 700;
    display: block;
    margin-bottom: 0.25rem;
  }

  .reason-content,
  .notes-content {
    color: #1e293b;
    line-height: 1.6;
    white-space: pre-wrap;
  }

  .trade-images {
    display: flex;
    gap: 0.75rem;
    margin-top: 1rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }

  .image-thumb {
    flex: 0 0 auto;
    width: 120px;
    height: 80px;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--border-color);
    transition: transform 0.2s;
  }

  .image-thumb:hover {
    transform: scale(1.05);
  }

  .image-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 1.5rem;
    margin-top: 3rem;
    color: var(--text-muted);
    font-size: 0.875rem;
  }

  .pagination span {
    color: #4a5568;
  }

  /* Mobile Responsive Optimizations */
  @media (max-width: 768px) {
    .trade-header-compact {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.5rem;
    }
    .compact-left {
      gap: 0.5rem;
    }
    .daily-plan-match-section {
      padding: 0.5rem;
      gap: 0.5rem;
    }
    .matched-plan-info {
      padding: 0;
    }
    .image-thumb {
      width: 100px;
      height: 70px;
    }
  }

  @media (max-width: 480px) {
    .trade-card {
      padding: 1rem;
    }
    .pnl {
      font-size: 1.1rem;
    }
    .pagination {
      gap: 0.75rem;
      flex-wrap: wrap;
    }
  }

  /* 圖片放大查看模態視窗 (同步至 TradeForm 樣式) */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    padding: 2rem;
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .image-modal-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
    background: white;
    border-radius: 12px;
    padding: 0;
    display: flex;
    flex-direction: column;
    animation: slideScaleIn 0.3s ease-out;
    overflow: hidden;
  }

  @keyframes slideScaleIn {
    from {
      transform: scale(0.9);
      opacity: 0;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }

  .image-modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid #e2e8f0;
    background: #f8fafc;
  }

  .image-modal-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .annotator-toggle-btn {
    padding: 0.5rem 1rem;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    font-size: 0.9rem;
    font-weight: 600;
    color: #475569;
    cursor: pointer;
    transition: all 0.2s;
  }

  .annotator-toggle-btn:hover {
    background: #e2e8f0;
  }

  .annotator-toggle-btn.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  .image-modal-title {
    font-size: 1.1rem;
    font-weight: 700;
    color: #1e293b;
    margin: 0;
  }

  .image-modal-close {
    width: 32px;
    height: 32px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.25rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .image-modal-close:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  .image-modal-body {
    flex: 1;
    overflow: auto;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #0f172a;
    padding: 1rem;
  }

  .image-modal-body.annotator-mode {
    align-items: flex-start;
  }

  .image-modal-img {
    max-width: 100%;
    max-height: calc(95vh - 4rem);
    object-fit: contain;
    border-radius: 4px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.3);
  }
</style>
