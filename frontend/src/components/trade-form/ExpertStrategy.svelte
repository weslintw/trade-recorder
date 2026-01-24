<script>
  import { createEventDispatcher } from 'svelte';
  import SignalGrid from './SignalGrid.svelte';
  const dispatch = createEventDispatcher();

  export let formData = {};

  function enlargeImage(image, title, context) {
    dispatch('enlarge', { image, title, context });
  }
</script>

<div class="signals-section">
  <label class="signals-label">選擇訊號（可多選）：</label>
  <SignalGrid bind:entry_signals={formData.entry_signals} bind:formData on:enlarge />
</div>

<style>
  .signals-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: var(--nav-group-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
  }

  .signals-label {
    display: block;
    font-weight: 600;
    margin-bottom: 1rem;
    color: var(--text-main);
    font-size: 0.95rem;
  }
</style>
