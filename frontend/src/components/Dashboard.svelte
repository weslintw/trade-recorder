<script>
  import { onMount } from 'svelte';
  import { statsAPI } from '../lib/api';
  import { selectedAccountId } from '../lib/stores';
  import EquityChart from './EquityChart.svelte';

  let summary = {
    total_trades: 0,
    winning_trades: 0,
    losing_trades: 0,
    win_rate: 0,
    total_pnl: 0,
    average_pnl: 0,
    largest_win: 0,
    largest_loss: 0,
    profit_factor: 0,
  };

  let symbolStats = [];
  let strategyStats = [];
  let colorStats = [];
  let equityCurve = [];
  let loading = true;

  $: if ($selectedAccountId) {
    loadStats();
  }

  async function loadStats() {
    try {
      loading = true;
      const params = { account_id: $selectedAccountId };

      const [summaryRes, symbolRes, strategyRes, equityRes, colorRes] = await Promise.all([
        statsAPI.getSummary(params).catch(e => ({ data: summary })),
        statsAPI.getBySymbol(params).catch(e => ({ data: [] })),
        statsAPI.getByStrategy(params).catch(e => ({ data: [] })),
        statsAPI.getEquityCurve(params).catch(e => ({ data: [] })),
        statsAPI.getByColorTag(params).catch(e => ({ data: [] })),
      ]);

      summary = summaryRes?.data || summary;
      symbolStats = symbolRes?.data || [];
      strategyStats = strategyRes?.data || [];
      equityCurve = equityRes?.data || [];
      colorStats = colorRes?.data || [];
    } catch (error) {
      console.error('載入統計資料失敗:', error);
    } finally {
      loading = false;
    }
  }

  function getStrategyName(strategy) {
    const map = {
      expert: '🏅 達人',
      elite: '💎 菁英',
      legend: '🔥 傳奇',
      unspecified: '⚪ 未指定',
    };
    return map[strategy] || strategy;
    return map[strategy] || strategy;
  }

  function getColorDescription(color) {
    const map = {
      green: '符合規則',
      yellow: '尚可接受',
      red: '衝動/不應進單',
    };
    return map[color] || color;
  }

  function getColorLabel(color) {
    const map = {
      green: '🟢 良好',
      yellow: '🟡 普通',
      red: '🔴 危險',
    };
    return map[color] || color;
  }

  function formatPnl(val) {
    if (val === undefined || val === null) return '0.00';
    return (parseFloat(val) || 0).toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  }

  function safeFixed(val, digits = 2) {
    if (val === undefined || val === null) return '0';
    return (parseFloat(val) || 0).toFixed(digits);
  }
</script>

<div class="dashboard-container">
  <header class="dashboard-header">
    <h1>📈 數據洞察儀表板</h1>
    <p class="subtitle">深度分析您的交易績效與行為樣態</p>
  </header>

  {#if loading}
    <div class="loading-overlay">
      <div class="loader"></div>
      <p>正在解析數據...</p>
    </div>
  {:else}
    <!-- 頂部核心指標 -->
    <div class="metrics-grid">
      <div class="metric-card glass">
        <span class="metric-icon">📊</span>
        <div class="metric-info">
          <span class="label">總交易數</span>
          <span class="value">{summary?.total_trades || 0}</span>
        </div>
      </div>

      <div class="metric-card glass success-glow">
        <span class="metric-icon">✅</span>
        <div class="metric-info">
          <span class="label">勝場</span>
          <span class="value text-success">{summary?.winning_trades || 0}</span>
        </div>
      </div>

      <div class="metric-card glass danger-glow">
        <span class="metric-icon">❌</span>
        <div class="metric-info">
          <span class="label">敗場</span>
          <span class="value text-danger">{summary?.losing_trades || 0}</span>
        </div>
      </div>

      <div class="metric-card glass primary-gradient">
        <span class="metric-icon">🎯</span>
        <div class="metric-info">
          <span class="label">勝率</span>
          <span class="value">{safeFixed(summary?.win_rate, 2)}%</span>
        </div>
      </div>

      <div
        class="metric-card glass {(summary?.total_pnl || 0) >= 0
          ? 'success-gradient'
          : 'danger-gradient'}"
      >
        <span class="metric-icon">💰</span>
        <div class="metric-info">
          <span class="label">總盈虧</span>
          <span class="value">
            {(summary?.total_pnl || 0) >= 0 ? '+' : ''}{formatPnl(summary?.total_pnl)}
          </span>
        </div>
      </div>

      <div class="metric-card glass">
        <span class="metric-icon">⚖️</span>
        <div class="metric-info">
          <span class="label">盈虧比</span>
          <span class="value">{safeFixed(summary?.profit_factor, 2)}</span>
        </div>
      </div>
    </div>

    <div class="dashboard-body">
      <!-- 左側欄位 -->
      <div class="main-column">
        <!-- 淨值曲線 -->
        {#if equityCurve && equityCurve.length > 0}
          <div class="chart-section glass-card">
            <div class="section-header">
              <h3>📉 淨值曲線 (Equity)</h3>
            </div>
            <EquityChart data={equityCurve} />
          </div>
        {/if}

        <!-- 品種統計 -->
        {#if symbolStats && symbolStats.length > 0}
          <div class="table-section glass-card">
            <div class="section-header">
              <h3>🎲 各品種績效</h3>
            </div>
            <div class="modern-table-wrapper">
              <table class="modern-table">
                <thead>
                  <tr>
                    <th>品種</th>
                    <th>次數</th>
                    <th>勝場</th>
                    <th>勝率</th>
                    <th>累積盈虧</th>
                  </tr>
                </thead>
                <tbody>
                  {#each symbolStats as stat}
                    <tr>
                      <td class="symbol-cell"><strong>{stat.symbol}</strong></td>
                      <td>{stat.total_trades}</td>
                      <td class="text-success">{stat.winning_trades}</td>
                      <td>
                        <div class="progress-bar-container">
                          <div
                            class="progress-bar {(stat.win_rate || 0) >= 50
                              ? 'bg-success'
                              : 'bg-danger'}"
                            style="width: {stat.win_rate || 0}%"
                          ></div>
                          <span class="progress-text">{safeFixed(stat.win_rate, 1)}%</span>
                        </div>
                      </td>
                      <td class={(stat.total_pnl || 0) >= 0 ? 'text-success' : 'text-danger'}>
                        {(stat.total_pnl || 0) >= 0 ? '+' : ''}{safeFixed(stat.total_pnl, 2)}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/if}
      </div>

      <!-- 右側欄位 -->
      <div class="sidebar-column">
        <div class="strategy-analysis-section glass-card">
          <div class="section-header">
            <h3>🎯 策略體系分析</h3>
          </div>

          {#if strategyStats && strategyStats.length > 0}
            {#each strategyStats as s}
              <div class="strategy-group">
                <div class="strategy-header-row">
                  <span class="strategy-badge {s.strategy}">{getStrategyName(s.strategy)}</span>
                  <span class="strategy-stats-summary">
                    {s.total_trades} 筆 | 勝率 {safeFixed(s.win_rate, 1)}% |
                    <strong class={(s.total_pnl || 0) >= 0 ? 'text-success' : 'text-danger'}
                      >{(s.total_pnl || 0) >= 0 ? '+' : ''}{safeFixed(s.total_pnl, 1)}</strong
                    >
                  </span>
                </div>

                <div class="sub-item-stats">
                  {#if s.sub_item_stats && s.sub_item_stats.length > 0}
                    {#each s.sub_item_stats as sub}
                      <div class="sub-item-row">
                        <div class="sub-item-name">{sub.name}</div>
                        <div class="sub-item-meta">
                          <span class="sub-count">{sub.total_trades} 筆</span>
                          <span class="sub-winrate">{safeFixed(sub.win_rate, 0)}% Win</span>
                          <span class="sub-pnl {(sub.total_pnl || 0) >= 0 ? 'pos' : 'neg'}">
                            {(sub.total_pnl || 0) >= 0 ? '+' : ''}{safeFixed(sub.total_pnl, 1)}
                          </span>
                        </div>
                      </div>
                    {/each}
                  {:else}
                    <!-- Only show if there are sub items to avoid clutter -->
                  {/if}
                </div>
              </div>
            {/each}
          {:else}
            <div class="empty-mini">尚無策略標籤紀錄</div>
          {/if}
        </div>

        <div class="color-analysis-section glass-card">
          <div class="section-header">
            <h3>🎨 執行品質分析</h3>
          </div>

          {#if colorStats && colorStats.length > 0}
            <div class="color-stats-list">
              {#each colorStats as cs}
                <div class="color-stat-row {cs.color}">
                  <div class="color-stat-header">
                    <div class="color-label-group">
                      <span class="color-dot bg-{cs.color}"></span>
                      <span class="color-name">{getColorLabel(cs.color)}</span>
                    </div>
                    <span class="color-desc">{getColorDescription(cs.color)}</span>
                  </div>
                  <div class="color-stat-metrics">
                    <div class="metric-mini">
                      <span class="label">次數</span>
                      <span class="value">{cs.total_trades}</span>
                    </div>
                    <div class="metric-mini">
                      <span class="label">勝率</span>
                      <span class="value">{safeFixed(cs.win_rate, 0)}%</span>
                    </div>
                    <div class="metric-mini">
                      <span class="label">盈虧</span>
                      <span
                        class="value {(cs.total_pnl || 0) >= 0 ? 'text-success' : 'text-danger'}"
                      >
                        {(cs.total_pnl || 0) >= 0 ? '+' : ''}{safeFixed(cs.total_pnl, 0)}
                      </span>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <div class="empty-mini">尚無顏色標記紀錄</div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  :root {
    --dashboard-bg: #f0f4f8;
    --glass-bg: rgba(255, 255, 255, 0.7);
    --glass-border: rgba(255, 255, 255, 0.4);
    --primary: #6366f1;
    --success: #10b981;
    --danger: #ef4444;
    --text-main: #1e293b;
    --text-muted: #64748b;
  }

  .dashboard-container {
    padding: 2rem;
    color: var(--text-main);
  }

  .dashboard-header {
    margin-bottom: 2.5rem;
    text-align: left;
  }

  .dashboard-header h1 {
    font-size: 2.25rem;
    font-weight: 800;
    margin-bottom: 0.5rem;
    background: linear-gradient(135deg, #4f46e5, #9333ea);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .subtitle {
    font-size: 1.1rem;
    color: var(--text-muted);
  }

  /* 頂部指標卡片 */
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1.25rem;
    margin-bottom: 2.5rem;
  }

  .metric-card {
    background: var(--glass-bg);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    border: 1px solid var(--glass-border);
    border-radius: 20px;
    padding: 1.25rem;
    display: flex;
    align-items: center;
    gap: 1rem;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.05);
    transition: transform 0.2s;
  }

  .metric-card:hover {
    transform: translateY(-5px);
  }

  .metric-icon {
    font-size: 2rem;
    width: 50px;
    height: 50px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: white;
    border-radius: 12px;
    box-shadow: 0 4px 10px rgba(0, 0, 0, 0.03);
  }

  .metric-info {
    display: flex;
    flex-direction: column;
  }

  .metric-info .label {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .metric-info .value {
    font-size: 1.5rem;
    font-weight: 800;
  }

  /* 特殊樣式卡片 */
  .primary-gradient {
    background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
    color: white;
  }
  .success-gradient {
    background: linear-gradient(135deg, #10b981 0%, #059669 100%);
    color: white;
  }
  .danger-gradient {
    background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
    color: white;
  }

  .primary-gradient .label,
  .success-gradient .label,
  .danger-gradient .label {
    color: rgba(255, 255, 255, 0.8);
  }

  /* 佈局主體 */
  .dashboard-body {
    display: grid;
    grid-template-columns: 2fr 1.2fr;
    gap: 2rem;
  }

  /* 響應式優化 */
  @media (max-width: 1200px) {
    .dashboard-body {
      grid-template-columns: 1fr;
    }
  }

  .glass-card {
    background: white;
    border-radius: 24px;
    padding: 2rem;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.04);
    border: 1px solid #edf2f7;
    margin-bottom: 2rem;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  .section-header h3 {
    font-size: 1.25rem;
    font-weight: 800;
    color: #1e293b;
    position: relative;
    padding-bottom: 0.5rem;
  }

  .section-header h3::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    width: 40px;
    height: 4px;
    background: var(--primary);
    border-radius: 2px;
  }

  /* 表格樣式 */
  .modern-table-wrapper {
    overflow-x: auto;
  }

  .modern-table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0 10px;
  }

  .modern-table th {
    padding: 1rem;
    text-align: left;
    color: var(--text-muted);
    font-weight: 600;
    font-size: 0.85rem;
    text-transform: uppercase;
  }

  .modern-table td {
    padding: 1.25rem 1rem;
    background: #f8fafc;
    border-top: 1px solid #f1f5f9;
    border-bottom: 1px solid #f1f5f9;
  }

  .modern-table td:first-child {
    border-left: 1px solid #f1f5f9;
    border-radius: 12px 0 0 12px;
  }
  .modern-table td:last-child {
    border-right: 1px solid #f1f5f9;
    border-radius: 0 12px 12px 0;
  }

  .symbol-cell strong {
    color: var(--primary);
  }

  /* 進度條勝率樣式 */
  .progress-bar-container {
    width: 120px;
    height: 8px;
    background: #e2e8f0;
    border-radius: 4px;
    position: relative;
    display: flex;
    align-items: center;
  }

  .progress-bar {
    height: 100%;
    border-radius: 4px;
  }

  .progress-text {
    position: absolute;
    left: 130px;
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--text-main);
  }

  /* 策略分析樣式 */
  .strategy-analysis-section {
    /* position: sticky; */
    /* top: 2rem; */
  }

  .strategy-group {
    margin-bottom: 1.5rem;
  }

  .strategy-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    background: #f8fafc;
    padding: 0.75rem 1rem;
    border-radius: 12px;
  }

  .strategy-badge {
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    font-weight: 800;
    font-size: 0.9rem;
    color: white;
  }

  .strategy-badge.expert {
    background: #059669;
  }
  .strategy-badge.elite {
    background: #1e3a8a;
  }
  .strategy-badge.legend {
    background: #78350f;
  }

  .strategy-stats-summary {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .sub-item-stats {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding-left: 0.5rem;
  }

  .sub-item-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.6rem 0;
    border-bottom: 1px dashed #e2e8f0;
  }

  .sub-item-name {
    font-weight: 600;
    font-size: 0.95rem;
    color: #334155;
  }

  .sub-item-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.85rem;
    align-items: center;
  }

  .sub-count {
    color: var(--text-muted);
  }
  .sub-winrate {
    font-weight: 700;
    color: var(--text-main);
  }
  .sub-pnl {
    font-weight: 800;
    padding: 2px 8px;
    border-radius: 4px;
    min-width: 60px;
    text-align: right;
  }
  .sub-pnl.pos {
    background: #dcfce7;
    color: #166534;
  }
  .sub-pnl.neg {
    background: #fee2e2;
    color: #991b1b;
  }

  /* 顏色分析樣式 */
  .color-analysis-section {
    margin-top: 2rem;
  }

  .color-stat-row {
    padding: 1rem;
    background: #f8fafc;
    border-radius: 12px;
    margin-bottom: 1rem;
    border-left: 4px solid transparent;
  }

  .color-stat-row.green {
    border-left-color: var(--success);
    background: #f0fdf4;
  }
  .color-stat-row.yellow {
    border-left-color: #fbbf24;
    background: #fffbeb;
  }
  .color-stat-row.red {
    border-left-color: var(--danger);
    background: #fef2f2;
  }

  .color-stat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.8rem;
  }

  .color-label-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .color-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }
  .bg-green {
    background-color: var(--success);
  }
  .bg-yellow {
    background-color: #fbbf24;
  }
  .bg-red {
    background-color: var(--danger);
  }

  .color-name {
    font-weight: 700;
    font-size: 0.95rem;
  }
  .color-desc {
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .color-stat-metrics {
    display: flex;
    justify-content: space-between;
    padding-top: 0.5rem;
    border-top: 1px dashed rgba(0, 0, 0, 0.05);
  }

  .metric-mini {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .metric-mini .label {
    font-size: 0.7rem;
    color: var(--text-muted);
    text-transform: uppercase;
  }
  .metric-mini .value {
    font-weight: 700;
    font-size: 0.95rem;
  }

  /* 輔助工具 */
  .text-success {
    color: var(--success);
  }
  .text-danger {
    color: var(--danger);
  }
  .bg-success {
    background: var(--success);
  }
  .bg-danger {
    background: var(--danger);
  }

  .empty-mini {
    padding: 1rem;
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    font-size: 0.9rem;
  }

  .loading-overlay {
    height: 400px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .loader {
    width: 48px;
    height: 48px;
    border: 5px solid var(--primary);
    border-bottom-color: transparent;
    border-radius: 50%;
    display: inline-block;
    box-sizing: border-box;
    animation: rotation 1s linear infinite;
    margin-bottom: 1.5rem;
  }

  @keyframes rotation {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }
</style>
