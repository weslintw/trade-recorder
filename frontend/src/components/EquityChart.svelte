<script>
  import { onMount, onDestroy, afterUpdate } from 'svelte';
  import * as echarts from 'echarts/core';
  import { LineChart, CustomChart } from 'echarts/charts';
  import {
    GridComponent,
    TooltipComponent,
    LegendComponent,
    DataZoomComponent,
    MarkLineComponent,
    TitleComponent,
  } from 'echarts/components';
  import { CanvasRenderer } from 'echarts/renderers';

  echarts.use([
    LineChart,
    CustomChart,
    GridComponent,
    TooltipComponent,
    LegendComponent,
    DataZoomComponent,
    MarkLineComponent,
    TitleComponent,
    CanvasRenderer,
  ]);

  export let data = [];

  let chartEl;
  let chartInstance;
  let resizeObserver;

  $: chartOption = buildOption(data);
  $: if (chartInstance && chartOption) {
    chartInstance.setOption(chartOption, { notMerge: true });
  }

  onMount(() => {
    if (!chartEl) return;
    chartInstance = echarts.init(chartEl, null, { renderer: 'canvas' });
    chartInstance.setOption(chartOption, { notMerge: true });

    resizeObserver = new ResizeObserver(() => {
      chartInstance && chartInstance.resize();
    });
    resizeObserver.observe(chartEl);
  });

  onDestroy(() => {
    if (resizeObserver) resizeObserver.disconnect();
    if (chartInstance) {
      chartInstance.dispose();
      chartInstance = null;
    }
  });

  function resetZoom() {
    if (!chartInstance) return;
    chartInstance.dispatchAction({
      type: 'dataZoom',
      start: 0,
      end: 100,
    });
  }

  function buildOption(points) {
    const isDark =
      typeof document !== 'undefined' && document.body.classList.contains('dark-mode');

    const axisColor = isDark ? '#94a3b8' : '#64748b';
    const gridColor = isDark ? 'rgba(148, 163, 184, 0.15)' : 'rgba(148, 163, 184, 0.25)';
    const bgColor = isDark ? '#0f172a' : 'transparent';
    const textColor = isDark ? '#f1f5f9' : '#1e293b';

    const tpColor = '#22c55e';
    const slColor = '#ef4444';
    const beColor = '#a3a3a3';

    const safe = Array.isArray(points) ? points : [];

    // 第一個點為基準線 (起始 0)
    const startLabel = 'Start';
    const labels = [startLabel, ...safe.map(p => p.time || p.date || '')];
    const equityValues = [0, ...safe.map(p => Number(p.equity) || 0)];

    // 每個點的顏色
    const pointColors = [beColor, ...safe.map(p => {
      if (p.result === 'tp' || (typeof p.pnl === 'number' && p.pnl > 0)) return tpColor;
      if (p.result === 'sl' || (typeof p.pnl === 'number' && p.pnl < 0)) return slColor;
      return beColor;
    })];

    // 把線段拆成綠/紅兩條 series（依該段是上漲或下跌）
    // 使用 visualMap 的 piecewise 不容易隨點變色，這裡用 custom segment 較直接：
    // 但為了簡潔，採用 chart-wide line + markers，並用 markLine 標出 base。

    function fmtTime(t) {
      if (!t) return '';
      const d = new Date(t);
      if (Number.isNaN(d.getTime())) return String(t);
      const yy = String(d.getFullYear()).slice(2);
      const mm = String(d.getMonth() + 1).padStart(2, '0');
      const dd = String(d.getDate()).padStart(2, '0');
      const hh = String(d.getHours()).padStart(2, '0');
      const mi = String(d.getMinutes()).padStart(2, '0');
      return `${yy}/${mm}/${dd} ${hh}:${mi}`;
    }

    function fmtAxis(t) {
      if (t === startLabel) return 'Start';
      if (!t) return '';
      const d = new Date(t);
      if (Number.isNaN(d.getTime())) return String(t);
      const yy = String(d.getFullYear()).slice(2);
      const mm = String(d.getMonth() + 1).padStart(2, '0');
      const dd = String(d.getDate()).padStart(2, '0');
      return `${yy}/${mm}/${dd}`;
    }

    return {
      backgroundColor: bgColor,
      animationDuration: 400,
      grid: {
        left: 60,
        right: 30,
        top: 30,
        bottom: 90,
      },
      legend: {
        data: [
          { name: '獲利', icon: 'circle', itemStyle: { color: tpColor } },
          { name: '虧損', icon: 'circle', itemStyle: { color: slColor } },
        ],
        right: 80,
        top: 4,
        textStyle: { color: textColor },
      },
      tooltip: {
        trigger: 'axis',
        backgroundColor: isDark ? 'rgba(15,23,42,0.95)' : 'rgba(255,255,255,0.95)',
        borderColor: isDark ? '#334155' : '#e2e8f0',
        textStyle: { color: textColor },
        formatter: (params) => {
          if (!params || !params.length) return '';
          const p = params[0];
          const idx = p.dataIndex;
          if (idx === 0) {
            return `<div style="font-weight:700;margin-bottom:4px;">起始</div>
                    累積：<b>0.00</b>`;
          }
          const point = safe[idx - 1];
          if (!point) return '';
          const resLabel =
            point.result === 'tp' ? '<span style="color:' + tpColor + ';">● 獲利</span>' :
            point.result === 'sl' ? '<span style="color:' + slColor + ';">● 虧損</span>' :
            '<span style="color:' + beColor + ';">● 平手</span>';
          const pnl = Number(point.pnl) || 0;
          const eq = Number(point.equity) || 0;
          return `
            <div style="font-weight:700;margin-bottom:4px;">${fmtTime(point.time || point.date)}</div>
            <div>${resLabel} ${point.symbol || ''} ${point.side ? '(' + point.side + ')' : ''}</div>
            <div>該筆：<b style="color:${pnl >= 0 ? tpColor : slColor}">${pnl >= 0 ? '+' : ''}${pnl.toFixed(2)}</b></div>
            <div>累積：<b>${eq.toFixed(2)}</b></div>
          `;
        },
      },
      xAxis: {
        type: 'category',
        data: labels,
        boundaryGap: false,
        axisLine: { lineStyle: { color: axisColor } },
        axisLabel: {
          color: axisColor,
          formatter: fmtAxis,
          hideOverlap: true,
        },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        scale: true,
        axisLine: { show: false },
        axisLabel: {
          color: axisColor,
          formatter: (v) => Number(v).toFixed(2),
        },
        splitLine: { lineStyle: { color: gridColor, type: 'dashed' } },
      },
      dataZoom: [
        {
          type: 'inside',
          start: 0,
          end: 100,
          zoomOnMouseWheel: true,
          moveOnMouseMove: true,
          moveOnMouseWheel: false,
        },
        {
          type: 'slider',
          start: 0,
          end: 100,
          height: 40,
          bottom: 10,
          borderColor: 'transparent',
          backgroundColor: isDark ? 'rgba(30,41,59,0.4)' : 'rgba(148,163,184,0.1)',
          fillerColor: isDark ? 'rgba(99,102,241,0.25)' : 'rgba(99,102,241,0.15)',
          handleStyle: { color: '#6366f1' },
          moveHandleStyle: { color: '#6366f1' },
          dataBackground: {
            lineStyle: { color: tpColor, width: 1 },
            areaStyle: { color: 'rgba(34,197,94,0.2)' },
          },
          selectedDataBackground: {
            lineStyle: { color: tpColor, width: 1.5 },
            areaStyle: { color: 'rgba(34,197,94,0.35)' },
          },
          textStyle: { color: axisColor },
        },
      ],
      series: [
        {
          name: '收益',
          type: 'line',
          showSymbol: true,
          symbolSize: 8,
          data: equityValues.map((v, i) => ({
            value: v,
            itemStyle: { color: pointColors[i], borderColor: '#fff', borderWidth: 1 },
          })),
          // 區段顏色：往上綠、往下紅
          lineStyle: { width: 2.5 },
          itemStyle: { color: tpColor },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(34,197,94,0.35)' },
              { offset: 1, color: 'rgba(34,197,94,0.01)' },
            ]),
          },
          smooth: false,
          emphasis: { focus: 'series' },
          markLine: {
            silent: true,
            symbol: 'none',
            lineStyle: { color: axisColor, type: 'dashed', width: 1 },
            label: { formatter: 'base', color: axisColor, position: 'insideStartTop' },
            data: [{ yAxis: 0 }],
          },
        },
        // 虧損區段（紅色覆蓋線）：以線段方式重繪「下跌段」
        {
          name: '虧損段',
          type: 'custom',
          renderItem: (params, api) => {
            const i = params.dataIndex;
            if (i === 0) return null;
            const y0 = equityValues[i - 1];
            const y1 = equityValues[i];
            if (y1 >= y0) return null;
            const p0 = api.coord([i - 1, y0]);
            const p1 = api.coord([i, y1]);
            return {
              type: 'line',
              shape: { x1: p0[0], y1: p0[1], x2: p1[0], y2: p1[1] },
              style: { stroke: slColor, lineWidth: 2.5 },
            };
          },
          data: equityValues.map((v, i) => [i, v]),
          z: 5,
          tooltip: { show: false },
          silent: true,
        },
        // legend dummy
        { name: '獲利', type: 'line', data: [], itemStyle: { color: tpColor } },
        { name: '虧損', type: 'line', data: [], itemStyle: { color: slColor } },
      ],
    };
  }

  // dark-mode 切換時重繪
  afterUpdate(() => {
    if (chartInstance) {
      chartInstance.setOption(buildOption(data), { notMerge: true });
    }
  });
</script>

<div class="equity-chart-card">
  <div class="chart-header">
    <div class="title-group">
      <span class="title-tag" aria-hidden="true"></span>
      <h3 class="title">收益走勢圖 <span class="subtitle">Equity Curve</span></h3>
    </div>
    <button class="reset-btn" type="button" on:click={resetZoom}>
      <span class="reset-icon" aria-hidden="true">⟳</span>
      重置視圖
    </button>
  </div>
  <p class="hint">拖動平移、滾輪縮放、底部拖動選取範圍</p>
  <div class="chart-canvas" bind:this={chartEl}></div>
</div>

<style>
  .equity-chart-card {
    width: 100%;
    background: #0f172a;
    border-radius: 16px;
    padding: 1rem 1.25rem 0.5rem;
    color: #f1f5f9;
    position: relative;
  }

  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.25rem;
  }

  .title-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .title-tag {
    width: 14px;
    height: 14px;
    background: #ef4444;
    border-radius: 3px;
    display: inline-block;
    position: relative;
  }
  .title-tag::after {
    content: '';
    position: absolute;
    inset: 3px;
    background: #0f172a;
    border-radius: 2px;
  }

  .title {
    font-size: 1.1rem;
    font-weight: 800;
    margin: 0;
    color: #f1f5f9;
  }

  .subtitle {
    font-size: 0.85rem;
    color: #94a3b8;
    font-weight: 500;
    margin-left: 0.4rem;
  }

  .hint {
    margin: 0 0 0.25rem;
    font-size: 0.75rem;
    color: #64748b;
  }

  .reset-btn {
    background: rgba(99, 102, 241, 0.15);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.4);
    border-radius: 8px;
    padding: 0.35rem 0.75rem;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    transition: background 0.15s ease;
  }
  .reset-btn:hover {
    background: rgba(99, 102, 241, 0.25);
  }
  .reset-icon {
    font-size: 0.95rem;
  }

  .chart-canvas {
    width: 100%;
    height: 420px;
  }

  /* Light mode 沿用深色背景以與附圖一致；若想跟著主題切換，把下面打開 */
  /*
  :global(body:not(.dark-mode)) .equity-chart-card {
    background: #ffffff;
    color: #1e293b;
  }
  :global(body:not(.dark-mode)) .title { color: #1e293b; }
  :global(body:not(.dark-mode)) .title-tag::after { background: #ffffff; }
  */
</style>
