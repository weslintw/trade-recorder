<script>
  import { createEventDispatcher } from 'svelte';
  import SignalGrid from './SignalGrid.svelte';
  const dispatch = createEventDispatcher();

  export let formData = {};
  export let signalImagesCache = {};
</script>

<div class="signals-section">
  <label class="signals-label">選擇訊號（可多選）：</label>
  <SignalGrid 
    bind:formData 
    bind:signalImagesCache 
    on:enlarge 
  />
</div>

<style>
  .signals-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: #fdfdfd;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
  }
  
  .signals-label {
    display: block;
    font-weight: 600;
    margin-bottom: 1rem;
    color: #4a5568;
  }
</style>

