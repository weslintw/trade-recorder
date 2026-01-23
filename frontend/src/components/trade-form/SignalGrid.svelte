<script>
  export let formData = {};
  export let entry_signals = []; 

  const expertSignalsLong = ['向下蘇美', '起漲靠山', '雙柱', '夾縫', '喇叭-上', '喇叭-中', '喇叭-下', '倚天', '攻城池上'];
  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '夾縫', '向上蘇美', '雷霆'];

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

<div class="signals-items">
  {#each expertSignals as signal}
    {@const isSelected = isSignalSelected(signal)}
    <div
      class="signal-btn"
      class:active={isSelected}
      tabindex="0"
      role="button"
      on:click={() => toggleSignal(signal)}
      on:keydown={e => {
        if (e.key === 'Enter' || e.key === ' ') {
          toggleSignal(signal);
        }
      }}
    >
      <span class="btn-text">{signal}</span>
    </div>
  {/each}
</div>

<style>
  .signals-items {
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .signal-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.35rem 0.75rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
    width: fit-content;
  }

  .signal-btn:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .signal-btn.active {
    border-color: #667eea;
    background: #667eea;
  }

  .btn-text {
    font-size: 0.9rem;
    font-weight: 600;
    color: #4a5568;
    transition: color 0.2s;
  }

  .signal-btn.active .btn-text {
    color: white;
  }
</style>
