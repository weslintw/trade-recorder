<script>
  export let data = [];
  export let width = 120;
  export let height = 36;
  export let side = 'long'; // 'long' or 'short'

  $: parsedData = typeof data === 'string' ? JSON.parse(data || '[]') : data;

  let min, max, range, zeroLineY;

  $: if (parsedData.length > 1) {
    const dataMin = Math.min(...parsedData);
    const dataMax = Math.max(...parsedData);
    min = Math.min(0, dataMin);
    max = Math.max(0, dataMax);
    range = Math.max(0.00001, max - min);
    zeroLineY = 90 - ((0 - min) / range) * 80;
  }

  $: points =
    parsedData.length > 1
      ? parsedData
          .map((val, i) => {
            const x = (i / (parsedData.length - 1)) * 100;
            const y = 90 - ((val - min) / range) * 80;
            return `${x},${y}`;
          })
          .join(' ')
      : '';

  const gradientId = 'sparkline-gradient-' + Math.random().toString(36).substr(2, 9);

  // Profit/Loss colors
  $: profitColor = '#22c55e';
  $: lossColor = '#ef4444';

  // For Long: Top (above baseline) is Profit. For Short: Bottom (below baseline) is Profit.
  $: topColor = side === 'short' ? lossColor : profitColor;
  $: bottomColor = side === 'short' ? profitColor : lossColor;
</script>

<div class="sparkline-container" style="width: {width}px; height: {height}px;" title={parsedData.join(', ')}>
  {#if parsedData && parsedData.length > 1}
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
  }
  .no-data {
    width: 60%;
    height: 1px;
    background: #e2e8f0;
  }
</style>
