<script>
  import { Router, Route, Link, navigate } from 'svelte-routing';
  import { onMount, onDestroy } from 'svelte';
  import TradeForm from './components/TradeForm.svelte';
  import TradeList from './components/TradeList.svelte';
  import Dashboard from './components/Dashboard.svelte';
  import DailyPlanList from './components/DailyPlanList.svelte';
  import DailyPlanForm from './components/DailyPlanForm.svelte';
  import Home from './components/Home.svelte';
  import AccountSelector from './components/AccountSelector.svelte';
  import AccountManagement from './components/AccountManagement.svelte';
  import SharedViewer from './components/SharedViewer.svelte';
  import { SYMBOLS, MARKET_SESSIONS } from './lib/constants';
  import { determineMarketSession } from './lib/utils';
  import { selectedSymbol } from './lib/stores';
  import { auth, logout, checkAuth } from './lib/auth';
  import Login from './components/Login.svelte';
  import ChangePasswordModal from './components/ChangePasswordModal.svelte';

  let activeNav = 'home';
  let currentTime = new Date();
  let timer;
  let showChangePassword = false;

  onMount(async () => {
    await checkAuth();
    timer = setInterval(() => {
      currentTime = new Date();
    }, 1000); // 1秒更新一次秒針，或60000更新分
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  $: currentSessionValue = determineMarketSession(currentTime);
  $: currentSession = MARKET_SESSIONS.find(s => s.value === currentSessionValue);

  function formatTime(date) {
    return date.toLocaleTimeString('zh-TW', { hour12: false, hour: '2-digit', minute: '2-digit' });
  }

  function formatDate(date) {
    return date.toLocaleDateString('zh-TW', { month: '2-digit', day: '2-digit' }).replace(/\//g, '/');
  }
  function handleLogout() {
    if (confirm('確定要登出嗎？')) {
      logout();
    }
  }
</script>

<Router>
  <div class="app">
    <nav class="navbar">
      <div class="navbar-content">
        <Link to="/" class="nav-brand" on:click={() => (activeNav = 'home')}>
          <div class="logo-image-container">
            <img src="/logo.png" alt="Trade Time Machine Logo" class="brand-logo-img" />
          </div>
          <span class="app-version-tag">v1.0.0</span>
        </Link>

        <div class="header-tools">
          <div class="symbol-selector-wrapper">
            <div class="symbol-selector">
              <span class="selector-icon">📊</span>
              <select bind:value={$selectedSymbol}>
                {#each SYMBOLS as sym}
                  <option value={sym}>{sym}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>

        <div class="market-status">
          <div class="current-time-box">
            <span class="date">{formatDate(currentTime)}</span>
            <span class="time">{formatTime(currentTime)}</span>
          </div>
          {#if currentSession}
            <div class="current-session-tag {currentSessionValue}">
              <span class="session-icon">{currentSession.icon}</span>
              <span class="session-label">{currentSession.label}</span>
            </div>
          {/if}
        </div>

        <div class="nav-links">
          <Link
            to="/dashboard"
            class={activeNav === 'dashboard' ? 'active' : ''}
            on:click={() => (activeNav = 'dashboard')}
          >
            統計面板
          </Link>
          <AccountSelector />
          <Link
            to="/accounts"
            class={activeNav === 'accounts' ? 'nav-settings-btn active' : 'nav-settings-btn'}
            on:click={() => (activeNav = 'accounts')}
            title="帳號管理"
          >
            ⚙️
          </Link>
          
          {#if $auth.isAuthenticated}
            <div class="user-profile">
              <span class="username" title="修改密碼" on:click={() => showChangePassword = true} role="button" tabindex="0">👤 {$auth.user?.username}</span>
              <button class="logout-btn" on:click={handleLogout} title="登出">🚪</button>
            </div>
          {/if}
        </div>
      </div>
    </nav>

    <main class="container">
      <Route path="/shared/:token" let:params>
        <SharedViewer token={params.token} />
      </Route>

      {#if $auth.isAuthenticated}
        <Route path="/" component={Home} />
        <Route path="/trades" component={TradeList} />
        <Route path="/plans" component={DailyPlanList} />
        <Route path="/plans/new" component={DailyPlanForm} />
        <Route path="/plans/edit/:id" component={DailyPlanForm} />
        <Route path="/new" component={TradeForm} />
        <Route path="/edit/:id" component={TradeForm} />
        <Route path="/dashboard" component={Dashboard} />
        <Route path="/accounts" component={AccountManagement} />
      {:else}
        <!-- 如果不是分享路徑，且未登入，顯示登入頁面 -->
        {#if !window.location.pathname.startsWith('/shared/')}
          <Login />
        {/if}
      {/if}
    </main>
  </div>
</Router>

<ChangePasswordModal show={showChangePassword} onClose={() => showChangePassword = false} />

<style>
  :global(:root) {
    --primary: #6366f1;
    --primary-hover: #4f46e5;
    --bg-main: #f8fafc;
    --card-bg: #ffffff;
    --text-main: #1e293b;
    --text-muted: #64748b;
    --border-color: #e2e8f0;
    --radius-lg: 16px;
    --radius-md: 12px;
    --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
    --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
  }

  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    font-family:
      'Inter',
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      Roboto,
      sans-serif;
    background-color: var(--bg-main);
    color: var(--text-main);
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  .app {
    min-height: 100vh;
  }

  .navbar {
    background: rgba(255, 255, 255, 0.8);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--border-color);
    padding: 0.75rem 0;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .navbar-content {
    max-width: 1400px;
    margin: 0 auto;
    padding: 0 2rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  /* 使用 :global 確保樣式能套用到 svelte-routing 的 Link 組件 */
  :global(.nav-brand) {
    display: flex !important;
    align-items: center;
    text-decoration: none !important;
    outline: none;
    user-select: none;
    gap: 0.75rem;
    padding: 2px 0;
  }

  .logo-image-container {
    height: 85px; /* 進一步提高到 85px，確保齒輪頂部與底部完全不被裁切 */
    width: 280px; 
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }

  .brand-logo-img {
    width: 100%;
    height: 100%;
    object-fit: cover; 
    object-position: center 48%; /* 稍微下移，確保頂部不被切到 */
    mix-blend-mode: multiply;
    pointer-events: none;
    transform: scale(1.1); 
  }

  :global(.app-version-tag) {
    font-size: 0.65rem;
    color: #94a3b8;
    background: #f1f5f9;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    font-weight: 600;
    pointer-events: none;
    margin-top: 3.2rem; /* 配合容器加高，版號位置再次微調 */
  }


  .nav-links {
    display: flex;
    gap: 0.5rem;
  }

  .nav-links :global(a) {
    text-decoration: none;
    color: var(--text-muted);
    font-weight: 500;
    font-size: 0.9375rem;
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    transition: all 0.2s ease;
  }

  .nav-links :global(a:hover) {
    color: var(--primary);
    background: #f1f5f9;
  }

  .nav-links .active {
    color: var(--primary);
  }

  .nav-settings-btn {
    text-decoration: none;
    font-size: 1.2rem;
    opacity: 0.6;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    padding: 0.5rem;
    border-radius: 8px;
  }

  /* 市場狀態樣式 */
  .market-status {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.4rem 1rem;
    background: #f8fafc;
    border-radius: 50px;
    border: 1px solid #e2e8f0;
    margin: 0 1rem;
  }

  .current-time-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    line-height: 1;
    border-right: 1px solid #e2e8f0;
    padding-right: 0.8rem;
  }

  .current-time-box .date {
    font-size: 0.65rem;
    color: #94a3b8;
    font-weight: 600;
    margin-bottom: 2px;
  }

  .current-time-box .time {
    font-size: 0.95rem;
    color: #1e293b;
    font-weight: 700;
    font-family: 'JetBrains Mono', monospace;
  }

  .current-session-tag {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    font-weight: 700;
    padding: 0.2rem 0.6rem;
    border-radius: 8px;
  }

  .current-session-tag.asian { background: #e0f2fe; color: #0369a1; }
  .current-session-tag.european { background: #fef3c7; color: #b45309; }
  .current-session-tag.us { background: #fce7f3; color: #be185d; }

  .session-icon {
    font-size: 1rem;
  }

  .nav-settings-btn:hover {
    opacity: 1;
    background: rgba(0, 0, 0, 0.05);
    transform: rotate(30deg);
  }

  .nav-settings-btn.active {
    opacity: 1;
    color: var(--primary);
  }

  .nav-links :global(a.active) {
    background: var(--primary);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.25);
  }

  .user-profile {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-left: 0.5rem;
    padding-left: 0.75rem;
    border-left: 1px solid var(--border-color);
  }

  .username {
    font-size: 0.85rem;
    font-weight: 700;
    color: var(--text-main);
    background: #f1f5f9;
    padding: 0.4rem 0.75rem;
    border-radius: 20px;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
    transition: all 0.2s;
  }

  .username:hover {
    background: #e2e8f0;
    transform: translateY(-1px);
  }

  .logout-btn {
    background: none;
    border: none;
    font-size: 1.1rem;
    cursor: pointer;
    padding: 0.4rem;
    border-radius: 8px;
    transition: all 0.2s;
    opacity: 0.6;
  }

  .logout-btn:hover {
    background: #fee2e2;
    opacity: 1;
    transform: scale(1.1);
  }


  .header-tools {
    flex: 1;
    display: flex;
    align-items: center;
    padding-left: 2rem;
    gap: 1rem;
  }

  .navbar-actions {
    display: flex;
    gap: 0.5rem;
    margin-left: auto;
    padding-right: 1.5rem;
    border-right: 1px solid var(--border-color);
  }

  .symbol-selector-wrapper {
    display: flex;
    justify-content: flex-start;
  }

  .symbol-selector {
    display: flex;
    align-items: center;
    background: #f1f5f9;
    border: 1px solid var(--border-color);
    padding: 0.4rem 0.75rem;
    border-radius: 12px;
    gap: 0.5rem;
    transition: all 0.2s;
  }

  .symbol-selector:hover {
    border-color: var(--primary);
    background: white;
    box-shadow: var(--shadow-sm);
  }

  .selector-icon {
    font-size: 1.1rem;
  }

  .symbol-selector select {
    border: none;
    background: transparent;
    font-weight: 700;
    color: var(--text-main);
    font-size: 1rem;
    cursor: pointer;
    outline: none;
    padding-right: 0.5rem;
  }

  .container {
    max-width: 1400px;
    margin: 1rem auto 2rem;
    padding: 0 2rem;
  }

  :global(.card) {
    background: var(--card-bg);
    border-radius: var(--radius-lg);
    padding: 2rem;
    border: 1px solid var(--border-color);
    box-shadow: var(--shadow-sm);
    transition: box-shadow 0.3s ease;
  }

  :global(.btn) {
    padding: 0.625rem 1.25rem;
    border: none;
    border-radius: var(--radius-md);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    font-size: 0.875rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    white-space: nowrap;
  }

  :global(.btn-sm) {
    padding: 0.4rem 0.8rem;
    font-size: 0.8rem;
  }

  :global(.btn-primary) {
    background: var(--primary);
    color: white;
  }

  :global(.btn-primary:hover) {
    background: var(--primary-hover);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.25);
  }

  :global(.btn-danger) {
    background: #f56565;
    color: white;
  }

  :global(.btn-danger:hover) {
    background: #e53e3e;
  }

  :global(.btn-warning) {
    background: #ed8936;
    color: white;
  }

  :global(.btn-warning:hover) {
    background: #dd6b20;
  }

  :global(.form-group) {
    margin-bottom: 1rem;
  }

  :global(.form-group label) {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 600;
    color: #2d3748;
  }

  :global(.form-control) {
    width: 100%;
    padding: 0.75rem;
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    font-size: 1rem;
    transition: border-color 0.3s ease;
  }

  :global(.form-control:focus) {
    outline: none;
    border-color: #667eea;
  }

  :global(.badge) {
    display: inline-block;
    padding: 0.25rem 0.75rem;
    border-radius: 12px;
    font-size: 0.875rem;
    font-weight: 600;
  }

  :global(.badge-success) {
    background: #c6f6d5;
    color: #22543d;
  }

  :global(.badge-danger) {
    background: #fed7d7;
    color: #742a2a;
  }

  :global(.badge-info) {
    background: #bee3f8;
    color: #2c5282;
  }
</style>
