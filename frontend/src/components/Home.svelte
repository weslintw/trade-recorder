<script>
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { navigate, Link } from 'svelte-routing';
  import { tradesAPI, dailyPlansAPI, imagesAPI, sharesAPI, accountsAPI } from '../lib/api';
  import { selectedSymbol, selectedAccountId, accounts } from '../lib/stores';
  import { MARKET_SESSIONS, SYMBOLS, TIMEFRAMES } from '../lib/constants';
  import {
    determineMarketSession,
    getStrategyLabel,
    parseJSONSafe,
    calculateBulletSize,
  } from '../lib/utils';
  import AccountModal from './AccountModal.svelte';
  import Sparkline from './Sparkline.svelte';
  import BatchShareModal from './BatchShareModal.svelte';
  import SyncOptionsModal from './SyncOptionsModal.svelte';
  import PlanSummaryTable from './PlanSummaryTable.svelte';
  import ImageAnnotator from './ImageAnnotator.svelte';

  let showAnnotator = false;
  let enlargedOriginalImage = null;
  let enlargedImageContext = null; // { tradeId, imageIndex, type: 'general' | 'expert' | ... }

  let groupedData = [];
  let loading = true;
  let loadError = null; // 新增：錯誤狀態
  let pagination = {
    page: 1,
    page_size: 100,
    total: 0,
  };
  let loadingMessage = '正在啟動時光機...';

  let todayString = new Date().toLocaleDateString('en-CA'); // 使用 YYYY-MM-DD 格式的本地日期
  let selectedImage = null;
  let modalTitle = '';
  let isSyncing = false;
  let showAccountModal = false;
  let showBatchShareModal = false;
  let pollingTimeout;
  let currentPollingInterval = 60000; // 預設閒置輪詢改為 60 秒 (僅作為 WebSocket 的備援)
  let ws;
  let isDocumentHidden = false;
  let handleVisibilityChange;

  // 批次分享相關狀態
  let selectionMode = false;
  let selectedTrades = new Set();
  let selectedPlans = new Set();
  let isSharing = false;
  let generatedShareToken = '';
  let showSyncOptionsModal = false;
  let activeFilterType = 'all'; // 'all', 'expert', 'elite', 'legend'
  let activeSubFilter = null;
  let activeColorFilter = null; // 紅綠燈過濾
  let activeExitFilter = 'all'; // 'all', 'tp', 'sl'
  let activeSideFilter = 'all'; // 'all', 'long', 'short'
  let plans = [];
  let trades = [];
  let globalSummary = {
    total_count: 0,
    win_count: 0,
    total_pnl: 0,
    expert_count: 0,
    elite_count: 0,
    legend_count: 0,
    green_count: 0,
    yellow_count: 0,
    red_count: 0,
  };

  // 智能快取系統
  let dataCache = {
    key: null, // Cache key: `${accountId}_${symbol}_${startDate}_${endDate}`
    plans: [],
    trades: [],
    summary: null,
    timestamp: null,
  };

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

  const PAGE_SIZE_OPTIONS = [50, 100, 200, 300, 400, 500, 1000, 2000, 3000, 5000, 10000];

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
    // 客戶端篩選：不需要重新載入資料，reactive statement 會自動更新顯示
    pagination.page = 1;
    console.log(`[Filter] Strategy filter changed to: ${activeFilterType}`);
  }

  function toggleSubFilter(value) {
    if (activeSubFilter === value) {
      activeSubFilter = null;
    } else {
      activeSubFilter = value;
    }
    // 客戶端篩選：不需要重新載入資料，reactive statement 會自動更新顯示
    pagination.page = 1;
    console.log(`[Filter] Sub-filter changed to: ${activeSubFilter}`);
  }

  function selectColorFilter(color) {
    if (activeColorFilter === color) {
      activeColorFilter = null;
    } else {
      activeColorFilter = color;
    }
    // 切換顏色時，重置回第一頁並重新載入資料
    pagination.page = 1;
    loadData();
  }

  $: filteredGroupedData = (() => {
    try {
      if (!groupedData || !Array.isArray(groupedData)) return [];
      return applyFilters(
        groupedData,
        activeFilterType,
        activeSubFilter,
        activeColorFilter,
        activeExitFilter,
        activeSideFilter
      );
    } catch (err) {
      console.error('[Home] filteredGroupedData error:', err);
      return [];
    }
  })();
  $: isAllMode = activeFilterType === 'all' && !activeSubFilter;
  $: statsLabel = isAllMode ? '全部統計：' : '篩選統計：';
  $: activeStrategyLabel = getStrategyLabel(activeFilterType) || '';

  // Client-side Pagination Logic
  // 我們基於「天」數進行分頁，每頁顯示 7 天的數據，避免一次渲染過多
  const DAYS_PER_PAGE = 7;
  $: totalPages =
    Math.ceil((filteredGroupedData ? filteredGroupedData.length : 0) / DAYS_PER_PAGE) || 1;

  $: paginatedGroupedData = (() => {
    if (!filteredGroupedData) return [];
    // console.log(`[Pagination] Recalculating page ${pagination.page}, total items: ${filteredGroupedData.length}`);
    const start = (pagination.page - 1) * DAYS_PER_PAGE;
    // 確保 page 不會超過範圍 (例如從很多頁的 filter 切換到很少頁的 filter)
    if (start >= filteredGroupedData.length && pagination.page > 1) {
      // 異步將頁碼重置為 1，避免在 render 過程中修改 state 導致錯誤
      setTimeout(() => (pagination.page = 1), 0);
      return filteredGroupedData.slice(0, DAYS_PER_PAGE);
    }
    return filteredGroupedData.slice(start, start + DAYS_PER_PAGE);
  })();

  $: filteredStats = (() => {
    let stats = {
      total: 0,
      wins: 0,
      winRate: '0.0',
      totalPnl: '0.00',
      hasTrades: false,
      green: 0,
      yellow: 0,
      red: 0,
      expert: 0,
      elite: 0,
      legend: 0,
    };
    if (!filteredGroupedData) return stats;

    let allTrades = [];
    for (let i = 0; i < filteredGroupedData.length; i++) {
      const day = filteredGroupedData[i];
      if (day.groupedTrades) {
        for (let j = 0; j < day.groupedTrades.length; j++) {
          const group = day.groupedTrades[j];
          if (group.trades) {
            for (let k = 0; k < group.trades.length; k++) {
              allTrades.push(group.trades[k]);
            }
          }
        }
      }
    }

    let total = 0;
    let wins = 0;
    let totalPnlValue = 0;

    for (let i = 0; i < allTrades.length; i++) {
      const t = allTrades[i];
      if (t.color_tag === 'green') stats.green++;
      else if (t.color_tag === 'yellow') stats.yellow++;
      else if (t.color_tag === 'red') stats.red++;

      const strat = String(t.entry_strategy || '').toLowerCase();
      if (strat === 'expert' || strat === '達人') stats.expert++;
      else if (strat === 'elite' || strat === '菁英') stats.elite++;
      else if (strat === 'legend' || strat === '傳奇') stats.legend++;

      if (t.trade_type === 'actual' && t.exit_time && t.pnl !== null && t.pnl !== undefined) {
        total++;
        if (t.pnl > 0) wins++;
        totalPnlValue += t.pnl || 0;
      }
    }

    stats.total = total;
    stats.wins = wins;
    stats.hasTrades = allTrades.length > 0;
    if (total > 0) stats.winRate = ((wins * 100) / total).toFixed(1);
    stats.totalPnl = totalPnlValue.toFixed(2);
    return stats;
  })();

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

    const formatDir = dirData => {
      if (!dirData?.wave_numbers?.length) return '';
      const nums = dirData.wave_numbers;
      const highlight = dirData.wave_highlight;
      return nums
        .map((n, i) => {
          const isHighlight = n.toString() === highlight?.toString();
          const val = isHighlight ? `[${n}]` : n;
          return (i > 0 ? ' => ' : '') + val;
        })
        .join('');
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
      const w = trend.wave_numbers
        .map((n, i) => {
          const isHighlight = n.toString() === trend.wave_highlight?.toString();
          const val = isHighlight ? `[${n}]` : n;
          return (i > 0 ? ' => ' : '') + val;
        })
        .join('');
      waves.push(w);
    }

    if (waves.length === 0) return '';
    return [...new Set(waves)].join(' | ');
  }

  function applyFilters(data, type, sub, color, exitFilter, sideFilter) {
    if (!data || !Array.isArray(data)) {
      console.log('[applyFilters] Invalid data, returning empty array');
      return [];
    }

    // 如果是全選且無任何子過濾，直接返回
    if (
      type === 'all' &&
      !sub &&
      !color &&
      (!exitFilter || exitFilter === 'all') &&
      (!sideFilter || sideFilter === 'all')
    )
      return data;

    const debug = !!sub;
    const cleanSub = sub ? String(sub).trim() : null;

    return data
      .map(day => {
        // 盤面規劃不參與篩選，始終顯示完整內容
        const filteredPlans = day.plans;

        const filteredGroupedTrades = day.groupedTrades
          .map(group => {
            const filteredTrades = group.trades.filter(trade => {
              // TP/SL Filter
              if (exitFilter === 'tp') {
                if (!(trade.pnl > 0)) return false;
              } else if (exitFilter === 'sl') {
                if (!(trade.pnl < 0)) return false;
              }

              // Side Filter (Long/Short)
              if (sideFilter === 'long') {
                if (trade.side !== 'long' && trade.side !== 'buy') return false;
              } else if (sideFilter === 'short') {
                if (trade.side !== 'short' && trade.side !== 'sell') return false;
              }
              // 1. 策略類型匹配 (高度容錯)
              const tStrat = String(trade.entry_strategy || trade.strategy || '')
                .trim()
                .toLowerCase();
              const fType = type.toLowerCase();

              const stratMatch =
                fType === 'all' ||
                (fType === 'expert' && (tStrat === 'expert' || tStrat === '達人')) ||
                (fType === 'elite' && (tStrat === 'elite' || tStrat === '菁英')) ||
                (fType === 'legend' && (tStrat === 'legend' || tStrat === '傳奇')) ||
                tStrat === fType;

              if (!stratMatch) return false;

              // Color Tag Filter (新增)
              if (color) {
                if (trade.color_tag !== color) return false;
              }

              if (!cleanSub) return true;

              // 2. 終極修正：全物件透視搜尋 (JSON String Search)
              // 直接看整筆交易的原始資料裡面有沒有「甲」
              try {
                const tradeStr = JSON.stringify(trade);
                const isMatch = tradeStr.includes(cleanSub);

                if (debug && !isMatch && stratMatch) {
                  console.warn(
                    `[Filter Debug] Trade #${trade.id} Fail. '${cleanSub}' not found in:`,
                    tradeStr
                  );
                }

                return isMatch;
              } catch (e) {
                return false;
              }
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
      .filter(day => {
        // 如果有設定策略篩選 (type !== 'all')、子篩選 (搜尋模式)、顏色篩選或 TP/SL 篩選，
        // 則嚴格只顯示有交易紀錄命中的日期，隱藏那些只有盤面規劃但沒有符合交易的空區塊
        if (
          type !== 'all' ||
          cleanSub ||
          color ||
          (exitFilter && exitFilter !== 'all') ||
          (sideFilter && sideFilter !== 'all')
        ) {
          return day.groupedTrades.length > 0;
        }
        // 一般全覽模式：顯示有規劃或有交易的日期
        return day.plans.length > 0 || day.groupedTrades.length > 0;
      });
  }

  // 追蹤當前選取的帳號詳情
  $: currentAccount = $accounts.find(a => a.id === $selectedAccountId);

  let loadController; // To abort in-flight requests

  // 追蹤頁面可見性，避免背景切回時重複載入
  let isPageVisible = !document.hidden;
  let lastReactiveValues = { accountId: null, symbol: null };

  // 響應式：當帳號或品種改變時，自動重新載入資料 (加上 Debounce 防止連點)
  let debounceTimer;
  $: if ($selectedAccountId && $selectedSymbol) {
    // 檢查是否真的有變化（避免 Svelte 重新評估時觸發）
    const hasChanged =
      lastReactiveValues.accountId !== $selectedAccountId ||
      lastReactiveValues.symbol !== $selectedSymbol;

    if (hasChanged) {
      // 更新追蹤值
      lastReactiveValues = {
        accountId: $selectedAccountId,
        symbol: $selectedSymbol,
      };

      if (debounceTimer) clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        console.log(
          `🏠 [Reactive] Account/Symbol changed: acc=${$selectedAccountId}, sym=${$selectedSymbol}, pageVisible=${isPageVisible}`
        );
        // 重設分頁
        pagination.page = 1;
        loadData();
      }, 500); // 500ms 防抖
    }
  } else {
    console.warn(
      `🏠 [Reactive] SKIPPED loadData: selectedAccountId=${$selectedAccountId}, selectedSymbol=${$selectedSymbol}`
    );
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
    const callId = ++loadDataCallCount;
    const now = Date.now();
    console.log(`🔵 [${INSTANCE_ID}] loadData #${callId} called, silent: ${silent}`);

    if (silent && loading && now - (window._lastLoadDataTime || 0) < 500) {
      return;
    }

    if (loadController) {
      loadController.abort();
    }

    window._lastLoadDataTime = now;
    activeLoadCallId = callId;
    loadController = new AbortController();
    const { signal } = loadController;

    try {
      // 預防空狀態：如果沒有數據且 silent=true，強制轉為 loading=true 讓用戶知道正在載入
      if (silent && (!dataCache.trades || dataCache.trades.length === 0)) {
        console.log(
          `[${INSTANCE_ID}] Silent load promoted to explicit load because cache is empty.`
        );
        silent = false;
      }

      if (!silent) {
        loading = true;
        loadError = null;
        loadingMessage = '正在準備連線...';
        // 不要在這裡清空 groupedData，這樣會導致畫面閃爍
        // groupedData = [];

        setTimeout(() => {
          if (loading && activeLoadCallId === callId) {
            console.warn(`[${INSTANCE_ID}] Loading state safety timeout (10s). Forcing OFF.`);
            loading = false;
          }
        }, 10000);
      }

      console.log(`[${INSTANCE_ID}] loadData #${callId} started for symbol:`, $selectedSymbol);

      const symbol = $selectedSymbol;
      todayString = new Date().toISOString().slice(0, 10);

      // 生成 cache key（基於 account, symbol, date range）
      const cacheKey = `${$selectedAccountId}_${symbol}_${customStartDate || 'all'}_${customEndDate || 'all'}`;
      const isCacheValid =
        dataCache.key === cacheKey &&
        dataCache.timestamp &&
        Date.now() - dataCache.timestamp < 300000; // 5分鐘有效期

      console.log(
        `[${INSTANCE_ID}] Cache check: key=${cacheKey}, valid=${isCacheValid}, hasData=${dataCache.trades.length > 0}`
      );

      let globalSummaryData = null;

      // 如果 cache 有效，直接使用
      if (isCacheValid && dataCache.trades.length > 0) {
        console.log(
          `[${INSTANCE_ID}] Using cached data (${dataCache.trades.length} trades, ${dataCache.plans.length} plans)`
        );
        plans = dataCache.plans;
        trades = dataCache.trades;
        globalSummaryData = dataCache.summary;

        // 即使使用 cache，也要更新 pagination（因為可能有 filter 改變）
        // 但這裡我們先簡單處理，後續可以優化
      } else {
        // Cache 無效，需要請求 API
        console.log(`[${INSTANCE_ID}] Cache invalid or empty, starting progressive fetch...`);

        // API 請求區塊
        try {
          loadingMessage = `正在快速載入最新資料...`;

          // 階段一：快速並行請求
          // Plans 通常不多，一次抓完；Trades 先抓 100 筆讓畫面出來
          const [plansRes, tradesFastRes] = await Promise.all([
            dailyPlansAPI.getAll(
              {
                account_id: $selectedAccountId,
                symbol,
                page_size: 10000,
                page: 1,
                start_date: activeDateRange === 'all' ? undefined : customStartDate,
                end_date:
                  activeDateRange === 'all'
                    ? undefined
                    : customEndDate
                      ? customEndDate + ' 23:59:59'
                      : undefined,
              },
              signal
            ),
            tradesAPI.getAll(
              {
                account_id: $selectedAccountId,
                symbol,
                page_size: 100, // 🚀 快速載入：只抓前 100 筆
                page: 1,
                color_tag: activeColorFilter || undefined,
              },
              signal
            ),
          ]);

          console.log(`[${INSTANCE_ID}] Fast fetch complete.`);

          plans = (Array.isArray(plansRes.data) ? plansRes.data : plansRes.data?.data) || [];
          const fastTrades =
            (Array.isArray(tradesFastRes.data) ? tradesFastRes.data : tradesFastRes.data?.data) ||
            [];

          // 立即更新畫面
          trades = fastTrades;
          globalSummaryData = tradesFastRes.data?.summary || null;

          if (tradesFastRes.data?.pagination) {
            pagination.total = tradesFastRes.data.pagination.total;
          }

          // 🔓 解除 Loading 狀態 (前提是沒有報錯)
          if (!silent) {
            loading = false;
            // 如果我們強制關閉了 loading，也要確保從 activeLoadCallId 機制中解除 timeout 警告?
            // 其實不需要特別做，timeout 只是會在 10s 後把 loading 強制設為 false
          }

          // 階段二：背景完整同步 (Background Full Sync)
          // 如果總數大於 100，或者為了確保 Cache 完整性，我們再抓一次全量
          if (pagination.total > 100 || fastTrades.length === 100) {
            console.log(`[${INSTANCE_ID}] Starting background fetch for full dataset...`);

            // 使用 Promise chain 進行背景處理，不阻塞當前執行
            tradesAPI
              .getAll(
                {
                  account_id: $selectedAccountId,
                  symbol,
                  page_size: 10000, // 抓所有
                  page: 1,
                  color_tag: activeColorFilter || undefined,
                },
                signal
              )
              .then(tradesFullRes => {
                if (signal.aborted) return;

                const fullTrades =
                  (Array.isArray(tradesFullRes.data)
                    ? tradesFullRes.data
                    : tradesFullRes.data?.data) || [];
                console.log(
                  `[${INSTANCE_ID}] Background fetch complete. Full Trades: ${fullTrades.length}`
                );

                // 更新數據 (Svelte 會自動處理 Diff 無縫更新)
                trades = fullTrades;

                // 如果後端有更準確的 summary，也可以更新
                if (tradesFullRes.data?.summary) {
                  globalSummaryData = tradesFullRes.data.summary;
                }

                // 只有在完整數據抓回來後，才更新 Cache
                dataCache = {
                  key: cacheKey,
                  plans: plans,
                  trades: fullTrades,
                  summary: globalSummaryData,
                  timestamp: Date.now(),
                };
              })
              .catch(err => {
                if (err.name !== 'CanceledError' && err.name !== 'AbortError') {
                  console.error('Background sync failed', err);
                }
              });
          } else {
            // 如果數據少於 100 筆，那第一階段就是完整的了
            dataCache = {
              key: cacheKey,
              plans: plans,
              trades: fastTrades,
              summary: globalSummaryData,
              timestamp: Date.now(),
            };
          }
        } catch (apiErr) {
          if (apiErr.name !== 'CanceledError' && apiErr.name !== 'AbortError') {
            console.error(`[${INSTANCE_ID}] API Error in loadData #${callId}:`, apiErr);
            // 如果 API 真的報錯，也讓使用者知道
            if (activeLoadCallId === callId) {
              loadError = {
                message: '讀取資料失敗',
                detail: `API Error: ${apiErr.message || 'Unknown'}`,
              };
            }
          }
        }
      }

      // 數據過濾與去重
      const seenTickets = new Set();
      const uniqueTrades = [];
      trades.forEach(t => {
        if (t.ticket && seenTickets.has(t.ticket)) return;
        if (t.ticket) seenTickets.add(t.ticket);
        uniqueTrades.push(t);
      });

      // 數據分組邏輯
      loadingMessage = `正在分組並解析數據...`;
      const dateMap = {};
      dateMap[todayString] = { date: todayString, plans: [], groupedTrades: [] };

      plans.forEach(plan => {
        plan.trendData = parseJSONSafe(plan.trend_analysis, {});
        if (!plan.plan_date) return;
        try {
          const d = new Date(plan.plan_date);
          if (isNaN(d.getTime())) return;
          const ds = d.toISOString().slice(0, 10);
          if (!dateMap[ds]) dateMap[ds] = { date: ds, plans: [], groupedTrades: [] };
          dateMap[ds].plans.push(plan);
        } catch (e) {}
      });

      uniqueTrades.forEach(trade => {
        if (!trade.entry_time) return;
        try {
          const d = new Date(trade.entry_time);
          if (isNaN(d.getTime())) return;
          const ds = d.toISOString().slice(0, 10);
          if (!dateMap[ds]) dateMap[ds] = { date: ds, plans: [], groupedTrades: [] };

          const entryTimeKey = trade.entry_time;
          let timeGroup = dateMap[ds].groupedTrades.find(g => g.entry_time === entryTimeKey);
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
            dateMap[ds].groupedTrades.push(timeGroup);
          }
          timeGroup.trades.push(trade);
          timeGroup.summary.totalPnl += trade.pnl || 0;
          timeGroup.summary.totalLot += trade.lot_size || 0;
        } catch (e) {}
      });

      // 排序與更新顯示
      const sortedResult = Object.values(dateMap).sort((a, b) => b.date.localeCompare(a.date));
      sortedResult.forEach(day => {
        day.groupedTrades.sort((a, b) => {
          const getT = g =>
            g.trades.some(t => !t.exit_time)
              ? Infinity
              : Math.max(...g.trades.map(t => new Date(t.exit_time || 0).getTime()));
          return getT(b) - getT(a);
        });
      });

      if (activeLoadCallId === callId) {
        groupedData = sortedResult;
        if (globalSummaryData) {
          globalSummary = globalSummaryData;
        }
      }
    } catch (globalErr) {
      if (globalErr.name !== 'CanceledError' && globalErr.name !== 'AbortError') {
        console.error('Fatal loadData error:', globalErr);
        if (activeLoadCallId === callId) {
          loadError = { message: '系統錯誤', detail: globalErr.message };
        }
      }
    } finally {
      if (activeLoadCallId === callId) {
        loading = false;
        loadController = null;
      }
    }
  }

  // 獨立更新帳號狀態 (不觸發 loading UI)
  let lastAccountsData = null;
  let isRefreshingAccounts = false;
  let refreshAccountsController = null;
  async function refreshAccounts(silent = false) {
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
        if (!silent) {
          alert('更新帳號資訊失敗: ' + (e.message || '網路超時'));
        }
      }
    } finally {
      isRefreshingAccounts = false;
    }
  }

  // 小優化：帳號列表改為 30 秒才允許自動刷新一次
  let lastRefreshTime = 0;
  async function safeRefreshAccounts() {
    const now = Date.now();
    if (now - lastRefreshTime < 30000) return;
    lastRefreshTime = now;
    return refreshAccounts(true);
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
            console.log('🚀 [Realtime] Trade update detected, invalidating cache...');
            if (isDocumentHidden) {
              console.log('[Realtime] Tab hidden, deferring reload until visible.');
              // 標記 cache 為過期，這樣切換回來時會自動重載
              if (dataCache) dataCache.timestamp = 0;
              return;
            }
            // 強制讓 cache 失效並重新載入
            if (dataCache) dataCache.timestamp = 0;
            loadData(true);
            refreshAccounts();
          }
        } else if (msg.type === 'PRICE_UPDATE') {
          if (!msg.account_id || msg.account_id === $selectedAccountId) {
            // 優化：直接更新記憶體中的盈虧，不要重新執行 loadData()
            const updatedTradesCount = updatePnLInMemory(msg.prices);
            if (updatedTradesCount > 0) {
              // 觸發 Svelte 響應式刷新
              groupedData = groupedData;
            }
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

  // 核心優化：直接更新記憶體中的盈虧，不發起網路請求
  function updatePnLInMemory(prices) {
    if (!prices || !groupedData) return 0;
    let count = 0;

    // 遍歷所有日期組下的交易群組
    groupedData.forEach(day => {
      day.groupedTrades.forEach(group => {
        let groupUpdated = false;
        group.trades.forEach(trade => {
          // 只針對未平倉位 (無平倉價格或平倉時間)
          if (!trade.exit_price && !trade.exit_time && trade.symbol) {
            const updateInfo = prices[trade.ticket] || prices[trade.symbol];
            if (updateInfo) {
              // 更新 PnL
              if (typeof updateInfo === 'number') {
                trade.pnl = updateInfo;
              } else if (updateInfo.pnl !== undefined) {
                trade.pnl = updateInfo.pnl;
              }

              // 更新價格
              if (updateInfo.price) {
                trade.current_price = updateInfo.price;
              }
              count++;
              groupUpdated = true;
            }
          }
        });

        // 重新計算群組總盈虧
        if (groupUpdated) {
          group.summary.totalPnl = group.trades.reduce((sum, t) => sum + (t.pnl || 0), 0);
        }
      });
    });
    return count;
  }
  function throttledLoadData() {
    const now = Date.now();
    if (now - lastRealtimeLoadTime > 5000) {
      lastRealtimeLoadTime = now;
      console.log('📈 [Realtime] Price update detected, refreshing...');
      loadData(true);
      safeRefreshAccounts();
    }
  }

  // 移除自動輪詢迴圈，改為完全依賴 WebSocket (事件驅動) 與 視野切換 (喚醒同步)
  // 原有的 poll 函式將不再循環執行

  // 修改：將初始啟動延後，且增加 loading 檢查
  function startDeferredServices() {
    console.log('[Home] Starting deferred services...');
    initRealtimeNotifications();

    // 如果 WebSocket 斷開，才考慮在一段時間後執行補償刷新，否則不主動輪詢
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.log('[Home] WS is not ready, performing initial sync.');
      loadData(true);
      refreshAccounts();
    }
  }

  onMount(async () => {
    console.log('=== 🏠 Home onMount START ===');

    // Step 0: 清理所有殘留狀態（防止上次未完成的請求干擾）
    console.log('[onMount] Cleaning up any residual state...');

    // 中斷任何殘留的請求控制器
    if (loadController) {
      console.log('[onMount] Aborting residual loadController');
      loadController.abort();
      loadController = null;
    }
    if (refreshAccountsController) {
      console.log('[onMount] Aborting residual refreshAccountsController');
      refreshAccountsController.abort();
      refreshAccountsController = null;
    }

    // 清除全域時間戳記
    window._lastLoadDataTime = 0;

    // 重置載入狀態
    loading = false;
    isRefreshingAccounts = false;

    // 清除任何殘留的計時器
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    if (pollingTimeout) {
      clearTimeout(pollingTimeout);
      pollingTimeout = null;
    }

    console.log('[onMount] Cleanup complete, starting fresh');

    // Step 1: 預加載帳號列表 (優化：如果已有資料就不重複抓取)
    if ($accounts.length === 0) {
      console.log('[onMount] Pre-fetching accounts...');
      await refreshAccounts();
    } else {
      console.log('[onMount] Accounts already in store, skipping redundant fetch.');
    }

    // Step 2: 檢查當前選取的帳號是否有效
    console.log(
      `[onMount] Current state: accounts=${$accounts.length}, selectedAccountId=${$selectedAccountId}, selectedSymbol=${$selectedSymbol}`
    );
    const accountExists = $accounts.some(a => a.id === $selectedAccountId);
    if ($accounts.length > 0) {
      if (!$selectedAccountId || !accountExists) {
        console.log(`[onMount] Auto-selecting first account: ${$accounts[0].id}`);
        selectedAccountId.set($accounts[0].id);
      } else {
        console.log(`[onMount] Account ${$selectedAccountId} is valid, keeping selection`);
        // 主動觸發一次載入，確保即使 Store 值沒變也能載入資料
        loadData();
      }
    } else {
      console.warn('[onMount] NO ACCOUNTS FOUND! User needs to create an account first.');
      loading = false; // No accounts, stop loading spinner
    }

    console.log(
      `[onMount] After account check: selectedAccountId=${$selectedAccountId}, selectedSymbol=${$selectedSymbol}`
    );

    // 初始化日期範圍 (Default to all)
    // setDateRange('all'); // Already default
    // We already call loadData below implicitly or explicitly?
    // Actually the symbol watcher debouncer will call loadData.
    // But we also want to ensure we don't double load.
    // The previous code had setDateRange('1W') here. Remove it.

    // Step 3: 延後啟動即時通知與備援輪詢，避免跟 Initial Data 搶瀏覽器併發連線
    setTimeout(startDeferredServices, 15000); // 增加到 15 秒

    // Restore scroll position
    const savedScrollPos = sessionStorage.getItem('home_scroll_pos');
    if (savedScrollPos) {
      setTimeout(() => {
        window.scrollTo(0, parseInt(savedScrollPos));
        sessionStorage.removeItem('home_scroll_pos');
      }, 300);
    }

    let lastVisibilityChangeTime = 0;
    handleVisibilityChange = () => {
      const hidden = document.hidden;
      const now = Date.now();

      if (isDocumentHidden !== hidden) {
        isDocumentHidden = hidden;
        isPageVisible = !hidden;
        console.log(`🏠 [Visibility] Page is now ${isPageVisible ? 'visible' : 'hidden'}`);

        if (!isDocumentHidden) {
          // 當分頁重新回到視野，檢查是否需要同步
          const timeSinceLastChange = now - lastVisibilityChangeTime;
          lastVisibilityChangeTime = now;

          // 只有在離開超過 5 秒才執行同步，避免快速切換時重複載入
          if (timeSinceLastChange > 5000) {
            console.log('[Home] Back in focus after 5+ seconds, performing catch-up sync...');
            // 使用 silent 模式避免顯示 loading spinner
            setTimeout(() => {
              loadData(true);
              refreshAccounts(true);
            }, 300); // 延遲 300ms 讓 Svelte 的響應式系統先穩定
          } else {
            console.log('[Home] Back in focus quickly, skipping redundant sync.');
          }
        }
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

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
    if (handleVisibilityChange) {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    }
  });

  // Manual reload function for when user changes selection from UI
  export function reloadData() {
    console.log('🔄 Manual reload triggered');
    pagination.page = 1;
    loadData();
  }

  function changePage(newPage) {
    if (newPage < 1 || newPage > totalPages) return;
    pagination.page = newPage;
    window.scrollTo({ top: 0, behavior: 'smooth' });
    // Client-side pagination: no need to reload data
    console.log(`[Pagination] Switched to page ${newPage}`);
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
      return {
        winRate: 0,
        realizedPnl: 0,
        floatingPnl: 0,
        totalPnl: 0,
        wins: 0,
        total: 0,
        hasFloating: false,
      };
    }

    const allTrades = dayGroup.groupedTrades.flatMap(group => group.trades);

    // 只計算已平倉的交易用於勝率和已實現盈虧
    const closedTrades = allTrades.filter(
      trade => trade.exit_time && trade.pnl !== null && trade.pnl !== undefined
    );

    // 計算未平倉的交易（浮動部分）
    const openTrades = allTrades.filter(
      trade => !trade.exit_time && trade.pnl !== null && trade.pnl !== undefined
    );

    const total = closedTrades.length;
    const wins = closedTrades.filter(trade => trade.pnl > 0).length;
    const realizedPnl = closedTrades.reduce((sum, trade) => sum + (trade.pnl || 0), 0);
    const floatingPnl = openTrades.reduce((sum, trade) => sum + (trade.pnl || 0), 0);
    const totalPnl = realizedPnl + floatingPnl;
    const winRate = total > 0 ? (wins / total) * 100 : 0;
    const hasFloating = openTrades.length > 0;

    const green = allTrades.filter(t => t.color_tag === 'green').length;
    const yellow = allTrades.filter(t => t.color_tag === 'yellow').length;
    const red = allTrades.filter(t => t.color_tag === 'red').length;

    const expert = allTrades.filter(t => {
      const s = String(t.entry_strategy || '').toLowerCase();
      return s === 'expert' || s === '達人';
    }).length;
    const elite = allTrades.filter(t => {
      const s = String(t.entry_strategy || '').toLowerCase();
      return s === 'elite' || s === '菁英';
    }).length;
    const legend = allTrades.filter(t => {
      const s = String(t.entry_strategy || '').toLowerCase();
      return s === 'legend' || s === '傳奇';
    }).length;

    return {
      winRate,
      realizedPnl,
      floatingPnl,
      totalPnl,
      wins,
      total,
      hasFloating,
      green,
      yellow,
      red,
      expert,
      elite,
      legend,
    };
  }

  function getMarketSessionLabel(trade) {
    let session = trade.market_session;
    // 如果資料庫中沒有時段資料，根據時間即時計算
    if (!session && trade.entry_time) {
      session = determineMarketSession(trade.entry_time);
    }
    return MARKET_SESSIONS.find(s => s.value === session)?.label || session || '未設定';
  }

  function openImageModal(imagePath, title = '查看圖片', context = null, originalPath = null) {
    if (!imagePath) return;
    modalTitle = title;
    enlargedImageContext = context;
    enlargedOriginalImage = originalPath || imagePath;
    selectedImage =
      imagePath.startsWith('http') || imagePath.startsWith('data:') || imagePath.startsWith('blob:')
        ? imagePath
        : imagesAPI.getUrl(imagePath);
    showAnnotator = false;
  }

  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  async function handleAnnotatedImage(annotatedImageSrc) {
    if (!enlargedImageContext) return;
    try {
      const res = await fetch(annotatedImageSrc);
      const blob = await res.blob();
      const file = new File([blob], 'annotated_home.png', { type: 'image/png' });

      const uploadData = new FormData();
      uploadData.append('image', file);
      uploadData.append('symbol', 'annotated');

      const uploadRes = await imagesAPI.upload(uploadData);
      const serverPath = uploadRes.data.path;

      const { tradeId, type, index, name } = enlargedImageContext;
      const fullTradeRes = await tradesAPI.getOne(tradeId);
      const fullTrade = fullTradeRes.data;

      const payload = sanitizeTradePayload(fullTrade);
      const originalPath = enlargedOriginalImage;

      // 1. 核心邏輯：更新指定的欄位
      if (type === 'general') {
        if (payload.images && payload.images[index]) {
          payload.images[index].image_path = serverPath;
        }
      } else if (type === 'signal') {
        const signals = parseJSONSafe(payload.entry_signals, []);
        const sIdx = signals.findIndex(s => s.name === name);
        if (sIdx >= 0) {
          const sigImages =
            signals[sIdx].images || (signals[sIdx].image ? [{ image: signals[sIdx].image }] : []);
          if (sigImages[index]) {
            sigImages[index].image = serverPath;
          }
          signals[sIdx].images = sigImages;
          payload.entry_signals = JSON.stringify(signals);
        }
      } else if (type === 'pattern') {
        const patterns = parseJSONSafe(payload.entry_pattern, []);
        const pIdx = patterns.findIndex(p => p.name === name);
        if (pIdx >= 0) {
          const patImages =
            patterns[pIdx].images ||
            (patterns[pIdx].image ? [{ image: patterns[pIdx].image }] : []);
          if (patImages[index]) {
            patImages[index].image = serverPath;
          }
          patterns[pIdx].images = patImages;
          payload.entry_pattern = JSON.stringify(patterns);
        }
      } else if (type === 'strategy') {
        payload.entry_strategy_image = serverPath;
      } else if (type === 'legend_htf') {
        payload.legend_htf_image = serverPath;
      } else if (type === 'legend_king') {
        payload.legend_king_image = serverPath;
      } else if (type === 'legend_images') {
        const lImages = parseJSONSafe(payload.legend_images, []);
        if (lImages[index]) {
          lImages[index].image = serverPath;
        }
        payload.legend_images = JSON.stringify(lImages);
      }

      // 2. 聰明邏輯：全域掃描。如果其他欄位也使用了同一個原始圖片路徑，一併更新為標註後的版本
      if (originalPath) {
        if (payload.images) {
          payload.images.forEach(img => {
            if (img.image_path === originalPath) img.image_path = serverPath;
          });
        }
        if (payload.entry_signals && typeof payload.entry_signals === 'string') {
          const sigs = parseJSONSafe(payload.entry_signals, []);
          sigs.forEach(sig => {
            if (sig.image === originalPath) sig.image = serverPath;
            if (sig.images) {
              sig.images.forEach(img => {
                if (img.image === originalPath) img.image = serverPath;
              });
            }
          });
          payload.entry_signals = JSON.stringify(sigs);
        }
        if (payload.entry_pattern && typeof payload.entry_pattern === 'string') {
          const pats = parseJSONSafe(payload.entry_pattern, []);
          pats.forEach(pat => {
            if (pat.image === originalPath) pat.image = serverPath;
            if (pat.images) {
              pat.images.forEach(img => {
                if (img.image === originalPath) img.image = serverPath;
              });
            }
          });
          payload.entry_pattern = JSON.stringify(pats);
        }
        if (payload.entry_strategy_image === originalPath)
          payload.entry_strategy_image = serverPath;
        if (payload.legend_htf_image === originalPath) payload.legend_htf_image = serverPath;
        if (payload.legend_king_image === originalPath) payload.legend_king_image = serverPath;
        if (payload.legend_images && typeof payload.legend_images === 'string') {
          const lgs = parseJSONSafe(payload.legend_images, []);
          lgs.forEach(lg => {
            if (lg.image === originalPath) lg.image = serverPath;
          });
          payload.legend_images = JSON.stringify(lgs);
        }
        if (payload.expert_images && typeof payload.expert_images === 'string') {
          const exps = parseJSONSafe(payload.expert_images, []);
          exps.forEach(exp => {
            if (exp.image === originalPath) exp.image = serverPath;
          });
          payload.expert_images = JSON.stringify(exps);
        }
        if (payload.elite_images && typeof payload.elite_images === 'string') {
          const elts = parseJSONSafe(payload.elite_images, []);
          elts.forEach(elt => {
            if (elt.image === originalPath) elt.image = serverPath;
          });
          payload.elite_images = JSON.stringify(elts);
        }
      }

      await tradesAPI.update(tradeId, payload);

      selectedImage = imagesAPI.getUrl(serverPath);
      showAnnotator = false;
      loadData(false);
    } catch (e) {
      console.error('Failed to save annotated image', e);
      alert('儲存標註圖片失敗');
    }
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
      legend_images: fullTrade.legend_images || '',
      expert_images: fullTrade.expert_images || '',
      elite_images: fullTrade.elite_images || '',
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
                class="sync-badge {currentAccount.sync_status} {currentAccount.sync_status &&
                !['success', 'failed', 'idle'].includes(currentAccount.sync_status.toLowerCase())
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

        <div class="filter-stats-spacer"></div>

        <div class="filter-stats-badge" class:has-data={globalSummary.total_count > 0}>
          <div class="stats-icon">✅</div>
          <div class="stats-content">
            <span class="stats-label">
              {statsLabel}
            </span>
            <span class="stats-value">{globalSummary.total_count} 筆</span>
            <span class="stats-sep">/</span>
            <span class="stats-label">勝率</span>
            <span class="stats-value win-rate"
              >{((globalSummary.win_count * 100) / (globalSummary.total_count || 1)).toFixed(
                1
              )}%</span
            >

            <div class="stats-color-groups">
              <span class="stats-color-dot green"></span>
              <span class="stats-color-count">{globalSummary.green_count}</span>
              <span class="stats-color-dot yellow"></span>
              <span class="stats-color-count">{globalSummary.yellow_count}</span>
              <span class="stats-color-dot red"></span>
              <span class="stats-color-count">{globalSummary.red_count}</span>
            </div>

            <div class="stats-strategy-groups">
              <span class="strat-tag expert">達 {globalSummary.expert_count}</span>
              <span class="strat-tag elite">菁 {globalSummary.elite_count}</span>
              <span class="strat-tag legend">傳 {globalSummary.legend_count}</span>
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

        <div class="divider"></div>

        <button
          class="filter-type-btn color-filter {activeColorFilter === 'green' ? 'active' : ''}"
          on:click={() => selectColorFilter('green')}
          title="篩選綠燈 (標準單)"
        >
          <span class="stats-color-dot green" style="width: 12px; height: 12px;"></span>
        </button>
        <button
          class="filter-type-btn color-filter {activeColorFilter === 'yellow' ? 'active' : ''}"
          on:click={() => selectColorFilter('yellow')}
          title="篩選黃燈 (討論空間)"
        >
          <span class="stats-color-dot yellow" style="width: 12px; height: 12px;"></span>
        </button>
        <button
          class="filter-type-btn color-filter {activeColorFilter === 'red' ? 'active' : ''}"
          on:click={() => selectColorFilter('red')}
          title="篩選紅燈 (非標準單)"
        >
          <span class="stats-color-dot red" style="width: 12px; height: 12px;"></span>
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
          on:click={() => (activeSideFilter = activeSideFilter === 'long' ? 'all' : 'long')}
        >
          📈 做多
        </button>
        <button
          class="filter-type-btn {activeSideFilter === 'short' ? 'active' : ''}"
          on:click={() => (activeSideFilter = activeSideFilter === 'short' ? 'all' : 'short')}
        >
          📉 做空
        </button>

        <div class="page-size-selector">
          <span class="selector-label">每頁顯示交易數量:</span>
          <select
            class="size-select"
            bind:value={pagination.page_size}
            on:change={() => {
              pagination.page = 1;
              loadData();
            }}
          >
            {#each PAGE_SIZE_OPTIONS as opt}
              <option value={opt}>{opt}</option>
            {/each}
          </select>
        </div>
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
      <p class="loading-text">{loadingMessage}</p>
      {#if loadingMessage.includes('網路連線')}
        <small style="color: var(--text-muted); font-size: 0.8rem; margin-top: 5px;"
          >建議切換至更穩定的 Wi-Fi</small
        >
      {/if}
    </div>
  {:else if loadError}
    <div class="error-state">
      <div class="error-icon">⚠️</div>
      <h3>{loadError.message}</h3>
      <p class="error-detail">{loadError.detail}</p>
      {#if loadError.canRetry}
        <button class="btn btn-primary" on:click={() => loadData(false)}> 🔄 重新載入 </button>
      {/if}
      <button
        class="btn btn-secondary"
        on:click={() => {
          loadError = null;
          groupedData = [];
        }}
      >
        關閉
      </button>
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
  {:else if filteredGroupedData.length === 0}
    <div class="empty-state">
      <div class="empty-icon">🏜️</div>
      {#if isAllMode && !activeColorFilter}
        <p>這裡空空如也，開始記錄您的第一筆 {$selectedSymbol} 規劃或交易吧！</p>
      {:else}
        <p>找不到符合篩選條件的資料。</p>
        <button
          class="btn btn-secondary btn-sm"
          style="margin-top: 10px;"
          on:click={() => {
            activeFilterType = 'all';
            activeSubFilter = null;
            activeColorFilter = null;
          }}>清除所有篩選</button
        >
      {/if}
    </div>
  {:else}
    <div class="timeline">
      {#each paginatedGroupedData as group}
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
            {#if dailyStats.total > 0 || dailyStats.hasFloating || dailyStats.green + dailyStats.yellow + dailyStats.red > 0}
              <div class="daily-stats-badge">
                <div class="stats-content">
                  <span class="stats-label">每日統計：</span>
                  <span class="stats-value">{dailyStats.total} 筆</span>
                  <span class="stats-sep">/</span>
                  <span class="stats-label">勝率</span>
                  <span class="stats-value win-rate">{dailyStats.winRate.toFixed(1)}%</span>

                  <div class="stats-color-groups">
                    <span class="stats-color-dot green"></span>
                    <span class="stats-color-count">{dailyStats.green}</span>
                    <span class="stats-color-dot yellow"></span>
                    <span class="stats-color-count">{dailyStats.yellow}</span>
                    <span class="stats-color-dot red"></span>
                    <span class="stats-color-count">{dailyStats.red}</span>
                  </div>

                  <div class="stats-strategy-groups small">
                    <span class="strat-tag expert">達 {dailyStats.expert}</span>
                    <span class="strat-tag elite">菁 {dailyStats.elite}</span>
                    <span class="strat-tag legend">傳 {dailyStats.legend}</span>
                  </div>

                  {#if Math.abs(dailyStats.realizedPnl) > 0.001}
                    <span class="stats-sep">|</span>
                    <span class="stats-label">已實現</span>
                    <span
                      class="stats-value pnl"
                      class:profit={dailyStats.realizedPnl > 0}
                      class:loss={dailyStats.realizedPnl < 0}
                    >
                      {dailyStats.realizedPnl >= 0 ? '+' : ''}{dailyStats.realizedPnl.toFixed(2)}
                    </span>
                  {/if}
                </div>
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
                          : ''} {timeGroup.trades.some(
                          t => t.trade_type === 'actual' && !t.exit_time
                        )
                          ? 'is-ongoing'
                          : ''}"
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
                            {#if timeGroup.trades[0].entry_strategy}
                              <span class="strategy-tag {timeGroup.trades[0].entry_strategy}"
                                >{getStrategyLabel(timeGroup.trades[0].entry_strategy)}</span
                              >
                            {/if}
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
                                  isOpen={timeGroup.trades.some(
                                    t => t.trade_type === 'actual' && !t.exit_time
                                  )}
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
                                      isOpen={trade.trade_type === 'actual' && !trade.exit_time}
                                    />
                                  </div>
                                {/if}
                              </div>
                              {#if trade.ticket}<span class="partial-ticket">#{trade.ticket}</span
                                >{/if}
                            </div>
                          {/each}
                        </div>
                        {#if timeGroup.trades.some(t => t.images && t.images.length > 0)}
                          {@const allImages = timeGroup.trades.reduce(
                            (acc, t) => [...acc, ...(t.images || [])],
                            []
                          )}
                          <div class="mini-gallery">
                            {#each allImages.slice(0, 3) as img, idx}
                              <div
                                class="mini-img"
                                on:click|stopPropagation={() =>
                                  openImageModal(
                                    img.image_path,
                                    `${img.image_type === 'entry' ? '進場' : img.image_type === 'exit' ? '平倉' : '圖片'}截圖`,
                                    {
                                      tradeId:
                                        timeGroup.trades.find(t => (t.images || []).includes(img))
                                          ?.id || timeGroup.trades[0].id,
                                      type: 'general',
                                      index: (
                                        timeGroup.trades.find(t => (t.images || []).includes(img))
                                          ?.images || []
                                      ).indexOf(img),
                                    }
                                  )}
                              >
                                <img src={imagesAPI.getUrl(img.image_path)} alt="trade" />
                              </div>
                            {/each}
                            {#if allImages.length > 3}
                              <div class="more-imgs">+{allImages.length - 3}</div>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    {:else}
                      <!-- 一般單 (單筆進出) -->
                      {@const trade = timeGroup.trades[0]}
                      <div
                        class="trade-item-card {trade.color_tag
                          ? `tag-${trade.color_tag}`
                          : ''} {selectionMode && selectedTrades.has(trade.id)
                          ? 'selected'
                          : ''} {trade.trade_type === 'actual' && !trade.exit_time
                          ? 'is-ongoing'
                          : ''}"
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
                              {#if trade.trade_type === 'observation'}
                                <span class="journal-tag">📓 記事</span>
                              {/if}
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
                                  isOpen={trade.trade_type === 'actual' && !trade.exit_time}
                                />
                              </div>
                            {/if}
                            {#if trade.trade_type === 'actual'}
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
                            {/if}
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
                          {#if trade.trade_type === 'actual'}
                            {@const bulletToDisplay =
                              calculateBulletSize(trade) || trade.bullet_size}
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
                                  {bulletToDisplay && Number(bulletToDisplay) > 0
                                    ? Number(bulletToDisplay).toFixed(1)
                                    : 'NA'}
                                </strong>
                                {#if bulletToDisplay && bulletToDisplay > 0 && (trade.rr_ratio || trade.rr_ratio === 0)}
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
                          {/if}
                          <div class="trade-time">
                            {formatDate(trade.entry_time).split(' ')[1]} - {trade.exit_time
                              ? formatDate(trade.exit_time).split(' ')[1]
                              : trade.trade_type === 'actual'
                                ? '進行中'
                                : ''}
                          </div>
                        </div>

                        {#if trade.images && trade.images.length > 0}
                          <div class="mini-gallery">
                            {#each trade.images.slice(0, 3) as img, idx}
                              <div
                                class="mini-img"
                                on:click|stopPropagation={() =>
                                  openImageModal(
                                    img.image_path,
                                    `${img.image_type === 'entry' ? '進場' : img.image_type === 'exit' ? '平倉' : '圖片'}截圖`,
                                    { tradeId: trade.id, type: 'general', index: idx }
                                  )}
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
              {:else if !activeSubFilter && !activeColorFilter}
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

    {#if totalPages > 1}
      <div class="pagination-container">
        <div class="pagination-info">
          第 {pagination.page} / {totalPages} 頁 (共 {filteredStats.total} 筆交易)
        </div>
        <div class="pagination-controls">
          <button
            class="pagination-btn"
            disabled={pagination.page === 1}
            on:click={() => changePage(pagination.page - 1)}
          >
            上一步
          </button>

          <div class="page-numbers">
            {#each Array(Math.min(5, totalPages)) as _, i}
              {@const pageNum =
                totalPages <= 5
                  ? i + 1
                  : pagination.page <= 3
                    ? i + 1
                    : pagination.page >= totalPages - 2
                      ? totalPages - 4 + i
                      : pagination.page - 2 + i}
              <button
                class="page-num-btn"
                class:active={pagination.page === pageNum}
                on:click={() => changePage(pageNum)}
              >
                {pageNum}
              </button>
            {/each}
          </div>

          <button
            class="pagination-btn"
            disabled={pagination.page === totalPages}
            on:click={() => changePage(pagination.page + 1)}
          >
            下一步
          </button>
        </div>
      </div>
    {/if}
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
  <div class="image-modal active" on:click={closeImageModal} transition:fade={{ duration: 200 }}>
    <div class="image-modal-content" on:click|stopPropagation>
      <div class="image-modal-header">
        <h3 class="image-modal-title">{modalTitle}</h3>
        <div class="image-modal-actions">
          {#if enlargedImageContext}
            <button
              class="annotator-toggle-btn"
              class:active={showAnnotator}
              on:click={toggleAnnotator}
              title="標註工具"
            >
              {showAnnotator ? '👁️ 查看' : '✏️ 標註'}
            </button>
          {/if}
          <button class="image-modal-close" on:click={closeImageModal}>&times;</button>
        </div>
      </div>
      <div class="image-modal-body" class:annotator-mode={showAnnotator}>
        {#if showAnnotator}
          <ImageAnnotator
            imageSrc={selectedImage}
            originalImageSrc={imagesAPI.getUrl(enlargedOriginalImage)}
            onSave={handleAnnotatedImage}
          />
        {:else}
          <img src={selectedImage} alt="全螢幕圖片" class="image-modal-img" />
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .timeline-container {
    padding-bottom: 5rem;
  }

  /* Pagination Styling */
  .pagination-container {
    margin-top: 3rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 2rem 0;
    border-top: 1px dashed var(--border-color);
  }

  .pagination-info {
    font-size: 0.85rem;
    color: var(--text-muted);
    font-weight: 500;
  }

  .pagination-controls {
    display: flex;
    align-items: center;
    gap: 1.5rem;
  }

  .pagination-btn {
    padding: 0.5rem 1.25rem;
    border-radius: 10px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-main);
    font-weight: 600;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s;
    box-shadow: var(--shadow-sm);
  }

  .pagination-btn:hover:not(:disabled) {
    background: var(--primary);
    color: white;
    border-color: var(--primary);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
  }

  .pagination-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-numbers {
    display: flex;
    gap: 0.5rem;
  }

  .page-num-btn {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    border: 1px solid transparent;
    background: transparent;
    color: var(--text-main);
    font-weight: 700;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .page-num-btn:hover {
    background: var(--nav-group-bg);
  }

  .page-num-btn.active {
    background: var(--primary);
    color: white;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.3);
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

  .daily-stats-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0.8rem;
    background: rgba(34, 197, 94, 0.06);
    border: 1px solid rgba(34, 197, 94, 0.12);
    border-radius: 12px;
    margin-left: 0.75rem;
    transition: all 0.3s ease;
  }

  :global(body.dark-mode) .daily-stats-badge {
    background: rgba(34, 197, 94, 0.04);
    border-color: rgba(34, 197, 94, 0.08);
  }

  .daily-stats-badge:hover {
    background: rgba(34, 197, 94, 0.1);
    transform: translateY(-1px);
  }

  .daily-stats-badge .stats-value.pnl.profit {
    color: #10b981;
  }

  .daily-stats-badge .stats-value.pnl.loss {
    color: #ef4444;
  }

  /* 強制將 day-wrapper 增加 padding 以騰出空間給 absolute 標籤 */
  .day-wrapper {
    position: relative;
    margin-bottom: 3.5rem;
    padding-top: 1.5rem;
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
    margin-top: 0.5rem; /* 配合 padding-top 調整，不再需要這麼大的 margin */
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

  .journal-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    line-height: 1;
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
    white-space: nowrap;
  }
  :global(body.dark-mode) .journal-tag {
    background: #374151;
    color: #f3f4f6;
    border-color: #4b5563;
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
    flex-wrap: wrap;
    justify-content: flex-end;
    flex: 1;
    min-width: 0;
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
  .empty-state,
  .error-state {
    text-align: center;
    padding: 5rem;
    color: #64748b;
  }

  .error-state {
    color: #dc2626;
  }

  .error-state h3 {
    color: #dc2626;
    margin: 1rem 0 0.5rem;
    font-size: 1.5rem;
  }

  .error-detail {
    color: #64748b;
    margin-bottom: 2rem;
    font-size: 0.9rem;
  }

  .error-icon {
    font-size: 4rem;
    margin-bottom: 1rem;
    animation: shake 0.5s;
  }

  @keyframes shake {
    0%,
    100% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(-10px);
    }
    75% {
      transform: translateX(10px);
    }
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

  @media (max-width: 950px) {
    .timeline-container {
      padding: 1rem;
    }
    .home-hero {
      padding: 1.5rem;
      margin-bottom: 2rem;
    }
    .hero-title .greeting {
      font-size: 1rem;
    }
  }

  @media (max-width: 1024px) {
    .day-card-container {
      grid-template-columns: 1fr;
    }
    .plan-column {
      border-right: none;
      border-bottom: 1px dashed #e2e8f0;
      padding-right: 0;
      padding-bottom: 1.5rem;
      width: 100% !important;
    }
    .top-actions-bar {
      display: none; /* Already in bottom nav */
    }
  }

  @media (max-width: 500px) {
    .filter-type-btn {
      padding: 0.5rem 0.8rem;
      font-size: 0.85rem;
    }
    .day-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.5rem;
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
    border-left: 1px solid rgba(0, 0, 0, 0.08);
  }

  :global(body.dark-mode) .stats-color-groups {
    border-left-color: rgba(255, 255, 255, 0.1);
  }

  .stats-strategy-groups {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-left: 0.75rem;
    padding-left: 0.75rem;
    border-left: 1px solid rgba(0, 0, 0, 0.08);
  }

  :global(body.dark-mode) .stats-strategy-groups {
    border-left-color: rgba(255, 255, 255, 0.1);
  }

  .stats-strategy-groups.small {
    gap: 0.25rem;
    margin-left: 0.5rem;
    padding-left: 0.5rem;
    transform: scale(0.9);
  }

  .strat-tag {
    font-size: 0.7rem;
    padding: 1px 6px;
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

  .page-size-selector {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding-left: 1rem;
    margin-left: auto;
    border-left: 1px solid rgba(0, 0, 0, 0.1);
  }

  :global(body.dark-mode) .page-size-selector {
    border-left-color: rgba(255, 255, 255, 0.1);
  }

  .selector-label {
    font-size: 0.8rem;
    color: #64748b;
    font-weight: 600;
  }

  .size-select {
    padding: 0.4rem 2rem 0.4rem 0.75rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    background-color: white;
    font-size: 0.85rem;
    font-weight: 600;
    color: #1e293b;
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke='%2364748b'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19 9l-7 7-7-7'%3E%3C/path%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.5rem center;
    background-size: 1rem;
    transition: all 0.2s;
  }

  .size-select:hover {
    border-color: #cbd5e1;
    background-color: #f8fafc;
  }

  :global(body.dark-mode) .size-select {
    background-color: #1e293b;
    border-color: #334155;
    color: #f1f5f9;
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
    from {
      opacity: 0;
      transform: translateY(-5px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* Image Modal Styles */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 2000;
    backdrop-filter: blur(8px);
    padding: 20px;
  }

  .image-modal-content {
    background: var(--card-bg);
    border-radius: 16px;
    max-width: 95vw;
    max-height: 95vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    border: 1px solid var(--border-color);
  }

  .image-modal-header {
    padding: 1rem 1.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border-color);
    background: var(--card-bg);
  }

  .image-modal-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .annotator-toggle-btn {
    padding: 0.5rem 1rem;
    background: var(--nav-group-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-main);
    cursor: pointer;
    transition: all 0.2s;
  }

  .annotator-toggle-btn:hover {
    background: var(--bg-main);
  }

  .annotator-toggle-btn.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  .image-modal-title {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--text-main);
  }

  .image-modal-close {
    background: var(--nav-group-bg);
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: var(--text-muted);
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    transition: all 0.2s;
  }

  .image-modal-close:hover {
    background: #ef4444;
    color: white;
    transform: rotate(90deg);
  }

  .image-modal-body {
    flex: 1;
    overflow: auto;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #0f172a;
    padding: 1rem;
  }

  .image-modal-body.annotator-mode {
    align-items: flex-start;
  }

  .image-modal-img {
    max-width: 100%;
    max-height: calc(95vh - 4rem);
    object-fit: contain;
    border-radius: 4px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  }
</style>
