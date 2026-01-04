<script>
  import { onMount, onDestroy } from 'svelte';
  import { navigate, Link } from 'svelte-routing';
  import { tradesAPI, dailyPlansAPI, imagesAPI } from '../lib/api';
  import { selectedSymbol, selectedAccountId, accounts } from '../lib/stores';
  import { MARKET_SESSIONS, SYMBOLS, TIMEFRAMES } from '../lib/constants';
  import { determineMarketSession, getStrategyLabel } from '../lib/utils';
  import { accountsAPI } from '../lib/api';
  import AccountModal from './AccountModal.svelte';

  let groupedData = [];
  let loading = true;
  let todayString = new Date().toLocaleDateString('en-CA'); // 使用 YYYY-MM-DD 格式的本地日期
  let selectedImage = null;
  let isSyncing = false;
  let showAccountModal = false;
  let pollingInterval;

  // 追蹤當前選取的帳號詳情
  $: currentAccount = $accounts.find(a => a.id === $selectedAccountId);

  function navigateWithScroll(path) {
    sessionStorage.setItem('home_scroll_pos', window.scrollY);
    navigate(path);
  }

  async function handleSync() {
    if (!$selectedAccountId || isSyncing) return;
    isSyncing = true;
    try {
      await accountsAPI.sync($selectedAccountId);
      // Refresh both account info (for storage usage) and data
      const accountsRes = await accountsAPI.getAll();
      accounts.set(accountsRes.data);
      await loadData();
    } catch (error) {
      console.error('Sync failed:', error);
      alert('同步失敗: ' + (error.response?.data?.error || error.message));
    } finally {
      isSyncing = false;
    }
  }

  async function loadData(silent = false) {
    try {
      if (!silent) loading = true;
      const symbol = $selectedSymbol;

      // 更新今天日期文字
      todayString = new Date().toISOString().slice(0, 10);

      // 獲取最近 20 天的規劃和最近 50 筆交易
      const [plansRes, tradesRes] = await Promise.all([
        dailyPlansAPI.getAll({ account_id: $selectedAccountId, symbol, page_size: 20 }),
        tradesAPI.getAll({ account_id: $selectedAccountId, symbol, page_size: 50 }),
      ]);

      const plans = (Array.isArray(plansRes.data) ? plansRes.data : plansRes.data?.data) || [];
      const trades = (Array.isArray(tradesRes.data) ? tradesRes.data : tradesRes.data?.data) || [];

      console.log('Loaded plans:', plans);
      console.log('Loaded trades:', trades);

      // 按日期分組 (YYYY-MM-DD)
      const dateMap = {};

      // 強制推入今天的日期，確保最上面有東西
      dateMap[todayString] = { date: todayString, plans: [], groupedTrades: [] };

      plans.forEach(plan => {
        try {
          if (!plan.plan_date) return;
          const date = new Date(plan.plan_date).toISOString().slice(0, 10);
          if (!dateMap[date]) dateMap[date] = { date, plans: [], groupedTrades: [] };
          dateMap[date].plans.push(plan);
        } catch (e) {
          console.warn('Skipping invalid plan:', plan, e);
        }
      });

      trades.forEach(trade => {
        try {
          if (!trade.entry_time) return; // Skip if no entry time
          const dateObj = new Date(trade.entry_time);
          if (isNaN(dateObj.getTime())) return; // Skip invalid date

          const date = dateObj.toISOString().slice(0, 10);
          if (!dateMap[date]) dateMap[date] = { date, plans: [], groupedTrades: [] };

          // 尋找是否已有相同開倉時間的群組
          const entryTimeKey = trade.entry_time;
          let timeGroup = dateMap[date].groupedTrades.find(g => g.entry_time === entryTimeKey);

          if (!timeGroup) {
            timeGroup = {
              entry_time: entryTimeKey,
              trades: [],
              summary: {
                totalPnl: 0,
                totalLot: 0,
                symbol: trade.symbol,
                entry_price: trade.entry_price,
                side: trade.side,
              },
            };
            dateMap[date].groupedTrades.push(timeGroup);
          }
          timeGroup.trades.push(trade);
          timeGroup.summary.totalPnl += trade.pnl || 0;
          timeGroup.summary.totalLot += trade.lot_size || 0;
        } catch (e) {
          console.warn('Skipping invalid trade:', trade, e);
        }
      });

      // 轉換為陣列並排序（日期降序，群組內按時間排序通常已由 API 處理）
      groupedData = Object.values(dateMap).sort((a, b) => b.date.localeCompare(a.date));

      // 針對組合單內的成員排序 (先平倉的在上面)
      groupedData.forEach(day => {
        day.groupedTrades.forEach(group => {
          if (group.trades.length > 1) {
            group.trades.sort((a, b) => new Date(a.exit_time || 0) - new Date(b.exit_time || 0));
          }
        });
      });
      console.log('Final groupedData:', groupedData);
    } catch (error) {
      console.error('載入首頁資料失敗:', error);
    } finally {
      loading = false;
    }
  }

  // 監聽品種或帳號變更
  $: if ($selectedSymbol && $selectedAccountId) {
    loadData();
  } else if ($accounts && $accounts.length === 0) {
    // 如果完全沒有帳號，停止載入狀態以顯示「新增帳號」UI
    loading = false;
  }

  onMount(async () => {
    // 獲取最新帳號清單，確保首頁能顯示狀態
    try {
      const res = await accountsAPI.getAll();
      accounts.set(res.data);
    } catch (e) {
      console.error('Failed to pre-fetch accounts:', e);
    }

    // 初始載入由下面的 $: 響應式語句處理
    // 設定每 30 秒自動輪詢一次資料
    pollingInterval = setInterval(() => {
      if (!$selectedAccountId) return;
      console.log('Auto-polling data...');
      loadData(true); // silent update
    }, 30000);

    // Restore scroll position
    const savedScrollPos = sessionStorage.getItem('home_scroll_pos');
    if (savedScrollPos) {
      setTimeout(() => {
        window.scrollTo(0, parseInt(savedScrollPos));
        sessionStorage.removeItem('home_scroll_pos');
      }, 500); // Wait for data to render
    }
  });

  onDestroy(() => {
    if (pollingInterval) {
      clearInterval(pollingInterval);
    }
  });

  function formatDate(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return dateString;
    return date.toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  }

  function calculateDuration(start, end) {
    if (!start || !end) return '';
    const startTime = new Date(start).getTime();
    const endTime = new Date(end).getTime();
    if (isNaN(startTime) || isNaN(endTime)) return '';

    const diffMs = endTime - startTime;
    if (diffMs < 0) return '';

    const diffSec = Math.floor(diffMs / 1000);
    const hours = Math.floor(diffSec / 3600);
    const minutes = Math.floor((diffSec % 3600) / 60);
    const seconds = diffSec % 60;

    const parts = [];
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`);

    return parts.join(' ');
  }

  function formatBytes(bytes, decimals = 2) {
    if (!bytes || bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }

  function formatDay(dateString) {
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return dateString;
    const weekdays = ['日', '一', '二', '三', '四', '五', '六'];
    return `${date.getFullYear()}/${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')} (週${weekdays[date.getDay()]})`;
  }

  function getMarketSessionLabel(trade) {
    let session = trade.market_session;
    // 如果資料庫中沒有時段資料，根據時間即時計算
    if (!session && trade.entry_time) {
      session = determineMarketSession(trade.entry_time);
    }
    return MARKET_SESSIONS.find(s => s.value === session)?.label || session || '未設定';
  }

  function parseJSONSafe(str, defaultValue) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  function openImageModal(imagePath) {
    if (!imagePath) return;
    selectedImage = imagePath.startsWith('http') ? imagePath : imagesAPI.getUrl(imagePath);
  }

  function closeImageModal() {
    selectedImage = null;
  }

  async function deleteTradeGroup(timeGroup) {
    if (!confirm(`確定要刪除這組交易嗎？(共 ${timeGroup.trades.length} 筆)`)) return;
    try {
      // 依序刪除所有該群組的交易
      for (const trade of timeGroup.trades) {
        await tradesAPI.delete(trade.id);
      }
      loadData();
    } catch (error) {
      alert('刪除失敗');
    }
  }

  function sanitizeTradePayload(fullTrade, newColor) {
    const payload = {
      ...fullTrade,
      color_tag: newColor !== undefined ? newColor || '' : fullTrade.color_tag || '',
      account_id: fullTrade.account_id,
      trade_type: fullTrade.trade_type || 'actual',
      symbol: fullTrade.symbol,
      side: fullTrade.side,
      entry_time: fullTrade.entry_time,
      entry_reason: fullTrade.entry_reason || '',
      exit_reason: fullTrade.exit_reason || '',
      entry_strategy: fullTrade.entry_strategy || '',
      entry_signals: fullTrade.entry_signals || '',
      entry_checklist: fullTrade.entry_checklist || '',
      entry_pattern: fullTrade.entry_pattern || '',
      trend_analysis: fullTrade.trend_analysis || '',
      entry_timeframe: fullTrade.entry_timeframe || '',
      trend_type: fullTrade.trend_type || '',
      market_session: fullTrade.market_session || '',
      legend_king_htf: fullTrade.legend_king_htf || '',
      legend_king_image: fullTrade.legend_king_image || '',
      legend_king_image_original: fullTrade.legend_king_image_original || '',
      legend_htf: fullTrade.legend_htf || '',
      legend_htf_image: fullTrade.legend_htf_image || '',
      legend_htf_image_original: fullTrade.legend_htf_image_original || '',
      legend_de_htf: fullTrade.legend_de_htf || '',
      entry_strategy_image: fullTrade.entry_strategy_image || '',
      entry_strategy_image_original: fullTrade.entry_strategy_image_original || '',
      notes: fullTrade.notes || '',
      timezone_offset:
        fullTrade.timezone_offset !== null && fullTrade.timezone_offset !== undefined
          ? fullTrade.timezone_offset
          : 0,
    };

    if (fullTrade.images) {
      payload.images = fullTrade.images.map(img => ({
        image_type: img.image_type,
        image_path: img.image_path,
      }));
    }

    if (fullTrade.tags) {
      payload.tags = fullTrade.tags.map(t => (typeof t === 'object' ? t.name : t));
    }

    return payload;
  }

  async function toggleColorTag(trade, color) {
    const newColor = trade.color_tag === color ? '' : color;
    try {
      const fullTradeRes = await tradesAPI.getOne(trade.id);
      const fullTrade = fullTradeRes.data;
      const payload = sanitizeTradePayload(fullTrade, newColor);
      await tradesAPI.update(trade.id, payload);
      trade.color_tag = newColor;
      groupedData = groupedData;
    } catch (e) {
      console.error('Failed to update color tag', e);
      const errMsg = e.response?.data?.error || e.message || 'Unknown error';
      alert(`更新顏色標記失敗: ${errMsg}`);
    }
  }

  async function toggleColorTagForGroup(timeGroup, color) {
    const firstTrade = timeGroup.trades[0];
    if (!firstTrade) return;
    const newColor = firstTrade.color_tag === color ? '' : color;

    try {
      // Update all trades in the group
      for (const trade of timeGroup.trades) {
        const fullTradeRes = await tradesAPI.getOne(trade.id);
        const fullTrade = fullTradeRes.data;
        const payload = sanitizeTradePayload(fullTrade, newColor);
        await tradesAPI.update(trade.id, payload);
        trade.color_tag = newColor;
      }
      groupedData = groupedData;
    } catch (e) {
      console.error('Failed to update group color tag', e);
      const errMsg = e.response?.data?.error || e.message || 'Unknown error';
      alert(`更新組合單顏色標記失敗: ${errMsg}`);
    }
  }

  async function deletePlan(id) {
    if (!confirm('確定要刪除此規劃嗎？')) return;
    try {
      await dailyPlansAPI.delete(id);
      loadData();
    } catch (error) {
      alert('刪除失敗');
    }
  }
</script>

<div class="timeline-container">
  <!-- 頂部快速操作區 -->
  <div class="top-actions-bar">
    <div class="account-status-bar">
      {#if currentAccount}
        <div class="status-badges">
          <span
            class="badge {currentAccount.type === 'local'
              ? 'badge-info'
              : currentAccount.type === 'metatrader'
                ? 'badge-mt5'
                : 'badge-ctrader'}"
          >
            {currentAccount.type === 'local'
              ? '本地帳號'
              : currentAccount.type === 'metatrader'
                ? 'MetaTrader 5'
                : 'cTrader'}
          </span>
          <span
            class="badge {currentAccount.status === 'active' ? 'badge-success' : 'badge-danger'}"
          >
            {currentAccount.status}
          </span>
          <span class="badge badge-utc"
            >UTC{currentAccount.timezone_offset >= 0
              ? '+'
              : ''}{currentAccount.timezone_offset}</span
          >
          <div class="account-details-inline">
            <span class="storage-info">
              <span class="label">📊 圖文佔用：</span>
              <strong>{formatBytes(currentAccount.storage_usage)}</strong>
            </span>
            {#if currentAccount.type === 'ctrader'}
              <span class="login-id">Login ID: {currentAccount.ctrader_account_id}</span>
            {:else}
              <span class="login-id">ID: {currentAccount.mt5_account_id}</span>
            {/if}
          </div>
          {#if currentAccount.type !== 'local'}
            <div class="sync-status-info">
              <span class="sync-badge {currentAccount.sync_status}"
                >{currentAccount.sync_status}</span
              >
              {#if currentAccount.last_synced_at}
                <span class="sync-time">
                  最後同步: {new Date(currentAccount.last_synced_at).toLocaleString('zh-TW', {
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                  })}
                </span>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>
    <div class="quick-btns">
      {#if currentAccount && currentAccount.type !== 'local'}
        <button
          class="small-action-btn sync {isSyncing ? 'syncing' : ''}"
          on:click={handleSync}
          disabled={isSyncing}
        >
          <span class="btn-icon">{isSyncing ? '⏳' : '🔄'}</span>
          {isSyncing ? '同步中...' : '手動同步'}
        </button>
      {/if}
      <button
        class="small-action-btn plan"
        data-testid="add-plan-btn"
        on:click={() => navigate('/plans/new?symbol=' + $selectedSymbol)}
      >
        <span class="btn-icon">📋</span> 新增規劃
      </button>
      <button
        class="small-action-btn trade"
        data-testid="add-trade-btn"
        on:click={() => navigate('/new?symbol=' + $selectedSymbol)}
      >
        <span class="btn-icon">💰</span> 新增交易
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading-state">
      <div class="loader"></div>
      <p>正在載入時光機資料...</p>
    </div>
  {:else if $accounts.length === 0}
    <div class="empty-account-state">
      <div class="welcome-card">
        <div class="welcome-icon">🚀</div>
        <p class="description">
          您尚未建立任何交易帳號。請先建立一個交易帳號來開始記錄您的交易旅程！
        </p>
        <button class="btn btn-primary btn-lg" on:click={() => (showAccountModal = true)}>
          <span class="icon">➕</span> 立即建立交易帳號
        </button>
      </div>
    </div>
  {:else if groupedData.length === 0}
    <div class="empty-state">
      <div class="empty-icon">🏜️</div>
      <p>這裡空空如也，開始記錄您的第一筆 {$selectedSymbol} 規劃或交易吧！</p>
    </div>
  {:else}
    <div class="timeline">
      {#each groupedData as group}
        <div class="day-wrapper">
          <div class="day-marker">
            <div class="date-tag">{formatDay(group.date)}</div>
          </div>

          <div class="day-card-container">
            <!-- 左側規劃 -->
            <div class="plan-column">
              {#if group.plans.length > 0}
                {#each group.plans as plan}
                  {@const trendData = parseJSONSafe(plan.trend_analysis, {})}
                  {@const isUnified = plan.market_session === 'all'}
                  <div class="plan-item-card" on:click={() => navigate(`/plans/edit/${plan.id}`)}>
                    <div class="item-header">
                      <span class="item-type">📌 盤面規劃</span>
                      <button
                        class="icon-btn delete"
                        on:click|stopPropagation={() => deletePlan(plan.id)}>🗑️</button
                      >
                    </div>

                    {#if isUnified}
                      <div class="mini-progression">
                        {#each TIMEFRAMES as tf}
                          {@const asianTrend = trendData.asian?.trends?.[tf]}
                          {@const europeanTrend = trendData.european?.trends?.[tf]}
                          {@const usTrend = trendData.us?.trends?.[tf]}
                          {#if asianTrend?.direction || europeanTrend?.direction || usTrend?.direction}
                            <div class="tf-row">
                              <span class="tf-name">{tf}:</span>
                              <div class="tf-steps">
                                {#each MARKET_SESSIONS as session, i}
                                  {@const trend = trendData[session.value]?.trends?.[tf]}
                                  <span class="mini-step {trend?.direction || 'na'}">
                                    {session.label}
                                    {trend?.direction === 'long'
                                      ? '多'
                                      : trend?.direction === 'short'
                                        ? '空'
                                        : 'NA'}
                                  </span>
                                  {#if i < MARKET_SESSIONS.length - 1}
                                    <span class="step-arrow">=></span>
                                  {/if}
                                {/each}
                              </div>
                            </div>
                          {/if}
                        {/each}
                      </div>
                      {#if trendData.asian?.notes || trendData.european?.notes || trendData.us?.notes}
                        <div class="mini-notes">
                          <div class="mini-notes-title">📝 備註事項</div>
                          {#if trendData.asian?.notes}<div class="mini-note-item">
                              <span class="note-session asian">亞</span>
                              {trendData.asian.notes}
                            </div>{/if}
                          {#if trendData.european?.notes}<div class="mini-note-item">
                              <span class="note-session european">歐</span>
                              {trendData.european.notes}
                            </div>{/if}
                          {#if trendData.us?.notes}<div class="mini-note-item">
                              <span class="note-session us">美</span>
                              {trendData.us.notes}
                            </div>{/if}
                        </div>
                      {/if}
                    {:else}
                      <p class="simple-notes">{plan.notes || '無備註'}</p>
                    {/if}
                  </div>
                {/each}
              {:else}
                <div
                  class="empty-placeholder dash-plan"
                  on:click={() =>
                    navigate(`/plans/new?date=${group.date}&symbol=${$selectedSymbol}`)}
                >
                  <div class="plus-icon">➕</div>
                  <span>新增規劃</span>
                </div>
              {/if}
            </div>

            <!-- 右側交易 -->
            <div class="trade-column">
              {#if group.groupedTrades.length > 0}
                <div class="trades-stack">
                  {#each group.groupedTrades as timeGroup}
                    {#if timeGroup.trades.length > 1}
                      <!-- 組合單 (多筆部分的平倉) -->
                      <div
                        class="trade-time-group is-multi {timeGroup.trades[0].color_tag
                          ? `tag-${timeGroup.trades[0].color_tag}`
                          : ''}"
                        on:click={() => navigateWithScroll(`/edit/${timeGroup.trades[0].id}`)}
                      >
                        <div class="group-header">
                          <div class="group-meta">
                            <span class="multi-indicator">📦 組合單</span>
                            <span class="symbol-inline-tag">{timeGroup.summary.symbol}</span>
                            <span class="side-tag {timeGroup.summary.side}"
                              >{timeGroup.summary.side === 'long' ? '📈 做多' : '📉 做空'}</span
                            >
                            <div class="info-group">
                              <span class="label">進場</span>
                              <strong>{timeGroup.summary.entry_price}</strong>
                            </div>
                            <div class="info-group">
                              <span class="label">平均平倉</span>
                              <strong
                                >{(
                                  timeGroup.trades.reduce(
                                    (acc, t) => acc + (t.exit_price || 0),
                                    0
                                  ) / timeGroup.trades.length
                                ).toFixed(2)}</strong
                              >
                            </div>
                            <div class="info-group">
                              <span class="label">總手數</span>
                              <strong>{timeGroup.summary.totalLot.toFixed(2)}</strong>
                            </div>
                          </div>
                          <div class="group-pnl">
                            <div class="color-tags" on:click|stopPropagation>
                              <button
                                class="color-btn green {timeGroup.trades[0].color_tag === 'green'
                                  ? 'active'
                                  : ''}"
                                on:click={() => toggleColorTagForGroup(timeGroup, 'green')}
                              ></button>
                              <button
                                class="color-btn yellow {timeGroup.trades[0].color_tag === 'yellow'
                                  ? 'active'
                                  : ''}"
                                on:click={() => toggleColorTagForGroup(timeGroup, 'yellow')}
                              ></button>
                              <button
                                class="color-btn red {timeGroup.trades[0].color_tag === 'red'
                                  ? 'active'
                                  : ''}"
                                on:click={() => toggleColorTagForGroup(timeGroup, 'red')}
                              ></button>
                            </div>
                            <span
                              class="pnl-tag {timeGroup.summary.totalPnl >= 0 ? 'profit' : 'loss'}"
                            >
                              {timeGroup.summary.totalPnl >= 0
                                ? '+'
                                : ''}{timeGroup.summary.totalPnl?.toFixed?.(2) || '0.00'}
                            </span>
                            <button
                              class="icon-btn delete"
                              on:click|stopPropagation={() => deleteTradeGroup(timeGroup)}
                              >🗑️</button
                            >
                          </div>
                        </div>

                        <div class="group-partial-closes">
                          {#each timeGroup.trades as trade}
                            <div class="partial-close-row">
                              <div class="info-group">
                                <span class="label"
                                  >{formatDate(trade.exit_time || trade.entry_time).split(
                                    ' '
                                  )[1]}</span
                                >
                                <span class="label">平倉</span>
                                <strong>{trade.exit_price || '-'}</strong>
                                <span class="label">({trade.lot_size} 手)</span>
                              </div>
                              <div class="info-group">
                                {#if trade.rr_ratio}
                                  <span class="label">風報比</span>
                                  <strong class="rr {trade.rr_ratio >= 0 ? 'profit' : 'loss'}"
                                    >{trade.rr_ratio.toFixed(2)}</strong
                                  >
                                {/if}
                                {#if trade.exit_time}
                                  <span class="label">持單時間</span>
                                  <strong
                                    >{calculateDuration(trade.entry_time, trade.exit_time)}</strong
                                  >
                                {/if}
                                <span class="partial-pnl {trade.pnl >= 0 ? 'profit' : 'loss'}"
                                  >{trade.pnl >= 0 ? '+' : ''}{trade.pnl?.toFixed(2)}</span
                                >
                              </div>
                              {#if trade.ticket}<span class="partial-ticket">#{trade.ticket}</span
                                >{/if}
                            </div>
                          {/each}
                        </div>
                      </div>
                    {:else}
                      <!-- 一般單 (單筆進出) -->
                      {@const trade = timeGroup.trades[0]}
                      <div
                        class="trade-item-card {trade.color_tag ? `tag-${trade.color_tag}` : ''}"
                        on:click={() => navigateWithScroll(`/edit/${trade.id}`)}
                      >
                        <div class="item-header">
                          <div class="trade-meta">
                            <span class="symbol-inline-tag">{trade.symbol}</span>
                            <span
                              class="session-tag {trade.market_session ||
                                determineMarketSession(trade.entry_time)}"
                              >{getMarketSessionLabel(trade)}</span
                            >
                            {#if trade.entry_strategy}<span
                                class="strategy-tag {trade.entry_strategy}"
                                >{getStrategyLabel(trade.entry_strategy)}</span
                              >{/if}
                            <span class="side-tag {trade.side}"
                              >{trade.side === 'long' ? '📈 做多' : '📉 做空'}</span
                            >
                            {#if trade.ticket}<span class="ticket-tag">#{trade.ticket}</span>{/if}
                          </div>
                          <div class="trade-right">
                            <div class="color-tags" on:click|stopPropagation>
                              <button
                                class="color-btn green {trade.color_tag === 'green'
                                  ? 'active'
                                  : ''}"
                                on:click={() => toggleColorTag(trade, 'green')}
                              ></button>
                              <button
                                class="color-btn yellow {trade.color_tag === 'yellow'
                                  ? 'active'
                                  : ''}"
                                on:click={() => toggleColorTag(trade, 'yellow')}
                              ></button>
                              <button
                                class="color-btn red {trade.color_tag === 'red' ? 'active' : ''}"
                                on:click={() => toggleColorTag(trade, 'red')}
                              ></button>
                            </div>
                            {#if trade.pnl !== null && trade.pnl !== undefined}
                              <span class="pnl-tag {trade.pnl >= 0 ? 'profit' : 'loss'}">
                                {trade.pnl >= 0 ? '+' : ''}{typeof trade.pnl === 'number'
                                  ? trade.pnl.toFixed(2)
                                  : trade.pnl}
                              </span>
                            {/if}
                            <button
                              class="icon-btn delete"
                              on:click|stopPropagation={() => deleteTradeGroup(timeGroup)}
                              >🗑️</button
                            >
                          </div>
                        </div>

                        <div class="trade-details">
                          <div class="detail-row">
                            <!-- 第一組：價格資訊 -->
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

                            <!-- 第二組：績效資訊 -->
                            <div class="info-group">
                              <span class="label">子彈</span>
                              <strong class="bullet">{trade.bullet_size?.toFixed(1) || '-'}</strong>
                              {#if trade.bullet_size > 0 && (trade.rr_ratio || trade.rr_ratio === 0)}
                                <span class="label">風報比</span>
                                <strong class="rr {trade.rr_ratio >= 0 ? 'profit' : 'loss'}"
                                  >{trade.rr_ratio.toFixed(2)}</strong
                                >
                              {/if}
                              <span class="label">手數</span>
                              <strong>{trade.lot_size}</strong>
                              {#if trade.exit_time}
                                <span class="label">持單時間</span>
                                <strong class="duration-text"
                                  >{calculateDuration(trade.entry_time, trade.exit_time)}</strong
                                >
                              {/if}
                            </div>
                          </div>
                          <div class="trade-time">
                            {formatDate(trade.entry_time).split(' ')[1]} - {trade.exit_time
                              ? formatDate(trade.exit_time).split(' ')[1]
                              : '進行中'}
                          </div>
                        </div>

                        {#if trade.images && trade.images.length > 0}
                          <div class="mini-gallery">
                            {#each trade.images.slice(0, 3) as img}
                              <div
                                class="mini-img"
                                on:click|stopPropagation={() => openImageModal(img.image_path)}
                              >
                                <img src={imagesAPI.getUrl(img.image_path)} alt="trade" />
                              </div>
                            {/each}
                            {#if trade.images.length > 3}<div class="more-imgs">
                                +{trade.images.length - 3}
                              </div>{/if}
                          </div>
                        {/if}
                      </div>
                    {/if}
                  {/each}
                </div>
              {:else}
                <div
                  class="empty-placeholder dash-trade"
                  on:click={() => navigate(`/new?symbol=${$selectedSymbol}`)}
                >
                  <div class="plus-icon">➕</div>
                  <span>新增交易紀錄</span>
                </div>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<AccountModal
  bind:show={showAccountModal}
  on:success={async e => {
    const { accountId } = e.detail;
    // 自動選取新建立的帳號並整頁重整以確保所有元件同步
    selectedAccountId.set(parseInt(accountId));
    window.location.reload();
  }}
/>

{#if selectedImage}
  <div class="modal" on:click={closeImageModal}>
    <div class="modal-content" on:click|stopPropagation>
      <button class="modal-close" on:click={closeImageModal}>×</button>
      <img src={selectedImage} alt="全螢幕圖片" />
    </div>
  </div>
{/if}

<style>
  .timeline-container {
    padding-bottom: 5rem;
  }

  .empty-account-state {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 4rem 2rem;
    min-height: 60vh;
  }

  .welcome-card {
    background: white;
    padding: 3rem;
    border-radius: 24px;
    text-align: center;
    max-width: 500px;
    width: 100%;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.05);
    border: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.5rem;
  }

  /* Account Status Bar */
  .account-status-bar {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex: 1;
  }

  .status-badges {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .account-details-inline {
    display: flex;
    align-items: center;
    gap: 1rem;
    font-size: 0.8rem;
    color: var(--text-muted);
    background: #f8fafc;
    padding: 0.3rem 0.75rem;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
  }

  .storage-info .label {
    opacity: 0.7;
  }

  .login-id {
    font-family: 'JetBrains Mono', monospace;
    opacity: 0.8;
  }

  .sync-status-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding-left: 0.75rem;
    border-left: 1px solid #e2e8f0;
    margin-left: 0.25rem;
  }

  .sync-badge {
    text-transform: uppercase;
    font-size: 0.65rem;
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-weight: 800;
  }
  .sync-badge.idle {
    background: #f1f5f9;
    color: #64748b;
  }
  .sync-badge.syncing {
    background: #e0f2fe;
    color: #0369a1;
    animation: pulse 2s infinite;
  }
  .sync-badge.success {
    background: #dcfce7;
    color: #166534;
  }
  .sync-badge.failed {
    background: #fee2e2;
    color: #991b1b;
  }

  @keyframes pulse {
    0% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
    100% {
      opacity: 1;
    }
  }

  .sync-time {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 500;
  }

  .badge {
    padding: 0.3rem 0.75rem;
    border-radius: 10px;
    font-size: 0.75rem;
    font-weight: 700;
  }
  .badge-info {
    background: #e0f2fe;
    color: #0369a1;
  }
  .badge-mt5 {
    background: #f1f5f9;
    color: #475569;
    border: 1px solid #e2e8f0;
  }
  .badge-ctrader {
    background: #ecfdf5;
    color: #065f46;
  }
  .badge-success {
    background: #dcfce7;
    color: #166534;
  }
  .badge-danger {
    background: #fee2e2;
    color: #991b1b;
  }
  .badge-utc {
    background: #f8fafc;
    color: #64748b;
    border: 1px solid #e2e8f0;
  }

  .top-actions-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0 0.5rem 2rem; /* 左側 padding 2rem 以對齊下面卡片 */
    margin-bottom: 1.5rem;
    background: transparent;
    border: none;
    box-shadow: none;
  }

  .welcome-icon {
    font-size: 4rem;
    margin-bottom: 0.5rem;
  }

  .welcome-card h2 {
    font-size: 1.75rem;
    font-weight: 800;
    color: var(--text-main);
    line-height: 1.3;
  }

  .welcome-card p {
    color: var(--text-muted);
    font-size: 1.1rem;
    line-height: 1.6;
  }

  .timeline-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: -0.5rem;
    margin-bottom: 2rem;
  }

  .timeline-header h2 {
    font-size: 1.75rem;
    font-weight: 800;
    color: var(--text-main);
    background: linear-gradient(135deg, #6366f1, #8b5cf6);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .header-actions {
    display: flex;
    gap: 1rem;
  }

  .timeline {
    position: relative;
    padding-left: 2rem;
  }

  .timeline::before {
    content: '';
    position: absolute;
    left: 8px;
    top: 10px;
    bottom: 0;
    width: 2px;
    background: linear-gradient(to bottom, #e2e8f0, #e2e8f0 50%, transparent 50%);
    background-size: 1px 20px;
  }

  .day-wrapper {
    position: relative;
    margin-bottom: 3rem;
  }

  .day-marker {
    position: absolute;
    left: -42px;
    top: 0;
    z-index: 2;
  }

  .date-tag {
    background: #6366f1;
    color: white;
    padding: 0.4rem 1rem;
    border-radius: 20px;
    font-size: 0.85rem;
    font-weight: 700;
    white-space: nowrap;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3);
  }

  .day-card-container {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 1.5rem;
    background: white;
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
    border-right: 1px dashed #e2e8f0;
    padding-right: 1.5rem;
  }

  /* Card Items */
  .plan-item-card,
  .trade-item-card {
    background: white;
    border-radius: 12px;
    padding: 1.25rem;
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid var(--border-color);
    position: relative;
    overflow: hidden;
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

  .color-tags {
    display: flex;
    gap: 0.3rem;
    margin-right: 0.75rem;
  }

  .color-btn {
    width: 1rem;
    height: 1rem;
    border-radius: 50%;
    border: 1px solid #ddd;
    cursor: pointer;
    transition:
      transform 0.1s,
      border-color 0.1s;
    padding: 0;
  }
  .color-btn:hover {
    transform: scale(1.1);
  }
  .color-btn.active {
    border: 2px solid #333;
    transform: scale(1.1);
  }
  .color-btn.green {
    background-color: #22c55e;
  }
  .color-btn.yellow {
    background-color: #eab308;
  }
  .color-btn.red {
    background-color: #ef4444;
  }

  .trade-item-card:hover {
    border-color: #6366f1;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
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

  .icon-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: background 0.2s;
  }

  .icon-btn:hover {
    background: #fee2e2;
  }

  /* Plan Mini styles */
  .mini-progression {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 0.75rem;
  }

  .tf-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
  }

  .tf-name {
    font-weight: 700;
    color: #475569;
    width: 30px;
  }

  .tf-steps {
    display: flex;
    gap: 3px;
  }

  .mini-step {
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .mini-step.long {
    background: #fef2f2;
    color: #991b1b;
  }
  .mini-step.short {
    background: #f0fdf4;
    color: #166534;
  }

  .mini-notes {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid #edf2f7;
  }

  .mini-notes-title {
    font-size: 0.75rem;
    font-weight: 700;
    color: #64748b;
    margin-bottom: 0.4rem;
  }

  .mini-note-item {
    font-size: 0.8rem;
    color: #4a5568;
    line-height: 1.4;
    display: flex;
    align-items: flex-start;
    gap: 0.4rem;
    margin-bottom: 0.25rem;
    white-space: pre-wrap;
  }

  .note-session {
    font-size: 0.7rem;
    font-weight: 800;
    padding: 2px 4px;
    border-radius: 3px;
    color: white;
    min-width: 1.2rem;
    text-align: center;
    flex-shrink: 0;
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

  /* Trade Mini styles */
  .trades-stack {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .trade-meta {
    display: flex;
    gap: 0.5rem;
  }

  .session-tag {
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    background: #e2e8f0;
    color: #475569;
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
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
  }

  .side-tag.long {
    background: #fee2e2;
    color: #991b1b;
  }
  .side-tag.short {
    background: #dcfce7;
    color: #166534;
  }

  .symbol-inline-tag {
    font-size: 0.75rem;
    font-weight: 800;
    color: #1e293b;
    padding: 2px 6px;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 4px;
  }

  .session-tag.none {
    background: #f1f5f9;
    color: #94a3b8;
    font-style: italic;
  }

  .ticket-tag {
    font-size: 0.75rem;
    color: #94a3b8;
    font-family: monospace;
    align-self: center;
  }

  .strategy-tag {
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
  }

  .strategy-tag.expert {
    background: #059669;
    color: white;
    border: none;
  }

  .strategy-tag.elite {
    background: #1e3a8a;
    color: white;
    border: none;
  }

  .strategy-tag.legend {
    background: #78350f;
    color: white;
    border: none;
  }

  /* 交易時間分組樣式 */
  .trade-time-group.is-multi {
    padding: 1.25rem;
    background: rgba(244, 114, 182, 0.03); /* 極淡粉紅背景 */
    border-radius: 16px;
    border: 1px dashed rgba(244, 114, 182, 0.3); /* 粉紅虛線邊框 */
    position: relative;
    overflow: hidden;
    margin-bottom: 0.5rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .trade-time-group.is-multi:hover {
    background: rgba(244, 114, 182, 0.06);
    border-color: rgba(244, 114, 182, 0.5);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(244, 114, 182, 0.1);
  }

  /* 組合單樣式 */
  .group-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid rgba(244, 114, 182, 0.1);
  }

  .group-meta {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .group-meta::-webkit-scrollbar {
    display: none;
  }

  .multi-indicator {
    background: #f472b6;
    color: white;
    font-size: 0.75rem;
    font-weight: 800;
    padding: 2px 8px;
    border-radius: 4px;
    box-shadow: 0 2px 4px rgba(244, 114, 182, 0.3);
    flex-shrink: 0;
  }

  .group-pnl {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .group-pnl .pnl-tag {
    font-size: 1.1rem;
    padding: 6px 12px;
  }

  .group-partial-closes {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    background: white;
    padding: 0.75rem;
    border-radius: 10px;
    border: 1px solid rgba(244, 114, 182, 0.1);
  }

  .partial-close-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    font-size: 0.8rem;
    color: #64748b;
    padding: 6px 0;
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .partial-close-row::-webkit-scrollbar {
    display: none;
  }

  .partial-close-row:not(:last-child) {
    border-bottom: 1px dashed #f1f5f9;
  }

  .partial-pnl {
    font-weight: 700;
    margin-left: 0.5rem;
    white-space: nowrap;
  }

  .partial-ticket {
    font-size: 0.7rem;
    color: #cbd5e1;
    font-family: monospace;
  }

  .partial-pnl.profit {
    color: #3b82f6;
  }
  .partial-pnl.loss {
    color: #ef4444;
  }

  .partial-ticket {
    font-family: monospace;
    font-size: 0.75rem;
    color: #94a3b8;
    text-align: right;
  }

  /* 側邊粉紅條（仿照使用者附圖） */
  .trade-time-group.is-multi::before {
    content: '';
    position: absolute;
    left: 4px;
    top: 15%;
    bottom: 45%; /* 只佔上半部，感覺較輕快 */
    width: 3px;
    background: #f472b6;
    border-radius: 2px;
    opacity: 0.8;
  }

  .trade-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .pnl-tag {
    font-weight: 800;
    font-size: 0.95rem;
  }

  .pnl-tag.profit {
    color: #3b82f6;
  }
  .pnl-tag.loss {
    color: #ef4444;
  }
  .mini-step.short {
    color: #ef4444;
  }
  .mini-step.na {
    color: #94a3b8;
  }
  .step-arrow {
    color: #cbd5e1;
    font-weight: bold;
    font-size: 0.8rem;
  }

  .trade-details {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-top: 0.75rem;
    padding-top: 0.5rem;
    border-top: 1px dashed #f1f5f9;
    gap: 0.75rem 1rem;
    flex-wrap: wrap;
  }

  .detail-row {
    flex: 1;
    min-width: 0;
    font-size: 0.8rem;
    color: #64748b;
    display: flex;
    gap: 0.5rem 1.5rem;
    flex-wrap: wrap;
  }

  .info-group {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    white-space: nowrap;
  }

  .info-group .label {
    color: #94a3b8;
    font-size: 0.75rem;
  }

  .info-group .arrow {
    color: #cbd5e1;
    margin: 0 0.2rem;
  }

  .bullet {
    color: #6366f1;
  }

  .rr.profit {
    color: #f59e0b;
  }
  .rr.loss {
    color: #ef4444;
  }

  .duration-text {
    color: #6366f1;
    font-weight: 600;
  }

  .trade-time {
    font-size: 0.75rem;
    color: #94a3b8;
    white-space: nowrap;
    text-align: right;
    flex-shrink: 0;
  }

  .mini-gallery {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .mini-img {
    width: 50px;
    height: 40px;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
  }

  .mini-img img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  /* Empty dash styles */
  .empty-placeholder {
    height: 100px;
    border: 2px dashed #e2e8f0;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s;
  }

  .empty-placeholder:hover {
    border-color: #6366f1;
    color: #6366f1;
    background: #f5f3ff;
  }

  .plus-icon {
    font-size: 1.25rem;
  }

  .loading-state,
  .empty-state {
    text-align: center;
    padding: 5rem;
    color: #64748b;
  }

  .empty-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .loader {
    width: 40px;
    height: 40px;
    border: 4px solid #f3f3f3;
    border-top: 4px solid #6366f1;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 1rem;
  }

  @keyframes spin {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }

  /* Modal */
  .modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.9);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .modal-content {
    position: relative;
    max-width: 90%;
    max-height: 90%;
  }

  .timeline-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
  }

  /* Hero Section Styles */
  .home-hero {
    margin-bottom: 3rem;
    background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
    padding: 3rem;
    border-radius: 24px;
    border: 1px solid #e2e8f0;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05);
    position: relative;
    overflow: hidden;
  }

  .home-hero::before {
    content: '';
    position: absolute;
    top: -50%;
    right: -10%;
    width: 400px;
    height: 400px;
    background: radial-gradient(circle, rgba(99, 102, 241, 0.05) 0%, transparent 70%);
    pointer-events: none;
  }

  .hero-title {
    margin-bottom: 2rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .greeting {
    font-size: 1.1rem;
    color: #64748b;
    font-weight: 500;
  }

  .top-actions-bar {
    display: flex;
    justify-content: flex-end;
    margin-top: -0.5rem;
    margin-bottom: 0.5rem;
    padding: 0 1rem;
  }

  .quick-btns {
    display: flex;
    gap: 0.75rem;
  }

  .small-action-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    border-radius: 10px;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid #e2e8f0;
    background: white;
    color: #1e293b;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .small-action-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  }

  .small-action-btn.sync {
    background: rgba(99, 102, 241, 0.1);
    color: #818cf8;
  }

  .small-action-btn.sync:hover:not(:disabled) {
    background: rgba(99, 102, 241, 0.2);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
  }

  .small-action-btn.syncing .btn-icon {
    display: inline-block;
    animation: rotate 2s linear infinite;
  }

  @keyframes rotate {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .small-action-btn.plan {
    background: rgba(99, 102, 241, 0.05);
    border-color: rgba(99, 102, 241, 0.2);
    color: #4f46e5;
  }

  .small-action-btn.plan:hover {
    background: rgba(99, 102, 241, 0.1);
    border-color: #6366f1;
  }

  .small-action-btn.trade {
    background: rgba(16, 185, 129, 0.05);
    border-color: rgba(16, 185, 129, 0.2);
    color: #059669;
  }

  .small-action-btn.trade:hover {
    background: rgba(16, 185, 129, 0.1);
    border-color: #10b981;
  }

  .btn-icon {
    font-size: 1.1rem;
  }

  .modal-content img {
    max-width: 100%;
    max-height: 90vh;
    border-radius: 8px;
  }

  .modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 3rem;
    cursor: pointer;
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
  }
  .trade-item-card {
    position: relative;
    overflow: hidden;
  }

  .trade-item-card.tag-green,
  .trade-time-group.is-multi.tag-green {
    border-left: 5px solid #22c55e;
  }
  .trade-item-card.tag-yellow,
  .trade-time-group.is-multi.tag-yellow {
    border-left: 5px solid #eab308;
  }
  .trade-item-card.tag-red,
  .trade-time-group.is-multi.tag-red {
    border-left: 5px solid #ef4444;
  }

  .color-tags {
    display: flex;
    gap: 0.3rem;
    margin-right: 0.75rem;
  }

  .color-btn {
    width: 1rem;
    height: 1rem;
    border-radius: 50%;
    border: 1px solid #ddd;
    cursor: pointer;
    transition:
      transform 0.1s,
      border-color 0.1s;
    padding: 0;
  }
  .color-btn:hover {
    transform: scale(1.1);
  }
  .color-btn.active {
    border: 2px solid #333;
    transform: scale(1.1);
  }
  .color-btn.green {
    background-color: #22c55e;
  }
  .color-btn.yellow {
    background-color: #eab308;
  }
  .color-btn.red {
    background-color: #ef4444;
  }
</style>
