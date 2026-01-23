<script>
  import SignalGrid from './SignalGrid.svelte';

  export let formData = {};

  const legendChecklist = [
    { id: 'item_618_786', label: '王者出現回調618或786' },
    { id: 'item_che', label: '大時區破[測]破' },
    { id: 'item_de', label: '整理段的ABC[D][E]' },
  ];

  const timeframes = [
    { label: '1分', value: 'M1' },
    { label: '5分', value: 'M5' },
    { label: '15分', value: 'M15' },
    { label: '30分', value: 'M30' },
    { label: '1小時', value: 'H1' },
    { label: '4小時', value: 'H4' },
    { label: '天', value: 'D1' },
  ];
</script>

<div class="checklist-section">
  <label class="checklist-label">傳奇檢查清單：</label>
  <div class="checklist-items">
    {#each legendChecklist as item}
      {@const isChecked = formData.entry_checklist[item.id] || false}
      <div
        class="checklist-btn"
        class:active={isChecked}
        role="button"
        tabindex="0"
        on:click={() => {
          formData.entry_checklist = {
            ...formData.entry_checklist,
            [item.id]: !isChecked,
          };
        }}
        on:keydown={e => {
          if (e.key === 'Enter' || e.key === ' ') {
            formData.entry_checklist = {
              ...formData.entry_checklist,
              [item.id]: !isChecked,
            };
          }
        }}
      >
        <span class="btn-text">{item.label}</span>
      </div>
    {/each}
  </div>
</div>

<!-- 王者回調 Section (HTF only, no image) -->
{#if formData.entry_checklist['item_618_786']}
  <div class="signals-section nested king-section">
    <label class="signals-label">王者出現回調618或786 - 請選擇時區：</label>
    <div class="htf-selector-row">
      <div class="timeframe-options">
        {#each timeframes as tf}
          <button
            type="button"
            class="timeframe-btn"
            class:active={formData.legend_king_htf === tf.value}
            on:click={() => (formData.legend_king_htf = tf.value)}
          >
            {tf.label}
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}

<!-- HTF Break Section (HTF only, no image) -->
{#if formData.entry_checklist['item_che']}
  <div class="signals-section nested htf-section">
    <label class="signals-label">大時區破"測"破 - 請選擇大時區：</label>
    <div class="htf-selector-row">
      <div class="timeframe-options">
        {#each timeframes as tf}
          <button
            type="button"
            class="timeframe-btn"
            class:active={formData.legend_htf === tf.value}
            on:click={() => (formData.legend_htf = tf.value)}
          >
            {tf.label}
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}

<!-- DE (ABCDE/Signal) Section (HTF + SignalGrid, no image) -->
{#if formData.entry_checklist['item_de']}
  <div class="signals-section nested">
    <label class="signals-label">達人整理段訊號 (ABC[D][E]):</label>
    <div class="htf-selector-row" style="margin-bottom: 1.5rem;">
      <div class="timeframe-options">
        {#each timeframes as tf}
          <button
            type="button"
            class="timeframe-btn"
            class:active={formData.legend_de_htf === tf.value}
            on:click={() => (formData.legend_de_htf = tf.value)}
          >
            {tf.label}
          </button>
        {/each}
      </div>
    </div>
    <SignalGrid bind:entry_signals={formData.entry_signals} bind:formData />
  </div>
{/if}

<style>
  .checklist-section {
    margin-top: 1rem;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .checklist-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.75rem;
  }

  .checklist-items {
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .checklist-btn {
    display: inline-flex;
    align-items: center;
    padding: 0.35rem 0.75rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
    width: fit-content;
  }

  .checklist-btn:hover {
    border-color: #805ad5;
    background: #f9f5ff;
  }

  .checklist-btn.active {
    border-color: #805ad5;
    background: #805ad5;
  }

  .btn-text {
    font-size: 0.9rem;
    font-weight: 500;
    color: #4a5568;
  }

  .checklist-btn.active .btn-text {
    color: white;
  }

  .signals-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: #fdfdfd;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
  }

  .signals-section.nested {
    background: #f8fafc;
    border: 1px dashed #cbd5e0;
    margin-left: 1rem;
    padding: 1rem;
  }

  .signals-label {
    display: block;
    font-weight: 600;
    margin-bottom: 1rem;
    color: #4a5568;
    font-size: 0.95rem;
  }

  .htf-selector-row {
    margin-bottom: 1rem;
    overflow-x: auto;
  }

  .timeframe-options {
    display: flex;
    gap: 0.5rem;
    padding-bottom: 0.25rem;
  }

  .timeframe-btn {
    padding: 0.35rem 0.75rem;
    border: 1px solid #cbd5e0;
    background: white;
    border-radius: 6px;
    font-size: 0.85rem;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.2s;
  }

  .timeframe-btn.active {
    background: #805ad5;
    color: white;
    border-color: #805ad5;
  }
</style>
