<script>
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { accounts } from '../lib/stores';

  let loading = true;
  let showAddModal = false;
  let newAccount = {
    name: '',
    type: 'local',
    mt5_account_id: '',
    mt5_token: '',
  };

  async function fetchAccounts() {
    try {
      const res = await accountsAPI.getAll();
      accounts.set(res.data);
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  onMount(fetchAccounts);

  async function addAccount() {
    // 前端驗證
    if (!newAccount.name.trim()) {
      alert('請輸入帳號名稱');
      return;
    }
    if (newAccount.type === 'metatrader') {
      if (!newAccount.mt5_account_id.trim() || !newAccount.mt5_token.trim()) {
        alert('請輸入 MetaApi Account ID 與 Token');
        return;
      }
    }

    try {
      await accountsAPI.create(newAccount);
      showAddModal = false;
      newAccount = { name: '', type: 'local', mt5_account_id: '', mt5_token: '' };
      fetchAccounts();
    } catch (e) {
      console.error(e);
      const errorMsg = e.response?.data?.error || e.message || '未知錯誤';
      alert('建立帳號失敗: ' + errorMsg);
    }
  }

  async function deleteAccount(id) {
    if (!confirm('確定要刪除此帳號嗎？相關的交易紀錄與規劃將會一併刪除！')) return;
    try {
      await accountsAPI.delete(id);
      fetchAccounts();
    } catch (e) {
      console.error(e);
      const errorMsg = e.response?.data?.error || e.message || '未知錯誤';
      alert('刪除帳號失敗: ' + errorMsg);
    }
  }

  let syncInterval;
  onMount(() => {
    fetchAccounts();
    syncInterval = setInterval(() => {
      // 如果有任何帳號正在同步中，就定時更新
      if ($accounts.some(a => a.sync_status === 'syncing')) {
        fetchAccounts();
      }
    }, 3000);
    return () => clearInterval(syncInterval);
  });

  async function syncAccount(id) {
    try {
      await accountsAPI.sync(id);
      fetchAccounts(); // 立即更新一次狀態
    } catch (e) {
      console.error(e);
      const errorMsg = e.response?.data?.error || e.message || '未知錯誤';
      alert('觸發同步失敗: ' + errorMsg);
    }
  }
</script>

<div class="account-mgmt">
  <div class="header">
    <h1>帳號管理</h1>
    <button class="btn btn-primary" on:click={() => (showAddModal = true)}>+ 新增帳號</button>
  </div>

  {#if loading}
    <p>載入中...</p>
  {:else}
    <div class="account-grid">
      {#each $accounts as acc}
        <div class="account-card card" class:mt5={acc.type === 'metatrader'}>
          <div class="acc-info">
            <h3>{acc.name}</h3>
            <div class="badges">
              <span class="badge {acc.type === 'local' ? 'badge-info' : 'badge-mt5'}">
                {acc.type === 'local' ? '本地帳號' : 'MetaTrader 5'}
              </span>
              <span class="badge {acc.status === 'active' ? 'badge-success' : 'badge-danger'}">
                {acc.status}
              </span>
            </div>
            {#if acc.type === 'metatrader'}
              <div class="mt5-detail">
                <p>ID: {acc.mt5_account_id}</p>
                <div class="sync-info">
                  <span class="badge sync-badge {acc.sync_status}">{acc.sync_status}</span>
                  {#if acc.last_synced_at}
                    <span class="sync-time"
                      >最後同步: {new Date(acc.last_synced_at).toLocaleString()}</span
                    >
                  {/if}
                </div>
                {#if acc.sync_status === 'failed' && acc.last_sync_error}
                  <div class="sync-error-msg">❌ {acc.last_sync_error}</div>
                {/if}
              </div>
            {/if}
          </div>
          <div class="acc-actions">
            {#if acc.type === 'metatrader'}
              <button class="btn btn-sync" on:click={() => syncAccount(acc.id)}>🔄 同步</button>
            {/if}
            {#if acc.id !== 1}
              <button class="btn btn-danger" on:click={() => deleteAccount(acc.id)}>刪除</button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if showAddModal}
    <div class="modal-overlay" on:click|self={() => (showAddModal = false)}>
      <div class="modal card">
        <h2>新增交易帳號</h2>
        <div class="form-group">
          <label>帳號名稱</label>
          <input
            type="text"
            class="form-control"
            bind:value={newAccount.name}
            placeholder="如：個人實盤"
          />
        </div>
        <div class="form-group">
          <label>帳號類型</label>
          <div class="type-selector">
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="local" /> 本地記錄 (完全手動)
            </label>
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="metatrader" /> MetaTrader 5 Cloud
              API
            </label>
          </div>
        </div>

        {#if newAccount.type === 'metatrader'}
          <div class="mt5-fields">
            <div class="form-group">
              <label>MetaApi Account ID</label>
              <input type="text" class="form-control" bind:value={newAccount.mt5_account_id} />
            </div>
            <div class="form-group">
              <label>MetaApi Token (API Key)</label>
              <input type="password" class="form-control" bind:value={newAccount.mt5_token} />
            </div>
            <p class="help-text">註：目前系統對接 MetaApi.cloud 服務以實現 MT5 雲端連線。</p>
          </div>
        {/if}

        <div class="modal-actions">
          <button class="btn" on:click={() => (showAddModal = false)}>取消</button>
          <button class="btn btn-primary" on:click={addAccount}>確認新增</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  .account-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
  }

  .account-card {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 180px;
  }

  .acc-info h3 {
    margin: 0 0 0.5rem 0;
  }

  .badges {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .badge-mt5 {
    background: #e0e7ff;
    color: #4338ca;
  }

  .mt5-detail {
    font-size: 0.85rem;
    color: #64748b;
  }

  .acc-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }

  .btn-sync {
    background: #f1f5f9;
    color: #475569;
  }

  .btn-sync:hover {
    background: #e2e8f0;
  }

  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    width: 100%;
    max-width: 500px;
    padding: 2rem;
  }

  .type-selector {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .radio-label {
    font-weight: normal !important;
    display: flex !important;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .mt5-fields {
    background: #f8fafc;
    padding: 1rem;
    border-radius: 8px;
    margin-top: 1rem;
  }

  .help-text {
    font-size: 0.75rem;
    color: #94a3b8;
    margin-top: 0.5rem;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
  }

  /* 同步狀態樣式 */
  .sync-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.5rem;
  }

  .sync-badge {
    text-transform: capitalize;
    font-size: 0.7rem;
  }

  .sync-badge.syncing {
    background: #fef1f2;
    color: #e11d48;
    animation: pulse 2s infinite;
  }

  .sync-badge.success {
    background: #f0fdf4;
    color: #16a34a;
  }

  .sync-badge.failed {
    background: #fff1f2;
    color: #be123c;
  }

  .sync-time {
    font-size: 0.75rem;
    color: #94a3b8;
  }

  .sync-error-msg {
    margin-top: 0.5rem;
    font-size: 0.75rem;
    color: #ef4444;
    background: #fef2f2;
    padding: 0.5rem;
    border-radius: 4px;
    border: 1px solid #fee2e2;
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
</style>
