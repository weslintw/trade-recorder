<script>
  export let plan;
  export let openModal = () => {};

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

  const timeframes = ['M5', 'M15', 'H1', 'H4', 'D1'];
  const sessionConfig = {
    asian: { label: '亞盤', icon: '🇯🇵', color: '#3b82f6' },
    european: { label: '歐盤', icon: '🇬🇧', color: '#d97706' },
    us: { label: '美盤', icon: '🇺🇸', color: '#dc2626' }
  };

  let activeSession = 'asian'; // Default

  $: trendAnalysis = parseJSON(plan.trend_analysis, {});
  
  function hasContent(session) {
    if (!session) return false;
    if (session.notes && session.notes.trim()) return true;
    if (!session.trends) return false;
    return Object.values(session.trends).some(t => {
      if (t.directions && t.directions.length > 0) return true;
      if (t.direction) return true;
      if (t.image || t.signals_image || t.wave_image) return true;
      if (t.long && (t.long.has_signals || t.long.has_wave || t.long.signals_image || t.long.wave_image)) return true;
      if (t.short && (t.short.has_signals || t.short.has_wave || t.short.signals_image || t.short.wave_image)) return true;
      return (t.signals && t.signals.length > 0) || (t.wave_numbers && t.wave_numbers.length > 0);
    });
  }

  // Set initial active session based on available data
  let initialized = false;
  $: if (!initialized && Object.keys(trendAnalysis).length > 0) {
    const sessions = ['asian', 'european', 'us'];
    // Find the first session that has content
    const firstWithContent = sessions.find(s => hasContent(trendAnalysis[s]));
    if (firstWithContent) {
      activeSession = firstWithContent;
    }
    initialized = true;
  }

  $: currentSessionData = trendAnalysis[activeSession] || { trends: {} };

  function isWaveNumberHighlighted(trend, direction, number) {
    const target = direction ? trend[direction] : trend;
    if (!target) return false;
    return target.wave_highlight === number.toString() || target.wave_highlight === parseInt(number);
  }

  function isWaveNumberSelected(trend, direction, number) {
    const target = direction ? trend[direction] : trend;
    if (!target) return false;
    return (target.wave_numbers || []).some(n => n.toString() === number.toString());
  }
</script>

<div class="plan-detail-view card">
  <div class="view-header">
    <div class="title-section">
      <span class="symbol-tag">{plan.symbol || '---'}</span>
      <h1>盤面規劃分享</h1>
    </div>
    <div class="date-section">
      <span class="plan-date-tag">📅 {plan.plan_date ? plan.plan_date.slice(0, 10) : ''}</span>
    </div>
  </div>

  {#if plan.notes && plan.notes !== 'Session-based unified plan'}
    <div class="section-box">
      <h3>📝 總體規劃備註</h3>
      <div class="notes-content ql-editor">{@html lazyLoadHTML(plan.notes)}</div>
    </div>
  {/if}

  <!-- 市場時段分頁 (貼合編輯頁樣式) -->
  <div class="market-session-tabs-container">
    <div class="market-session-tabs">
      {#each ['asian', 'european', 'us'] as sessionKey}
        <button
          type="button"
          class="session-tab"
          class:active={activeSession === sessionKey}
          on:click={() => (activeSession = sessionKey)}
        >
          <span class="sess-icon">{sessionConfig[sessionKey].icon}</span>
          <span class="sess-label">{sessionConfig[sessionKey].label}</span>
        </button>
      {/each}
    </div>
  </div>

  <div class="session-content">
    {#if currentSessionData.notes}
      <div class="section-box session-notes">
        <h3>📝 {sessionConfig[activeSession].label} 時段備註</h3>
        <div class="notes-content ql-editor">{@html lazyLoadHTML(currentSessionData.notes)}</div>
      </div>
    {/if}

    <div class="trend-grid">
      {#each timeframes as tf}
        {@const trend = currentSessionData.trends?.[tf]}
        {#if trend && (trend.direction || trend.has_signals || trend.has_wave)}
          {@const directionsToShow = trend.directions && trend.directions.length > 0 ? trend.directions : (trend.direction && trend.direction !== 'both' ? [trend.direction] : [])}
          <div class="trend-item">
            <div class="timeframe-label">{tf}</div>

            <!-- 多空顯示 -->
            <div class="trend-options-view">
              <div class="trend-option-box long" class:active={trend.directions?.includes('long') || trend.direction === 'long' || trend.direction === 'both'}>
                多
              </div>
              <div class="trend-option-box short" class:active={trend.directions?.includes('short') || trend.direction === 'short' || trend.direction === 'both'}>
                空
              </div>
            </div>

            <!-- 分析區塊 (支持多個方向) -->
            {#each directionsToShow as dir}
              {@const analysis = trend[dir] || trend}
              <div class="analysis-box-container" class:long={dir === 'long'} class:short={dir === 'short'}>
                <div class="dir-header">{dir === 'long' ? '📈 多頭分析' : '📉 空頭分析'}</div>
                
                <!-- 達人訊號 -->
                {#if analysis.has_signals}
                  <div class="analysis-section">
                    <div class="section-title">✔️ 達人訊號</div>
                    <div class="signal-chips">
                      {#each (analysis.signals || []) as sig}
                        <span class="signal-chip active">{sig}</span>
                      {/each}
                    </div>
                    {#if analysis.signals_image}
                      <div class="trend-image-preview" on:click={() => openModal(analysis.signals_image, `${tf} ${dir === 'long' ? '多頭' : '空頭'} 訊號圖`)}>
                        <img src={analysis.signals_image} alt="signals" loading="lazy" />
                      </div>
                    {/if}
                  </div>
                {/if}

                <!-- 波浪浪數 -->
                {#if analysis.has_wave}
                  <div class="analysis-section">
                    <div class="section-title">✔️ 波浪浪數</div>
                    <div class="wave-numbers">
                      {#each ['1', '2', '3', '4', '5'] as num}
                        <span 
                          class="wave-number-box" 
                          class:selected={isWaveNumberSelected(trend, dir, num)}
                          class:highlighted={isWaveNumberHighlighted(trend, dir, num)}
                        >
                          {num}
                        </span>
                      {/each}
                    </div>
                    {#if analysis.wave_image}
                      <div class="trend-image-preview" on:click={() => openModal(analysis.wave_image, `${tf} ${dir === 'long' ? '多頭' : '空頭'} 波浪圖`)}>
                        <img src={analysis.wave_image} alt="wave" loading="lazy" />
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}

            <!-- 舊格式兼容: 如果沒有 directions 且沒有明確方向但有資料 -->
            {#if directionsToShow.length === 0 && (trend.has_signals || trend.has_wave)}
               <div class="analysis-box-container">
                  {#if trend.has_signals}
                    <div class="analysis-section">
                      <div class="section-title">✔️ 達人訊號</div>
                      <div class="signal-chips">
                        {#each (trend.signals || []) as sig}
                          <span class="signal-chip active">{sig}</span>
                        {/each}
                      </div>
                      {#if trend.signals_image}
                        <div class="trend-image-preview" on:click={() => openModal(trend.signals_image, `${tf} 訊號圖`)}>
                          <img src={trend.signals_image} alt="signals" loading="lazy" />
                        </div>
                      {/if}
                    </div>
                  {/if}

                  {#if trend.has_wave}
                    <div class="analysis-section">
                      <div class="section-title">✔️ 波浪浪數</div>
                      <div class="wave-numbers">
                        {#each ['1', '2', '3', '4', '5'] as num}
                          <span 
                            class="wave-number-box" 
                            class:selected={isWaveNumberSelected(trend, null, num)}
                            class:highlighted={isWaveNumberHighlighted(trend, null, num)}
                          >
                            {num}
                          </span>
                        {/each}
                      </div>
                      {#if trend.wave_image}
                        <div class="trend-image-preview" on:click={() => openModal(trend.wave_image, `${tf} 波浪圖`)}>
                          <img src={trend.wave_image} alt="wave" loading="lazy" />
                        </div>
                      {/if}
                    </div>
                  {/if}
               </div>
            {/if}

            <!-- 趨勢主圖 -->
            {#if trend.image}
              <div class="trend-main-image">
                 <div class="trend-image-preview" on:click={() => openModal(trend.image, `${tf} 趨勢圖`)}>
                    <img src={trend.image} alt="trend" loading="lazy" />
                  </div>
              </div>
            {/if}
          </div>
        {:else}
           <div class="trend-item empty">
             <div class="timeframe-label">{tf}</div>
             <p class="na-txt">無紀錄</p>
           </div>
        {/if}
      {/each}
    </div>
  </div>
</div>

<style>
  .card { background: white; border-radius: 1.5rem; padding: 2.5rem; box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05); border: 1px solid #f1f5f9; margin-bottom: 2rem; }
  .view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; border-bottom: 1px solid #f1f5f9; padding-bottom: 1.5rem; }
  .symbol-tag { display: inline-flex; align-items: center; justify-content: center; background: #4f46e5; color: white; padding: 0.25rem 0.75rem; border-radius: 6px; font-weight: 800; font-size: 0.875rem; line-height: 1; }
  .plan-date-tag { font-size: 0.9rem; color: #64748b; font-weight: 700; background: #f8fafc; padding: 0.4rem 1rem; border-radius: 99px; border: 1px solid #e2e8f0; }
  
  .section-box { margin-bottom: 2rem; }
  .section-box h3 { font-size: 1.1rem; font-weight: 800; color: #1e293b; margin-bottom: 1rem; }
  .notes-content { padding: 1.5rem; background: #f8fafc; border-radius: 1rem; line-height: 1.6; color: #334155; }

  /* Market Session Tabs */
  .market-session-tabs-container { margin-bottom: 2rem; }
  .market-session-tabs {
    display: flex;
    background: #f1f5f9;
    padding: 0.4rem;
    border-radius: 14px;
    gap: 0.4rem;
  }
  .session-tab {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.75rem;
    border: none;
    border-radius: 10px;
    background: transparent;
    color: #64748b;
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }
  .session-tab.active {
    background: white;
    color: #4f46e5;
    box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);
  }
  .sess-icon { font-size: 1.2rem; }

  /* Trend Grid - Matching Editor */
  .trend-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.5rem;
  }
  .trend-item {
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 16px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .trend-item.empty { opacity: 0.5; min-height: 100px; justify-content: center; align-items: center; background: #f8fafc; }
  .timeframe-label {
    font-size: 1.1rem;
    font-weight: 800;
    color: #1e293b;
    border-bottom: 2px solid #f1f5f9;
    padding-bottom: 0.5rem;
  }

  /* Directions */
  .trend-options-view {
    display: flex;
    gap: 0.5rem;
  }
  .trend-option-box {
    flex: 1;
    padding: 0.6rem;
    text-align: center;
    border-radius: 8px;
    font-weight: 800;
    border: 2px solid #e2e8f0;
    color: #cbd5e1;
  }
  .trend-option-box.long.active {
    background: #fef2f2;
    color: #dc2626;
    border-color: #fee2e2;
  }
  .trend-option-box.short.active {
    background: #f0fdf4;
    color: #16a34a;
    border-color: #dcfce7;
  }

  /* Analysis Sections */
  .analysis-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .section-title {
    font-size: 0.85rem;
    font-weight: 800;
    color: #3b82f6;
  }
  .signal-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .signal-chip {
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 700;
    background: white;
    border: 1px solid #e2e8f0;
    color: #64748b;
  }
  .signal-chip.active {
      border-color: #6366f1;
      background: #eef2ff;
      color: #4338ca;
  }

  .wave-numbers {
    display: flex;
    gap: 0.4rem;
  }
  .wave-number-box {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 0.9rem;
    font-weight: 800;
    color: #cbd5e1;
  }
  .wave-number-box.selected {
      background: #f0fdf4;
      color: #166534;
      border-color: #dcfce7;
  }
  .wave-number-box.highlighted {
      background: #fef2f2;
      color: #dc2626;
      border-color: #fee2e2;
  }

  /* Images */
  .trend-image-preview {
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    cursor: pointer;
    background: #f8fafc;
    position: relative;
    box-shadow: 0 4px 12px rgba(0,0,0,0.05);
    transition: transform 0.2s;
  }
  .trend-image-preview:hover { transform: scale(1.02); }
  .trend-image-preview img { width: 100%; height: auto; display: block; }
  
  .analysis-box-container {
    padding: 1rem;
    border-radius: 12px;
    background: #fcfcfc;
    border: 1px solid #f1f5f9;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .analysis-box-container.long { border-left: 4px solid #ef4444; }
  .analysis-box-container.short { border-left: 4px solid #10b981; }

  .dir-header {
    font-size: 0.8rem;
    font-weight: 800;
    color: #475569;
    padding: 2px 8px;
    background: #f1f5f9;
    border-radius: 4px;
    display: inline-block;
    align-self: flex-start;
  }

  .na-txt { color: #94a3b8; font-style: italic; font-size: 0.9rem; }

  @media (max-width: 640px) {
    .trend-grid { grid-template-columns: 1fr; }
    .market-session-tabs { flex-direction: column; }
  }
</style>
