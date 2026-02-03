<script>
  import { onMount } from 'svelte';
  import { sharesAPI, imagesAPI } from '../lib/api';
  import {
    getStrategyLabel,
    determineMarketSession,
    getMarketSessionLabel,
    calculateDuration,
    calculateBulletSize,
    toTradingDateString,
    formatDate,
  } from '../lib/utils';
  import 'quill/dist/quill.snow.css';
  import Sparkline from './Sparkline.svelte';
  import SharedTradeDetail from './SharedTradeDetail.svelte';
  import SharedPlanDetail from './SharedPlanDetail.svelte';
  import PlanSummaryTable from './PlanSummaryTable.svelte';
import ImageAnnotator from './ImageAnnotator.svelte';
  import { isDarkMode } from '../lib/stores';

  export let token = '';

  let loading = true;
  let error = null;
  let sharedData = null; // { type: 'trade'|'plan'|'account'|'batch', data: ... }

  // Filtering State
  let activeFilterType = 'all'; // 'all', 'expert', 'elite', 'legend'
  let activeColorFilter = null; // null, 'green', 'yellow', 'red'
  let activeSubFilter = null; // for expert/elite sub-filters
  let activeDateRange = 'all'; // 'all', '1D', '1W', '1M', 'custom'
  let customStartDate = '';
  let customEndDate = '';
  let activeExitFilter = 'all'; // 'all', 'tp', 'sl'
  let activeSideFilter = 'all'; // 'all', 'long', 'short'

  const EXPERT_SIGNALS = [
    '向下蘇美',
    '起漲靠山',
    '雙柱',
    '夾縫',
    '喇叭-上',
    '喇叭-中',
    '喇叭-下',
    '倚天',
    '攻城池上',
    '起跌靠山',
    '君臨城下',
    '雙塔',
    '向上蘇美',
    '雷霆',
  ];
  const ELITE_PATTERNS = ['甲', '乙', '丙', '丁', '大Leading', '小Leading'];
  const LEGEND_CHECKLIST = [
    { id: 'item_618_786', label: '王者回調' },
    { id: 'item_che', label: '大時區破測破' },
    { id: 'item_de', label: '整理段訊號' },
    { id: 'item_legend_wave', label: '傳奇波段' },
  ];

  const subFilters = {
    expert: EXPERT_SIGNALS.map(s => ({ value: s, label: s })),
    elite: ELITE_PATTERNS.map(p => ({ value: p, label: p })),
    legend: LEGEND_CHECKLIST.map(l => ({ value: l.id, label: l.label })),
  };

  // Detail View State
  let selectedItem = null;

  // Image Modal State
  let enlargedImage = null;
  let enlargedImageTitle = '';
  let isAnnotating = false;

  function openModal(src, title) {
    if (!src) return;
    enlargedImage = src;
    enlargedImageTitle = title || '圖片';
    isAnnotating = false;
  }

  function closeModal() {
    enlargedImage = null;
    enlargedImageTitle = '';
    zoom = 1;
    offsetX = 0;
    offsetY = 0;
  }

  // Image Modal Advanced State
  let zoom = 1;
  let offsetX = 0;
  let offsetY = 0;
  let isDragging = false;
  let startX = 0;
  let startY = 0;

  function handleWheel(e) {
    if (isAnnotating) return; // 標註模式下允許原生捲動
    if (!enlargedImage) return;
    e.preventDefault();
    const delta = e.deltaY;
    const zoomStep = 0.1;
    const prevZoom = zoom;

    if (delta < 0) {
      zoom = Math.min(zoom + zoomStep, 5);
    } else {
      zoom = Math.max(zoom - zoomStep, 0.5);
    }

    // When zooming out to <= 1, reset offsets
    if (zoom <= 1) {
      offsetX = 0;
      offsetY = 0;
    }
  }

  function handleMouseDown(e) {
    if (zoom <= 1) return;
    isDragging = true;
    startX = e.clientX - offsetX;
    startY = e.clientY - offsetY;
    e.preventDefault();
  }

  function handleMouseMove(e) {
    if (!isDragging) return;
    offsetX = e.clientX - startX;
    offsetY = e.clientY - startY;
  }

  function handleMouseUp() {
    isDragging = false;
  }

  onMount(async () => {
    try {
      const res = await sharesAPI.getPublic(token);
      if (res.data && res.data.type) {
        sharedData = res.data;
      } else {
        error = '後端回傳資料格式不正確';
      }
    } catch (e) {
      console.error('[SharedViewer] Fetch Error:', e);
      error = e.response?.data?.error || '查無此分享內容或連結已過期';
    } finally {
      loading = false;
    }
  });

  function toggleDarkMode() {
    isDarkMode.update(v => !v);
  }

  function getTimeRange(trades) {
    if (!trades || trades.length === 0) return '';
    const times = trades.filter(t => t.entry_time).map(t => new Date(t.entry_time).getTime());
    if (times.length === 0) return '';
    const min = new Date(Math.min(...times));
    const max = new Date(Math.max(...times));

    const fmt = d =>
      d.toLocaleDateString('zh-TW', { year: 'numeric', month: '2-digit', day: '2-digit' });

    if (fmt(min) === fmt(max)) return fmt(min);
    return `${fmt(min)} ~ ${fmt(max)}`;
  }


  function formatTime(dateStr) {
    if (!dateStr) return '';
    try {
      const d = new Date(dateStr);
      return d.toLocaleTimeString('zh-TW', { hour: '2-digit', minute: '2-digit', hour12: false });
    } catch (e) {
      return '';
    }
  }

  function formatDay(dateStr) {
    if (!dateStr || dateStr === 'unknown') return '日期不詳';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-TW', { month: 'long', day: 'numeric', weekday: 'short' });
  }

  function parseJSON(str, defaultValue = null) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  function lazyLoadHTML(html) {
    if (!html) return '';
    return html.replace(/<img /g, '<img loading="lazy" ');
  }


  function groupDataByDate(trades, plans) {
    const groups = {};

    trades.forEach(t => {
      // Use trading date logic
      const date = t.entry_time ? toTradingDateString(t.entry_time) : 'unknown';
      if (!groups[date]) groups[date] = { date, trades: [], plans: [] };
      groups[date].trades.push(t);
    });

    plans.forEach(p => {
      // Plans use their intrinsic date
      const date = p.plan_date ? p.plan_date.slice(0, 10) : 'unknown';
      if (!groups[date]) groups[date] = { date, trades: [], plans: [] };
      groups[date].plans.push(p);
    });

    // Sort days DESC (newest first)
    return Object.values(groups)
      .sort((a, b) => b.date.localeCompare(a.date))
      .map(g => {
        // Sort trades DESC (newest first)
        g.trades.sort((a, b) => {
          const timeA = a.entry_time ? new Date(a.entry_time).getTime() : 0;
          const timeB = b.entry_time ? new Date(b.entry_time).getTime() : 0;
          return timeB - timeA;
        });
        // Sort plans DESC (newest first)
        g.plans.sort((a, b) => {
          const timeA = a.plan_date ? new Date(a.plan_date).getTime() : 0;
          const timeB = b.plan_date ? new Date(b.plan_date).getTime() : 0;
          // If dates are same, sort by ID
          if (timeB === timeA) return b.id - a.id;
          return timeB - timeA;
        });
        return g;
      });
  }

  $: filteredTrades = (() => {
    if (!sharedData || !sharedData.data || !sharedData.data.trades) return [];
    return sharedData.data.trades.filter(t => {
      // Date Filter
      if (activeDateRange !== 'all') {
        const entryDate = t.entry_time ? toTradingDateString(t.entry_time) : 'unknown';
        if (entryDate === 'unknown') return false;
        if (customStartDate && entryDate < customStartDate) return false;
        if (customEndDate && entryDate > customEndDate) return false;
      }

      // TP/SL Filter
      if (activeExitFilter === 'tp') {
        if (!(t.pnl > 0)) return false;
      } else if (activeExitFilter === 'sl') {
        if (!(t.pnl < 0)) return false;
      }

      // Color Filter
      if (activeColorFilter && t.color_tag !== activeColorFilter) return false;

      // Strategy Type Filter
      const tStrat = String(t.entry_strategy || '').toLowerCase();
      const stratMatch =
        activeFilterType === 'all' ||
        (activeFilterType === 'expert' && (tStrat === 'expert' || tStrat === '達人')) ||
        (activeFilterType === 'elite' && (tStrat === 'elite' || tStrat === '菁英')) ||
        (activeFilterType === 'legend' && (tStrat === 'legend' || tStrat === '傳奇')) ||
        tStrat === activeFilterType;

      if (!stratMatch) return false;

      // Sub Filter (JSON String Search like Home.svelte)
      if (activeSubFilter) {
        try {
          const tradeStr = JSON.stringify(t);
          if (!tradeStr.includes(String(activeSubFilter))) return false;
        } catch (e) {
          return false;
        }
      }

      // Side Filter
      if (activeSideFilter === 'long') {
        if (t.side !== 'long' && t.side !== 'buy') return false;
      } else if (activeSideFilter === 'short') {
        if (t.side !== 'short' && t.side !== 'sell') return false;
      }

      return true;
    });
  })();

  $: filteredPlans = (() => {
    if (!sharedData || !sharedData.data || !sharedData.data.plans) return [];
    return sharedData.data.plans.filter(p => {
      // Date Filter
      if (activeDateRange !== 'all') {
        const planDate = p.plan_date ? p.plan_date.slice(0, 10) : 'unknown';
        if (planDate === 'unknown') return false;
        if (customStartDate && planDate < customStartDate) return false;
        if (customEndDate && planDate > customEndDate) return false;
      }
      return true;
    });
  })();

  $: groupedData = groupDataByDate(filteredTrades, filteredPlans);

  $: filteredStats = (() => {
    let stats = {
      total: 0,
      wins: 0,
      winRate: '0.0',
      green: 0,
      yellow: 0,
      red: 0,
      expert: 0,
      elite: 0,
      legend: 0,
      hasTrades: false,
    };
    if (!filteredTrades) return stats;

    filteredTrades.forEach(t => {
      if (t.color_tag === 'green') stats.green++;
      else if (t.color_tag === 'yellow') stats.yellow++;
      else if (t.color_tag === 'red') stats.red++;

      const strat = String(t.entry_strategy || '').toLowerCase();
      if (strat === 'expert' || strat === '達人') stats.expert++;
      else if (strat === 'elite' || strat === '菁英') stats.elite++;
      else if (strat === 'legend' || strat === '傳奇') stats.legend++;

      if (t.trade_type === 'actual' && t.exit_time && t.pnl !== null) {
        stats.total++;
        if (t.pnl > 0) stats.wins++;
      }
    });

    stats.hasTrades = filteredTrades.length > 0;
    if (stats.total > 0) stats.winRate = ((stats.wins * 100) / stats.total).toFixed(1);
    return stats;
  })();

  function selectFilterType(type) {
    if (activeFilterType === type) {
      activeFilterType = 'all';
    } else {
      activeFilterType = type;
    }
    activeSubFilter = null;
  }

  function setDateRange(range) {
    if (activeDateRange === range) {
      activeDateRange = 'all';
      customStartDate = '';
      customEndDate = '';
    } else {
      activeDateRange = range;
      const now = new Date();
      const todayStr = toTradingDateString(now);

      if (range === '1D') {
        customStartDate = todayStr;
        customEndDate = todayStr;
      } else if (range === '1W') {
        const d = new Date();
        d.setDate(d.getDate() - 7);
        customStartDate = toTradingDateString(d);
        customEndDate = todayStr;
      } else if (range === '1M') {
        const d = new Date();
        d.setMonth(d.getMonth() - 1);
        customStartDate = toTradingDateString(d);
        customEndDate = todayStr;
      }
    }
  }

  function selectColorFilter(color) {
    activeColorFilter = activeColorFilter === color ? null : color;
  }

  function toggleSubFilter(val) {
    activeSubFilter = activeSubFilter === val ? null : val;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && enlargedImage) {
      closeModal();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="shared-view-container">
  {#if loading}
    <div class="loading-overlay">
      <div class="loader"></div>
      <p>正在載入分享內容...</p>
    </div>
  {:else if error}
    <div class="status-box card error">
      <div class="error-icon">⚠️</div>
      <h2>存取失敗</h2>
      <p>{error}</p>
      <a href="/" class="btn btn-primary">回到首頁</a>
    </div>
  {:else if sharedData}
    <div class="shared-content">
      <div class="shared-top-bar">
        <div class="logo-area">
          <div class="logo-image-container">
            {#if $isDarkMode}
              <img
                src="/logo-dark.png"
                alt="Trade Time Machine Logo"
                class="brand-logo-img dark"
              />
            {:else}
              <img src="/logo.png" alt="Trade Time Machine Logo" class="brand-logo-img" />
            {/if}
          </div>
          <div class="public-badge-small">👁️ 唯讀分享模式</div>
        </div>
        <button
          class="theme-toggle-btn"
          on:click={toggleDarkMode}
          title={$isDarkMode ? '切換至淺色模式' : '切換至深色模式'}
        >
          {$isDarkMode ? '🌙' : '☀️'}
        </button>
      </div>

      {#if selectedItem}
        <div class="detail-overlay-header">
          <button class="back-btn" on:click={() => (selectedItem = null)}>
            <span class="icon">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path d="M9 14 4 9l5-5" /><path d="M4 9h12a4 4 0 0 1 4 4v2" /></svg
              >
            </span> 返回列表
          </button>
        </div>
        {#if selectedItem.type === 'trade'}
          <SharedTradeDetail trade={selectedItem.data} {openModal} />
        {:else if selectedItem.type === 'plan'}
          <SharedPlanDetail plan={selectedItem.data} {openModal} />
        {/if}
      {:else if sharedData.type === 'trade' && sharedData.data}
        <SharedTradeDetail trade={sharedData.data} {openModal} />
      {:else if sharedData.type === 'plan' && sharedData.data}
        <SharedPlanDetail plan={sharedData.data} {openModal} />
      {:else if (sharedData.type === 'account' || sharedData.type === 'batch') && sharedData.data}
        {@const data = sharedData.data}
        {@const grouped = groupDataByDate(data.trades || [], data.plans || [])}
        <div class="batch-viewer">
          <div class="view-header-main">
            <h1>
              {sharedData.type === 'account'
                ? `${sharedData.username} 的 ${data.account.name}`
                : `${sharedData.username} 的精選分享`}
            </h1>

            <div class="header-info-line">
              {#if data.account}
                <div class="account-badges">
                  <span class="acc-badge type"
                    >{data.account.type === 'ctrader' ? 'cTrader' : 'Local'}</span
                  >
                  {#if data.account.ctrader_env}
                    <span class="acc-badge env {data.account.ctrader_env}"
                      >{data.account.ctrader_env.toUpperCase()}</span
                    >
                  {/if}
                  {#if data.account.ctrader_account_id}
                    <span class="acc-badge id">ID: {data.account.ctrader_account_id}</span>
                  {/if}
                </div>
              {/if}
              <div class="time-range-badge">
                📅 {getTimeRange(data.trades)}
              </div>
            </div>

            <p class="batch-meta">
              包含 {data.trades ? data.trades.length : 0} 筆交易與 {data.plans
                ? data.plans.length
                : 0} 筆規劃
            </p>

            <!-- Ported Filter UI -->
            <div class="filter-section">
              <div class="filter-glass-container">
                <!-- 日期篩選區 -->
                <div class="filter-date-row">
                  <div class="date-presets">
                    <button
                      class="filter-type-btn {activeDateRange === '1D' ? 'active' : ''}"
                      on:click={() => setDateRange('1D')}
                    >
                      1日
                    </button>
                    <button
                      class="filter-type-btn {activeDateRange === '1W' ? 'active' : ''}"
                      on:click={() => setDateRange('1W')}
                    >
                      1週
                    </button>
                    <button
                      class="filter-type-btn {activeDateRange === '1M' ? 'active' : ''}"
                      on:click={() => setDateRange('1M')}
                    >
                      1月
                    </button>
                    <button
                      class="filter-type-btn {activeDateRange === 'custom' ? 'active' : ''}"
                      on:click={() => {
                        if (activeDateRange === 'custom') {
                          activeDateRange = 'all';
                          customStartDate = '';
                          customEndDate = '';
                        } else {
                          activeDateRange = 'custom';
                          if (!customStartDate) {
                            const d = new Date();
                            d.setDate(d.getDate() - 7);
                            customStartDate = toTradingDateString(d);
                            customEndDate = toTradingDateString(new Date());
                          }
                        }
                      }}
                    >
                      📅 自訂
                    </button>
                  </div>

                  {#if activeDateRange === 'custom'}
                    <div class="custom-date-inputs">
                      <input type="date" class="date-input" bind:value={customStartDate} />
                      <span class="date-sep">~</span>
                      <input type="date" class="date-input" bind:value={customEndDate} />
                    </div>
                  {/if}

                  <div class="filter-stats-spacer"></div>

                  <div class="filter-stats-badge" class:has-data={filteredStats.hasTrades}>
                    <div class="stats-icon">✅</div>
                    <div class="stats-content">
                      <span class="stats-label">結果：</span>
                      <span class="stats-value">{filteredTrades.length} 筆</span>
                      {#if filteredStats.total > 0}
                        <span class="stats-sep">/</span>
                        <span class="stats-label">勝率</span>
                        <span class="stats-value win-rate">{filteredStats.winRate}%</span>
                      {/if}
                      <div class="stats-color-groups">
                        <span class="stats-color-dot green"></span>
                        <span class="stats-color-count">{filteredStats.green}</span>
                        <span class="stats-color-dot yellow"></span>
                        <span class="stats-color-count">{filteredStats.yellow}</span>
                        <span class="stats-color-dot red"></span>
                        <span class="stats-color-count">{filteredStats.red}</span>
                      </div>

                      <div class="stats-strategy-groups">
                        <span class="strat-tag expert">達 {filteredStats.expert}</span>
                        <span class="strat-tag elite">菁 {filteredStats.elite}</span>
                        <span class="strat-tag legend">傳 {filteredStats.legend}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="divider-horizontal"></div>

                <div class="filter-main-types">
                  <button
                    class="filter-type-btn {activeFilterType === 'all' ? 'active' : ''}"
                    on:click={() => selectFilterType('all')}
                  >
                    全部
                  </button>
                  <div class="divider"></div>
                  <button
                    class="filter-type-btn {activeFilterType === 'expert' ? 'active' : ''}"
                    on:click={() => selectFilterType('expert')}
                  >
                    👨‍🏫 達人
                  </button>
                  <button
                    class="filter-type-btn {activeFilterType === 'elite' ? 'active' : ''}"
                    on:click={() => selectFilterType('elite')}
                  >
                    🛡️ 菁英
                  </button>
                  <button
                    class="filter-type-btn {activeFilterType === 'legend' ? 'active' : ''}"
                    on:click={() => selectFilterType('legend')}
                  >
                    👑 傳奇
                  </button>

                  <div class="divider"></div>

                  <button
                    class="filter-type-btn color-filter {activeColorFilter === 'green'
                      ? 'active'
                      : ''}"
                    on:click={() => selectColorFilter('green')}
                  >
                    <span class="stats-color-dot green"></span>
                  </button>
                  <button
                    class="filter-type-btn color-filter {activeColorFilter === 'yellow'
                      ? 'active'
                      : ''}"
                    on:click={() => selectColorFilter('yellow')}
                  >
                    <span class="stats-color-dot yellow"></span>
                  </button>
                  <button
                    class="filter-type-btn color-filter {activeColorFilter === 'red'
                      ? 'active'
                      : ''}"
                    on:click={() => selectColorFilter('red')}
                  >
                    <span class="stats-color-dot red"></span>
                  </button>

                  <div class="divider"></div>

                  <button
                    class="filter-type-btn {activeExitFilter === 'tp' ? 'active' : ''}"
                    on:click={() => (activeExitFilter = activeExitFilter === 'tp' ? 'all' : 'tp')}
                  >
                    🎯 TP
                  </button>
                  <button
                    class="filter-type-btn {activeExitFilter === 'sl' ? 'active' : ''}"
                    on:click={() => (activeExitFilter = activeExitFilter === 'sl' ? 'all' : 'sl')}
                  >
                    🛑 SL
                  </button>

                  <div class="divider"></div>

                  <button
                    class="filter-type-btn {activeSideFilter === 'long' ? 'active' : ''}"
                    on:click={() =>
                      (activeSideFilter = activeSideFilter === 'long' ? 'all' : 'long')}
                  >
                    📈 做多
                  </button>
                  <button
                    class="filter-type-btn {activeSideFilter === 'short' ? 'active' : ''}"
                    on:click={() =>
                      (activeSideFilter = activeSideFilter === 'short' ? 'all' : 'short')}
                  >
                    📉 做空
                  </button>
                </div>

                {#if activeFilterType !== 'all'}
                  <div class="sub-filter-scroll-wrapper">
                    <div class="sub-filter-container">
                      {#each subFilters[activeFilterType] as sub}
                        <button
                          class="sub-filter-chip {activeSubFilter === sub.value ? 'active' : ''}"
                          on:click={() => toggleSubFilter(sub.value)}
                        >
                          {sub.label}
                        </button>
                      {/each}
                    </div>
                  </div>
                {/if}
              </div>
            </div>
          </div>
          <div class="timeline">
            {#each groupedData as group}
              <div class="day-group">
                <div class="date-header"><span class="date-tag">{formatDay(group.date)}</span></div>
                <div class="day-card-container">
                  <!-- 左側規劃 -->
                  <div class="plan-column">
                    {#if group.plans.length > 0}
                      {#each group.plans as plan}
                        {@const trendData = parseJSON(plan.trend_analysis, {})}
                        <div
                          class="plan-item-card clickable"
                          on:click={() => (selectedItem = { type: 'plan', data: plan })}
                        >
                          <div class="item-header">
                            <span class="item-type">📌 盤面規劃</span>
                            <span class="symbol-inline-tag">{plan.symbol}</span>
                          </div>

                          <PlanSummaryTable {trendData} detailed={false} />

                          {#if trendData.asian?.notes || trendData.european?.notes || trendData.us?.notes}
                            <div class="mini-notes">
                              {#if trendData.asian?.notes}<div class="mini-note-item">
                                  <span class="note-session asian">亞</span
                                  >{trendData.asian.notes.slice(0, 30)}{trendData.asian.notes
                                    .length > 30
                                    ? '...'
                                    : ''}
                                </div>{/if}
                              {#if trendData.european?.notes}<div class="mini-note-item">
                                  <span class="note-session european">歐</span
                                  >{trendData.european.notes.slice(0, 30)}{trendData.european.notes
                                    .length > 30
                                    ? '...'
                                    : ''}
                                </div>{/if}
                              {#if trendData.us?.notes}<div class="mini-note-item">
                                  <span class="note-session us">美</span>{trendData.us.notes.slice(
                                    0,
                                    30
                                  )}{trendData.us.notes.length > 30 ? '...' : ''}
                                </div>{/if}
                            </div>
                          {:else if plan.notes && plan.notes !== 'Session-based unified plan'}
                            <p class="simple-notes">{plan.notes}</p>
                          {/if}
                        </div>
                      {/each}
                    {:else}
                      <div class="empty-placeholder-shared">無規劃紀錄</div>
                    {/if}
                  </div>

                  <!-- 右側交易 -->
                  <div class="trade-column">
                    {#if group.trades.length > 0}
                      <div class="trades-stack">
                        {#each group.trades as trade}
                          {@const bulletToDisplay = calculateBulletSize(trade) || trade.bullet_size}
                          <div
                            class="trade-item-card clickable {trade.color_tag
                              ? `tag-${trade.color_tag}`
                              : ''} {trade.trade_type === 'actual' && !trade.exit_time
                              ? 'is-ongoing'
                              : ''}"
                            on:click={() => (selectedItem = { type: 'trade', data: trade })}
                          >
                            <div class="item-header">
                              <div class="trade-meta">
                                <span class="symbol-inline-tag">{trade.symbol}</span>
                                <span class="session-tag {determineMarketSession(trade.entry_time)}"
                                  >{getMarketSessionLabel(trade)}</span
                                >
                                {#if trade.entry_strategy}<span
                                    class="strategy-tag {trade.entry_strategy}"
                                    >{getStrategyLabel(trade.entry_strategy)}</span
                                  >{/if}
                                <span class="side-tag {trade.side}"
                                  >{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span
                                >
                              </div>
                              <div class="trade-right">
                                <div class="color-tags-static">
                                  <span
                                    class="color-dot green {trade.color_tag === 'green'
                                      ? 'active'
                                      : ''}"
                                  ></span>
                                  <span
                                    class="color-dot yellow {trade.color_tag === 'yellow'
                                      ? 'active'
                                      : ''}"
                                  ></span>
                                  <span
                                    class="color-dot red {trade.color_tag === 'red'
                                      ? 'active'
                                      : ''}"
                                  ></span>
                                </div>
                                {#if trade.pnl_series}
                                  <div class="header-sparkline">
                                    <Sparkline
                                      data={trade.pnl_series}
                                      isOpen={trade.trade_type === 'actual' && !trade.exit_time}
                                      width={120}
                                      height={40}
                                    />
                                  </div>
                                {/if}
                                <span class="pnl-tag {trade.pnl >= 0 ? 'profit' : 'loss'}">
                                  {trade.pnl >= 0 ? '+' : ''}{Number(trade.pnl || 0).toFixed(2)}
                                </span>
                              </div>
                            </div>

                            <div class="trade-details-shared">
                              <div class="detail-row">
                                <div class="info-group">
                                  <span class="label">進場</span>
                                  <strong>{trade.entry_price}</strong>
                                  <span class="arrow">→</span>
                                  <span class="label">平倉</span>
                                  <strong>{trade.exit_price || '-'}</strong>
                                </div>
                                <div class="info-group">
                                  <span class="label">初始ＳＬ</span>
                                  <strong>{trade.initial_sl || '-'}</strong>
                                  {#if trade.exit_sl}
                                    <span class="label">平倉ＳＬ</span>
                                    <strong>{trade.exit_sl}</strong>
                                  {/if}
                                </div>
                                <div class="info-group">
                                  <span class="label">子彈</span>
                                  <strong class="bullet">
                                    {bulletToDisplay ? bulletToDisplay.toFixed(1) : 'NA'}
                                  </strong>
                                  {#if bulletToDisplay && (trade.rr_ratio || trade.rr_ratio === 0)}
                                    <span class="label">風報比</span>
                                    <strong class="rr {trade.rr_ratio >= 0 ? 'profit' : 'loss'}"
                                      >{trade.rr_ratio.toFixed(2)}</strong
                                    >
                                  {/if}
                                  <span class="label">手數</span>
                                  <strong>{trade.lot_size}</strong>
                                </div>
                              </div>
                              <div class="trade-time-shared">
                                {formatTime(trade.entry_time)} - {trade.exit_time
                                  ? formatTime(trade.exit_time)
                                  : trade.trade_type === 'actual'
                                    ? '進行中'
                                    : ''}
                                {#if trade.exit_time}
                                  <span class="duration-text"
                                    >({calculateDuration(trade.entry_time, trade.exit_time)})</span
                                  >
                                {/if}
                              </div>
                            </div>

                            {#if trade.images && trade.images.length > 0}
                              <div class="mini-gallery-shared">
                                {#each trade.images.slice(0, 3) as img}
                                  <div
                                    class="mini-img"
                                    on:click|stopPropagation={() =>
                                      openModal(
                                        imagesAPI.getUrl(img.image_path),
                                        trade.symbol + ' 交易圖表'
                                      )}
                                  >
                                    <img src={imagesAPI.getUrl(img.image_path)} alt="trade" loading="lazy" />
                                  </div>
                                {/each}
                                {#if trade.images.length > 3}<div class="more-imgs">
                                    +{trade.images.length - 3}
                                  </div>{/if}
                              </div>
                            {/if}

                            {#if trade.journal || trade.exit_reason || trade.notes}
                              <div class="card-notes-section">
                                {#if trade.journal}
                                  <div class="note-block">
                                    <div class="note-label">📌 記事備註</div>
                                    <div class="note-content">{@html trade.journal}</div>
                                  </div>
                                {/if}
                                {#if trade.exit_reason}
                                  <div class="note-block">
                                    <div class="note-label">🚪 平倉理由</div>
                                    <div class="note-content">{@html trade.exit_reason}</div>
                                  </div>
                                {/if}
                                {#if trade.notes}
                                  <div class="note-block">
                                    <div class="note-label">📝 復盤日記</div>
                                    <div class="note-content">{@html trade.notes}</div>
                                  </div>
                                {/if}
                              </div>
                            {/if}
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-placeholder-shared">無交易紀錄</div>
                    {/if}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <div class="status-box card error">
          <p>找不到該類型資料</p>
        </div>
      {/if}
    </div>
  {:else}
    <div class="status-box card">
      <p>無內容可顯示</p>
    </div>
  {/if}
</div>

{#if enlargedImage}
  <div
    class="image-modal"
    on:click={closeModal}
    role="button"
    tabindex="0"
    on:keydown={e => e.key === 'Escape' && closeModal()}
    on:wheel={handleWheel}
  >
    <div
      class="image-modal-content"
      on:click|stopPropagation
      role="button"
      tabindex="0"
      on:keypress|stopPropagation
      on:mousedown={handleMouseDown}
      on:mousemove={handleMouseMove}
      on:mouseup={handleMouseUp}
      on:mouseleave={handleMouseUp}
    >
      <button class="image-modal-close" on:click={closeModal}>×</button>

      {#if isAnnotating}
        <div class="annotator-wrapper-modal">
          <ImageAnnotator
            imageSrc={enlargedImage}
            originalImageSrc={enlargedImage}
            showSaveButton={false}
          />
        </div>
        <button class="annotate-toggle-btn active" on:click={() => (isAnnotating = false)}>
          ❌ 結束標註 (無法存檔)
        </button>
      {:else}
        <div
          class="zoom-container"
          style="transform: scale({zoom}) translate({offsetX / zoom}px, {offsetY /
            zoom}px); cursor: {zoom > 1 ? (isDragging ? 'grabbing' : 'grab') : 'default'}"
        >
          <img src={enlargedImage} alt={enlargedImageTitle} class="modal-img" />
        </div>
        {#if enlargedImageTitle}<div class="image-modal-caption">
            {enlargedImageTitle} (滾輪可縮放，放大的圖片可拖動)
          </div>{/if}

        <button class="annotate-toggle-btn" on:click|stopPropagation={() => (isAnnotating = true)}>
          ✏️ 標註
        </button>
      {/if}
    </div>
  </div>
{/if}

<style>
  .annotator-wrapper-modal {
    width: 95vw;
    height: 85vh; /* 避免太滿 */
    background: white;
    border-radius: 8px;
    padding: 1rem;
    overflow: auto; /* 允許內部捲動 */
    display: flex;
    flex-direction: column; /* 確保內容垂直排列 */
    align-items: center;
    justify-content: flex-start; /* 從頂部開始 */
    margin-top: 50px; /* 避開頂部的關閉按鈕 */
  }

  .annotate-toggle-btn {
    position: absolute;
    top: 20px;
    left: 20px;
    background: rgba(255, 255, 255, 0.2);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.4);
    padding: 8px 16px;
    border-radius: 99px;
    cursor: pointer;
    font-weight: 700;
    backdrop-filter: blur(4px);
    transition: all 0.2s;
    z-index: 20;
  }
  .annotate-toggle-btn:hover {
    background: rgba(255, 255, 255, 0.4);
  }
  .annotate-toggle-btn.active {
    background: #ef4444; /* red for exit */
    border-color: #f87171;
  }
  .shared-view-container {
    max-width: 1200px;
    margin: 3rem auto;
    padding: 0 1.25rem;
    font-family: 'Inter', sans-serif;
  }
  .detail-overlay-header {
    margin-bottom: 1.5rem;
  }
  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.6rem 1.25rem;
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    color: #64748b;
    font-weight: 700;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }
  .back-btn:hover {
    background: #f8fafc;
    border-color: #cbd5e1;
    color: #334155;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }
  .back-btn .icon {
    font-size: 1.1rem;
  }

  .card {
    background: var(--card-bg);
    border-radius: 1.5rem;
    padding: 2.5rem;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05);
    border: 1px solid var(--border-color);
    margin-bottom: 2rem;
  }
  .public-badge {
    background: var(--bg-main);
    color: var(--text-muted);
    padding: 0.5rem 1.25rem;
    border-radius: 99px;
    font-size: 0.8rem;
    font-weight: 700;
    margin-bottom: 1.5rem;
    border: 1px solid var(--border-color);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    line-height: 1;
  }

  /* Filter UI Port */
  .shared-filter-bar {
    margin: 1.5rem 0 2rem;
    padding: 1.25rem;
    background: #fff;
    border-radius: 16px;
    border: 1px solid #e2e8f0;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
  }
  .filter-main-types {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .filter-type-btn {
    padding: 0.5rem 1rem;
    border: 1px solid #e2e8f0;
    background: #fff;
    border-radius: 8px;
    font-weight: 600;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s;
    color: #475569;
  }
  .filter-type-btn.active {
    background: #6366f1;
    color: white;
    border-color: #6366f1;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.2);
  }
  .divider {
    width: 1px;
    height: 20px;
    background: #e2e8f0;
    margin: 0 0.25rem;
  }
  .filter-stats-spacer {
    flex: 1;
  }
  .filter-stats-badge {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.4rem 0.8rem;
    background: #f8fafc;
    border-radius: 99px;
    border: 1px solid #e2e8f0;
    font-size: 0.8rem;
  }
  .stats-label {
    color: #94a3b8;
    font-weight: 600;
  }
  .stats-value {
    color: #1e293b;
    font-weight: 700;
  }
  .stats-sep {
    color: #cbd5e1;
  }
  .win-rate {
    color: #10b981;
  }
  .sub-filter-scroll-wrapper {
    margin-top: 1rem;
    border-top: 1px solid #f1f5f9;
    padding-top: 1rem;
  }
  .sub-filter-container {
    display: flex;
    gap: 0.5rem;
    overflow-x: auto;
    padding-bottom: 4px;
    scrollbar-width: none;
  }
  .sub-filter-container::-webkit-scrollbar {
    display: none;
  }
  .sub-filter-chip {
    padding: 0.4rem 1rem;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 99px;
    font-size: 0.75rem;
    white-space: nowrap;
    cursor: pointer;
    transition: all 0.2s;
    color: #475569;
  }
  .sub-filter-chip.active {
    background: #6366f1;
    color: white;
    border-color: #6366f1;
  }
  .color-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    display: inline-block;
  }
  .color-dot.green {
    background: #22c55e;
  }
  .color-dot.yellow {
    background: #eab308;
  }
  .color-dot.red {
    background: #ef4444;
  }

  .stats-strategy-groups {
    display: flex;
    gap: 0.4rem;
    margin-left: 0.6rem;
    padding-left: 0.6rem;
    border-left: 1px solid rgba(0, 0, 0, 0.1);
  }

  :global(body.dark-mode) .stats-strategy-groups {
    border-left-color: rgba(255, 255, 255, 0.1);
  }

  .strat-tag {
    font-size: 0.75rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    color: white;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .strat-tag.expert {
    background: #6366f1;
  }
  .strat-tag.elite {
    background: #f59e0b;
  }
  .strat-tag.legend {
    background: #ef4444;
  }

  :global(body.dark-mode) .shared-filter-bar {
    background: #1e293b;
    border-color: #334155;
  }
  :global(body.dark-mode) .filter-type-btn {
    background: #1e293b;
    border-color: #334155;
    color: #94a3b8;
  }
  :global(body.dark-mode) .filter-stats-badge {
    background: #0f172a;
    border-color: #334155;
  }
  :global(body.dark-mode) .sub-filter-chip {
    background: #334155;
    border-color: #475569;
    color: #cbd5e1;
  }
  .view-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 2rem;
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 1.5rem;
  }
  .symbol-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #4f46e5;
    color: white;
    padding: 0.25rem 0.75rem;
    border-radius: 6px;
    font-weight: 800;
    font-size: 0.875rem;
    line-height: 1;
  }

  .pnl-value {
    font-size: 2.5rem;
    font-weight: 900;
  }
  .pnl-value.profit {
    color: #10b981;
  }
  .pnl-value.loss {
    color: #ef4444;
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
  }
  .info-item label {
    display: block;
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-bottom: 0.25rem;
    font-weight: 700;
  }
  .info-item span {
    font-size: 1rem;
    font-weight: 700;
    color: var(--text-main);
  }
  .notes-content {
    padding: 1.5rem;
    background: var(--bg-main);
    border-radius: 1rem;
    line-height: 1.6;
  }

  /* 過濾器樣式 */
  .filter-section {
    margin: 0.5rem 0 1.5rem 0;
    z-index: 10;
  }

  .filter-glass-container {
    background: rgba(255, 255, 255, 0.4);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border-radius: 20px;
    padding: 0.75rem;
    border: 1px solid rgba(255, 255, 255, 0.3);
    box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.07);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(body.dark-mode) .filter-glass-container {
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.3);
  }

  .filter-main-types {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    width: 100%;
  }

  .filter-stats-spacer {
    flex: 1;
    min-width: 1rem;
  }

  .filter-stats-badge {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.4rem 1rem;
    background: rgba(34, 197, 94, 0.08);
    border: 1px solid rgba(34, 197, 94, 0.15);
    border-radius: 12px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    animation: fadeIn 0.4s ease-out;
  }

  :global(body.dark-mode) .filter-stats-badge {
    background: rgba(34, 197, 94, 0.05);
    border-color: rgba(34, 197, 94, 0.1);
  }

  .filter-stats-badge:hover {
    transform: translateY(-1px);
    background: rgba(34, 197, 94, 0.12);
    box-shadow: 0 4px 12px rgba(34, 197, 94, 0.1);
  }

  .stats-icon {
    font-size: 1rem;
    filter: drop-shadow(0 0 2px rgba(34, 197, 94, 0.5));
  }

  .stats-content {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    font-weight: 600;
  }

  .stats-label {
    color: var(--text-muted);
    font-size: 0.75rem;
    font-weight: 500;
  }

  .stats-value {
    color: #16a34a;
    font-weight: 700;
  }

  :global(body.dark-mode) .stats-value {
    color: #4ade80;
  }

  .stats-value.win-rate {
    font-weight: 800;
    font-size: 0.95rem;
  }

  .stats-sep {
    color: rgba(34, 197, 94, 0.25);
    font-weight: 300;
  }

  .stats-color-groups {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    margin-left: 0.75rem;
    padding-left: 0.75rem;
    border-left: 1px solid rgba(34, 197, 94, 0.15);
  }

  .stats-color-dot {
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 50%;
  }

  .stats-color-dot.green {
    background-color: #22c55e;
  }
  .stats-color-dot.yellow {
    background-color: #eab308;
  }
  .stats-color-dot.red {
    background-color: #ef4444;
  }

  .stats-color-count {
    font-size: 0.8rem;
    color: var(--text-main);
    font-weight: 700;
    margin-right: 0.2rem;
  }

  .filter-type-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 1.2rem;
    border: none;
    background: transparent;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
    color: #475569;
    font-weight: 600;
    font-size: 0.95rem;
  }

  :global(body.dark-mode) .filter-type-btn {
    color: #94a3b8;
  }

  .filter-type-btn:hover {
    background: rgba(255, 255, 255, 0.5);
    transform: translateY(-1px);
  }

  :global(body.dark-mode) .filter-type-btn:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .filter-type-btn.active {
    background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  .divider {
    width: 1px;
    height: 24px;
    background: rgba(0, 0, 0, 0.1);
    margin: 0 0.25rem;
  }

  :global(body.dark-mode) .divider {
    background: rgba(255, 255, 255, 0.1);
  }

  .divider-horizontal {
    height: 1px;
    background: rgba(0, 0, 0, 0.05);
    margin: 0.75rem 0;
    width: 100%;
  }

  :global(body.dark-mode) .divider-horizontal {
    background: rgba(255, 255, 255, 0.05);
  }

  .sub-filter-scroll-wrapper {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    overflow-x: auto;
    scrollbar-width: none;
  }

  .sub-filter-scroll-wrapper::-webkit-scrollbar {
    display: none;
  }

  :global(body.dark-mode) .sub-filter-scroll-wrapper {
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }

  .sub-filter-container {
    display: flex;
    gap: 0.5rem;
    padding-bottom: 0.25rem;
  }

  .sub-filter-chip {
    white-space: nowrap;
    padding: 0.4rem 1rem;
    border-radius: 100px;
    border: 1px solid rgba(0, 0, 0, 0.1);
    background: white;
    font-size: 0.85rem;
    color: #64748b;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  :global(body.dark-mode) .sub-filter-chip {
    background: #1e293b;
    border-color: rgba(255, 255, 255, 0.1);
    color: #94a3b8;
  }

  .sub-filter-chip:hover {
    border-color: #6366f1;
    color: #6366f1;
  }

  .sub-filter-chip.active {
    background: #6366f1;
    border-color: #6366f1;
    color: white;
    box-shadow: 0 2px 8px rgba(99, 102, 241, 0.2);
  }

  .filter-date-row {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
    margin-bottom: 0.5rem;
  }

  .date-presets {
    display: flex;
    gap: 0.5rem;
  }

  .custom-date-inputs {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    animation: fadeIn 0.3s ease;
  }

  .date-input {
    padding: 0.4rem 0.6rem;
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.5);
    font-size: 0.9rem;
    color: #475569;
    font-family: inherit;
    outline: none;
    transition: all 0.2s;
  }

  .date-input:focus {
    border-color: #6366f1;
    background: white;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.1);
  }

  :global(body.dark-mode) .date-input {
    background: rgba(30, 41, 59, 0.5);
    border-color: rgba(255, 255, 255, 0.1);
    color: #cbd5e1;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(5px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .view-header-main {
    margin-bottom: 2rem;
  }

  .batch-viewer {
    margin-top: 1rem;
  }
  .view-header-main h1 {
    font-size: 2rem;
    font-weight: 800;
    color: var(--text-main);
    text-align: center;
    margin-bottom: 0.5rem;
    letter-spacing: -0.02em;
  }
  .header-info-line {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 1.5rem;
    margin-bottom: 1rem;
  }
  .account-badges {
    display: flex;
    gap: 0.5rem;
  }
  .acc-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 4px;
  }
  .acc-badge.type {
    background: #f1f5f9;
    color: #475569;
    border: 1px solid #e2e8f0;
  }
  .acc-badge.env.live {
    background: #fef2f2;
    color: #ef4444;
    border: 1px solid #fee2e2;
  }
  .acc-badge.env.demo {
    background: #f0fdf4;
    color: #22c55e;
    border: 1px solid #dcfce7;
  }
  .acc-badge.id {
    background: #fafafa;
    color: #94a3b8;
    border: 1px solid #f1f5f9;
  }
  .time-range-badge {
    font-size: 0.85rem;
    color: var(--text-muted);
    font-weight: 600;
    background: var(--bg-main);
    padding: 4px 12px;
    border-radius: 99px;
    border: 1px solid var(--border-color);
  }
  .batch-meta {
    text-align: center;
    color: #94a3b8;
    margin-bottom: 3rem;
    font-weight: 500;
  }

  /* Timeline */
  .timeline {
    border-left: 2px dashed #e2e8f0;
    padding-left: 1.5rem;
    position: relative;
    margin-left: 2rem;
  }
  .day-group {
    margin-bottom: 4rem;
    position: relative;
  }
  .date-header {
    position: absolute;
    left: -1.5rem;
    top: 0;
    transform: translateX(-50%);
    z-index: 10;
  }
  .date-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #6366f1;
    color: white;
    padding: 0.4rem 1rem;
    border-radius: 20px;
    font-weight: 700;
    font-size: 0.85rem;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3);
    white-space: nowrap;
    line-height: 1;
  }

  /* Day Card Container */
  .day-card-container {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 1.5rem;
    background: var(--card-bg);
    padding: 1.5rem;
    border-radius: 20px;
    border: 1px solid var(--border-color);
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.03);
  }

  .plan-column,
  .trade-column {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .plan-column {
    border-right: 1px dashed var(--border-color);
    padding-right: 1.5rem;
  }

  /* Cards */
  .plan-item-card,
  .trade-item-card {
    background: var(--card-bg);
    border-radius: 12px;
    padding: 1.25rem;
    border: 1px solid var(--border-color);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.02);
    position: relative;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Ongoing Trade Animation */
  @keyframes float {
    0% {
      transform: translateY(0px);
    }
    50% {
      transform: translateY(-6px);
    }
    100% {
      transform: translateY(0px);
    }
  }

  @keyframes pulse-bg {
    0% {
      opacity: 0.6;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: 0.6;
    }
  }

  .trade-item-card.is-ongoing {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.08) 0%,
      rgba(168, 85, 247, 0.08) 100%
    ) !important;
    border-color: rgba(99, 102, 241, 0.4) !important;
    border-style: solid;
    animation: float 4s ease-in-out infinite;
    box-shadow: 0 15px 35px rgba(99, 102, 241, 0.12);
    z-index: 5;
  }

  .trade-item-card.is-ongoing::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: radial-gradient(circle at top right, rgba(99, 102, 241, 0.1), transparent 70%);
    pointer-events: none;
    animation: pulse-bg 3s ease-in-out infinite;
  }

  :global(body.dark-mode) .trade-item-card.is-ongoing {
    background: linear-gradient(
      135deg,
      rgba(99, 102, 241, 0.15) 0%,
      rgba(168, 85, 247, 0.15) 100%
    ) !important;
    border-color: rgba(99, 102, 241, 0.5) !important;
    box-shadow: 0 15px 40px rgba(0, 0, 0, 0.4);
  }

  .trade-item-card.tag-green {
    border-left: 5px solid #22c55e;
  }
  .trade-item-card.tag-yellow {
    border-left: 5px solid #eab308;
  }
  .trade-item-card.tag-red {
    border-left: 5px solid #ef4444;
  }

  .item-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }
  .item-type {
    font-size: 0.75rem;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
  }
  .symbol-inline-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 800;
    color: var(--text-main);
    padding: 2px 6px;
    background: var(--nav-group-bg);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    line-height: 1;
  }


  .mini-notes {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border-color);
  }
  .mini-note-item {
    font-size: 0.8rem;
    color: var(--text-main);
    line-height: 1.4;
    display: flex;
    align-items: flex-start;
    gap: 0.4rem;
    margin-bottom: 0.3rem;
  }
  .note-session {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    font-weight: 800;
    padding: 2px 4px;
    border-radius: 3px;
    color: white;
    min-width: 1.2rem;
    text-align: center;
    flex-shrink: 0;
    line-height: 1;
  }
  .note-session.asian {
    background: #3b82f6;
  }
  .note-session.european {
    background: #d97706;
  }
  .note-session.us {
    background: #dc2626;
  }
  .simple-notes {
    font-size: 0.8rem;
    color: #64748b;
    margin-top: 0.5rem;
    font-style: italic;
    white-space: pre-wrap;
  }

  /* Trade Mini */
  .trade-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .session-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    background: #e2e8f0;
    color: #475569;
    line-height: 1;
  }
  .session-tag.asian {
    background: #dbeafe;
    color: #1e40af;
  }
  .session-tag.european {
    background: #fef9c3;
    color: #854d0e;
  }
  .session-tag.us {
    background: #fee2e2;
    color: #991b1b;
  }

  .side-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .side-tag.long {
    background: #fee2e2;
    color: #991b1b;
  }
  .side-tag.short {
    background: #dcfce7;
    color: #166534;
  }

  .strategy-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    color: white;
    line-height: 1;
  }
  .strategy-tag.expert {
    background: #059669;
  }
  .strategy-tag.elite {
    background: #1e3a8a;
  }
  .strategy-tag.legend {
    background: #78350f;
  }

  .trade-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .color-tags-static {
    display: flex;
    gap: 4px;
  }
  .color-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
  }
  .color-dot.active.green {
    background: #22c55e;
    border-color: #16a34a;
  }
  .color-dot.active.yellow {
    background: #eab308;
    border-color: #ca8a04;
  }
  .color-dot.active.red {
    background: #ef4444;
    border-color: #dc2626;
  }

  .pnl-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 1rem;
    font-weight: 900;
    padding: 4px 10px;
    border-radius: 8px;
    line-height: 1;
  }
  .pnl-tag.profit {
    background: #f0fdf4;
    color: #16a34a;
  }
  .pnl-tag.loss {
    background: #fef2f2;
    color: #dc2626;
  }

  .trade-details-shared {
    margin-top: 1rem;
  }
  .detail-row {
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
    align-items: center;
    margin-bottom: 0.5rem;
  }
  .info-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .info-group strong {
    color: var(--text-main);
  }
  .info-group .bullet {
    color: var(--primary);
    font-weight: 800;
  }
  .info-group .rr.profit {
    color: #10b981;
  }
  .info-group .rr.loss {
    color: #ef4444;
  }
  .arrow {
    color: #cbd5e1;
  }
  .trade-time-shared {
    font-size: 0.75rem;
    color: #94a3b8;
  }
  .duration-text {
    font-weight: 600;
    color: #64748b;
    margin-left: 0.5rem;
  }

  /* Gallery */
  .mini-gallery-shared {
    display: flex;
    gap: 0.5rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #f1f5f9;
  }
  .mini-img {
    width: 40px;
    height: 40px;
    border-radius: 4px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    cursor: pointer;
  }
  .mini-img img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .more-imgs {
    background: #f1f5f9;
    color: #64748b;
    font-size: 0.7rem;
    font-weight: 700;
    width: 40px;
    height: 40px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid #e2e8f0;
  }

  .empty-placeholder-shared {
    text-align: center;
    padding: 2rem;
    background: var(--nav-group-bg);
    border: 1px dashed var(--border-color);
    border-radius: 12px;
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  @media (max-width: 900px) {
    .day-card-container {
      grid-template-columns: 1fr;
    }
    .plan-column {
      border-right: none;
      border-bottom: 1px dashed #e2e8f0;
      padding-right: 0;
      padding-bottom: 1.5rem;
    }
    .timeline {
      margin-left: 1rem;
    }
  }

  .plan-item-card.clickable,
  .trade-item-card.clickable {
    cursor: pointer;
    transition: all 0.2s;
  }
  .plan-item-card.clickable:hover,
  .trade-item-card.clickable:hover {
    border-color: #6366f1;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }

  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.85);
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(8px);
    overflow: hidden;
  }
  .image-modal-content {
    position: relative;
    width: 100vw;
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }
  .zoom-container {
    transition: transform 0.05s ease-out;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
  }
  .modal-img {
    max-width: 90vw;
    max-height: 90vh;
    object-fit: contain;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
    border-radius: 4px;
    pointer-events: none;
  }
  .image-modal-close {
    position: absolute;
    top: 20px;
    right: 20px;
    color: white;
    font-size: 2rem;
    background: rgba(0, 0, 0, 0.5);
    border: none;
    cursor: pointer;
    width: 50px;
    height: 50px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
    transition: all 0.2s;
  }
  .image-modal-close:hover {
    background: rgba(255, 255, 255, 0.2);
    transform: rotate(90deg);
  }
  .image-modal-caption {
    position: absolute;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    color: white;
    background: rgba(0, 0, 0, 0.6);
    padding: 8px 20px;
    border-radius: 99px;
    font-size: 0.9rem;
    font-weight: 500;
    pointer-events: none;
    white-space: nowrap;
    backdrop-filter: blur(4px);
  }
  .status-box {
    text-align: center;
    padding: 4rem 2rem;
  }
  /* Card Notes Styling */
  .card-notes-section {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px dashed var(--border-color);
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }

  .note-block {
    font-size: 0.9rem;
    color: var(--text-main);
  }

  .note-label {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-muted);
    margin-bottom: 0.25rem;
    opacity: 0.8;
  }

  .note-content {
    line-height: 1.5;
    background: var(--nav-group-bg);
    padding: 0.5rem 0.75rem;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    color: var(--text-main) !important;
  }
  
  .note-content :global(*) {
    color: var(--text-main) !important;
    background-color: transparent !important;
  }

  .note-content :global(p) {
    margin: 0.25rem 0;
  }
  .note-content :global(p:first-child) {
    margin-top: 0;
  }
  .note-content :global(p:last-child) {
    margin-bottom: 0;
  }
  .note-content :global(img) {
    max-width: 100%;
    border-radius: 4px;
    margin-top: 0.5rem;
  }

  .shared-top-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-color);
  }

  .logo-area {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .logo-image-container {
    height: 60px;
    width: 200px;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }

  .brand-logo-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center 48%;
    pointer-events: none;
    transform: scale(1.1);
  }

  .public-badge-small {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-muted);
    background: var(--nav-group-bg);
    padding: 0.35rem 0.75rem;
    border-radius: 99px;
    border: 1px solid var(--border-color);
  }

  .theme-toggle-btn {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 10px;
    width: 38px;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 1.2rem;
    transition: all 0.2s;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }
  .theme-toggle-btn:hover {
    transform: rotate(15deg);
    background: var(--nav-group-bg);
  }
</style>
