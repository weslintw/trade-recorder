<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { dailyPlansAPI, imagesAPI } from '../lib/api';
  import { SYMBOLS, TIMEFRAMES, MARKET_SESSIONS } from '../lib/constants';
  import { selectedAccountId } from '../lib/stores';
  import ImageAnnotator from './ImageAnnotator.svelte';
  import PlanSelectionModal from './PlanSelectionModal.svelte';
  import ShareModal from './ShareModal.svelte';
  import PlanSummaryTable from './PlanSummaryTable.svelte';
  import TradeChart from './TradeChart.svelte';

  import { determineMarketSession, toTradingDateString, formatDate } from '../lib/utils';

  export let id = null;

  let activeSession = determineMarketSession(new Date()); // 預設為當前市場時段
  let loading = false;

  // 複製規劃相關狀態
  let showPlanSelectionModal = false;
  let plansToSelect = [];
  let showShareModal = false;

  // 使用從 constants 引入的時限
  const timeframes = TIMEFRAMES;

  function createDirectionData() {
    return {
      has_signals: false,
      signals: [],
      has_expected_signals: false,
      expected_signals: [], // { name: string, image: string, originalImage: string }
      has_wave: false,
      wave_numbers: [],
      wave_highlight: '',
      signals_image: '',
      signals_originalImage: '',
      expected_signals_image: '',
      expected_signals_originalImage: '',
      wave_image: '',
      wave_originalImage: '',
    };
  }

  // 初始化單個時段的結構
  function createInitialSessionData() {
    const trends = {};
    timeframes.forEach(tf => {
      trends[tf] = {
        directions: [], // 支持多個方向
        long: createDirectionData(),
        short: createDirectionData(),
        image: '',
        originalImage: '',
        chart_config: null,
        trendlines: [],
        // 為了向後兼容，保留舊欄位名稱但主要使用上述結構
      };
    });
    return {
      notes: '',
      trends: trends,
    };
  }

  const symbols = SYMBOLS;

  let formData = {
    account_id: $selectedAccountId,
    plan_date: toTradingDateString(new Date()),
    symbol: SYMBOLS[0],
    sessions: {
      asian: createInitialSessionData(),
      european: createInitialSessionData(),
      us: createInitialSessionData(),
    },
  };

  // 快捷獲取當前分頁資料
  $: currentSessionData = formData.sessions[activeSession];
  $: currentTrends = currentSessionData?.trends || {};

  // 確保 currentTrends 內的所有時區都有基本的方向結構 (long/short/neutral)
  $: {
    if (currentTrends) {
      for (let tf in currentTrends) {
        if (!currentTrends[tf].long) currentTrends[tf].long = createDirectionData();
        if (!currentTrends[tf].short) currentTrends[tf].short = createDirectionData();
        if (currentTrends[tf].chart_config === undefined) currentTrends[tf].chart_config = null;
        if (currentTrends[tf].trendlines === undefined) currentTrends[tf].trendlines = [];
      }
    }
  }
  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const dateParam = params.get('date');
    const sessionParam = params.get('session');
    const symbolParam = params.get('symbol');

    if (dateParam) formData.plan_date = dateParam;
    if (sessionParam) activeSession = sessionParam;
    if (symbolParam) formData.symbol = symbolParam;
  });

  // 達人訊號選項 (同步 SignalGrid.svelte)
  const expertSignalsLong = [
    '向下蘇美',
    '起漲靠山',
    '雙柱',
    '夾縫',
    '喇叭-上',
    '喇叭-中',
    '喇叭-下',
    '倚天',
    '攻城池上',
  ];
  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '夾縫', '向上蘇美', '雷霆'];

  // 全部訊號清單（去除重複）
  const allExpertSignals = Array.from(new Set([...expertSignalsLong, ...expertSignalsShort]));

  // 波浪數字選項
  const waveNumbers = ['1', '2', '3', '4', '5'];

  // 切換時區的訊號選擇
  function toggleTimeframeSignal(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.signals || [];
    const index = signals.indexOf(signalName);

    if (index >= 0) {
      // 取消選擇
      target.signals = signals.filter((_, i) => i !== index);
    } else {
      // 新增選擇
      target.signals = [...signals, signalName];
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 切換預期訊號
  function toggleExpectedSignal(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    if (!target.expected_signals) target.expected_signals = [];

    const index = target.expected_signals.findIndex(s => s.name === signalName);

    if (index >= 0) {
      // 取消選擇
      target.expected_signals = target.expected_signals.filter((_, i) => i !== index);
    } else {
      // 新增選擇
      target.expected_signals = [
        ...target.expected_signals,
        { name: signalName, image: '', originalImage: '' },
      ];
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 檢查預期訊號是否被選中
  function isExpectedSignalSelected(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.expected_signals || [];
    return signals.some(s => s.name === signalName);
  }

  // 檢查時區訊號是否被選中
  function isTimeframeSignalSelected(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.signals || [];
    return signals.includes(signalName);
  }

  // 點擊波浪數字
  function clickWaveNumber(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const selectedNumbers = target.wave_numbers || [];
    const currentHighlight = target.wave_highlight || '';

    // 如果這個數字已經被選中
    if (selectedNumbers.includes(number)) {
      // 如果是綠色（未高亮），變成紅色（高亮）
      if (currentHighlight !== number) {
        target.wave_highlight = number;
      } else {
        // 如果已經是紅色，變回綠色
        target.wave_highlight = '';
      }
    } else {
      // 數字未被選中，嘗試選中
      if (selectedNumbers.length === 0) {
        target.wave_numbers = [number];
        target.wave_highlight = '';
      } else if (selectedNumbers.length === 1) {
        const existingNum = parseInt(selectedNumbers[0]);
        const newNum = parseInt(number);

        if (Math.abs(existingNum - newNum) === 1) {
          target.wave_numbers = [selectedNumbers[0], number].sort();
          target.wave_highlight = '';
        }
      } else if (selectedNumbers.length === 2) {
        target.wave_numbers = [number];
        target.wave_highlight = '';
      }
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 切換趨勢方向 (多/空/整理 三選一)，再次點選可取消
  function toggleTrendDirection(timeframe, selection) {
    const directions = currentTrends[timeframe].directions || [];

    // 判斷目前狀態
    const isLong = directions.length === 1 && directions[0] === 'long';
    const isShort = directions.length === 1 && directions[0] === 'short';
    const isNeutral =
      directions.length === 2 && directions.includes('long') && directions.includes('short');

    let newDirections = [];

    if (selection === 'long') {
      // 如果目前就是純多頭，則取消；否則設為純多頭
      newDirections = isLong ? [] : ['long'];
    } else if (selection === 'short') {
      // 如果目前就是純空頭，則取消；否則設為純空頭
      newDirections = isShort ? [] : ['short'];
    } else if (selection === 'neutral') {
      // 如果目前就是整理(雙向)，則取消；否則設為整理(雙向)
      newDirections = isNeutral ? [] : ['long', 'short'];
    }

    currentTrends[timeframe].directions = newDirections;

    // 向後兼容：如果只有一個方向，也更新舊的 direction 欄位
    if (newDirections.length === 1) {
      currentTrends[timeframe].direction = newDirections[0];
    } else if (newDirections.length === 0) {
      currentTrends[timeframe].direction = '';
    } else {
      currentTrends[timeframe].direction = 'both';
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 檢查波浪數字是否被選中（綠色）
  function isWaveNumberSelected(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const selectedNumbers = target?.wave_numbers || [];
    return (
      selectedNumbers.includes(number.toString()) || selectedNumbers.includes(parseInt(number))
    );
  }

  // 檢查波浪數字是否被高亮（紅色）
  function isWaveNumberHighlighted(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const highlight = target?.wave_highlight;
    return highlight === number.toString() || highlight === parseInt(number);
  }

  // 圖片放大相關
  let enlargedImage = null;
  let enlargedImageTitle = '';
  let enlargedImageContext = null;
  let enlargedOriginalImage = null;
  let showAnnotator = false;

  // 用於強制重新渲染波浪按鈕的響應式變量
  let waveButtonKey = 0;

  // 載入規劃（如果是編輯模式）
  if (id) {
    loadPlan();
  }

  async function loadPlan() {
    try {
      loading = true;
      const response = await dailyPlansAPI.getOne(id);
      const data = response.data;
      const trendAnalysis = data.trend_analysis ? JSON.parse(data.trend_analysis) : null;

      formData.plan_date = toTradingDateString(new Date(data.plan_date));
      formData.symbol = data.symbol || SYMBOLS[0];

      if (trendAnalysis && trendAnalysis.asian) {
        // 新格式：包含各時段
        formData.sessions = trendAnalysis;
      } else if (trendAnalysis) {
        // 舊格式：遷移至當前 market_session
        const session = data.market_session || 'asian';
        formData.sessions[session] = {
          notes: data.notes || '',
          trends: trendAnalysis,
        };
      }

      // 檢查並補足資料結構 (用於相容舊資料與新格式)
      Object.keys(formData.sessions).forEach(s => {
        const sess = formData.sessions[s];
        if (sess && sess.trends) {
          Object.keys(sess.trends).forEach(tf => {
            const t = sess.trends[tf];
            if (!t) return;

            // 初始化新欄位
            if (!t.directions) {
              t.directions = t.direction
                ? t.direction === 'both'
                  ? ['long', 'short']
                  : [t.direction]
                : [];
            }
            if (!t.long) t.long = createDirectionData();
            if (!t.short) t.short = createDirectionData();
            if (!t.chart_config) t.chart_config = null;
            if (!t.trendlines) t.trendlines = [];

            // 遷移舊資料到 long 或 short 下 (如果舊資料有 direction)
            if (t.direction && t.direction !== 'both') {
              const dir = t.direction; // 'long' 或 'short'
              const target = t[dir];

              // 如果 target 目前是空的，則遷移過來
              if (target.signals.length === 0 && !target.signals_image) {
                target.signals = t.signals || [];
                target.has_signals = t.has_signals || false;
                target.signals_image = t.signals_image || '';
                target.signals_originalImage = t.signals_originalImage || '';
              }
              if (target.wave_numbers.length === 0 && !target.wave_image) {
                target.wave_numbers = t.wave_numbers || [];
                target.has_wave = t.has_wave || false;
                target.wave_highlight = t.wave_highlight || '';
                target.wave_image = t.wave_image || '';
                target.wave_originalImage = t.wave_originalImage || '';
              }
            } else if (!t.direction && (t.signals?.length > 0 || t.wave_numbers?.length > 0)) {
              // 如果沒有方向但有資料，暫且放到 long 或者保持在頂層？
              // 為了兼容性，UI 會在 directions 有值時顯示對應區塊。
              // 若 directions 為空，我們可能需要一個『通用』顯示區塊，或者引導使用者選方向。
            }

            // 更新標籤 (優先尊重資料庫中的 boolean 值，若為 undefined 才依據資料內容推斷)
            if (t.long.has_signals === undefined || t.long.has_signals === null) {
              t.long.has_signals = t.long.signals?.length > 0 || !!t.long.signals_image;
            }
            if (t.long.has_expected_signals === undefined || t.long.has_expected_signals === null) {
              t.long.has_expected_signals = t.long.expected_signals?.length > 0;
            }
            if (t.long.has_wave === undefined || t.long.has_wave === null) {
              t.long.has_wave = t.long.wave_numbers?.length > 0 || !!t.long.wave_image;
            }
            if (t.short.has_signals === undefined || t.short.has_signals === null) {
              t.short.has_signals = t.short.signals?.length > 0 || !!t.short.signals_image;
            }
            if (
              t.short.has_expected_signals === undefined ||
              t.short.has_expected_signals === null
            ) {
              t.short.has_expected_signals =
                t.short.expected_signals?.length > 0 || !!t.short.expected_signals_image;
            }
            if (t.short.has_wave === undefined || t.short.has_wave === null) {
              t.short.has_wave = t.short.wave_numbers?.length > 0 || !!t.short.wave_image;
            }
          });
        }
      });
      formData = formData;
    } catch (error) {
      console.error('載入規劃失敗:', error);
      alert('載入規劃資料失敗');
    } finally {
      loading = false;
    }
  }

  async function handleSubmit() {
    try {
      const submitData = {
        account_id: $selectedAccountId,
        plan_date: new Date(formData.plan_date).toISOString(),
        symbol: formData.symbol,
        market_session: 'all', // 標記為整合格式
        notes: 'Session-based unified plan',
        trend_analysis: JSON.stringify(formData.sessions),
      };

      if (id) {
        await dailyPlansAPI.update(id, submitData);
        alert('規劃已更新');
      } else {
        const response = await dailyPlansAPI.create(submitData);
        alert('規劃已建立');
        // 如果 API 有回傳新建立的 ID，跳轉到編輯頁面以繼續編輯
        if (response.data && response.data.id) {
          navigate(`/plans/edit/${response.data.id}`, { replace: true });
        } else {
          navigate('/plans');
        }
      }
    } catch (error) {
      console.error('保存失敗:', error);
      const errorMessage = error.response?.data?.error || '保存規劃失敗';
      alert(errorMessage);
    }
  }

  // 處理趨勢圖片貼上 (優化版：改為直接上傳伺服器，不再存 Base64)
  async function handleTrendImagePaste(
    event,
    timeframe,
    imageType = 'trend',
    direction = null,
    signalName = null
  ) {
    const items = (event.clipboardData || event.originalEvent.clipboardData).items;

    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = item.getAsFile();
        await uploadTrendImage(file, timeframe, imageType, direction, signalName);
        break;
      }
    }
  }

  async function uploadTrendImage(
    file,
    timeframe,
    imageType = 'trend',
    direction = null,
    signalName = null
  ) {
    if (!file) return;
    try {
      const formDataToUpload = new FormData();
      formDataToUpload.append('image', file);
      formDataToUpload.append('symbol', formData.symbol || 'plan');

      // 上傳並取得 URL
      const response = await imagesAPI.upload(formDataToUpload);
      const imageUrl = response.data.path; // 後端回傳的路徑

      const trends = currentTrends[timeframe];
      const target = direction ? trends[direction] : trends;

      // 根據 imageType 設置不同的圖片欄位
      if (imageType === 'signals') {
        target.signals_image = imageUrl;
        if (!target.signals_originalImage) {
          target.signals_originalImage = imageUrl;
        }
      } else if (imageType === 'expected_signals') {
        target.expected_signals_image = imageUrl;
        if (!target.expected_signals_originalImage) {
          target.expected_signals_originalImage = imageUrl;
        }
      } else if (imageType === 'wave') {
        target.wave_image = imageUrl;
        if (!target.wave_originalImage) {
          target.wave_originalImage = imageUrl;
        }
      } else {
        trends.image = imageUrl;
        if (!trends.originalImage) {
          trends.originalImage = imageUrl;
        }
      }

      // 強制觸發 Svelte 響應式更新
      formData = formData;
      waveButtonKey++;
    } catch (error) {
      console.error('圖片上傳失敗:', error);
      alert('圖片處理失敗，請重試');
    }
  }

  let trendFileInput;
  let currentUploadContext = null;

  function triggerTrendUpload(timeframe, imageType = 'trend', direction = null, signalName = null) {
    currentUploadContext = { timeframe, imageType, direction, signalName };
    if (trendFileInput) trendFileInput.click();
  }

  function handleTrendFileSelect(e) {
    const file = e.target.files[0];
    if (file && currentUploadContext) {
      const { timeframe, imageType, direction, signalName } = currentUploadContext;
      uploadTrendImage(file, timeframe, imageType, direction, signalName);
    }
    e.target.value = ''; // Reset
    currentUploadContext = null;
  }

  // 移除趨勢圖片
  function removeTrendImage(timeframe, imageType = 'trend', direction = null, signalName = null) {
    const trends = currentTrends[timeframe];
    const target = direction ? trends[direction] : trends;

    if (imageType === 'signals') {
      target.signals_image = '';
      target.signals_originalImage = '';
    } else if (imageType === 'expected_signals') {
      target.expected_signals_image = '';
      target.expected_signals_originalImage = '';
    } else if (imageType === 'wave') {
      target.wave_image = '';
      target.wave_originalImage = '';
    } else {
      trends.image = '';
      trends.originalImage = '';
    }

    // 強制觸發 Svelte 響應式更新
    formData = formData;
    waveButtonKey++;
  }

  // 放大圖片
  function enlargeImage(imageSrc, title, context = null) {
    if (!imageSrc) return;
    enlargedImage = imageSrc;
    enlargedImageTitle = title;
    enlargedImageContext = context;
    showAnnotator = false;

    // 獲取原始圖片
    if (context) {
      const trends = currentTrends[context.key];
      const target = context.direction ? trends[context.direction] : trends;

      if (context.type === 'trend') {
        enlargedOriginalImage = trends?.originalImage || imageSrc;
      } else if (context.type === 'signals') {
        enlargedOriginalImage = target?.signals_originalImage || imageSrc;
      } else if (context.type === 'wave') {
        enlargedOriginalImage = target?.wave_originalImage || imageSrc;
      } else if (context.type === 'expected_signals') {
        enlargedOriginalImage = target?.expected_signals_originalImage || imageSrc;
      } else {
        enlargedOriginalImage = imageSrc;
      }
    } else {
      enlargedOriginalImage = imageSrc;
    }
  }

  // 關閉放大圖片
  function closeEnlargedImage() {
    enlargedImage = null;
    enlargedImageTitle = '';
    enlargedImageContext = null;
    showAnnotator = false;
  }

  // 切換標註工具顯示
  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  // 處理標註後的圖片
  async function handleAnnotatedImage(annotatedImageSrc) {
    try {
      // 標註後的圖片是 base64，必須上傳到伺服器 (遵循 MinIO 規則)
      // 將 base64 轉換為 Blob
      const res = await fetch(annotatedImageSrc);
      const blob = await res.blob();
      const file = new File([blob], 'annotated_plan.png', { type: 'image/png' });

      const uploadData = new FormData();
      uploadData.append('image', file);
      uploadData.append('symbol', formData.symbol || 'plan');

      const uploadRes = await imagesAPI.upload(uploadData);
      const serverPath = uploadRes.data.path;

      if (!enlargedImageContext) {
        enlargedImage = serverPath;
        return;
      }

      const { type, key, direction } = enlargedImageContext;
      const trends = currentTrends[key];
      const target = direction ? trends[direction] : trends;

      if (type === 'trend') {
        trends.image = serverPath;
      } else if (type === 'signals') {
        target.signals_image = serverPath;
      } else if (type === 'wave') {
        target.wave_image = serverPath;
      } else if (type === 'expected_signals') {
        target.expected_signals_image = serverPath;
      }

      // 強制觸發 Svelte 響應式更新
      formData = formData;
      waveButtonKey++;

      // 更新目前顯示的圖片路徑
      enlargedImage = serverPath;
      showAnnotator = false; // 保存後切換回查看模式
    } catch (error) {
      console.error('保存標註圖片失敗:', error);
      alert('無法儲存標註後的圖片，請稍後再試');
    }
  }
  // 複製上一次的規劃 (開啟選單)
  async function copyLastPlan() {
    try {
      const response = await dailyPlansAPI.getAll({
        page: 1,
        page_size: 3, // 取最近的 3 筆讓使用者選
        account_id: formData.account_id, // 必須指定帳號
        symbol: formData.symbol, // 必須指定品種
        sort: 'plan_date', // 假設後端預設就是依日期排序
        desc: true,
      });

      if (response.data && response.data.data && response.data.data.length > 0) {
        plansToSelect = response.data.data;
        showPlanSelectionModal = true;
      } else {
        alert('找不到該帳號與品種過去的規劃紀錄。');
      }
    } catch (error) {
      console.error('複製規劃失敗:', error);
      alert('無法取得上一筆規劃資料');
    }
  }

  // 處理選單確認後的動作
  function handlePlanSelection({ plan, sourceContent, targetSession, sourceSessionKey }) {
    if (plan && sourceContent && targetSession) {
      executeCopyPlan(plan, sourceContent, targetSession, sourceSessionKey);
    }
  }

  // 執行複製邏輯
  function executeCopyPlan(lastPlan, sourceContent, targetSession, sourceSessionKey) {
    if (sourceContent) {
      // 深拷貝以確保圖片字串是複製的，非引用
      const copiedData = JSON.parse(JSON.stringify(sourceContent));

      // 這裡的 copiedData 應該是單一個 Session 的資料結構 { notes:..., trends:... }
      // 或者如果是舊格式 (sourceSessionKey === 'all')，可能是包含 trends 的大物件

      if (sourceSessionKey === 'all') {
        // 舊格式處理：嘗試把整包舊資料塞進目標 session
        // 如果舊資料結構像 { asian:..., european:... } 則無法直接塞
        // 但如果是更舊的 { notes:..., trends: { H1:..., H4:... } } 則可以
        if (copiedData.trends && !copiedData.asian) {
          formData.sessions[targetSession] = copiedData;
        } else {
          // 結構複雜，無法精確轉換，提示使用者
          alert('該規劃格式過舊，無法精確複製到單一時段。');
          return;
        }
      } else {
        // 新格式：直接覆蓋目標 session
        // 保險起見，檢查一下結構
        if (copiedData.trends) {
          formData.sessions[targetSession] = copiedData;
        } else {
          console.error('複製來源結構異常', copiedData);
          alert('複製來源資料結構異常。');
          return;
        }
      }

      // 重新計算 has_signals / has_wave 標記，確保 UI 正確顯示
      const sess = formData.sessions[targetSession];
      if (sess && sess.trends) {
        Object.keys(sess.trends).forEach(tf => {
          const t = sess.trends[tf];
          // 處理可能有 null 的情況
          if (!t) return;
          if (t.signals?.length > 0 || t.signals_image) t.has_signals = true;
          if (t.wave_numbers?.length > 0 || t.wave_image) t.has_wave = true;
        });
      }

      formData = formData; // 觸發更新
      waveButtonKey++; // 強制刷新 UI 元件

      // 切換到目標分頁，讓使用者立刻看到結果
      activeSession = targetSession;

      alert(`已成功將 ${formatDate(lastPlan.plan_date)} 的內容複製到 ${targetSession} 時段！`);
    } else {
      alert('該筆規劃沒有詳細內容可複製。');
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && enlargedImage) {
      closeEnlargedImage();
    }
  }

  // 取得圖片 URL 的 Helper (相容 Base64 與 伺服器路徑)
  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) {
      return src;
    }
    return imagesAPI.getUrl(src);
  }

  // 檢查時段是否有任何資料
  function hasSessionData(sessionKey, currentData) {
    const session = currentData.sessions[sessionKey];
    if (!session) return false;

    // 檢查是否有備註
    if (session.notes && session.notes.trim()) return true;

    // 檢查各時區是否有資料
    if (session.trends) {
      for (const tf of TIMEFRAMES) {
        const trend = session.trends[tf];
        if (!trend) continue;

        // 檢查方向
        if (trend.directions && trend.directions.length > 0) return true;

        // 檢查圖片 (頂層)
        if (trend.image) return true;

        // 檢查圖表配置與線條
        if (trend.chart_config) return true;
        if (trend.trendlines && trend.trendlines.length > 0) return true;

        // 檢查多/空具體內容
        for (const dir of ['long', 'short']) {
          const dData = trend[dir];
          if (!dData) continue;

          if (dData.signals && dData.signals.length > 0) return true;
          if (dData.expected_signals && dData.expected_signals.length > 0) return true;
          if (dData.wave_numbers && dData.wave_numbers.length > 0) return true;
          if (dData.signals_image || dData.wave_image) return true;
        }
      }
    }

    return false;
  }

  // 響應式追蹤各時段狀態
  $: sessionStatus = {
    asian: hasSessionData('asian', formData),
    european: hasSessionData('european', formData),
    us: hasSessionData('us', formData),
  };

  // 儲存規劃圖表配置
  function handleChartConfigSave(timeframe, config) {
    if (!currentTrends[timeframe]) return;
    currentTrends[timeframe].chart_config = config;
    // 這裡我們不強制請求後端，交由最後的 handleSubmit 統一儲存，
    // 但為了讓 Svelte 偵測到深層變動，我們觸發一次參考更新
    formData = formData;
  }

  // 儲存規劃趨勢線
  function handleTrendlinesSave(timeframe, lines) {
    if (!currentTrends[timeframe]) return;
    currentTrends[timeframe].trendlines = lines;
    formData = formData;
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if loading}
  <div class="loading-overlay">
    <div class="loader"></div>
    <div class="loading-text">正在讀取規劃資料...</div>
  </div>
{:else}
  <div class="card plan-form-card">
    <div class="card-header-actions">
      <h2>{id ? '編輯每日盤面規劃' : '新增每日盤面規劃'}</h2>
      <div class="header-btns">
        <button type="button" class="btn btn-primary" on:click={handleSubmit}>
          {id ? '💾 更新規劃' : '✅ 建立規劃'}
        </button>

        {#if id}
          <button
            type="button"
            class="btn btn-outline-share"
            on:click={() => (showShareModal = true)}
          >
            📤 分享
          </button>
        {/if}

        <button type="button" class="btn btn-secondary" on:click={() => navigate('/')}>
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
          </span> 返回
        </button>
      </div>
    </div>

    <form on:submit|preventDefault={handleSubmit}>
      <!-- 基本資料 -->
      <div class="form-section">
        <h3>📅 基本資料</h3>

        <!-- 規劃日期 -->
        <div class="form-group">
          <label for="plan_date">規劃日期</label>
          <div class="date-input-group">
            <input
              type="date"
              id="plan_date"
              class="form-control"
              bind:value={formData.plan_date}
              required
            />
            <button
              type="button"
              class="btn btn-outline-info"
              on:click={copyLastPlan}
              title="複製上一筆規劃的內容（含圖片）"
            >
              📋 複製上次規劃
            </button>
          </div>
        </div>

        <!-- 交易品種 -->
        <div class="form-group">
          <label for="symbol">交易品種</label>
          <select id="symbol" class="form-control" bind:value={formData.symbol}>
            {#each symbols as sym}
              <option value={sym}>{sym}</option>
            {/each}
          </select>
        </div>

        <!-- 市場時段 (分頁切換已移至下方趨勢區塊) -->

        <!-- 快速總覽 (全時段) -->
        <div class="quick-overview-section">
          <div class="section-title">
            <span>📊 快速總覽 (全時段)</span>
            <div class="summary-legend-inline">
              <div class="legend-item"><span class="tag-mini established">達</span> 成立</div>
              <div class="legend-item"><span class="tag-mini expected">預</span> 預期</div>
              <div class="legend-item"><span class="tag-mini wave-tag">波</span> 波浪</div>
            </div>
          </div>
          <PlanSummaryTable trendData={formData.sessions} detailed={true} />
        </div>
      </div>

      <!-- 當前各時區趨勢與時段選擇 (整合式表格佈局) -->
      <div class="session-trend-layout">
        <!-- 左側垂直時段選擇 -->
        <div class="session-sidebar-vertical">
          {#each MARKET_SESSIONS as session}
            <button
              type="button"
              class="session-tab-vertical {session.value}"
              class:active={activeSession === session.value}
              on:click={() => (activeSession = session.value)}
            >
              <div class="tab-content">
                <span class="tab-icon">{session.icon}</span>
                <span class="tab-label">{session.label[0]}</span>
                {#if sessionStatus[session.value]}
                  <span class="data-indicator-v"></span>
                {/if}
              </div>
            </button>
          {/each}
        </div>

        <!-- 右側趨勢內容 -->
        <div class="trend-content-main">
          <!-- 該時段備註 -->
          <div class="form-group session-notes-area">
            <label for="notes" class="trend-label"
              >📝 備註 ({MARKET_SESSIONS.find(s => s.value === activeSession)?.label})</label
            >
            <textarea
              id="notes"
              class="form-control"
              bind:value={currentSessionData.notes}
              rows="3"
              placeholder="今日時段盤面重點、注意事項..."
            ></textarea>
          </div>

          <!-- 各時區趨勢 -->
          <div class="form-group trend-analysis-section">
            <label class="trend-label"
              >📊 當前各時區趨勢 ({MARKET_SESSIONS.find(s => s.value === activeSession)
                ?.label})</label
            >
            <div class="trend-grid">
              {#each TIMEFRAMES as timeframe}
                <div
                  class="trend-item"
                  tabindex="0"
                  on:paste={e => handleTrendImagePaste(e, timeframe)}
                >
                  <label class="timeframe-label">{timeframe}</label>

                  <!-- cTrader K線圖表 -->
                  <div class="planning-chart-container">
                    <TradeChart
                      mode="plan"
                      accountId={$selectedAccountId}
                      symbol={formData.symbol}
                      planTimeframe={timeframe}
                      planDate={formData.plan_date ? formData.plan_date.slice(0, 10) : null}
                      planSession={activeSession}
                      initialConfig={currentTrends[timeframe]?.chart_config}
                      initialTrendlines={currentTrends[timeframe]?.trendlines || []}
                      onSaveConfig={config => handleChartConfigSave(timeframe, config)}
                      onSaveTrendlines={lines => handleTrendlinesSave(timeframe, lines)}
                    />
                  </div>

                  <!-- 多空選擇 -->
                  <div class="trend-options">
                    <button
                      type="button"
                      class="trend-option long"
                      class:active={currentTrends[timeframe]?.directions?.length === 1 &&
                        currentTrends[timeframe]?.directions?.includes('long')}
                      on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'long')}
                    >
                      <span class="trend-name">多</span>
                    </button>
                    <button
                      type="button"
                      class="trend-option neutral"
                      class:active={currentTrends[timeframe]?.directions?.length === 2 &&
                        currentTrends[timeframe]?.directions?.includes('long') &&
                        currentTrends[timeframe]?.directions?.includes('short')}
                      on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'neutral')}
                    >
                      <span class="trend-name">整</span>
                    </button>
                    <button
                      type="button"
                      class="trend-option short"
                      class:active={currentTrends[timeframe]?.directions?.length === 1 &&
                        currentTrends[timeframe]?.directions?.includes('short')}
                      on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'short')}
                    >
                      <span class="trend-name">空</span>
                    </button>
                  </div>

                  <!-- 分析區塊：根據選擇的方向顯示 -->
                  {#each currentTrends[timeframe]?.directions || [] as dir}
                    <div
                      class="direction-analysis-box"
                      class:long={dir === 'long'}
                      class:short={dir === 'short'}
                    >
                      <div class="direction-badge">
                        {dir === 'long' ? '📈 多頭分析' : '📉 空頭分析'}
                      </div>

                      <!-- 已成立的達人訊號選擇 -->
                      <div class="timeframe-signals">
                        <label class="section-label inline-check">
                          <input
                            type="checkbox"
                            bind:checked={currentTrends[timeframe][dir].has_signals}
                          />
                          已成立的達人訊號
                        </label>

                        {#if currentTrends[timeframe][dir].has_signals}
                          <div class="signal-chips">
                            {#each allExpertSignals as signal (waveButtonKey + '-' + timeframe + '-' + dir + '-established-' + signal)}
                              <button
                                type="button"
                                class="signal-chip"
                                class:active={isTimeframeSignalSelected(timeframe, dir, signal)}
                                on:click|stopPropagation={() =>
                                  toggleTimeframeSignal(timeframe, dir, signal)}
                              >
                                {signal}
                              </button>
                            {/each}
                          </div>

                          <!-- 達人訊號圖片 -->
                          {#if currentTrends[timeframe][dir].signals_image}
                            <div
                              class="trend-image-preview"
                              on:click|stopPropagation={() =>
                                enlargeImage(
                                  currentTrends[timeframe][dir].signals_image,
                                  `${timeframe} ${dir === 'long' ? '多頭' : '空頭'} 已成立達人訊號圖`,
                                  { type: 'signals', key: timeframe, direction: dir }
                                )}
                            >
                              <img
                                src={getImageUrl(currentTrends[timeframe][dir].signals_image)}
                                alt="{timeframe} 已成立達人訊號"
                                style="pointer-events: none;"
                              />
                              <button
                                type="button"
                                class="remove-image-btn"
                                on:click|stopPropagation={() =>
                                  removeTrendImage(timeframe, 'signals', dir)}
                                title="移除圖片"
                              >
                                ×
                              </button>
                            </div>
                          {:else}
                            <div class="split-upload-area">
                              <button
                                type="button"
                                class="split-upload-btn"
                                on:click|stopPropagation={() =>
                                  triggerTrendUpload(timeframe, 'signals', dir)}
                                title="上傳圖片"
                              >
                                📸
                              </button>
                              <div
                                class="split-paste-zone"
                                on:paste|preventDefault|stopPropagation={e =>
                                  handleTrendImagePaste(e, timeframe, 'signals', dir)}
                                tabindex="0"
                              >
                                貼上訊號圖 (Ctrl+V)
                              </div>
                            </div>
                          {/if}
                        {/if}
                      </div>

                      <!-- 預期產生的達人訊號選擇 -->
                      <div class="timeframe-signals expected">
                        <label class="section-label inline-check">
                          <input
                            type="checkbox"
                            bind:checked={currentTrends[timeframe][dir].has_expected_signals}
                          />
                          預期產生的達人訊號
                        </label>

                        {#if currentTrends[timeframe][dir].has_expected_signals}
                          <div class="signal-chips">
                            {#each allExpertSignals as signal (waveButtonKey + '-' + timeframe + '-' + dir + '-expected-' + signal)}
                              <button
                                type="button"
                                class="signal-chip expected"
                                class:active={isExpectedSignalSelected(timeframe, dir, signal)}
                                on:click|stopPropagation={() =>
                                  toggleExpectedSignal(timeframe, dir, signal)}
                              >
                                {signal}
                              </button>
                            {/each}
                          </div>

                          <!-- 預期訊號圖片 -->
                          {#if currentTrends[timeframe][dir].expected_signals_image}
                            <div
                              class="trend-image-preview"
                              on:click|stopPropagation={() =>
                                enlargeImage(
                                  currentTrends[timeframe][dir].expected_signals_image,
                                  `${timeframe} ${dir === 'long' ? '多頭' : '空頭'} 預期訊號圖`,
                                  { type: 'expected_signals', key: timeframe, direction: dir }
                                )}
                            >
                              <img
                                src={getImageUrl(
                                  currentTrends[timeframe][dir].expected_signals_image
                                )}
                                alt="{timeframe} 預期訊號"
                                style="pointer-events: none;"
                              />
                              <button
                                type="button"
                                class="remove-image-btn"
                                on:click|stopPropagation={() =>
                                  removeTrendImage(timeframe, 'expected_signals', dir)}
                                title="移除圖片"
                              >
                                ×
                              </button>
                            </div>
                          {:else}
                            <div class="split-upload-area">
                              <button
                                type="button"
                                class="split-upload-btn"
                                on:click|stopPropagation={() =>
                                  triggerTrendUpload(timeframe, 'expected_signals', dir)}
                                title="上傳圖片"
                              >
                                📸
                              </button>
                              <div
                                class="split-paste-zone"
                                on:paste|preventDefault|stopPropagation={e =>
                                  handleTrendImagePaste(e, timeframe, 'expected_signals', dir)}
                                tabindex="0"
                              >
                                貼上訊號圖 (Ctrl+V)
                              </div>
                            </div>
                          {/if}
                        {/if}
                      </div>

                      <!-- 波浪浪數選擇 -->
                      <div class="timeframe-wave">
                        <label class="section-label inline-check">
                          <input
                            type="checkbox"
                            bind:checked={currentTrends[timeframe][dir].has_wave}
                          />
                          波浪浪數
                        </label>

                        {#if currentTrends[timeframe][dir].has_wave}
                          <div class="wave-numbers">
                            {#each waveNumbers as num (waveButtonKey + '-' + timeframe + '-' + dir + '-' + num)}
                              <button
                                type="button"
                                class="wave-number-btn"
                                class:selected={isWaveNumberSelected(timeframe, dir, num)}
                                class:highlighted={isWaveNumberHighlighted(timeframe, dir, num)}
                                on:click|stopPropagation={() =>
                                  clickWaveNumber(timeframe, dir, num)}
                              >
                                {num}
                              </button>
                            {/each}
                          </div>

                          <!-- 波浪圖片 -->
                          {#if currentTrends[timeframe][dir].wave_image}
                            <div
                              class="trend-image-preview"
                              on:click|stopPropagation={() =>
                                enlargeImage(
                                  currentTrends[timeframe][dir].wave_image,
                                  `${timeframe} ${dir === 'long' ? '多頭' : '空頭'} 波浪圖`,
                                  {
                                    type: 'wave',
                                    key: timeframe,
                                    direction: dir,
                                  }
                                )}
                            >
                              <img
                                src={getImageUrl(currentTrends[timeframe][dir].wave_image)}
                                alt="{timeframe} 波浪"
                                style="pointer-events: none;"
                              />
                              <button
                                type="button"
                                class="remove-image-btn"
                                on:click|stopPropagation={() =>
                                  removeTrendImage(timeframe, 'wave', dir)}
                                title="移除圖片"
                              >
                                ×
                              </button>
                            </div>
                          {:else}
                            <div class="split-upload-area">
                              <button
                                type="button"
                                class="split-upload-btn"
                                on:click|stopPropagation={() =>
                                  triggerTrendUpload(timeframe, 'wave', dir)}
                                title="上傳圖片"
                              >
                                📸
                              </button>
                              <div
                                class="split-paste-zone"
                                on:paste|preventDefault|stopPropagation={e =>
                                  handleTrendImagePaste(e, timeframe, 'wave', dir)}
                                tabindex="0"
                              >
                                貼上波浪圖 (Ctrl+V)
                              </div>
                            </div>
                          {/if}
                        {/if}
                      </div>
                    </div>
                  {/each}

                  <!-- 如果沒有選方向，顯示提示 -->
                  {#if (currentTrends[timeframe]?.directions || []).length === 0}
                    <div class="no-direction-hint">請選擇「多」或「空」以開始分析</div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        </div>
      </div>

      <!-- 操作按鈕 -->
      <div class="form-actions">
        <button type="submit" class="btn btn-primary">
          {id ? '💾 更新規劃' : '✅ 建立規劃'}
        </button>
        <button type="button" class="btn btn-secondary" on:click={() => navigate('/')}>
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
          </span> 返回
        </button>
      </div>
    </form>
  </div>

  <!-- 圖片放大模態框 -->
  {#if enlargedImage}
    <div class="image-modal" on:click={closeEnlargedImage}>
      <div class="image-modal-content" on:click|stopPropagation>
        <div class="image-modal-header">
          <h3>{enlargedImageTitle}</h3>
          <div class="image-modal-actions">
            <button class="modal-action-btn" on:click={toggleAnnotator}>
              {showAnnotator ? '👁️ 查看' : '✏️ 標註'}
            </button>
            <button class="image-modal-close" on:click={closeEnlargedImage}>×</button>
          </div>
        </div>

        {#if showAnnotator}
          <ImageAnnotator
            imageSrc={getImageUrl(enlargedImage)}
            originalImageSrc={getImageUrl(enlargedOriginalImage)}
            onSave={handleAnnotatedImage}
          />
        {:else}
          <img src={getImageUrl(enlargedImage)} alt={enlargedImageTitle} class="image-modal-img" />
        {/if}
      </div>
    </div>
  {/if}

  <!-- 規劃選擇模態框 -->
  <PlanSelectionModal
    show={showPlanSelectionModal}
    plans={plansToSelect}
    {activeSession}
    onConfirm={handlePlanSelection}
    onClose={() => (showPlanSelectionModal = false)}
  />

  <ShareModal
    show={showShareModal}
    resourceType="plan"
    resourceId={id}
    resourceTitle={formData.plan_date.replace(/-/g, '') + '_DailyPlan'}
    onClose={() => (showShareModal = false)}
  />

  <input
    type="file"
    accept="image/*"
    style="display: none;"
    bind:this={trendFileInput}
    on:change={handleTrendFileSelect}
  />

  <style>
    /* 讓規劃頁面可以橫向擴張到全螢幕寬度 */
    :global(.container:has(.plan-form-card)) {
      max-width: 95% !important;
      width: 95% !important;
    }

    .plan-form-card {
      width: 100%;
      max-width: 100%;
    }

    .card-header-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
    }

    .header-btns {
      display: flex;
      gap: 0.75rem;
    }

    h2 {
      margin-bottom: 0;
      color: var(--text-main);
    }

    h3 {
      font-size: 1.2rem;
      color: var(--text-main);
      margin-bottom: 1rem;
    }

    .form-section {
      margin-bottom: 2rem;
      padding: 1.5rem;
      background: var(--bg-main);
      border-radius: 12px;
    }

    /* 市場時段垂直側邊欄 */
    .session-trend-layout {
      display: flex;
      gap: 1rem;
      margin-top: 2rem;
      min-height: 400px;
    }

    .trend-content-main {
      flex: 1;
      width: 0; /* 防止子元素撐破 flex 容器 */
      min-width: 0;
    }

    .session-sidebar-vertical {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      padding-top: 0; /* 移除對齊，讓時段分頁不影響備註區塊 */
      position: sticky;
      top: 1rem;
      height: fit-content;
    }

    .session-tab-vertical {
      width: 50px;
      height: 120px;
      border: 1px solid var(--border-color);
      background: var(--card-bg);
      border-radius: 12px;
      cursor: pointer;
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 0;
      position: relative;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.02);
    }

    .session-tab-vertical .tab-content {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 0.5rem;
    }

    .session-tab-vertical .tab-icon {
      font-size: 1.2rem;
    }

    .session-tab-vertical .tab-label {
      writing-mode: vertical-rl;
      text-orientation: upright;
      font-weight: 800;
      font-size: 1rem;
      letter-spacing: 0.2rem;
      color: var(--text-muted);
    }

    .session-tab-vertical.active {
      transform: translateX(-5px);
      box-shadow: 4px 4px 12px rgba(0, 0, 0, 0.08);
      z-index: 2;
    }

    .session-tab-vertical.active .tab-label {
      color: var(--text-main);
    }

    /* 不同時段的活動狀態顏色 */
    .session-tab-vertical.asian.active {
      border-left: 5px solid #3b82f6;
      border-color: rgba(59, 130, 246, 0.3);
    }
    .session-tab-vertical.european.active {
      border-left: 5px solid #ea580c;
      border-color: rgba(234, 88, 12, 0.3);
    }
    .session-tab-vertical.us.active {
      border-left: 5px solid #dc2626;
      border-color: rgba(220, 38, 38, 0.3);
    }

    .session-tab-vertical:hover:not(.active) {
      background: var(--bg-main);
      border-color: var(--border-color);
    }

    .session-notes-area {
      margin-bottom: 2rem;
      background: var(--card-bg);
      padding: 1.5rem;
      border-radius: 12px;
      border: 1px solid var(--border-color);
    }

    .data-indicator-v {
      position: absolute;
      top: 8px;
      right: 8px;
      width: 10px;
      height: 10px;
      background-color: #10b981;
      border-radius: 50%;
      box-shadow: 0 0 0 2px white;
      animation: pulse-green 2s infinite;
    }

    @keyframes pulse-green {
      0% {
        box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
      }
      70% {
        box-shadow: 0 0 0 6px rgba(16, 185, 129, 0);
      }
      100% {
        box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
      }
    }

    .dark-mode .data-indicator-v {
      box-shadow: 0 0 0 2px #2d3748;
    }

    .dark-mode .section-label,
    .dark-mode .signal-chip.expected,
    .dark-mode .wave-number-btn,
    .dark-mode .split-paste-zone {
      color: #ffffff !important;
    }

    /* 趨勢分析 */
    .trend-analysis-section {
      margin-top: 2rem;
    }

    .trend-label {
      display: block;
      font-size: 1.1rem;
      font-weight: 600;
      color: var(--text-main);
      margin-bottom: 1rem;
    }

    .trend-grid {
      display: flex;
      flex-direction: column;
      gap: 2rem;
      width: 100%;
    }

    .trend-item {
      padding: 1rem;
      background: var(--card-bg);
      border: 2px solid var(--border-color);
      border-radius: 12px;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .trend-item:hover {
      border-color: var(--border-color);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    }

    .trend-item:focus {
      outline: none;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    .planning-chart-container {
      width: 100%;
      height: 550px; /* 增加高度以配合寬度 */
      margin-bottom: 1.5rem;
      border-radius: 12px;
      overflow: hidden;
      border: 1px solid var(--border-color);
      background: #0f172a; /* 配合 Lightweight Charts 背景 */
    }

    .timeframe-label {
      display: block;
      font-weight: 600;
      color: var(--text-main);
      margin-bottom: 0.5rem;
      font-size: 0.9rem;
    }

    .trend-options {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 0.75rem;
    }

    .trend-option {
      flex: 1;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0.5rem;
      border: 2px solid var(--border-color);
      background: var(--card-bg);
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.2s ease;
      user-select: none;
      outline: none;
      line-height: 1;
    }

    .trend-option:hover {
      border-color: #667eea;
      background: var(--bg-main);
    }

    .trend-option.active.long {
      background: #fef2f2;
      color: #dc2626;
      border-color: #fee2e2;
    }

    .trend-option.active.short {
      background: #f0fdf4;
      color: #16a34a;
      border-color: #dcfce7;
    }

    .trend-option.active.neutral {
      background: #1d4ed8; /* Deep Blue */
      color: #ffffff;
      border-color: #1e40af;
      font-weight: 700;
    }

    .date-input-group {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }

    .date-input-group input {
      flex: 1;
    }

    .btn-outline-info {
      background: var(--card-bg);
      border: 1px solid #0bc5ea;
      color: #0bc5ea;
      padding: 0.625rem 1rem;
      white-space: nowrap;
    }

    .btn-outline-info:hover {
      background: var(--bg-main);
    }

    .trend-option.active.long {
      border-color: #ef4444;
      background: #ef4444;
    }

    .trend-option.active.short {
      border-color: #10b981;
      background: #10b981;
    }

    .trend-option input[type='radio'] {
      display: none;
    }

    .trend-name {
      font-size: 0.9rem;
      font-weight: 500;
      color: var(--text-main);
    }

    .trend-option.active .trend-name {
      color: white;
    }

    /* 時區訊號選擇 */
    .timeframe-signals {
      margin-top: 0.75rem;
    }

    .section-label {
      display: block;
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--text-muted);
      margin-bottom: 0.5rem;
    }

    .section-label.inline-check {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      cursor: pointer;
      user-select: none;
    }

    .section-label.inline-check input {
      width: 16px;
      height: 16px;
      cursor: pointer;
    }

    .signal-chips {
      display: flex;
      flex-wrap: wrap;
      gap: 0.4rem;
    }

    .signal-chip {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0.3rem 0.6rem;
      border: 1.5px solid var(--border-color);
      border-radius: 6px;
      background: var(--card-bg);
      color: var(--text-main);
      cursor: pointer;
      transition: all 0.2s ease;
      font-size: 0.75rem;
      user-select: none;
      line-height: 1;
    }

    .signal-chip:hover {
      border-color: #667eea;
      background: var(--bg-main);
    }

    .signal-chip.active {
      border-color: #667eea;
      background: #667eea;
      color: white;
    }

    /* 波浪浪數選擇 */
    .timeframe-wave {
      margin-top: 0.75rem;
    }

    .wave-numbers {
      display: flex;
      gap: 0.4rem;
    }

    .wave-number-btn {
      flex: 1;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0.4rem;
      border: 1.5px solid var(--border-color);
      border-radius: 6px;
      background: var(--card-bg);
      cursor: pointer;
      transition: all 0.2s ease;
      font-size: 0.8rem;
      font-weight: 600;
      user-select: none;
      color: var(--text-main);
      line-height: 1;
    }

    .wave-number-btn:hover {
      border-color: #48bb78;
      background: var(--bg-main);
    }

    .wave-number-btn.selected {
      border-color: #48bb78 !important;
      background: #48bb78 !important;
      color: white !important;
    }

    .wave-number-btn.highlighted {
      border-color: #e53e3e !important;
      background: #e53e3e !important;
      color: white !important;
    }

    .trend-image-preview {
      position: relative;
      margin-top: 0.5rem;
      border-radius: 8px;
      overflow: hidden;
      cursor: pointer;
      border: 2px solid #e2e8f0;
    }

    .trend-image-preview:hover {
      border-color: #667eea;
    }

    .trend-image-preview img {
      width: 100%;
      height: auto;
      display: block;
    }

    .remove-image-btn {
      position: absolute;
      top: 0.5rem;
      right: 0.5rem;
      width: 28px;
      height: 28px;
      border: none;
      border-radius: 50%;
      background: rgba(239, 68, 68, 0.9);
      color: white;
      font-size: 1.5rem;
      line-height: 1;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease;
    }

    .remove-image-btn:hover {
      background: rgb(239, 68, 68);
      transform: scale(1.1);
    }

    .trend-image-placeholder {
      margin-top: 0.5rem;
      padding: 1.5rem;
      border: 2px dashed var(--border-color);
      border-radius: 8px;
      text-align: center;
      color: var(--text-muted);
      font-size: 0.85rem;
      cursor: pointer;
      transition: all 0.2s ease;
      outline: none;
    }

    .trend-image-placeholder:hover {
      border-color: #667eea;
      background: var(--bg-main);
      color: #667eea;
    }

    .trend-image-placeholder:focus {
      border-color: #667eea;
      background: var(--bg-main);
      color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    /* 操作按鈕 */
    .form-actions {
      display: flex;
      justify-content: flex-end;
      gap: 1rem;
      margin-top: 2rem;
      padding-top: 2rem;
      border-top: 2px solid #e2e8f0;
    }

    .btn-secondary {
      background: #e2e8f0;
      color: #2d3748;
    }

    .btn-secondary:hover {
      background: #cbd5e0;
    }

    .btn-outline-share {
      background: white;
      color: #64748b;
      border: 1px solid #e2e8f0;
      font-weight: 700;
    }
    .btn-outline-share:hover {
      background: #f8fafc;
      color: #4f46e5;
      border-color: #6366f1;
    }

    .header-btns {
      display: flex;
      gap: 0.75rem;
    }

    /* 圖片放大模態框 */
    .image-modal {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.8);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 1000;
      padding: 2rem;
    }

    .image-modal-content {
      background: var(--card-bg);
      border-radius: 12px;
      max-width: 90vw;
      max-height: 90vh;
      overflow: auto;
      position: relative;
    }

    .image-modal-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1rem 1.5rem;
      border-bottom: 2px solid #e2e8f0;
      position: sticky;
      top: 0;
      background: var(--card-bg);
      z-index: 10;
    }

    .image-modal-header h3 {
      margin: 0;
      font-size: 1.2rem;
      color: #2d3748;
    }

    .image-modal-actions {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }

    .modal-action-btn {
      padding: 0.5rem 1rem;
      border: 2px solid #667eea;
      border-radius: 6px;
      background: white;
      color: #667eea;
      cursor: pointer;
      font-weight: 600;
      transition: all 0.2s ease;
    }

    .modal-action-btn:hover {
      background: #667eea;
      color: white;
    }

    .image-modal-close {
      width: 36px;
      height: 36px;
      border: none;
      border-radius: 50%;
      background: #f56565;
      color: white;
      font-size: 1.5rem;
      line-height: 1;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease;
    }

    .image-modal-close:hover {
      background: #e53e3e;
      transform: rotate(90deg);
    }

    .image-modal-img {
      display: block;
      max-width: 100%;
      height: auto;
      padding: 1rem;
    }

    textarea.form-control {
      resize: vertical;
      font-family: inherit;
    }

    .quick-overview-section {
      margin-top: 1.5rem;
      padding: 1rem;
      background: var(--bg-main);
      border-radius: 12px;
      border: 1px dashed var(--border-color);
    }

    .quick-overview-section .section-title {
      font-size: 0.9rem;
      font-weight: 700;
      color: var(--text-main);
      margin-bottom: 0.75rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .quick-overview-section .section-title {
      color: var(--text-muted);
    }

    .summary-legend-inline {
      display: flex;
      gap: 10px;
      margin-left: auto;
      align-items: center;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 0.7rem;
      font-weight: 600;
      color: var(--text-muted);
    }

    .tag-mini {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 15px;
      height: 15px;
      border-radius: 3px;
      font-size: 0.5rem;
      font-weight: 900;
      color: white;
      flex-shrink: 0;
    }

    .tag-mini.established {
      background: #475569;
    }
    .tag-mini.expected {
      background: #8b5cf6;
    }
    .tag-mini.wave-tag {
      background: #0ea5e9;
    }

    .trend-image-placeholderSmall {
      border: 1.5px dashed var(--border-color);
      border-radius: 8px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.75rem;
      color: var(--text-muted);
      margin-top: 0.5rem;
      cursor: pointer;
      transition: all 0.2s;
    }

    .trend-image-placeholderSmall:hover,
    .trend-image-placeholderSmall:focus {
      background: var(--bg-main);
      border-color: #667eea;
      color: #667eea;
    }

    .direction-analysis-box {
      margin-top: 1rem;
      padding: 1rem;
      border-radius: 10px;
      border: 1px solid var(--border-color);
      background: var(--bg-main);
    }

    .direction-analysis-box.long {
      border-left: 4px solid #ef4444;
    }

    .direction-analysis-box.short {
      border-left: 4px solid #10b981;
    }

    .direction-badge {
      font-size: 0.8rem;
      font-weight: 700;
      margin-bottom: 0.5rem;
      display: inline-block;
      padding: 2px 8px;
      border-radius: 4px;
      background: var(--bg-main);
      color: var(--text-main);
    }

    .no-direction-hint {
      margin-top: 1rem;
      padding: 1.5rem;
      border: 1px dashed var(--border-color);
      border-radius: 10px;
      text-align: center;
      color: var(--text-muted);
      font-size: 0.85rem;
      background: var(--bg-main);
    }

    /* 預期產生的達人訊號 */
    .timeframe-signals.expected {
      margin-top: 1.5rem;
      padding-top: 1rem;
      border-top: 1px dashed var(--border-color);
    }

    .signal-chip.expected {
      border-style: dashed;
      background: var(--card-bg);
      color: #1e40af;
    }

    .dark-mode .signal-chip.expected {
      color: #ffffff;
    }

    .signal-chip.expected.active {
      background: #1e40af;
      color: white;
      border-style: solid;
    }

    .expected-signals-images {
      margin-top: 1rem;
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }

    .expected-signal-item {
      border: 1px solid var(--border-color);
      padding: 0.75rem;
      border-radius: 8px;
      background: var(--card-bg);
    }

    .signal-name-label {
      display: block;
      font-size: 0.75rem;
      font-weight: 700;
      color: var(--text-main);
      margin-bottom: 0.5rem;
    }

    .trend-image-placeholderExtraSmall {
      border: 1.5px dashed var(--border-color);
      border-radius: 6px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.7rem;
      color: var(--text-muted);
      cursor: pointer;
      transition: all 0.2s;
    }

    .trend-image-placeholderExtraSmall:hover,
    .trend-image-placeholderExtraSmall:focus {
      background: var(--bg-main);
      border-color: #3b82f6;
      color: #3b82f6;
    }

    /* Mobile Responsive Optimizations */
    @media (max-width: 950px) {
      .trend-grid {
        grid-template-columns: repeat(3, 1fr);
      }
    }

    @media (max-width: 768px) {
      .form-header {
        flex-direction: column;
        align-items: stretch;
        gap: 1rem;
      }
      .header-btns {
        width: 100%;
      }
      .header-btns .btn {
        flex: 1;
      }
      .trend-grid {
        grid-template-columns: repeat(2, 1fr);
      }
      .form-actions {
        flex-direction: column-reverse;
      }
      .form-actions .btn {
        width: 100%;
      }
    }

    /* 趨勢圖片區域 */
    .trend-image-section {
      margin-top: 1rem;
      margin-bottom: 1rem;
    }

    /* 分離式上傳區域樣式 */
    .split-upload-area {
      display: flex;
      gap: 0.5rem;
      margin-top: 0.5rem;
      height: 48px;
    }

    .split-upload-area.mini {
      height: 40px;
    }

    .split-upload-btn {
      flex: 0 0 44px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--card-bg);
      border: 2px dashed var(--border-color);
      border-radius: 8px;
      cursor: pointer;
      font-size: 1.2rem;
      transition: all 0.2s;
    }

    .split-upload-btn:hover {
      border-color: #ef4444;
      background: rgba(239, 68, 68, 0.05);
    }

    .split-paste-zone {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--bg-main);
      border: 2px dashed var(--border-color);
      border-radius: 8px;
      padding: 0 0.75rem;
      font-size: 0.8rem;
      font-weight: 500;
      color: var(--text-muted);
      cursor: pointer;
      transition: all 0.2s;
      outline: none;
      text-align: center;
    }

    .split-paste-zone:hover,
    .split-paste-zone:focus {
      border-color: #667eea;
      background: rgba(102, 126, 234, 0.05);
      color: var(--text-main);
    }

    @media (max-width: 480px) {
      .trend-grid {
        grid-template-columns: 1fr;
      }
      .wave-numbers {
        flex-wrap: wrap;
      }
      .wave-number-btn {
        flex: none;
        width: calc(20% - 0.4rem);
      }
    }
  </style>
{/if}
