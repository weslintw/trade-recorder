<script>
  export let formData = {};

  const eliteChecklist = [
    { id: 'trend_line', label: '破趨勢線' },
    { id: 'price_level', label: '破價位' },
    { id: 'impulse_wave', label: '有驅動浪' },
    { id: 'high_low', label: '不過高低' },
    { id: 'sentiment', label: '情緒轉換' },
  ];

  const entryPatterns = ['甲', '乙', '丙', '丁', '大Leading', '小Leading'];

  function togglePattern(patternName) {
    const index = formData.entry_pattern.findIndex(p => p.name === patternName);
    if (index >= 0) {
      formData.entry_pattern = formData.entry_pattern.filter(p => p.name !== patternName);
    } else {
      formData.entry_pattern = [...formData.entry_pattern, { name: patternName, images: [] }];
    }
  }
</script>

<div class="checklist-section">
  <label class="checklist-label">菁英檢查清單：</label>
  <div class="checklist-items">
    {#each eliteChecklist as item}
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

<div class="entry-pattern-section">
  <span class="entry-pattern-label">進場樣態：</span>
  <div class="entry-pattern-options">
    {#each entryPatterns as patternName}
      {@const isSelected = formData.entry_pattern.some(p => p.name === patternName)}
      <div
        class="pattern-option"
        class:active={isSelected}
        role="button"
        tabindex="0"
        on:click={() => togglePattern(patternName)}
        on:keydown={e => (e.key === 'Enter' || e.key === ' ') && togglePattern(patternName)}
      >
        <span class="pattern-name">{patternName}</span>
      </div>
    {/each}
  </div>
</div>

<style>
  .checklist-section {
    margin-top: 1rem;
    padding: 1rem;
    background: var(--nav-group-bg);
    border-radius: 8px;
    border: 1px solid var(--border-color);
  }

  .checklist-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-main);
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
    border: 2px solid var(--border-color);
    border-radius: 8px;
    background: var(--card-bg);
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
    width: fit-content;
  }

  .checklist-btn:hover {
    border-color: var(--primary);
    background: var(--bg-main);
  }

  .checklist-btn.active {
    border-color: var(--primary);
    background: var(--primary);
  }

  .btn-text {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-main);
  }

  .checklist-btn.active .btn-text {
    color: white;
  }

  .entry-pattern-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: var(--nav-group-bg);
    border-radius: 8px;
    border: 1px solid var(--border-color);
  }

  .entry-pattern-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-main);
    margin-bottom: 0.75rem;
  }

  .entry-pattern-options {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .pattern-option {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 1rem;
    border: 2px solid var(--border-color);
    border-radius: 8px;
    background: var(--card-bg);
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
  }

  .pattern-option:hover {
    border-color: var(--primary);
    background: var(--bg-main);
  }

  .pattern-option.active {
    border-color: var(--primary);
    background: var(--primary);
  }

  .pattern-name {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-main);
  }

  .pattern-option.active .pattern-name {
    color: white;
  }
</style>
