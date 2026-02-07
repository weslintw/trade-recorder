<script>
  import { fade, scale } from 'svelte/transition';
  import { MARKET_SESSIONS } from '../lib/constants';

  export let show = false;
  export let plans = [];
  export let activeSession = 'asian'; // 當前編輯的時段，作為預設目標
  export let onConfirm = () => {};
  export let onClose = () => {};

  let selectedPlanId = null;
  let selectedSourceSession = null;
  let selectedTargetSession = activeSession;

  // 用來儲存解析後的當前選中規劃的 Session 資料
  let currentPlanSessions = [];

  // 當 plans 更新時，預設選中第一筆（最新的）
  $: if (plans && plans.length > 0 && !selectedPlanId) {
    selectedPlanId = plans[0].id;
  }

  // 當選中的 Plan 改變時，解析其內容以供選擇來源時段
  $: if (selectedPlanId) {
    const plan = plans.find(p => p.id === selectedPlanId);
    if (plan) {
      parsePlanSessions(plan);
    }
  }

  // 當 activeSession 改變時（例如外部傳入），同步更新 target
  $: if (activeSession) {
    selectedTargetSession = activeSession;
  }

  function parsePlanSessions(plan) {
    currentPlanSessions = [];
    if (!plan.trend_analysis) return;

    try {
      const analysis = JSON.parse(plan.trend_analysis);

      // 檢查是否為新格式 (包含 asian, european, us)
      const sessionKeys = ['asian', 'european', 'us'];
      let hasSessions = false;

      // 倒序檢查，這樣預設可以選中最後一個有資料的時段（通常是最相關的）
      // 但顯示時我們還是照時間順序
      const availableSessions = [];

      sessionKeys.forEach(key => {
        if (analysis[key]) {
          // 簡單檢查是否有內容 (不僅僅是空殼)
          const hasContent = checkSessionHasContent(analysis[key]);
          if (hasContent) {
            hasSessions = true;
            availableSessions.push({
              key: key,
              label: MARKET_SESSIONS.find(s => s.value === key)?.label || key,
              data: analysis[key],
            });
          }
        }
      });

      if (hasSessions) {
        currentPlanSessions = availableSessions;
        // 預設選中最後一個有資料的時段 (例如昨天最後是美盤，通常會想延續美盤的圖)
        if (availableSessions.length > 0) {
          selectedSourceSession = availableSessions[availableSessions.length - 1].key;
        }
      } else {
        // 舊格式或未知格式，整包當作一個來源
        currentPlanSessions = [{ key: 'all', label: '完整內容 (舊格式)', data: analysis }];
        selectedSourceSession = 'all';
      }
    } catch (e) {
      console.error('解析規劃失敗', e);
      currentPlanSessions = [];
    }
  }

  function checkSessionHasContent(sessionData) {
    if (!sessionData.trends) return false;
    // 檢查任意時區是否有方向、圖片、訊號、圖表配置或趨勢線
    return Object.values(sessionData.trends).some(
      t =>
        t.direction ||
        t.image ||
        t.has_signals ||
        t.has_wave ||
        t.notes ||
        t.chart_config ||
        (t.trendlines && t.trendlines.length > 0)
    );
  }

  function handleConfirm() {
    const plan = plans.find(p => p.id === selectedPlanId);
    if (plan && selectedSourceSession) {
      // 找出選中的 source content
      let sourceContent = null;
      if (selectedSourceSession === 'all') {
        // 舊格式：可能是直接的 trends 物件，或是不分 session 的結構
        // 我們需要父層做相容性處理，這裡直接傳回原始解析資料
        try {
          sourceContent = JSON.parse(plan.trend_analysis);
        } catch (e) {}
      } else {
        // 新格式：從解析出的陣列中拿
        const sessionObj = currentPlanSessions.find(s => s.key === selectedSourceSession);
        if (sessionObj) sourceContent = sessionObj.data; // 這裡的 data 應該是 { notes:..., trends:... }
      }

      onConfirm({
        plan: plan,
        sourceContent: sourceContent,
        targetSession: selectedTargetSession,
        sourceSessionKey: selectedSourceSession, // 傳回來源 key 方便除錯或標記
      });
      onClose();
    }
  }

  function formatDate(dateStr) {
    return new Date(dateStr).toLocaleDateString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={onClose} transition:fade={{ duration: 200 }}>
    <div class="modal-card" transition:scale={{ start: 0.95, duration: 200 }}>
      <div class="modal-header">
        <h3>📋 複製歷史規劃</h3>
        <button class="close-btn" on:click={onClose}>&times;</button>
      </div>

      <div class="modal-body">
        <!-- 步驟 1: 選擇規劃日期 -->
        <h4 class="step-title">1. 選擇來源日期</h4>
        <div class="plan-list">
          {#each plans as plan}
            <label class="plan-item" class:selected={selectedPlanId === plan.id}>
              <input type="radio" name="selectedPlan" value={plan.id} bind:group={selectedPlanId} />
              <div class="plan-info">
                <div class="plan-date">{formatDate(plan.plan_date)}</div>
                <div class="plan-summary">
                  {#if plan.symbol}
                    <span class="badge detail">{plan.symbol}</span>
                  {/if}
                </div>
              </div>
            </label>
          {/each}
        </div>

        {#if currentPlanSessions.length > 0}
          <div class="selection-row">
            <!-- 步驟 2: 選擇來源時段 -->
            <div class="selection-col">
              <h4 class="step-title">2. 選擇來源內容</h4>
              <div class="radio-group-vertical">
                {#each currentPlanSessions as sess}
                  <label class="radio-label">
                    <input type="radio" bind:group={selectedSourceSession} value={sess.key} />
                    {sess.label}
                  </label>
                {/each}
              </div>
            </div>

            <!-- 箭頭 -->
            <div class="arrow-col">➔ 複製到 ➔</div>

            <!-- 步驟 3: 選擇目標時段 -->
            <div class="selection-col">
              <h4 class="step-title">3. 選擇目標時段</h4>
              <div class="radio-group-vertical">
                {#each MARKET_SESSIONS as ms}
                  <label class="radio-label">
                    <input type="radio" bind:group={selectedTargetSession} value={ms.value} />
                    {ms.label} (今天)
                  </label>
                {/each}
              </div>
            </div>
          </div>
        {:else}
          <div class="empty-sessions">該規劃無有效內容可供複製。</div>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={onClose}>取消</button>
        <button
          class="btn btn-primary"
          on:click={handleConfirm}
          disabled={!selectedPlanId || !selectedSourceSession}
        >
          確認覆蓋
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }

  .modal-card {
    background: white;
    width: 90%;
    max-width: 600px;
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    max-height: 90vh;
  }

  .modal-header {
    padding: 1.2rem;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.25rem;
    color: #2d3748;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: #a0aec0;
    padding: 0;
    line-height: 1;
  }

  .close-btn:hover {
    color: #4a5568;
  }

  .modal-body {
    padding: 1.5rem;
    overflow-y: auto;
  }

  .step-title {
    font-size: 0.95rem;
    color: #4a5568;
    margin-bottom: 0.75rem;
    font-weight: 700;
  }

  .plan-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
    max-height: 150px;
    overflow-y: auto;
  }

  .plan-item {
    display: flex;
    align-items: center;
    padding: 0.75rem;
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .plan-item:hover {
    border-color: #cbd5e0;
    background: #f7fafc;
  }

  .plan-item.selected {
    border-color: #4299e1;
    background: #ebf8ff;
  }

  .plan-item input[type='radio'] {
    margin-right: 0.75rem;
  }

  .plan-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .plan-date {
    font-weight: 700;
    color: #2d3748;
  }

  .badge {
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    background: #edf2f7;
    color: #4a5568;
  }

  .badge.detail {
    background: #e6fffa;
    color: #2c7a7b;
  }

  /* 來源目標選擇區 */
  .selection-row {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #e2e8f0;
  }

  .selection-col {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .arrow-col {
    display: flex;
    align-items: center;
    justify-content: center;
    padding-top: 2rem;
    color: #a0aec0;
    font-weight: 700;
    font-size: 0.9rem;
  }

  .radio-group-vertical {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    background: #f7fafc;
    padding: 0.75rem;
    border-radius: 8px;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    color: #2d3748;
    cursor: pointer;
  }

  .radio-label input {
    cursor: pointer;
  }

  .empty-sessions {
    padding: 1rem;
    text-align: center;
    color: #e53e3e;
    background: #fff5f5;
    border-radius: 8px;
    font-size: 0.9rem;
  }

  .modal-footer {
    padding: 1.2rem;
    background: #f7fafc;
    border-top: 1px solid #e2e8f0;
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    border-bottom-left-radius: 12px;
    border-bottom-right-radius: 12px;
  }

  .btn {
    padding: 0.6rem 1.2rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: background 0.2s;
  }

  .btn-primary {
    background: #4299e1;
    color: white;
  }

  .btn-primary:hover {
    background: #3182ce;
  }

  .btn-primary:disabled {
    background: #a0aec0;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: #e2e8f0;
    color: #4a5568;
  }

  .btn-secondary:hover {
    background: #cbd5e0;
  }
</style>
