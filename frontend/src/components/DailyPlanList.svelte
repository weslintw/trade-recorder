<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { dailyPlansAPI } from '../lib/api';
  import { SYMBOLS, MARKET_SESSIONS, TIMEFRAMES } from '../lib/constants';
  import { selectedSymbol, selectedAccountId } from '../lib/stores';

  export let isCompact = false;

  let plans = [];
  let filteredPlans = [];
  let filters = {
    startDate: '',
    endDate: '',
    symbol: '',
    marketSession: '',
  };

  onMount(() => {
    // 初始載入由下面的 $: 響應式語句處理
  });

  // 當全局品種改變時，更新篩選器並重新載入
  $: if ($selectedSymbol && $selectedAccountId) {
    filters.symbol = $selectedSymbol;
    loadPlans();
  }

  async function loadPlans() {
    try {
      const params = {
        account_id: $selectedAccountId,
      };
      if (filters.startDate) params.start_date = filters.startDate;
      if (filters.endDate) params.end_date = filters.endDate;
      if (filters.symbol) params.symbol = filters.symbol;
      if (filters.marketSession) params.market_session = filters.marketSession;

      const response = await dailyPlansAPI.getAll(params);
      plans = response.data.data || [];
      filteredPlans = plans;
    } catch (error) {
      console.error('載入規劃失敗:', error);
      alert('載入規劃失敗');
    }
  }

  function applyFilters() {
    loadPlans();
  }

  function clearFilters() {
    filters = {
      startDate: '',
      endDate: '',
      symbol: '',
      marketSession: '',
    };
    loadPlans();
  }

  async function deletePlan(id) {
    if (!confirm('確定要刪除此規劃嗎？')) return;

    try {
      await dailyPlansAPI.delete(id);
      loadPlans();
    } catch (error) {
      console.error('刪除失敗:', error);
      alert('刪除規劃失敗');
    }
  }

  function getMarketSessionLabel(session) {
    const s = MARKET_SESSIONS.find(s => s.value === session);
    return s ? s.label : '';
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  }
</script>

<div class="card">
  {#if !isCompact}
    <div class="header">
      <h2>📅 每日盤面規劃</h2>
      <button
        class="btn btn-primary"
        on:click={() => navigate(`/plans/new?symbol=${$selectedSymbol}`)}
      >
        ➕ 新增規劃
      </button>
    </div>

    <!-- 篩選器 -->
    <div class="filters">
      <div class="filter-group">
        <label>開始日期</label>
        <input type="date" class="form-control" bind:value={filters.startDate} />
      </div>

      <div class="filter-group">
        <label>結束日期</label>
        <input type="date" class="form-control" bind:value={filters.endDate} />
      </div>

      <div class="filter-group">
        <label>交易品種</label>
        <select class="form-control" bind:value={filters.symbol}>
          <option value="">全部</option>
          {#each SYMBOLS as sym}
            <option value={sym}>{sym}</option>
          {/each}
        </select>
      </div>

      <div class="filter-group">
        <label>市場時段</label>
        <select class="form-control" bind:value={filters.marketSession}>
          <option value="">全部時段</option>
          {#each MARKET_SESSIONS as session}
            <option value={session.value}>{session.label}</option>
          {/each}
        </select>
      </div>

      <div class="filter-actions">
        <button class="btn btn-primary" on:click={applyFilters}> 套用篩選 </button>
        <button class="btn btn-secondary" on:click={clearFilters}> 清除 </button>
      </div>
    </div>
  {:else}
    <div class="header">
      <h3>📅 最新盤面規劃</h3>
    </div>
  {/if}

  <!-- 規劃列表 -->
  {#if filteredPlans.length === 0}
    <div class="empty-state">
      <p>📋 尚無規劃記錄</p>
      <button class="btn btn-primary" on:click={() => navigate('/plans/new')}>
        建立第一筆規劃
      </button>
    </div>
  {:else}
    <div class="plans-list">
      {#each filteredPlans as plan}
        {@const trendData = JSON.parse(plan.trend_analysis || '{}')}
        {@const isUnified = plan.market_session === 'all'}
        <div class="plan-card" on:click={() => navigate(`/plans/edit/${plan.id}`)}>
          <div class="plan-header">
            <div class="plan-info">
              <h3>{formatDate(plan.plan_date)}</h3>
              <span class="symbol-badge">{plan.symbol || SYMBOLS[0]}</span>
            </div>
            <div class="plan-actions">
              <button
                class="action-btn delete"
                on:click|stopPropagation={() => deletePlan(plan.id)}
              >
                🗑️
              </button>
            </div>
          </div>

          {#if isUnified}
            <div class="timeframe-progression">
              {#each TIMEFRAMES as tf}
                {@const asianTrend = trendData.asian?.trends?.[tf]}
                {@const europeanTrend = trendData.european?.trends?.[tf]}
                {@const usTrend = trendData.us?.trends?.[tf]}

                {#if asianTrend?.direction || europeanTrend?.direction || usTrend?.direction}
                  <div class="progression-row">
                    <span class="tf-name">{tf}:</span>
                    <div class="steps">
                      {#if asianTrend?.direction}
                        <span
                          class="step"
                          class:long={asianTrend.direction === 'long'}
                          class:short={asianTrend.direction === 'short'}
                        >
                          亞盤 {asianTrend.direction === 'long' ? '多' : '空'}
                        </span>
                      {/if}

                      {#if europeanTrend?.direction}
                        {#if asianTrend?.direction}<span class="arrow">=></span>{/if}
                        <span
                          class="step"
                          class:long={europeanTrend.direction === 'long'}
                          class:short={europeanTrend.direction === 'short'}
                        >
                          歐盤 {europeanTrend.direction === 'long' ? '多' : '空'}
                        </span>
                      {/if}

                      {#if usTrend?.direction}
                        {#if asianTrend?.direction || europeanTrend?.direction}<span class="arrow"
                            >=></span
                          >{/if}
                        <span
                          class="step"
                          class:long={usTrend.direction === 'long'}
                          class:short={usTrend.direction === 'short'}
                        >
                          美盤 {usTrend.direction === 'long' ? '多' : '空'}
                        </span>
                      {/if}
                    </div>
                  </div>
                {/if}
              {/each}
            </div>

            {#if trendData.asian?.notes || trendData.european?.notes || trendData.us?.notes}
              <div class="unified-notes">
                {#each ['asian', 'european', 'us'] as session}
                  {#if trendData[session]?.notes}
                    <div class="note-item">
                      <span class="note-session {session}"
                        >{getMarketSessionLabel(session)}備註：</span
                      >
                      <span class="note-text">{trendData[session].notes}</span>
                    </div>
                  {/if}
                {/each}
              </div>
            {/if}
          {:else}
            {#if plan.notes}
              <div class="plan-notes">
                {plan.notes}
              </div>
            {/if}

            <div class="plan-trends">
              <h4>📊 趨勢分析</h4>
              <div class="trends-grid">
                {#each Object.entries(trendData) as [timeframe, trend]}
                  {#if trend.direction || trend.image || (trend.signals && trend.signals.length > 0) || (trend.wave_numbers && trend.wave_numbers.length > 0)}
                    <div class="trend-summary">
                      <div class="trend-summary-header">
                        <span class="timeframe">{timeframe}</span>
                        {#if trend.direction}
                          <span
                            class="direction"
                            class:long={trend.direction === 'long'}
                            class:short={trend.direction === 'short'}
                          >
                            {trend.direction === 'long' ? '多' : '空'}
                          </span>
                        {/if}
                      </div>
                      {#if trend.signals && trend.signals.length > 0}
                        <div class="trend-signals">
                          {#each trend.signals.slice(0, 2) as signal}
                            <span class="signal-tag">{signal}</span>
                          {/each}
                        </div>
                      {/if}
                    </div>
                  {/if}
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  h2 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-main);
    letter-spacing: -0.025em;
  }

  h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 700;
    color: var(--text-main);
  }

  h4 {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.75rem;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  /* 篩選器 */
  .filters {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f1f5f9;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-color);
    flex-wrap: wrap;
  }

  .filter-group {
    flex: 1;
    min-width: 180px;
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

  /* 空狀態 */
  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
    background: var(--card-bg);
    border-radius: var(--radius-lg);
    border: 2px dashed var(--border-color);
  }

  .empty-state p {
    font-size: 1.125rem;
    margin-bottom: 1.5rem;
  }

  /* 規劃列表 */
  .plans-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .plan-card {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    padding: 1.25rem;
    transition: all 0.2s ease;
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    position: relative;
  }

  .plan-card:hover {
    border-color: var(--primary);
    box-shadow: var(--shadow-md);
  }

  .plan-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .plan-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .symbol-badge {
    background: #edf2f7;
    color: #4a5568;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
  }

  .plan-info h3 {
    margin: 0;
    font-size: 1.1rem;
    color: var(--text-main);
  }

  .filter-actions {
    display: flex;
    align-items: flex-end;
    gap: 0.5rem;
  }

  .plan-actions {
    display: flex;
    gap: 0.375rem;
  }

  .action-btn {
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1rem;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    background: transparent;
    color: var(--text-muted);
  }

  .action-btn:hover {
    background: #f1f5f9;
    color: var(--primary);
  }

  .session-badges {
    display: flex;
    gap: 0.25rem;
  }

  .badge-session {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-weight: 600;
  }

  .badge-session.asian {
    background: #ebf4ff;
    color: #2b6cb0;
  }
  .badge-session.european {
    background: #fefcbf;
    color: #975a16;
  }
  .badge-session.us {
    background: #fed7d7;
    color: #c53030;
  }

  .timeframe-progression {
    margin-top: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 1rem;
    background: #f8fafc;
    border-radius: 12px;
    border: 1px solid #e2e8f0;
  }

  .progression-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.9rem;
  }

  .tf-name {
    font-weight: 700;
    color: #475569;
    min-width: 40px;
  }

  .steps {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: nowrap;
  }

  .step {
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.85rem;
  }

  .step.long {
    background: #fef2f2;
    color: #991b1b;
  }

  .step.short {
    background: #f0fdf4;
    color: #166534;
  }

  .arrow {
    color: #94a3b8;
    font-weight: bold;
    font-size: 0.8rem;
  }

  .unified-notes {
    margin-top: 0.75rem;
    padding: 0.25rem 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .note-item {
    display: flex;
    gap: 0.5rem;
    font-size: 0.85rem;
    line-height: 1.4;
  }

  .note-session {
    font-weight: 700;
    white-space: nowrap;
    font-size: 0.8rem;
  }

  .note-session.asian {
    color: #2b6cb0;
  }
  .note-session.european {
    color: #975a16;
  }
  .note-session.us {
    color: #c53030;
  }

  .note-text {
    color: #4a5568;
    font-style: italic;
    white-space: pre-wrap;
  }

  .plan-notes {
    font-size: 0.9rem;
    color: #4a5568;
    margin-bottom: 1rem;
    padding: 0.75rem;
    background: #f8fafc;
    border-radius: 8px;
    white-space: pre-wrap;
  }

  .unified-trend-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding: 0.5rem 0.75rem;
    background: #f1f5f9;
    border-radius: 8px;
    margin-bottom: 0.5rem;
  }

  .summary-item {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.8rem;
    font-weight: 500;
  }

  .summary-item .session-label {
    font-weight: 700;
    font-size: 0.75rem;
    text-transform: uppercase;
  }

  .summary-item.clickable {
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    transition: background 0.2s;
  }

  .summary-item.clickable:hover {
    background: #e2e8f0;
  }

  .summary-item.asian {
    color: #2b6cb0;
  }
  .summary-item.european {
    color: #975a16;
  }
  .summary-item.us {
    color: #c53030;
  }

  .summary-item .trends-list {
    color: var(--text-main);
  }

  .outlook-group {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .trend-outlook {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
  }

  .trend-outlook.long,
  .direction.long {
    background: #fef2f2;
    color: #991b1b;
  }

  .trend-outlook.short,
  .direction.short {
    background: #f0fdf4;
    color: #166534;
  }

  .direction {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 1px 6px;
    border-radius: 4px;
    display: inline-block;
  }
</style>
