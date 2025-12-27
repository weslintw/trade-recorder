<script>
  import { Router, Route, Link } from 'svelte-routing';
  import TradeForm from './components/TradeForm.svelte';
  import TradeList from './components/TradeList.svelte';
  import Dashboard from './components/Dashboard.svelte';
  import DailyPlanList from './components/DailyPlanList.svelte';
  import DailyPlanForm from './components/DailyPlanForm.svelte';

  let activeNav = 'list';
</script>

<Router>
  <div class="app">
    <nav class="navbar">
      <div class="nav-brand">
        <h1>📊 打單紀錄器</h1>
      </div>
      <div class="nav-links">
        <Link to="/" class={activeNav === 'list' ? 'active' : ''} on:click={() => activeNav = 'list'}>
          📋 交易紀錄
        </Link>
        <Link to="/plans" class={activeNav === 'plans' ? 'active' : ''} on:click={() => activeNav = 'plans'}>
          📅 每日規劃
        </Link>
        <Link to="/new" class={activeNav === 'new' ? 'active' : ''} on:click={() => activeNav = 'new'}>
          ➕ 新增交易
        </Link>
        <Link to="/dashboard" class={activeNav === 'dashboard' ? 'active' : ''} on:click={() => activeNav = 'dashboard'}>
          📈 統計面板
        </Link>
      </div>
    </nav>

    <main class="container">
      <Route path="/" component={TradeList} />
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
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
  }

  .app {
    min-height: 100vh;
  }

  .navbar {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    box-shadow: 0 2px 20px rgba(0, 0, 0, 0.1);
    padding: 1rem 2rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .nav-brand h1 {
    font-size: 1.5rem;
    color: #667eea;
    font-weight: 700;
  }

  .nav-links {
    display: flex;
    gap: 1.5rem;
  }

  .nav-links :global(a) {
    text-decoration: none;
    color: #4a5568;
    font-weight: 500;
    padding: 0.5rem 1rem;
    border-radius: 8px;
    transition: all 0.3s ease;
  }

  .nav-links :global(a:hover) {
    background: #f7fafc;
    color: #667eea;
  }

  .nav-links :global(a.active) {
    background: #667eea;
    color: white;
  }

  .container {
    max-width: 1400px;
    margin: 2rem auto;
    padding: 0 2rem;
  }

  :global(.card) {
    background: white;
    border-radius: 16px;
    padding: 2rem;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
    margin-bottom: 2rem;
  }

  :global(.btn) {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    font-size: 1rem;
  }

  :global(.btn-primary) {
    background: #667eea;
    color: white;
  }

  :global(.btn-primary:hover) {
    background: #5568d3;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
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

