<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { dailyPlansAPI } from '../lib/api';

  let plans = [];
  let filteredPlans = [];
  let filters = {
    startDate: '',
    endDate: '',
    marketSession: ''
  };

  onMount(() => {
    loadPlans();
  });

  async function loadPlans() {
    try {
      const params = {};
      if (filters.startDate) params.start_date = filters.startDate;
      if (filters.endDate) params.end_date = filters.endDate;
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
      marketSession: ''
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
    const labels = {
      asian: '亞盤',
      european: '歐盤',
      us: '美盤'
    };
    return labels[session] || '';
  }

  function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('zh-TW', { 
      year: 'numeric', 
      month: '2-digit', 
      day: '2-digit' 
    });
  }
</script>

<div class="card">
  <div class="header">
    <h2>📅 每日盤面規劃</h2>
    <button class="btn btn-primary" on:click={() => navigate('/plans/new')}>
      ➕ 新增規劃
    </button>
  </div>

  <!-- 篩選器 -->
  <div class="filters">
    <div class="filter-group">
      <label>開始日期</label>
      <input 
        type="date" 
        class="form-control"
        bind:value={filters.startDate}
      />
    </div>

    <div class="filter-group">
      <label>結束日期</label>
      <input 
        type="date" 
        class="form-control"
        bind:value={filters.endDate}
      />
    </div>

    <div class="filter-group">
      <label>市場時段</label>
      <select class="form-control" bind:value={filters.marketSession}>
        <option value="">全部時段</option>
        <option value="asian">亞盤</option>
        <option value="european">歐盤</option>
        <option value="us">美盤</option>
      </select>
    </div>

    <div class="filter-actions">
      <button class="btn btn-primary" on:click={applyFilters}>
        套用篩選
      </button>
      <button class="btn btn-secondary" on:click={clearFilters}>
        清除
      </button>
    </div>
  </div>

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
        <div class="plan-card">
          <div class="plan-header">
            <div class="plan-info">
              <h3>{formatDate(plan.plan_date)}</h3>
              {#if plan.market_session}
                <span class="badge badge-info">{getMarketSessionLabel(plan.market_session)}</span>
              {/if}
            </div>
            <div class="plan-actions">
              <button class="action-btn edit" on:click={() => navigate(`/plans/edit/${plan.id}`)}>
                ✏️
              </button>
              <button class="action-btn delete" on:click={() => deletePlan(plan.id)}>
                🗑️
              </button>
            </div>
          </div>

          {#if plan.notes}
            <div class="plan-notes">
              {plan.notes}
            </div>
          {/if}

          <div class="plan-trends">
            <h4>📊 趨勢分析</h4>
            <div class="trends-grid">
              {#each Object.entries(JSON.parse(plan.trend_analysis || '{}')) as [timeframe, trend]}
                {#if trend.direction || trend.image || (trend.signals && trend.signals.length > 0) || (trend.wave_numbers && trend.wave_numbers.length > 0)}
                  <div class="trend-summary">
                    <div class="trend-summary-header">
                      <span class="timeframe">{timeframe}</span>
                      {#if trend.direction}
                        <span class="direction" class:long={trend.direction === 'long'} class:short={trend.direction === 'short'}>
                          {trend.direction === 'long' ? '多' : '空'}
                        </span>
                      {/if}
                    </div>
                    {#if trend.signals && trend.signals.length > 0}
                      <div class="trend-signals">
                        {#each trend.signals as signal}
                          <span class="signal-tag">{signal}</span>
                        {/each}
                      </div>
                    {/if}
                    {#if trend.wave_numbers && trend.wave_numbers.length > 0}
                      <div class="trend-wave">
                        <span class="wave-label">浪數:</span>
                        {#each trend.wave_numbers as num}
                          <span class="wave-value" class:highlight={trend.wave_highlight === num}>{num}</span>
                        {/each}
                      </div>
                    {/if}
                    {#if trend.image}
                      <span class="has-image">📸</span>
                    {/if}
                  </div>
                {/if}
              {/each}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  h2 {
    margin: 0;
    color: #2d3748;
  }

  h3 {
    margin: 0;
    font-size: 1.1rem;
    color: #2d3748;
  }

  h4 {
    font-size: 0.95rem;
    color: #4a5568;
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
    background: #f7fafc;
    border-radius: 12px;
    flex-wrap: wrap;
  }

  .filter-group {
    flex: 1;
    min-width: 200px;
  }

  .filter-group label {
    display: block;
    font-size: 0.9rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.5rem;
  }

  .filter-actions {
    display: flex;
    gap: 0.5rem;
    align-items: flex-end;
  }

  .btn-secondary {
    background: #e2e8f0;
    color: #2d3748;
  }

  .btn-secondary:hover {
    background: #cbd5e0;
  }

  /* 空狀態 */
  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    color: #a0aec0;
  }

  .empty-state p {
    font-size: 1.2rem;
    margin-bottom: 1.5rem;
  }

  /* 規劃列表 */
  .plans-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .plan-card {
    padding: 1.5rem;
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    background: white;
    transition: all 0.2s ease;
  }

  .plan-card:hover {
    border-color: #cbd5e0;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }

  .plan-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .plan-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .plan-actions {
    display: flex;
    gap: 0.5rem;
  }

  .action-btn {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-size: 1.2rem;
    transition: all 0.2s ease;
  }

  .action-btn.edit {
    background: #bee3f8;
  }

  .action-btn.edit:hover {
    background: #90cdf4;
  }

  .action-btn.delete {
    background: #fed7d7;
  }

  .action-btn.delete:hover {
    background: #fc8181;
  }

  .plan-notes {
    margin-bottom: 1rem;
    padding: 1rem;
    background: #f7fafc;
    border-radius: 8px;
    color: #4a5568;
    line-height: 1.6;
  }

  .plan-trends {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #e2e8f0;
  }

  .trends-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .trend-summary {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.75rem;
    background: white;
    border: 1.5px solid #e2e8f0;
    border-radius: 8px;
    font-size: 0.85rem;
  }

  .trend-summary-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .timeframe {
    font-weight: 600;
    color: #2d3748;
  }

  .direction {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-weight: 500;
    font-size: 0.75rem;
  }

  .direction.long {
    background: #c6f6d5;
    color: #22543d;
  }

  .direction.short {
    background: #fed7d7;
    color: #742a2a;
  }

  .trend-signals {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }

  .signal-tag {
    padding: 0.2rem 0.4rem;
    background: #edf2f7;
    border-radius: 4px;
    font-size: 0.7rem;
    color: #4a5568;
    font-weight: 500;
  }

  .trend-wave {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    flex-wrap: wrap;
  }

  .wave-label {
    font-size: 0.75rem;
    color: #718096;
  }

  .wave-value {
    padding: 0.2rem 0.5rem;
    background: #c6f6d5;
    border-radius: 4px;
    font-weight: 600;
    color: #22543d;
    font-size: 0.75rem;
  }

  .wave-value.highlight {
    background: #fed7d7;
    color: #742a2a;
  }

  .has-image {
    opacity: 0.6;
    font-size: 0.9rem;
  }
</style>

