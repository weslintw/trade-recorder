<script>
  import { onMount } from 'svelte';
  import { adminAPI } from '../lib/api';
  import { auth } from '../lib/auth';
  import { navigate } from 'svelte-routing';

  let userUsage = [];
  let loading = true;
  let error = null;

  onMount(async () => {
    if (!$auth.user || !$auth.user.is_admin) {
      alert('權限不足');
      navigate('/');
      return;
    }

    try {
      const res = await adminAPI.getUsage();
      userUsage = res.data;
    } catch (e) {
      console.error('Fetch usage error:', e);
      error = e.response?.data?.error || '無法取得資料';
    } finally {
      loading = false;
    }
  });

  function formatDate(isoStr) {
    if (!isoStr) return '-';
    return new Date(isoStr).toLocaleString('zh-TW', { hour12: false });
  }

  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
</script>

<div class="admin-dashboard">
  <h2>👥 系統使用狀況 (Admin)</h2>

  {#if loading}
    <div class="loading">載入中...</div>
  {:else if error}
    <div class="error-msg">{error}</div>
  {:else}
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>使用者</th>
            <th>帳號名稱</th>
            <th>空間占用</th>
            <th>總交易數</th>
            <th>最後交易時間</th>
            <th>總規劃數</th>
            <th>最後規劃日期</th>
          </tr>
        </thead>
        <tbody>
          {#each userUsage as user}
            {#if user.accounts && user.accounts.length > 0}
              {#each user.accounts as acc, i}
                <tr class:user-row-start={i === 0}>
                  {#if i === 0}
                    <td rowspan={user.accounts.length} class="user-cell">
                      {user.username} 
                      {#if user.is_admin}<span class="admin-tag">ADMIN</span>{/if}
                      <div class="user-id">ID: {user.user_id}</div>
                    </td>
                  {/if}
                  <td>{acc.account_name || '-'} <span class="acc-id">#{acc.account_id}</span></td>
                  <td class="num-cell">{formatBytes(acc.storage_usage || 0)}</td>
                  <td class="num-cell">{acc.trade_count}</td>
                  <td>{formatDate(acc.last_trade_time)}</td>
                  <td class="num-cell">{acc.plan_count}</td>
                  <td>{formatDate(acc.last_plan_date)}</td>
                </tr>
              {/each}
            {:else}
              <tr class="user-row-start">
                <td class="user-cell">
                  {user.username}
                  {#if user.is_admin}<span class="admin-tag">ADMIN</span>{/if}
                  <div class="user-id">ID: {user.user_id}</div>
                </td>
                <td colspan="6" class="no-data">尚無帳號資料</td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .admin-dashboard {
    padding: 2rem;
    max-width: 1400px;
    margin: 0 auto;
  }

  h2 {
    margin-bottom: 1.5rem;
    color: #1e293b;
  }

  .loading, .error-msg {
    text-align: center;
    padding: 2rem;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .error-msg {
    color: #ef4444;
  }

  .table-container {
    background: white;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);
    overflow-x: auto;
    border: 1px solid #e2e8f0;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    min-width: 800px;
  }

  th {
    background: #f8fafc;
    color: #475569;
    font-weight: 700;
    text-align: left;
    padding: 1rem;
    border-bottom: 2px solid #e2e8f0;
    white-space: nowrap;
  }

  td {
    padding: 1rem;
    border-bottom: 1px solid #f1f5f9;
    color: #334155;
    vertical-align: top;
  }

  .user-cell {
    background: #fcfcfd;
    border-right: 1px solid #e2e8f0;
    font-weight: 600;
  }

  .user-id {
    font-size: 0.75rem;
    color: #94a3b8;
    margin-top: 4px;
  }

  .acc-id {
    color: #94a3b8;
    font-size: 0.8rem;
    margin-left: 4px;
  }

  .admin-tag {
    background: #0f172a;
    color: white;
    font-size: 0.65rem;
    padding: 2px 6px;
    border-radius: 4px;
    margin-left: 6px;
  }

  .num-cell {
    font-family: 'JetBrains Mono', monospace;
    font-weight: 700;
  }

  .no-data {
    color: #94a3b8;
    text-align: center;
    font-style: italic;
  }

  .user-row-start td:not(.user-cell) {
     /* Visual hint for grouping possibly */
  }
  
  tbody tr:last-child td {
    border-bottom: none;
  }
</style>
