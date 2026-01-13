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

<div class="plan-summary-table-wrapper" class:detailed>
  <table class="plan-summary-table">
    <thead>
      <tr>
        <th></th>
        {#each TIMEFRAMES as tf}
          <th>{tf}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each MARKET_SESSIONS as session}
        {@const sessionData = trendData[session.value]}
        <tr>
          <td class="session-header {session.value}">{session.shortLabel || session.label[0]}</td>
          {#each TIMEFRAMES as tf}
            {@const trend = sessionData?.trends?.[tf]}
            <td class="trend-cell {trend?.direction || 'na'}">
              {#if trend?.direction}
                <div class="cell-direction">
                  {getDirectionLabel(trend.direction)}
                </div>
                
                {#if detailed}
                  {#if trend.direction === 'both'}
                    <div class="cell-details double">
                      <div class="side-box long">
                        {#if trend.long?.has_signals && trend.long.signals?.length > 0}
                          <div class="detail-item">達: {trend.long.signals.join(',')}</div>
                        {/if}
                        {#if trend.long?.has_expected_signals && trend.long.expected_signals?.length > 0}
                          <div class="detail-item">預: {trend.long.expected_signals.map(s => s.name).join(',')}</div>
                        {/if}
                        {#if trend.long?.has_wave && trend.long.wave_numbers?.length > 0}
                          <div class="detail-item">波: {trend.long.wave_numbers.map(n => n.toString() === trend.long.wave_highlight?.toString() ? `(${n})` : n).join('')}</div>
                        {/if}
                      </div>
                      <div class="side-box short">
                        {#if trend.short?.has_signals && trend.short.signals?.length > 0}
                          <div class="detail-item">達: {trend.short.signals.join(',')}</div>
                        {/if}
                        {#if trend.short?.has_expected_signals && trend.short.expected_signals?.length > 0}
                          <div class="detail-item">預: {trend.short.expected_signals.map(s => s.name).join(',')}</div>
                        {/if}
                        {#if trend.short?.has_wave && trend.short.wave_numbers?.length > 0}
                          <div class="detail-item">波: {trend.short.wave_numbers.map(n => n.toString() === trend.short.wave_highlight?.toString() ? `(${n})` : n).join('')}</div>
                        {/if}
                      </div>
                    </div>
                  {:else}
                    {@const dir = trend.direction}
                    {@const analysis = trend[dir] || trend}
                    <div class="cell-details single">
                      {#if analysis.has_signals && analysis.signals?.length > 0}
                        <div class="detail-item">達: {analysis.signals.join(',')}</div>
                      {/if}
                      {#if analysis.has_expected_signals && analysis.expected_signals?.length > 0}
                        <div class="detail-item">預: {analysis.expected_signals.map(s => s.name).join(',')}</div>
                      {/if}
                      {#if analysis.has_wave && analysis.wave_numbers?.length > 0}
                        <div class="detail-item">波: {analysis.wave_numbers.map(n => n.toString() === analysis.wave_highlight?.toString() ? `(${n})` : n).join('')}</div>
                      {/if}
                    </div>
                  {/if}
                {/if}
              {/if}
            </td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .plan-summary-table-wrapper {
    width: 100%;
    overflow-x: auto;
    border-radius: 8px;
    border: 1px solid var(--border-color, #e2e8f0);
    background: white;
  }

  .plan-summary-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.7rem;
    table-layout: fixed;
  }

  .plan-summary-table th, 
  .plan-summary-table td {
    border: 1px solid var(--border-color, #e2e8f0);
    padding: 4px;
    text-align: center;
    vertical-align: top;
  }

  .plan-summary-table th {
    background: #f8fafc;
    color: #64748b;
    font-weight: 800;
  }

  .session-header {
    font-weight: 800;
    width: 24px;
    background: #f8fafc;
  }
  .session-header.asian { color: #3b82f6; }
  .session-header.european { color: #d97706; }
  .session-header.us { color: #dc2626; }

  .trend-cell.long { background: #fff1f2; }
  .trend-cell.short { background: #f0fdf4; }
  .trend-cell.both { background: #f5f3ff; }
  .trend-cell.na { background: transparent; opacity: 0.3; }

  .cell-direction {
    font-weight: 900;
    font-size: 0.75rem;
  }
  .trend-cell.long .cell-direction { color: #dc2626; }
  .trend-cell.short .cell-direction { color: #16a34a; }
  .trend-cell.both .cell-direction { color: #6366f1; }

  /* Detailed View specific */
  .detailed .cell-direction {
    margin-bottom: 2px;
  }

  .cell-details {
    display: flex;
    flex-direction: column;
    gap: 1px;
    text-align: left;
    line-height: 1.1;
  }

  .detail-item {
    font-size: 0.6rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .double {
    gap: 4px;
  }

  .side-box {
    padding: 2px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.5);
  }
  .side-box.long { border-left: 2px solid #fee2e2; }
  .side-box.short { border-left: 2px solid #dcfce7; }

  .side-box.long::before { content: "多:"; display: block; font-size: 0.55rem; color: #dc2626; font-weight: 800; margin-bottom: 2px; }
  .side-box.short::before { content: "空:"; display: block; font-size: 0.55rem; color: #16a34a; font-weight: 800; margin-bottom: 2px; }

  :global(body.dark-mode) .plan-summary-table-wrapper {
    background: #1e293b;
    border-color: #334155;
  }
  
  :global(body.dark-mode) .plan-summary-table th,
  :global(body.dark-mode) .session-header {
    background: #0f172a;
    color: #94a3b8;
    border-color: #334155;
  }

  :global(body.dark-mode) .plan-summary-table td {
    border-color: #334155;
  }

  :global(body.dark-mode) .side-box {
    background: rgba(0, 0, 0, 0.2);
  }

  /* Responsive styling tip: horizontal scroll is handled by wrapper */
</style>
