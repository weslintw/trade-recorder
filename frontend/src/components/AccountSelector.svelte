<script>
  import { onMount } from 'svelte';
  import { accountsAPI } from '../lib/api';
  import { selectedAccountId, accounts } from '../lib/stores';
  import { auth } from '../lib/auth';

  let loading = true;
  let firstLoad = true;

  async function fetchAccounts() {
    console.log('🟣 AccountSelector.fetchAccounts called');
    if (!$auth.isAuthenticated) return;

    try {
      loading = true;
      const res = await accountsAPI.getAll();
      const data = res.data || [];
      accounts.set(data);
    } catch (e) {
      console.error('Failed to fetch accounts:', e);
    } finally {
      loading = false;
      firstLoad = false;
    }
  }

  onMount(() => {
    console.log('🟣 AccountSelector mounted');
    if ($auth.isAuthenticated) {
      console.log('🟣 AccountSelector: Initial fetch');
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
        {#if $accounts.length === 0}
          <option value={null} disabled>尚未建立交易帳號</option>
        {:else}
          {#each $accounts as account}
            <option value={account.id}>
              {account.name}
              {account.type === 'ctrader'
                  ? '(cTrader)'
                  : '(本地)'}
            </option>
          {/each}
        {/if}
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
    background: var(--card-bg);
    padding: 0.4rem 0.8rem;
    border-radius: 12px;
    border: 1px solid var(--border-color);
    transition: all 0.2s ease;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.03);
  }

  .selector-wrapper:hover {
    border-color: var(--primary);
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.1);
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
    background: rgba(99, 102, 241, 0.05);
  }

  option {
    background: var(--card-bg);
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
