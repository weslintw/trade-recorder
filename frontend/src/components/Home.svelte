<script>
  import { onMount, onDestroy } from 'svelte';
  import { navigate, Link } from 'svelte-routing';
  import { tradesAPI, dailyPlansAPI, imagesAPI, sharesAPI, accountsAPI } from '../lib/api';
  import { selectedSymbol, selectedAccountId, accounts } from '../lib/stores';
  import { MARKET_SESSIONS, SYMBOLS, TIMEFRAMES } from '../lib/constants';
  import { determineMarketSession, getStrategyLabel, parseJSONSafe } from '../lib/utils';
  import AccountModal from './AccountModal.svelte';
  import Sparkline from './Sparkline.svelte';
  import BatchShareModal from './BatchShareModal.svelte';
  import SyncOptionsModal from './SyncOptionsModal.svelte';
  import PlanSummaryTable from './PlanSummaryTable.svelte';

  let groupedData = [];
  let loading = true;
  let todayString = new Date().toLocaleDateString('en-CA'); // 使用 YYYY-MM-DD 格式的本地日期
  let selectedImage = null;
  let isSyncing = false;
  let showAccountModal = false;
  let showBatchShareModal = false;
  let pollingTimeout;
  let currentPollingInterval = 60000; // 預設閒置輪詢改為 60 秒 (僅作為 WebSocket 的備援)
  let ws;

  // 批次分享相關狀態
  let selectionMode = false;
  let selectedTrades = new Set();
  let selectedPlans = new Set();
  let isSharing = false;
  let generatedShareToken = '';
  let showSyncOptionsModal = false;
  let activeFilterType = 'all'; // 'all', 'expert', 'elite', 'legend'
  let activeSubFilter = null;

  const EXPERT_SIGNALS = [
    '向下蘇美',
    '起漲靠山',
    '雙柱',
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
  ];

  const subFilters = {
    expert: EXPERT_SIGNALS.map(s => ({ value: s, label: s })),
    elite: ELITE_PATTERNS.map(p => ({ value: p, label: p })),
    legend: LEGEND_CHECKLIST.map(l => ({ value: l.id, label: l.label })),
  };

  const colorTagMeanings = {
    green: '有照標準進單',
    yellow: '有討論空間',
    red: '衝動，沒有照標準',
  };

  // Date Filter State
  // Date Filter State
  let activeDateRange = 'all'; // Default to 'all' (no filter)
  let customStartDate = '';
  let customEndDate = '';

  function setDateRange(range) {
    if (activeDateRange === range) {
      // Toggle off if clicking the same range again
      activeDateRange = 'all';
      customStartDate = '';
      customEndDate = '';
    } else {
      activeDateRange = range;
      const now = new Date();
      const todayStr = now.toISOString().split('T')[0];
      
      if (range === '1D') {
        customStartDate = todayStr;
        customEndDate = todayStr;
      } else if (range === '1W') {
        const d = new Date();
        d.setDate(d.getDate() - 7);
        customStartDate = d.toISOString().split('T')[0];
        customEndDate = todayStr;
      } else if (range === '1M') {
        const d = new Date();
        d.setMonth(d.getMonth() - 1);
        customStartDate = d.toISOString().split('T')[0];
        customEndDate = todayStr;
      }
      // For 'custom', we check in the button click handler, usually sets activeDateRange directly
    }
    loadData();
  }

  // Initial call to set defaults correctly if needed, though loadData handles it.
  // We prefer to update the date variables when the range button is clicked.

  function selectFilterType(type) {
    if (activeFilterType === type) {
      activeFilterType = 'all';
      activeSubFilter = null;
    } else {
      activeFilterType = type;
      activeSubFilter = null;
    }
  }

  function toggleSubFilter(value) {
    if (activeSubFilter === value) {
      activeSubFilter = null;
    } else {
      activeSubFilter = value;
    }
  }

  $: filteredGroupedData = applyFilters(groupedData, activeFilterType, activeSubFilter);

  function formatSignalsSummary(trend) {
    if (!trend) return '';
    let signals = [];
    // 檢查是否有方向性的訊號 (優先看是否有勾選 has_signals/has_expected_signals)
    if (trend.long) {
      if (trend.long.has_signals && trend.long.signals?.length > 0) {
        signals.push(...trend.long.signals);
      }
      if (trend.long.has_expected_signals && trend.long.expected_signals?.length > 0) {
        signals.push(...trend.long.expected_signals.map(s => s.name));
      }
    }
    if (trend.short) {
      if (trend.short.has_signals && trend.short.signals?.length > 0) {
        signals.push(...trend.short.signals);
      }
      if (trend.short.has_expected_signals && trend.short.expected_signals?.length > 0) {
        signals.push(...trend.short.expected_signals.map(s => s.name));
      }
    }
    
    // 如果是舊格式，訊號可能在頂層
    if (trend.signals?.length > 0) signals.push(...trend.signals);

    if (signals.length === 0) return '';
    return [...new Set(signals)].join(',');
  }

  function formatWaveSummary(trend) {
    if (!trend) return '';
    
    const formatDir = (dirData) => {
      if (!dirData?.wave_numbers?.length) return '';
      const nums = dirData.wave_numbers;
      const highlight = dirData.wave_highlight;
      return nums.map((n, i) => {
        const isHighlight = n.toString() === highlight?.toString();
        const val = isHighlight ? `[${n}]` : n;
        return (i > 0 ? ' => ' : '') + val;
      }).join('');
    };

    let waves = [];
    if (trend.long && trend.long.has_wave) {
      const w = formatDir(trend.long);
      if (w) waves.push(trend.short?.has_wave ? `多:${w}` : w);
    }
    if (trend.short && trend.short.has_wave) {
      const w = formatDir(trend.short);
      if (w) waves.push(trend.long?.has_wave ? `空:${w}` : w);
    }
    
    // 舊格式
    if (waves.length === 0 && trend.wave_numbers?.length > 0) {
      const w = trend.wave_numbers.map((n, i) => {
        const isHighlight = n.toString() === trend.wave_highlight?.toString();
        const val = isHighlight ? `[${n}]` : n;
        return (i > 0 ? ' => ' : '') + val;
      }).join('');
      waves.push(w);
    }

    if (waves.length === 0) return '';
    return [...new Set(waves)].join(' | ');
  }

  function applyFilters(data, type, sub) {
    if (!data) return [];
    if (type === 'all' && !sub) return data;

    return data
      .map(day => {
        const filteredPlans = day.plans.filter(plan => {
          // 規劃目前僅支持達人訊號過濾
          if (type !== 'expert' && type !== 'all') return false;

          const trendData = parseJSONSafe(plan.trend_analysis, {});
          // 檢查所有時段與時區
          for (const sessionKey of ['asian', 'european', 'us']) {
            const sessData = trendData[sessionKey];
            if (!sessData || !sessData.trends) continue;

            for (const tf of TIMEFRAMES) {
              const trend = sessData.trends[tf];
              if (!trend) continue;

              // 檢查 long 和 short 方向
              for (const dir of ['long', 'short']) {
                const dData = trend[dir];
                if (!dData) continue;

                if (type === 'expert' || type === 'all') {
                  const signals = dData.has_signals ? (dData.signals || []) : [];
                  const expected = dData.has_expected_signals ? (dData.expected_signals || []) : [];

                  if (!sub) {
                    // 如果沒選子項目，只要是達人有訊號就顯示
                    if (signals.length > 0 || expected.length > 0) return true;
                  } else {
                    if (dData.has_signals && signals.includes(sub)) return true;
                    if (dData.has_expected_signals && expected.some(s => s.name === sub)) return true;
                  }
                }
              }
            }
          }
          return false;
        });

        const filteredGroupedTrades = day.groupedTrades
          .map(group => {
            const filteredTrades = group.trades.filter(trade => {
              if (type !== 'all' && trade.entry_strategy !== type) return false;

              if (type === 'expert') {
                const signals = parseJSONSafe(trade.entry_signals, []);
                if (!sub) return true;
                return signals.some(s => (typeof s === 'object' ? s.name === sub : s === sub));
              }

              if (type === 'elite') {
                const patterns = parseJSONSafe(trade.entry_pattern, []);
                if (!sub) return true;
                return patterns.some(p => (typeof p === 'object' ? p.name === sub : p === sub));
              }

              if (type === 'legend') {
                const checklist = parseJSONSafe(trade.entry_checklist, {});
                if (!sub) return true;
                return checklist[sub] === true;
              }

              // 如果 type 為 all 且有 sub，這情況較少見，目前暫不特別處理
              return type === 'all';
            });

            if (filteredTrades.length === 0) return null;

            return {
              ...group,
              trades: filteredTrades,
              summary: {
                ...group.summary,
                totalPnl: filteredTrades.reduce((sum, t) => sum + (t.pnl || 0), 0),
                totalLot: filteredTrades.reduce((sum, t) => sum + (t.lot_size || 0), 0),
              },
            };
          })
          .filter(Boolean);

        return {
          ...day,
          plans: filteredPlans,
          groupedTrades: filteredGroupedTrades,
        };
      })
      .filter(day => day.plans.length > 0 || day.groupedTrades.length > 0);
  }

  // 追蹤當前選取的帳號詳情
  $: currentAccount = $accounts.find(a => a.id === $selectedAccountId);

  let loadController; // To abort in-flight requests

  // 響應式：當帳號或品種改變時，自動重新載入資料 (加上 Debounce 防止連點)
  let debounceTimer;
  $: if ($selectedAccountId && $selectedSymbol) {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      console.log(
        `🏠 [Reactive] Account/Symbol changed: acc=${$selectedAccountId}, sym=${$selectedSymbol}`
      );
      // Initialize date range based on default or stored logic? 
      // For now, we just reload, using current activeDateRange/custom dates.
      // If we want to reset date on symbol change, do it here.
      // setDateRange('1W'); // Optional: reset to 1W on symbol change
      loadData();
    }, 300); // 300ms 防抖
  }

  // 響應式派生交易清單 (供 polling 檢查有無未平倉)
  $: timeGroupedTrades = (groupedData || []).flatMap(day => day.groupedTrades || []);

  function navigateWithScroll(path) {
    sessionStorage.setItem('home_scroll_pos', window.scrollY);
    navigate(path);
  }

  async function handleSync(options = {}) {
    if (!$selectedAccountId || isSyncing) return;

    // If it's cTrader and no options provided, show modal
    if (
      currentAccount?.type === 'ctrader' &&
      Object.keys(options).length === 0 &&
      !showSyncOptionsModal
    ) {
      showSyncOptionsModal = true;
      return;
    }

    showSyncOptionsModal = false;
    isSyncing = true;
    try {
      await accountsAPI.sync($selectedAccountId, options);
      // Refresh both account info (for storage usage) and data
      await refreshAccounts();
      await loadData(true);
    } catch (error) {
      console.error('Sync failed:', error);
      alert('同步失敗: ' + (error.response?.data?.error || error.message));
    } finally {
      isSyncing = false;
    }
  }

  // Unique instance ID to track multiple instances
  const INSTANCE_ID = `Home-${Math.random().toString(36).substr(2, 9)}`;
  console.log(`🏠 Home component created: ${INSTANCE_ID}`);

  let loadDataCallCount = 0;
  let refreshAccountsCallCount = 0;
  let activeLoadCallId = 0; // 新增：追蹤當前最新的加載 ID，防止舊請求覆蓋新請求

  async function loadData(silent = false) {
    const callId = ++loadDataCallCount; // 每次調用產生唯一 ID
    activeLoadCallId = callId;

    // Abort previous request before starting new one
    if (loadController) {
      console.log(`[${INSTANCE_ID}] Aborting previous in-flight requests...`);
      loadController.abort();
    }
    loadController = new AbortController();
    const { signal } = loadController;

    console.log(`🔵 [${INSTANCE_ID}] loadData #${callId} called, silent:`, silent);

    try {
      if (!silent) {
        loading = true;
        groupedData = [];
        // 保險機制：如果 8 秒後還在轉圈圈且資料沒回來，強制關閉轉圈 (可能是網路連線排隊或 ECONNABORTED)
        setTimeout(() => {
          if (loading && activeLoadCallId === callId) {
            console.warn(
              `[${INSTANCE_ID}] Loading state safety timeout triggered (8s). Forcing spinner OFF.`
            );
            loading = false;
          }
        }, 8000);
      }
      const symbol = $selectedSymbol;

      // 更新今天日期文字
      todayString = new Date().toISOString().slice(0, 10);

      console.log(
        `🔵 [${INSTANCE_ID}] Fetching data for account=${$selectedAccountId}, symbol=${symbol}`
      );

      // 分別獲取，避免一個失敗全部失敗
      let plans = [];
      let trades = [];

      console.time(`🔵 [${INSTANCE_ID}] loadData #${callId} API Calls`);
      try {
        console.log(`⏱️ [#${callId}] Parallel Fetching Plans & Trades...`);
        
        const [plansRes, tradesRes] = await Promise.all([
          dailyPlansAPI.getAll(
            { 
              account_id: $selectedAccountId, 
              symbol, 
              page_size: activeDateRange === 'all' ? 20 : 1000, 
              start_date: activeDateRange === 'all' ? undefined : customStartDate, 
              end_date: activeDateRange === 'all' ? undefined : (customEndDate ? customEndDate + ' 23:59:59' : undefined)
            },
            signal
          ).catch(e => {
            if (e.name === 'CanceledError' || e.name === 'AbortError') return { data: [] };
            console.error('Plans fetch error:', e);
            return { data: [] };
          }),
          tradesAPI.getAll(
            { 
              account_id: $selectedAccountId, 
              symbol, 
              page_size: activeDateRange === 'all' ? 50 : 1000,
              start_date: activeDateRange === 'all' ? undefined : customStartDate, 
              end_date: activeDateRange === 'all' ? undefined : (customEndDate ? customEndDate + ' 23:59:59' : undefined)
            },
            signal
          ).catch(e => {
            if (e.name === 'CanceledError' || e.name === 'AbortError') return { data: [] };
            console.error('Trades fetch error:', e);
            return { data: [] };
          })
        ]);

        plans = (Array.isArray(plansRes.data) ? plansRes.data : plansRes.data?.data) || [];
        trades = (Array.isArray(tradesRes.data) ? tradesRes.data : tradesRes.data?.data) || [];

        console.log(
          `⏱️ [#${callId}] Sequence finished: ${plans.length} plans, ${trades.length} trades.`
        );
      } catch (err) {
        if (err.name === 'CanceledError' || err.name === 'AbortError') {
          console.log('Request was aborted intentionally.');
        } else {
          console.error('Critical failure in sequence fetch:', err);
        }
      }
      console.timeEnd(`🔵 [${INSTANCE_ID}] loadData #${callId} API Calls`);
      console.log(`🔵 [${INSTANCE_ID}] API Calls finished. Starting data processing...`);

      console.time(`🔵 [${INSTANCE_ID}] loadData #${callId} Data Processing`);

      // De-duplicate trades on client side as a safety measure
      // Priority: Keep the one with ID if others don't have it, or keeps latest
      const seenTickets = new Set();
      const uniqueTrades = [];
      trades.forEach(trade => {
         // If trade has ticket, use it for checking uniqueness
         if (trade.ticket && trade.ticket.startsWith('ctrader-')) {
            if (seenTickets.has(trade.ticket)) {
                console.warn(`[Client Dedup] Duplicate ticket found and skipped: ${trade.ticket} (ID: ${trade.id})`);
                return;
            }
            seenTickets.add(trade.ticket);
         }
         // Also check by ID if possible (though API returns unique IDs usually, but just in case of merge)
         uniqueTrades.push(trade);
      });
      trades = uniqueTrades;

      console.log(
        `🔵 loadData #${loadDataCallCount}: Loaded ${plans.length} plans, ${trades.length} trades`
      );

      // 按日期分組 (YYYY-MM-DD)
      const dateMap = {};

      // 強制推入今天的日期，確保最上面有東西
      dateMap[todayString] = { date: todayString, plans: [], groupedTrades: [] };

      plans.forEach(plan => {
        try {
          // Pre-parse trend analysis to avoid template errors & const limitations
          plan.trendData = parseJSONSafe(plan.trend_analysis, {});

          if (!plan.plan_date) return;
          const planDateObj = new Date(plan.plan_date);
          if (isNaN(planDateObj.getTime())) {
            console.warn('Invalid plan date:', plan.plan_date);
            return;
          }
          const date = planDateObj.toISOString().slice(0, 10);
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
      const newGroupedData = Object.values(dateMap).sort((a, b) => b.date.localeCompare(a.date));

      // 針對組合單內的成員排序 (先平倉的在上面)
      newGroupedData.forEach(day => {
        // 先對卡片進行排序：最新平倉的在最上面 (未平倉視為最新)
        day.groupedTrades.sort((a, b) => {
          const getTime = g => {
            if (g.trades.some(t => !t.exit_time)) return Infinity; // 未平倉置頂
            return Math.max(...g.trades.map(t => new Date(t.exit_time || 0).getTime()));
          };
          return getTime(b) - getTime(a);
        });

        // 再對群組內部的交易排序
        day.groupedTrades.forEach(group => {
          if (group.trades.length > 1) {
            group.trades.sort((a, b) => new Date(a.exit_time || 0) - new Date(b.exit_time || 0));
          }
        });
      });

      if (activeLoadCallId !== callId) {
        console.log(
          `[${INSTANCE_ID}] loadData #${callId} superseded by #${activeLoadCallId}, skipping UI update.`
        );
        return;
      }

      groupedData = newGroupedData;
      console.timeEnd(`🔵 [${INSTANCE_ID}] loadData #${callId} Data Processing`);
      console.log(`Final groupedData (Call #${callId}):`, groupedData);
    } catch (error) {
      if (error.name === 'CanceledError' || error.name === 'AbortError') {
        console.log(`[${INSTANCE_ID}] loadData #${callId} caught abort.`);
      } else {
        console.error(`載入首頁資料失敗 (Call #${callId}):`, error);
      }
    } finally {
      // 只有當前這個 call 是最後一個發出的，才解除 Loading 狀態
      if (activeLoadCallId === callId) {
        loading = false;
      }
    }
  }

  // 獨立更新帳號狀態 (不觸發 loading UI)
  let lastAccountsData = null;
  let isRefreshingAccounts = false;
  let refreshAccountsController = null;
  async function refreshAccounts() {
    if (isRefreshingAccounts) {
      console.log('🟢 [refreshAccounts] Already refreshing, skipping.');
      return;
    }
    
    // Abort previous refresh if any
    if (refreshAccountsController) refreshAccountsController.abort();
    refreshAccountsController = new AbortController();
    const { signal } = refreshAccountsController;

    isRefreshingAccounts = true;

    refreshAccountsCallCount++;
    console.log(`🟢 refreshAccounts #${refreshAccountsCallCount} called`);

    try {
      const startTime = performance.now();
      const res = await accountsAPI.getAll(signal);
      const duration = (performance.now() - startTime).toFixed(2);

      if (res && res.data) {
        console.log(
          `🟢 refreshAccounts #${refreshAccountsCallCount}: API took ${duration}ms, returned ${res.data.length} accounts`
        );
        // 只有當資料真正變化時才更新 store
        const newData = JSON.stringify(res.data);
        if (newData !== lastAccountsData) {
          console.log(
            `🟢 refreshAccounts #${refreshAccountsCallCount}: Data changed, updating store`
          );
          lastAccountsData = newData;
          accounts.set(res.data);
        } else {
          // console.log(`🟢 refreshAccounts #${refreshAccountsCallCount}: Data unchanged, skipping update`);
        }
      }
    } catch (e) {
      if (e.name === 'CanceledError' || e.name === 'AbortError') {
        // console.log('🟢 [refreshAccounts] Request was aborted.');
      } else {
        console.error('Failed to refresh accounts:', e);
      }
    } finally {
      isRefreshingAccounts = false;
    }
  }
  function initRealtimeNotifications() {
    if (ws) {
      ws.onclose = null;
      ws.close();
    }

    // 取得當前 API 的 Base URL 並轉換為 ws/wss
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // 假設後端與前端同 Host，或是從 API 配置中取得
    const host = window.location.host.includes('localhost')
      ? 'localhost:8080'
      : window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/ws`;

    console.log(`[Realtime] Connecting to ${wsUrl}...`);
    ws = new WebSocket(wsUrl);

    ws.onmessage = event => {
      try {
        const msg = JSON.parse(event.data);
        // console.log('[Realtime] Message received:', msg);

        if (msg.type === 'TRADE_UPDATE') {
          if (!msg.account_id || msg.account_id === $selectedAccountId) {
            console.log('🚀 [Realtime] Trade update detected, reloading...');
            loadData(true);
            refreshAccounts();
          }
        } else if (msg.type === 'PRICE_UPDATE') {
          if (!msg.account_id || msg.account_id === $selectedAccountId) {
            throttledLoadData();
          }
        }
      } catch (e) {
        console.error('[Realtime] Parse error:', e);
      }
    };

    ws.onclose = () => {
      console.log('[Realtime] Disconnected, retrying in 5s...');
      setTimeout(initRealtimeNotifications, 5000);
    };

    ws.onerror = err => {
      console.error('[Realtime] WS Error:', err);
    };
  }

  // 節流處理價格更新，避免過度頻繁的 API 請求
  let lastRealtimeLoadTime = 0;
  function throttledLoadData() {
    const now = Date.now();
    if (now - lastRealtimeLoadTime > 5000) {
      lastRealtimeLoadTime = now;
      console.log('📈 [Realtime] Price update detected, refreshing...');
      loadData(true);
      refreshAccounts();
    }
  }

  async function poll() {
    if (pollingTimeout) clearTimeout(pollingTimeout);

    // 如果 WebSocket 斷線，則維持基本的輪詢
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      if ($selectedAccountId) {
        console.log(`[Fallback Polling] Triggering refresh...`);
        await Promise.all([loadData(true), refreshAccounts()]);
      }
    } else {
      // 如果 WebSocket 正常，則每分鐘執行一次「大檢查」即可
      if ($selectedAccountId) {
        // console.log(`[Health Check] Routine refresh...`);
        await refreshAccounts(); // 僅刷新帳號狀態
      }
    }

    pollingTimeout = setTimeout(poll, currentPollingInterval);
  }

  onMount(async () => {
    console.log('=== 🏠 Home onMount START ===');

    // Step 1: 預加載帳號列表
    await refreshAccounts();

    // Step 2: 檢查當前選取的帳號是否有效
    const accountExists = $accounts.some(a => a.id === $selectedAccountId);
    if ($accounts.length > 0) {
      if (!$selectedAccountId || !accountExists) {
        console.log(`[onMount] Auto-selecting first account: ${$accounts[0].id}`);
        selectedAccountId.set($accounts[0].id);
      }
    } else {
      loading = false; // No accounts, stop loading spinner
    }

    // 初始化日期範圍 (Default to all)
    // setDateRange('all'); // Already default
    // We already call loadData below implicitly or explicitly?
    // Actually the symbol watcher debouncer will call loadData.
    // But we also want to ensure we don't double load.
    // The previous code had setDateRange('1W') here. Remove it.

    // Step 3: 延後啟動即時通知與備援輪詢，避免跟 Initial Data 搶瀏覽器併發連線
    setTimeout(() => {
      initRealtimeNotifications();
      poll();
    }, 5000);

    // Restore scroll position
    const savedScrollPos = sessionStorage.getItem('home_scroll_pos');
    if (savedScrollPos) {
      setTimeout(() => {
        window.scrollTo(0, parseInt(savedScrollPos));
        sessionStorage.removeItem('home_scroll_pos');
      }, 300);
    }

    console.log('=== 🏠 Home onMount END ===');
  });

  onDestroy(() => {
    console.log('=== onDestroy: Cleaning up ===');
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }
    if (pollingTimeout) {
      clearTimeout(pollingTimeout);
    }
    if (loadController) {
      loadController.abort();
    }
    if (ws) {
      ws.onclose = null; // 阻止自動重連
      ws.close();
    }
  });

  // Manual reload function for when user changes selection from UI
  export function reloadData() {
    console.log('🔄 Manual reload triggered');
    loadData();
  }

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

  function calculateDailyStats(dayGroup) {
    if (!dayGroup || !dayGroup.groupedTrades) {
      return { winRate: 0, totalPnl: 0, floatingPnl: 0, wins: 0, total: 0, hasFloating: false };
    }

    const allTrades = dayGroup.groupedTrades.flatMap(group => group.trades);
    
    // 只計算已平倉的交易用於勝率和已實現盈虧
    const closedTrades = allTrades.filter(trade => trade.exit_time && trade.pnl !== null && trade.pnl !== undefined);
    
    // 計算未平倉的交易（浮動盈虧）
    const openTrades = allTrades.filter(trade => !trade.exit_time && trade.pnl !== null && trade.pnl !== undefined);

    const total = closedTrades.length;
    const wins = closedTrades.filter(trade => trade.pnl > 0).length;
    const totalPnl = closedTrades.reduce((sum, trade) => sum + (trade.pnl || 0), 0);
    const floatingPnl = openTrades.reduce((sum, trade) => sum + (trade.pnl || 0), 0);
    const winRate = total > 0 ? (wins / total) * 100 : 0;
    const hasFloating = openTrades.length > 0;

    return { winRate, totalPnl, floatingPnl, wins, total, hasFloating };
  }

  function getMarketSessionLabel(trade) {
    let session = trade.market_session;
    // 如果資料庫中沒有時段資料，根據時間即時計算
    if (!session && trade.entry_time) {
      session = determineMarketSession(trade.entry_time);
    }
    return MARKET_SESSIONS.find(s => s.value === session)?.label || session || '未設定';
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
      journal: fullTrade.journal || '',
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

  async function syncSingleTrade(id) {
    try {
      await tradesAPI.sync(id);
      alert('已送出手動同步請求，資料將在幾秒內更新');
      setTimeout(() => loadData(true), 3000);
    } catch (error) {
      console.error('Sync failed:', error);
      alert('同步失敗: ' + (error.response?.data?.error || error.message));
    }
  }

  // 批次分享邏輯
  function startSelection() {
    selectionMode = true;
    selectedTrades = new Set();
    selectedPlans = new Set();
  }

  function cancelSelection() {
    selectionMode = false;
    selectedTrades = new Set();
    selectedPlans = new Set();
  }

  function toggleTradeSelection(id) {
    if (selectedTrades.has(id)) {
      selectedTrades.delete(id);
    } else {
      selectedTrades.add(id);
    }
    selectedTrades = selectedTrades;
  }

  function togglePlanSelection(id) {
    if (selectedPlans.has(id)) {
      selectedPlans.delete(id);
    } else {
      selectedPlans.add(id);
    }
    selectedPlans = selectedPlans;
  }

  function toggleDaySelection(group) {
    const tradeIds = group.groupedTrades.flatMap(gt => gt.trades.map(t => t.id));
    const planIds = group.plans.map(p => p.id);

    const allSelected =
      tradeIds.every(id => selectedTrades.has(id)) && planIds.every(id => selectedPlans.has(id));

    if (allSelected) {
      tradeIds.forEach(id => selectedTrades.delete(id));
      planIds.forEach(id => selectedPlans.delete(id));
    } else {
      tradeIds.forEach(id => selectedTrades.add(id));
      planIds.forEach(id => selectedPlans.add(id));
    }

    selectedTrades = selectedTrades;
    selectedPlans = selectedPlans;
  }

  async function submitBatchShare() {
    if (selectedTrades.size === 0 && selectedPlans.size === 0) {
      alert('請至少選擇一項內容進行分享');
      return;
    }

    isSharing = true;
    try {
      const res = await sharesAPI.create({
        resource_type: 'batch',
        resource_id: 0,
        resource_ids: {
          trades: Array.from(selectedTrades),
          plans: Array.from(selectedPlans),
        },
        share_type: 'public',
      });
      generatedShareToken = res.data.token;

      // 顯示成功訊息並關閉選取模式
      const shareUrl = `${window.location.origin}/shared/${generatedShareToken}?title=SharedSelection`;
      await navigator.clipboard.writeText(shareUrl);
      alert('批次分享連結已產生並複製到剪貼簿！');
      cancelSelection();
    } catch (e) {
      console.error(e);
      alert('批次分享失敗');
    } finally {
      isSharing = false;
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
            {currentAccount.type === 'local' ? '本地帳號' : 'cTrader'}
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
            <div class="storage-info-chip">
              <span class="chip-icon">📊</span>
              <span class="label">圖文佔用</span>
              <span class="value">{formatBytes(currentAccount.storage_usage)}</span>
            </div>
            {#if currentAccount.type === 'ctrader'}
              <div class="login-id-chip">
                <span class="chip-icon">🆔</span>
                <span class="label">Login ID</span>
                <span class="value">{currentAccount.ctrader_account_id}</span>
              </div>
            {/if}
          </div>
          {#if currentAccount.type !== 'local'}
            <div class="sync-status-info">
              <span
                class="sync-badge {currentAccount.sync_status} {currentAccount.sync_status
                  ?.toLowerCase()
                  .includes('syncing') ||
                currentAccount.sync_status?.toLowerCase().includes('fetching') ||
                currentAccount.sync_status?.toLowerCase().includes('scanning')
                  ? 'syncing'
                  : ''}">{currentAccount.sync_status}</span
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
              <button
                class="sync-icon-btn {isSyncing ? 'syncing' : ''}"
                on:click={handleSync}
                disabled={isSyncing}
                title={isSyncing ? '同步中...' : '手動同步'}
              >
                <span class="btn-icon">
                  {#if isSyncing}
                    ⏳
                  {:else}
                    <svg
                      width="20"
                      height="20"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2.5"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    >
                      <path d="M21 2v6h-6"></path>
                      <path d="M3 12a9 9 0 0 1 15-6.7L21 8"></path>
                      <path d="M3 22v-6h6"></path>
                      <path d="M21 12a9 9 0 0 1-15 6.7L3 16"></path>
                    </svg>
                  {/if}
                </span>
              </button>
            </div>
          {/if}
        </div>
      {/if}
    </div>
    <div class="quick-btns">
      <button
        class="small-action-btn share"
        on:click={() => (showBatchShareModal = true)}
        title="分享"
      >
        <span class="btn-icon">🔗</span> 分享
      </button>
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

  <!-- 現代感過濾器 -->
  <div class="filter-section">
    <div class="filter-glass-container">
      <!-- 日期篩選區 (新增) -->
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
                 activeDateRange = 'all'; // Toggle off
                 customStartDate = '';
                 customEndDate = '';
                 loadData();
              } else {
                 activeDateRange = 'custom';
                 // Set default custom range if empty
                 if (!customStartDate) {
                    const d = new Date();
                    d.setDate(d.getDate() - 7);
                    customStartDate = d.toISOString().split('T')[0];
                    customEndDate = new Date().toISOString().split('T')[0];
                 }
                 // Don't auto-load, let user pick dates
              }
            }}
          >
            <span class="btn-icon">📅</span> 自訂
          </button>
        </div>
        
        {#if activeDateRange === 'custom'}
          <div class="custom-date-inputs">
            <input 
              type="date" 
              class="date-input" 
              bind:value={customStartDate} 
              on:change={() => loadData()}
            />
            <span class="date-sep">~</span>
            <input 
              type="date" 
              class="date-input" 
              bind:value={customEndDate}
              on:change={() => loadData()}
            />
          </div>
        {/if}
      </div>

      <div class="divider-horizontal"></div>

      <div class="filter-main-types">
        <button
          class="filter-type-btn {activeFilterType === 'all' ? 'active' : ''}"
          on:click={() => selectFilterType('all')}
        >
          <span class="btn-text">全部</span>
        </button>
        <div class="divider"></div>
        <button
          class="filter-type-btn {activeFilterType === 'expert' ? 'active' : ''}"
          on:click={() => selectFilterType('expert')}
        >
          <span class="btn-icon">👨‍🏫</span>
          <span class="btn-text">達人</span>
        </button>
        <button
          class="filter-type-btn {activeFilterType === 'elite' ? 'active' : ''}"
          on:click={() => selectFilterType('elite')}
        >
          <span class="btn-icon">🛡️</span>
          <span class="btn-text">菁英</span>
        </button>
        <button
          class="filter-type-btn {activeFilterType === 'legend' ? 'active' : ''}"
          on:click={() => selectFilterType('legend')}
        >
          <span class="btn-icon">👑</span>
          <span class="btn-text">傳奇</span>
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

  {#if selectionMode}
    <div class="selection-bar">
      <div class="selection-info">
        已選擇 <strong>{selectedTrades.size}</strong> 筆交易與 <strong>{selectedPlans.size}</strong> 筆規劃
      </div>
      <div class="selection-actions">
        <button class="btn btn-secondary btn-sm" on:click={cancelSelection}>取消</button>
        <button class="btn btn-primary btn-sm" on:click={submitBatchShare} disabled={isSharing}>
          {isSharing ? '產生中...' : '確認分享'}
        </button>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="loading-overlay">
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
      {#each filteredGroupedData as group}
        {@const dailyStats = calculateDailyStats(group)}
        <div class="day-wrapper">
          <div class="day-marker">
            {#if selectionMode}
              <input
                type="checkbox"
                class="day-checkbox"
                checked={group.groupedTrades
                  .flatMap(gt => gt.trades.map(t => t.id))
                  .every(id => selectedTrades.has(id)) &&
                  group.plans.map(p => p.id).every(id => selectedPlans.has(id))}
                on:change={() => toggleDaySelection(group)}
              />
            {/if}
            <div class="date-tag">{formatDay(group.date)}</div>
            {#if dailyStats.total > 0 || dailyStats.hasFloating}
              <div class="daily-stats">
                {#if dailyStats.total > 0}
                  <span class="stat-item win-rate" class:high-win={dailyStats.winRate >= 60} class:low-win={dailyStats.winRate < 50}>
                    <span class="stat-label">勝率</span>
                    <span class="stat-value">{dailyStats.winRate.toFixed(1)}%</span>
                    <span class="stat-detail">({dailyStats.wins}/{dailyStats.total})</span>
                  </span>
                  <span class="stat-divider">|</span>
                {/if}
                <span class="stat-item pnl" class:profit={dailyStats.totalPnl > 0} class:loss={dailyStats.totalPnl < 0}>
                  <span class="stat-label">盈虧</span>
                  <span class="stat-value">{dailyStats.totalPnl >= 0 ? '+' : ''}{dailyStats.totalPnl.toFixed(2)}</span>
                </span>
                {#if dailyStats.hasFloating}
                  <span class="stat-divider">|</span>
                  <span class="stat-item pnl floating" class:profit={dailyStats.floatingPnl > 0} class:loss={dailyStats.floatingPnl < 0}>
                    <span class="stat-label">浮收</span>
                    <span class="stat-value">{dailyStats.floatingPnl >= 0 ? '+' : ''}{dailyStats.floatingPnl.toFixed(2)}</span>
                  </span>
                {/if}
              </div>
            {/if}
          </div>

          <div class="day-card-container">
            <!-- 左側規劃 -->
            <div class="plan-column">
              {#if group.plans.length > 0}
                {#each group.plans as plan}
                  {@const trendData = parseJSONSafe(plan.trend_analysis, {})}
                  {@const isUnified = plan.market_session === 'all'}
                  <div
                    class="plan-item-card {selectionMode && selectedPlans.has(plan.id)
                      ? 'selected'
                      : ''}"
                    on:click={() =>
                      selectionMode
                        ? togglePlanSelection(plan.id)
                        : navigate(`/plans/edit/${plan.id}`)}
                  >
                    {#if selectionMode}
                      <div class="card-selection-overlay">
                        <input
                          type="checkbox"
                          checked={selectedPlans.has(plan.id)}
                          on:click|stopPropagation
                        />
                      </div>
                    {/if}
                    <div class="item-header">
                      <span class="item-type">📌 盤面規劃</span>
                      <button
                        class="icon-btn delete"
                        on:click|stopPropagation={() => deletePlan(plan.id)}>🗑️</button
                      >
                    </div>

                    {#if isUnified}
                      <PlanSummaryTable {trendData} detailed={false} />
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
                          : ''} {selectionMode &&
                        timeGroup.trades.every(t => selectedTrades.has(t.id))
                          ? 'selected'
                          : ''} {timeGroup.trades.some(t => !t.exit_time) ? 'is-ongoing' : ''}"
                        on:click={() =>
                          selectionMode
                            ? timeGroup.trades.forEach(t => toggleTradeSelection(t.id))
                            : navigateWithScroll(`/edit/${timeGroup.trades[0].id}`)}
                      >
                        {#if selectionMode}
                          <div class="card-selection-overlay">
                            <input
                              type="checkbox"
                              checked={timeGroup.trades.every(t => selectedTrades.has(t.id))}
                              on:click|stopPropagation={() =>
                                timeGroup.trades.forEach(t => toggleTradeSelection(t.id))}
                            />
                          </div>
                        {/if}
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
                                  title={colorTagMeanings.green}
                                ></button>
                                <button
                                  class="color-btn yellow {timeGroup.trades[0].color_tag === 'yellow'
                                    ? 'active'
                                    : ''}"
                                  on:click={() => toggleColorTagForGroup(timeGroup, 'yellow')}
                                  title={colorTagMeanings.yellow}
                                ></button>
                                <button
                                  class="color-btn red {timeGroup.trades[0].color_tag === 'red'
                                    ? 'active'
                                    : ''}"
                                  on:click={() => toggleColorTagForGroup(timeGroup, 'red')}
                                  title={colorTagMeanings.red}
                                ></button>
                            </div>
                            {#if timeGroup.trades[0]?.pnl_series}
                              <div class="header-sparkline">
                                <Sparkline
                                  data={timeGroup.trades[0].pnl_series}
                                  width={80}
                                  height={28}
                                  isOpen={timeGroup.trades.some(t => !t.exit_time)}
                                />
                              </div>
                            {/if}
                            <span
                              class="pnl-tag {timeGroup.summary.totalPnl >= 0
                                ? timeGroup.summary.totalPnl === 0 &&
                                  !timeGroup.trades.some(t => t.pnl !== null)
                                  ? 'na'
                                  : 'profit'
                                : 'loss'}"
                            >
                              {timeGroup.summary.totalPnl === 0 &&
                              !timeGroup.trades.some(t => t.pnl !== null)
                                ? 'NA'
                                : (timeGroup.summary.totalPnl >= 0 ? '+' : '') +
                                  timeGroup.summary.totalPnl?.toFixed?.(2)}
                            </span>
                            <button
                              class="sync-btn-card"
                              on:click|stopPropagation={() =>
                                syncSingleTrade(timeGroup.trades[0].id)}
                              title="重新整理此交易資料"
                            >
                              <svg
                                width="12"
                                height="12"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2.5"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path
                                  d="M21 3v5h-5"
                                /></svg
                              >
                            </button>
                            {#if !timeGroup.trades[0]?.ticket?.startsWith('ctrader-')}
                              <button
                                class="icon-btn delete"
                                on:click|stopPropagation={() => deleteTradeGroup(timeGroup)}
                                >🗑️</button
                              >
                            {/if}
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
                                <span
                                  class="partial-pnl {trade.pnl >= 0
                                    ? trade.pnl === null
                                      ? ''
                                      : 'profit'
                                    : 'loss'}"
                                  >{trade.pnl === null || trade.pnl === undefined
                                    ? 'NA'
                                    : (trade.pnl >= 0 ? '+' : '') + trade.pnl?.toFixed(2)}</span
                                >
                                {#if trade.pnl_series}
                                  <div class="partial-sparkline">
                                    <Sparkline
                                      data={trade.pnl_series}
                                      width={60}
                                      height={20}
                                      isOpen={!trade.exit_time}
                                    />
                                  </div>
                                {/if}
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
                        class="trade-item-card {trade.color_tag
                          ? `tag-${trade.color_tag}`
                          : ''} {selectionMode && selectedTrades.has(trade.id) ? 'selected' : ''} {!trade.exit_time ? 'is-ongoing' : ''}"
                        on:click={() =>
                          selectionMode
                            ? toggleTradeSelection(trade.id)
                            : navigateWithScroll(`/edit/${trade.id}`)}
                      >
                        {#if selectionMode}
                          <input
                            type="checkbox"
                            class="selection-checkbox"
                            checked={selectedTrades.has(trade.id)}
                            on:click|stopPropagation={() => toggleTradeSelection(trade.id)}
                          />
                        {/if}
                        <div class="item-header">
                          <div class="trade-title-area">
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
                            </div>
                            {#if trade.ticket}<div class="ticket-tag">#{trade.ticket}</div>{/if}
                          </div>
                          <div class="trade-right">
                            <div class="color-tags" on:click|stopPropagation>
                                <button
                                  class="color-btn green {trade.color_tag === 'green'
                                    ? 'active'
                                    : ''}"
                                  on:click={() => toggleColorTag(trade, 'green')}
                                  title={colorTagMeanings.green}
                                ></button>
                                <button
                                  class="color-btn yellow {trade.color_tag === 'yellow'
                                    ? 'active'
                                    : ''}"
                                  on:click={() => toggleColorTag(trade, 'yellow')}
                                  title={colorTagMeanings.yellow}
                                ></button>
                                <button
                                  class="color-btn red {trade.color_tag === 'red' ? 'active' : ''}"
                                  on:click={() => toggleColorTag(trade, 'red')}
                                  title={colorTagMeanings.red}
                                ></button>
                            </div>
                            {#if trade.pnl_series}
                              <div class="header-sparkline">
                                <Sparkline
                                  data={trade.pnl_series}
                                  width={100}
                                  height={32}
                                  isOpen={!trade.exit_time}
                                />
                              </div>
                            {/if}
                            <span
                              class="pnl-tag {trade.pnl >= 0
                                ? trade.pnl === null
                                  ? ''
                                  : 'profit'
                                : 'loss'}"
                            >
                              {trade.pnl === null || trade.pnl === undefined
                                ? 'NA'
                                : (trade.pnl >= 0 ? '+' : '') +
                                  (typeof trade.pnl === 'number'
                                    ? trade.pnl.toFixed(2)
                                    : trade.pnl)}
                            </span>
                            <button
                              class="sync-btn-card"
                              on:click|stopPropagation={() => syncSingleTrade(trade.id)}
                              title="重新整理此交易資料"
                            >
                              <svg
                                width="12"
                                height="12"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2.5"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path
                                  d="M21 3v5h-5"
                                /></svg
                              >
                            </button>
                            {#if !timeGroup.trades[0]?.ticket?.startsWith('ctrader-')}
                              <button
                                class="icon-btn delete"
                                on:click|stopPropagation={() => deleteTradeGroup(timeGroup)}
                                >🗑️</button
                              >
                            {/if}
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
                              <strong class="bullet">
                                {trade.bullet_size && trade.bullet_size > 0
                                  ? trade.bullet_size.toFixed(1)
                                  : 'NA'}
                              </strong>
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
    const { accountId } = e.detail || {};
    // 自動選取新建立的帳號並整頁重整以確保所有元件同步
    if (accountId) {
      selectedAccountId.set(parseInt(accountId));
    }
    window.location.reload();
  }}
/>

<BatchShareModal
  bind:show={showBatchShareModal}
  resourceTitle={currentAccount ? currentAccount.name + '_Shared' : 'SharedAccount'}
  on:startSelection={startSelection}
/>

<SyncOptionsModal
  show={showSyncOptionsModal}
  on:close={() => (showSyncOptionsModal = false)}
  on:sync={e => handleSync(e.detail)}
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
    gap: 0.75rem;
    margin-left: 0.5rem;
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
    padding-top: 0.8rem;
    margin-top: 0.4rem;
    border-top: 1px dashed #e2e8f0;
    flex-basis: 100%;
    width: 100%;
  }

  .sync-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    text-transform: uppercase;
    font-size: 0.65rem;
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-weight: 800;
    line-height: 1;
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

  .sync-icon-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    color: #6366f1;
  }

  .sync-icon-btn:hover:not(:disabled) {
    background: rgba(99, 102, 241, 0.15);
    transform: scale(1.1);
  }

  .sync-icon-btn:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  .sync-icon-btn.syncing .btn-icon {
    display: inline-block;
    animation: rotate 2s linear infinite;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.3rem 0.75rem;
    border-radius: 10px;
    font-size: 0.75rem;
    font-weight: 700;
    line-height: 1;
  }
  .badge-info {
    background: var(--nav-group-bg);
    color: var(--primary);
    border: 1px solid var(--border-color);
  }
  .badge-mt5 {
    background: #f1f5f9;
    color: #475569;
    border: 1px solid #e2e8f0;
  }
  /* Dark mode overrides for specific light-colored badges if needed */
  :global(body.dark-mode) .badge-mt5 {
    background: #1e293b;
    color: #94a3b8;
    border-color: #334155;
  }
  .badge-ctrader {
    background: rgba(16, 185, 129, 0.1);
    color: #10b981;
    border: 1px solid rgba(16, 185, 129, 0.2);
  }
  .badge-success {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border: 1px solid rgba(34, 197, 94, 0.2);
  }
  .badge-danger {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.2);
  }
  .badge-utc {
    background: var(--nav-group-bg);
    color: var(--text-muted);
    border: 1px solid var(--border-color);
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
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #6366f1;
    color: white;
    padding: 0.4rem 1rem;
    border-radius: 20px;
    font-size: 0.85rem;
    font-weight: 700;
    white-space: nowrap;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3);
    line-height: 1;
  }

  .daily-stats {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    margin-left: 0.75rem;
    padding: 0.4rem 1rem;
    background: rgba(255, 255, 255, 0.95);
    border-radius: 20px;
    border: 1px solid #e2e8f0;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }

  :global(body.dark-mode) .daily-stats {
    background: rgba(30, 41, 59, 0.95);
    border-color: #334155;
  }

  .stat-item {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.8rem;
    font-weight: 600;
  }

  .stat-label {
    color: #64748b;
    font-size: 0.75rem;
    font-weight: 500;
  }

  :global(body.dark-mode) .stat-label {
    color: #94a3b8;
  }

  .stat-value {
    font-weight: 700;
    font-size: 0.85rem;
  }

  .stat-detail {
    font-size: 0.7rem;
    color: #94a3b8;
    font-weight: 500;
  }

  .stat-divider {
    color: #cbd5e1;
    font-weight: 300;
  }

  :global(body.dark-mode) .stat-divider {
    color: #475569;
  }

  /* 勝率顏色 */
  .stat-item.win-rate.high-win .stat-value {
    color: #22c55e;
  }

  .stat-item.win-rate.low-win .stat-value {
    color: #ef4444;
  }

  .stat-item.win-rate:not(.high-win):not(.low-win) .stat-value {
    color: #f59e0b;
  }

  /* 盈虧顏色 */
  .stat-item.pnl.profit .stat-value {
    color: #22c55e;
  }

  .stat-item.pnl.loss .stat-value {
    color: #ef4444;
  }

  .stat-item.pnl.floating .stat-value {
    color: #6366f1; /* 浮收使用紫色/藍色區分 */
  }

  .stat-item.pnl.floating.profit .stat-value {
    color: #10b981;
  }

  .stat-item.pnl.floating.loss .stat-value {
    color: #f43f5e;
  }

  .day-card-container {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 1.5rem;
    background: var(--card-bg);
    padding: 1.5rem;
    border-radius: 20px;
    border: 1px solid var(--border-color);
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.03);
    margin-top: 1.25rem; /* 新增 margin-top 防止與上方日期/統計重疊 */
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

  /* Card Items */
  .plan-item-card,
  .trade-item-card {
    background: var(--card-bg);
    border-radius: 12px;
    padding: 1.25rem;
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid var(--border-color);
    position: relative;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Ongoing Trade Animation */
  @keyframes float {
    0% { transform: translateY(0px); }
    50% { transform: translateY(-6px); }
    100% { transform: translateY(0px); }
  }

  @keyframes pulse-bg {
    0% { opacity: 0.6; }
    50% { opacity: 1; }
    100% { opacity: 0.6; }
  }

  .trade-item-card.is-ongoing,
  .trade-time-group.is-multi.is-ongoing {
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.08) 0%, rgba(168, 85, 247, 0.08) 100%);
    border-color: rgba(99, 102, 241, 0.4);
    border-style: solid;
    animation: float 4s ease-in-out infinite;
    box-shadow: 0 15px 35px rgba(99, 102, 241, 0.12);
    z-index: 5;
  }

  .trade-item-card.is-ongoing::after,
  .trade-time-group.is-multi.is-ongoing::after {
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

  :global(body.dark-mode) .trade-item-card.is-ongoing,
  :global(body.dark-mode) .trade-time-group.is-multi.is-ongoing {
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.15) 0%, rgba(168, 85, 247, 0.15) 100%);
    border-color: rgba(99, 102, 241, 0.5);
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

  .sync-btn-card {
    position: absolute;
    top: 0.3rem;
    right: 0.3rem;
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    color: #94a3b8;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    opacity: 0;
    z-index: 10;
  }

  .trade-item-card:hover .sync-btn-card,
  .trade-time-group:hover .sync-btn-card {
    opacity: 0.8;
  }

  .sync-btn-card:hover {
    color: var(--primary);
    opacity: 1 !important;
    transform: rotate(30deg);
  }

  .is-multi .sync-btn-card {
    right: 2rem; /* 在組合單中避開可能的垃圾桶 */
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
    color: var(--text-muted);
    width: 30px;
  }

  .tf-steps {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    align-items: center;
  }

  .mini-step {
    padding: 3px 6px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .mini-step.long {
    background: #fef2f2;
    color: #991b1b;
  }
  .mini-step.short {
    background: #f0fdf4;
    color: #166534;
  }

  .step-label {
    white-space: nowrap;
  }

  .step-details {
    display: flex;
    flex-wrap: wrap;
    gap: 3px;
    font-size: 0.6rem;
    align-items: center;
  }
  .s-text {
    background: rgba(99, 102, 241, 0.1);
    color: #6366f1;
    padding: 0 4px;
    border-radius: 3px;
    border: 1px solid rgba(99, 102, 241, 0.2);
    font-weight: 700;
    line-height: 1.2;
  }
  .w-text {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
    padding: 0 4px;
    border-radius: 3px;
    border: 1px solid rgba(245, 158, 11, 0.2);
    font-weight: 700;
    line-height: 1.2;
  }

  .mini-notes {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border-color);
  }

  .mini-notes-title {
    font-size: 0.75rem;
    font-weight: 700;
    color: #64748b;
    margin-bottom: 0.4rem;
  }

  .mini-note-item {
    font-size: 0.8rem;
    color: var(--text-main);
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
    white-space: nowrap;
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

  .session-tag.none {
    background: #f1f5f9;
    color: #94a3b8;
    font-style: italic;
  }

  .trade-title-area {
    display: flex;
    flex-direction: column;
    gap: 0.35rem; /* 稍微分開標籤行與 ID 行 */
  }

  .ticket-tag {
    display: inline-flex;
    align-items: center;
    font-size: 0.7rem; /* 稍微縮小 ID 大小 */
    color: #94a3b8;
    font-family: 'JetBrains Mono', monospace;
    opacity: 0.8;
    line-height: 1;
  }

  .strategy-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    line-height: 1;
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
    align-items: flex-start;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid rgba(244, 114, 182, 0.1);
    flex-wrap: wrap; /* 容許在空間不足時換行 */
    gap: 1rem;
  }

  .group-meta {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap; /* 讓資訊多時能自動排到下一行，避免撐開容器 */
    flex: 1;
    min-width: 0; /* 防止 flex item 溢出 */
  }
  .group-meta::-webkit-scrollbar {
    display: none;
  }

  .multi-indicator {
    background: #f472b6;
    color: white;
    font-size: 0.7rem;
    font-weight: 800;
    padding: 3px 8px;
    border-radius: 6px;
    box-shadow: 0 2px 4px rgba(244, 114, 182, 0.2);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 3px;
    border: 1px solid rgba(255, 255, 255, 0.2);
  }

  /* 組合單核心指標優化 */
  .group-header .info-group {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    background: rgba(255, 255, 255, 0.5);
    padding: 2px 8px;
    border-radius: 6px;
    border: 1px solid rgba(244, 114, 182, 0.1);
  }

  .group-header .info-group .label {
    font-size: 0.65rem;
    color: #f472b6;
    font-weight: 700;
  }

  .group-header .info-group strong {
    font-size: 0.85rem;
    color: var(--text-main);
    font-family: 'JetBrains Mono', monospace;
  }

  /* Dark Mode Overrides for Multi-trade Groups */
  :global(body.dark-mode) .trade-time-group.is-multi {
    background: rgba(244, 114, 182, 0.08);
    border-color: rgba(244, 114, 182, 0.4);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  }

  :global(body.dark-mode) .group-header .info-group {
    background: rgba(0, 0, 0, 0.2);
    border-color: rgba(244, 114, 182, 0.2);
  }

  :global(body.dark-mode) .group-header {
    border-bottom-color: rgba(244, 114, 182, 0.15);
  }

  :global(body.dark-mode) .partial-close-row:not(:last-child) {
    border-bottom-color: rgba(255, 255, 255, 0.05);
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
    background: var(--card-bg);
    padding: 0.75rem;
    border-radius: 10px;
    border: 1px solid rgba(244, 114, 182, 0.1);
  }

  .partial-close-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: #64748b;
    padding: 8px 0;
    flex-wrap: wrap; /* 行內內容過多時自動換行 */
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
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 800;
    font-size: 0.95rem;
    line-height: 1;
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
    border-top: 1px dashed var(--border-color);
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

  .header-sparkline {
    margin: 0 0.75rem;
    display: flex;
    align-items: center;
    opacity: 0.8;
  }

  .btn-icon svg {
    display: block;
  }

  .partial-sparkline {
    margin-left: 0.5rem;
    display: flex;
    align-items: center;
    opacity: 0.7;
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
    padding: 0.5rem 1.2rem;
    border-radius: 12px;
    font-size: 0.95rem;
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-main);
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.05);
  }

  .small-action-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(0, 0, 0, 0.08);
    border-color: var(--primary);
  }

  .small-action-btn.share {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    color: var(--text-main);
  }

  .small-action-btn.share .btn-icon {
    color: #818cf8;
  }

  .small-action-btn.share:hover {
    background: var(--nav-group-bg);
    border-color: #818cf8;
    box-shadow: 0 5px 15px rgba(129, 140, 248, 0.15);
  }

  .small-action-btn.plan {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    color: var(--text-main);
  }

  .small-action-btn.plan .btn-icon {
    color: #fb923c; /* Orange-ish to match the screenshot */
  }

  .small-action-btn.plan:hover {
    background: var(--nav-group-bg);
    border-color: #fb923c;
    box-shadow: 0 5px 15px rgba(251, 146, 60, 0.15);
  }

  .small-action-btn.trade {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    color: var(--text-main);
  }

  .small-action-btn.trade .btn-icon {
    color: #facc15; /* Yellowish to match the screenshot */
  }

  .small-action-btn.trade:hover {
    background: var(--nav-group-bg);
    border-color: #facc15;
    box-shadow: 0 5px 15px rgba(250, 204, 21, 0.15);
  }

  .sync-status-info {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding-left: 0.8rem;
    border-left: 1px solid var(--border-color);
    margin-left: 0.5rem;
  }

  .sync-badge {
    font-size: 0.7rem;
    font-weight: 700;
    padding: 0.25rem 0.6rem;
    border-radius: 6px;
    background: var(--nav-group-bg);
    color: var(--text-muted);
    border: 1px solid var(--border-color);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .sync-badge.success {
    background: #dcfce7;
    color: #15803d;
    border-color: #bbf7d0;
  }

  .sync-badge.syncing {
    background: #eff6ff;
    color: #1d4ed8;
    border-color: #dbeafe;
    animation: pulse 2s infinite;
  }

  @keyframes pulse {
    0% {
      opacity: 1;
    }
    50% {
      opacity: 0.6;
    }
    100% {
      opacity: 1;
    }
  }

  .sync-time {
    font-size: 0.7rem;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', monospace;
    opacity: 0.8;
  }

  .sync-icon-btn {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    width: 32px;
    height: 32px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;
    color: var(--text-muted);
    padding: 0;
  }

  .sync-icon-btn:hover:not(:disabled) {
    background: var(--nav-group-bg);
    color: var(--primary);
    border-color: var(--primary);
    transform: rotate(15deg);
  }

  .sync-icon-btn.syncing .btn-icon {
    animation: rotate 2s linear infinite;
  }

  .account-details-inline {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-left: 0.5rem;
  }

  .storage-info-chip,
  .login-id-chip {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    background: var(--nav-group-bg);
    padding: 0.25rem 0.6rem;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .storage-info-chip .label,
  .login-id-chip .label {
    font-weight: 600;
    opacity: 0.8;
  }

  .storage-info-chip .value,
  .login-id-chip .value {
    color: var(--text-main);
    font-weight: 700;
    font-family: 'JetBrains Mono', monospace;
  }

  .chip-icon {
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

  /* 批次選取樣式 */
  .selection-bar {
    position: fixed;
    top: 2rem;
    left: 50%;
    transform: translateX(-50%);
    background: #1e293b;
    color: white;
    padding: 1rem 2rem;
    border-radius: 99px;
    display: flex;
    align-items: center;
    gap: 2rem;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3);
    z-index: 1000;
    animation: barSlideDown 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes barSlideDown {
    from {
      top: -5rem;
      opacity: 0;
    }
    to {
      top: 2rem;
      opacity: 1;
    }
  }

  .selection-info {
    font-size: 0.95rem;
    font-weight: 500;
  }

  .selection-info strong {
    color: #818cf8;
    font-size: 1.1rem;
    margin: 0 0.2rem;
  }

  .selection-actions {
    display: flex;
    gap: 0.75rem;
  }

  .btn-sm {
    padding: 0.4rem 1rem;
    font-size: 0.85rem;
    border-radius: 99px;
  }

  .card-selection-overlay {
    position: absolute;
    top: 0.75rem;
    left: 0.75rem;
    z-index: 20;
  }

  .selection-checkbox {
    width: 20px;
    height: 20px;
    cursor: pointer;
    accent-color: #6366f1;
  }

  .day-check {
    width: 24px;
    height: 24px;
    margin-bottom: 0.5rem;
  }

  .plan-item-card.selected,
  .trade-item-card.selected,
  .trade-time-group.selected {
    border-color: #6366f1 !important;
    background: #f5f3ff !important;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  /* 過濾器樣式 */
  .filter-section {
    margin: 0.5rem 0 1.5rem 0;
    padding: 0 1rem;
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

  .sub-filter-scroll-wrapper {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    overflow-x: auto;
    scrollbar-width: none; /* Firefox */
  }

  .sub-filter-scroll-wrapper::-webkit-scrollbar {
    display: none; /* Chrome/Safari */
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
  /* 日期篩選樣式 */
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
  
  :global(body.dark-mode) .date-input:focus {
    background: #1e293b;
    border-color: #818cf8;
  }

  .date-sep {
    color: #94a3b8;
    font-weight: bold;
  }

  .divider-horizontal {
    height: 1px;
    width: 100%;
    background: rgba(0, 0, 0, 0.05);
    margin: 0.5rem 0;
  }

  :global(body.dark-mode) .divider-horizontal {
    background: rgba(255, 255, 255, 0.05);
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-5px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
