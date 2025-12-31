<script>
  import { onMount } from 'svelte';
  import { Link, navigate } from 'svelte-routing';
  import { tradesAPI, tagsAPI, imagesAPI, dailyPlansAPI } from '../lib/api';
  import { SYMBOLS, MARKET_SESSIONS } from '../lib/constants';
  import { determineMarketSession, getStrategyLabel } from '../lib/utils';
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

  // 篩選條件
  let filters = {
    symbol: '',
    side: '',
    tag: '',
    start_date: '',
    end_date: '',
  };

  let allTags = [];
  let selectedImage = null;

  onMount(() => {
    loadTags();
  });

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

  function parseJSONSafe(str, defaultValue) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  function openImageModal(imagePath) {
    if (!imagePath) return;
    if (imagePath.startsWith('data:image/') || imagePath.startsWith('blob:')) {
      selectedImage = imagePath;
    } else {
      selectedImage = imagesAPI.getUrl(imagePath);
    }
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
    <div class="loading">載入中...</div>
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
          class="trade-card"
          role="button"
          tabindex="0"
          on:click={() => navigate(`/edit/${trade.id}`)}
          on:keydown={e => (e.key === 'Enter' || e.key === ' ') && navigate(`/edit/${trade.id}`)}
        >
          <!-- 刪除按鈕（右上角叉叉）-->
          <button
            class="delete-btn"
            on:click={e => {
              e.stopPropagation();
              deleteTrade(trade.id);
            }}
            title="刪除交易"
          >
            ×
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
              <span class="compact-item">
                <span class="compact-label">時間:</span>
                <span class="compact-value">{formatDate(trade.entry_time)}</span>
              </span>
            </div>
            {#if trade.pnl !== null}
              <span class="pnl {trade.pnl >= 0 ? 'profit' : 'loss'}" style="margin-right: 1.5rem;">
                {trade.pnl >= 0 ? '+' : ''}{trade.pnl.toFixed(2)}
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
              {#each trade.images as image}
                <button
                  class="image-thumb"
                  on:click={e => {
                    e.stopPropagation();
                    openImageModal(image.image_path);
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
                      👁️ 觀察
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
                        openImageModal(signal.image);
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
                          openImageModal(pattern.image);
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
                      openImageModal(trade.entry_strategy_image);
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
  <div class="modal" on:click={closeImageModal}>
    <div class="modal-content" on:click|stopPropagation>
      <button class="modal-close" on:click={closeImageModal}>×</button>
      <img src={selectedImage} alt="交易圖片" />
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

  .loading,
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

  .delete-btn:hover {
    background: #fee2e2;
    color: #ef4444;
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
    color: #dc2626;
  }

  .trade-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem;
    margin-top: 0.75rem;
  }

  .tag {
    background: #f1f5f9;
    color: var(--text-muted);
    padding: 0.125rem 0.5rem;
    border-radius: 6px;
    font-size: 0.75rem;
    font-weight: 500;
    border: 1px solid var(--border-color);
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
    font-size: 0.75rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    background: #dcfce7;
    color: #166534;
    white-space: nowrap;
  }

  .plan-badge.missing {
    background: #fee2e2;
    color: #991b1b;
  }

  .strategy-badge {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 6px;
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

  .plan-summary-group {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .trend-item {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    white-space: nowrap;
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

  .modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.9);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    cursor: pointer;
  }

  .modal-content {
    position: relative;
    max-width: 90%;
    max-height: 90%;
    cursor: default;
  }

  .modal-content img {
    max-width: 100%;
    max-height: 90vh;
    border-radius: 8px;
  }

  .modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 3rem;
    cursor: pointer;
    line-height: 1;
  }
</style>
