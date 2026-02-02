<script>
  import { onMount, onDestroy } from 'svelte';
  import { createChart, CandlestickSeries, LineSeries, createSeriesMarkers } from 'lightweight-charts';
  import { tradesAPI } from '../lib/api';

  export let tradeId;
  export let trade = null; 

  let chartContainer;
  let chart;
  let candlestickSeries;
  let loading = true;
  let error = null;
  let timeframe = '';
  let copying = false;
  let drawingActive = false;
  let firstPoint = null;
  let drawnLines = []; // Array of { series, p1: {time, price}, p2: {time, price} }
  let isFullscreen = false;
  let chartWrapper;
  
  // Interactive drawing states
  let previewLine = null;
  let draggingPoint = null; // { lineIndex, pointIndex (0 or 1) }
  let selectedLineIndex = null;
  let rafId = null;

  onMount(function() {
    initChart();
    loadData();

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);
    window.addEventListener('keydown', handleKeydown);
    return function() {
      if (rafId) {
        cancelAnimationFrame(rafId);
      }
      if (chart) {
        chart.remove();
      }
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('mozfullscreenchange', handleFullscreenChange);
      document.removeEventListener('MSFullscreenChange', handleFullscreenChange);
      window.removeEventListener('keydown', handleKeydown);
    };
  });

  function initChart() {
    if (!chartContainer) return;
    
    chart = createChart(chartContainer, {
      layout: {
        background: { color: '#0f172a' },
        textColor: '#94a3b8',
      },
      grid: {
        vertLines: { color: 'rgba(255, 255, 255, 0.05)' },
        horzLines: { color: 'rgba(255, 255, 255, 0.05)' },
      },
      rightPriceScale: {
        borderColor: 'rgba(255, 255, 255, 0.1)',
        scaleMargins: {
          top: 0.05,
          bottom: 0.05,
        },
      },
      timeScale: {
        borderColor: 'rgba(255, 255, 255, 0.1)',
        timeVisible: true,
        secondsVisible: false,
      },
      crosshair: {
        mode: 0,
      },
    });

    chart.subscribeClick(handleChartClick);
    chart.subscribeCrosshairMove(handleCrosshairMove);

    candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#ef4444',
      downColor: '#ffffff',
      borderVisible: false,
      wickUpColor: '#ef4444',
      wickDownColor: '#ffffff',
    });

    function handleResize() {
      if (chart && chartContainer) {
        chart.applyOptions({ width: chartContainer.clientWidth });
      }
    }
    
    window.addEventListener('resize', handleResize);
    
    return function() {
      window.removeEventListener('resize', handleResize);
    };
  }

  async function loadData() {
    if (!tradeId) return;
    try {
      loading = true;
      error = null;
      const res = await tradesAPI.getChartData(tradeId);
      
      const resData = res.data;
      const data = resData.data;
      const digits = resData.digits || 2;
      timeframe = resData.timeframe || '';

      if (!data || !data.trendbar || data.trendbar.length === 0) {
        error = "無法獲取 K 線數據 (數據可能已過期或伺服器暫時無法連接)";
        return;
      }

      // cTrader 價格數據在 protobuf 中統一縮放 10^5
      const scale = 100000;
      const chartData = [];
      for (let i = 0; i < data.trendbar.length; i++) {
        const bar = data.trendbar[i];
        chartData.push({
          time: bar.utcTimestampInMinutes * 60,
          open: (bar.low + bar.deltaOpen) / scale,
          high: (bar.low + bar.deltaHigh) / scale,
          low: bar.low / scale,
          close: (bar.low + bar.deltaClose) / scale,
        });
      }

      // 設置價格精細度
      candlestickSeries.applyOptions({
        priceFormat: {
          type: 'price',
          precision: digits,
          minMove: 1 / Math.pow(10, digits),
        },
      });

      // 排序並過濾重複時間點
      chartData.sort(function(a, b) { return a.time - b.time; });
      const uniqueData = [];
      let lastTime = 0;
      for (let j = 0; j < chartData.length; j++) {
        const d = chartData[j];
        if (d.time > lastTime) {
          uniqueData.push(d);
          lastTime = d.time;
        }
      }
      
      candlestickSeries.setData(uniqueData);

      // 設置標註 (Markers)
      const markers = [];
      if (trade) {
        const entryTs = Math.floor(new Date(trade.entry_time).getTime() / 1000);
        const exitTs = trade.exit_time ? Math.floor(new Date(trade.exit_time).getTime() / 1000) : null;
        
        // 修正：確保標記的價格也符合縮放
        markers.push({
          time: entryTs,
          position: 'belowBar', // 始終放在下方避免擋到 K 線
          color: trade.side === 'long' ? '#10b981' : '#ef4444',
          shape: trade.side === 'long' ? 'arrowUp' : 'arrowDown',
          text: 'Entry @ ' + trade.entry_price,
        });

        if (exitTs) {
          markers.push({
            time: exitTs,
            position: 'aboveBar', // 始終放在上方
            color: '#3b82f6',
            shape: 'balloon',
            text: 'Exit @ ' + trade.exit_price,
          });
        }
      }
      
      markers.sort(function(a, b) { return a.time - b.time; });
      createSeriesMarkers(candlestickSeries, markers);
      
      // 視野管理：獲取了 1200 根，但初始視野只顯示中間的 400 根
      const totalLen = uniqueData.length;
      if (totalLen > 400) {
        const midIdx = Math.floor(totalLen / 2);
        chart.timeScale().setVisibleLogicalRange({
          from: midIdx - 200,
          to: midIdx + 200,
        });
      } else {
        chart.timeScale().fitContent();
      }

    } catch (e) {
      console.error('[Chart Error]', e);
      const errorMsg = e.response && e.response.data && e.response.data.error ? e.response.data.error : e.message;
      error = "載入失敗: " + errorMsg;
    } finally {
      loading = false;
    }
  }

  function handleChartClick(param) {
    if (!param || !param.point || !param.time) return;
    
    const price = candlestickSeries.coordinateToPrice(param.point.y);
    if (!price) return;

    // --- Dragging & Selection Logic (Outside drawing mode) ---
    if (!drawingActive) {
      if (draggingPoint) {
        draggingPoint = null;
        chart.applyOptions({ handleScroll: true, handleScale: true });
        return;
      }
      
      // Try to pick up a point (15px threshold)
      for (let i = 0; i < drawnLines.length; i++) {
        const line = drawnLines[i];
        const p1Coord = candlestickSeries.priceToCoordinate(line.p1.price);
        const p1TimeCoord = chart.timeScale().timeToCoordinate(line.p1.time);
        const p2Coord = candlestickSeries.priceToCoordinate(line.p2.price);
        const p2TimeCoord = chart.timeScale().timeToCoordinate(line.p2.time);
        
        const dist1 = Math.sqrt(Math.pow(param.point.x - p1TimeCoord, 2) + Math.pow(param.point.y - p1Coord, 2));
        const dist2 = Math.sqrt(Math.pow(param.point.x - p2TimeCoord, 2) + Math.pow(param.point.y - p2Coord, 2));
        
        if (dist1 < 15) {
          draggingPoint = { lineIndex: i, pointIndex: 0 };
          selectedLineIndex = i;
          updateSelectedStyles();
          chart.applyOptions({ handleScroll: false, handleScale: false });
          return;
        } else if (dist2 < 15) {
          draggingPoint = { lineIndex: i, pointIndex: 1 };
          selectedLineIndex = i;
          updateSelectedStyles();
          chart.applyOptions({ handleScroll: false, handleScale: false });
          return;
        }
      }

      // Try to select a line (click detection on the segment)
      for (let i = 0; i < drawnLines.length; i++) {
        const line = drawnLines[i];
        const p1Y = candlestickSeries.priceToCoordinate(line.p1.price);
        const p1X = chart.timeScale().timeToCoordinate(line.p1.time);
        const p2Y = candlestickSeries.priceToCoordinate(line.p2.price);
        const p2X = chart.timeScale().timeToCoordinate(line.p2.time);

        // Distance from point to segment
        const A = param.point.x - p1X;
        const B = param.point.y - p1Y;
        const C = p2X - p1X;
        const D = p2Y - p1Y;
        const dot = A * C + B * D;
        const len_sq = C * C + D * D;
        let param_t = -1;
        if (len_sq !== 0) param_t = dot / len_sq;
        let xx, yy;
        if (param_t < 0) { xx = p1X; yy = p1Y; }
        else if (param_t > 1) { xx = p2X; yy = p2Y; }
        else { xx = p1X + param_t * C; yy = p1Y + param_t * D; }
        const dx = param.point.x - xx;
        const dy = param.point.y - yy;
        const dist = Math.sqrt(dx * dx + dy * dy);

        if (dist < 10) {
          selectedLineIndex = i;
          updateSelectedStyles();
          return;
        }
      }

      if (selectedLineIndex !== null) {
        selectedLineIndex = null;
        updateSelectedStyles();
      }
      return;
    }
    
    // --- Drawing Logic ---
    if (!firstPoint) {
      firstPoint = { time: param.time, price: price };
      // Create preview line
      previewLine = chart.addSeries(LineSeries, {
        color: 'rgba(245, 158, 11, 0.5)',
        lineWidth: 1,
        lineStyle: 2, // Dashed
        lastValueVisible: false,
        priceLineVisible: false,
        crosshairMarkerVisible: false,
      });
    } else {
      const secondPoint = { time: param.time, price: price };
      addTrendline(firstPoint, secondPoint);
      
      // Explicitly exit drawing mode
      drawingActive = false;
      if (previewLine) {
        chart.removeSeries(previewLine);
        previewLine = null;
      }
      firstPoint = null;
      chart.applyOptions({ handleScroll: true, handleScale: true });
    }
  }

  function handleCrosshairMove(param) {
    if (!param || !param.point || !param.time) return;
    
    // Use requestAnimationFrame to avoid "Maximum call stack size exceeded"
    // which can happen if setData triggers a layout change that triggers crosshairMove.
    if (rafId) cancelAnimationFrame(rafId);
    
    rafId = requestAnimationFrame(function() {
      const price = candlestickSeries.coordinateToPrice(param.point.y);
      if (!price) return;

      // Drawing preview
      if (drawingActive && firstPoint && previewLine) {
        const lineData = [
          { time: firstPoint.time, value: firstPoint.price },
          { time: param.time, value: price }
        ];
        lineData.sort(function(a, b) { return a.time - b.time; });
        previewLine.setData(lineData);
      }

      // Dragging update
      if (draggingPoint) {
        const line = drawnLines[draggingPoint.lineIndex];
        if (draggingPoint.pointIndex === 0) {
          line.p1 = { time: param.time, price: price };
        } else {
          line.p2 = { time: param.time, price: price };
        }
        
        const lineData = [
          { time: line.p1.time, value: line.p1.price },
          { time: line.p2.time, value: line.p2.price }
        ];
        lineData.sort(function(a, b) { return a.time - b.time; });
        line.series.setData(lineData);
      }
    });
  }

  function addTrendline(p1, p2) {
    const series = chart.addSeries(LineSeries, {
      color: '#f59e0b',
      lineWidth: 2,
      lastValueVisible: false,
      priceLineVisible: false,
      crosshairMarkerVisible: false,
    });
    
    const lineData = [
      { time: p1.time, value: p1.price },
      { time: p2.time, value: p2.price }
    ];
    lineData.sort(function(a, b) { return a.time - b.time; });
    series.setData(lineData);
    
    drawnLines.push({ series: series, p1: p1, p2: p2 });
    drawnLines = drawnLines;
    selectedLineIndex = drawnLines.length - 1;
    updateSelectedStyles();
  }

  function updateSelectedStyles() {
    for (let i = 0; i < drawnLines.length; i++) {
      drawnLines[i].series.applyOptions({
        color: i === selectedLineIndex ? '#3b82f6' : '#f59e0b',
        lineWidth: i === selectedLineIndex ? 3 : 2,
      });
    }
  }

  function deleteSelectedLine() {
    if (selectedLineIndex === null) return;
    chart.removeSeries(drawnLines[selectedLineIndex].series);
    drawnLines.splice(selectedLineIndex, 1);
    drawnLines = drawnLines;
    selectedLineIndex = null;
  }

  function toggleDrawing() {
    drawingActive = !drawingActive;
    if (drawingActive) {
      selectedLineIndex = null;
      updateSelectedStyles();
    } else {
      if (previewLine) {
        chart.removeSeries(previewLine);
        previewLine = null;
      }
      firstPoint = null;
    }
    draggingPoint = null;
    chart.applyOptions({ handleScroll: true, handleScale: true });
  }

  function clearDrawings() {
    for (let i = 0; i < drawnLines.length; i++) {
       chart.removeSeries(drawnLines[i].series);
    }
    if (previewLine) {
      chart.removeSeries(previewLine);
      previewLine = null;
    }
    drawnLines = [];
    firstPoint = null;
    drawingActive = false;
    draggingPoint = null;
    selectedLineIndex = null;
    chart.applyOptions({ handleScroll: true, handleScale: true });
  }

  async function copyChartImage() {
    if (!chart || copying) return;
    try {
      copying = true;
      const canvas = chart.takeScreenshot();
      canvas.toBlob(function(blob) {
        if (!blob) {
          copying = false;
          return;
        }
        
        const data = [new ClipboardItem({ 'image/png': blob })];
        navigator.clipboard.write(data).then(function() {
          setTimeout(function() {
            copying = false;
          }, 2000);
        }).catch(function(err) {
          console.error('Clipboard error:', err);
          copying = false;
        });
      });
    } catch (e) {
      console.error('Screenshot error:', e);
      copying = false;
    }
  }

  function toggleFullscreen() {
    if (!chartWrapper) return;
    
    if (!isFullscreen) {
      if (chartWrapper.requestFullscreen) {
        chartWrapper.requestFullscreen();
      } else if (chartWrapper.webkitRequestFullscreen) {
        chartWrapper.webkitRequestFullscreen();
      } else if (chartWrapper.msRequestFullscreen) {
        chartWrapper.msRequestFullscreen();
      }
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen();
      } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
      } else if (document.msExitFullscreen) {
        document.msExitFullscreen();
      }
    }
  }

  function handleFullscreenChange() {
    isFullscreen = !!document.fullscreenElement;
    // 重置圖表大小以填滿全螢幕
    setTimeout(function() {
      if (chart && chartContainer) {
        chart.applyOptions({ 
          width: chartContainer.clientWidth,
          height: chartContainer.clientHeight 
        });
      }
    }, 100);
  }

  function handleKeydown(e) {
    if (e.key === 'Delete' || e.key === 'Backspace') {
      if (selectedLineIndex !== null && !drawingActive) {
        deleteSelectedLine();
      }
    }
  }

</script>

<div bind:this={chartWrapper} class="chart-wrapper" class:is-fullscreen={isFullscreen}>
  <div class="chart-info-overlay">
    <div class="tags-group">
      <span class="symbol-tag">{trade?.symbol || ''}</span>
      <span class="timeframe-tag">{timeframe}</span>
      <span class="timezone-tag">時區: UTC+8 (Local)</span>
    </div>

    <div class="tools-group">
      {#if selectedLineIndex !== null}
        <button class="tool-button clear-button" on:click={deleteSelectedLine} title="刪除選中線條">
          <span class="icon">🗑️</span>
        </button>
      {:else if drawnLines.length > 0}
        <button class="tool-button clear-button" on:click={clearDrawings} title="清除所有線條">
          <span class="icon">🗑️</span>
        </button>
      {/if}
      
      <button class="tool-button draw-button" class:active={drawingActive} on:click={toggleDrawing} title="趨勢線工具">
        <span class="icon">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="19" cy="5" r="3"/>
            <circle cx="5" cy="19" r="3"/>
            <line x1="7.1" y1="16.9" x2="16.9" y2="7.1"/>
          </svg>
        </span>
      </button>

      <button class="copy-button" on:click={copyChartImage} disabled={copying} title="複製圖表截圖">
        {#if copying}
          <span class="icon">✅</span>
        {:else}
          <span class="icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
              <circle cx="12" cy="13" r="4"/>
            </svg>
          </span>
        {/if}
      </button>

      <button class="tool-button fullscreen-button" on:click={toggleFullscreen} title={isFullscreen ? "退出全螢幕" : "全螢幕檢視"}>
        <span class="icon">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/>
          </svg>
        </span>
      </button>
    </div>
  </div>

  {#if loading}
    <div class="status-overlay">
      <div class="spinner"></div>
      <span>正在從 cTrader 獲取實時數據...</span>
    </div>
  {/if}
  
  {#if error}
    <div class="status-overlay error">
      <div class="error-box">
        <span class="icon">⚠️</span>
        <span class="msg">{error}</span>
      </div>
    </div>
  {/if}

  <div bind:this={chartContainer} class="chart-container"></div>
</div>

<style>
  .chart-wrapper {
    position: relative;
    width: 100%;
    height: 450px;
    background: #0f172a;
    border-radius: 16px;
    overflow: hidden;
    border: 1px solid rgba(255, 255, 255, 0.08);
    margin: 1rem 0;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
  }

  .chart-wrapper.is-fullscreen {
    position: fixed;
    inset: 0;
    width: 100vw;
    height: 100vh;
    margin: 0;
    border-radius: 0;
    z-index: 9999;
  }

  .chart-container {
    width: 100%;
    height: 100%;
  }

  .chart-info-overlay {
    position: absolute;
    top: 16px;
    left: 16px;
    right: 16px;
    z-index: 5;
    display: flex;
    justify-content: space-between;
    align-items: center;
    pointer-events: none;
  }

  .tags-group, .tools-group {
    display: flex;
    gap: 8px;
    pointer-events: none;
  }

  .copy-button, .tool-button {
    pointer-events: auto;
    background: rgba(30, 41, 59, 0.8);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    color: #cbd5e1;
    padding: 6px 14px;
    border-radius: 8px;
    font-size: 0.8rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  }

  .copy-button:hover {
    background: rgba(51, 65, 85, 0.95);
    border-color: rgba(59, 130, 246, 0.4);
    color: #fff;
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.3);
  }

  .copy-button:active, .tool-button:active {
    transform: translateY(0) scale(0.96);
  }

  .tool-button.active {
    background: rgba(59, 130, 246, 0.2);
    border-color: #3b82f6;
  }

  .tool-button.active .icon {
    color: #3b82f6;
  }

  .tool-button.clear-button {
    padding: 6px 10px;
    color: #fca5a5;
  }
  
  .tool-button.clear-button:hover {
    color: #ef4444;
    border-color: rgba(239, 68, 68, 0.4);
  }

  .symbol-tag, .timeframe-tag, .timezone-tag {
    background: rgba(30, 41, 59, 0.7);
    backdrop-filter: blur(4px);
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 0.75rem;
    color: #cbd5e1;
    border: 1px solid rgba(255, 255, 255, 0.1);
    font-family: 'JetBrains Mono', monospace;
  }

  .symbol-tag {
    color: #3b82f6;
    font-weight: 600;
  }

  .timeframe-tag {
    color: #f59e0b; /* Amber/Orange color */
    font-weight: 600;
  }

  .status-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: rgba(15, 23, 42, 0.9);
    z-index: 10;
    color: #94a3b8;
    gap: 16px;
    backdrop-filter: blur(4px);
  }

  .status-overlay.error {
    background: rgba(15, 23, 42, 0.8);
  }

  .error-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: 80%;
    text-align: center;
    gap: 8px;
  }

  .error-box .icon {
    font-size: 2rem;
  }

  .error-box .msg {
    color: #fca5a5;
    font-size: 0.95rem;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid rgba(59, 130, 246, 0.1);
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 0.8s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  :global(.tv-lightweight-charts) {
    border-radius: 16px;
  }
</style>
