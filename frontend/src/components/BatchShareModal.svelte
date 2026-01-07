<script>
  import { createEventDispatcher } from 'svelte';
  import { sharesAPI } from '../lib/api';
  import { selectedAccountId } from '../lib/stores';

  export let show = false;
  
  const dispatch = createEventDispatcher();

  let loading = false;
  let shareToken = '';
  let copySuccess = false;

  $: if (show) {
    shareToken = '';
    copySuccess = false;
  }

  $: shareUrl = shareToken ? `${window.location.origin}/shared/${shareToken}` : '';

  function handleClose() {
    show = false;
    dispatch('close');
  }

  async function handleFullShare() {
    if (!$selectedAccountId) return;
    loading = true;
    try {
      const res = await sharesAPI.create({
        resource_type: 'account',
        resource_id: Number($selectedAccountId),
        share_type: 'public'
      });
      shareToken = res.data.token;
    } catch (e) {
      console.error(e);
      alert('建立分享失敗');
    } finally {
      loading = false;
    }
  }

  function handlePartialShare() {
    dispatch('startSelection');
    handleClose();
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(shareUrl).then(() => {
      copySuccess = true;
      setTimeout(() => (copySuccess = false), 2000);
    });
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={handleClose} role="presentation">
    <div class="modal card batch-share-modal">
      <div class="modal-header">
        <h2>進階分享功能</h2>
        <button class="close-btn" on:click={handleClose}>×</button>
      </div>

      {#if !shareToken}
        <div class="share-options">
          <div class="share-option-card" on:click={handleFullShare}>
            <div class="option-icon">🌐</div>
            <div class="option-info">
              <strong>全部分享</strong>
              <span>分享此交易帳號的所有內容 (唯讀)</span>
            </div>
          </div>

          <div class="share-option-card" on:click={handlePartialShare}>
            <div class="option-icon">✅</div>
            <div class="option-info">
              <strong>部分分享</strong>
              <span>手動勾選想要分享的規劃或交易</span>
            </div>
          </div>
        </div>
      {:else}
        <div class="share-result">
          <div class="success-icon">🔗</div>
          <h3>分享連結已產生</h3>
          <div class="url-box">
            <input type="text" value={shareUrl} readonly />
            <button class="btn btn-secondary" on:click={copyToClipboard}>
              {copySuccess ? '已複製!' : '複製'}
            </button>
          </div>
          <p class="share-tip">💡 取得連結的人僅能查看此帳號的內容，無法進行修改</p>
        </div>
      {/if}

      {#if loading}
        <div class="loading-overlay">
          <div class="loader"></div>
          <span>正在產生連結...</span>
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
    background: rgba(15, 23, 42, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
    backdrop-filter: blur(8px);
  }

  .batch-share-modal {
    width: 90%;
    max-width: 480px;
    padding: 2rem;
    position: relative;
    overflow: hidden;
    animation: modalScaleIn 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes modalScaleIn {
    from { opacity: 0; transform: scale(0.9); }
    to { opacity: 1; transform: scale(1); }
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  .modal-header h2 {
    font-size: 1.5rem;
    font-weight: 800;
    margin: 0;
    color: #1e293b;
  }

  .close-btn {
    background: #f1f5f9;
    border: none;
    font-size: 1.5rem;
    color: #64748b;
    cursor: pointer;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: #e2e8f0;
    color: #1e293b;
  }

  .share-options {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .share-option-card {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    padding: 1.25rem;
    background: #f8fafc;
    border: 2px solid #f1f5f9;
    border-radius: 16px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .share-option-card:hover {
    background: white;
    border-color: #6366f1;
    transform: translateY(-2px);
    box-shadow: 0 10px 15px -3px rgba(99, 102, 241, 0.1);
  }

  .option-icon {
    font-size: 2rem;
    background: white;
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
  }

  .option-info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .option-info strong {
    font-size: 1.1rem;
    color: #1e293b;
  }

  .option-info span {
    font-size: 0.875rem;
    color: #64748b;
  }

  .share-result {
    text-align: center;
    padding: 1rem 0;
  }

  .success-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .share-result h3 {
    margin-bottom: 1.5rem;
    color: #1e293b;
  }

  .url-box {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
    background: #f1f5f9;
    padding: 0.5rem;
    border-radius: 12px;
  }

  .url-box input {
    flex: 1;
    background: transparent;
    border: none;
    padding: 0.5rem 0.75rem;
    font-size: 0.9rem;
    color: #334155;
    outline: none;
  }

  .share-tip {
    font-size: 0.85rem;
    color: #64748b;
    margin: 0;
    line-height: 1.5;
  }

  .loading-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(255, 255, 255, 0.8);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    z-index: 10;
  }

  .loader {
    width: 40px;
    height: 40px;
    border: 4px solid #f3f3f3;
    border-top: 4px solid #6366f1;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }
</style>
