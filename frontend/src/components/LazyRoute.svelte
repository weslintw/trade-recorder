<script>
  import { Route } from 'svelte-routing';
  export let path;
  export let loader; // () => import('./SomePage.svelte')
</script>

<Route {path} let:params>
  {#await loader()}
    <div class="lazy-route-loading">
      <div class="lazy-route-spinner"></div>
    </div>
  {:then mod}
    <svelte:component this={mod.default} {...params} />
  {:catch}
    <div class="lazy-route-error">頁面載入失敗，請重新整理頁面。</div>
  {/await}
</Route>

<style>
  .lazy-route-loading {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 60vh;
  }
  .lazy-route-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #e2e8f0;
    border-top-color: #6366f1;
    border-radius: 50%;
    animation: lazy-route-spin 0.8s linear infinite;
  }
  @keyframes lazy-route-spin {
    to {
      transform: rotate(360deg);
    }
  }
  .lazy-route-error {
    padding: 2rem;
    text-align: center;
    color: #ef4444;
    font-weight: 600;
  }
  :global(body.dark-mode) .lazy-route-spinner {
    border-color: #334155;
    border-top-color: #818cf8;
  }
</style>
