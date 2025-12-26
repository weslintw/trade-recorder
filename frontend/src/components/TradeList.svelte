<script>
  import { onMount } from 'svelte';
  import { Link, navigate } from 'svelte-routing';
  import { tradesAPI, tagsAPI, imagesAPI } from '../lib/api';

  let trades = [];
  let loading = true;
  let pagination = {
    page: 1,
    page_size: 20,
    total: 0
  };

  // 篩選條件
  let filters = {
    symbol: '',
    side: '',
    tag: '',
    start_date: '',
    end_date: ''
  };

  let allTags = [];
  let selectedImage = null;

  onMount(() => {
    loadTrades();
    loadTags();
  });

  async function loadTrades() {
    try {
      loading = true;
      const params = {
        page: pagination.page,
        page_size: pagination.page_size,
        ...filters
      };

      // 移除空值
      Object.keys(params).forEach(key => {
        if (params[key] === '') delete params[key];
      });

      const response = await tradesAPI.getAll(params);
      trades = response.data.data || [];
      pagination = response.data.pagination;
    } catch (error) {
      console.error('載入交易列表失敗:', error);
      alert('載入交易列表失敗');
    } finally {
      loading = false;
    }
  }

  async function loadTags() {
    try {
      const response = await tagsAPI.getAll();
      allTags = response.data || [];
    } catch (error) {
      console.error('載入標籤失敗:', error);
    }
  }

  async function deleteTrade(id) {
    if (!confirm('確定要刪除此交易紀錄嗎？')) return;

    try {
      await tradesAPI.delete(id);
      alert('刪除成功');
      loadTrades();
    } catch (error) {
      console.error('刪除失敗:', error);
      alert('刪除失敗：' + (error.response?.data?.error || error.message));
    }
  }

  function applyFilters() {
    pagination.page = 1;
    loadTrades();
  }

  function clearFilters() {
    filters = {
      symbol: '',
      side: '',
      tag: '',
      start_date: '',
      end_date: ''
    };
    pagination.page = 1;
    loadTrades();
  }

  function changePage(newPage) {
    pagination.page = newPage;
    loadTrades();
  }

  function formatDate(dateString) {
    return new Date(dateString).toLocaleString('zh-TW', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function openImageModal(imagePath) {
    selectedImage = imagesAPI.getUrl(imagePath);
  }

  function closeImageModal() {
    selectedImage = null;
  }
</script>

<div class="card">
  <div class="header">
    <h2>📋 交易歷史紀錄</h2>
    <Link to="/new" class="btn btn-primary">➕ 新增交易</Link>
  </div>

  <!-- 篩選器 -->
  <div class="filters">
    <div class="filter-group">
      <label>品種</label>
      <select bind:value={filters.symbol} class="form-control">
        <option value="">全部品種</option>
        <option value="XAUUSD">XAUUSD</option>
        <option value="NAS100">NAS100</option>
        <option value="US30">US30</option>
        <option value="EURUSD">EURUSD</option>
        <option value="GBPUSD">GBPUSD</option>
        <option value="USDJPY">USDJPY</option>
      </select>
    </div>

    <div class="filter-group">
      <label>方向</label>
      <select bind:value={filters.side} class="form-control">
        <option value="">全部方向</option>
        <option value="long">做多</option>
        <option value="short">做空</option>
      </select>
    </div>

    <div class="filter-group">
      <label>標籤</label>
      <select bind:value={filters.tag} class="form-control">
        <option value="">全部標籤</option>
        {#each allTags as tag}
          <option value={tag.name}>{tag.name}</option>
        {/each}
      </select>
    </div>

    <div class="filter-group">
      <label>開始日期</label>
      <input type="date" bind:value={filters.start_date} class="form-control" />
    </div>

    <div class="filter-group">
      <label>結束日期</label>
      <input type="date" bind:value={filters.end_date} class="form-control" />
    </div>

    <div class="filter-actions">
      <button class="btn btn-primary" on:click={applyFilters}>套用篩選</button>
      <button class="btn" on:click={clearFilters}>清除</button>
    </div>
  </div>

  <!-- 交易列表 -->
  {#if loading}
    <div class="loading">載入中...</div>
  {:else if trades.length === 0}
    <div class="empty">
      <p>📭 尚無交易紀錄</p>
      <Link to="/new" class="btn btn-primary">開始記錄第一筆交易</Link>
    </div>
  {:else}
    <div class="trades-grid">
      {#each trades as trade (trade.id)}
        <div class="trade-card" on:click={() => navigate(`/edit/${trade.id}`)}>
          <!-- 刪除按鈕（右上角叉叉）-->
          <button 
            class="delete-btn" 
            on:click={(e) => {
              e.stopPropagation();
              deleteTrade(trade.id);
            }}
            title="刪除交易"
          >
            ×
          </button>

          <!-- 單一行：品種 + 方向 + 所有資訊 + 盈虧 -->
          <div class="trade-header-compact">
            <div class="compact-left">
              <h3>{trade.symbol}</h3>
              <span class="badge {trade.side === 'long' ? 'badge-info' : 'badge-danger'}">
                {trade.side === 'long' ? '📈 做多' : '📉 做空'}
              </span>
              <span class="compact-item">
                <span class="compact-label">進場:</span>
                <span class="compact-value">{trade.entry_price}</span>
              </span>
              {#if trade.exit_price}
                <span class="compact-item">
                  <span class="compact-label">平倉:</span>
                  <span class="compact-value">{trade.exit_price}</span>
                </span>
              {/if}
              <span class="compact-item">
                <span class="compact-label">手數:</span>
                <span class="compact-value">{trade.lot_size}</span>
              </span>
              <span class="compact-item">
                <span class="compact-label">時間:</span>
                <span class="compact-value">{formatDate(trade.entry_time)}</span>
              </span>
            </div>
            {#if trade.pnl !== null}
              <span class="pnl {trade.pnl >= 0 ? 'profit' : 'loss'}">
                {trade.pnl >= 0 ? '+' : ''}{trade.pnl.toFixed(2)}
              </span>
            {/if}
          </div>

          {#if trade.tags && trade.tags.length > 0}
            <div class="trade-tags">
              {#each trade.tags as tag}
                <span class="tag">#{tag.name}</span>
              {/each}
            </div>
          {/if}

          {#if trade.entry_reason || trade.exit_reason}
            <div class="trade-reasons">
              {#if trade.entry_reason}
                <div class="reason-item">
                  <span class="reason-label">📍 進場分析：</span>
                  <div class="reason-content" on:click={(e) => e.stopPropagation()}>
                    {@html trade.entry_reason}
                  </div>
                </div>
              {/if}
              {#if trade.exit_reason}
                <div class="reason-item">
                  <span class="reason-label">🎯 平倉理由：</span>
                  <div class="reason-content" on:click={(e) => e.stopPropagation()}>
                    {@html trade.exit_reason}
                  </div>
                </div>
              {/if}
            </div>
          {/if}

          {#if trade.notes}
            <div class="trade-notes">
              <div class="notes-content" on:click={(e) => e.stopPropagation()}>
                {@html trade.notes}
              </div>
            </div>
          {/if}

          {#if trade.images && trade.images.length > 0}
            <div class="trade-images">
              {#each trade.images as image}
                <button 
                  class="image-thumb" 
                  on:click={(e) => {
                    e.stopPropagation();
                    openImageModal(image.image_path);
                  }}
                  title="點擊查看圖片">
                  <img 
                    src={imagesAPI.getUrl(image.image_path)} 
                    alt={image.image_type}
                    on:error={(e) => {
                      console.error('圖片載入失敗:', image.image_path);
                      e.target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgZmlsbD0iI2VlZSIvPjx0ZXh0IHg9IjUwJSIgeT0iNTAlIiBmb250LXNpemU9IjE0IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBkeT0iLjNlbSI+5ZyW54mH6Yyy5aSx5pWXPC90ZXh0Pjwvc3ZnPg=='
                    }}
                  />
                  <span class="image-label">
                    {#if image.image_type === 'entry'}
                      📈 進場
                    {:else if image.image_type === 'exit'}
                      📉 平倉
                    {:else if image.image_type === 'trailing_stop'}
                      🎯 移動停利
                    {:else if image.image_type === 'observation'}
                      👁️ 觀察
                    {:else}
                      📷 圖片
                    {/if}
                  </span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- 分頁 -->
    <div class="pagination">
      <button 
        class="btn" 
        disabled={pagination.page === 1}
        on:click={() => changePage(pagination.page - 1)}>
        上一頁
      </button>
      <span>第 {pagination.page} 頁，共 {Math.ceil(pagination.total / pagination.page_size)} 頁</span>
      <button 
        class="btn" 
        disabled={pagination.page >= Math.ceil(pagination.total / pagination.page_size)}
        on:click={() => changePage(pagination.page + 1)}>
        下一頁
      </button>
    </div>
  {/if}
</div>

<!-- 圖片模態框 -->
{#if selectedImage}
  <div class="modal" on:click={closeImageModal}>
    <div class="modal-content" on:click|stopPropagation>
      <button class="modal-close" on:click={closeImageModal}>×</button>
      <img src={selectedImage} alt="交易圖片" />
    </div>
  </div>
{/if}

<style>
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  h2 {
    margin: 0;
    color: #2d3748;
  }

  .filters {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
    padding: 1.5rem;
    background: #f7fafc;
    border-radius: 12px;
  }

  .filter-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .filter-group label {
    font-size: 0.875rem;
    font-weight: 600;
    color: #4a5568;
  }

  .filter-actions {
    display: flex;
    gap: 0.5rem;
    align-items: flex-end;
  }

  .loading, .empty {
    text-align: center;
    padding: 3rem;
    color: #718096;
  }

  .empty p {
    font-size: 1.5rem;
    margin-bottom: 1.5rem;
  }

  .trades-grid {
    display: grid;
    gap: 1.5rem;
  }

  .trade-card {
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    padding: 2.5rem 1.5rem 1.5rem 1.5rem;
    transition: all 0.3s ease;
    cursor: pointer;
    position: relative;
  }

  .trade-card:hover {
    border-color: #667eea;
    box-shadow: 0 8px 24px rgba(102, 126, 234, 0.25);
    transform: translateY(-2px);
  }

  .trade-card:active {
    transform: translateY(0);
  }

  /* 右上角刪除按鈕 */
  .delete-btn {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 28px;
    height: 28px;
    border: none;
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border-radius: 50%;
    font-size: 1.3rem;
    line-height: 1;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    z-index: 10;
    opacity: 0;
  }

  .trade-card:hover .delete-btn {
    opacity: 1;
  }

  .delete-btn:hover {
    background: #ef4444;
    color: white;
    transform: scale(1.1);
  }

  .delete-btn:active {
    transform: scale(0.95);
  }

  /* 緊湊單行版面 */
  .trade-header-compact {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 2px solid #e2e8f0;
    flex-wrap: wrap;
    gap: 1rem;
  }

  .compact-left {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .trade-header-compact h3 {
    margin: 0;
    color: #2d3748;
    font-size: 1.25rem;
    font-weight: 700;
  }

  .compact-item {
    display: inline-flex;
    align-items: baseline;
    gap: 0.25rem;
    white-space: nowrap;
  }

  .compact-label {
    color: #718096;
    font-size: 0.85rem;
    font-weight: 500;
  }

  .compact-value {
    color: #2d3748;
    font-weight: 600;
    font-size: 0.9rem;
  }

  .pnl {
    font-size: 1.5rem;
    font-weight: 700;
  }

  .pnl.profit {
    color: #38a169;
  }

  .pnl.loss {
    color: #e53e3e;
  }


  .trade-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .tag {
    background: #edf2f7;
    color: #667eea;
    padding: 0.25rem 0.75rem;
    border-radius: 12px;
    font-size: 0.875rem;
    font-weight: 600;
  }

  .trade-reasons {
    background: #f0f9ff;
    border-left: 4px solid #3b82f6;
    padding: 1rem;
    margin-bottom: 1rem;
    border-radius: 4px;
  }

  .reason-item {
    margin-bottom: 1rem;
  }

  .reason-item:last-child {
    margin-bottom: 0;
  }

  .reason-label {
    color: #1e40af;
    font-weight: 600;
    font-size: 0.9rem;
    display: block;
    margin-bottom: 0.5rem;
  }

  .reason-content {
    color: #1e3a8a;
    font-size: 0.9rem;
    line-height: 1.6;
    padding-left: 1.5rem;
  }

  .reason-content :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
    margin: 0.5rem 0;
    cursor: pointer;
    transition: transform 0.2s ease;
  }

  .reason-content :global(img:hover) {
    transform: scale(1.02);
  }

  .reason-content :global(p) {
    margin: 0.5rem 0;
  }

  .reason-content :global(strong) {
    font-weight: 600;
  }

  .reason-content :global(ul), 
  .reason-content :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.5rem;
  }

  .trade-notes {
    background: #fffaf0;
    border-left: 4px solid #ed8936;
    padding: 1rem;
    margin-bottom: 1rem;
    border-radius: 4px;
  }

  .notes-content {
    color: #744210;
    font-size: 0.9rem;
    line-height: 1.6;
  }

  .notes-content :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
    margin: 0.5rem 0;
    cursor: pointer;
    transition: transform 0.2s ease;
  }

  .notes-content :global(img:hover) {
    transform: scale(1.02);
  }

  .notes-content :global(p) {
    margin: 0.5rem 0;
  }

  .notes-content :global(p:first-child) {
    margin-top: 0;
  }

  .notes-content :global(p:last-child) {
    margin-bottom: 0;
  }

  .notes-content :global(strong) {
    font-weight: 600;
  }

  .notes-content :global(ul), 
  .notes-content :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.5rem;
  }

  .trade-images {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .image-thumb {
    position: relative;
    width: 150px;
    height: 100px;
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    border: 2px solid #e2e8f0;
    transition: transform 0.3s ease;
    background: none;
    padding: 0;
  }

  .image-thumb:hover {
    transform: scale(1.05);
    border-color: #667eea;
  }

  .image-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    background: #f7fafc;
  }

  .image-thumb img[src*="data:image"] {
    object-fit: contain;
  }

  .image-label {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    background: rgba(0, 0, 0, 0.7);
    color: white;
    padding: 0.25rem;
    font-size: 0.75rem;
    text-align: center;
  }


  .pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 1rem;
    margin-top: 2rem;
    padding-top: 2rem;
    border-top: 2px solid #e2e8f0;
  }

  .pagination span {
    color: #4a5568;
  }

  .modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.9);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    cursor: pointer;
  }

  .modal-content {
    position: relative;
    max-width: 90%;
    max-height: 90%;
    cursor: default;
  }

  .modal-content img {
    max-width: 100%;
    max-height: 90vh;
    border-radius: 8px;
  }

  .modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 3rem;
    cursor: pointer;
    line-height: 1;
  }
</style>

