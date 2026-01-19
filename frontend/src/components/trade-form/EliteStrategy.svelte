<script>
  import { createEventDispatcher } from 'svelte';
  import { imagesAPI } from '../../lib/api';
  const dispatch = createEventDispatcher();

  export let formData = {};
  export let patternImagesCache = {};

  const eliteChecklist = [
    { id: 'trend_line', label: '破趨勢線了嗎?' },
    { id: 'price_level', label: '破價位了嗎?' },
    { id: 'impulse_wave', label: '有驅動浪了嗎?' },
    { id: 'high_low', label: '不過高低了嗎?' },
    { id: 'sentiment', label: '情緒轉換了嗎?' },
  ];

  const entryPatterns = ['甲', '乙', '丙', '丁', '大Leading', '小Leading'];

  // Initialize cache if needed (similar to signals)
  $: if (formData.entry_pattern && Array.isArray(formData.entry_pattern)) {
    if (Object.keys(patternImagesCache).length === 0) {
      formData.entry_pattern.forEach(pattern => {
        if (pattern.name && pattern.image) {
          patternImagesCache[pattern.name] = {
            image: pattern.image,
            originalImage: pattern.originalImage || pattern.image,
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
          originalImage: pattern.originalImage || pattern.image,
        };
      }
      formData.entry_pattern = formData.entry_pattern.filter(p => p.name !== patternName);
    } else {
      // Add
      const cached = patternImagesCache[patternName];
      if (cached) {
        formData.entry_pattern = [
          ...formData.entry_pattern,
          {
            name: patternName,
            image: cached.image,
            originalImage: cached.originalImage,
          },
        ];
      } else {
        formData.entry_pattern = [...formData.entry_pattern, { name: patternName, image: '' }];
      }
    }
  }

  function enlargeImage(image, title, context) {
    dispatch('enlarge', { image, title, context });
  }

  // 處理圖片顯示 Helper
  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) return src;
    return imagesAPI.getUrl(src);
  }

  async function handlePatternImagePaste(e, pattern) {
    const items = (e.clipboardData || e.originalEvent.clipboardData).items;
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        e.preventDefault();
        const file = item.getAsFile();

        try {
          const formDataToUpload = new FormData();
          formDataToUpload.append('image', file);
          formDataToUpload.append('symbol', formData.symbol || 'trade');

          const response = await imagesAPI.upload(formDataToUpload);
          const imageUrl = response.data.path;
          const imageSize = response.data.size;

          pattern.image = imageUrl;
          pattern.originalImage = imageUrl;
          pattern.size = imageSize;
          // Sync to cache
          patternImagesCache[pattern.name] = {
            image: imageUrl,
            originalImage: imageUrl,
            size: imageSize
          };
          formData.entry_pattern = formData.entry_pattern; // Trigger reactivity
          formData = formData;
        } catch (error) {
          console.error('圖片貼上失敗:', error);
          alert('圖片處理失敗');
        }
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
  async function handleEliteImagePaste(e, index) {
    const items = (e.clipboardData || e.originalEvent.clipboardData).items;
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        e.preventDefault();
        const file = item.getAsFile();

        try {
          const formDataToUpload = new FormData();
          formDataToUpload.append('image', file);
          formDataToUpload.append('symbol', formData.symbol || 'trade');

          const response = await imagesAPI.upload(formDataToUpload);
          const imageUrl = response.data.path;
          const imageSize = response.data.size;

          if (!formData.elite_images) {
            formData.elite_images = [];
          }
          
          const newImages = [...formData.elite_images];
          newImages[index] = {
            image: imageUrl,
            originalImage: imageUrl,
            size: imageSize
          };
          
          formData.elite_images = newImages;
          formData = formData;
        } catch (error) {
          console.error('菁英觀察圖上傳失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }

  function removeEliteImage(index) {
    if (formData.elite_images && formData.elite_images[index]) {
      const newImages = formData.elite_images.filter((_, i) => i !== index);
      formData.elite_images = newImages;
      formData = formData;
    }
  }

  // Calculate how many image slots to show (always show at least one empty slot)
  $: eliteImageSlots = formData.elite_images && formData.elite_images.length > 0 
    ? [...formData.elite_images, null] 
    : [null];
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
        on:keydown={e => (e.key === 'Enter' || e.key === ' ') && togglePattern(patternName)}
      >
        <span class="pattern-name">{patternName}</span>
      </div>
    {/each}
  </div>

  {#if formData.entry_pattern.length > 0}
    <div class="pattern-cards-grid">
      {#each formData.entry_pattern as pattern}
        <div class="pattern-image-card" on:paste={e => handlePatternImagePaste(e, pattern)}>
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
                on:keydown={e =>
                  (e.key === 'Enter' || e.key === ' ') &&
                  enlargeImage(pattern.image, pattern.name + ' 樣態圖', {
                    type: 'pattern',
                    key: pattern.name,
                  })}
              >
                <img src={getImageUrl(pattern.image)} alt={pattern.name} />
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

<!-- 菁英觀察圖 (多圖連貼，原 entry_strategy_image 位置) -->
<div class="observation-section">
  <label class="section-label">菁英觀察圖 (Ctrl+V 貼上)：</label>
  <div class="strategy-images-grid">
    {#each eliteImageSlots as imageData, index}
      <div
        class="signal-card elite-image-card"
        tabindex="0"
        role="button"
        on:paste={e => handleEliteImagePaste(e, index)}
        on:click={() => {
          if (imageData?.image) {
            dispatch('enlarge', { 
              image: imageData.image, 
              title: `菁英觀察圖 ${index + 1}`, 
              context: { type: 'elite_strategy', index }
            });
          }
        }}
        on:keydown={e => {
          if (e.key === 'Enter' || e.key === ' ') {
            if (imageData?.image) {
              dispatch('enlarge', { 
                image: imageData.image, 
                title: `菁英觀察圖 ${index + 1}`, 
                context: { type: 'elite_strategy', index }
              });
            }
          }
        }}
      >
        {#if imageData?.image}
          <div class="signal-image-preview">
            <img src={getImageUrl(imageData.image)} alt={`菁英觀察圖 ${index + 1}`} />
            <button
              type="button"
              class="remove-signal-image"
              on:click={e => {
                e.stopPropagation();
                removeEliteImage(index);
              }}
            >
              ×
            </button>
          </div>
        {:else}
          <div class="signal-image-placeholder">
            <span class="placeholder-text">點擊此處並按 Ctrl+V 貼上菁英觀察圖</span>
          </div>
        {/if}
      </div>
    {/each}
  </div>
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

  .strategy-images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1rem;
    margin-top: 0.5rem;
  }

  .observation-section {
    margin-top: 1.5rem;
    padding: 1rem;
    background: #fdfdfd;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
  }

  .section-label {
    display: block;
    font-size: 0.95rem;
    font-weight: 600;
    color: #4a5568;
    margin-bottom: 0.75rem;
  }

  .signal-card {
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    padding: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    background: white;
  }

  .signal-card:hover {
    border-color: #cbd5e0;
  }

  .elite-image-card {
    min-height: 150px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .signal-image-preview {
    width: 100%;
    position: relative;
    border-radius: 8px;
    overflow: hidden;
  }

  .signal-image-preview img {
    width: 100%;
    height: auto;
    max-height: 300px;
    display: block;
    object-fit: contain;
  }

  .remove-signal-image {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 24px;
    height: 24px;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 1.2rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .remove-signal-image:hover {
    background: #ef4444;
  }

  .signal-image-placeholder {
    padding: 2rem;
    text-align: center;
    color: #718096;
    border: 2px dashed #e2e8f0;
    border-radius: 8px;
    width: 100%;
  }
</style>
