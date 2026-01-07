<script>
  import { sharesAPI } from '../lib/api';
  import { onMount } from 'svelte';

  export let show = false;
  export let resourceType = 'trade'; // 'trade' or 'plan'
  export let resourceId = null;
  export let onClose = () => {};

  let loading = false;
  let shareType = 'public'; // 'public' or 'specific'
  let shareToken = '';
  let copySuccess = false;

  $: if (show) {
    shareToken = '';
    copySuccess = false;
  }

  $: shareUrl = shareToken ? `${window.location.origin}/shared/${shareToken}` : '';

  async function handleCreateShare() {
    loading = true;
    try {
      const res = await sharesAPI.create({
        resource_type: resourceType,
        resource_id: Number(resourceId),
        share_type: shareType
      });
      shareToken = res.data.token;
    } catch (e) {
      console.error(e);
      alert('建立分享失敗');
    } finally {
      loading = false;
    }
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(shareUrl).then(() => {
      copySuccess = true;
      setTimeout(() => (copySuccess = false), 2000);
    });
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={onClose} role="presentation">
    <div class="modal card share-modal">
      <div class="modal-header">
        <h2>分享內容</h2>
        <button class="close-btn" on:click={onClose}>×</button>
      </div>

      <div class="share-options">
        <label class="radio-label">
          <input type="radio" bind:group={shareType} value="public" />
          <div class="option-content">
            <strong>公開連結</strong>
            <span>任何擁有連結的人都可以查看此頁面</span>
          </div>
        </label>
        
        <!-- 暫不實作特定帳號分享，因為這需要使用者搜尋功能 -->
      </div>

      {#if !shareToken}
        <button class="btn btn-primary btn-block" on:click={handleCreateShare} disabled={loading}>
          {loading ? '產生中...' : '產生分享連結'}
        </button>
      {:else}
        <div class="share-result">
          <div class="url-box">
            <input type="text" value={shareUrl} readonly />
            <button class="btn btn-secondary" on:click={copyToClipboard}>
              {copySuccess ? '已複製!' : '複製'}
            </button>
          </div>
          <p class="share-tip">💡 取得連結的人僅能查看，無法修改內容</p>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .share-modal {
    width: 90%;
    max-width: 500px;
    padding: 2rem;
    animation: modalIn 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  @keyframes modalIn {
    from { opacity: 0; transform: translateY(20px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 2rem;
    color: #94a3b8;
    cursor: pointer;
  }

  .share-options {
    margin-bottom: 1.5rem;
  }

  .radio-label {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    border: 2px solid #f1f5f9;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .radio-label:hover {
    background: #f8fafc;
  }

  .radio-label input:checked + .option-content {
    /* Styles when checked */
  }

  .radio-label:has(input:checked) {
    border-color: #6366f1;
    background: #f5f3ff;
  }

  .option-content {
    display: flex;
    flex-direction: column;
  }

  .option-content strong {
    font-size: 1rem;
    color: #1e293b;
  }

  .option-content span {
    font-size: 0.85rem;
    color: #64748b;
  }

  .share-result {
    margin-top: 1.5rem;
    background: #f8fafc;
    padding: 1rem;
    border-radius: 12px;
  }

  .url-box {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .url-box input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    font-size: 0.9rem;
    background: white;
  }

  .share-tip {
    font-size: 0.8rem;
    color: #64748b;
    margin: 0;
  }

  .btn-block {
    width: 100%;
  }
</style>
