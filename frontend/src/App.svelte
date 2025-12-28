<script>
  import { Router, Route, Link } from 'svelte-routing';
  import TradeForm from './components/TradeForm.svelte';
  import TradeList from './components/TradeList.svelte';
  import Dashboard from './components/Dashboard.svelte';
  import DailyPlanList from './components/DailyPlanList.svelte';
  import DailyPlanForm from './components/DailyPlanForm.svelte';
  import Home from './components/Home.svelte';
  import { SYMBOLS } from './lib/constants';
  import { selectedSymbol } from './lib/stores';

  let activeNav = 'home';
</script>

<Router>
  <div class="app">
    <nav class="navbar">
      <div class="navbar-content">
        <Link to="/" class="nav-brand" on:click={() => (activeNav = 'home')}>
          <div class="modern-logo">
            <div class="logo-bars">
              <div class="bar bar-short"></div>
              <div class="bar bar-tall"></div>
              <div class="bar bar-medium"></div>
            </div>
            <div class="logo-accent"></div>
          </div>
        </Link>

        <!-- 全局交易品種切換 -->
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

        <div class="nav-links">
          <Link
            to="/trades"
            class={activeNav === 'list' ? 'active' : ''}
            on:click={() => (activeNav = 'list')}
          >
            交易紀錄
          </Link>
          <Link
            to="/plans"
            class={activeNav === 'plans' ? 'active' : ''}
            on:click={() => (activeNav = 'plans')}
          >
            每日規劃
          </Link>
          <Link
            to="/new"
            class={activeNav === 'new' ? 'active' : ''}
            on:click={() => (activeNav = 'new')}
          >
            新增交易
          </Link>
          <Link
            to="/dashboard"
            class={activeNav === 'dashboard' ? 'active' : ''}
            on:click={() => (activeNav = 'dashboard')}
          >
            統計面板
          </Link>
        </div>
      </div>
    </nav>

    <main class="container">
      <Route path="/" component={Home} />
      <Route path="/trades" component={TradeList} />
      <Route path="/plans" component={DailyPlanList} />
      <Route path="/plans/new" component={DailyPlanForm} />
      <Route path="/plans/edit/:id" component={DailyPlanForm} />
      <Route path="/new" component={TradeForm} />
      <Route path="/edit/:id" component={TradeForm} />
      <Route path="/dashboard" component={Dashboard} />
    </main>
  </div>
</Router>

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

  .nav-brand {
    display: flex;
    align-items: center;
    text-decoration: none !important;
    outline: none;
    user-select: none;
  }

  .nav-brand:hover,
  .nav-brand:focus,
  .nav-brand:active {
    text-decoration: none !important;
  }

  .modern-logo {
    position: relative;
    width: 40px;
    height: 40px;
    background: linear-gradient(135deg, #6366f1, #8b5cf6);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .modern-logo:hover {
    transform: scale(1.05) rotate(-2deg);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
  }

  .logo-bars {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    height: 18px;
  }

  .bar {
    width: 4px;
    background: white;
    border-radius: 2px;
    transition: height 0.3s ease;
  }

  .bar-short {
    height: 8px;
  }
  .bar-tall {
    height: 18px;
  }
  .bar-medium {
    height: 13px;
  }

  .logo-accent {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 6px;
    height: 6px;
    background: #fbbf24;
    border-radius: 50%;
    box-shadow: 0 0 8px rgba(251, 191, 36, 0.6);
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

  .nav-links :global(a.active) {
    background: var(--primary);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
  }

  .symbol-selector-wrapper {
    flex: 1;
    display: flex;
    justify-content: flex-start;
    padding-left: 2rem;
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
    margin: 2rem auto;
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

  :global(.form-group) {
    margin-bottom: 1.5rem;
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
