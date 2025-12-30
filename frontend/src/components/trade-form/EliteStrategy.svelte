<script>
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  export let formData = {};
  export let patternImagesCache = {};

  const eliteChecklist = [
    { id: 'item_trend', label: '順勢' },
    { id: 'item_zone_s_d', label: 'Zone (S/D)' },
    { id: 'item_f_b_break', label: '假突破 or 真突破/回踩' },
    { id: 'item_space', label: '空間' },
    { id: 'item_signal', label: '訊號' },
  ];

  const entryPatterns = [
    '破底翻',
    '破底翻(破)',
    '破底翻(破+回測)',
    '優質供給/需求區',
    '優質供給/需求區(破)',
    '優質供給/需求區(破+回測)',
    '2B',
    '2B(破)',
    '2B(破+回測)',
  ];

  // Initialize cache if needed (similar to signals)
  $: if (formData.entry_pattern && Array.isArray(formData.entry_pattern)) {
      if (Object.keys(patternImagesCache).length === 0) {
          formData.entry_pattern.forEach(pattern => {
              if (pattern.name && pattern.image) {
                  patternImagesCache[pattern.name] = {
                      image: pattern.image,
                      originalImage: pattern.originalImage || pattern.image
                  };
              }
          });
      }
  }

  function togglePattern(patternName) {
    const index = formData.entry_pattern.findIndex(p => p.name === patternName);
    if (index >= 0) {
      // Remove
      const pattern = formData.entry_pattern[index];
      if (pattern.image) {
        patternImagesCache[patternName] = { 
            image: pattern.image, 
            originalImage: pattern.originalImage || pattern.image 
        };
      }
      formData.entry_pattern = formData.entry_pattern.filter(p => p.name !== patternName);
    } else {
      // Add
      const cached = patternImagesCache[patternName];
      if (cached) {
          formData.entry_pattern = [...formData.entry_pattern, { 
              name: patternName, 
              image: cached.image,
              originalImage: cached.originalImage
          }];
      } else {
          formData.entry_pattern = [...formData.entry_pattern, { name: patternName, image: '' }];
      }
    }
  }

  function enlargeImage(image, title, context) {
    dispatch('enlarge', { image, title, context });
  }

  function handlePatternImagePaste(e, pattern) {
    const items = (e.clipboardData || e.originalEvent.clipboardData).items;
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        e.preventDefault();
        const file = item.getAsFile();
        const reader = new FileReader();
        reader.onload = event => {
          const imgData = event.target.result;
          pattern.image = imgData;
          pattern.originalImage = imgData;
          // Sync to cache
          patternImagesCache[pattern.name] = {
            image: imgData,
            originalImage: imgData,
          };
          formData.entry_pattern = formData.entry_pattern; // Trigger reactivity
        };
        reader.readAsDataURL(file);
        break;
      }
    }
  }

  function removePatternImage(pattern) {
      pattern.image = '';
      pattern.originalImage = '';
      // Clear from cache too? The original code did remove from cache when manually removing image?
      // Yes: lines around 2640 in original code
      delete patternImagesCache[pattern.name];
      formData.entry_pattern = formData.entry_pattern;
  }
</script>

<div class="checklist-section">
  <label class="checklist-label">菁英檢查清單：</label>
  <div class="checklist-items">
    {#each eliteChecklist as item}
      <label class="checkbox-item">
        <input
          type="checkbox"
          checked={formData.entry_checklist[item.id] || false}
          on:change={e => {
            formData.entry_checklist = {
              ...formData.entry_checklist,
              [item.id]: e.target.checked,
            };
          }}
        />
        <span class="checkbox-label">{item.label}</span>
      </label>
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
        on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && togglePattern(patternName)}
      >
        <span class="pattern-name">{patternName}</span>
      </div>
    {/each}
  </div>

  {#if formData.entry_pattern.length > 0}
    <div class="pattern-cards-grid">
      {#each formData.entry_pattern as pattern}
        <div
          class="pattern-image-card"
          on:paste={e => handlePatternImagePaste(e, pattern)}
        >
          <div class="pattern-card-header">
            <span class="pattern-card-title">{pattern.name}</span>
          </div>
          <div class="pattern-card-body">
            {#if pattern.image}
               <div
                class="pattern-image-preview"
                role="button"
                tabindex="0"
                on:click={() =>
                  enlargeImage(pattern.image, pattern.name + ' 樣態圖', {
                    type: 'pattern',
                    key: pattern.name,
                  })}
                on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && 
                  enlargeImage(pattern.image, pattern.name + ' 樣態圖', {
                    type: 'pattern',
                    key: pattern.name,
                  })}
              >
                <img src={pattern.image} alt={pattern.name} />
                <button
                  type="button"
                  class="remove-pattern-image"
                  on:click|stopPropagation={() => removePatternImage(pattern)}
                >
                  ×
                </button>
              </div>
            {:else}
              <div class="signal-image-placeholder">
                <span class="placeholder-text">點擊此處或按 Ctrl+V 貼上圖片</span>
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .checklist-section {
    margin-top: 1rem;
    padding: 1rem;
    background: #f8fafc;
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
    flex-wrap: wrap;
    gap: 1rem;
  }

  .checkbox-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .checkbox-item input[type='checkbox'] {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #667eea;
  }

  .checkbox-label {
    font-size: 0.9rem;
    color: #2d3748;
    user-select: none;
  }

  /* 進場樣態 */
  .entry-pattern-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
  }

  .entry-pattern-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
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
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    user-select: none;
  }

  .pattern-option:hover {
    border-color: #667eea;
    background: #f7fafc;
  }

  .pattern-option.active {
    border-color: #667eea;
    background: #667eea;
  }

  .pattern-name {
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
  }

  .pattern-option.active .pattern-name {
    color: white;
  }

  .pattern-cards-grid {
    margin-top: 1.5rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
  }

  .pattern-image-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transition: all 0.2s ease;
  }

  .pattern-image-card:hover {
    border-color: #667eea;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  }

  .pattern-card-header {
    padding: 0.5rem 0.75rem;
    background: #edf2f7;
    border-bottom: 1px solid #e2e8f0;
  }

  .pattern-card-title {
    font-size: 0.85rem;
    font-weight: 700;
    color: #4a5568;
  }

  .pattern-card-body {
    padding: 0.75rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 120px;
    position: relative;
    justify-content: center;
    align-items: center;
  }

  .pattern-image-preview {
    width: 100%;
    cursor: zoom-in;
    border-radius: 6px;
    overflow: hidden;
    position: relative;
  }

  .pattern-image-preview img {
    width: 100%;
    height: 120px;
    object-fit: cover;
    display: block;
  }

  .remove-pattern-image {
    position: absolute;
    top: 4px;
    right: 4px;
    width: 20px;
    height: 20px;
    background: rgba(0, 0, 0, 0.5);
    color: white;
    border: none;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    cursor: pointer;
    font-size: 0.9rem;
    transition: background 0.2s;
    line-height: 1;
    padding-bottom: 2px;
  }

  .remove-pattern-image:hover {
    background: rgba(220, 38, 38, 0.9);
  }

  .signal-image-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  .placeholder-text {
    font-size: 0.75rem;
    color: #718096;
    pointer-events: none;
  }
</style>
