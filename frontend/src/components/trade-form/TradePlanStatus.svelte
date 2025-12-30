<script>
  import { navigate } from 'svelte-routing';

  export let matchedPlan = null;
  export let formData = {};

  function handleCreatePlan() {
    const date = new Date(formData.entry_time).toISOString().slice(0, 10);
    navigate(
      `/plans/new?date=${date}&session=${formData.market_session}&symbol=${formData.symbol}`
    );
  }
</script>

<div class="trade-plan-status-section">
  <div class="section-label-group">
    <label class="strategy-label">🗺️ 盤面規劃</label>
    {#if matchedPlan}
      <button
        type="button"
        class="plan-status-badge linked"
        on:click={() => navigate(`/plans/edit/${matchedPlan.id}`)}
      >
        ✅ 已有規劃 <span class="view-link">查看 ↗</span>
      </button>
    {:else}
      <button
        type="button"
        class="plan-status-badge missing"
        on:click={handleCreatePlan}
      >
        ❓ 尚無規劃 <span class="add-link">建立 ➕</span>
      </button>
    {/if}
  </div>

  {#if matchedPlan}
    <div class="plan-details-summary">
      {#if matchedPlan.notes && matchedPlan.notes !== 'Session-based unified plan'}
        <div class="plan-general-notes">{matchedPlan.notes}</div>
      {/if}

      {#if matchedPlan.market_session === 'all'}
        {@const trendData = JSON.parse(matchedPlan.trend_analysis || '{}')}
        <div class="progression-view">
          {#each ['M5', 'M15', 'M30', 'H1', 'H4', 'D1'] as tf}
            {@const asianTrend = trendData.asian?.trends?.[tf]}
            {@const europeanTrend = trendData.european?.trends?.[tf]}
            {@const usTrend = trendData.us?.trends?.[tf]}

            {#if asianTrend?.direction || europeanTrend?.direction || usTrend?.direction}
              <div class="timeframe-step">
                <span class="tf-label">{tf}</span>
                <div class="session-path">
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
          <div class="plan-session-notes">
            {#each ['asian', 'european', 'us'] as session}
              {#if trendData[session]?.notes}
                <div class="plan-note-item">
                  <span class="session-tag {session}"
                    >{session === 'asian' ? '亞盤' : session === 'european' ? '歐盤' : '美盤'}備註：</span
                  >
                  <span class="note-text">{trendData[session].notes}</span>
                </div>
              {/if}
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .trade-plan-status-section {
    background: #f8fafc;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1.5rem;
    border: 1px solid #e2e8f0;
  }

  .section-label-group {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }
  
  .strategy-label {
    font-weight: 600;
    color: #4a5568;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .plan-status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.8rem;
    border-radius: 6px;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    border: 1px solid transparent;
    transition: all 0.2s;
  }

  .plan-status-badge.linked {
    background: #f0fdf4;
    color: #166534;
    border-color: #bbf7d0;
  }
  .plan-status-badge.linked:hover {
    background: #dcfce7;
  }

  .plan-status-badge.missing {
    background: #fff1f2;
    color: #be123c;
    border-color: #fecdd3;
  }
  .plan-status-badge.missing:hover {
    background: #ffe4e6;
  }

  .view-link,
  .add-link {
    font-size: 0.8rem;
    opacity: 0.8;
  }

  .plan-details-summary {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px dashed #cbd5e0;
  }

  .plan-general-notes {
    font-size: 0.95rem;
    color: #4a5568;
    padding: 0.75rem;
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    margin-bottom: 1rem;
  }

  .progression-view {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .timeframe-step {
    display: flex;
    align-items: center;
    gap: 1rem;
    font-size: 0.9rem;
    padding: 0.25rem 0;
  }

  .tf-label {
    font-family: monospace;
    font-weight: 700;
    color: #718096;
    width: 32px;
  }

  .session-path {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .step {
    padding: 2px 6px;
    border-radius: 4px;
    background: #edf2f7;
    color: #718096;
    font-size: 0.8rem;
  }
  .step.long {
    background: #f0fdf4;
    color: #15803d;
  }
  .step.short {
    background: #fef2f2;
    color: #b91c1c;
  }

  .arrow {
    color: #cbd5e0;
    font-size: 0.8rem;
  }

  .plan-session-notes {
    margin-top: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .plan-note-item {
    display: flex;
    gap: 0.5rem;
    font-size: 0.85rem;
  }
  .session-tag {
    font-weight: 600;
  }
  .session-tag.asian {
    color: #6366f1;
  }
  .session-tag.european {
    color: #f43f5e;
  }
  .session-tag.us {
    color: #0ea5e9;
  }
</style>
