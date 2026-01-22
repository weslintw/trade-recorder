<script>
  import { TIMEFRAMES, MARKET_SESSIONS } from '../lib/constants';

  export let trendData = {};
  export let detailed = false;

  function getDirectionLabel(dir) {
    if (dir === 'long') return '多';
    if (dir === 'short') return '空';
    if (dir === 'both') return '整';
    return '';
  }
</script>

<div class="plan-summary-container" class:is-detailed={detailed}>
  <div class="summary-scroll-wrapper">
    <table class="modern-summary-table">
      <thead>
        <tr>
          <th class="sticky-col header-corner"></th>
          {#each TIMEFRAMES as tf}
            <th>{tf}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each MARKET_SESSIONS as session}
          {@const sessionData = trendData[session.value]}
          <tr>
            <td class="sticky-col session-label {session.value}">
              <div class="session-text">{session.shortLabel || session.label[0]}</div>
            </td>
            {#each TIMEFRAMES as tf}
              {@const trend = sessionData?.trends?.[tf]}
              <td class="trend-cell-container {trend?.direction || 'na'}">
                {#if trend?.direction}
                  <div class="direction-badge">
                    {getDirectionLabel(trend.direction)}
                  </div>
                  
                  {#if detailed}
                    <div class="content-stack">
                      {#if trend.direction === 'both'}
                        <div class="dual-content">
                          <div class="side-panel long">
                            {#if trend.long?.has_signals && trend.long.signals?.length > 0}
                              <div class="info-row"><span class="tag established">達</span>{trend.long.signals.join(', ')}</div>
                            {/if}
                            {#if trend.long?.has_expected_signals && trend.long.expected_signals?.length > 0}
                              <div class="info-row"><span class="tag expected">預</span>{trend.long.expected_signals.map(s => s.name).join(', ')}</div>
                            {/if}
                            {#if trend.long?.has_wave && trend.long.wave_numbers?.length > 0}
                              <div class="info-row wave">
                                <span class="tag wave-tag">波</span>
                                {trend.long.wave_numbers.map((n, i) => {
                                  const isHighlight = n.toString() === trend.long.wave_highlight?.toString();
                                  const val = isHighlight ? `[${n}]` : n;
                                  return (i > 0 ? ' => ' : '') + val;
                                }).join('')}
                              </div>
                            {/if}
                          </div>
                          <div class="side-panel short">
                            {#if trend.short?.has_signals && trend.short.signals?.length > 0}
                              <div class="info-row"><span class="tag established">達</span>{trend.short.signals.join(', ')}</div>
                            {/if}
                            {#if trend.short?.has_expected_signals && trend.short.expected_signals?.length > 0}
                              <div class="info-row"><span class="tag expected">預</span>{trend.short.expected_signals.map(s => s.name).join(', ')}</div>
                            {/if}
                            {#if trend.short?.has_wave && trend.short.wave_numbers?.length > 0}
                              <div class="info-row wave">
                                <span class="tag wave-tag">波</span>
                                {trend.short.wave_numbers.map((n, i) => {
                                  const isHighlight = n.toString() === trend.short.wave_highlight?.toString();
                                  const val = isHighlight ? `[${n}]` : n;
                                  return (i > 0 ? ' => ' : '') + val;
                                }).join('')}
                              </div>
                            {/if}
                          </div>
                        </div>
                      {:else}
                        {@const dir = trend.direction}
                        {@const analysis = trend[dir] || trend}
                        <div class="single-content">
                          {#if analysis.has_signals && analysis.signals?.length > 0}
                            <div class="info-row"><span class="tag established">達</span>{analysis.signals.join(', ')}</div>
                          {/if}
                          {#if analysis.has_expected_signals && analysis.expected_signals?.length > 0}
                            <div class="info-row"><span class="tag expected">預</span>{analysis.expected_signals.map(s => s.name).join(', ')}</div>
                          {/if}
                          {#if analysis.has_wave && analysis.wave_numbers?.length > 0}
                            <div class="info-row wave">
                              <span class="tag wave-tag">波</span>
                              {analysis.wave_numbers.map((n, i) => {
                                const isHighlight = n.toString() === analysis.wave_highlight?.toString();
                                const val = isHighlight ? `[${n}]` : n;
                                return (i > 0 ? ' => ' : '') + val;
                              }).join('')}
                            </div>
                          {/if}
                        </div>
                      {/if}
                    </div>
                  {/if}
                {:else}
                  <div class="na-placeholder">NA</div>
                {/if}
              </td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<style>
  .plan-summary-container {
    --primary-font: 'Outfit', 'Inter', 'Noto Sans TC', 'PingFang TC', 'Microsoft JhengHei', sans-serif;
    --bg-long: hsla(354, 100%, 98%, 1);
    --bg-short: hsla(142, 70%, 98%, 1);
    --bg-both: hsla(255, 100%, 98.5%, 1);
    --text-long: #e11d48;
    --text-short: #16a34a;
    --text-both: #6366f1;
    --border-color: #f1f5f9;
    
    width: 100%;
    background: #ffffff;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -1px rgba(0, 0, 0, 0.03);
    border: 1px solid #e2e8f0;
    overflow: hidden;
  }

  .summary-scroll-wrapper {
    overflow-x: auto;
    width: 100%;
  }

  .modern-summary-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 4px;
    table-layout: fixed;
    font-family: var(--primary-font);
  }

  /* Sticky First Column for Session Labels */
  .sticky-col {
    position: sticky;
    left: 0;
    z-index: 10;
    background: #ffffff;
    width: 32px;
    min-width: 32px;
  }

  :global(body.dark-mode) .sticky-col {
    background: #0f172a;
  }

  .modern-summary-table th {
    background: transparent;
    color: #94a3b8;
    font-size: 0.7rem;
    font-weight: 700;
    padding: 8px 4px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border: none;
  }

  .trend-cell-container {
    vertical-align: top;
    padding: 10px;
    min-height: 80px;
    border-radius: 8px;
    background: #f8fafc;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    border: 1px solid transparent;
  }
  
  :global(body.dark-mode) .trend-cell-container {
    background: rgba(255, 255, 255, 0.02);
  }

  /* Direction Styling */
  /* Direction Styling - Removed background as requested */
  .trend-cell-container.long { border-color: rgba(225, 29, 72, 0.1); }
  .trend-cell-container.short { border-color: rgba(22, 163, 74, 0.1); }
  .trend-cell-container.both { border-color: rgba(99, 102, 241, 0.1); }
  .trend-cell-container.na { 
    background-color: #f8fafc;
    opacity: 0.6;
  }

  :global(body.dark-mode) .trend-cell-container.na {
    background-color: rgba(255, 255, 255, 0.03);
  }

  .session-label {
    text-align: center;
    vertical-align: middle;
  }

  .session-text {
    font-weight: 900;
    font-size: 0.85rem;
    writing-mode: horizontal-tb;
    opacity: 0.8;
    line-height: 1;
  }

  .session-label.asian .session-text { color: #3b82f6; }
  .session-label.european .session-text { color: #ea580c; }
  .session-label.us .session-text { color: #dc2626; }

  .na-placeholder {
    color: #94a3b8;
    font-size: 0.65rem;
    font-weight: 700;
    text-align: center;
    letter-spacing: 1px;
    line-height: 1;
  }

  .direction-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2px 0;
    width: 100%;
    border-radius: 4px;
    font-weight: 800;
    font-size: 0.9rem;
    margin-bottom: 0;
    letter-spacing: 0.05em;
    line-height: 1;
  }

  .is-detailed .direction-badge {
    margin-bottom: 8px;
  }

  .long .direction-badge { color: var(--text-long); }
  .short .direction-badge { color: var(--text-short); }
  .both .direction-badge { color: var(--text-both); }

  /* Detailed Layout Improvements */
  .content-stack {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .info-row {
    font-size: 0.72rem;
    color: #334155;
    line-height: 1.4;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    gap: 4px;
    font-weight: 500;
  }

  .icon {
    font-size: 0.75rem;
    flex-shrink: 0;
  }

  /* Tag style instead of plain text icon */
  .tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border-radius: 4px;
    font-size: 0.65rem;
    font-weight: 800;
    color: white;
    flex-shrink: 0;
    margin-top: 1px;
    box-shadow: 0 1px 2px rgba(0,0,0,0.1);
  }

  .tag.established { background: #64748b; }
  .tag.expected { background: #a855f7; }
  .tag.wave-tag { background: #38bdf8; }

  .info-row.wave {
    color: #1e293b;
    font-family: inherit;
    letter-spacing: 1px;
    font-weight: 700;
  }

  /* Dual Panel for "Both" direction */
  .dual-content {
    display: flex;
    gap: 8px;
    padding-top: 2px;
  }

  .side-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 4px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.4);
    box-shadow: inset 0 0 0 1px rgba(0,0,0,0.03);
  }

  .side-panel::before {
    font-size: 0.75rem;
    font-weight: 800;
    text-transform: uppercase;
    display: block;
    margin-bottom: 4px;
    text-align: center;
    padding: 2px 0;
    background: rgba(255, 255, 255, 0.6);
    border-radius: 3px;
  }
  .side-panel.long::before { content: '多'; color: var(--text-long); }
  .side-panel.short::before { content: '空'; color: var(--text-short); }

  /* Dark Mode Support */
  :global(body.dark-mode) .plan-summary-container {
    background: #0f172a;
    border-color: #334155;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
  }

  :global(body.dark-mode) .modern-summary-table th,
  :global(body.dark-mode) .sticky-col {
    background: #1e293b;
    color: #94a3b8;
    border-color: #334155;
  }

  :global(body.dark-mode) .trend-cell-container {
    border-color: #1e293b;
  }

  :global(body.dark-mode) .trend-cell-container.long { background-color: transparent; }
  :global(body.dark-mode) .trend-cell-container.short { background-color: transparent; }
  :global(body.dark-mode) .trend-cell-container.both { background-color: transparent; }

  :global(body.dark-mode) .info-row {
    color: #cbd5e1;
  }

  :global(body.dark-mode) .side-panel {
    background: rgba(0, 0, 0, 0.2);
    box-shadow: inset 0 0 0 1px rgba(255,255,255,0.05);
  }

  @media (max-width: 640px) {
    .modern-summary-table {
      font-size: 0.65rem;
    }
    .sticky-col {
      width: 32px;
      min-width: 32px;
    }
  }
</style>
