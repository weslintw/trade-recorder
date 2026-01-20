<script>
  export let data = [];
  export let width = 120;
  export let height = 36;
  export let isOpen = false;

  $: parsedData = typeof data === 'string' ? JSON.parse(data || '[]') : data;

  let min, max, range, zeroLineY;

  $: if (!isOpen && parsedData.length > 1) {
    const dataMin = Math.min(...parsedData);
    const dataMax = Math.max(...parsedData);
    min = Math.min(0, dataMin);
    max = Math.max(0, dataMax);
    range = Math.max(0.00001, max - min);
    zeroLineY = 90 - ((0 - min) / range) * 80;
  }

  $: points =
    !isOpen && parsedData.length > 1
      ? parsedData
          .map((val, i) => {
            const x = (i / (parsedData.length - 1)) * 100;
            const y = 90 - ((val - min) / range) * 80;
            return `${x},${y}`;
          })
          .join(' ')
      : '';

  const gradientId = 'sparkline-gradient-' + Math.random().toString(36).substr(2, 9);
  const patternId = 'diagonal-stripes-' + Math.random().toString(36).substr(2, 9);

  // Profit/Loss colors
  $: profitColor = '#22c55e';
  $: lossColor = '#ef4444';

  // Since we are plotting PnL values (calculated by backend):
  // Positive values are Profit (Green), Negative values are Loss (Red).
  // The graph Top (0%) is positive, Bottom (100%) is negative.
  $: topColor = profitColor;
  $: bottomColor = lossColor;
</script>

<div
  class="sparkline-container"
  class:is-open={isOpen}
  style="width: {width}px; height: {height}px;"
  title={isOpen ? '開倉中，暫不顯示 PnL 波動' : parsedData.join(', ')}
>
  {#if isOpen}
    <svg viewBox="0 0 100 100" preserveAspectRatio="none" style="width: 100%; height: 100%;">
      <defs>
        <pattern id={patternId} patternUnits="userSpaceOnUse" width="10" height="10" patternTransform="rotate(45)">
          <line x1="0" y1="0" x2="0" y2="10" class="stripe-line" stroke-width="4" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#{patternId})" />
    </svg>
  {:else if parsedData && parsedData.length > 1}
    <svg viewBox="0 0 100 100" preserveAspectRatio="none" style="width: 100%; height: 100%;">
      <defs>
        <linearGradient id={gradientId} x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stop-color={topColor} />
          <stop offset="{zeroLineY}%" stop-color={topColor} />
          <stop offset="{zeroLineY}%" stop-color={bottomColor} />
          <stop offset="100%" stop-color={bottomColor} />
        </linearGradient>
      </defs>

      <!-- Baseline (Entry Price) -->
      <line
        x1="0"
        y1={zeroLineY}
        x2="100"
        y2={zeroLineY}
        stroke="#cbd5e1"
        stroke-width="1"
        stroke-dasharray="4,2"
        style="vector-effect: non-scaling-stroke;"
      />

      <polyline
        fill="none"
        stroke="url(#{gradientId})"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        {points}
        style="vector-effect: non-scaling-stroke;"
      />
    </svg>
  {:else}
    <div class="no-data"></div>
  {/if}
</div>

<style>
  .sparkline-container {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    position: relative;
    padding: 2px;
    background: rgba(248, 250, 252, 0.5);
    border-radius: 4px;
    border: 1px solid #f1f5f9;
    overflow: hidden;
  }

  .stripe-line {
    stroke: #e2e8f0;
  }

  .sparkline-container.is-open {
    background: #f8fafc;
    border-color: #e2e8f0;
  }

  :global(body.dark-mode) .sparkline-container {
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(body.dark-mode) .sparkline-container.is-open {
    background: rgba(0, 0, 0, 0.2);
  }

  :global(body.dark-mode) .stripe-line {
    stroke: #334155;
  }

  .no-data {
    width: 60%;
    height: 1px;
    background: #e2e8f0;
  }
</style>
