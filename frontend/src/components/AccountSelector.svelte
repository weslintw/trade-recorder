<script>
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { selectedAccountId, accounts } from '../lib/stores';
  import { auth } from '../lib/auth';

  let loading = true;
  let firstLoad = true;

  async function fetchAccounts() {
    if (!$auth.isAuthenticated) return;
    
    try {
      loading = true;
      const res = await accountsAPI.getAll();
      const data = res.data;
      accounts.set(data);

      if (data.length > 0) {
        // 如果目前沒有選擇帳號，或目前選擇的帳號不在清單中，則預設選擇第一個
        const exists = data.find(a => a.id === $selectedAccountId);
        if (!$selectedAccountId || !exists) {
          selectedAccountId.set(data[0].id);
        }
      }
    } catch (e) {
      console.error('Failed to fetch accounts:', e);
    } finally {
      loading = false;
      firstLoad = false;
    }
  }

  // 當登入狀態改變時重新獲取帳號
  $: if ($auth.isAuthenticated) {
    fetchAccounts();
  }

  onMount(() => {
    if ($auth.isAuthenticated) {
      fetchAccounts();
    }
  });

  function handleAccountChange(e) {
    selectedAccountId.set(parseInt(e.target.value));
    // 重整頁面以確保所有元件重新加載數據
    window.location.reload();
  }
</script>

<div class="account-selector">
  {#if !loading || !firstLoad}
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
    </div>
  {/if}
</div>

<style>
  .account-selector {
    /* 移動到 nav-links 後不需要額外 margin */
  }

  .selector-wrapper {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    background: #f1f5f9;
    padding: 0.4rem 0.75rem;
    border-radius: 12px;
    border: 1px solid var(--border-color);
    transition: all 0.2s ease;
  }

  .selector-wrapper:hover {
    border-color: var(--primary);
    background: white;
    box-shadow: var(--shadow-sm);
  }

  .label {
    font-size: 0.85rem;
    color: var(--text-muted);
    white-space: nowrap;
    font-weight: 600;
  }

  select {
    background: transparent;
    color: var(--text-main);
    border: none;
    font-size: 0.95rem;
    font-weight: 700;
    cursor: pointer;
    outline: none;
    padding: 2px 4px;
    border-radius: 6px;
  }

  select:focus {
    background: rgba(0, 0, 0, 0.05);
  }

  option {
    background: white;
    color: var(--text-main);
  }

  .manage-btn {
    text-decoration: none;
    font-size: 1.1rem;
    opacity: 0.7;
    transition:
      transform 0.2s,
      opacity 0.2s;
    display: flex;
    align-items: center;
    padding-left: 0.5rem;
    margin-left: 0.2rem;
    border-left: 1px solid var(--border-color);
  }

  .manage-btn:hover {
    opacity: 1;
    transform: rotate(30deg);
  }
</style>
