<script>
  import { navigate } from 'svelte-routing';
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { accounts, selectedAccountId } from '../lib/stores';

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

  // --- CSV 匯入相關 ---
  let showImportModal = false;
  let importingAccountId = null;
  let importFile = null;
  let importing = false;

  function openImportModal(id) {
    importingAccountId = id;
    showImportModal = true;
    importFile = null;
  }

  async function handleImportCSV() {
    if (!importFile) {
      alert('請選擇 CSV 檔案');
      return;
    }
    importing = true;
    try {
      const formData = new FormData();
      formData.append('file', importFile);
      const res = await accountsAPI.importCSV(importingAccountId, formData);
      alert(res.data.message);
      showImportModal = false;
      importFile = null;
    } catch (e) {
      console.error(e);
      const errorMsg = e.response?.data?.error || e.message || '未知錯誤';
      alert('匯入失敗: ' + errorMsg);
    } finally {
      importing = false;
    }
  }

  // --- 帳號重新命名相關 ---
  let editingId = null;
  let editingName = '';

  function startEditing(acc) {
    editingId = acc.id;
    editingName = acc.name;
  }

  function cancelEditing() {
    editingId = null;
    editingName = '';
  }

  async function saveName(id) {
    if (!editingName.trim()) {
      alert('名稱不能為空');
      return;
    }
    try {
      await accountsAPI.update(id, { name: editingName.trim() });
      editingId = null;
      fetchAccounts();
    } catch (e) {
      console.error(e);
      alert('更新名稱失敗');
    }
  }

  // --- 選取帳號相關 ---
  function selectAccount(id) {
    if (editingId) return;
    selectedAccountId.set(id);
    navigate('/');
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
        <div
          class="account-card card"
          class:mt5={acc.type === 'metatrader'}
          on:click={() => selectAccount(acc.id)}
        >
          <div class="acc-info">
            {#if editingId === acc.id}
              <div class="edit-name-wrapper" on:click|stopPropagation>
                <input
                  type="text"
                  class="form-control edit-name-input"
                  bind:value={editingName}
                  on:keypress={e => e.key === 'Enter' && saveName(acc.id)}
                  autoFocus
                />
                <button class="btn-icon save" on:click={() => saveName(acc.id)} title="儲存"
                  >✅</button
                >
                <button class="btn-icon cancel" on:click={cancelEditing} title="取消">❌</button>
              </div>
            {:else}
              <div class="name-display">
                <h3>{acc.name}</h3>
                <button
                  class="btn-edit-small"
                  on:click|stopPropagation={() => startEditing(acc)}
                  title="重新命名">✏️</button
                >
              </div>
            {/if}
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
            <button
              class="btn btn-secondary"
              on:click|stopPropagation={() => openImportModal(acc.id)}>📤 匯入 CSV</button
            >
            {#if acc.type === 'metatrader'}
              <button class="btn btn-sync" on:click|stopPropagation={() => syncAccount(acc.id)}
                >🔄 同步</button
              >
            {/if}
            {#if acc.id !== 1}
              <button class="btn btn-danger" on:click|stopPropagation={() => deleteAccount(acc.id)}
                >刪除</button
              >
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
  {#if showImportModal}
    <div class="modal-overlay" on:click|self={() => (showImportModal = false)}>
      <div class="modal card">
        <h2>匯入交易紀錄 (CSV)</h2>
        <div class="import-instructions">
          <p>目前支援格式：<strong>FTMO CSV</strong></p>
          <p class="help-text">請從 FTMO 交易控制面板下載完整交易紀錄 CSV。</p>
        </div>

        <div class="form-group">
          <label for="csvFile">選擇檔案</label>
          <input
            type="file"
            id="csvFile"
            accept=".csv"
            class="form-control"
            on:change={e => (importFile = e.target.files[0])}
          />
        </div>

        <div class="modal-actions">
          <button class="btn" on:click={() => (showImportModal = false)} disabled={importing}
            >取消</button
          >
          <button class="btn btn-primary" on:click={handleImportCSV} disabled={importing}>
            {importing ? '⌛ 處理中...' : '開始匯入'}
          </button>
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
    cursor: pointer;
    transition: all 0.2s ease;
    border: 2px solid transparent;
  }

  .account-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
    border-color: #e2e8f0;
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

  .import-instructions {
    background: #f0fdf4;
    padding: 1rem;
    border-radius: 8px;
    margin-bottom: 1.5rem;
    border-left: 4px solid #16a34a;
  }

  .import-instructions p {
    margin: 0;
    font-size: 0.9rem;
    color: #166534;
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

  /* 重新命名樣式 */
  .name-display {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .btn-edit-small {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 0.9rem;
    padding: 2px;
    opacity: 0.3;
    transition: opacity 0.2s;
  }

  .account-card:hover .btn-edit-small {
    opacity: 1;
  }

  .edit-name-wrapper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .edit-name-input {
    margin: 0 !important;
    padding: 0.25rem 0.5rem !important;
    font-size: 1.1rem !important;
    font-weight: 600;
  }

  .btn-icon {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 1.1rem;
    padding: 0;
    line-height: 1;
  }
</style>
