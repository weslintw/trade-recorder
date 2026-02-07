<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    createChart,
    CandlestickSeries,
    LineSeries,
    createSeriesMarkers,
  } from 'lightweight-charts';
  import { tradesAPI, sharesAPI, dailyPlansAPI } from '../lib/api';
  import { TIMEFRAMES } from '../lib/constants';

  export let tradeId = null;
  export let trade = null;
  export let useSharedAPI = false;
  export let shareToken = null;

  // 每日規劃模式
  export let mode = 'trade'; // 'trade' or 'plan'
  export let accountId = null;
  export let symbol = '';
  export let planTimeframe = ''; // e.g. 'M5'
  export let initialConfig = null;
  export let initialTrendlines = [];
  export let onSaveConfig = null; // (config) => void
  export let onSaveTrendlines = null; // (lines) => void
  export let planDate = null;
  export let planSession = null;
  export let lazy = false;

  let chartContainer;
  let chart;
  let candlestickSeries;
  let lastKnownData = [];
  let loading = true;
  let error = null;
  let timeframe = '';
  let copying = false;
  let drawingMode = null; // null, 'trendline', 'arrow', 'fib', 'channel'
  let drawingActive = false;
  let firstPoint = null;
  let secondPoint = null; // For 3-point tools like channel
  let drawnLines = []; // Array of { series, p1: {time, price}, p2: {time, price}, p3: {time, price}, color, lineWidth }
  let isFullscreen = false;
  let chartWrapper;

  // Interactive drawing states
  let previewLine = null;
  let draggingPoint = null; // { lineIndex, pointIndex (0 or 1) }
  let selectedLineIndex = null;
  let rafId = null;

  const FIB_LEVELS = [
    { level: 0, color: '#f59e0b', label: '0' },
    { level: 0.382, color: '#d97706', label: '0.382' },
    { level: 0.5, color: '#22c55e', label: '0.5' },
    { level: 0.618, color: '#15803d', label: '0.618' },
    { level: 0.702, color: '#3b82f6', label: '0.702' },
    { level: 0.786, color: '#06b6d4', label: '0.786' },
    { level: 1, color: '#94a3b8', label: '1' },
  ];
  let fibPreviewLines = [];
  let fibLabels = []; // { x, y, text, color }
  let cp1 = { x: -1000, y: -1000 };
  let cp2 = { x: -1000, y: -1000 };
  let cp3 = { x: -1000, y: -1000 };
  let lastFibPreviewPoints = null; // Store preview points for label sync
  let isDraggingCP = false;
  let activeCPIndex = null; // 0 or 1
  let isDraggingLine = false;
  let dragStartPoint = null;

  // Drawing style options
  let selectedColor = '#f59e0b';
  let selectedLineWidth = 2;
  const colorOptions = [
    '#f59e0b',
    '#3b82f6',
    '#ef4444',
    '#10b981',
    '#8b5cf6',
    '#ec4899',
    '#FACC15',
    '#4ADE80',
    '#60A5FA',
  ];
  const lineWidthOptions = [1, 2, 3, 4];
  let showStyleMenu = false;

  let selectedPeriod = '';
  const periods = [
    { value: '', label: 'Auto' },
    { value: 'm1', label: '1m' },
    { value: 'm5', label: '5m' },
    { value: 'm15', label: '15m' },
    { value: 'm30', label: '30m' },
    { value: 'h1', label: '1h' },
    { value: 'h4', label: '4h' },
    { value: 'd1', label: 'D1' },
  ];

  let configApplied = false;
  let saveTimer = null;

  $: if (tradeId || (mode === 'plan' && (accountId || symbol))) {
    configApplied = false; // Reset when trade changes
    if (chart) loadData();
  }

  let lastAppliedLines = null;
  $: if (mode === 'plan' && initialTrendlines !== lastAppliedLines && chart && candlestickSeries) {
    clearAllDrawnLines();
    if (initialTrendlines) applyLinesData(initialTrendlines);
    lastAppliedLines = initialTrendlines;
  }

  let lastAppliedConfig = null;
  $: if (mode === 'plan' && initialConfig !== lastAppliedConfig && chart) {
    applyChartConfig(initialConfig);
    lastAppliedConfig = initialConfig;
  }

  function debounceSaveConfig() {
    if (useSharedAPI || !tradeId) return;
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(saveChartConfig, 2000);
  }

  async function saveChartConfig() {
    if (!chart || useSharedAPI) return;

    // 如果是交易模式，且沒有 tradeId，則中止
    if (mode === 'trade' && !tradeId) return;

    const timeScale = chart.timeScale();
    const priceScale = chart.priceScale('right');

    const range = timeScale.getVisibleLogicalRange();
    const priceRange = priceScale.getVisibleRange();
    const priceScaleOpts = priceScale.options();

    const config = {
      period: selectedPeriod,
      range: range,
      priceRange: priceRange,
      autoScale: priceScaleOpts.autoScale,
    };

    if (mode === 'plan' && onSaveConfig) {
      onSaveConfig(config);
      return;
    }

    try {
      await tradesAPI.saveChartConfig(tradeId, { chart_config: JSON.stringify(config) });
      console.log('[Chart] Config saved:', config);
    } catch (e) {
      console.error('[Chart] Failed to save config:', e);
    }
  }

  onMount(function () {
    if (mode === 'trade' && trade && trade.chart_config) {
      try {
        const config = JSON.parse(trade.chart_config);
        if (config.period) {
          // 檢查載入的週期是否還在選單中 (處理已移除的 10T, 20T)
          let found = false;
          for (let i = 0; i < periods.length; i++) {
            if (periods[i].value === config.period) {
              found = true;
              break;
            }
          }
          if (found) {
            selectedPeriod = config.period;
          } else {
            selectedPeriod = ''; // 不存在則回退到 Auto
          }
        }
      } catch (e) {}
    } else if (mode === 'plan') {
      // 設定預設時區
      if (planTimeframe) {
        selectedPeriod = planTimeframe.toLowerCase();
      }
      if (initialConfig) {
        selectedPeriod = initialConfig.period || selectedPeriod;
      }
    }

    initChart();

    if (lazy && 'IntersectionObserver' in window) {
      const observer = new IntersectionObserver(
        entries => {
          if (entries[0].isIntersecting) {
            loadData();
            observer.disconnect();
          }
        },
        { threshold: 0.1 }
      );
      observer.observe(chartContainer);
    } else {
      loadData();
    }

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);
    window.addEventListener('keydown', handleKeydown);
    return function () {
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
          top: 0.2,
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

    chart.timeScale().subscribeVisibleLogicalRangeChange(function () {
      updateControlPoints();
      debounceSaveConfig();
    });
    chart.timeScale().subscribeVisibleTimeRangeChange(function () {
      // Update arrows/fib on zoom/scroll to maintain arrowhead shape and size
      for (let i = 0; i < drawnLines.length; i++) {
        const line = drawnLines[i];
        if (line.type === 'arrow') updateLineWings(line);
        if (line.type === 'fib') updateFibLevels(line);
      }
      // Update labels positioning
      updateControlPoints();
    });

    chart.subscribeClick(handleChartClick);
    chart.subscribeCrosshairMove(handleCrosshairMove);

    // 添加mousedown事件以支援按住拖曳
    chartContainer.addEventListener('mousedown', function (e) {
      const rect = chartContainer.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const t = chart.timeScale().coordinateToTime(x);
      const point = getEffectivePoint({ point: { x, y }, time: t });
      if (point) {
        handleChartMouseDown({ point: { x, y }, time: point.time });
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

    return function () {
      window.removeEventListener('resize', handleResize);
    };
  }

  async function loadData() {
    if (mode === 'trade' && !tradeId) return;
    if (mode === 'plan' && (!accountId || !symbol)) return;

    try {
      loading = true;
      error = null;

      let res;
      if (mode === 'plan') {
        res = await dailyPlansAPI.getChartData({
          account_id: accountId,
          symbol: symbol,
          period: selectedPeriod,
          date: planDate,
          session: planSession,
        });
      } else if (useSharedAPI && shareToken) {
        res = await sharesAPI.getChartData(shareToken, tradeId, selectedPeriod);
      } else {
        res = await tradesAPI.getChartData(tradeId, selectedPeriod);
      }

      const resData = res.data;
      const data = resData.data;
      const digits = resData.digits || 2;
      timeframe = resData.timeframe || '';

      if (!data || !data.trendbar || data.trendbar.length === 0) {
        error = '無法獲取 K 線數據 (數據可能已過期或伺服器暫時無法連接)';
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
      chartData.sort(function (a, b) {
        return a.time - b.time;
      });
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
        lastKnownData = uniqueData;
      }

      // 設置標註 (Markers)
      const markers = [];
      if (trade) {
        const entryTs = Math.floor(new Date(trade.entry_time).getTime() / 1000) + TZ_OFFSET;
        const exitTs = trade.exit_time
          ? Math.floor(new Date(trade.exit_time).getTime() / 1000) + TZ_OFFSET
          : null;

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

      markers.sort(function (a, b) {
        return a.time - b.time;
      });
      createSeriesMarkers(candlestickSeries, markers);

      // 視野管理：智慧聚焦交易區間
      if (trade && uniqueData.length > 0) {
        const entryTs = Math.floor(new Date(trade.entry_time).getTime() / 1000) + TZ_OFFSET;

        // 尋找進場點的索引
        let entryIdx = -1;
        // 先嘗試精確匹配 (誤差 5 分鐘內)
        for (let i = 0; i < uniqueData.length; i++) {
          if (Math.abs(uniqueData[i].time - entryTs) < 300) {
            entryIdx = i;
            break;
          }
        }

        // 如果找不到，找最接近的
        if (entryIdx === -1) {
          let minDiff = Infinity;
          for (let i = 0; i < uniqueData.length; i++) {
            const diff = Math.abs(uniqueData[i].time - entryTs);
            if (diff < minDiff) {
              minDiff = diff;
              entryIdx = i;
            }
          }
        }

        // 設定視野終點：如果有出場，則以出場為準；如果沒有，以最新數據為準
        let exitIdx = uniqueData.length - 1;
        if (trade.exit_time) {
          const exitTs = Math.floor(new Date(trade.exit_time).getTime() / 1000) + TZ_OFFSET;
          let foundExit = -1;
          for (let i = 0; i < uniqueData.length; i++) {
            if (Math.abs(uniqueData[i].time - exitTs) < 300) {
              foundExit = i;
              break;
            }
          }
          if (foundExit !== -1) {
            exitIdx = foundExit;
          } else {
            // 同樣找最接近的
            let minDiff = Infinity;
            for (let i = 0; i < uniqueData.length; i++) {
              const diff = Math.abs(uniqueData[i].time - exitTs);
              if (diff < minDiff) {
                minDiff = diff;
                exitIdx = i;
              }
            }
          }
        }

        // Apply chart config range if available
        let configToApply = null;
        if (mode === 'trade' && trade && trade.chart_config) {
          try {
            configToApply = JSON.parse(trade.chart_config);
          } catch (e) {}
        } else if (mode === 'plan' && initialConfig) {
          configToApply = initialConfig;
        }

        if (configToApply && !configApplied) {
          try {
            const config = configToApply;
            let applied = false;

            if (config.range) {
              console.log('[Chart] Applying saved time range:', config.range);
              chart.timeScale().setVisibleLogicalRange(config.range);
              applied = true;
            }

            if (config.priceRange) {
              console.log('[Chart] Applying saved price range:', config.priceRange);
              const priceScale = chart.priceScale('right');

              if (config.autoScale === false) {
                priceScale.applyOptions({ autoScale: false });
                priceScale.setVisibleRange(config.priceRange);
              } else {
                priceScale.applyOptions({ autoScale: true });
              }
              applied = true;
            }

            if (applied) {
              configApplied = true;
            } else {
              applyDefaultFocus();
            }
          } catch (e) {
            console.error('[Chart] Failed to apply config range:', e);
            applyDefaultFocus();
          }
        } else if (!configApplied) {
          applyDefaultFocus();
        }

        function applyDefaultFocus() {
          // 計算顯示範圍
          // 右側留白：如果是進行中交易留多一點(30)，歷史交易留少一點(15)
          const rightOffset = 80;

          // 最小顯示根數，確保視野不會縮太小
          const minVisibleBars = 400;

          let to = exitIdx + rightOffset;
          let from = entryIdx - 30; // 左側預設留白 30 根

          // 如果區間太小，向左擴展以滿足最小顯示根數
          if (to - from < minVisibleBars) {
            from = to - minVisibleBars;
          }

          chart.timeScale().setVisibleLogicalRange({
            from: from,
            to: to,
          });
        }
      } else {
        // 無交易數據時的 Fallback：顯示最後 100 根
        const totalLen = uniqueData.length;
        chart.timeScale().setVisibleLogicalRange({
          from: totalLen - 400,
          to: totalLen + 50,
        });
      }
    } catch (e) {
      console.error('[Chart Error]', e);
      let errorMsg = e.message;
      if (e.response && e.response.data && e.response.data.error) {
        errorMsg = e.response.data.error;
      }
      error = '載入失敗: ' + errorMsg;
    } finally {
      loading = false;
      loadTrendlines();
    }
  }

  function handleChartClick(param) {
    const point = getEffectivePoint(param);
    if (!point) return;
    const { time, price } = point;

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
        if (param_t < 0) {
          xx = p1X;
          yy = p1Y;
        } else if (param_t > 1) {
          xx = p2X;
          yy = p2Y;
        } else {
          xx = p1X + param_t * C;
          yy = p1Y + param_t * D;
        }
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
      firstPoint = { time, price };
      // Create preview line
      previewLine = chart.addSeries(LineSeries, {
        color: drawingMode === 'fib' ? 'rgba(0, 0, 0, 0)' : 'rgba(245, 158, 11, 0.5)',
        lineWidth: 1,
        lineStyle: 2, // Dashed
        lastValueVisible: false,
        priceLineVisible: false,
        crosshairMarkerVisible: false,
      });
    } else if (drawingMode === 'channel' && !secondPoint) {
      secondPoint = { time, price };
      // Establishing baseline, previewLine remains between p1-p2
    } else {
      const currentPoint = { time, price };
      if (drawingMode === 'arrow') {
        addArrow(firstPoint, currentPoint);
      } else if (drawingMode === 'fib') {
        addFib(firstPoint, currentPoint);
      } else if (drawingMode === 'channel') {
        addChannel(firstPoint, secondPoint, currentPoint);
      } else {
        addTrendline(firstPoint, currentPoint);
      }

      console.log('[Chart] Drawing completed');
      toggleDrawing(drawingMode);
    }
  }

  function handleCrosshairMove(param) {
    if (rafId) cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(function () {
      const point = getEffectivePoint(param);
      if (!point) return;
      const { time, price } = point;

      // 拖曳整條線（只在按住時才拖動）
      if (isDraggingLine && dragStartPoint && selectedLineIndex !== null) {
        const line = drawnLines[selectedLineIndex];
        const deltaTime = time - dragStartPoint.time;
        const deltaPrice = price - dragStartPoint.price;

        line.p1.time += deltaTime;
        line.p1.price += deltaPrice;
        line.p2.time += deltaTime;
        line.p2.price += deltaPrice;

        if (line.p3) {
          line.p3.time += deltaTime;
          line.p3.price += deltaPrice;
        }

        const lineData = [
          { time: line.p1.time, value: line.p1.price },
          { time: line.p2.time, value: line.p2.price },
        ];
        lineData.sort(function (a, b) {
          return a.time - b.time;
        });
        line.series.setData(lineData);
        if (line.type === 'arrow') updateLineWings(line);
        if (line.type === 'fib') updateFibLevels(line);
        if (line.type === 'channel') updateChannelLines(line);

        dragStartPoint = { time, price, x: param.point.x, y: param.point.y };
        updateControlPoints();
        return;
      }

      // Drawing preview
      if (drawingActive && firstPoint && previewLine) {
        if (drawingMode === 'channel' && secondPoint) {
          // Already have baseline, p3 is at cursor
          updateChannelPreview(firstPoint, secondPoint, { time, price });
          hideFibPreview();
          hideArrowPreviewWings();
        } else {
          const lineData = [
            { time: firstPoint.time, value: firstPoint.price },
            { time, value: price },
          ];
          lineData.sort(function (a, b) {
            return a.time - b.time;
          });
          previewLine.setData(lineData);

          // Update arrow/fib preview
          if (drawingMode === 'arrow') {
            updateArrowPreviewWings(firstPoint, { time, price });
            hideFibPreview();
            hideChannelPreview();
          } else if (drawingMode === 'fib') {
            updateFibPreview(firstPoint, { time, price });
            hideArrowPreviewWings();
            hideChannelPreview();
          } else if (drawingMode === 'channel') {
            hideArrowPreviewWings();
            hideFibPreview();
            hideChannelPreview();
          } else {
            hideArrowPreviewWings();
            hideFibPreview();
            hideChannelPreview();
          }
        }
      }
    });
  }

  let arrowPreviewWings = [];

  function updateArrowPreviewWings(p1, p2) {
    if (!candlestickSeries || !chart) return;

    // Calculate wings in screen coordinates
    const x1 = chart.timeScale().timeToCoordinate(p1.time);
    const y1 = candlestickSeries.priceToCoordinate(p1.price);
    const x2 = chart.timeScale().timeToCoordinate(p2.time);
    const y2 = candlestickSeries.priceToCoordinate(p2.price);

    if (x1 === null || y1 === null || x2 === null || y2 === null) return;

    const dx = x2 - x1;
    const dy = y2 - y1;
    const angle = Math.atan2(dy, dx);
    const length = 15; // Wing length in pixels
    const wingAngle = Math.PI / 7; // About 25 degrees

    const x3 = x2 - length * Math.cos(angle - wingAngle);
    const y3 = y2 - length * Math.sin(angle - wingAngle);
    const x4 = x2 - length * Math.cos(angle + wingAngle);
    const y4 = y2 - length * Math.sin(angle + wingAngle);

    const t3 = chart.timeScale().coordinateToTime(x3);
    const v3 = candlestickSeries.coordinateToPrice(y3);
    const t4 = chart.timeScale().coordinateToTime(x4);
    const v4 = candlestickSeries.coordinateToPrice(y4);

    if (t3 && v3 && t4 && v4) {
      if (arrowPreviewWings.length === 0) {
        const wingOptions = {
          color: selectedColor,
          lineWidth: selectedLineWidth,
          lineStyle: 2, // Dashed
          lastValueVisible: false,
          priceLineVisible: false,
          crosshairMarkerVisible: false,
        };
        arrowPreviewWings = [
          chart.addSeries(LineSeries, wingOptions),
          chart.addSeries(LineSeries, wingOptions),
        ];
      } else {
        // Update styles if they changed
        arrowPreviewWings[0].applyOptions({ color: selectedColor, lineWidth: selectedLineWidth });
        arrowPreviewWings[1].applyOptions({ color: selectedColor, lineWidth: selectedLineWidth });
      }

      const d1 = [
        { time: p2.time, value: p2.price },
        { time: t3, value: v3 },
      ];
      const d2 = [
        { time: p2.time, value: p2.price },
        { time: t4, value: v4 },
      ];
      d1.sort(function (a, b) {
        return a.time - b.time;
      });
      d2.sort(function (a, b) {
        return a.time - b.time;
      });
      arrowPreviewWings[0].setData(d1);
      arrowPreviewWings[1].setData(d2);
    }
  }

  function hideArrowPreviewWings() {
    if (arrowPreviewWings.length > 0) {
      chart.removeSeries(arrowPreviewWings[0]);
      chart.removeSeries(arrowPreviewWings[1]);
      arrowPreviewWings = [];
    }
  }

  let channelPreviewWings = [];
  function updateChannelPreview(p1, p2, p3) {
    if (!p1 || !p2 || !p3 || !chart) return;

    if (channelPreviewWings.length === 0) {
      const options = {
        color: selectedColor,
        lineWidth: selectedLineWidth,
        lineStyle: 2, // Dashed
        lastValueVisible: false,
        priceLineVisible: false,
        crosshairMarkerVisible: false,
      };
      channelPreviewWings = [chart.addSeries(LineSeries, options)];
    } else {
      channelPreviewWings[0].applyOptions({ color: selectedColor, lineWidth: selectedLineWidth });
    }

    const dt = p2.time - p1.time;
    if (dt === 0) return;

    const m = (p2.price - p1.price) / dt;
    const offset = p3.price - (m * (p3.time - p1.time) + p1.price);

    const tMin = Math.min(p1.time, p2.time);
    const tMax = Math.max(p1.time, p2.time);

    channelPreviewWings[0].setData([
      { time: tMin, value: m * (tMin - p1.time) + p1.price + offset },
      { time: tMax, value: m * (tMax - p1.time) + p1.price + offset },
    ]);
  }

  function hideChannelPreview() {
    if (channelPreviewWings.length > 0) {
      for (let i = 0; i < channelPreviewWings.length; i++) {
        chart.removeSeries(channelPreviewWings[i]);
      }
      channelPreviewWings = [];
    }
  }

  function updateFibPreview(p1, p2) {
    if (!candlestickSeries || !chart) return;
    const diff = p1.price - p2.price;

    if (fibPreviewLines.length === 0) {
      for (let i = 0; i < FIB_LEVELS.length; i++) {
        const level = FIB_LEVELS[i];
        const series = chart.addSeries(LineSeries, {
          color: level.color,
          lineWidth: 1,
          lineStyle: 2, // Dashed
          lastValueVisible: false,
          priceLineVisible: false,
          crosshairMarkerVisible: false,
        });
        fibPreviewLines.push(series);
      }
    }

    const tMin = Math.min(p1.time, p2.time);
    const tMax = Math.max(p1.time, p2.time);

    for (let i = 0; i < FIB_LEVELS.length; i++) {
      const level = FIB_LEVELS[i];
      const price = p2.price + diff * level.level;
      const data = [
        { time: tMin, value: price },
        { time: tMax, value: price },
      ];
      fibPreviewLines[i].setData(data);
    }

    lastFibPreviewPoints = { p1, p2 };
    syncFibLabels();
  }

  function hideFibPreview() {
    for (let i = 0; i < fibPreviewLines.length; i++) {
      fibPreviewLines[i].setData([]);
    }
    lastFibPreviewPoints = null;
    syncFibLabels();
  }

  function syncFibLabels() {
    if (!chart || !candlestickSeries || !chartContainer) {
      fibLabels = [];
      return;
    }

    const containerWidth = chartContainer.clientWidth;
    const labels = [];

    function addLineLabels(p1, p2) {
      const diff = p1.price - p2.price;
      const tMax = Math.max(p1.time, p2.time);
      const xMax = chart.timeScale().timeToCoordinate(tMax);

      let xPos = xMax !== null ? xMax + 5 : containerWidth - 50;
      if (xPos > containerWidth - 50) xPos = containerWidth - 50;
      if (xPos < 5) xPos = 5;

      for (let i = 0; i < FIB_LEVELS.length; i++) {
        const level = FIB_LEVELS[i];
        const price = p2.price + diff * level.level;
        const y = candlestickSeries.priceToCoordinate(price);
        if (y !== null) {
          labels.push({ x: xPos, y: y - 10, text: level.label, color: level.color });
        }
      }
    }

    // 1. Existing drawings
    for (let i = 0; i < drawnLines.length; i++) {
      const line = drawnLines[i];
      if (line.type === 'fib') addLineLabels(line.p1, line.p2);
    }

    // 2. Preview
    if (drawingActive && drawingMode === 'fib' && lastFibPreviewPoints) {
      addLineLabels(lastFibPreviewPoints.p1, lastFibPreviewPoints.p2);
    }

    fibLabels = labels;
  }

  function addFib(p1, p2) {
    const lineObj = addLineObject(p1, p2, 'fib');
    updateFibLevels(lineObj);
    saveTrendlines();
    syncFibLabels();
  }

  function addArrow(p1, p2) {
    const lineObj = addLineObject(p1, p2, 'arrow');
    updateLineWings(lineObj);
    saveTrendlines();
  }

  function addTrendline(p1, p2) {
    addLineObject(p1, p2, 'trendline');
    saveTrendlines();
  }

  function addChannel(p1, p2, p3) {
    const lineObj = addLineObject(p1, p2, 'channel');
    lineObj.p3 = p3;
    updateChannelLines(lineObj);
    saveTrendlines();
  }

  function addLineObject(p1, p2, type = 'trendline') {
    const series = chart.addSeries(LineSeries, {
      color: type === 'fib' ? 'rgba(0, 0, 0, 0)' : selectedColor,
      lineWidth: type === 'fib' ? 1 : selectedLineWidth,
      lineStyle: type === 'fib' || type === 'channel' ? 2 : 0,
      lastValueVisible: false,
      priceLineVisible: false,
      crosshairMarkerVisible: false,
    });

    const lineData = [
      { time: p1.time, value: p1.price },
      { time: p2.time, value: p2.price },
    ];
    lineData.sort(function (a, b) {
      return a.time - b.time;
    });
    series.setData(lineData);

    const lineObj = {
      series: series,
      p1: p1,
      p2: p2,
      p3: null, // For channel
      type: type,
      color: selectedColor,
      lineWidth: selectedLineWidth,
      wings: [], // For arrow, fib, channel type
    };

    drawnLines.push(lineObj);
    drawnLines = drawnLines;
    selectedLineIndex = drawnLines.length - 1;
    updateSelectedStyles();
    return lineObj;
  }

  function updateFibLevels(line) {
    if (line.type !== 'fib') return;
    const diff = line.p1.price - line.p2.price;
    const tMin = Math.min(line.p1.time, line.p2.time);
    const tMax = Math.max(line.p1.time, line.p2.time);

    if (line.wings.length === 0) {
      for (let i = 0; i < FIB_LEVELS.length; i++) {
        const level = FIB_LEVELS[i];
        const series = chart.addSeries(LineSeries, {
          color: level.color,
          lineWidth: 1,
          lineStyle: 2, // Dashed
          lastValueVisible: false,
          priceLineVisible: false,
          crosshairMarkerVisible: false,
        });
        line.wings.push(series);
      }
    }

    for (let i = 0; i < FIB_LEVELS.length; i++) {
      const level = FIB_LEVELS[i];
      const price = line.p2.price + diff * level.level;
      const data = [
        { time: tMin, value: price },
        { time: tMax, value: price },
      ];
      data.sort(function (a, b) {
        return a.time - b.time;
      });
      line.wings[i].setData(data);
    }
  }

  function updateLineWings(line) {
    if (line.type !== 'arrow') return;

    // Calculate wings
    const x1 = chart.timeScale().timeToCoordinate(line.p1.time);
    const y1 = candlestickSeries.priceToCoordinate(line.p1.price);
    const x2 = chart.timeScale().timeToCoordinate(line.p2.time);
    const y2 = candlestickSeries.priceToCoordinate(line.p2.price);

    if (x1 === null || y1 === null || x2 === null || y2 === null) return;

    const dx = x2 - x1;
    const dy = y2 - y1;
    const angle = Math.atan2(dy, dx);
    const length = 15;
    const wingAngle = Math.PI / 7;

    const x3 = x2 - length * Math.cos(angle - wingAngle);
    const y3 = y2 - length * Math.sin(angle - wingAngle);
    const x4 = x2 - length * Math.cos(angle + wingAngle);
    const y4 = y2 - length * Math.sin(angle + wingAngle);

    const t3 = chart.timeScale().coordinateToTime(x3);
    const v3 = candlestickSeries.coordinateToPrice(y3);
    const t4 = chart.timeScale().coordinateToTime(x4);
    const v4 = candlestickSeries.coordinateToPrice(y4);

    if (t3 && v3 && t4 && v4) {
      if (line.wings.length === 0) {
        line.wings = [
          chart.addSeries(LineSeries, {
            color: line.color,
            lineWidth: line.lineWidth,
            lastValueVisible: false,
            priceLineVisible: false,
            crosshairMarkerVisible: false,
          }),
          chart.addSeries(LineSeries, {
            color: line.color,
            lineWidth: line.lineWidth,
            lastValueVisible: false,
            priceLineVisible: false,
            crosshairMarkerVisible: false,
          }),
        ];
      }

      const d1 = [
        { time: line.p2.time, value: line.p2.price },
        { time: t3, value: v3 },
      ];
      const d2 = [
        { time: line.p2.time, value: line.p2.price },
        { time: t4, value: v4 },
      ];
      d1.sort(function (a, b) {
        return a.time - b.time;
      });
      d2.sort(function (a, b) {
        return a.time - b.time;
      });
      line.wings[0].setData(d1);
      line.wings[1].setData(d2);

      // Sync wing styles
      const wingOptions = {
        color: line.color,
        lineWidth: line.lineWidth,
      };
      line.wings[0].applyOptions(wingOptions);
      line.wings[1].applyOptions(wingOptions);
    }
  }

  function updateChannelLines(line) {
    if (line.type !== 'channel' || !line.p3 || !chart) return;

    if (line.wings.length === 0) {
      line.wings = [
        chart.addSeries(LineSeries, {
          color: line.color,
          lineWidth: line.lineWidth,
          lastValueVisible: false,
          priceLineVisible: false,
          crosshairMarkerVisible: false,
          lineStyle: 2,
        }),
      ];
    }

    const p1 = line.p1;
    const p2 = line.p2;
    const p3 = line.p3;

    const dt = p2.time - p1.time;
    if (dt === 0) return;

    const m = (p2.price - p1.price) / dt;
    const offset = p3.price - (m * (p3.time - p1.time) + p1.price);

    const tMin = Math.min(p1.time, p2.time);
    const tMax = Math.max(p1.time, p2.time);

    const parallelData = [
      { time: tMin, value: m * (tMin - p1.time) + p1.price + offset },
      { time: tMax, value: m * (tMax - p1.time) + p1.price + offset },
    ];

    line.wings[0].setData(parallelData);

    line.wings[0].applyOptions({ color: line.color, lineWidth: line.lineWidth });
  }

  function updateActiveLineStyle() {
    if (selectedLineIndex === null || !drawnLines[selectedLineIndex]) return;
    const line = drawnLines[selectedLineIndex];
    line.color = selectedColor;
    line.lineWidth = selectedLineWidth;
    if (line.type === 'arrow') updateLineWings(line);
    if (line.type === 'fib') updateFibLevels(line);
    if (line.type === 'channel') updateChannelLines(line);
    updateSelectedStyles();
    saveTrendlines();
  }

  function updateSelectedStyles() {
    for (let i = 0; i < drawnLines.length; i++) {
      const line = drawnLines[i];
      const isSelected = i === selectedLineIndex;
      const effectiveLineWidth = line.lineWidth || 2;

      line.series.applyOptions({
        color: line.type === 'fib' ? 'rgba(0, 0, 0, 0)' : line.color || '#f59e0b',
        lineWidth: effectiveLineWidth,
      });

      if (line.type === 'arrow' && line.wings && line.wings.length === 2) {
        line.wings[0].applyOptions({ color: line.color, lineWidth: effectiveLineWidth });
        line.wings[1].applyOptions({ color: line.color, lineWidth: effectiveLineWidth });
      }
      if (line.type === 'fib' && line.wings) {
        line.wings.forEach(function (s) {
          s.applyOptions({ lineWidth: 1 });
        });
      }
      if (line.type === 'channel' && line.wings && line.wings.length >= 1) {
        line.wings[0].applyOptions({ color: line.color, lineWidth: effectiveLineWidth });
      }
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

    let x3 = null,
      y3 = null;
    if (line.type === 'channel' && line.p3) {
      x3 = chart.timeScale().timeToCoordinate(line.p3.time);
      y3 = candlestickSeries.priceToCoordinate(line.p3.price);
    }

    // 計算線條角度並將控制點放在線條外側
    if (x1 != null && y1 != null && x2 != null && y2 != null) {
      const dx = x2 - x1;
      const dy = y2 - y1;
      const length = Math.sqrt(dx * dx + dy * dy);
      const offsetDistance = 6;

      if (length > 0) {
        const unitX = dx / length;
        const unitY = dy / length;

        cp1 = {
          x: x1 - unitX * offsetDistance,
          y: y1 - unitY * offsetDistance,
        };
        cp2 = {
          x: x2 + unitX * offsetDistance,
          y: y2 + unitY * offsetDistance,
        };
      } else {
        cp1 = { x: x1, y: y1 };
        cp2 = { x: x2, y: y2 };
      }

      if (x3 != null && y3 != null) {
        cp3 = { x: x3, y: y3 };
      } else {
        cp3 = { x: -1000, y: -1000 };
      }
    } else {
      cp1 = { x: -1000, y: -1000 };
      cp2 = { x: -1000, y: -1000 };
      cp3 = { x: -1000, y: -1000 };
    }

    syncFibLabels();
  }

  function startDragCP(e, index) {
    if (drawingActive || useSharedAPI) return; // Disable for shared mode
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
    rafId = requestAnimationFrame(function () {
      const rect = chartContainer.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;

      const point = getEffectivePoint({ point: { x, y } });
      if (point) {
        const { time, price } = point;
        const line = drawnLines[selectedLineIndex];
        if (activeCPIndex === 0) {
          line.p1 = { time, price };
        } else if (activeCPIndex === 1) {
          line.p2 = { time, price };
        } else if (activeCPIndex === 2) {
          line.p3 = { time, price };
        }

        const lineData = [
          { time: line.p1.time, value: line.p1.price },
          { time: line.p2.time, value: line.p2.price },
        ];
        lineData.sort(function (a, b) {
          return a.time - b.time;
        });
        line.series.setData(lineData);
        if (line.type === 'arrow') updateLineWings(line);
        if (line.type === 'fib') updateFibLevels(line);
        if (line.type === 'channel') updateChannelLines(line);
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
    if (!useSharedAPI) {
      // Only save if not in shared mode
      saveTrendlines();
    }
  }

  function handleMouseDownCP0(e) {
    startDragCP(e, 0);
  }
  function handleMouseDownCP1(e) {
    startDragCP(e, 1);
  }
  function handleMouseDownCP2(e) {
    startDragCP(e, 2);
  }

  function deleteSelectedLine() {
    if (selectedLineIndex === null || useSharedAPI) return; // Disable for shared mode
    const line = drawnLines[selectedLineIndex];
    chart.removeSeries(line.series);
    if (line.wings) {
      for (let i = 0; i < line.wings.length; i++) {
        chart.removeSeries(line.wings[i]);
      }
    }
    drawnLines.splice(selectedLineIndex, 1);
    drawnLines = drawnLines;
    selectedLineIndex = null;
    saveTrendlines();
  }

  function toggleDrawing(mode = 'trendline') {
    if (useSharedAPI) return; // Disable for shared mode

    if (drawingActive && drawingMode === mode) {
      // Toggle off if same mode
      drawingActive = false;
      drawingMode = null;
    } else if (drawingActive && drawingMode !== mode) {
      // Switch mode
      drawingMode = mode;
      // Reset firstPoint if switching tools? Probably safer.
      if (previewLine) {
        chart.removeSeries(previewLine);
        previewLine = null;
      }
      firstPoint = null;
    } else {
      // Toggle on
      drawingActive = true;
      drawingMode = mode;
      selectedLineIndex = null;
      updateSelectedStyles();
    }

    if (!drawingActive) {
      hideFibPreview();
      hideArrowPreviewWings();
      hideChannelPreview();
      if (previewLine) {
        chart.removeSeries(previewLine);
        previewLine = null;
      }
      firstPoint = null;
      secondPoint = null;
    }
    draggingPoint = null;
    chart.applyOptions({ handleScroll: true, handleScale: true });
  }

  function clearDrawings() {
    if (useSharedAPI) return; // Disable for shared mode
    for (let i = 0; i < drawnLines.length; i++) {
      const line = drawnLines[i];
      chart.removeSeries(line.series);
      if (line.wings) {
        for (let j = 0; j < line.wings.length; j++) {
          chart.removeSeries(line.wings[j]);
        }
      }
    }
    if (previewLine) {
      chart.removeSeries(previewLine);
      previewLine = null;
    }
    drawnLines = [];
    firstPoint = null;
    secondPoint = null;
    drawingActive = false;
    draggingPoint = null;
    selectedLineIndex = null;
    chart.applyOptions({ handleScroll: true, handleScale: true });
    saveTrendlines();
  }

  async function saveTrendlines() {
    if (useSharedAPI) return;

    if (mode === 'trade' && !tradeId) return;
    if (mode === 'plan' && !onSaveTrendlines) return;

    try {
      const linesData = drawnLines.map(function (line) {
        return {
          p1: { time: line.p1.time, price: line.p1.price },
          p2: { time: line.p2.time, price: line.p2.price },
          p3: line.p3 ? { time: line.p3.time, price: line.p3.price } : null,
          type: line.type || 'trendline',
          color: line.color,
          lineWidth: line.lineWidth,
        };
      });

      if (mode === 'plan') {
        onSaveTrendlines(linesData);
        return;
      }

      await tradesAPI.saveTrendlines(tradeId, linesData);
    } catch (e) {
      console.error('[Chart] Failed to save trendlines:', e);
    }
  }

  async function loadTrendlines() {
    if (mode === 'trade' && (!tradeId || !chart || !candlestickSeries)) return;
    if (mode === 'plan' && (!initialTrendlines || !chart || !candlestickSeries)) {
      if (mode === 'plan' && initialTrendlines) {
        applyLinesData(initialTrendlines);
      }
      return;
    }

    try {
      let linesData;
      if (mode === 'plan') {
        linesData = initialTrendlines;
      } else {
        let res;
        if (useSharedAPI && shareToken) {
          res = await sharesAPI.getTrendlines(shareToken, tradeId);
        } else {
          res = await tradesAPI.getTrendlines(tradeId);
        }
        linesData = res.data;
      }

      if (!linesData || linesData.length === 0) return;
      applyLinesData(linesData);
    } catch (e) {
      console.error('[Chart] Failed to load trendlines:', e);
    }
  }

  function applyLinesData(linesData) {
    for (const lineData of linesData) {
      const type = lineData.type || 'trendline';
      const series = chart.addSeries(LineSeries, {
        color: type === 'fib' ? 'rgba(0, 0, 0, 0)' : lineData.color || '#f59e0b',
        lineWidth: type === 'fib' ? 1 : lineData.lineWidth || 2,
        lineStyle: type === 'fib' || type === 'channel' ? 2 : 0,
        lastValueVisible: false,
        priceLineVisible: false,
        crosshairMarkerVisible: false,
      });

      const data = [
        { time: lineData.p1.time, value: lineData.p1.price },
        { time: lineData.p2.time, value: lineData.p2.price },
      ];
      data.sort(function (a, b) {
        return a.time - b.time;
      });
      series.setData(data);

      const lineObj = {
        series: series,
        p1: lineData.p1,
        p2: lineData.p2,
        p3: lineData.p3,
        type: type,
        color: lineData.color || '#f59e0b',
        lineWidth: lineData.lineWidth || 2,
        wings: [],
      };

      drawnLines.push(lineObj);
      if (type === 'arrow') updateLineWings(lineObj);
      if (type === 'fib') updateFibLevels(lineObj);
      if (type === 'channel') updateChannelLines(lineObj);
    }
    drawnLines = drawnLines;
    syncFibLabels();
  }

  function clearAllDrawnLines() {
    if (!chart) return;
    for (let i = 0; i < drawnLines.length; i++) {
      const line = drawnLines[i];
      chart.removeSeries(line.series);
      if (line.wings) {
        for (let j = 0; j < line.wings.length; j++) {
          chart.removeSeries(line.wings[j]);
        }
      }
    }
    drawnLines = [];
    syncFibLabels();
  }

  function applyChartConfig(config) {
    if (!chart || !config) return;
    try {
      if (config.period && config.period !== selectedPeriod) {
        selectedPeriod = config.period;
        loadData();
      }

      if (config.range) {
        chart.timeScale().setVisibleLogicalRange(config.range);
      }

      if (config.priceRange) {
        const priceScale = chart.priceScale('right');
        if (config.autoScale === false) {
          priceScale.applyOptions({ autoScale: false });
          priceScale.setVisibleRange(config.priceRange);
        } else {
          priceScale.applyOptions({ autoScale: true });
        }
      }
    } catch (e) {
      console.error('[Chart] Failed to apply config:', e);
    }
  }

  function getEffectivePoint(param) {
    if (!param || !param.point || !candlestickSeries || !chart || lastKnownData.length === 0)
      return null;

    const price = candlestickSeries.coordinateToPrice(param.point.y);
    if (price === null) return null;

    // 如果 param 有 time 直接用
    let time = param.time;
    if (time !== null && time !== undefined) {
      return { time, price };
    }

    // 否則嘗試根據邏輯座標推算 (支持在無 K 線區域繪圖)
    const logical = chart.timeScale().coordinateToLogical(param.point.x);
    if (logical === null) return null;

    const lastIndex = lastKnownData.length - 1;
    const lastBar = lastKnownData[lastIndex];

    // 計算間隔
    let interval = 300; // 預設 M5 (300秒)
    if (lastKnownData.length >= 2) {
      interval = lastKnownData[lastIndex].time - lastKnownData[lastIndex - 1].time;
    } else {
      // 根據 timeframe 字串回退
      if (timeframe.includes('1分')) interval = 60;
      else if (timeframe.includes('5分')) interval = 300;
      else if (timeframe.includes('15分')) interval = 900;
      else if (timeframe.includes('30分')) interval = 1800;
      else if (timeframe.includes('1小時')) interval = 3600;
      else if (timeframe.includes('4小時')) interval = 14400;
      else if (timeframe.includes('天')) interval = 86400;
    }

    const offset = logical - lastIndex;
    time = lastBar.time + Math.round(offset * interval);

    return { time, price };
  }

  function handleChartMouseUp() {
    if (isDraggingLine) {
      isDraggingLine = false;
      dragStartPoint = null;
      chart.applyOptions({ handleScroll: true, handleScale: true });
      if (!useSharedAPI) {
        saveTrendlines();
      }
    }
    debounceSaveConfig();
  }

  function handleChartMouseDown(param) {
    if (!param || !param.point || drawingActive || useSharedAPI) return; // Disable for shared mode

    const point = getEffectivePoint(param);
    if (!point) return;
    const { time, price } = point;

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
      if (param_t < 0) {
        xx = p1X;
        yy = p1Y;
      } else if (param_t > 1) {
        xx = p2X;
        yy = p2Y;
      } else {
        xx = p1X + param_t * C;
        yy = p1Y + param_t * D;
      }
      const dx = param.point.x - xx;
      const dy = param.point.y - yy;
      const dist = Math.sqrt(dx * dx + dy * dy);

      if (dist < 10) {
        // 開始拖曳整條線
        isDraggingLine = true;
        dragStartPoint = { time, price, x: param.point.x, y: param.point.y };
        chart.applyOptions({ handleScroll: false, handleScale: false });
      }
    }
  }

  async function copyChartImage() {
    if (!chart || copying) return;
    try {
      copying = true;
      const canvas = chart.takeScreenshot();
      canvas.toBlob(function (blob) {
        if (!blob) {
          copying = false;
          return;
        }

        const data = [new ClipboardItem({ 'image/png': blob })];
        navigator.clipboard
          .write(data)
          .then(function () {
            setTimeout(function () {
              copying = false;
            }, 2000);
          })
          .catch(function (err) {
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
    setTimeout(function () {
      if (chart && chartContainer) {
        chart.applyOptions({
          width: chartContainer.clientWidth,
          height: chartContainer.clientHeight,
        });
        syncFibLabels();
      }
    }, 100);
  }

  function toggleStyleMenu() {
    showStyleMenu = !showStyleMenu;
  }
  function selectColor(color) {
    selectedColor = color;
    updateActiveLineStyle();
    showStyleMenu = false;
  }
  function selectLineWidth(width) {
    selectedLineWidth = width;
    updateActiveLineStyle();
    showStyleMenu = false;
  }
  function handlePeriodChange() {
    loadData();
    debounceSaveConfig();
  }

  function isModeActive(mode, active, currentMode) {
    return active && currentMode === mode;
  }

  function canDelete(index, active, shared) {
    if (index === null) return false;
    if (active) return false;
    if (shared) return false;
    return true;
  }

  function isChannel(index, lines) {
    if (index === null) return false;
    const line = lines[index];
    if (!line) return false;
    return line.type === 'channel';
  }

  function handleKeydown(e) {
    if (e.key === 'Delete' || e.key === 'Backspace') {
      if (canDelete(selectedLineIndex, drawingActive, useSharedAPI)) {
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

      <select class="period-select" bind:value={selectedPeriod} on:change={handlePeriodChange}>
        {#each periods as p}
          <option value={p.value}>{p.label}</option>
        {/each}
      </select>
    </div>

    {#if !useSharedAPI}
      <div class="tools-group">
        {#if selectedLineIndex !== null}
          <button
            type="button"
            class="tool-button clear-button"
            on:click={deleteSelectedLine}
            title="刪除選中線條"
          >
            <span class="icon"
              ><svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path
                  d="M3 6h18m-2 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                /></svg
              ></span
            >
          </button>
        {:else if drawnLines.length > 0}
          <button
            type="button"
            class="tool-button clear-button"
            on:click={clearDrawings}
            title="清除所有線條"
          >
            <span class="icon"
              ><svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path
                  d="M3 6h18m-2 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                /></svg
              ></span
            >
          </button>
        {/if}

        <!-- 樣式選擇器 -->
        <div class="style-selector">
          <button
            type="button"
            class="tool-button style-button"
            on:click={toggleStyleMenu}
            title="線條樣式"
            style="display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px;"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="icon"
              ><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"></path></svg
            >
            <div
              style="width: 16px; height: 3px; background-color: {selectedColor}; border-radius: 2px;"
            ></div>
          </button>
          {#if showStyleMenu}
            <div class="style-menu">
              <div class="style-section">
                <label>顏色</label>
                <div class="color-options">
                  {#each colorOptions as color}
                    <button
                      type="button"
                      class="color-btn"
                      class:active={selectedColor === color}
                      style="background: {color}"
                      on:click={() => selectColor(color)}
                    ></button>
                  {/each}
                </div>
              </div>
              <div class="style-section">
                <label>粗細</label>
                <div class="width-options">
                  {#each lineWidthOptions as width}
                    <button
                      type="button"
                      class="width-btn"
                      class:active={selectedLineWidth === width}
                      on:click={() => selectLineWidth(width)}
                    >
                      {width}px
                    </button>
                  {/each}
                </div>
              </div>
            </div>
          {/if}
        </div>

        <button
          type="button"
          class="tool-button draw-button"
          class:active={isModeActive('trendline', drawingActive, drawingMode)}
          on:click={() => toggleDrawing('trendline')}
          title="趨勢線工具"
        >
          <span class="icon">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="19" cy="5" r="3" />
              <circle cx="5" cy="19" r="3" />
              <line x1="7.1" y1="16.9" x2="16.9" y2="7.1" />
            </svg>
          </span>
        </button>

        <button
          type="button"
          class="tool-button draw-button"
          class:active={isModeActive('arrow', drawingActive, drawingMode)}
          on:click={() => toggleDrawing('arrow')}
          title="箭頭工具"
        >
          <span class="icon">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="5" cy="19" r="3" />
              <path d="M19 5l-7 7" />
              <path d="M14 5h5v5" />
            </svg>
          </span>
        </button>

        <button
          type="button"
          class="tool-button draw-button"
          class:active={isModeActive('fib', drawingActive, drawingMode)}
          on:click={() => toggleDrawing('fib')}
          title="斐波那契工具"
        >
          <span class="icon">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="4" y1="6" x2="20" y2="6" />
              <line x1="4" y1="12" x2="20" y2="12" />
              <line x1="4" y1="18" x2="20" y2="18" />
              <circle cx="17" cy="12" r="2.5" />
              <circle cx="7" cy="18" r="2.5" />
            </svg>
          </span>
        </button>

        <button
          type="button"
          class="tool-button draw-button"
          class:active={isModeActive('channel', drawingActive, drawingMode)}
          on:click={() => toggleDrawing('channel')}
          title="平行通道線"
        >
          <span class="icon">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="3" y1="16" x2="21" y2="8" />
              <line x1="3" y1="21" x2="21" y2="13" />
            </svg>
          </span>
        </button>

        <button
          type="button"
          class="copy-button"
          on:click={copyChartImage}
          disabled={copying}
          title="複製圖表截圖"
        >
          {#if copying}
            <span class="icon"
              ><svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"><polyline points="20 6 9 17 4 12" /></svg
              ></span
            >
          {:else}
            <span class="icon">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path
                  d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"
                />
                <circle cx="12" cy="13" r="4" />
              </svg>
            </span>
          {/if}
        </button>

        <button
          type="button"
          class="tool-button fullscreen-button"
          on:click={toggleFullscreen}
          title={isFullscreen ? '退出全螢幕' : '全螢幕檢視'}
        >
          <span class="icon">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path
                d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"
              />
            </svg>
          </span>
        </button>
      </div>
    {:else}
      <!-- Shared Mode: Only show Read Only badge -->
      <div class="tools-group">
        <span class="readonly-badge">唯讀模式</span>
      </div>
    {/if}
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
        <span class="icon"
          ><svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path
              d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
            /><line x1="12" y1="9" x2="12" y2="13" /><line
              x1="12"
              y1="17"
              x2="12.01"
              y2="17"
            /></svg
          ></span
        >
        <span class="msg">{error}</span>
      </div>
    </div>
  {/if}

  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div bind:this={chartContainer} class="chart-container" on:mouseup={handleChartMouseUp}></div>

  {#each fibLabels as label}
    <div class="fib-label" style="left: {label.x}px; top: {label.y}px; color: {label.color};">
      {label.text}
    </div>
  {/each}

  {#if selectedLineIndex !== null}
    {#if !useSharedAPI}
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
      {#if isChannel(selectedLineIndex, drawnLines)}
        <div
          class="control-point-handle"
          style="left: {cp3.x}px; top: {cp3.y}px;"
          on:mousedown={handleMouseDownCP2}
        ></div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .chart-wrapper {
    position: relative;
    width: 100%;
    height: 100%;
    min-height: 500px;
    background: #0f172a;
    border-radius: 16px;
    overflow: hidden;
    border: 1px solid rgba(255, 255, 255, 0.08);
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

  .tags-group,
  .tools-group {
    display: flex;
    gap: 8px;
    pointer-events: none;
  }

  .copy-button,
  .tool-button {
    pointer-events: auto;
    background: rgba(30, 41, 59, 0.8);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    color: #cbd5e1;
    padding: 5px 10px;
    border-radius: 6px;
    font-size: 0.75rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
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

  .copy-button:active,
  .tool-button:active {
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

  .symbol-tag,
  .timeframe-tag,
  .timezone-tag {
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

  .period-select {
    background: rgba(30, 41, 59, 0.7);
    backdrop-filter: blur(4px);
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 0.75rem;
    color: #cbd5e1;
    border: 1px solid rgba(255, 255, 255, 0.1);
    font-family: 'JetBrains Mono', monospace;
    cursor: pointer;
    outline: none;
    margin-left: 8px;
    pointer-events: auto;
  }

  .period-select:hover {
    background: rgba(51, 65, 85, 0.9);
    border-color: rgba(59, 130, 246, 0.4);
    color: #fff;
  }

  .period-select option {
    background: #1e293b;
    color: #cbd5e1;
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

  .fib-label {
    position: absolute;
    pointer-events: none;
    font-size: 10px;
    font-family: 'JetBrains Mono', monospace;
    font-weight: bold;
    text-shadow: 0 0 3px rgba(0, 0, 0, 0.8);
    z-index: 15;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
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
