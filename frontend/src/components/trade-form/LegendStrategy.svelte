<script>
  import { createEventDispatcher } from 'svelte';
  import SignalGrid from './SignalGrid.svelte';
  import { imagesAPI } from '../../lib/api';

  const dispatch = createEventDispatcher();

  export let formData = {};
  export let signalImagesCache = {};

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

  function enlargeImage(image, title, context) {
    dispatch('enlarge', { image, title, context });
  }

  // 處理圖片顯示 Helper
  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) return src;
    return imagesAPI.getUrl(src);
  }

  // Handle King/Queen Image
  async function handleLegendKingImagePaste(e) {
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

          formData.legend_king_image = imageUrl;
          formData.legend_king_image_original = imageUrl;
          formData.legend_king_image_size = imageSize;
          formData = formData;
        } catch (error) {
          console.error('王后圖上傳失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }

  function removeLegendKingImage() {
    formData.legend_king_image = '';
    formData.legend_king_image_original = '';
    formData = formData;
  }

  // Handle HTF Image
  async function handleLegendHTFImagePaste(e) {
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

          formData.legend_htf_image = imageUrl;
          formData.legend_htf_image_original = imageUrl;
          formData.legend_htf_image_size = imageSize;
          formData = formData;
        } catch (error) {
          console.error('大時區圖上傳失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }

  function removeLegendHTFImage() {
    formData.legend_htf_image = '';
    formData.legend_htf_image_original = '';
    formData = formData;
  }

  // Handle Strategy Images (Multiple Legend Images)
  // Initialize legend_images array if not exists
  $: if (!formData.legend_images || !Array.isArray(formData.legend_images)) {
    formData.legend_images = [];
  }

  async function handleStrategyImagePaste(e, index) {
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

          // Update the specific index
          if (!formData.legend_images) {
            formData.legend_images = [];
          }
          
          // Create a copy of the array
          const newImages = [...formData.legend_images];
          newImages[index] = {
            image: imageUrl,
            originalImage: imageUrl,
            size: imageSize
          };
          
          formData.legend_images = newImages;
          formData = formData;
        } catch (error) {
          console.error('傳奇策略圖上傳失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }

  function removeStrategyImage(index) {
    if (formData.legend_images && formData.legend_images[index]) {
      const newImages = formData.legend_images.filter((_, i) => i !== index);
      formData.legend_images = newImages;
      formData = formData;
    }
  }

  // Calculate how many image slots to show (always show at least one empty slot)
  $: imageSlots = formData.legend_images && formData.legend_images.length > 0 
    ? [...formData.legend_images, null] // Add one empty slot at the end
    : [null]; // Show one empty slot if no images
</script>

<div class="checklist-section">
  <label class="checklist-label">傳奇檢查清單：</label>
  <div class="checklist-items">
    {#each legendChecklist as item}
      <label class="checkbox-item">
        <input
          type="checkbox"
          checked={formData.entry_checklist[item.id] || false}
          on:change={e => {
            formData.entry_checklist = {
              ...formData.entry_checklist,
              [item.id]: e.target.checked,
            };
            // Clean up if unchecked? Usually better to keep data in case of accidental uncheck.
          }}
        />
        <span class="checkbox-label">{item.label}</span>
      </label>
    {/each}
  </div>
</div>

<!-- 王者回調 Section -->
{#if formData.entry_checklist['item_618_786']}
  <div class="signals-section nested king-section">
    <label class="signals-label">王者出現回調618或786 - 請選擇時區並貼圖：</label>

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

    <div
      class="signal-card htf-image-card"
      tabindex="0"
      on:paste={handleLegendKingImagePaste}
      on:click={() => {
        if (formData.legend_king_image) {
          enlargeImage(
            formData.legend_king_image,
            `王者回調 (${formData.legend_king_htf || '未選擇'})`,
            { type: 'legend_king', originalImage: formData.legend_king_image_original || formData.legend_king_image }
          );
        }
      }}
    >
      {#if formData.legend_king_image}
        <div class="signal-image-preview">
          <img src={getImageUrl(formData.legend_king_image)} alt="王者回調截圖" />
          <button
            type="button"
            class="remove-signal-image"
            on:click={e => {
              e.stopPropagation();
              removeLegendKingImage();
            }}
          >
            ×
          </button>
        </div>
      {:else}
        <div class="signal-image-placeholder">
          <span class="placeholder-text">點擊此處並按 Ctrl+V 貼上王者回調截圖</span>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- HTF Break Section -->
{#if formData.entry_checklist['item_che']}
  <div class="signals-section nested htf-section">
    <label class="signals-label">大時區破"測"破 - 請選擇大時區並貼圖：</label>

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

    <div
      class="signal-card htf-image-card"
      tabindex="0"
      on:paste={handleLegendHTFImagePaste}
      on:click={() => {
        if (formData.legend_htf_image) {
          enlargeImage(
            formData.legend_htf_image,
            `大時區破"測"破 (${formData.legend_htf || '未選擇'})`,
            { type: 'legend_htf', originalImage: formData.legend_htf_image_original || formData.legend_htf_image }
          );
        }
      }}
    >
      {#if formData.legend_htf_image}
        <div class="signal-image-preview">
          <img src={getImageUrl(formData.legend_htf_image)} alt="大時區截圖" />
          <button
            type="button"
            class="remove-signal-image"
            on:click={e => {
              e.stopPropagation();
              removeLegendHTFImage();
            }}
          >
            ×
          </button>
        </div>
      {:else}
        <div class="signal-image-placeholder">
          <span class="placeholder-text">點擊此處並按 Ctrl+V 貼上大時區截圖</span>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- DE (ABCDE/Signal) Section -->
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

    <SignalGrid bind:formData bind:signalImagesCache on:enlarge />
  </div>
{/if}

<!-- General Legend Images Section (Multiple Images) -->
<div class="signals-section">
  <label class="signals-label">傳奇觀察圖 (Ctrl+V 貼上)：</label>
  <div class="legend-images-grid">
    {#each imageSlots as imageData, index}
      <div
        class="signal-card legend-image-card"
        tabindex="0"
        role="button"
        on:paste={e => handleStrategyImagePaste(e, index)}
        on:click={() => {
          if (imageData?.image) {
            enlargeImage(imageData.image, `傳奇觀察圖 ${index + 1}`, { 
              type: 'legend_images', 
              index, 
              originalImage: imageData.originalImage || imageData.image 
            });
          }
        }}
        on:keydown={e => {
          if (e.key === 'Enter' || e.key === ' ') {
            if (imageData?.image) {
              enlargeImage(imageData.image, `傳奇觀察圖 ${index + 1}`, { 
              type: 'legend_images', 
              index, 
              originalImage: imageData.originalImage || imageData.image 
            });
            }
          }
        }}
      >
        {#if imageData?.image}
          <div class="signal-image-preview">
            <img src={getImageUrl(imageData.image)} alt={`傳奇觀察圖 ${index + 1}`} />
            <button
              type="button"
              class="remove-signal-image"
              on:click={e => {
                e.stopPropagation();
                removeStrategyImage(index);
              }}
            >
              ×
            </button>
          </div>
        {:else}
          <div class="signal-image-placeholder">
            <span class="placeholder-text">點擊此處並按 Ctrl+V 貼上傳奇觀察圖</span>
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

  /* Signals Section Styles (reused) */
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
    padding: 1rem 0.5rem; /* Compact padding */
  }

  .signals-label {
    display: block;
    font-weight: 600;
    margin-bottom: 1rem;
    color: #4a5568;
    font-size: 0.95rem;
  }

  /* HTF Selector Row */
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
    background: #805ad5; /* Purple for Legend HTF */
    color: white;
    border-color: #805ad5;
  }

  /* Image Cards */
  .legend-images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1rem;
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

  .htf-image-card,
  .legend-image-card {
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

  .placeholder-text {
    font-size: 0.9rem;
  }
</style>
