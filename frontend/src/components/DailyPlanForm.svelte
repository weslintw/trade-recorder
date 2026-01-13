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

  import { determineMarketSession } from '../lib/utils';

  export let id = null;

  let activeSession = determineMarketSession(new Date()); // 預設為當前市場時段
  let loading = false;

  // 銴ˊ閬??賊????
  let showPlanSelectionModal = false;
  let plansToSelect = [];
  let showShareModal = false;

  // 雿輻敺?constants 撘????
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
      wave_image: '',
      wave_originalImage: '',
    };
  }

  // ?????畾萇?蝯?
  function createInitialSessionData() {
    const trends = {};
    timeframes.forEach(tf => {
      trends[tf] = {
        directions: [], // ?舀?憭??
        long: createDirectionData(),
        short: createDirectionData(),
        image: '',
        originalImage: '',
        // ?箔????澆捆嚗???甈??迂雿蜓閬蝙?其?餈啁?瑽?
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
    plan_date: new Date().toISOString().slice(0, 10),
    symbol: SYMBOLS[0],
    sessions: {
      asian: createInitialSessionData(),
      european: createInitialSessionData(),
      us: createInitialSessionData(),
    },
  };

  // 敹急?脣??嗅???鞈?
  $: currentSessionData = formData.sessions[activeSession];
  $: currentTrends = currentSessionData.trends;
  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const dateParam = params.get('date');
    const sessionParam = params.get('session');
    const symbolParam = params.get('symbol');

    if (dateParam) formData.plan_date = dateParam;
    if (sessionParam) activeSession = sessionParam;
    if (symbolParam) formData.symbol = symbolParam;
  });

  // ?犖閮??賊?
  const expertSignalsLong = ['向下蘇美', '起漲靠山', '雙柱', '倚天', '攻城池上'];
  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '向上蘇美', '雷霆'];

  // ?券閮?皜
  const allExpertSignals = [...expertSignalsLong, ...expertSignalsShort];

  // 瘜Ｘ答?詨??賊?
  const waveNumbers = ['1', '2', '3', '4', '5'];

  // ?????????
  function toggleTimeframeSignal(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.signals || [];
    const index = signals.indexOf(signalName);

    if (index >= 0) {
      // ???豢?
      target.signals = signals.filter((_, i) => i !== index);
    } else {
      // ?啣??豢?
      target.signals = [...signals, signalName];
    }

    // 撘瑕閫貊 Svelte ?踵?撘??
    formData = formData;
    waveButtonKey++;
  }

  // ????閮?
  function toggleExpectedSignal(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    if (!target.expected_signals) target.expected_signals = [];

    const index = target.expected_signals.findIndex(s => s.name === signalName);

    if (index >= 0) {
      // ???豢?
      target.expected_signals = target.expected_signals.filter((_, i) => i !== index);
    } else {
      // ?啣??豢?
      target.expected_signals = [
        ...target.expected_signals,
        { name: signalName, image: '', originalImage: '' },
      ];
    }

    // 撘瑕閫貊 Svelte ?踵?撘??
    formData = formData;
    waveButtonKey++;
  }

  // 瑼Ｘ??閮??臬鋡恍銝?
  function isExpectedSignalSelected(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.expected_signals || [];
    return signals.some(s => s.name === signalName);
  }

  // 瑼Ｘ??閮??臬鋡恍銝?
  function isTimeframeSignalSelected(timeframe, direction, signalName) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const signals = target.signals || [];
    return signals.includes(signalName);
  }

  // 暺?瘜Ｘ答?詨?
  function clickWaveNumber(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const selectedNumbers = target.wave_numbers || [];
    const currentHighlight = target.wave_highlight || '';

    // 憒??摮歇蝬◤?訾葉
    if (selectedNumbers.includes(number)) {
      // 憒??舐??莎??芷?鈭殷?嚗????莎?擃漁嚗?
      if (currentHighlight !== number) {
        target.wave_highlight = number;
      } else {
        // 憒?撌脩??舐??莎?霈?蝬
        target.wave_highlight = '';
      }
    } else {
      // ?詨??芾◤?訾葉嚗?閰阡銝?
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

    // 撘瑕閫貊 Svelte ?踵?撘??
    formData = formData;
    waveButtonKey++;
  }

  // ??頞典?孵? (憭?蝛??渡? 銝銝)嚗?甈⊿??詨??
  function toggleTrendDirection(timeframe, selection) {
    const directions = currentTrends[timeframe].directions || [];

    // ?斗?桀????
    const isLong = directions.length === 1 && directions[0] === 'long';
    const isShort = directions.length === 1 && directions[0] === 'short';
    const isNeutral =
      directions.length === 2 &&
      directions.includes('long') &&
      directions.includes('short');

    let newDirections = [];

    if (selection === 'long') {
      // 憒??桀?撠望蝝??哨???瘨??血?閮剔蝝???
      newDirections = isLong ? [] : ['long'];
    } else if (selection === 'short') {
      // 憒??桀?撠望蝝征?哨???瘨??血?閮剔蝝征??
      newDirections = isShort ? [] : ['short'];
    } else if (selection === 'neutral') {
      // 憒??桀?撠望?渡?(??)嚗???嚗?身?箸????)
      newDirections = isNeutral ? [] : ['long', 'short'];
    }

    currentTrends[timeframe].directions = newDirections;

    // ???澆捆嚗???????銋?啗???direction 甈?
    if (newDirections.length === 1) {
      currentTrends[timeframe].direction = newDirections[0];
    } else if (newDirections.length === 0) {
      currentTrends[timeframe].direction = '';
    } else {
      currentTrends[timeframe].direction = 'both';
    }

    // 撘瑕閫貊 Svelte ?踵?撘??
    formData = formData;
    waveButtonKey++;
  }

  // 瑼Ｘ瘜Ｘ答?詨??臬鋡恍銝哨?蝬嚗?
  function isWaveNumberSelected(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const selectedNumbers = target?.wave_numbers || [];
    return (
      selectedNumbers.includes(number.toString()) || selectedNumbers.includes(parseInt(number))
    );
  }

  // 瑼Ｘ瘜Ｘ答?詨??臬鋡恍?鈭殷?蝝嚗?
  function isWaveNumberHighlighted(timeframe, direction, number) {
    const target = direction ? currentTrends[timeframe][direction] : currentTrends[timeframe];
    const highlight = target?.wave_highlight;
    return highlight === number.toString() || highlight === parseInt(number);
  }

  // ???曉之?賊?
  let enlargedImage = null;
  let enlargedImageTitle = '';
  let enlargedImageContext = null;
  let enlargedOriginalImage = null;
  let showAnnotator = false;

  // ?冽撘瑕?皜脫?瘜Ｘ答?????霈?
  let waveButtonKey = 0;

  // 頛閬?嚗??蝺刻摩璅∪?嚗?
  if (id) {
    loadPlan();
  }

  async function loadPlan() {
    try {
      loading = true;
      const response = await dailyPlansAPI.getOne(id);
      const data = response.data;
      const trendAnalysis = data.trend_analysis ? JSON.parse(data.trend_analysis) : null;

      formData.plan_date = new Date(data.plan_date).toLocaleDateString('en-CA');
      formData.symbol = data.symbol || SYMBOLS[0];

      if (trendAnalysis && trendAnalysis.asian) {
        // ?唳撘????畾?
        formData.sessions = trendAnalysis;
      } else if (trendAnalysis) {
        // ?撘??瑞宏?喟??market_session
        const session = data.market_session || 'asian';
        formData.sessions[session] = {
          notes: data.notes || '',
          trends: trendAnalysis,
        };
      }

      // 瑼Ｘ銝西?頞唾???瑽?(?冽?詨捆?????唳撘?
      Object.keys(formData.sessions).forEach(s => {
        const sess = formData.sessions[s];
        if (sess && sess.trends) {
          Object.keys(sess.trends).forEach(tf => {
            const t = sess.trends[tf];
            if (!t) return;

            // ???甈?
            if (!t.directions) {
              t.directions = t.direction
                ? t.direction === 'both'
                  ? ['long', 'short']
                  : [t.direction]
                : [];
            }
            if (!t.long) t.long = createDirectionData();
            if (!t.short) t.short = createDirectionData();

            // ?瑞宏??? long ??short 銝?(憒????? direction)
            if (t.direction && t.direction !== 'both') {
              const dir = t.direction; // 'long' ??'short'
              const target = t[dir];

              // 憒? target ?桀??舐征???蝘駁?靘?
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
              // 憒?瘝??孵?雿?鞈?嚗銝??long ????惜嚗?
              // ?箔??澆捆?改?UI ? directions ?潭?憿舐內撠??憛?
              // ??directions ?箇征嚗???賡?閬???＊蝷箏?憛???撠蝙?刻?孵???
            }

            // ?湔璅惜 (?芸?撠?鞈?摨思葉??boolean ?潘??亦 undefined ?????摰寞??
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
            if (t.short.has_expected_signals === undefined || t.short.has_expected_signals === null) {
              t.short.has_expected_signals = t.short.expected_signals?.length > 0;
            }
            if (t.short.has_wave === undefined || t.short.has_wave === null) {
              t.short.has_wave = t.short.wave_numbers?.length > 0 || !!t.short.wave_image;
            }
          });
        }
      });
      formData = formData;
    } catch (error) {
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
        market_session: 'all', // 璅??箸?撘?
        notes: 'Session-based unified plan',
        trend_analysis: JSON.stringify(formData.sessions),
      };

      if (id) {
        await dailyPlansAPI.update(id, submitData);
        alert('規劃已更新');
      } else {
        const response = await dailyPlansAPI.create(submitData);
        alert('規劃已建立');
        // 憒? API ???單撱箇???ID嚗歲頧蝺刻摩?隞亦匱蝥楊頛?
        if (response.data && response.data.id) {
          navigate(`/plans/edit/${response.data.id}`, { replace: true });
        } else {
          navigate('/plans');
        }
      }
    } catch (error) {
      const errorMessage = error.response?.data?.error || '保存規劃失敗';
      alert(errorMessage);
    }
  }

  // ??頞典??鞎潔? (?芸????寧?湔銝隡箸??剁?銝?摮?Base64)
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

        try {
          const formDataToUpload = new FormData();
          formDataToUpload.append('image', file);
          formDataToUpload.append('symbol', formData.symbol || 'plan');

          // 銝銝血?敺?URL
          const response = await imagesAPI.upload(formDataToUpload);
          const imageUrl = response.data.path; // 敺垢??楝敺?

          const trends = currentTrends[timeframe];
          const target = direction ? trends[direction] : trends;

          // ?寞? imageType 閮剔蔭銝?????雿?
          if (imageType === 'signals') {
            target.signals_image = imageUrl;
            if (!target.signals_originalImage) {
              target.signals_originalImage = imageUrl;
            }
          } else if (imageType === 'expected_signals') {
            if (target.expected_signals) {
              const signal = target.expected_signals.find(s => s.name === signalName);
              if (signal) {
                signal.image = imageUrl;
                if (!signal.originalImage) {
                  signal.originalImage = imageUrl;
                }
              }
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

          // 撘瑕閫貊 Svelte ?踵?撘??
          formData = formData;
          waveButtonKey++;
        } catch (error) {
          alert('圖片處理失敗，請重試');
        }
        break;
      }
    }
  }

  // 蝘駁頞典??
  function removeTrendImage(timeframe, imageType = 'trend', direction = null, signalName = null) {
    const trends = currentTrends[timeframe];
    const target = direction ? trends[direction] : trends;

    if (imageType === 'signals') {
      target.signals_image = '';
      target.signals_originalImage = '';
    } else if (imageType === 'expected_signals') {
      if (target.expected_signals) {
        const signal = target.expected_signals.find(s => s.name === signalName);
        if (signal) {
          signal.image = '';
          signal.originalImage = '';
        }
      }
    } else if (imageType === 'wave') {
      target.wave_image = '';
      target.wave_originalImage = '';
    } else {
      trends.image = '';
      trends.originalImage = '';
    }

    // 撘瑕閫貊 Svelte ?踵?撘??
    formData = formData;
    waveButtonKey++;
  }

  // ?曉之??
  function enlargeImage(imageSrc, title, context = null) {
    if (!imageSrc) return;
    enlargedImage = imageSrc;
    enlargedImageTitle = title;
    enlargedImageContext = context;
    showAnnotator = false;

    // ?脣?????
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
        const signal = target?.expected_signals?.find(s => s.name === context.signalName);
        enlargedOriginalImage = signal?.originalImage || imageSrc;
      } else {
        enlargedOriginalImage = imageSrc;
      }
    } else {
      enlargedOriginalImage = imageSrc;
    }
  }

  // ???曉之??
  function closeEnlargedImage() {
    enlargedImage = null;
    enlargedImageTitle = '';
    enlargedImageContext = null;
    showAnnotator = false;
  }

  // ??璅酉撌亙憿舐內
  function toggleAnnotator() {
    showAnnotator = !showAnnotator;
  }

  // ??璅酉敺???
  async function handleAnnotatedImage(annotatedImageSrc) {
    try {
      // 璅酉敺?????base64嚗????喳隡箸???(?萄儐 MinIO 閬?)
      // 撠?base64 頧???Blob
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
        if (target.expected_signals) {
          const signal = target.expected_signals.find(
            s => s.name === enlargedImageContext.signalName
          );
          if (signal) {
            signal.image = serverPath;
          }
        }
      }

      // 撘瑕閫貊 Svelte ?踵?撘??
      formData = formData;
      waveButtonKey++;

      // ?湔?桀?憿舐內???楝敺?
      enlargedImage = serverPath;
      showAnnotator = false; // 靽?敺????亦?璅∪?
    } catch (error) {
      alert('無法儲存標註後的圖片，請稍後再試');
    }
  }
  // 銴ˊ銝?甈∠?閬? (???詨)
  async function copyLastPlan() {
    try {
      const response = await dailyPlansAPI.getAll({
        page: 1,
        page_size: 3, // ??餈? 3 蝑?雿輻?
        account_id: formData.account_id, // 敹???撣唾?
        symbol: formData.symbol, // 敹????車
        sort: 'plan_date', // ?身敺垢?身撠望靘??摨?
        desc: true,
      });

      if (response.data && response.data.data && response.data.data.length > 0) {
        plansToSelect = response.data.data;
        showPlanSelectionModal = true;
      } else {
        alert('找不到該帳號與品種過去的規劃紀錄。');
      }
    } catch (error) {
      console.error('銴ˊ閬?憭望?:', error);
      alert('無法取得上一筆規劃資料');
    }
  }

  // ???詨蝣箄?敺???
  function handlePlanSelection({ plan, sourceContent, targetSession, sourceSessionKey }) {
    if (plan && sourceContent && targetSession) {
      executeCopyPlan(plan, sourceContent, targetSession, sourceSessionKey);
    }
  }

  // ?瑁?銴ˊ?摩
  function executeCopyPlan(lastPlan, sourceContent, targetSession, sourceSessionKey) {
    if (sourceContent) {
      // 瘛望鞎誑蝣箔???摮葡?航?鋆賜?嚗?撘
      const copiedData = JSON.parse(JSON.stringify(sourceContent));

      // ?ㄐ??copiedData ?府?臬銝??Session ????瑽?{ notes:..., trends:... }
      // ????撘?(sourceSessionKey === 'all')嚗?賣? trends ?之?拐辣

      if (sourceSessionKey === 'all') {
        // ?撘????岫???鞈?憛脩璅?session
        // 憒?????瑽? { asian:..., european:... } ?瘜?亙?
        // 雿???渲???{ notes:..., trends: { H1:..., H4:... } } ?隞?
        if (copiedData.trends && !copiedData.asian) {
          formData.sessions[targetSession] = copiedData;
        } else {
          // 結構複雜，無法精確轉換，提示使用者
          alert('該規劃格式過舊，無法精確複製到單一時段。');
          return;
        }
      } else {
        // ?唳撘??湔閬??格? session
        // 靽韏瑁?嚗炎?乩?銝?瑽?
        if (copiedData.trends) {
          formData.sessions[targetSession] = copiedData;
        } else {
          console.error('複製來源結構異常', copiedData);
          alert('複製來源資料結構異常。');
          return;
        }
      }

      // ?閮? has_signals / has_wave 璅?嚗Ⅱ靽?UI 甇?Ⅱ憿舐內
      const sess = formData.sessions[targetSession];
      if (sess && sess.trends) {
        Object.keys(sess.trends).forEach(tf => {
          const t = sess.trends[tf];
          // ???航??null ??瘜?
          if (!t) return;
          if (t.signals?.length > 0 || t.signals_image) t.has_signals = true;
          if (t.wave_numbers?.length > 0 || t.wave_image) t.has_wave = true;
        });
      }

      formData = formData; // 閫貊?湔
      waveButtonKey++; // 撘瑕?瑟 UI ?辣

      // ???啁璅???霈蝙?刻??餌??啁???
      activeSession = targetSession;

      alert(
        `已成功將 ${new Date(lastPlan.plan_date).toLocaleDateString()} 的內容複製到 ${targetSession} 時段！`
      );
    } else {
      alert('該筆規劃沒有詳細內容可複製。');
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && enlargedImage) {
      closeEnlargedImage();
    }
  }

  // ???? URL ??Helper (?詨捆 Base64 ??隡箸??刻楝敺?
  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) {
      return src;
    }
    return imagesAPI.getUrl(src);
  }

  // 瑼Ｘ?挾?臬?遙雿???
  function hasSessionData(sessionKey, currentData) {
    const session = currentData.sessions[sessionKey];
    if (!session) return false;

    // 瑼Ｘ?臬??閮?
    if (session.notes && session.notes.trim()) return true;

    // 瑼Ｘ????臬????
    if (session.trends) {
      for (const tf of TIMEFRAMES) {
        const trend = session.trends[tf];
        if (!trend) continue;

        // 瑼Ｘ?孵?
        if (trend.directions && trend.directions.length > 0) return true;

        // 瑼Ｘ?? (?惜)
        if (trend.image) return true;

        // 瑼Ｘ憭?蝛箏擃摰?
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

  // ?踵?撘蕭頩文??挾???
  $: sessionStatus = {
    asian: hasSessionData('asian', formData),
    european: hasSessionData('european', formData),
    us: hasSessionData('us', formData),
  };
</script>

<svelte:window on:keydown={handleKeydown} />

{#if loading}
  <div class="loading-overlay">
    <div class="loader"></div>
    <div class="loading-text">甇?霈??????..</div>
  </div>
{:else}
  <div class="card">
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
            ? ?澈
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
          </span> 餈?
        </button>
      </div>
    </div>

    <form on:submit|preventDefault={handleSubmit}>
      <!-- ?箸鞈? -->
      <div class="form-section">
        <h3>?? ?箸鞈?</h3>

        <!-- 閬??交? -->
        <div class="form-group">
          <label for="plan_date">閬??交?</label>
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
              title="複製上一筆規劃紀錄"
            >
              ?? 銴ˊ銝活閬?
            </button>
          </div>
        </div>

        <!-- 鈭斗??車 -->
        <div class="form-group">
          <label for="symbol">鈭斗??車</label>
          <select id="symbol" class="form-control" bind:value={formData.symbol}>
            {#each symbols as sym}
              <option value={sym}>{sym}</option>
            {/each}
          </select>
        </div>

        <!-- 撣?挾 (????撌脩宏?喃??寡隅?Ｗ?憛? -->

        <!-- 敹恍蜇閬?(?冽?畾? -->
        <div class="quick-overview-section">
          <div class="section-title">
            <span>?? 敹恍蜇閬?(?冽?畾?</span>
            <div class="summary-legend-inline">
              <div class="legend-item"><span class="tag-mini established">??/span> ??</div>
              <div class="legend-item"><span class="tag-mini expected">??/span> ??</div>
              <div class="legend-item"><span class="tag-mini wave-tag">瘜?/span> 瘜Ｘ答</div>
            </div>
          </div>
          <PlanSummaryTable trendData={formData.sessions} detailed={true} />
        </div>
      </div>

      <!-- ?嗅??挾?酉 (?函??澆??銋?嚗??? -->
      <div class="form-group session-notes-area">
        <label for="notes" class="trend-label">?? ?酉 ({MARKET_SESSIONS.find(s => s.value === activeSession)?.label})</label>
        <textarea
          id="notes"
          class="form-control"
          bind:value={currentSessionData.notes}
          rows="3"
          placeholder="隞?挾?日???釣????.."
        ></textarea>
      </div>

      <!-- ?嗅????頞典??畾菟??(?游?撘”?潔?撅) -->
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
          <!-- ???頞典 -->
          <div class="form-group trend-analysis-section">
            <label class="trend-label">?? ?嗅????頞典 ({MARKET_SESSIONS.find(s => s.value === activeSession)?.label})</label>
            <div class="trend-grid">
              {#each TIMEFRAMES as timeframe}
            <div
              class="trend-item"
              tabindex="0"
              on:paste={e => handleTrendImagePaste(e, timeframe)}
              on:click={e => {
                if (!e.target.closest('.trend-options')) {
                  e.currentTarget.focus();
                }
              }}
            >
              <label class="timeframe-label">{timeframe}</label>

              <!-- 憭征?豢? -->
              <div class="trend-options">
                <button
                  type="button"
                  class="trend-option long"
                  class:active={currentTrends[timeframe]?.directions?.length === 1 &&
                    currentTrends[timeframe]?.directions?.includes('long')}
                  on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'long')}
                >
                  <span class="trend-name">憭?/span>
                </button>
                <button
                  type="button"
                  class="trend-option neutral"
                  class:active={currentTrends[timeframe]?.directions?.length === 2 &&
                    currentTrends[timeframe]?.directions?.includes('long') &&
                    currentTrends[timeframe]?.directions?.includes('short')}
                  on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'neutral')}
                >
                  <span class="trend-name">??/span>
                </button>
                <button
                  type="button"
                  class="trend-option short"
                  class:active={currentTrends[timeframe]?.directions?.length === 1 &&
                    currentTrends[timeframe]?.directions?.includes('short')}
                  on:click|stopPropagation={() => toggleTrendDirection(timeframe, 'short')}
                >
                  <span class="trend-name">蝛?/span>
                </button>
              </div>

              <!-- ???憛??寞??豢???＊蝷?-->
              {#each currentTrends[timeframe]?.directions || [] as dir}
                <div
                  class="direction-analysis-box"
                  class:long={dir === 'long'}
                  class:short={dir === 'short'}
                >
                  <div class="direction-badge">
                    {dir === 'long' ? '?? 憭??' : '?? 蝛粹??'}
                  </div>

                  <!-- 撌脫?蝡??犖閮??豢? -->
                  <div class="timeframe-signals">
                    <label class="section-label inline-check">
                      <input
                        type="checkbox"
                        bind:checked={currentTrends[timeframe][dir].has_signals}
                      />
                      撌脫?蝡??犖閮?
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

                      <!-- ?犖閮??? -->
                      {#if currentTrends[timeframe][dir].signals_image}
                        <div
                          class="trend-image-preview"
                          on:click|stopPropagation={() =>
                            enlargeImage(
                              currentTrends[timeframe][dir].signals_image,
                              `${timeframe} ${dir === 'long' ? '憭' : '蝛粹'} 撌脫?蝡?鈭箄???`,
                              { type: 'signals', key: timeframe, direction: dir }
                            )}
                        >
                          <img
                            src={getImageUrl(currentTrends[timeframe][dir].signals_image)}
                            alt="{timeframe} 已建立訊號圖"
                            style="pointer-events: none;"
                          />
                          <button
                            type="button"
                            class="remove-image-btn"
                            on:click|stopPropagation={() =>
                              removeTrendImage(timeframe, 'signals', dir)}
                            title="蝘駁??"
                          >
                            ?
                          </button>
                        </div>
                      {:else}
                        <div
                          class="trend-image-placeholderSmall"
                          tabindex="0"
                          on:paste|preventDefault|stopPropagation={e =>
                            handleTrendImagePaste(e, timeframe, 'signals', dir)}
                          on:click|stopPropagation={e => e.target.focus()}
                          role="textbox"
                        >
                          ?? 鞎潔?撌脫?蝡???
                        </div>
                      {/if}
                    {/if}
                  </div>

                  <!-- ???Ｙ???鈭箄????-->
                  <div class="timeframe-signals expected">
                    <label class="section-label inline-check">
                      <input
                        type="checkbox"
                        bind:checked={currentTrends[timeframe][dir].has_expected_signals}
                      />
                      ???Ｙ???鈭箄???
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

                      <!-- ??閮????? -->
                      {#if currentTrends[timeframe][dir].expected_signals && currentTrends[timeframe][dir].expected_signals.length > 0}
                        <div class="expected-signals-images">
                          {#each currentTrends[timeframe][dir].expected_signals as signal (waveButtonKey + '-' + timeframe + '-' + dir + '-expected-img-' + signal.name)}
                            <div class="expected-signal-item">
                              <span class="signal-name-label">{signal.name}</span>
                              {#if signal.image}
                                <div
                                  class="trend-image-preview"
                                  on:click|stopPropagation={() =>
                                    enlargeImage(
                                      signal.image,
                                      `${timeframe} ${dir === 'long' ? '憭' : '蝛粹'} ??閮?: ${signal.name}`,
                                      {
                                        type: 'expected_signals',
                                        key: timeframe,
                                        direction: dir,
                                        signalName: signal.name,
                                      }
                                    )}
                                >
                                  <img
                                    src={getImageUrl(signal.image)}
                                    alt="{timeframe} ??閮?: {signal.name}"
                                    style="pointer-events: none;"
                                  />
                                  <button
                                    type="button"
                                    class="remove-image-btn"
                                    on:click|stopPropagation={() =>
                                      removeTrendImage(
                                        timeframe,
                                        'expected_signals',
                                        dir,
                                        signal.name
                                      )}
                                    title="蝘駁??"
                                  >
                                    ?
                                  </button>
                                </div>
                              {:else}
                                <div
                                  class="trend-image-placeholderExtraSmall"
                                  tabindex="0"
                                  on:paste|preventDefault|stopPropagation={e =>
                                    handleTrendImagePaste(
                                      e,
                                      timeframe,
                                      'expected_signals',
                                      dir,
                                      signal.name
                                    )}
                                  on:click|stopPropagation={e => e.target.focus()}
                                  role="textbox"
                                >
                                  ?? 鞎潔? {signal.name} 蝷箸???
                                </div>
                              {/if}
                            </div>
                          {/each}
                        </div>
                      {/if}
                    {/if}
                  </div>

                  <!-- 瘜Ｘ答瘚芣?豢? -->
                  <div class="timeframe-wave">
                    <label class="section-label inline-check">
                      <input
                        type="checkbox"
                        bind:checked={currentTrends[timeframe][dir].has_wave}
                      />
                      瘜Ｘ答瘚芣
                    </label>

                    {#if currentTrends[timeframe][dir].has_wave}
                      <div class="wave-numbers">
                        {#each waveNumbers as num (waveButtonKey + '-' + timeframe + '-' + dir + '-' + num)}
                          <button
                            type="button"
                            class="wave-number-btn"
                            class:selected={isWaveNumberSelected(timeframe, dir, num)}
                            class:highlighted={isWaveNumberHighlighted(timeframe, dir, num)}
                            on:click|stopPropagation={() => clickWaveNumber(timeframe, dir, num)}
                          >
                            {num}
                          </button>
                        {/each}
                      </div>

                      <!-- 瘜Ｘ答?? -->
                      {#if currentTrends[timeframe][dir].wave_image}
                        <div
                          class="trend-image-preview"
                          on:click|stopPropagation={() =>
                            enlargeImage(
                              currentTrends[timeframe][dir].wave_image,
                              `${timeframe} ${dir === 'long' ? '多' : '空'} 波浪圖`,
                              {
                                type: 'wave',
                                key: timeframe,
                                direction: dir,
                              }
                            )}
                        >
                          <img
                            src={getImageUrl(currentTrends[timeframe][dir].wave_image)}
                            alt="{timeframe} 瘜Ｘ答"
                            style="pointer-events: none;"
                          />
                          <button
                            type="button"
                            class="remove-image-btn"
                            on:click|stopPropagation={() =>
                              removeTrendImage(timeframe, 'wave', dir)}
                            title="蝘駁??"
                          >
                            ?
                          </button>
                        </div>
                      {:else}
                        <div
                          class="trend-image-placeholderSmall"
                          tabindex="0"
                          on:paste|preventDefault|stopPropagation={e =>
                            handleTrendImagePaste(e, timeframe, 'wave', dir)}
                          on:click|stopPropagation={e => e.target.focus()}
                          role="textbox"
                        >
                          ?? 鞎潔?瘜Ｘ答??
                        </div>
                      {/if}
                    {/if}
                  </div>
                </div>
              {/each}

              <!-- 憒?瘝??豢??憿舐內?內 -->
              {#if (currentTrends[timeframe]?.directions || []).length === 0}
                <div class="no-direction-hint">隢?????征?誑????</div>
              {/if}
            </div>
              {/each}
            </div>
          </div>
        </div>
      </div>

      <!-- ???? -->
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
          </span> 餈?
        </button>
      </div>
    </form>
  </div>

  <!-- ???曉之璅⊥?獢?-->
  {#if enlargedImage}
    <div class="image-modal" on:click={closeEnlargedImage}>
      <div class="image-modal-content" on:click|stopPropagation>
        <div class="image-modal-header">
          <h3>{enlargedImageTitle}</h3>
          <div class="image-modal-actions">
            <button class="modal-action-btn" on:click={toggleAnnotator}>
              {showAnnotator ? '??儭??亦?' : '?? 璅酉'}
            </button>
            <button class="image-modal-close" on:click={closeEnlargedImage}>?</button>
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

  <!-- 閬??豢?璅⊥?獢?-->
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

  <style>
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

    /* 撣?挾??湧?甈?*/
    .session-trend-layout {
      display: flex;
      gap: 1rem;
      margin-top: 2rem;
      min-height: 400px;
    }

    .session-sidebar-vertical {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      padding-top: 2.8rem; /* 撠?銝頞典璅惜??摨?*/
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

    /* 銝??挾?暑??????*/
    .session-tab-vertical.asian.active { border-left: 5px solid #3b82f6; border-color: rgba(59, 130, 246, 0.3); }
    .session-tab-vertical.european.active { border-left: 5px solid #ea580c; border-color: rgba(234, 88, 12, 0.3); }
    .session-tab-vertical.us.active { border-left: 5px solid #dc2626; border-color: rgba(220, 38, 38, 0.3); }

    .session-tab-vertical:hover:not(.active) {
      background: #f8fafc;
      border-color: #cbd5e0;
    }

    .session-notes-area {
      margin-bottom: 2rem;
      background: var(--card-bg);
      padding: 1rem;
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

    /* 頞典?? */
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
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
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
      border-color: #cbd5e0;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    }

    .trend-item:focus {
      outline: none;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
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
      background: #f7fafc;
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
      background: white;
      border: 1px solid #0bc5ea;
      color: #0bc5ea;
      padding: 0.625rem 1rem;
      white-space: nowrap;
    }

    .btn-outline-info:hover {
      background: #e6fffa;
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

    /* ??閮??豢? */
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
      border: 1.5px solid #cbd5e0;
      border-radius: 6px;
      background: white;
      cursor: pointer;
      transition: all 0.2s ease;
      font-size: 0.75rem;
      user-select: none;
      line-height: 1;
    }

    .signal-chip:hover {
      border-color: #667eea;
      background: #f7fafc;
    }

    .signal-chip.active {
      border-color: #667eea;
      background: #667eea;
      color: white;
    }

    /* 瘜Ｘ答瘚芣?豢? */
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
      border: 1.5px solid #cbd5e0;
      border-radius: 6px;
      background: white;
      cursor: pointer;
      transition: all 0.2s ease;
      font-size: 0.8rem;
      font-weight: 600;
      user-select: none;
      color: #2d3748;
      line-height: 1;
    }

    .wave-number-btn:hover {
      border-color: #48bb78;
      background: #f7fafc;
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
      border: 2px dashed #cbd5e0;
      border-radius: 8px;
      text-align: center;
      color: #a0aec0;
      font-size: 0.85rem;
      cursor: pointer;
      transition: all 0.2s ease;
      outline: none;
    }

    .trend-image-placeholder:hover {
      border-color: #667eea;
      background: #f7fafc;
      color: #667eea;
    }

    .trend-image-placeholder:focus {
      border-color: #667eea;
      background: #edf2f7;
      color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    /* ???? */
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

    /* ???曉之璅⊥?獢?*/
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
      background: white;
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
      background: white;
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
      background: #f8fafc;
      border-radius: 12px;
      border: 1px dashed #cbd5e1;
    }

    .quick-overview-section .section-title {
      font-size: 0.9rem;
      font-weight: 700;
      color: #475569;
      margin-bottom: 0.75rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .quick-overview-section .section-title {
      color: #94a3b8;
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
      color: #64748b;
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

    .tag-mini.established { background: #475569; }
    .tag-mini.expected { background: #8b5cf6; }
    .tag-mini.wave-tag { background: #0ea5e9; }

    .trend-image-placeholderSmall {
      border: 1.5px dashed #cbd5e0;
      border-radius: 8px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.75rem;
      color: #718096;
      margin-top: 0.5rem;
      cursor: pointer;
      transition: all 0.2s;
    }

    .trend-image-placeholderSmall:hover,
    .trend-image-placeholderSmall:focus {
      background: #edf2f7;
      border-color: #667eea;
      color: #667eea;
    }

    .direction-analysis-box {
      margin-top: 1rem;
      padding: 1rem;
      border-radius: 10px;
      border: 1px solid #e2e8f0;
      background: #fcfcfc;
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
      background: #f1f5f9;
      color: #475569;
    }

    .no-direction-hint {
      margin-top: 1rem;
      padding: 1.5rem;
      border: 1px dashed #cbd5e0;
      border-radius: 10px;
      text-align: center;
      color: #94a3b8;
      font-size: 0.85rem;
      background: #f8fafc;
    }

    /* ???Ｙ???鈭箄???*/
    .timeframe-signals.expected {
      margin-top: 1.5rem;
      padding-top: 1rem;
      border-top: 1px dashed #e2e8f0;
    }

    .signal-chip.expected {
      border-style: dashed;
      background: #eff6ff;
      color: #1e40af;
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
      border: 1px solid #e2e8f0;
      padding: 0.75rem;
      border-radius: 8px;
      background: #ffffff;
    }

    .signal-name-label {
      display: block;
      font-size: 0.75rem;
      font-weight: 700;
      color: #475569;
      margin-bottom: 0.5rem;
    }

    .trend-image-placeholderExtraSmall {
      border: 1.5px dashed #cbd5e0;
      border-radius: 6px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.7rem;
      color: #94a3b8;
      cursor: pointer;
      transition: all 0.2s;
    }

    .trend-image-placeholderExtraSmall:hover,
    .trend-image-placeholderExtraSmall:focus {
      background: #f1f5f9;
      border-color: #3b82f6;
      color: #3b82f6;
    }
  </style>
{/if}
