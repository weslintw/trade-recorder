<script>
  export let formData = {};
  export let entry_signals = []; 

  const expertSignalsLong = ['向下蘇美', '起漲靠山', '雙柱', '倚天', '攻城池上'];
  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '向上蘇美', '雷霆'];

  $: expertSignals = formData.side === 'long' ? expertSignalsLong : expertSignalsShort;

  function isSignalSelected(signalName) {
    if (!entry_signals || !Array.isArray(entry_signals)) return false;
    return entry_signals.some(s =>
      typeof s === 'string' ? s === signalName : s.name === signalName
    );
  }

  function toggleSignal(signalName) {
    if (!entry_signals) entry_signals = [];

    const index = entry_signals.findIndex(s =>
      typeof s === 'string' ? s === signalName : s.name === signalName
    );

    if (index >= 0) {
      entry_signals = entry_signals.filter((_, i) => i !== index);
    } else {
      entry_signals = [
        ...entry_signals,
        {
          name: signalName,
          image: '',
          originalImage: '',
        },
      ];
    }
  }
</script>

<div class="signals-card-grid">
  {#each expertSignals as signal}
    {@const isSelected = isSignalSelected(signal)}
    <div
      class="signal-card"
      class:selected={isSelected}
      tabindex="0"
      role="button"
      on:click={e => {
        if (!e.target.closest('.signal-checkbox')) {
          toggleSignal(signal);
        }
      }}
      on:keydown={e => {
        if (e.key === 'Enter' || e.key === ' ') {
          toggleSignal(signal);
        }
      }}
    >
      <label class="signal-checkbox-wrapper">
        <input
          type="checkbox"
          class="signal-checkbox"
          checked={isSelected}
          on:change={() => toggleSignal(signal)}
          on:click|stopPropagation
        />
        <span class="signal-name">{signal}</span>
      </label>
    </div>
  {/each}
</div>

<style>
  .signals-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 1rem;
  }

  .signal-card {
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    padding: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    background: white;
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 60px;
  }

  .signal-card:hover {
    border-color: #cbd5e0;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
  }

  .signal-card.selected {
    border-color: #667eea;
    background: #f0f4ff;
  }

  .signal-checkbox-wrapper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .signal-checkbox {
    width: 18px;
    height: 18px;
    accent-color: #667eea;
    cursor: pointer;
  }

  .signal-name {
    font-weight: 600;
    color: #2d3748;
    font-size: 0.95rem;
  }

  .signal-card.selected .signal-name {
    color: #667eea;
  }
</style>
