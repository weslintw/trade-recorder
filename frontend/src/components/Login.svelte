<script>
    import { login, register } from '../lib/auth';
    import { onMount } from 'svelte';
    import { fade, fly } from 'svelte/transition';

    export let onLoginSuccess = () => {};

    let username = '';
    let password = '';
    let passwordVisible = false;
    let isRegister = false;
    let loading = false;
    let error = '';
    const buildTime = __BUILD_TIME__;

    async function handleSubmit() {
        if (!username || !password) {
            error = '請輸入使用者名稱和密碼';
            return;
        }

        loading = true;
        error = '';

        const result = isRegister 
            ? await register(username, password)
            : await login(username, password);

        loading = false;
        
        if (result.success) {
            onLoginSuccess();
        } else {
            error = result.error;
        }
    }

    function toggleMode() {
        isRegister = !isRegister;
        error = '';
    }
</script>

<div class="login-container" in:fade={{ duration: 300 }}>
    <div class="login-card" in:fly={{ y: 20, duration: 500 }}>
        <div class="logo-area">
            <div class="login-logo-container">
                <img src="/logo.png" alt="Trade Time Machine Logo" class="login-logo-img" />
            </div>
            <span class="app-version-tag">{buildTime}</span>
            <p class="subtitle">{isRegister ? '加入我們，優化您的交易流程' : '歡迎回來，紀錄您的每一步成長'}</p>
        </div>

        {#if error}
            <div class="error-msg" transition:fade>
                {error}
            </div>
        {/if}

        <form on:submit|preventDefault={handleSubmit}>
            <div class="form-group">
                <label for="username">使用者名稱</label>
                <div class="input-wrapper">
                    <span class="input-icon">👤</span>
                    <input 
                        id="username"
                        type="text" 
                        bind:value={username} 
                        placeholder="請輸入使用者名稱"
                        required
                    />
                </div>
            </div>

            <div class="form-group">
                <label for="password">密碼</label>
                <div class="input-wrapper">
                    <span class="input-icon">🔒</span>
                    {#if passwordVisible}
                        <input 
                            id="password-text"
                            type="text" 
                            bind:value={password} 
                            placeholder="請輸入密碼"
                            required
                        />
                    {:else}
                        <input 
                            id="password"
                            type="password" 
                            bind:value={password} 
                            placeholder="請輸入密碼"
                            required
                        />
                    {/if}
                    <button 
                        type="button" 
                        class="toggle-password"
                        on:mousedown|preventDefault={() => passwordVisible = true}
                        on:mouseup|preventDefault={() => passwordVisible = false}
                        on:mouseleave|preventDefault={() => passwordVisible = false}
                        on:touchstart|preventDefault={() => passwordVisible = true}
                        on:touchend|preventDefault={() => passwordVisible = false}
                        tabindex="-1"
                        title="查看密碼"
                    >
                        {passwordVisible ? '👁️' : '👁️‍🗨️'}
                    </button>
                </div>
            </div>

            <button type="submit" class="submit-btn" disabled={loading}>
                {#if loading}
                    <span class="spinner"></span>
                    處理中...
                {:else}
                    {isRegister ? '註冊並登入' : '登入系統'}
                {/if}
            </button>
        </form>

        <div class="toggle-mode">
            <span>{isRegister ? '已經有帳號了？' : '還沒有帳號？'}</span>
            <button type="button" on:click={toggleMode}>
                {isRegister ? '立即登入' : '免費註冊'}
            </button>
        </div>

        <div class="hint">
            <p>💡 提示：註冊成功後的預設帳號即為管理員帳號。</p>
        </div>
    </div>
</div>

<style>
    .login-container {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: radial-gradient(circle at top right, #f8fafc, #e2e8f0);
        z-index: 9999;
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
    }

    .login-card {
        width: 100%;
        max-width: 500px;
        padding: 3rem;
        background: white;
        border-radius: 1.5rem;
        box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
        border: 1px solid rgba(0, 0, 0, 0.05);
    }

    .logo-area {
        text-align: center;
        margin-bottom: 2.5rem;
        position: relative; /* 為了版本標籤定位 */
    }

    .login-logo-container {
        width: 100%;
        height: 180px; /* 從 140 加高到 180，增加顯示範圍 */
        margin: -1.5rem auto 1rem;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
    }

    .login-logo-img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transform: scale(1.2); /* 從 1.6 縮小到 1.2，避免文字被切掉 */
    }

    .subtitle {
        color: #64748b;
        font-size: 0.9375rem;
    }

    .app-version-tag {
        position: absolute;
        top: 0;
        right: 0;
        font-size: 0.7rem;
        color: #94a3b8;
        background: #f1f5f9;
        padding: 0.1rem 0.4rem;
        border-radius: 4px;
        font-weight: 600;
        z-index: 10;
    }

    .form-group {
        margin-bottom: 1.5rem;
    }

    label {
        display: block;
        font-size: 0.875rem;
        font-weight: 600;
        color: #475569;
        margin-bottom: 0.5rem;
    }

    .input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
    }

    .input-icon {
        position: absolute;
        left: 1rem;
        color: #94a3b8;
        font-size: 1.1rem;
    }

    input {
        width: 100%;
        padding: 0.75rem 1rem 0.75rem 3rem;
        border: 1.5px solid #e2e8f0;
        border-radius: 0.75rem;
        font-size: 1rem;
        transition: all 0.2s;
        color: #1e293b;
        outline: none;
    }

    input:focus {
        border-color: #3b82f6;
        box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.1);
    }

    .toggle-password {
        position: absolute;
        right: 1rem;
        background: none;
        border: none;
        padding: 0;
        margin: 0;
        cursor: pointer;
        font-size: 1.2rem;
        color: #94a3b8;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: color 0.2s;
        user-select: none;
        -webkit-user-select: none;
    }

    .toggle-password:hover {
        color: #64748b;
    }

    .error-msg {
        background-color: #fef2f2;
        border: 1px solid #fee2e2;
        color: #dc2626;
        padding: 0.75rem 1rem;
        border-radius: 0.75rem;
        font-size: 0.875rem;
        margin-bottom: 1.5rem;
        text-align: center;
    }

    .submit-btn {
        width: 100%;
        padding: 0.875rem;
        background: linear-gradient(to right, #3b82f6, #2563eb);
        color: white;
        border: none;
        border-radius: 0.75rem;
        font-size: 1rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        margin-top: 2rem;
    }

    .submit-btn:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 10px 15px -3px rgba(37, 99, 235, 0.4);
    }

    .submit-btn:active:not(:disabled) {
        transform: translateY(0);
    }

    .submit-btn:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .toggle-mode {
        text-align: center;
        margin-top: 1.5rem;
        font-size: 0.875rem;
        color: #64748b;
    }

    .toggle-mode button {
        background: none;
        border: none;
        color: #3b82f6;
        font-weight: 600;
        cursor: pointer;
        padding: 0 0.25rem;
    }

    .toggle-mode button:hover {
        text-decoration: underline;
    }

    .hint {
        margin-top: 2rem;
        padding-top: 1.5rem;
        border-top: 1px solid #f1f5f9;
        font-size: 0.75rem;
        color: #94a3b8;
        text-align: center;
    }

    .spinner {
        width: 18px;
        height: 18px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-radius: 50%;
        border-top-color: white;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }
</style>
