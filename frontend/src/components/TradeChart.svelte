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
  let drawnLines = []; // Array of { series, p1: {time, price}, p2: {time, price}, color, lineWidth }
  let isFullscreen = false;
  let chartWrapper;
  
  // Interactive drawing states
  let previewLine = null;
  let draggingPoint = null; // { lineIndex, pointIndex (0 or 1) }
  let selectedLineIndex = null;
  let rafId = null;

  // New interactive handles
  let cp1 = { x: -1000, y: -1000 };
  let cp2 = { x: -1000, y: -1000 };
  let isDraggingCP = false;
  let activeCPIndex = null; // 0 or 1
  let isDraggingLine = false;
  let dragStartPoint = null;

  // Drawing style options
  let selectedColor = '#f59e0b';
  let selectedLineWidth = 2;
  const colorOptions = ['#f59e0b', '#3b82f6', '#ef4444', '#10b981', '#8b5cf6', '#ec4899', '#FACC15', '#4ADE80', '#60A5FA'];
  const lineWidthOptions = [1, 2, 3, 4];
  let showStyleMenu = false;

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
      window.removeEventListener('mousemove', handleCPMouseMove);
      window.removeEventListener('mouseup', handleCPMouseUp);
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
      localization: {
        locale: 'zh-TW',
      },
      crosshair: {
        mode: 0,
      },
    });

    chart.timeScale().subscribeVisibleLogicalRangeChange(updateControlPoints);

    chart.subscribeClick(handleChartClick);
    chart.subscribeCrosshairMove(handleCrosshairMove);
    
    // 添加mousedown事件以支援按住拖曳
    chartContainer.addEventListener('mousedown', function(e) {
      const rect = chartContainer.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const time = chart.timeScale().coordinateToTime(x);
      const price = candlestickSeries.coordinateToPrice(y);
      if (time && price) {
        handleChartMouseDown({ point: { x, y }, time });
      }
    });

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
      const TZ_OFFSET = 8 * 3600; // UTC+8 偏移 (秒)

      for (let i = 0; i < data.trendbar.length; i++) {
        const bar = data.trendbar[i];
        chartData.push({
          time: bar.utcTimestampInMinutes * 60 + TZ_OFFSET,
          open: (bar.low + bar.deltaOpen) / scale,
          high: (bar.low + bar.deltaHigh) / scale,
          low: bar.low / scale,
          close: (bar.low + bar.deltaClose) / scale,
        });
      }

      // 設置價格精細度
      if (candlestickSeries) {
        candlestickSeries.applyOptions({
          priceFormat: {
            type: 'price',
            precision: digits,
            minMove: 1 / Math.pow(10, digits),
          },
        });
      }

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
      
      if (candlestickSeries) {
        candlestickSeries.setData(uniqueData);
      }

      // 設置標註 (Markers)
      const markers = [];
      if (trade) {
        const entryTs = Math.floor(new Date(trade.entry_time).getTime() / 1000) + TZ_OFFSET;
        const exitTs = trade.exit_time ? Math.floor(new Date(trade.exit_time).getTime() / 1000) + TZ_OFFSET : null;
        
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
      loadTrendlines();
    }
  }

  function handleChartClick(param) {
    if (!param || !param.point || !param.time) return;
    
    const price = candlestickSeries.coordinateToPrice(param.point.y);
    if (!price) return;

    // --- Dragging & Selection Logic (Outside drawing mode) ---
    if (!drawingActive) {
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
      
      console.log('[Chart] Line completed, exiting drawing mode');
      // Use the central toggle function to ensure all states are reset properly
      toggleDrawing(); 
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

      // 拖曳整條線（只在按住時才拖動）
      if (isDraggingLine && dragStartPoint && selectedLineIndex !== null) {
        const line = drawnLines[selectedLineIndex];
        const deltaTime = param.time - dragStartPoint.time;
        const deltaPrice = price - dragStartPoint.price;
        
        line.p1.time += deltaTime;
        line.p1.price += deltaPrice;
        line.p2.time += deltaTime;
        line.p2.price += deltaPrice;
        
        const lineData = [
          { time: line.p1.time, value: line.p1.price },
          { time: line.p2.time, value: line.p2.price }
        ];
        lineData.sort(function(a, b) { return a.time - b.time; });
        line.series.setData(lineData);
        
        dragStartPoint = { time: param.time, price: price, x: param.point.x, y: param.point.y };
        updateControlPoints();
        return;
      }

      // Drawing preview
      if (drawingActive && firstPoint && previewLine) {
        const lineData = [
          { time: firstPoint.time, value: firstPoint.price },
          { time: param.time, value: price }
        ];
        lineData.sort(function(a, b) { return a.time - b.time; });
        previewLine.setData(lineData);
      }
    });
  }

  function addTrendline(p1, p2) {
    const series = chart.addSeries(LineSeries, {
      color: selectedColor,
      lineWidth: selectedLineWidth,
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
    
    drawnLines.push({ 
      series: series, 
      p1: p1, 
      p2: p2,
      color: selectedColor,
      lineWidth: selectedLineWidth
    });
    drawnLines = drawnLines;
    selectedLineIndex = drawnLines.length - 1;
    updateSelectedStyles();
    saveTrendlines();
  }

  function updateSelectedStyles() {
    for (let i = 0; i < drawnLines.length; i++) {
      const line = drawnLines[i];
      line.series.applyOptions({
        color: line.color || '#f59e0b',
        lineWidth: i === selectedLineIndex ? (line.lineWidth || 2) + 2 : (line.lineWidth || 2),
      });
    }
    updateControlPoints();
  }

  function updateControlPoints() {
    if (selectedLineIndex === null || !drawnLines[selectedLineIndex] || !candlestickSeries) {
      cp1 = { x: -1000, y: -1000 };
      cp2 = { x: -1000, y: -1000 };
      return;
    }
    const line = drawnLines[selectedLineIndex];
    const x1 = chart.timeScale().timeToCoordinate(line.p1.time);
    const y1 = candlestickSeries.priceToCoordinate(line.p1.price);
    const x2 = chart.timeScale().timeToCoordinate(line.p2.time);
    const y2 = candlestickSeries.priceToCoordinate(line.p2.price);

    // 計算線條角度並將控制點放在線條外側
    if (x1 != null && y1 != null && x2 != null && y2 != null) {
      const dx = x2 - x1;
      const dy = y2 - y1;
      const length = Math.sqrt(dx * dx + dy * dy);
      const offsetDistance = 6; // 控制點偏移距離（減少以更靠近線條）
      
      if (length > 0) {
        const unitX = dx / length;
        const unitY = dy / length;
        
        cp1 = { 
          x: x1 - unitX * offsetDistance, 
          y: y1 - unitY * offsetDistance 
        };
        cp2 = { 
          x: x2 + unitX * offsetDistance, 
          y: y2 + unitY * offsetDistance 
        };
      } else {
        cp1 = { x: x1, y: y1 };
        cp2 = { x: x2, y: y2 };
      }
    } else {
      cp1 = { x: -1000, y: -1000 };
      cp2 = { x: -1000, y: -1000 };
    }
  }

  function startDragCP(e, index) {
    if (drawingActive) return;
    e.preventDefault();
    e.stopPropagation();
    isDraggingCP = true;
    activeCPIndex = index;
    chart.applyOptions({ handleScroll: false, handleScale: false });
    window.addEventListener('mousemove', handleCPMouseMove);
    window.addEventListener('mouseup', handleCPMouseUp);
  }

  function handleCPMouseMove(e) {
    if (!isDraggingCP || selectedLineIndex === null || !chartContainer) return;
    
    // 使用 requestAnimationFrame 優化效能
    if (rafId) cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(function() {
      const rect = chartContainer.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      
      const time = chart.timeScale().coordinateToTime(x);
      const price = candlestickSeries.coordinateToPrice(y);
      
      if (time && price) {
        const line = drawnLines[selectedLineIndex];
        if (activeCPIndex === 0) {
          line.p1 = { time, price };
        } else {
          line.p2 = { time, price };
        }
        
        const lineData = [
          { time: line.p1.time, value: line.p1.price },
          { time: line.p2.time, value: line.p2.price }
        ];
        lineData.sort(function(a, b) { return a.time - b.time; });
        line.series.setData(lineData);
        updateControlPoints();
      }
    });
  }

  function handleCPMouseUp() {
    isDraggingCP = false;
    activeCPIndex = null;
    chart.applyOptions({ handleScroll: true, handleScale: true });
    window.removeEventListener('mousemove', handleCPMouseMove);
    window.removeEventListener('mouseup', handleCPMouseUp);
  }

  function handleMouseDownCP0(e) { startDragCP(e, 0); }
  function handleMouseDownCP1(e) { startDragCP(e, 1); }

  function deleteSelectedLine() {
    if (selectedLineIndex === null) return;
    chart.removeSeries(drawnLines[selectedLineIndex].series);
    drawnLines.splice(selectedLineIndex, 1);
    drawnLines = drawnLines;
    selectedLineIndex = null;
    saveTrendlines();
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
    saveTrendlines();
  }

  async function saveTrendlines() {
    if (!tradeId || drawnLines.length === 0) return;
    try {
      const linesData = drawnLines.map(line => ({
        p1: { time: line.p1.time, price: line.p1.price },
        p2: { time: line.p2.time, price: line.p2.price },
        color: line.color,
        lineWidth: line.lineWidth
      }));
      await tradesAPI.saveTrendlines(tradeId, linesData);
    } catch (e) {
      console.error('[Chart] Failed to save trendlines:', e);
    }
  }

  async function loadTrendlines() {
    if (!tradeId || !chart || !candlestickSeries) return;
    try {
      const res = await tradesAPI.getTrendlines(tradeId);
      const linesData = res.data;
      if (!linesData || linesData.length === 0) return;
      
      for (const lineData of linesData) {
        const series = chart.addSeries(LineSeries, {
          color: lineData.color || '#f59e0b',
          lineWidth: lineData.lineWidth || 2,
          lastValueVisible: false,
          priceLineVisible: false,
          crosshairMarkerVisible: false,
        });
        
        const data = [
          { time: lineData.p1.time, value: lineData.p1.price },
          { time: lineData.p2.time, value: lineData.p2.price }
        ];
        data.sort((a, b) => a.time - b.time);
        series.setData(data);
        
        drawnLines.push({
          series: series,
          p1: lineData.p1,
          p2: lineData.p2,
          color: lineData.color,
          lineWidth: lineData.lineWidth
        });
      }
      drawnLines = drawnLines;
    } catch (e) {
      console.error('[Chart] Failed to load trendlines:', e);
    }
  }

  function handleChartMouseUp() {
    if (isDraggingLine) {
      isDraggingLine = false;
      dragStartPoint = null;
      chart.applyOptions({ handleScroll: true, handleScale: true });
      saveTrendlines();
    }
  }

  function handleChartMouseDown(param) {
    if (!param || !param.point || !param.time || drawingActive) return;
    
    const price = candlestickSeries.coordinateToPrice(param.point.y);
    if (!price) return;

    // 檢查是否點擊在已選中的線條上
    if (selectedLineIndex !== null) {
      const line = drawnLines[selectedLineIndex];
      const p1Y = candlestickSeries.priceToCoordinate(line.p1.price);
      const p1X = chart.timeScale().timeToCoordinate(line.p1.time);
      const p2Y = candlestickSeries.priceToCoordinate(line.p2.price);
      const p2X = chart.timeScale().timeToCoordinate(line.p2.time);

      // 計算點到線段的距離
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
        // 開始拖曳整條線
        isDraggingLine = true;
        dragStartPoint = { time: param.time, price: price, x: param.point.x, y: param.point.y };
        chart.applyOptions({ handleScroll: false, handleScale: false });
      }
    }
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
      
      <!-- 樣式選擇器 -->
      <div class="style-selector">
        <button class="tool-button style-button" on:click={() => showStyleMenu = !showStyleMenu} title="線條樣式" style="display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px;">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"></path></svg>
          <div style="width: 16px; height: 3px; background-color: {selectedColor}; border-radius: 2px;"></div>
        </button>
        {#if showStyleMenu}
          <div class="style-menu">
            <div class="style-section">
              <label>顏色</label>
              <div class="color-options">
                {#each colorOptions as color}
                  <button 
                    class="color-btn" 
                    class:active={selectedColor === color}
                    style="background: {color}"
                    on:click={() => { selectedColor = color; showStyleMenu = false; }}
                  ></button>
                {/each}
              </div>
            </div>
            <div class="style-section">
              <label>粗細</label>
              <div class="width-options">
                {#each lineWidthOptions as width}
                  <button 
                    class="width-btn" 
                    class:active={selectedLineWidth === width}
                    on:click={() => { selectedLineWidth = width; showStyleMenu = false; }}
                  >
                    {width}px
                  </button>
                {/each}
              </div>
            </div>
          </div>
        {/if}
      </div>
      
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

  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div bind:this={chartContainer} class="chart-container" on:mouseup={handleChartMouseUp}></div>

  {#if selectedLineIndex !== null}
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div 
      class="control-point-handle" 
      style="left: {cp1.x}px; top: {cp1.y}px;"
      on:mousedown={handleMouseDownCP0}
    ></div>
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div 
      class="control-point-handle" 
      style="left: {cp2.x}px; top: {cp2.y}px;"
      on:mousedown={handleMouseDownCP1}
    ></div>
  {/if}
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

  .control-point-handle {
    position: absolute;
    width: 14px;
    height: 14px;
    background: transparent;
    border: 3px solid #3b82f6;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    pointer-events: auto;
    z-index: 20;
    cursor: move;
    box-shadow: 0 0 4px rgba(0, 0, 0, 0.4);
    transition: transform 0.1s;
  }

  .control-point-handle:hover {
    transform: translate(-50%, -50%) scale(1.2);
    background: rgba(59, 130, 246, 0.1);
  }

  .style-selector {
    position: relative;
    pointer-events: auto;
  }

  .style-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 8px;
    background: rgba(30, 41, 59, 0.95);
    backdrop-filter: blur(12px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 12px;
    padding: 12px;
    min-width: 200px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    z-index: 100;
  }

  .style-section {
    margin-bottom: 12px;
  }

  .style-section:last-child {
    margin-bottom: 0;
  }

  .style-section label {
    display: block;
    color: #cbd5e1;
    font-size: 0.75rem;
    font-weight: 600;
    margin-bottom: 8px;
    text-transform: uppercase;
  }

  .color-options {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .color-btn {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .color-btn:hover {
    transform: scale(1.1);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  }

  .color-btn.active {
    border-color: #fff;
    box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.2);
  }

  .width-options {
    display: flex;
    gap: 6px;
  }

  .width-btn {
    flex: 1;
    padding: 6px 12px;
    background: rgba(51, 65, 85, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: #cbd5e1;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .width-btn:hover {
    background: rgba(51, 65, 85, 0.8);
    border-color: rgba(59, 130, 246, 0.4);
  }

  .width-btn.active {
    background: rgba(59, 130, 246, 0.3);
    border-color: #3b82f6;
    color: #fff;
  }
</style>
