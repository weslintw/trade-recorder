<script>
  import { authAPI } from '../lib/api';
  import { fade, scale } from 'svelte/transition';

  export let show = false;
  export let onClose = () => {};

  let oldPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let loading = false;
  let error = '';
  let success = '';

  async function handleSubmit() {
    if (!oldPassword || !newPassword || !confirmPassword) {
      error = '請填寫所有欄位';
      return;
    }
    if (newPassword !== confirmPassword) {
      error = '新密碼與確認密碼不符';
      return;
    }
    if (newPassword.length < 6) {
      error = '新密碼長度至少需 6 個字元';
      return;
    }

    loading = true;
    error = '';
    success = '';

    try {
      await authAPI.changePassword({
        old_password: oldPassword,
        new_password: newPassword
      });
      success = '密碼修改成功！';
      oldPassword = '';
      newPassword = '';
      confirmPassword = '';
      setTimeout(() => {
        onClose();
        success = '';
      }, 2000);
    } catch (e) {
      console.error(e);
      error = e.response?.data?.error || '修改密碼失敗，請檢查舊密碼是否正確';
    } finally {
      loading = false;
    }
  }

  function handleClose() {
    if (loading) return;
    error = '';
    success = '';
    oldPassword = '';
    newPassword = '';
    confirmPassword = '';
    onClose();
  }
</script>

{#if show}
  <div class="modal-overlay" on:click|self={handleClose} in:fade={{ duration: 200 }} out:fade={{ duration: 150 }}>
    <div class="modal-card" in:scale={{ start: 0.95, duration: 200 }} out:scale={{ start: 0.95, duration: 150 }}>
      <div class="modal-header">
        <h2>🔒 修改密碼</h2>
        <button class="close-btn" on:click={handleClose}>&times;</button>
      </div>

      <div class="modal-body">
        {#if error}
          <div class="alert error">{error}</div>
        {/if}
        {#if success}
          <div class="alert success">{success}</div>
        {/if}

        <div class="form-group">
          <label for="old-password">舊密碼</label>
          <input 
            id="old-password"
            type="password" 
            bind:value={oldPassword} 
            placeholder="請輸入目前密碼"
            class="form-control"
          />
        </div>

        <div class="form-group">
          <label for="new-password">新密碼</label>
          <input 
            id="new-password"
            type="password" 
            bind:value={newPassword} 
            placeholder="請輸入新密碼 (至少 6 字元)"
            class="form-control"
          />
        </div>

        <div class="form-group">
          <label for="confirm-password">確認新密碼</label>
          <input 
            id="confirm-password"
            type="password" 
            bind:value={confirmPassword} 
            placeholder="請再次輸入新密碼"
            class="form-control"
          />
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={handleClose} disabled={loading}>取消</button>
        <button class="btn btn-primary" on:click={handleSubmit} disabled={loading}>
          {#if loading}
            正在更新...
          {:else}
            確認修改
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .modal-card {
    background: white;
    width: 90%;
    max-width: 400px;
    border-radius: 12px;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
    overflow: hidden;
  }

  .modal-header {
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .modal-header h2 {
    font-size: 1.25rem;
    margin: 0;
    color: #1e293b;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    color: #94a3b8;
    cursor: pointer;
    line-height: 1;
  }

  .modal-body {
    padding: 1.5rem;
  }

  .alert {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-bottom: 1rem;
    font-size: 0.875rem;
    font-weight: 500;
  }

  .alert.error {
    background: #fef2f2;
    color: #dc2626;
    border: 1px solid #fee2e2;
  }

  .alert.success {
    background: #f0fdf4;
    color: #16a34a;
    border: 1px solid #dcfce7;
  }

  .form-group {
    margin-bottom: 1.25rem;
  }

  .form-group label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: #475569;
    margin-bottom: 0.5rem;
  }

  .form-control {
    width: 100%;
    padding: 0.625rem 0.875rem;
    border: 1.5px solid #e2e8f0;
    border-radius: 8px;
    font-size: 1rem;
    transition: all 0.2s;
  }

  .form-control:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .modal-footer {
    padding: 1rem 1.5rem;
    background: #f8fafc;
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
  }

  .btn {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    font-size: 0.875rem;
    border: 1px solid transparent;
    transition: all 0.2s;
  }

  .btn-primary {
    background: #3b82f6;
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: #2563eb;
  }

  .btn-secondary {
    background: white;
    border-color: #e2e8f0;
    color: #64748b;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #f1f5f9;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
