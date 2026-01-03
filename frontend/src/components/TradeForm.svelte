<script>
  import { navigate } from 'svelte-routing';
  import { onMount } from 'svelte';
  import { tradesAPI, dailyPlansAPI } from '../lib/api';
  import { SYMBOLS, MARKET_SESSIONS } from '../lib/constants';
  import { selectedAccountId, accounts, selectedSymbol } from '../lib/stores';
  import RichTextEditor from './RichTextEditor.svelte';
  import ImageAnnotator from './ImageAnnotator.svelte';
  import WatchlistSelectionModal from './WatchlistSelectionModal.svelte';
  import ShareModal from './ShareModal.svelte';
  import TradePlanStatus from './trade-form/TradePlanStatus.svelte';
  import EntryStrategySelector from './trade-form/EntryStrategySelector.svelte';
  import ExpertStrategy from './trade-form/ExpertStrategy.svelte';
  import EliteStrategy from './trade-form/EliteStrategy.svelte';
  import LegendStrategy from './trade-form/LegendStrategy.svelte';

  export let id = null;
  const symbols = SYMBOLS;

  let formData = {
    account_id: $selectedAccountId,
    trade_type: 'observation', // actual=有進單, observation=純觀察
    symbol: $selectedSymbol || 'XAUUSD',
    side: 'long',
    entry_price: '',
    exit_price: '',
    lot_size: '',
    pnl: '',
    pnl_points: '',
    notes: '',
    entry_reason: '',
    exit_reason: '',
    entry_strategy: '', // expert=達人, elite=菁英, legend=傳奇
    entry_signals: [], // 達人訊號（多選），格式：[{name: "訊號名稱", image: "base64圖片", originalImage: "base64原始圖片"}]
    entry_checklist: {}, // 菁英/傳傳奇檢查清單
    entry_pattern: [], // 進場樣態（僅菁英使用），格式：[{name: "名稱", image: "base64", originalImage: "base64"}]
    entry_timeframe: '', // 進場時區
    trend_type: '', // 順勢/逆勢
    market_session: '', // asian=亞盤, european=歐盤, us=美盤
    initial_sl: '', // 初始停損價
    bullet_size: '', // 子彈大小 (風險金額)
    rr_ratio: '', // 風報比
    timezone_offset: new Date().getTimezoneOffset() / -60, // 預設系統時區
    entry_time: (() => {
      const now = new Date();
      const year = now.getFullYear();
      const month = String(now.getMonth() + 1).padStart(2, '0');
      const day = String(now.getDate()).padStart(2, '0');
      const hours = String(now.getHours()).padStart(2, '0');
      const minutes = String(now.getMinutes()).padStart(2, '0');
      return `${year}-${month}-${day}T${hours}:${minutes}`;
    })(),
    exit_time: '',
    tags: [],
    entry_strategy_image: '', // 用於儲存菁英/傳奇的樣態圖或觀察圖
    entry_strategy_image_original: '',
    legend_htf: '', // 傳奇：大時區破測破的時區
    legend_htf_image: '', // 傳奇：大時區破測破的圖片
    legend_htf_image_original: '',
    legend_king_htf: '', // 傳奇：王者回調的時區
    legend_king_image: '', // 傳奇：王者回調的圖片
    legend_king_image_original: '',
    legend_de_htf: '', // 傳奇：整理段的時區
    exit_sl: '', // 平倉時的停損價
    color_tag: '', // 顏色標記 (red, yellow, green)
  };

  // 觀察單併入相關
  let showWatchlistModal = false;
  let watchlistTrades = [];
  let showShareModal = false;

  // 開啟觀察單選擇視窗
  async function openWatchlistModal() {
    if (!formData.symbol) {
      alert('請先選擇交易品種');
      return;
    }

    try {
      // 取得觀察單資料
      const response = await tradesAPI.getAll({
        account_id: formData.account_id,
        symbol: formData.symbol,
        page: 1,
        page_size: 50,
      });

      if (response.data && response.data.data) {
        // 過濾出 "observation" 且 symbol 相同的單子
        watchlistTrades = response.data.data.filter(
          t => t.trade_type === 'observation' && t.symbol === formData.symbol
        );

        // 排序：最新的在最上面
        watchlistTrades.sort((a, b) => new Date(b.entry_time) - new Date(a.entry_time));

        if (watchlistTrades.length > 0) {
          showWatchlistModal = true;
        } else {
          alert(`找不到 ${formData.symbol} 的觀察單。`);
        }
      } else {
        alert('無法取得交易紀錄。');
      }
    } catch (error) {
      console.error('Fetch trades error:', error);
      alert('讀取觀察單失敗');
    }
  }

  // 處理確認併入觀察單資料
  function handleMergeWatchlist(sourceTrade) {
    if (!sourceTrade) return;

    if (
      confirm(
        `確定要併入觀察單 (${new Date(sourceTrade.entry_time).toLocaleString()}) 的分析資料嗎？\n這將會覆蓋目前的進/出場分析與標籤。`
      )
    ) {
      // 1. 併入進場分析 (Entry Analysis)
      formData.entry_reason = sourceTrade.entry_reason || '';
      formData.entry_strategy = sourceTrade.entry_strategy || '';
      formData.entry_strategy_image = sourceTrade.entry_strategy_image || '';
      formData.entry_strategy_image_original = sourceTrade.entry_strategy_image_original || '';

      if (sourceTrade.entry_signals) {
        try {
          formData.entry_signals =
            typeof sourceTrade.entry_signals === 'string'
              ? JSON.parse(sourceTrade.entry_signals)
              : sourceTrade.entry_signals;
        } catch (e) {
          formData.entry_signals = [];
        }
      }

      if (sourceTrade.entry_checklist) {
        try {
          formData.entry_checklist =
            typeof sourceTrade.entry_checklist === 'string'
              ? JSON.parse(sourceTrade.entry_checklist)
              : sourceTrade.entry_checklist;
        } catch (e) {
          formData.entry_checklist = {};
        }
      }

      if (sourceTrade.entry_pattern) {
        try {
          formData.entry_pattern =
            typeof sourceTrade.entry_pattern === 'string'
              ? JSON.parse(sourceTrade.entry_pattern)
              : sourceTrade.entry_pattern;
        } catch (e) {
          formData.entry_pattern = [];
        }
      }

      formData.exit_reason = sourceTrade.exit_reason || '';
      if (sourceTrade.tags && Array.isArray(sourceTrade.tags)) {
        formData.tags = sourceTrade.tags
          .map(t => (t && typeof t === 'object' ? t.name : t))
          .filter(t => t);
      }

      if (sourceTrade.initial_sl) {
        formData.initial_sl = sourceTrade.initial_sl;
      }

      formData = formData;
      alert('觀察單資料併入完成！');
    }
  }

  // 開啟實單選擇視窗
  async function openActualTradesModal() {
    if (!formData.symbol) {
      alert('請先選擇交易品種');
      return;
    }

    try {
      const response = await tradesAPI.getAll({
        account_id: formData.account_id,
        symbol: formData.symbol,
        page: 1,
        page_size: 50,
      });

      if (response.data && response.data.data) {
        // 過濾出 "actual" 且 symbol 相同的單子
        watchlistTrades = response.data.data.filter(
          t => t.trade_type === 'actual' && t.symbol === formData.symbol
        );

        watchlistTrades.sort((a, b) => new Date(b.entry_time) - new Date(a.entry_time));

        if (watchlistTrades.length > 0) {
          showWatchlistModal = true;
        } else {
          alert(`找不到 ${formData.symbol} 的實單紀錄。`);
        }
      }
    } catch (error) {
      console.error('Fetch actual trades error:', error);
      alert('讀取實單失敗');
    }
  }

  // 處理從實單併入資料
  function handleMergeActualTrade(sourceTrade) {
    if (!sourceTrade) return;

    if (
      confirm(
        `確定要將實單 (${new Date(sourceTrade.entry_time).toLocaleString()}) 的交易資料併入這筆觀察記錄嗎？\n這將會同步進場價格、手數與盈虧，並將本紀錄轉為「實單」。`
      )
    ) {
      // 同步實單的核心數據
      formData.entry_price = sourceTrade.entry_price;
      formData.exit_price = sourceTrade.exit_price;
      formData.lot_size = sourceTrade.lot_size;
      formData.pnl = sourceTrade.pnl;
      formData.pnl_points = sourceTrade.pnl_points;
      formData.initial_sl = sourceTrade.initial_sl;
      formData.exit_sl = sourceTrade.exit_sl;
      formData.ticket = sourceTrade.ticket;

      // 自動轉為實單類型
      formData.trade_type = 'actual';

      formData = formData; // Trigger update
      alert('實單資料併入成功，已自動轉為「有進單」模式。');
    }
  }

  // 根據選擇的帳號自動同步時區設定
  $: currentAccount = $accounts.find(a => a.id === $selectedAccountId);
  $: if (currentAccount) {
    formData.timezone_offset = currentAccount.timezone_offset;
  }

  // 確保當前選中的帳號 ID 與表單同步
  $: if ($selectedAccountId) {
    formData.account_id = $selectedAccountId;
  }

  // 確保當前選中的品種與表單同步（僅限新增模式）
  $: if (!id && $selectedSymbol) {
    formData.symbol = $selectedSymbol;
  }

  // 響應式：根據交易類型判斷是否顯示交易相關欄位
  $: isActualTrade = formData.trade_type === 'actual';

  // 訊號圖片緩存（保留所有訊號的圖片，即使取消勾選）
  let signalImagesCache = {}; // { signalName: { image: '...', originalImage: '...' } }
  let patternImagesCache = {}; // { patternName: { image: '...', originalImage: '...' } }

  // 時區選項 (UTC-12 到 UTC+14)

  // 時區選項 (UTC-12 到 UTC+14)
  const timezoneOptions = [];
  for (let i = -12; i <= 14; i++) {
    timezoneOptions.push({
      value: i,
      label: i >= 0 ? `UTC+${i}` : `UTC${i}`,
    });
  }

  // 市場時段判別函數
  function determineMarketSession(entryTime, timezoneOffset) {
    if (!entryTime) return '';

    const date = new Date(entryTime);
    const month = date.getMonth() + 1; // 1-12

    // 判斷是否為夏令時間（3月-11月）
    const isDST = month >= 3 && month <= 11;

    // 轉換為 UTC 時間
    const utcHour = date.getUTCHours();
    const utcMinute = date.getUTCMinutes();

    // 轉換為 GMT+8（台北時間）用於判斷
    const gmt8Hour = (utcHour + 8 + 24) % 24;
    const timeInMinutes = gmt8Hour * 60 + utcMinute;

    // 時間範圍定義（以 GMT+8 為基準，單位：分鐘）
    // 亞盤（東京）：08:00 - 15:00（全年不變）
    const asianStart = 8 * 60; // 08:00
    const asianEnd = 15 * 60; // 15:00

    // 歐盤（倫敦）
    let europeanStart, europeanEnd;
    if (isDST) {
      // 夏令時間：15:00 - 23:00
      europeanStart = 15 * 60; // 15:00
      europeanEnd = 23 * 60; // 23:00
    } else {
      // 冬令時間：16:00 - 00:00
      europeanStart = 16 * 60; // 16:00
      europeanEnd = 24 * 60; // 00:00 (midnight)
    }

    // 美盤（紐約）
    let usStart, usEnd;
    if (isDST) {
      // 夏令時間：20:00 - 04:00（跨日）
      usStart = 20 * 60; // 20:00
      usEnd = 4 * 60; // 04:00
    } else {
      // 冬令時間：21:00 - 05:00（跨日）
      usStart = 21 * 60; // 21:00
      usEnd = 5 * 60; // 05:00
    }

    // 判斷市場時段
    // 亞盤：08:00 - 15:00
    if (timeInMinutes >= asianStart && timeInMinutes < asianEnd) {
      return 'asian';
    }

    // 美盤優先（處理跨日情況）
    if (timeInMinutes >= usStart || timeInMinutes < usEnd) {
      return 'us';
    }

    // 歐盤
    if (isDST) {
      // 夏令時間：15:00 - 23:00
      if (timeInMinutes >= europeanStart && timeInMinutes < europeanEnd) {
        return 'european';
      }
    } else {
      // 冬令時間：16:00 - 00:00（處理跨日）
      if (timeInMinutes >= europeanStart || timeInMinutes < 0) {
        return 'european';
      }
    }

    // 其他時間（間隙）預設為亞盤
    return 'asian';
  }

  // 取得交易日（處理美盤跨日：凌晨的時間算前一天的交易日）
  function getTradingDate(entryTime) {
    if (!entryTime) return '';
    const date = new Date(entryTime);

    // 轉換為 GMT+8 用於判斷
    const utcHour = date.getUTCHours();
    const gmt8Hour = (utcHour + 8 + 24) % 24;

    // 如果是凌晨 00:00 - 06:00 且不是亞盤時間，通常屬於前一天的美盤
    // 這裡我們簡化處理：如果是 00:00 - 06:00，我們回傳前一天的日期字串
    if (gmt8Hour >= 0 && gmt8Hour < 6) {
      const prevDay = new Date(date);
      prevDay.setDate(date.getDate() - 1);
      return prevDay.toLocaleDateString('en-CA'); // YYYY-MM-DD
    }

    return date.toLocaleDateString('en-CA'); // YYYY-MM-DD
  }

  // 盈虧點數與風險指標自動計算
  $: {
    const { trade_type, entry_price, exit_price, lot_size, initial_sl, pnl, symbol, side } =
      formData;
    if (trade_type === 'actual' && entry_price) {
      const entry = parseFloat(entry_price);
      const exit = parseFloat(exit_price);
      const sl = parseFloat(initial_sl);
      const lots = parseFloat(lot_size);

      let multiplier = 1; // 預設 (金子 XAUUSD: $1 = 1點, 指數: 1.0 = 1點)
      if (symbol.includes('JPY')) multiplier = 100;
      else if (
        symbol.includes('EUR') ||
        symbol.includes('GBP') ||
        symbol.includes('AUD') ||
        (symbol.includes('USD') && !symbol.includes('XAU'))
      ) {
        multiplier = 10000;
      }

      // 1. 盈虧點數計算
      if (!isNaN(entry) && !isNaN(exit)) {
        const diff = exit - entry;
        const result = Math.round(diff * (side === 'long' ? 1 : -1) * multiplier * 100) / 100;
        if (formData.pnl_points !== result) {
          formData.pnl_points = result;
        }
      }

      // 2. 子彈大小計算 (Bullet Size / Risk Amount)
      if (!isNaN(entry) && !isNaN(sl)) {
        const riskPoints = Math.abs(entry - sl);
        const result = Math.round(riskPoints * multiplier * 100) / 100;
        if (formData.bullet_size !== result) {
          formData.bullet_size = result;
        }
      }

      // 3. 風報比計算 (RR Ratio)
      const currentPoints = parseFloat(formData.pnl_points);
      const currentBullet = parseFloat(formData.bullet_size);
      if (!isNaN(currentPoints) && !isNaN(currentBullet) && currentBullet !== 0) {
        const result = Math.round((currentPoints / currentBullet) * 100) / 100;
        if (formData.rr_ratio !== result) {
          formData.rr_ratio = result;
        }
      }
    }
  }

  // 監聽進場時間和時區變化，自動更新市場時段
  $: {
    if (formData.entry_time && formData.timezone_offset !== null) {
      formData.market_session = determineMarketSession(
        formData.entry_time,
        formData.timezone_offset
      );
    }
  }

  // 市場時段顯示名稱
  const marketSessionNames = MARKET_SESSIONS.reduce((acc, current) => {
    acc[current.value] = current.label;
    return acc;
  }, {});

  // 取得市場時段時間範圍文字
  function getMarketSessionTime(session) {
    if (!session || !formData.entry_time) return '';

    const date = new Date(formData.entry_time);
    const month = date.getMonth() + 1;
    const isDST = month >= 3 && month <= 11;

    switch (session) {
      case 'asian':
        return '08:00 - 15:00';
      case 'european':
        return isDST ? '15:00 - 23:00' : '16:00 - 00:00';
      case 'us':
        return isDST ? '20:00 - 04:00' : '21:00 - 05:00';
      default:
        return '';
    }
  }

  // 取得夏/冬令時間標示
  function getSeasonLabel() {
    if (!formData.entry_time) return '';
    const date = new Date(formData.entry_time);
    const month = date.getMonth() + 1;
    const isDST = month >= 3 && month <= 11;
    return isDST ? '夏令時間' : '冬令時間';
  }

  // 格式化日期為本地 ISO 格式 (YYYY-MM-DDTHH:mm)
  function formatToLocalISO(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';
    const offset = date.getTimezoneOffset() * 60000;
    return new Date(date.getTime() - offset).toISOString().slice(0, 16);
  }

  function parseJSONSafe(str, defaultValue) {
    if (!str) return defaultValue;
    try {
      return JSON.parse(str);
    } catch (e) {
      return defaultValue;
    }
  }

  let tagInput = '';
  let saving = false;

  // 富文本編輯器引用
  let entryReasonEditor;
  let exitReasonEditor;
  let notesEditor;

  let isGroup = false;
  let groupTrades = [];

  // 計算組合單總計
  $: totalLot = groupTrades.reduce((sum, t) => sum + (t.lot_size || 0), 0);
  $: totalPnl = groupTrades.reduce((sum, t) => sum + (t.pnl || 0), 0);

  // 圖片放大查看
  let enlargedImage = null;
  let enlargedImageTitle = '';
  let enlargedImageContext = null; // 記錄圖片來源上下文：{type: 'signal'|'trend', key: string}
  let showAnnotator = false;

  let allPlans = [];

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const symbolParam = params.get('symbol');
    if (
      symbolParam &&
      symbolParam !== 'undefined' &&
      symbolParam !== 'null' &&
      !symbolParam.includes('{$')
    ) {
      formData.symbol = symbolParam;
    } else if (!id) {
      formData.symbol = 'XAUUSD';
    }

    if (id) {
      loadTrade();
    }
    loadPlans();
  });

  async function loadPlans() {
    try {
      const response = await dailyPlansAPI.getAll({
        account_id: $selectedAccountId,
        page_size: 100,
      });
      allPlans = response.data.data || [];
    } catch (error) {
      console.error('載入規劃失敗:', error);
    }
  }

  // 響應式：取得相匹配的每日規劃
  $: matchedPlan = (() => {
    if (!formData.entry_time || !formData.market_session || allPlans.length === 0) return null;

    try {
      const tradeDate = new Date(formData.entry_time).toISOString().slice(0, 10);
      return allPlans.find(plan => {
        const planDate = new Date(plan.plan_date).toISOString().slice(0, 10);
        if (planDate !== tradeDate) return false;

        // 同時匹配品種 (如果有 symbol 欄位的話，舊資料預設 XAUUSD)
        const planSymbol = plan.symbol || SYMBOLS[0];
        if (planSymbol !== formData.symbol) return false;

        if (plan.market_session === 'all') {
          // 新格式：檢查該時段在 JSON 中是否有任何趨勢或備註
          try {
            const trendData = JSON.parse(plan.trend_analysis || '{}');
            const sessionData = trendData[formData.market_session];
            // 如果該時段有備註或任何時區有方向，視為匹配
            return (
              sessionData &&
              (sessionData.notes ||
                (sessionData.trends && Object.values(sessionData.trends).some(t => t.direction)))
            );
          } catch (e) {
            return false;
          }
        } else {
          // 舊格式：直接匹配時段
          return plan.market_session === formData.market_session;
        }
      });
    } catch (e) {
      return null;
    }
  })();

  async function loadTrade() {
    try {
      const response = await tradesAPI.getOne(id);
      formData = {
        ...response.data,
        initial_sl: response.data.initial_sl || '',
        bullet_size: response.data.bullet_size || '',
        rr_ratio: response.data.rr_ratio || '',
        entry_reason: response.data.entry_reason || '',
        exit_reason: response.data.exit_reason || '',
        notes: response.data.notes || '',
        entry_strategy: response.data.entry_strategy || '',
        entry_signals: (() => {
          const parsed = parseJSONSafe(response.data.entry_signals, []);
          // 如果是舊格式（字串陣列），轉換成新格式（物件陣列）
          if (Array.isArray(parsed) && parsed.length > 0 && typeof parsed[0] === 'string') {
            return parsed.map(name => ({ name, image: '' }));
          }
          return parsed;
        })(),
        entry_checklist: parseJSONSafe(response.data.entry_checklist, {}),
        entry_pattern: response.data.entry_pattern || '',
        entry_timeframe: response.data.entry_timeframe || '',
        trend_type: response.data.trend_type || '',
        market_session: response.data.market_session || '',
        timezone_offset:
          response.data.timezone_offset !== null
            ? response.data.timezone_offset
            : new Date().getTimezoneOffset() / -60,
        entry_time: formatToLocalISO(response.data.entry_time),
        exit_time: response.data.exit_time ? formatToLocalISO(response.data.exit_time) : '',
        entry_strategy_image: response.data.entry_strategy_image || '',
        entry_strategy_image_original: response.data.entry_strategy_image_original || '',
        entry_pattern: parseJSONSafe(response.data.entry_pattern, []),
        legend_king_htf: response.data.legend_king_htf || '',
        legend_king_image: response.data.legend_king_image || '',
        legend_king_image_original: response.data.legend_king_image_original || '',
        legend_htf: response.data.legend_htf || '',
        legend_htf_image: response.data.legend_htf_image || '',
        legend_htf_image_original: response.data.legend_htf_image_original || '',
        legend_de_htf: response.data.legend_de_htf || '',
        tags: response.data.tags?.map(t => t.name) || [],
        color_tag: response.data.color_tag || '',
      };

      // 初始化緩存：將已載入的訊號圖片也加入緩存
      if (formData.entry_signals && Array.isArray(formData.entry_signals)) {
        formData.entry_signals.forEach(signal => {
          if (signal.name && (signal.image || signal.originalImage)) {
            signalImagesCache[signal.name] = {
              image: signal.image || '',
              originalImage: signal.originalImage || '',
            };
          }
        });
      }

      // 初始化樣態緩存
      if (formData.entry_pattern && Array.isArray(formData.entry_pattern)) {
        formData.entry_pattern.forEach(pattern => {
          if (pattern.name && (pattern.image || pattern.originalImage)) {
            patternImagesCache[pattern.name] = {
              image: pattern.image || '',
              originalImage: pattern.originalImage || '',
            };
          }
        });
      }
      // 檢查是否為組合單（相同進場時間、帳號、品種）
      const allTradesRes = await tradesAPI.getAll({
        account_id: formData.account_id,
        symbol: formData.symbol,
        page_size: 100,
      });
      const allTradesData =
        (Array.isArray(allTradesRes.data) ? allTradesRes.data : allTradesRes.data?.data) || [];
      groupTrades = allTradesData
        .filter(t => t.entry_time === response.data.entry_time)
        .sort((a, b) => new Date(a.exit_time || 0) - new Date(b.exit_time || 0));
      isGroup = groupTrades.length > 1;
    } catch (error) {
      console.error('載入交易失敗:', error);
      alert('載入交易資料失敗');
    }
  }

  function addTag() {
    if (tagInput.trim() && !formData.tags.includes(tagInput.trim())) {
      formData.tags = [...formData.tags, tagInput.trim()];
      tagInput = '';
    }
  }

  function removeTag(tag) {
    formData.tags = formData.tags.filter(t => t !== tag);
  }

  // 監聽方向變化，清空已選訊號（避免做多訊號和做空訊號混淆）
  let previousSide = formData.side;
  $: {
    if (formData.side !== previousSide && formData.entry_strategy === 'expert') {
      formData.entry_signals = [];
      previousSide = formData.side;
    }
  }

  // 放大查看圖片
  let enlargedOriginalImage = null; // 保存當前放大圖片的原始版本

  function enlargeImage(imageSrc, title, context = null) {
    if (!imageSrc) return;
    enlargedImage = imageSrc;
    enlargedImageTitle = title;
    enlargedImageContext = context;
    showAnnotator = false; // 預設不顯示標註工具

    // 獲取原始圖片
    if (context) {
      const { type, key } = context;
      if (type === 'signal') {
        const signal = formData.entry_signals.find(s =>
          typeof s === 'string' ? s === key : s.name === key
        );
        enlargedOriginalImage = signal?.originalImage || imageSrc;
      } else if (type === 'trend') {
        enlargedOriginalImage = formData.trend_analysis[key]?.originalImage || imageSrc;
      } else if (type === 'pattern') {
        const pattern = formData.entry_pattern.find(p => p.name === key);
        enlargedOriginalImage = pattern?.originalImage || imageSrc;
      } else if (type === 'strategy') {
        enlargedOriginalImage = formData.entry_strategy_image_original || imageSrc;
      } else if (type === 'legend_htf') {
        enlargedOriginalImage = formData.legend_htf_image_original || imageSrc;
      } else if (type === 'legend_king') {
        enlargedOriginalImage = formData.legend_king_image_original || imageSrc;
      }
    } else {
      enlargedOriginalImage = imageSrc;
    }
  }

  // 切換標註工具顯示
  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  // 處理標註後的圖片保存
  function handleAnnotatedImage(annotatedImageSrc) {
    if (!enlargedImageContext) {
      // 如果沒有上下文，只更新顯示的圖片
      enlargedImage = annotatedImageSrc;
      return;
    }

    const { type, key } = enlargedImageContext;

    if (type === 'signal') {
      // 更新訊號圖片（只更新 image，保持 originalImage 不變）
      const index = formData.entry_signals.findIndex(s =>
        typeof s === 'string' ? s === key : s.name === key
      );

      if (index >= 0) {
        const currentSignal = formData.entry_signals[index];
        const signal =
          typeof currentSignal === 'string'
            ? { name: key, image: annotatedImageSrc, originalImage: annotatedImageSrc }
            : { ...currentSignal, image: annotatedImageSrc };
        formData.entry_signals[index] = signal;
        formData = formData;
      }
    } else if (type === 'trend') {
      // 更新趨勢圖片（只更新 image，保持 originalImage 不變）
      if (formData.trend_analysis[key]) {
        formData.trend_analysis[key] = {
          ...formData.trend_analysis[key],
          image: annotatedImageSrc,
        };
        formData = formData;
      }
    } else if (type === 'strategy') {
      // 更新策略圖片
      formData.entry_strategy_image = annotatedImageSrc;
      formData = formData;
    } else if (type === 'legend_htf') {
      // 更新傳奇大時區圖片
      formData.legend_htf_image = annotatedImageSrc;
      formData = formData;
    } else if (type === 'legend_king') {
      // 更新傳奇王者圖片
      formData.legend_king_image = annotatedImageSrc;
      formData = formData;
    } else if (type === 'pattern') {
      const index = formData.entry_pattern.findIndex(p => p.name === key);
      if (index >= 0) {
        formData.entry_pattern[index].image = annotatedImageSrc;
        // 同步到緩存
        patternImagesCache[key] = {
          ...patternImagesCache[key],
          image: annotatedImageSrc,
        };
        formData = formData;
      }
    }

    // 更新顯示的圖片
    enlargedImage = annotatedImageSrc;
    showAnnotator = false; // 保存後隱藏標註工具
  }

  // 關閉放大圖片
  function closeEnlargedImage() {
    enlargedImage = null;
    enlargedImageTitle = '';
    enlargedImageContext = null;
    showAnnotator = false;
  }

  async function handleSubmit() {
    try {
      saving = true;

      // 確保 entry_signals 格式正確（轉換成物件陣列）
      const normalizedSignals = formData.entry_signals.map(s =>
        typeof s === 'string' ? { name: s, image: '' } : s
      );

      // 從富文本編輯器取得內容
      const submitData = {
        ...formData,
        account_id: $selectedAccountId,
        entry_reason: entryReasonEditor ? entryReasonEditor.getContent() : formData.entry_reason,
        exit_reason: exitReasonEditor ? exitReasonEditor.getContent() : formData.exit_reason,
        notes: notesEditor ? notesEditor.getContent() : formData.notes,
        entry_signals: JSON.stringify(normalizedSignals),
        entry_checklist: JSON.stringify(formData.entry_checklist),
        entry_pattern: JSON.stringify(formData.entry_pattern),
        entry_strategy_image: formData.entry_strategy_image,
        entry_strategy_image_original: formData.entry_strategy_image_original,
        entry_timeframe: formData.entry_timeframe,
        trend_type: formData.trend_type,
        entry_time: new Date(formData.entry_time).toISOString(),
        exit_time: formData.exit_time ? new Date(formData.exit_time).toISOString() : null,
      };

      // 處理數值欄位轉換
      const parseNumber = val => {
        if (val === null || val === undefined || val === '') return null;
        const num = parseFloat(val);
        return isNaN(num) ? null : num;
      };

      submitData.initial_sl = parseNumber(formData.initial_sl);
      submitData.exit_sl = parseNumber(formData.exit_sl);
      submitData.color_tag = formData.color_tag;
      submitData.bullet_size = parseNumber(formData.bullet_size);
      submitData.rr_ratio = parseNumber(formData.rr_ratio);

      // 如果是實際交易，添加交易相關欄位
      if (isActualTrade) {
        submitData.entry_price = parseNumber(formData.entry_price);
        submitData.exit_price = parseNumber(formData.exit_price);
        submitData.lot_size = parseNumber(formData.lot_size);
        submitData.pnl = parseNumber(formData.pnl);
        submitData.pnl_points = parseNumber(formData.pnl_points);
      } else {
        // 純觀察記錄，這些執行相關欄位設為 null
        submitData.entry_price = parseNumber(formData.entry_price); // 觀察單也可能有預計進場價
        submitData.exit_price = null;
        submitData.lot_size = null;
        submitData.pnl = null;
        submitData.pnl_points = null;
        submitData.exit_time = null;
      }

      if (id) {
        if (isGroup) {
          // 如果是組合單，同步更新所有子交易的分析欄位
          for (const sibling of groupTrades) {
            // 只保留執行相關欄位（exit, lot, pnl, ticket），覆蓋分析欄位
            const siblingData = {
              ...submitData,
              id: sibling.id,
              exit_time: sibling.exit_time,
              exit_price: sibling.exit_price,
              lot_size: sibling.lot_size,
              pnl: sibling.pnl,
              pnl_points: sibling.pnl_points,
              ticket: sibling.ticket,
              exit_sl: sibling.exit_sl,
              exit_reason: sibling.exit_reason, // 部分平倉可能有不同原因，但通常也是共用的，這裡暫跟隨主單
            };
            await tradesAPI.update(sibling.id, siblingData);
          }
        } else {
          await tradesAPI.update(id, submitData);
        }
        alert('交易紀錄更新成功！');
      } else {
        await tradesAPI.create(submitData);
        alert('交易紀錄建立成功！');
      }

      navigate('/');
    } catch (error) {
      console.error('儲存失敗:', error);
      alert('儲存失敗：' + (error.response?.data?.error || error.message));
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && enlargedImage) {
      closeEnlargedImage();
    }
  }

  // 解析 SL 歷史資料 (相容新舊格式)
  function parseSLHistory(json) {
    if (!json) return [];
    try {
      const data = JSON.parse(json);
      return data.map(item => {
        if (typeof item === 'number') return { price: item, time: null };
        return item;
      });
    } catch (e) {
      return [];
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="card {formData.color_tag ? 'tag-' + formData.color_tag : ''}">
  <div class="card-header-actions">
    <h2>{id ? '編輯' : '新增'}交易紀錄</h2>
    <div class="header-form-actions">
      <button type="button" class="btn btn-sm" on:click={() => navigate('/')}>返回</button>
      <button
        type="button"
        class="btn btn-sm btn-primary"
        on:click={handleSubmit}
        disabled={saving}
      >
        {#if saving}
          儲存中...
        {:else}
          {id ? '更新' : '儲存'}交易
        {/if}
      </button>

      {#if id}
        <button type="button" class="btn btn-sm btn-share" on:click={() => (showShareModal = true)}>
          📤 分享
        </button>
      {/if}

      <div class="merge-action-container header-merge">
        {#if formData.trade_type === 'actual'}
          <button
            type="button"
            class="btn-merge btn-sm"
            on:click={openWatchlistModal}
            title="從過去的觀察單匯入分析資料"
          >
            <span class="icon">📋</span> 併入觀察單
          </button>
        {:else}
          <button
            type="button"
            class="btn-merge btn-sm"
            on:click={openActualTradesModal}
            title="併入現有的實單交易"
          >
            <span class="icon">💰</span> 併入實單
          </button>
        {/if}
      </div>
    </div>
  </div>

  <form on:submit|preventDefault={handleSubmit}>
    <!-- 交易類型選擇 -->
    <div class="form-group trade-type-section">
      <label class="trade-type-label">紀錄類型</label>
      <div class="trade-type-options">
        <label class="radio-option" class:active={formData.trade_type === 'observation'}>
          <input type="radio" bind:group={formData.trade_type} value="observation" />
          <span class="radio-label">
            <span class="radio-icon">👁️</span>
            <span class="radio-text">
              <strong>觀察中 (尚未進單)</strong>
              <small>純觀察記錄</small>
            </span>
          </span>
        </label>
        <label class="radio-option" class:active={formData.trade_type === 'actual'}>
          <input type="radio" bind:group={formData.trade_type} value="actual" />
          <span class="radio-label">
            <span class="radio-icon">💰</span>
            <span class="radio-text">
              <strong>有進單</strong>
              <small>實際交易記錄</small>
            </span>
          </span>
        </label>
      </div>
    </div>

    <!-- 顏色標記 -->
    <div class="form-group color-tag-section">
      <label class="section-label">顏色標記</label>
      <div class="color-tags-options">
        <button
          type="button"
          class="color-select-btn green {formData.color_tag === 'green' ? 'active' : ''}"
          on:click={() => (formData.color_tag = formData.color_tag === 'green' ? '' : 'green')}
        ></button>
        <button
          type="button"
          class="color-select-btn yellow {formData.color_tag === 'yellow' ? 'active' : ''}"
          on:click={() => (formData.color_tag = formData.color_tag === 'yellow' ? '' : 'yellow')}
        ></button>
        <button
          type="button"
          class="color-select-btn red {formData.color_tag === 'red' ? 'active' : ''}"
          on:click={() => (formData.color_tag = formData.color_tag === 'red' ? '' : 'red')}
        ></button>
      </div>
    </div>

    <!-- 基本資訊 -->
    <div class="form-row">
      <div class="form-group">
        <label for="symbol">交易品種</label>
        <select id="symbol" class="form-control" bind:value={formData.symbol} required>
          {#each symbols as symbol}
            <option value={symbol}>{symbol}</option>
          {/each}
        </select>
      </div>

      <div class="form-group">
        <label for="side">做多或做空</label>
        <select id="side" class="form-control" bind:value={formData.side} required>
          <option value="long">做多 (Long)</option>
          <option value="short">做空 (Short)</option>
        </select>
      </div>

      {#if isActualTrade}
        <div class="form-group">
          <label for="lot_size">手數</label>
          {#if isGroup}
            <div class="readonly-value-badge">
              總共 {totalLot.toFixed(2)} 手 ({groupTrades.length} 次平倉)
            </div>
          {:else}
            <input
              type="number"
              step="0.01"
              id="lot_size"
              class="form-control"
              bind:value={formData.lot_size}
              required
            />
          {/if}
        </div>
      {/if}
    </div>

    {#if isActualTrade && !isGroup}
      <div class="form-row four-cols">
        <div class="form-group">
          <label for="entry_price">進場價格</label>
          <input
            type="number"
            step="0.00001"
            id="entry_price"
            class="form-control"
            bind:value={formData.entry_price}
            required
          />
        </div>

        <div class="form-group">
          <label for="initial_sl">初始停損 (SL)</label>
          <input
            type="number"
            step="0.00001"
            id="initial_sl"
            class="form-control"
            bind:value={formData.initial_sl}
            placeholder="用於計算子彈大小"
          />
          {#if formData.sl_history}
            <div class="sl-history-chips">
              {#each parseSLHistory(formData.sl_history) as entry}
                <button
                  type="button"
                  class="sl-chip {parseFloat(formData.initial_sl) === entry.price ? 'active' : ''}"
                  on:click={() => (formData.initial_sl = entry.price)}
                  title={entry.time ? new Date(entry.time).toLocaleString() : ''}
                >
                  <span class="sl-price">{entry.price}</span>
                  {#if entry.time}
                    <span class="sl-time"
                      >{new Date(entry.time).toLocaleTimeString('zh-TW', {
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                      })}</span
                    >
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <div class="form-group">
          <label for="exit_price">平倉價格</label>
          <input
            type="number"
            step="0.00001"
            id="exit_price"
            class="form-control"
            bind:value={formData.exit_price}
          />
        </div>

        <div class="form-group">
          <label for="exit_sl">平倉 SL</label>
          <input
            type="number"
            step="0.00001"
            id="exit_sl"
            class="form-control"
            bind:value={formData.exit_sl}
            placeholder="平倉當下的 SL"
          />
        </div>
      </div>

      <div class="form-row four-cols">
        <div class="form-group">
          <label for="pnl">盈虧金額</label>
          <input
            type="number"
            step="0.01"
            id="pnl"
            class="form-control"
            bind:value={formData.pnl}
          />
        </div>

        <div class="form-group">
          <label for="pnl_points">盈虧點數</label>
          <input
            type="number"
            step="0.1"
            id="pnl_points"
            class="form-control readonly-calc"
            bind:value={formData.pnl_points}
            readonly
            placeholder="自動計算"
          />
        </div>

        <div class="form-group">
          <label for="bullet_size">子彈大小 (Bullet)</label>
          <input
            type="number"
            id="bullet_size"
            class="form-control readonly-calc"
            bind:value={formData.bullet_size}
            readonly
            placeholder="自動計算"
          />
        </div>

        <div class="form-group">
          <label for="rr_ratio">風報比 (R:R)</label>
          <input
            type="number"
            id="rr_ratio"
            class="form-control readonly-calc"
            bind:value={formData.rr_ratio}
            readonly
            placeholder="自動計算"
          />
        </div>
      </div>
      {#if !formData.entry_price || !formData.initial_sl}
        <div style="margin-top: -0.5rem; margin-bottom: 1rem;">
          <small class="form-hint"
            >💡 請填寫「進場價格」與「初始停損」以自動計算子彈大小與風報比</small
          >
        </div>
      {/if}
    {:else if isActualTrade && isGroup}
      <!-- 組合單專用 Execution 配置 -->
      <div class="form-row">
        <div class="form-group">
          <label for="entry_price">進場價格</label>
          <input
            type="number"
            step="0.00001"
            id="entry_price"
            class="form-control"
            bind:value={formData.entry_price}
            required
          />
        </div>
        <div class="form-group">
          <label for="initial_sl">初始停損 (SL)</label>
          <input
            type="number"
            step="0.00001"
            id="initial_sl"
            class="form-control"
            bind:value={formData.initial_sl}
          />
          {#if formData.sl_history}
            <div class="sl-history-chips">
              {#each parseSLHistory(formData.sl_history) as entry}
                <button
                  type="button"
                  class="sl-chip {parseFloat(formData.initial_sl) === entry.price ? 'active' : ''}"
                  on:click={() => (formData.initial_sl = entry.price)}
                  title={entry.time ? new Date(entry.time).toLocaleString() : ''}
                >
                  <span class="sl-price">{entry.price}</span>
                  {#if entry.time}
                    <span class="sl-time"
                      >{new Date(entry.time).toLocaleTimeString('zh-TW', {
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                      })}</span
                    >
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <div class="form-group">
          <label>總計盈虧</label>
          <div class="readonly-value-badge pnl {totalPnl >= 0 ? 'profit' : 'loss'}">
            {totalPnl >= 0 ? '+' : ''}{totalPnl.toFixed(2)} USD
          </div>
        </div>
      </div>

      <div class="execution-timeline-section">
        <label class="section-subtitle">📋 平倉時間軸 (分批出場記錄)</label>
        <div class="timeline-container-mini">
          {#each groupTrades as t, i}
            <div class="timeline-item-mini">
              <div class="item-time">
                平倉 {i + 1}:
                <strong
                  >{new Date(t.exit_time).toLocaleString('zh-TW', {
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                  })}</strong
                >
              </div>
              <div class="item-details">
                <span class="badge-mini">價格: {t.exit_price}</span>
                <span class="badge-mini">手數: {t.lot_size}</span>
                <span class="badge-mini pnl {t.pnl >= 0 ? 'profit' : 'loss'}"
                  >盈虧: {t.pnl >= 0 ? '+' : ''}{t.pnl?.toFixed(2)}</span
                >
                {#if t.ticket}<span class="badge-mini ticket">#{t.ticket}</span>{/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <div class="form-row time-row">
      <div class="form-group">
        <label for="entry_time">
          開倉時間
          <span class="utc-label-info"
            >(UTC{formData.timezone_offset >= 0 ? '+' : ''}{formData.timezone_offset})</span
          >
        </label>
        <input
          type="datetime-local"
          id="entry_time"
          class="form-control"
          bind:value={formData.entry_time}
          step="1"
          required
        />
      </div>

      {#if formData.market_session}
        <div class="form-group">
          <label>市場時段與規劃</label>
          <div class="session-status-card {formData.market_session}">
            <div class="session-badge-mini">
              {marketSessionNames[formData.market_session]}
            </div>
            <div class="session-info-line">
              <span class="session-time-text">{getMarketSessionTime(formData.market_session)}</span>
              <span class="session-dot">·</span>
              <span class="session-season-text">{getSeasonLabel()}</span>
            </div>

            <div class="plan-status-mini">
              {#if matchedPlan}
                <span class="status-yes" on:click={() => navigate(`/plans/edit/${matchedPlan.id}`)}>
                  <i class="icon">✅</i> 已有規劃
                </span>
              {:else}
                <span
                  class="status-no"
                  on:click={() => {
                    const date = new Date(formData.entry_time).toISOString().slice(0, 10);
                    navigate(
                      `/plans/new?date=${date}&session=${formData.market_session}&symbol=${formData.symbol}`
                    );
                  }}
                >
                  <i class="icon">❓</i> 缺規劃
                </span>
              {/if}
            </div>
          </div>
        </div>
      {/if}
    </div>

    {#if isActualTrade && !isGroup}
      <div class="form-row">
        <div class="form-group">
          <label for="exit_time">
            平倉時間
            <span class="utc-label-info"
              >(UTC{formData.timezone_offset >= 0 ? '+' : ''}{formData.timezone_offset})</span
            >
          </label>
          <input
            type="datetime-local"
            id="exit_time"
            class="form-control"
            bind:value={formData.exit_time}
          />
        </div>
      </div>
    {/if}

    <div class="form-group highlight-label">
      <label>📍 進場分析</label>
    </div>

    <!-- 進場種類選擇 -->
    <div class="form-group entry-strategy-section">
      <!-- 盤面規劃狀態 (從上方移至此處) -->
      <TradePlanStatus {matchedPlan} {formData} />

      <!-- 進場種類和進場時區 -->
      <!-- 進場種類和進場時區 -->
      <EntryStrategySelector bind:formData />

      <!-- 達人訊號（卡片形式，可貼圖） -->
      {#if formData.entry_strategy === 'expert'}
        <ExpertStrategy
          bind:formData
          bind:signalImagesCache
          on:enlarge={e => enlargeImage(e.detail.image, e.detail.title, e.detail.context)}
        />
      {/if}

      {#if formData.entry_strategy === 'elite'}
        <EliteStrategy
          bind:formData
          bind:patternImagesCache
          on:enlarge={e => enlargeImage(e.detail.image, e.detail.title, e.detail.context)}
        />
      {/if}

      {#if formData.entry_strategy === 'legend'}
        <LegendStrategy
          bind:formData
          bind:signalImagesCache
          on:enlarge={e => enlargeImage(e.detail.image, e.detail.title, e.detail.context)}
        />
      {/if}
    </div>

    <div class="form-group">
      <label for="exit_reason">
        🎯 平倉理由
        <span class="hint-inline">（支援圖片貼上：Ctrl+V 或點擊工具列圖片按鈕）</span>
      </label>
      <RichTextEditor
        bind:this={exitReasonEditor}
        bind:value={formData.exit_reason}
        placeholder="為什麼平倉？止盈/止損/訊號反轉？可以貼上圖片說明..."
        height="180px"
      />
    </div>

    <div class="form-group">
      <label for="notes">
        📝 交易復盤
        <span class="hint-inline">（支援圖片貼上：Ctrl+V 或點擊工具列圖片按鈕）</span>
      </label>
      <RichTextEditor
        bind:this={notesEditor}
        bind:value={formData.notes}
        placeholder="記錄當下的心態、策略、失誤等...可以貼上圖片說明..."
        height="200px"
      />
    </div>

    <div class="form-group">
      <label for="trade-tags">標籤</label>
      <div class="tag-input-wrapper">
        <input
          id="trade-tags"
          type="text"
          class="form-control"
          bind:value={tagInput}
          placeholder="輸入標籤（如：突破、回踩、新聞單）"
          on:keypress={e => e.key === 'Enter' && (e.preventDefault(), addTag())}
        />
        <button type="button" class="btn btn-primary" on:click={addTag}>新增</button>
      </div>
      <div class="tags-container">
        {#each formData.tags as tag}
          <span class="tag">
            #{tag}
            <button type="button" class="tag-remove" on:click={() => removeTag(tag)}>×</button>
          </span>
        {/each}
      </div>
    </div>

    <div class="form-actions">
      <button type="button" class="btn" on:click={() => navigate('/')}>返回</button>
      <button type="submit" class="btn btn-primary" disabled={saving}>
        {#if saving}
          儲存中...
        {:else}
          {id ? '更新' : '建立'}交易
        {/if}
      </button>
    </div>
  </form>
</div>

<!-- 圖片放大查看模態視窗 -->
{#if enlargedImage}
  <div class="image-modal" on:click={closeEnlargedImage} role="presentation">
    <div class="image-modal-content" on:click={e => e.stopPropagation()} role="presentation">
      <div class="image-modal-header">
        <h3 class="image-modal-title">{enlargedImageTitle}</h3>
        <div class="image-modal-actions">
          <button
            class="annotator-toggle-btn"
            class:active={showAnnotator}
            on:click={e => {
              e.stopPropagation();
              toggleAnnotator();
            }}
            title="標註工具"
          >
            {showAnnotator ? '👁️ 查看' : '✏️ 標註'}
          </button>
          <button class="image-modal-close" on:click={closeEnlargedImage}>×</button>
        </div>
      </div>

      {#if showAnnotator}
        <ImageAnnotator
          imageSrc={enlargedImage}
          originalImageSrc={enlargedOriginalImage}
          onSave={handleAnnotatedImage}
        />
      {:else}
        <img src={enlargedImage} alt={enlargedImageTitle} class="image-modal-img" />
      {/if}
    </div>
  </div>
{/if}

<!-- 觀察單選擇模態框 -->
<WatchlistSelectionModal
  show={showWatchlistModal}
  trades={watchlistTrades}
  currentSymbol={formData.symbol}
  onConfirm={formData.trade_type === 'actual' ? handleMergeWatchlist : handleMergeActualTrade}
  onClose={() => (showWatchlistModal = false)}
/>

<ShareModal
  show={showShareModal}
  resourceType="trade"
  resourceId={id}
  onClose={() => (showShareModal = false)}
/>

<style>
  .card-header-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid #edf2f7;
  }

  h2 {
    margin-bottom: 0;
    color: #2d3748;
  }

  .header-form-actions {
    display: flex;
    gap: 0.75rem;
  }

  .btn-sm {
    padding: 0.5rem 1.25rem;
    font-size: 0.9rem;
  }

  .color-tag-section {
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
    padding: 1.5rem;
    margin-bottom: 2rem;
  }

  .color-tags-options {
    display: flex;
    gap: 1rem;
  }

  .color-select-btn {
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    border: 2px solid #e2e8f0;
    cursor: pointer;
    transition: all 0.2s;
    padding: 0;
  }

  .color-select-btn:hover {
    transform: scale(1.1);
  }

  .color-select-btn.active {
    border: 3px solid #4a5568;
    transform: scale(1.1);
    box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
  }

  .color-select-btn.green {
    background-color: #22c55e;
  }
  .color-select-btn.yellow {
    background-color: #eab308;
  }
  .color-select-btn.red {
    background-color: #ef4444;
  }

  /* 交易類型選擇 */
  .trade-type-section {
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
  }

  .trade-type-label {
    display: block;
    font-size: 1.1rem;
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 1rem;
  }

  .trade-type-options {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .radio-option {
    position: relative;
    cursor: pointer;
    border: 2px solid #cbd5e0;
    border-radius: 12px;
    padding: 1.25rem;
    background: white;
    transition: all 0.2s ease;
  }

  .radio-option:hover {
    border-color: #667eea;
    background: #f7fafc;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.1);
  }

  .radio-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .radio-option input[type='radio'] {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .radio-icon {
    font-size: 2rem;
    line-height: 1;
  }

  .radio-text {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .radio-text strong {
    font-size: 1rem;
    color: #2d3748;
  }

  .radio-text small {
    font-size: 0.85rem;
    color: #718096;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1.5fr 1fr; /* 品種, 方向, 手數 分配比例 */
    gap: 1.25rem;
    margin-bottom: 0.85rem;
  }

  /* 針對特定行數調整欄位 */
  /* 針對特定行數調整欄位 */
  .form-row.four-cols {
    grid-template-columns: repeat(4, 1fr);
  }

  .form-row.time-row {
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
  }

  /* 當寬度足夠時，限制最大寬度以避免過度展開 */
  :global(.card) {
    max-width: 960px;
    margin: 0 auto;
    padding: 2rem !important;
  }

  .readonly-calc {
    background-color: #f8fafc;
    color: #4a5568;
    cursor: default;
    font-weight: 600;
    border: 1px solid #e2e8f0;
  }

  .time-input-container {
    display: flex;
    align-items: center;
  }

  .utc-label-info {
    font-size: 0.8rem;
    color: #a0aec0;
    margin-left: 0.5rem;
    font-weight: 500;
  }

  .form-hint {
    display: block;
    margin-top: 0.4rem;
    color: #718096;
    font-size: 0.8rem;
    font-style: italic;
  }

  /* 進場分析區塊 */
  .highlight-label {
    margin-bottom: 1rem;
    border-left: 4px solid #667eea;
    padding-left: 0.75rem;
  }

  .highlight-label label {
    font-size: 1.1rem;
    font-weight: 700;
    color: #2d3748;
  }

  .entry-strategy-section {
    margin: 1.5rem 0;
    padding: 1.5rem;
    background: #f8fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
  }

  .trade-plan-status-section {
    margin-bottom: 2rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px dashed #cbd5e0;
  }

  .section-label-group {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .plan-status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    border-radius: 20px;
    font-size: 0.85rem;
    font-weight: 700;
    border: 1px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
  }

  .plan-status-badge.linked {
    background: #f0fdf4;
    color: #166534;
    border-color: #bcf0da;
  }

  .plan-status-badge.linked:hover {
    background: #dcfce7;
  }

  .plan-status-badge.missing {
    background: #fff5f5;
    color: #c53030;
    border-color: #feb2b2;
  }

  .plan-status-badge.missing:hover {
    background: #fff5f5;
  }

  .view-link,
  .add-link {
    font-size: 0.75rem;
    text-decoration: underline;
    opacity: 0.8;
  }

  .plan-details-summary {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .plan-general-notes {
    font-size: 0.9rem;
    color: #4a5568;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid #edf2f7;
    font-style: italic;
  }

  /* 時序進展視圖 */
  .progression-view {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .progression-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.9rem;
  }

  .tf-name {
    font-weight: 700;
    color: #475569;
    min-width: 40px;
  }

  .steps {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: nowrap;
  }

  .step {
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.85rem;
  }

  .step.long {
    background: #f0fdf4;
    color: #166534;
  }

  .step.short {
    background: #fef2f2;
    color: #991b1b;
  }

  .arrow {
    color: #94a3b8;
    font-weight: bold;
    font-size: 0.8rem;
  }

  .plan-session-notes {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding-top: 0.75rem;
    border-top: 1px solid #edf2f7;
  }

  .plan-note-item {
    display: flex;
    gap: 0.5rem;
    font-size: 0.85rem;
  }

  .session-tag {
    font-weight: 700;
    white-space: nowrap;
    font-size: 0.8rem;
  }

  .session-tag.asian {
    color: #2b6cb0;
  }
  .session-tag.european {
    color: #975a16;
  }
  .session-tag.us {
    color: #c53030;
  }

  .note-text {
    color: #4a5568;
  }

  .strategy-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.25rem;
    flex-wrap: wrap;
    gap: 1rem;
  }

  .strategy-label {
    font-size: 1rem;
    font-weight: 700;
    color: #4a5568;
    margin-bottom: 0;
  }

  .timeframe-trend-row {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    margin-bottom: 1rem;
    display: grid;
    grid-template-columns: auto 1fr;
    align-items: start;
    gap: 2rem;
  }

  .timeframe-trend-row .form-group {
    margin-bottom: 0;
  }

  .timeframe-trend-row label {
    font-size: 0.85rem;
    font-weight: 600;
    color: #718096;
    margin-bottom: 0.4rem;
    display: block;
  }

  .strategy-options {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .strategy-option {
    position: relative;
    cursor: pointer;
    padding: 0.75rem 1.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    transition: all 0.2s ease;
  }

  .strategy-option:hover {
    border-color: #667eea;
    background: #f7fafc;
    transform: translateY(-1px);
  }

  .strategy-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .strategy-options.mini {
    gap: 0.5rem;
  }

  .strategy-options.mini .strategy-option {
    padding: 0.5rem 0.85rem;
    border-width: 1.5px;
    border-radius: 6px;
  }

  .strategy-options.mini .strategy-name {
    font-size: 0.85rem;
  }

  /* 精緻合併按鈕樣式 */
  .merge-action-container {
    display: flex;
    align-items: center;
    padding-left: 1rem;
    margin-left: 1rem;
    border-left: 1.5px dashed #e2e8f0;
  }

  .merge-action-container.header-merge {
    border-left: none;
    padding-left: 0.5rem;
    margin-left: 0.5rem;
  }

  .btn-merge {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.5rem 1rem;
    background: linear-gradient(135deg, #ffffff 0%, #f5f3ff 100%);
    border: 1px solid #c4b5fd;
    border-radius: 10px;
    color: #6d28d9;
    font-weight: 700;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow:
      0 4px 6px -1px rgba(109, 40, 217, 0.05),
      0 2px 4px -1px rgba(109, 40, 217, 0.03);
    white-space: nowrap;
  }

  .btn-merge:hover {
    background: linear-gradient(135deg, #6d28d9 0%, #7c3aed 100%);
    color: white;
    border-color: #7c3aed;
    transform: translateY(-2px);
    box-shadow: 0 10px 15px -3px rgba(109, 40, 217, 0.2);
  }

  .btn-merge:active {
    transform: translateY(0);
  }

  .btn-merge .icon {
    font-size: 1.1rem;
    filter: drop-shadow(0 0 2px rgba(0, 0, 0, 0.1));
  }

  .strategy-option input[type='radio'] {
    position: absolute;
    opacity: 0;
  }

  .strategy-name {
    font-weight: 600;
    color: #2d3748;
  }

  .strategy-option.active .strategy-name {
    color: #667eea;
  }

  /* 訊號和檢查清單 */
  .signals-section,
  .checklist-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .signals-label,
  .checklist-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.75rem;
  }

  .signals-section.nested {
    margin-top: 1rem;
    padding: 1.25rem;
    background: #f8fafc;
    border: 2px dashed #6366f1;
    border-radius: 12px;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .htf-selector-row {
    margin-bottom: 1rem;
  }

  .signal-card.htf-image-card {
    max-width: 500px;
    min-height: 250px;
  }

  /* 訊號卡片網格 */
  .signals-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
  }

  /* 訊號卡片 */
  .signal-card {
    background: white;
    border: 2px solid #cbd5e0;
    border-radius: 12px;
    padding: 1rem;
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .signal-card.legend-image-card {
    max-width: 400px;
    min-height: 200px;
  }

  .signal-card:hover {
    border-color: #667eea;
    box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
    transform: translateY(-2px);
  }

  .signal-card:focus {
    border-color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .signal-card.selected {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .signal-checkbox-wrapper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    user-select: none;
  }

  .signal-checkbox {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #667eea;
  }

  .signal-name {
    font-weight: 600;
    color: #2d3748;
    font-size: 0.95rem;
  }

  .signal-card.selected .signal-name {
    color: #667eea;
  }

  /* 訊號圖片預覽 */
  .signal-image-preview {
    position: relative;
    margin-top: 0.5rem;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
  }

  .signal-image-preview img {
    width: 100%;
    height: auto;
    display: block;
    max-height: 200px;
    object-fit: contain;
    background: white;
  }

  .remove-signal-image {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 24px;
    height: 24px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.2rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .remove-signal-image:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  /* 訊號圖片佔位符 */
  .signal-image-placeholder {
    margin-top: 0.5rem;
    padding: 2rem 1rem;
    border: 2px dashed #cbd5e0;
    border-radius: 8px;
    text-align: center;
    background: #f7fafc;
    transition: all 0.2s ease;
  }

  .signal-card:hover .signal-image-placeholder {
    border-color: #667eea;
    background: #edf2f7;
  }

  .placeholder-text {
    font-size: 0.85rem;
    color: #718096;
    display: block;
  }

  .checklist-items {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .checkbox-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 6px;
    transition: background 0.2s ease;
  }

  .checkbox-item:hover {
    background: #f7fafc;
  }

  .checkbox-item input[type='checkbox'] {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #667eea;
  }

  .checkbox-label {
    font-size: 0.9rem;
    color: #2d3748;
    user-select: none;
  }

  /* 進場樣態 */
  .entry-pattern-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .entry-pattern-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.75rem;
  }

  .entry-pattern-options {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .pattern-option {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 1rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
  }

  .pattern-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .pattern-option.active {
    border-color: #667eea;
    background: #667eea;
  }

  .pattern-option input[type='radio'] {
    display: none;
  }

  .pattern-name {
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
  }

  .pattern-option.active .pattern-name {
    color: white;
  }

  .pattern-cards-grid {
    margin-top: 1.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
  }

  .pattern-image-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transition: all 0.2s ease;
  }

  .pattern-image-card:hover {
    border-color: #667eea;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  }

  .pattern-card-header {
    padding: 0.5rem 0.75rem;
    background: #edf2f7;
    border-bottom: 1px solid #e2e8f0;
  }

  .pattern-card-title {
    font-size: 0.85rem;
    font-weight: 700;
    color: #4a5568;
  }

  .pattern-card-body {
    padding: 0.75rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 120px;
    position: relative;
    justify-content: center;
    align-items: center;
  }

  .pattern-image-preview {
    width: 100%;
    cursor: zoom-in;
    border-radius: 6px;
    overflow: hidden;
    position: relative;
  }

  .pattern-image-preview img {
    width: 100%;
    height: 120px;
    object-fit: cover;
    display: block;
  }

  .remove-pattern-image {
    position: absolute;
    top: 4px;
    right: 4px;
    width: 20px;
    height: 20px;
    background: rgba(0, 0, 0, 0.5);
    color: white;
    border: none;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    cursor: pointer;
    font-size: 0.9rem;
    transition: background 0.2s;
    line-height: 1;
    padding-bottom: 2px;
  }

  .remove-pattern-image:hover {
    background: rgba(0, 0, 0, 0.8);
  }

  .pattern-image-placeholder {
    width: 100%;
    height: 120px;
    display: flex;
    justify-content: center;
    align-items: center;
    text-align: center;
    border: 2px dashed #cbd5e0;
    border-radius: 8px;
    padding: 0.5rem;
  }

  .placeholder-text {
    font-size: 0.75rem;
    color: #a0aec0;
    line-height: 1.4;
  }

  /* 進場時區按鈕組 */
  .timeframe-options {
    display: flex;
    gap: 2px;
    background: #1a1a1a;
    padding: 4px;
    border-radius: 8px;
    width: fit-content;
  }

  .timeframe-btn {
    padding: 6px 10px;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: #888;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: fit-content;
  }

  .timeframe-btn:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
  }

  .timeframe-btn.active {
    background: #333;
    color: #60a5fa; /* 藍色亮顯，符合交易軟體習慣 */
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }

  /* 市場時段狀態卡片 */
  .session-status-card {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    padding: 0 1rem;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    height: 42px; /* 精準匹配 input 高度 */
  }

  .session-badge-mini {
    padding: 2px 10px;
    border-radius: 6px;
    font-size: 0.85rem;
    font-weight: 700;
    color: white;
  }

  .session-status-card.asian .session-badge-mini {
    background: #6366f1;
  }
  .session-status-card.european .session-badge-mini {
    background: #f43f5e;
  }
  .session-status-card.us .session-badge-mini {
    background: #0ea5e9;
  }

  .session-status-card.asian {
    border-left: 4px solid #6366f1;
    background: rgba(99, 102, 241, 0.05);
  }
  .session-status-card.european {
    border-left: 4px solid #f43f5e;
    background: rgba(244, 63, 94, 0.05);
  }
  .session-status-card.us {
    border-left: 4px solid #0ea5e9;
    background: rgba(14, 165, 233, 0.05);
  }

  .session-info-line {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    color: #64748b;
  }

  .session-time-text {
    font-weight: 600;
    color: #334155;
  }
  .session-dot {
    opacity: 0.5;
  }
  .session-season-text {
    font-size: 0.75rem;
  }

  .plan-status-mini {
    margin-left: auto;
    display: flex;
    align-items: center;
  }

  .plan-status-mini span {
    font-size: 0.8rem;
    font-weight: 700;
    cursor: pointer;
    padding: 2px 8px;
    border-radius: 4px;
    transition: all 0.2s;
  }

  .status-yes {
    color: #16a34a;
    background: #f0fdf4;
    border: 1px solid #dcfce7;
  }
  .status-yes:hover {
    background: #dcfce7;
    transform: translateY(-1px);
  }
  .status-no {
    color: #dc2626;
    background: #fef2f2;
    border: 1px solid #fee2e2;
  }
  .status-no:hover {
    background: #fee2e2;
    transform: translateY(-1px);
  }

  .plan-status-mini .icon {
    font-style: normal;
    margin-right: 2px;
  }

  .plan-link-section {
    display: flex;
    align-items: center;
  }

  .plan-status {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.625rem 1.25rem;
    border-radius: 12px;
    border: 1px solid #e2e8f0;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 600;
  }

  .plan-status.linked {
    background: #f0fdf4;
    border-color: #bbf7d0;
    color: #166534;
  }

  .plan-status.linked:hover {
    background: #dcfce7;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(22, 101, 52, 0.1);
  }

  .plan-status.missing {
    background: #fffaf0;
    border-color: #fbd38d;
    color: #9c4221;
  }

  .plan-status.missing:hover {
    background: #fff0d6;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(156, 66, 33, 0.1);
  }

  .status-icon {
    font-size: 1.1rem;
    margin-top: 0.1rem;
  }

  .status-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    text-align: left;
  }

  .status-top {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .plan-mini-summary {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .mini-trend {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0px 4px;
    border-radius: 4px;
  }

  .mini-trend.bulls {
    color: #166534;
    background: #dcfce7;
  }

  .mini-trend.bears {
    color: #991b1b;
    background: #fee2e2;
  }

  .view-link,
  .add-link {
    font-size: 0.8rem;
    opacity: 0.8;
    margin-left: 0.25rem;
    padding-left: 0.75rem;
    border-left: 1px solid currentColor;
  }

  /* 當前趨勢選擇 */
  .trend-analysis-section {
    margin: 1.5rem 0;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
    border: 2px solid #e2e8f0;
  }

  .trend-label {
    display: block;
    font-size: 1rem;
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 1rem;
  }

  .trend-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .trend-item {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    outline: none;
  }

  .trend-item:hover {
    border-color: #667eea;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  }

  .trend-item:focus {
    border-color: #667eea;
    box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  }

  .timeframe-label {
    display: block;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
  }

  .trend-options {
    display: flex;
    gap: 0.5rem;
  }

  .trend-option {
    flex: 1;
    position: relative;
    cursor: pointer;
    padding: 0.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    transition: all 0.2s ease;
    text-align: center;
  }

  .trend-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .trend-option.active {
    border-color: #667eea;
    background: #edf2f7;
    box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
  }

  .trend-option input[type='radio'] {
    position: absolute;
    opacity: 0;
  }

  .trend-name {
    font-weight: 600;
    color: #2d3748;
    font-size: 0.9rem;
  }

  .trend-option.active .trend-name {
    color: #667eea;
  }

  /* 趨勢圖片預覽 */
  .trend-image-preview {
    position: relative;
    margin-top: 0.5rem;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    cursor: zoom-in;
  }

  .trend-image-preview img {
    width: 100%;
    height: auto;
    display: block;
    max-height: 200px;
    object-fit: contain;
    background: #f7fafc;
  }

  .remove-trend-image {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 24px;
    height: 24px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.2rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .remove-trend-image:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  /* 圖片放大查看模態視窗 */
  .image-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    padding: 2rem;
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .image-modal-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
    background: white;
    border-radius: 12px;
    padding: 0;
    display: flex;
    flex-direction: column;
    animation: slideIn 0.3s ease-out;
    overflow: hidden;
  }

  .image-modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem 2rem;
    border-bottom: 1px solid #e2e8f0;
    background: #f7fafc;
  }

  .image-modal-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .annotator-toggle-btn {
    padding: 0.5rem 1rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    color: #4a5568;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s ease;
  }

  .annotator-toggle-btn:hover {
    border-color: #667eea;
    background: #edf2f7;
  }

  .annotator-toggle-btn.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  @keyframes slideIn {
    from {
      transform: scale(0.9);
      opacity: 0;
    }
    to {
      transform: scale(1);
      opacity: 1;
    }
  }

  .image-modal-close {
    width: 36px;
    height: 36px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.5rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    padding: 0;
  }

  .image-modal-close:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  .image-modal-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #2d3748;
    margin: 0;
  }

  .image-modal-img {
    max-width: 100%;
    max-height: calc(90vh - 8rem);
    object-fit: contain;
    padding: 1rem;
  }

  .image-modal-content :global(.annotator-container) {
    padding: 1rem;
    max-height: calc(90vh - 6rem);
    overflow: auto;
  }

  .tag-input-wrapper {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .tag {
    background: #667eea;
    color: white;
    padding: 0.5rem 1rem;
    border-radius: 20px;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
  }

  .tag-remove {
    background: none;
    border: none;
    color: white;
    font-size: 1.5rem;
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
    padding-top: 2rem;
    border-top: 2px solid #e2e8f0;
  }

  textarea.form-control {
    resize: vertical;
    font-family: inherit;
  }

  .hint-inline {
    color: #a0aec0;
    font-size: 0.85rem;
    font-weight: normal;
    margin-left: 0.5rem;
  }

  label {
    display: flex;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  /* 組合單 Execution 樣式 */
  .readonly-value-badge {
    padding: 0.625rem 1rem;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    font-weight: 700;
    color: #475569;
    font-size: 0.95rem;
  }

  .readonly-value-badge.pnl.profit {
    background: #f0fdf4;
    color: #166534;
    border-color: #bbf7d0;
  }
  .readonly-value-badge.pnl.loss {
    background: #fef2f2;
    color: #991b1b;
    border-color: #fecaca;
  }

  .execution-timeline-section {
    margin-top: 1.5rem;
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f8fafc;
    border-radius: 12px;
    border: 1px solid #e2e8f0;
  }

  .section-subtitle {
    display: block !important;
    font-size: 1rem;
    font-weight: 800;
    color: #1e293b;
    margin-bottom: 1rem !important;
  }

  .timeline-container-mini {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .timeline-item-mini {
    background: white;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .item-time {
    font-size: 0.9rem;
    color: #64748b;
  }

  .item-details {
    display: flex;
    gap: 0.75rem;
    align-items: center;
  }

  .badge-mini {
    padding: 2px 8px;
    background: #f1f5f9;
    border-radius: 4px;
    font-size: 0.8rem;
    font-weight: 600;
    color: #475569;
  }

  .badge-mini.pnl.profit {
    color: #059669;
    background: #ecfdf5;
  }
  .badge-mini.pnl.loss {
    color: #dc2626;
    background: #fef2f2;
  }
  .badge-mini.ticket {
    font-family: monospace;
    color: #94a3b8;
  }
  /* 圖片放大模態框相關 styles ...略... */

  .btn-icon {
    background: none;
    border: 1px solid #63b3ed;
    color: #3182ce;
    border-radius: 4px;
    padding: 0.2rem 0.6rem;
    font-size: 0.85rem;
    cursor: pointer;
    margin-left: 1rem;
    transition: all 0.2s;
  }

  .btn-icon:hover {
    background: #ebf8ff;
  }

  .btn-share {
    background: #f8fafc;
    color: #64748b;
    border: 1px solid #e2e8f0;
    font-weight: 700;
  }

  .btn-share:hover {
    background: #f1f5f9;
    color: #4f46e5;
    border-color: #6366f1;
  }

  /* Color Tags for Card */
  .card.tag-green {
    border-left: 5px solid #28a745;
  }
  .card.tag-yellow {
    border-left: 5px solid #ffc107;
  }
  .card.tag-red {
    border-left: 5px solid #dc3545;
  }
  .sl-history-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.5rem;
  }

  .sl-chip {
    padding: 0.3rem 0.6rem;
    background: #f1f5f9;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 60px;
    line-height: 1.2;
  }

  .sl-price {
    font-size: 0.75rem;
    color: #334155;
    font-weight: 700;
  }

  .sl-time {
    font-size: 0.6rem;
    color: #64748b;
    font-weight: 500;
  }

  .sl-chip:hover {
    background: #e2e8f0;
    border-color: #cbd5e1;
    transform: translateY(-1px);
  }

  .sl-chip.active {
    background: #0ea5e9;
    border-color: #0284c7;
    box-shadow: 0 2px 4px rgba(14, 165, 233, 0.2);
  }

  .sl-chip.active .sl-price,
  .sl-chip.active .sl-time {
    color: white;
  }
</style>
