<script>
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { selectedAccountId, accounts } from '../lib/stores';

  let loading = true;

  async function fetchAccounts() {
    try {
      const res = await accountsAPI.getAll();
      const data = res.data;
      console.log('Fetched accounts:', data);
      accounts.set(data);

      // 如果目前選擇的帳號不在列表中，預設選擇第一個
      if (data.length > 0 && !data.find(a => a.id === $selectedAccountId)) {
        selectedAccountId.set(data[0].id);
      }
    } catch (e) {
      console.error('Failed to fetch accounts:', e);
    } finally {
      loading = false;
    }
  }

  onMount(fetchAccounts);

  function handleAccountChange(e) {
    selectedAccountId.set(parseInt(e.target.value));
    // 重整頁面以確保所有元件重新加載數據
    window.location.reload();
  }
</script>

<div class="account-selector">
  {#if !loading}
    <div class="selector-wrapper">
      <span class="label">切換帳號:</span>
      <select value={$selectedAccountId} on:change={handleAccountChange}>
        {#each $accounts as account}
          <option value={account.id}>
            {account.name}
            {account.type === 'metatrader' ? '(MT5)' : '(本地)'}
          </option>
        {/each}
      </select>
      <a href="/accounts" class="manage-btn" title="管理帳號">⚙️</a>
    </div>
  {/if}
</div>

<style>
  .account-selector {
    margin-right: 1.5rem;
  }

  .selector-wrapper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: rgba(255, 255, 255, 0.05);
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .label {
    font-size: 0.8rem;
    color: var(--text-muted);
    white-space: nowrap;
  }

  select {
    background: transparent;
    color: white;
    border: none;
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    outline: none;
    padding-right: 0.5rem;
  }

  option {
    background: #1a1a1a;
    color: white;
  }

  .manage-btn {
    text-decoration: none;
    font-size: 1rem;
    opacity: 0.6;
    transition: opacity 0.2s;
  }

  .manage-btn:hover {
    opacity: 1;
  }
</style>
