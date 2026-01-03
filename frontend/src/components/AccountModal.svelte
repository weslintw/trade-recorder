<script>
  import { createEventDispatcher } from 'svelte';
  import { accountsAPI } from '../lib/api';

  export let show = false;
  export let account = null;

  const dispatch = createEventDispatcher();

  let newAccount = {
    name: '',
    type: 'local',
    mt5_account_id: '',
    mt5_token: '',
    ctrader_account_id: '',
    ctrader_token: '',
    ctrader_client_id: '',
    ctrader_client_secret: '',
    ctrader_env: 'live',
    timezone_offset: 8,
  };

  let lastAccountId = undefined;
  let lastShow = false;

  // Reactively update form when account changes OR modal opens
  $: if (show && (account?.id !== lastAccountId || show !== lastShow)) {
    lastAccountId = account?.id;
    lastShow = show;

    if (account) {
      newAccount = {
        name: account.name || '',
        type: account.type || 'local',
        mt5_account_id: account.mt5_account_id || '',
        mt5_token: account.mt5_token || '',
        ctrader_account_id: account.ctrader_account_id || '',
        ctrader_token: account.ctrader_token || '',
        ctrader_client_id: account.ctrader_client_id || '',
        ctrader_client_secret: account.ctrader_client_secret || '',
        ctrader_env: account.ctrader_env || 'live',
        timezone_offset: account.timezone_offset || 8,
      };
    } else {
      newAccount = {
        name: '',
        type: 'local',
        mt5_account_id: '',
        mt5_token: '',
        ctrader_account_id: '',
        ctrader_token: '',
        ctrader_client_id: '',
        ctrader_client_secret: '',
        ctrader_env: 'live',
        timezone_offset: 8,
      };
    }
  } else if (!show) {
    lastShow = false;
    lastAccountId = undefined;
  }

  let importFile = null;
  let processing = false;

  async function saveAccount() {
    if (!newAccount.name.trim()) {
      alert('請輸入帳號名稱');
      return;
    }
    if (!account && newAccount.type === 'ftmo' && !importFile) {
      alert('請選擇 FTMO CSV 檔案');
      return;
    }

    try {
      processing = true;
      let accountId;

      const payload = { ...newAccount };
      if (payload.type === 'ftmo') payload.type = 'local';

      if (account) {
        // Update mode
        await accountsAPI.update(account.id, payload);
        accountId = account.id;
        alert('帳號更新成功！');
      } else {
        // Create mode
        const res = await accountsAPI.create(payload);
        accountId = res.data.id;

        // If FTMO, Import CSV (only on creation)
        if (newAccount.type === 'ftmo' && importFile) {
          const formData = new FormData();
          formData.append('file', importFile);
          formData.append('source', 'ftmo');
          await accountsAPI.importCSV(accountId, formData);
          alert('帳號建立並匯入完成！');
        } else {
          alert('帳號建立成功！');
        }
      }

      dispatch('success', { accountId });
      show = false;
    } catch (e) {
      console.error(e);
      const errorMsg = e.response?.data?.error || e.message || '未知錯誤';
      alert('操作失敗: ' + errorMsg);
    } finally {
      processing = false;
    }
  }

  function close() {
    show = false;
    dispatch('close');
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={close} role="presentation">
    <div class="modal card">
      <h2>{account ? '編輯交易帳號' : '新增交易帳號'}</h2>
      <div class="form-group">
        <label for="new-acc-name">帳號名稱</label>
        <input
          id="new-acc-name"
          type="text"
          class="form-control"
          bind:value={newAccount.name}
          placeholder="如：個人實盤"
        />
      </div>

      {#if !account}
        <div class="form-group">
          <label>帳號類型</label>
          <div class="type-selector">
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="local" /> 本地記錄 (完全手動)
            </label>
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="ftmo" /> 從 FTMO CSV 匯入
            </label>
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="metatrader" /> MetaTrader 5 (自動同步)
            </label>
            <label class="radio-label">
              <input type="radio" bind:group={newAccount.type} value="ctrader" /> cTrader (自動同步)
            </label>
          </div>
        </div>
      {/if}

      {#if newAccount.type === 'ftmo'}
        <div class="form-group import-field">
          <label for="ftmo-csv">選擇 FTMO CSV 檔案</label>
          <input
            id="ftmo-csv"
            type="file"
            accept=".csv"
            class="form-control"
            on:change={e => {
              importFile = e.target.files[0];
              if (importFile && !newAccount.name.trim()) {
                newAccount.name = importFile.name.replace(/\.[^/.]+$/, '');
              }
            }}
          />
          <p class="help-text">建立帳號後將自動匯入此 CSV 內的交易紀錄。</p>
        </div>
      {/if}

      {#if newAccount.type === 'metatrader'}
        <div class="metatrader-fields">
          <div class="form-group">
            <label for="mt5-id">MT5 帳號 ID</label>
            <input
              id="mt5-id"
              type="text"
              class="form-control"
              bind:value={newAccount.mt5_account_id}
              placeholder="您的 MT5 登入帳號"
            />
          </div>
          <div class="form-group">
            <label for="mt5-token">MT5 Token (MetaApi)</label>
            <input
              id="mt5-token"
              type="password"
              class="form-control"
              bind:value={newAccount.mt5_token}
              placeholder="您的 MetaApi Token"
            />
          </div>
        </div>
      {/if}

      {#if newAccount.type === 'ctrader'}
        <div class="ctrader-fields">
          <div class="form-group">
            <label for="ctrader-id">cTrader 交易帳號 ID (Login)</label>
            <input
              id="ctrader-id"
              type="text"
              class="form-control"
              bind:value={newAccount.ctrader_account_id}
              placeholder="例如：6543210"
            />
          </div>
          <div class="form-group">
            <label for="ctrader-token">cTrader API Access Token</label>
            <textarea
              id="ctrader-token"
              class="form-control"
              bind:value={newAccount.ctrader_token}
              placeholder="輸入您的 Access Token"
              rows="2"
            ></textarea>
          </div>
          <div class="form-group">
            <label for="ctrader-client-id">Client ID</label>
            <input
              id="ctrader-client-id"
              type="text"
              class="form-control"
              bind:value={newAccount.ctrader_client_id}
              placeholder="您的 Open API App Client ID"
            />
          </div>
          <div class="form-group">
            <label for="ctrader-client-secret">Client Secret</label>
            <input
              id="ctrader-client-secret"
              type="password"
              class="form-control"
              bind:value={newAccount.ctrader_client_secret}
              placeholder="您的 Open API App Client Secret"
            />
          </div>
          <div class="form-group">
            <label for="ctrader-env">Environment</label>
            <select id="ctrader-env" class="form-control" bind:value={newAccount.ctrader_env}>
              <option value="live">Live</option>
              <option value="demo">Demo</option>
            </select>
            <p class="help-text">根據您的帳號類型選擇伺服器環境。</p>
          </div>
        </div>
      {/if}

      <div class="form-group">
        <label for="new-acc-timezone">時區設定 (UTC)</label>
        <select id="new-acc-timezone" class="form-control" bind:value={newAccount.timezone_offset}>
          {#each Array.from({ length: 25 }, (_, i) => i - 12) as offset}
            <option value={offset}>UTC{offset >= 0 ? '+' : ''}{offset}</option>
          {/each}
        </select>
        <p class="help-text">此設定將套用於此帳號下的所有交易紀錄時間。</p>
      </div>

      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={close} disabled={processing}>取消</button>
        <button class="btn btn-primary" on:click={saveAccount} disabled={processing}>
          {processing ? '處理中...' : account ? '儲存變更' : '建立帳號'}
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
    background: white;
    border-radius: 12px;
  }

  .form-group {
    margin-bottom: 1.5rem;
    text-align: left;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 600;
  }

  .form-control {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    box-sizing: border-box;
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

  .help-text {
    font-size: 0.75rem;
    color: #94a3b8;
    margin-top: 0.5rem;
  }

  .metatrader-fields,
  .ctrader-fields {
    background: #f8fafc;
    padding: 1rem;
    border-radius: 8px;
    margin-bottom: 1.5rem;
    border: 1px solid #e2e8f0;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
  }

  .btn {
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    border: 1px solid #ddd;
    background: white;
  }

  .btn-primary {
    background: #6366f1;
    color: white;
    border: none;
  }
</style>
