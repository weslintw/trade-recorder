<script>
  import { createEventDispatcher } from 'svelte';
  import { imagesAPI } from '../../lib/api';
  const dispatch = createEventDispatcher();

  export let formData = {};
  export let entry_signals = []; // Explicit binding for better reactivity
  export let signalImagesCache = {};

  const expertSignalsLong = ['向下蘇美', '起漲靠山', '雙柱', '倚天', '攻城池上'];
  const expertSignalsShort = ['起跌靠山', '君臨城下', '雙塔', '向上蘇美', '雷霆'];

  $: expertSignals = formData.side === 'long' ? expertSignalsLong : expertSignalsShort;

  // Initialize cache from existing signals if cache is empty
  $: if (entry_signals && Array.isArray(entry_signals)) {
    if (Object.keys(signalImagesCache).length === 0) {
      entry_signals.forEach(signal => {
        if (signal.name && (signal.image || signal.originalImage)) {
          signalImagesCache[signal.name] = {
            image: signal.image || '',
            originalImage: signal.originalImage || '',
          };
        }
      });
    }
  }

  function getSignalData(signalName) {
    if (!entry_signals) return { name: signalName, image: '' };
    const existing = entry_signals.find(s =>
      typeof s === 'string' ? s === signalName : s.name === signalName
    );
    if (existing) {
      if (typeof existing === 'string') {
        return { name: signalName, image: '' };
      }
      return existing;
    }
    return { name: signalName, image: '' };
  }

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
      // Deselect: save to cache
      const signal = entry_signals[index];
      if (signal.image || signal.originalImage) {
        signalImagesCache[signalName] = {
          image: signal.image || '',
          originalImage: signal.originalImage || '',
        };
      }
      entry_signals = entry_signals.filter((_, i) => i !== index);
    } else {
      // Select: restore from cache
      const cachedImages = signalImagesCache[signalName];
      if (cachedImages) {
        entry_signals = [
          ...entry_signals,
          {
            name: signalName,
            image: cachedImages.image,
            originalImage: cachedImages.originalImage,
          },
        ];
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
    // Svelte reactivity triggers on assignment
  }

  function removeSignalImage(signal) {
    const index = entry_signals.findIndex(s => s.name === signal.name);
    if (index >= 0) {
      // Create a copy to trigger update
      let newSignals = [...entry_signals];
      newSignals[index] = {
        ...newSignals[index],
        image: '',
        originalImage: '',
      };
      entry_signals = newSignals;
      // Also clear from cache
      delete signalImagesCache[signal.name];
    }
  }

  // 處理圖片顯示 Helper
  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) return src;
    return imagesAPI.getUrl(src);
  }

  async function handleSignalImagePaste(event, signalName) {
    const items = (event.clipboardData || event.originalEvent.clipboardData).items;
    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = item.getAsFile();

        try {
          const formDataToUpload = new FormData();
          formDataToUpload.append('image', file);
          formDataToUpload.append('symbol', formData.symbol || 'trade');

          const response = await imagesAPI.upload(formDataToUpload);
          const imageUrl = response.data.path;
          const imageSize = response.data.size;

          if (!entry_signals) entry_signals = [];
          const index = entry_signals.findIndex(s =>
            typeof s === 'string' ? s === signalName : s.name === signalName
          );

          if (index >= 0) {
            let newSignals = [...entry_signals];
            const newSignal =
              typeof newSignals[index] === 'string'
                ? { name: signalName, images: [{ image: imageUrl, originalImage: imageUrl, size: imageSize }] }
                : {
                    ...newSignals[index],
                    images: [
                      ...(newSignals[index].images || (newSignals[index].image ? [{ image: newSignals[index].image, originalImage: newSignals[index].originalImage || newSignals[index].image, size: newSignals[index].size }] : [])),
                      { image: imageUrl, originalImage: imageUrl, size: imageSize }
                    ].filter((img, i, self) => img.image && self.findIndex(t => t.image === img.image) === i)
                  };
            newSignals[index] = newSignal;
            entry_signals = newSignals;

            signalImagesCache[signalName] = {
              images: newSignal.images,
            };
          } else {
            const newSignal = {
              name: signalName,
              images: [{ image: imageUrl, originalImage: imageUrl, size: imageSize }]
            };
            entry_signals = [...entry_signals, newSignal];
            signalImagesCache[signalName] = {
              images: newSignal.images,
            };
          }
        } catch (error) {
          console.error('圖片貼上失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }
</script>

<div class="signals-card-grid">
  {#each expertSignals as signal}
    {@const isSelected = isSignalSelected(signal)}
    {@const signalData = getSignalData(signal)}
    <div
      class="signal-card"
      class:selected={isSelected}
      tabindex="0"
      role="button"
      on:paste={e => handleSignalImagePaste(e, signal)}
      on:click={e => {
        if (!e.target.closest('.signal-checkbox') && !e.target.closest('.signal-image-preview')) {
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

      {#if isSelected}
        <div class="signal-card-header">
          <span class="signal-card-title">達人觀察圖</span>
        </div>
        
        {@const images = signalData.images || (signalData.image ? [{image: signalData.image, originalImage: signalData.originalImage, size: signalData.size}] : [])}
        {@const slots = [...images, null]}

        <div class="strategy-images-grid">
          {#each slots as imageData, imgIndex}
            <div 
              class="image-slot"
              class:empty={!imageData}
              on:paste|stopPropagation={e => handleSignalImagePaste(e, signal)}
            >
              {#if imageData?.image}
                <div class="signal-image-preview">
                  <img
                    src={getImageUrl(imageData.image)}
                    alt={signal}
                    on:click={e => {
                      e.stopPropagation();
                      dispatch('enlarge', {
                        image: imageData.image,
                        title: `達人觀察圖 (${signal})`,
                        context: {
                          type: 'signal',
                          name: signalData.name,
                          index: imgIndex,
                          originalImage: imageData.originalImage,
                        },
                      });
                    }}
                  />
                  <button
                    type="button"
                    class="remove-signal-image"
                    on:click={e => {
                      e.stopPropagation();
                      // Remove specific image
                      let newSignals = [...entry_signals];
                      const sIdx = newSignals.findIndex(s => s.name === signal);
                      if (sIdx >= 0) {
                        const currentImages = newSignals[sIdx].images || (newSignals[sIdx].image ? [{image: newSignals[sIdx].image, originalImage: newSignals[sIdx].originalImage, size: newSignals[sIdx].size}] : []);
                        newSignals[sIdx].images = currentImages.filter((_, i) => i !== imgIndex);
                        newSignals[sIdx].image = ''; // Clear legacy field
                        entry_signals = newSignals;
                      }
                    }}
                  >
                    ×
                  </button>
                </div>
              {:else}
                <div class="signal-image-placeholder compact">
                  <span class="placeholder-text">Ctrl+V</span>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>

<style>
  .signals-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 1rem;
  }

  .signal-card {
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    padding: 0.75rem;
    cursor: auto !important;
    transition: all 0.2s ease;
    background: white;
    display: flex;
    flex-direction: column;
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
    margin-bottom: 0.5rem;
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

  .signal-card-header {
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid #e2e8f0;
    margin-bottom: 0.5rem;
  }

  .signal-card-title {
    font-size: 0.8rem;
    font-weight: 700;
    color: #718096;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .strategy-images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    gap: 0.5rem;
    width: 100%;
  }

  .image-slot {
    aspect-ratio: 1;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    position: relative;
    cursor: pointer;
  }

  .image-slot.empty {
    border: 1px dashed #cbd5e0;
  }

  .image-slot:hover {
    border-color: #667eea;
  }

  .signal-image-preview {
    position: relative;
    width: 100%;
    height: 100%;
  }

  .signal-image-preview img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
    background: white;
    cursor: zoom-in;
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
    transition: all 0.2s ease;
    padding: 0;
  }

  .remove-signal-image:hover {
    background: rgba(239, 68, 68, 0.9);
    transform: scale(1.1);
  }

  .signal-image-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  .signal-image-placeholder.compact {
    padding: 0.5rem;
  }

  .placeholder-text {
    font-size: 0.75rem;
    color: #a0aec0;
    pointer-events: none;
  }
</style>
