<script>
  import { createEventDispatcher } from 'svelte';
  import { imagesAPI } from '../../lib/api';
  import SignalGrid from './SignalGrid.svelte';
  const dispatch = createEventDispatcher();

  export let formData = {};
  export let signalImagesCache = {};

  async function handleExpertImagePaste(e, index) {
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

          if (!formData.expert_images) {
            formData.expert_images = [];
          }
          
          const newImages = [...formData.expert_images];
          newImages[index] = {
            image: imageUrl,
            originalImage: imageUrl,
            size: imageSize
          };
          
          formData.expert_images = newImages;
          formData = formData;
        } catch (error) {
          console.error('達人觀察圖上傳失敗:', error);
          alert('圖片處理失敗');
        }
        break;
      }
    }
  }

  function removeExpertImage(index) {
    if (formData.expert_images && formData.expert_images[index]) {
      const newImages = formData.expert_images.filter((_, i) => i !== index);
      formData.expert_images = newImages;
      formData = formData;
    }
  }

  // Calculate how many image slots to show (always show at least one empty slot)
  $: imageSlots = formData.expert_images && formData.expert_images.length > 0 
    ? [...formData.expert_images, null] 
    : [null];

  function enlargeImage(image, title, context) {
    dispatch('enlarge', { image, title, context });
  }

  function getImageUrl(src) {
    if (!src) return '';
    if (src.startsWith('data:') || src.startsWith('http')) return src;
    return imagesAPI.getUrl(src);
  }
</script>

<div class="signals-section">
  <label class="signals-label">選擇訊號（可多選）：</label>
  <SignalGrid bind:entry_signals={formData.entry_signals} bind:formData bind:signalImagesCache on:enlarge />
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
    font-size: 0.95rem;
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

  /* Image Cards and Grid */
  .strategy-images-grid {
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

  .expert-image-card {
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
