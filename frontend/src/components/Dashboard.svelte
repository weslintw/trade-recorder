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
  let equityCurve = [];
  let loading = true;

  $: if ($selectedAccountId) {
    loadStats();
  }

  async function loadStats() {
    try {
      loading = true;

      const params = { account_id: $selectedAccountId };

      // 載入統計摘要
      const summaryResponse = await statsAPI.getSummary(params);
      summary = summaryResponse.data;

      // 載入品種統計
      const symbolResponse = await statsAPI.getBySymbol(params);
      symbolStats = symbolResponse.data;

      // 載入淨值曲線
      const equityResponse = await statsAPI.getEquityCurve(params);
      equityCurve = equityResponse.data;
    } catch (error) {
      console.error('載入統計資料失敗:', error);
      alert('載入統計資料失敗');
    } finally {
      loading = false;
    }
  }
</script>

<div class="dashboard">
  <h2>📈 交易統計儀表板</h2>

  {#if loading}
    <div class="loading">載入中...</div>
  {:else}
    <!-- 統計卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">📊</div>
        <div class="stat-content">
          <div class="stat-label">總交易數</div>
          <div class="stat-value">{summary.total_trades}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon success">✅</div>
        <div class="stat-content">
          <div class="stat-label">勝場數</div>
          <div class="stat-value success">{summary.winning_trades}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon danger">❌</div>
        <div class="stat-content">
          <div class="stat-label">敗場數</div>
          <div class="stat-value danger">{summary.losing_trades}</div>
        </div>
      </div>

      <div class="stat-card highlight">
        <div class="stat-icon">🎯</div>
        <div class="stat-content">
          <div class="stat-label">勝率</div>
          <div class="stat-value">{summary.win_rate.toFixed(2)}%</div>
        </div>
      </div>

      <div class="stat-card {summary.total_pnl >= 0 ? 'success-bg' : 'danger-bg'}">
        <div class="stat-icon">💰</div>
        <div class="stat-content">
          <div class="stat-label">總盈虧</div>
          <div class="stat-value">
            {summary.total_pnl >= 0 ? '+' : ''}{summary.total_pnl.toFixed(2)}
          </div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">📈</div>
        <div class="stat-content">
          <div class="stat-label">平均盈虧</div>
          <div class="stat-value">{summary.average_pnl.toFixed(2)}</div>
        </div>
      </div>

      <div class="stat-card success-bg">
        <div class="stat-icon">🏆</div>
        <div class="stat-content">
          <div class="stat-label">最大盈利</div>
          <div class="stat-value">+{summary.largest_win.toFixed(2)}</div>
        </div>
      </div>

      <div class="stat-card danger-bg">
        <div class="stat-icon">⚠️</div>
        <div class="stat-content">
          <div class="stat-label">最大虧損</div>
          <div class="stat-value">{summary.largest_loss.toFixed(2)}</div>
        </div>
      </div>

      <div class="stat-card highlight">
        <div class="stat-icon">⚖️</div>
        <div class="stat-content">
          <div class="stat-label">盈虧比</div>
          <div class="stat-value">{summary.profit_factor.toFixed(2)}</div>
        </div>
      </div>
    </div>

    <!-- 淨值曲線圖 -->
    {#if equityCurve.length > 0}
      <div class="card chart-card">
        <h3>📉 淨值曲線</h3>
        <EquityChart data={equityCurve} />
      </div>
    {/if}

    <!-- 品種統計表 -->
    {#if symbolStats.length > 0}
      <div class="card">
        <h3>🎲 各品種統計</h3>
        <div class="table-container">
          <table>
            <thead>
              <tr>
                <th>品種</th>
                <th>交易數</th>
                <th>勝場數</th>
                <th>勝率</th>
                <th>總盈虧</th>
              </tr>
            </thead>
            <tbody>
              {#each symbolStats as stat}
                <tr>
                  <td class="symbol">{stat.symbol}</td>
                  <td>{stat.total_trades}</td>
                  <td class="success">{stat.winning_trades}</td>
                  <td>
                    <span class="badge {stat.win_rate >= 50 ? 'badge-success' : 'badge-danger'}">
                      {stat.win_rate.toFixed(1)}%
                    </span>
                  </td>
                  <td class={stat.total_pnl >= 0 ? 'profit' : 'loss'}>
                    {stat.total_pnl >= 0 ? '+' : ''}{stat.total_pnl.toFixed(2)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}

    {#if summary.total_trades === 0}
      <div class="empty">
        <p>📭 尚無交易資料</p>
        <p>開始記錄交易後，統計資料將顯示在這裡</p>
      </div>
    {/if}
  {/if}
</div>

<style>
  .dashboard {
    max-width: 1400px;
  }

  h2 {
    color: white;
    text-align: center;
    margin-bottom: 2rem;
    font-size: 2rem;
    text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }

  h3 {
    margin-bottom: 1.5rem;
    color: #2d3748;
    font-size: 1.25rem;
  }

  .loading {
    text-align: center;
    padding: 3rem;
    color: white;
    font-size: 1.5rem;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .stat-card {
    background: white;
    border-radius: 16px;
    padding: 1.5rem;
    display: flex;
    align-items: center;
    gap: 1rem;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
    transition:
      transform 0.3s ease,
      box-shadow 0.3s ease;
  }

  .stat-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  }

  .stat-card.highlight {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
  }

  .stat-card.success-bg {
    background: linear-gradient(135deg, #48bb78 0%, #38a169 100%);
    color: white;
  }

  .stat-card.danger-bg {
    background: linear-gradient(135deg, #f56565 0%, #e53e3e 100%);
    color: white;
  }

  .stat-icon {
    font-size: 2.5rem;
    opacity: 0.9;
  }

  .stat-icon.success {
    filter: drop-shadow(0 2px 4px rgba(72, 187, 120, 0.4));
  }

  .stat-icon.danger {
    filter: drop-shadow(0 2px 4px rgba(245, 101, 101, 0.4));
  }

  .stat-content {
    flex: 1;
  }

  .stat-label {
    font-size: 0.875rem;
    opacity: 0.8;
    margin-bottom: 0.25rem;
    font-weight: 500;
  }

  .stat-value {
    font-size: 1.75rem;
    font-weight: 700;
    line-height: 1;
  }

  .stat-value.success {
    color: #38a169;
  }

  .stat-value.danger {
    color: #e53e3e;
  }

  .chart-card {
    margin-bottom: 2rem;
  }

  .table-container {
    overflow-x: auto;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  thead {
    background: #f7fafc;
  }

  th,
  td {
    padding: 1rem;
    text-align: left;
    border-bottom: 1px solid #e2e8f0;
  }

  th {
    font-weight: 600;
    color: #4a5568;
    font-size: 0.875rem;
    text-transform: uppercase;
  }

  td.symbol {
    font-weight: 700;
    color: #667eea;
  }

  td.success {
    color: #38a169;
    font-weight: 600;
  }

  td.profit {
    color: #38a169;
    font-weight: 700;
  }

  td.loss {
    color: #e53e3e;
    font-weight: 700;
  }

  tbody tr:hover {
    background: #f7fafc;
  }

  .empty {
    text-align: center;
    padding: 3rem;
    color: white;
  }

  .empty p {
    font-size: 1.25rem;
    margin-bottom: 0.5rem;
  }
</style>
