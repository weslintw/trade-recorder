<script>
  import { onMount, onDestroy } from 'svelte';
  import * as LightweightCharts from 'lightweight-charts';
  import { tradesAPI } from '../lib/api';

  export let tradeId;
  export let trade = null; 

  let chartContainer;
  let chart;
  let candlestickSeries;
  let loading = true;
  let error = null;

  onMount(function() {
    initChart();
    loadData();
  });

  onDestroy(function() {
    if (chart) {
      chart.remove();
    }
  });

  function initChart() {
    if (!chartContainer) return;
    
    // Ensure we use the correct createChart function
    const createChartFunc = LightweightCharts.createChart;
    if (typeof createChartFunc !== 'function') {
      console.error('LightweightCharts.createChart is not a function', LightweightCharts);
      error = "圖表套件載入錯誤";
      return;
    }

    chart = createChartFunc(chartContainer, {
      layout: {
        background: { color: 'transparent' },
        textColor: '#94a3b8',
      },
      grid: {
        vertLines: { color: 'rgba(255, 255, 255, 0.05)' },
        horzLines: { color: 'rgba(255, 255, 255, 0.05)' },
      },
      rightPriceScale: {
        borderColor: 'rgba(255, 255, 255, 0.1)',
        scaleMargins: {
          top: 0.1,
          bottom: 0.2,
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

    candlestickSeries = chart.addCandlestickSeries({
      upColor: '#10b981',
      downColor: '#ef4444',
      borderVisible: false,
      wickUpColor: '#10b981',
      wickDownColor: '#ef4444',
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
      const digits = resData.digits;

      if (!data || !data.trendbar || data.trendbar.length === 0) {
        error = "無法獲取 K 線數據 (數據可能已過期或伺服器暫時無法連接)";
        return;
      }

      const scale = Math.pow(10, digits);
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
        
        markers.push({
          time: entryTs,
          position: trade.side === 'long' ? 'belowBar' : 'aboveBar',
          color: trade.side === 'long' ? '#10b981' : '#ef4444',
          shape: trade.side === 'long' ? 'arrowUp' : 'arrowDown',
          text: 'Entry @ ' + trade.entry_price,
        });

        if (exitTs) {
          markers.push({
            time: exitTs,
            position: trade.side === 'long' ? 'aboveBar' : 'belowBar',
            color: '#3b82f6',
            shape: 'balloon',
            text: 'Exit @ ' + trade.exit_price,
          });
        }
      }
      
      markers.sort(function(a, b) { return a.time - b.time; });
      candlestickSeries.setMarkers(markers);
      
      chart.timeScale().fitContent();

    } catch (e) {
      console.error('[Chart Error]', e);
      const errorMsg = e.response && e.response.data && e.response.data.error ? e.response.data.error : e.message;
      error = "載入失敗: " + errorMsg;
    } finally {
      loading = false;
    }
  }
</script>

<div class="chart-wrapper">
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

  .chart-container {
    width: 100%;
    height: 100%;
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
