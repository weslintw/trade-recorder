<script>
  import { navigate } from 'svelte-routing';
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { accounts, selectedAccountId } from '../lib/stores';
  import AccountModal from './AccountModal.svelte';
  import SyncOptionsModal from './SyncOptionsModal.svelte';

  let loading = true;
  let showAddModal = false;
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

  onMount(() => {
    fetchAccounts();
    // 安全超時：如果 API 響應太慢，6 秒後強制關閉載入動畫
    setTimeout(() => {
      if (loading) {
        console.warn('[AccountManagement] Safety timeout triggered (6s). Forcing spinner OFF.');
        loading = false;
      }
    }, 6000);
  });

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

  async function clearAccountData(id) {
    if (
      !confirm(
        '🚨 警告：確定要清除此帳號的所有交易紀錄與規劃嗎？\n此動作將刪除所有數據且無法撤回！'
      )
    )
      return;
    try {
      await accountsAPI.clearData(id);
      alert('帳號資料已清除成功');
      fetchAccounts();
    } catch (e) {
      console.error(e);
      alert('清除資料失敗');
    }
  }


  let showSyncOptionsModal = false;
  let syncingId = null;

  function openSyncModal(id) {
    syncingId = id;
    showSyncOptionsModal = true;
  }

  async function syncAccount(id, options = {}) {
    try {
      showSyncOptionsModal = false;
      await accountsAPI.sync(id, options);
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
  let importSource = 'ftmo';
  let importResult = null;

  function openImportModal(id) {
    importingAccountId = id;
    showImportModal = true;
    importFile = null;
    importSource = 'ftmo';
    importResult = null;
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
      formData.append('source', importSource);
      const res = await accountsAPI.importCSV(importingAccountId, formData);
      importResult = res.data;
      // alert(res.data.message); // 改用內嵌顯示
      // showImportModal = false; // 暫不關閉，顯示結果
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
  let editingOffset = 8;

  function startEditing(acc) {
    editingId = acc.id;
    editingName = acc.name;
    editingOffset = acc.timezone_offset;
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
      await accountsAPI.update(id, {
        name: editingName.trim(),
        timezone_offset: parseInt(editingOffset),
      });
      editingId = null;
      fetchAccounts();
    } catch (e) {
      console.error(e);
      alert('更新名稱失敗');
    }
  }
  // --- 彈窗編輯相關 ---
  let editingAccount = null;

  function openAddModal() {
    editingAccount = null;
    showAddModal = true;
  }

  function openEditModal(acc) {
    editingAccount = acc;
    showAddModal = true;
  }

  // --- 選取帳號相關 ---
  function selectAccount(id) {
    if (editingId) return;
    selectedAccountId.set(id);
    navigate('/');
  }

  function formatBytes(bytes, decimals = 2) {
    if (!bytes || bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }
</script>

<div class="account-mgmt">
  <div class="header">
    <h1 data-testid="accounts-header">交易帳號管理</h1>
    <button class="btn btn-primary" data-testid="add-account-btn" on:click={openAddModal}
      >+ 新增交易帳號</button
    >
  </div>

  {#if loading}
    <div class="loading-overlay">
      <div class="loader"></div>
      <p>正在載入帳號資料...</p>
    </div>
  {:else}
    <div class="account-grid">
      {#each $accounts as acc}
        <div
          class="account-card card"
          class:ctrader={acc.type === 'ctrader'}
          on:click={() => selectAccount(acc.id)}
          role="button"
          tabindex="0"
          on:keydown={e => (e.key === 'Enter' || e.key === ' ') && selectAccount(acc.id)}
        >
            <button
              class="delete-acc-btn"
              on:click|stopPropagation={() => deleteAccount(acc.id)}
              title="刪除帳號"
            >
              ✕
            </button>
          <div class="acc-info">
            {#if editingId === acc.id}
              <div class="edit-name-wrapper" on:click|stopPropagation role="presentation">
                <input
                  type="text"
                  class="form-control edit-name-input"
                  bind:value={editingName}
                  on:keypress={e => e.key === 'Enter' && saveName(acc.id)}
                />
                <select class="form-control edit-offset-select" bind:value={editingOffset}>
                  {#each Array.from({ length: 25 }, (_, i) => i - 12) as offset}
                    <option value={offset}>UTC{offset >= 0 ? '+' : ''}{offset}</option>
                  {/each}
                </select>
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
                <button
                  class="btn-edit-small"
                  on:click|stopPropagation={() => openEditModal(acc)}
                  title="帳號設定">⚙️</button
                >
              </div>
            {/if}
            <div class="badges">
              <span
                class="badge {acc.type === 'local'
                  ? 'badge-info'
                  : acc.type === 'metatrader'
                    ? 'badge-mt5'
                    : acc.type === 'myfxbook'
                      ? 'badge-mt5'
                      : 'badge-ctrader'}"
              >
                {acc.type === 'local'
                  ? '本地帳號'
                  : acc.type === 'myfxbook'
                    ? 'Myfxbook'
                    : 'cTrader'}
              </span>
              <span class="badge {acc.status === 'active' ? 'badge-success' : 'badge-danger'}">
                {acc.status}
              </span>
              <span class="badge badge-utc"
                >UTC{acc.timezone_offset >= 0 ? '+' : ''}{acc.timezone_offset}</span
              >
            </div>
            <div class="storage-usage-info">
              <span class="icon">📊</span> 圖文佔用：<strong
                >{formatBytes(acc.storage_usage)}</strong
              >
            </div>
            {#if acc.type === 'myfxbook'}
              <div class="mt5-detail">
                <p>Myfxbook Email: {acc.myfxbook_email}</p>
                <div class="sync-info">
                  <span class="badge sync-badge {acc.sync_status} {
                    acc.sync_status?.toLowerCase().includes('syncing') ||
                    acc.sync_status?.toLowerCase().includes('fetching') ||
                    acc.sync_status?.toLowerCase().includes('scanning')
                    ? 'syncing'
                    : ''}">{acc.sync_status}</span>
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
            {#if acc.type === 'ctrader'}
              <div class="ctrader-detail">
                <p>Login ID: {acc.ctrader_account_id}</p>
                <div class="sync-info">
                  <span class="badge sync-badge {acc.sync_status} {
                    acc.sync_status?.toLowerCase().includes('syncing') ||
                    acc.sync_status?.toLowerCase().includes('fetching') ||
                    acc.sync_status?.toLowerCase().includes('scanning')
                    ? 'syncing'
                    : ''}">{acc.sync_status}</span>
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
              data-testid="import-csv-btn"
              on:click|stopPropagation={() => openImportModal(acc.id)}>📤 匯入 CSV</button
            >
            {#if acc.type === 'ctrader' || acc.type === 'myfxbook'}
              <button class="btn btn-sync" on:click|stopPropagation={() => acc.type === 'ctrader' ? openSyncModal(acc.id) : syncAccount(acc.id)}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 4px;">
                  <path d="M21 2v6h-6"></path>
                  <path d="M3 12a9 9 0 0 1 15-6.7L21 8"></path>
                  <path d="M3 22v-6h6"></path>
                  <path d="M21 12a9 9 0 0 1-15 6.7L3 16"></path>
                </svg>
                同步
              </button>
            {/if}
            <button
              class="btn btn-warning"
              data-testid="clear-data-btn"
              on:click|stopPropagation={() => clearAccountData(acc.id)}
              title="清除所有交易與規劃">🧹 清除資料</button
            >
          </div>
        </div>
      {/each}
    </div>
  {/if}
  {#if showAddModal}
    <AccountModal
      show={showAddModal}
      account={editingAccount}
      on:close={() => (showAddModal = false)}
      on:success={() => {
        showAddModal = false;
        fetchAccounts();
      }}
    />
  {/if}
  {#if showImportModal}
    <div class="modal-overlay" on:click|self={() => (showImportModal = false)} role="presentation">
      <div class="modal card">
        <h2>匯入交易紀錄 (CSV)</h2>
        <div class="form-group">
          <label for="importSource">匯入來源</label>
          <select id="importSource" class="form-control" bind:value={importSource}>
            <option value="ftmo">FTMO</option>
            <option value="myfxbook">Myfxbook</option>
          </select>
        </div>

        <div class="import-instructions">
          {#if importSource === 'ftmo'}
            <p>目前支援格式：<strong>FTMO CSV</strong></p>
            <p class="help-text">請從 FTMO 交易控制面板下載完整交易紀錄 CSV。</p>
          {:else if importSource === 'myfxbook'}
            <p>目前支援格式：<strong>Myfxbook CSV</strong></p>
            <p class="help-text">請從 Myfxbook 交易歷史頁面匯出 CSV。</p>
          {/if}
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
          {#if importResult}
            <button class="btn btn-primary" on:click={() => (showImportModal = false)}>完成</button>
          {:else}
            <button class="btn" on:click={() => (showImportModal = false)} disabled={importing}
              >取消</button
            >
            <button class="btn btn-primary" on:click={handleImportCSV} disabled={importing}>
              {importing ? '⌛ 處理中...' : '開始匯入'}
            </button>
          {/if}
        </div>

        {#if importResult}
          <div class="import-result-details">
            <div class="summary-banner">
              {importResult.message}
            </div>

            {#if importResult.imported_tickets?.length > 0}
              <div class="ticket-section imported">
                <h4>🟢 新匯入 ({importResult.imported_count})</h4>
                <div class="ticket-list">
                  {importResult.imported_tickets.join(', ')}
                </div>
              </div>
            {/if}

            {#if importResult.duplicate_tickets?.length > 0}
              <div class="ticket-section duplicate">
                <h4>🟡 重複跳過 ({importResult.duplicate_count})</h4>
                <div class="ticket-list">
                  {importResult.duplicate_tickets.join(', ')}
                </div>
              </div>
            {/if}

            {#if importResult.error_tickets?.length > 0}
              <div class="ticket-section error">
                <h4>🔴 匯入失敗 ({importResult.error_count})</h4>
                <div class="ticket-list">
                  {importResult.error_tickets.join(', ')}
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  {/if}
  {#if showSyncOptionsModal}
    <SyncOptionsModal
      show={showSyncOptionsModal}
      on:close={() => (showSyncOptionsModal = false)}
      on:sync={(e) => syncAccount(syncingId, e.detail)}
    />
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
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
    align-items: stretch;
  }

  .account-card {
    flex: 1 1 320px;
    max-width: calc(50% - 0.75rem);
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 200px;
    cursor: pointer;
    transition: all 0.2s ease;
    border: 2px solid transparent;
    position: relative;
    box-sizing: border-box;
  }

  /* 行動端全寬 */
  @media (max-width: 768px) {
    .account-card {
      max-width: 100%;
      flex-basis: 100%;
    }
  }

  .account-card.ctrader {
    border-left: 4px solid #10b981;
  }
  .account-card.mt5 {
    border-left: 4px solid #6366f1;
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

  .badge-ctrader {
    background: #ecfdf5;
    color: #059669;
  }

  .badge-utc {
    background: #f3f4f6;
    color: #4b5563;
    border: 1px solid #e5e7eb;
  }

  .mt5-detail,
  .ctrader-detail {
    font-size: 0.85rem;
    color: #64748b;
  }

  .acc-actions {
    display: flex;
    justify-content: flex-start;
    gap: 0.75rem;
    margin-top: 1.5rem;
  }

  .btn-sync {
    background: #f1f5f9;
    color: #475569;
  }

  .storage-usage-info {
    margin-top: 0.75rem;
    font-size: 0.85rem;
    color: #64748b;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    background: #f8fafc;
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    width: fit-content;
    border: 1px solid #f1f5f9;
  }

  .storage-usage-info strong {
    color: #4338ca;
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
    max-height: 90vh;
    overflow-y: auto;
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
    flex: 1;
    min-width: 120px;
  }

  .edit-offset-select {
    width: 90px !important;
    padding: 2px 4px !important;
    font-size: 0.85rem !important;
    height: auto !important;
  }

  .btn-icon {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 1.1rem;
    padding: 0;
    line-height: 1;
  }

  /* 刪除帳號叉叉按鈕 */
  .delete-acc-btn {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    width: 28px;
    height: 28px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    border-radius: 50%;
    font-size: 1.1rem;
    font-weight: bold;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    opacity: 0.4;
    z-index: 5;
  }

  .account-card:hover .delete-acc-btn {
    opacity: 1;
  }

  .delete-acc-btn:hover {
    background: #fee2e2;
    color: #ef4444;
    transform: rotate(90deg);
  }
  /* 匯入結果詳情樣式 */
  .import-result-details {
    margin-top: 1.5rem;
    padding-top: 1.5rem;
    border-top: 1px solid #e2e8f0;
    max-height: 300px;
    overflow-y: auto;
  }

  .summary-banner {
    padding: 0.75rem;
    background: #f1f5f9;
    border-radius: 8px;
    font-weight: 700;
    margin-bottom: 1.25rem;
    text-align: center;
    color: var(--text-color);
  }

  .ticket-section {
    margin-bottom: 1rem;
  }

  .ticket-section h4 {
    margin-bottom: 0.4rem;
    font-size: 0.9rem;
  }

  .ticket-list {
    font-family: monospace;
    font-size: 0.8rem;
    padding: 0.6rem;
    background: #f8fafc;
    border-radius: 6px;
    color: #64748b;
    word-break: break-all;
    line-height: 1.4;
  }

  .ticket-section.imported h4 {
    color: #059669;
  }
  .ticket-section.duplicate h4 {
    color: #d97706;
  }
  .ticket-section.error h4 {
    color: #dc2626;
  }
</style>
