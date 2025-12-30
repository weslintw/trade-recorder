<script>
  import { onMount, onDestroy } from 'svelte';

  export let imageSrc = '';
  export let originalImageSrc = ''; // 最原始的圖片版本
  export let onSave = null; // 回調函數，接收標註後的圖片 base64

  let canvas;
  let ctx;
  let image;
  let isDrawing = false;
  let lastX = 0;
  let lastY = 0;
  let startX = 0;
  let startY = 0;
  let savedImageData = null; // 保存當前圖片數據（包含標註）
  let originalImageData = null; // 保存最原始的圖片數據（不含標註）

  // 工具選項
  let tool = 'brush'; // 'brush', 'line', 或 'text'
  let color = '#ff0000'; // 預設紅色
  let lineWidth = 3;
  let textInput = ''; // 文字輸入內容
  let textPosition = null; // {x, y} 文字位置

  // 預設顏色選項
  const colors = [
    '#ff0000', // 紅
    '#00ff00', // 綠
    '#0000ff', // 藍
    '#ffff00', // 黃
    '#ff00ff', // 洋紅
    '#00ffff', // 青
    '#000000', // 黑
    '#ffffff', // 白
    '#ffa500'  // 橙
  ];

  // 線條粗度選項（移除最大的三個）
  const lineWidths = [1, 2, 3];

  onMount(() => {
    if (canvas) {
      ctx = canvas.getContext('2d');
      if (imageSrc) {
        loadImage();
      }
    }
  });

  function loadImage() {
    if (!imageSrc || !ctx || !canvas) return;

    image = new Image();
    image.crossOrigin = 'anonymous';
    image.onload = () => {
      // 設定 canvas 實際尺寸（用於繪圖，保持原始圖片尺寸）
      canvas.width = image.width;
      canvas.height = image.height;
      
      // 繪製原始圖片
      ctx.drawImage(image, 0, 0);
      
      // 保存原始圖片數據（不含標註）
      originalImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
      savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    };
    image.onerror = () => {
      console.error('圖片載入失敗:', imageSrc);
    };
    image.src = imageSrc;
  }

  // 恢復原始圖片
  function restoreImage() {
    if (savedImageData && ctx) {
      ctx.putImageData(savedImageData, 0, 0);
    } else if (image && ctx) {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.drawImage(image, 0, 0);
    }
  }

  // 響應式：當圖片來源改變時重新載入
  let lastImageSrc = '';
  $: {
    if (imageSrc && canvas && imageSrc !== lastImageSrc) {
      lastImageSrc = imageSrc;
      if (!ctx) {
        ctx = canvas.getContext('2d');
      }
      loadImage();
    }
  }

  function startDrawing(e) {
    if (!canvas || !ctx) return;
    
    // 文字工具：點擊設置文字位置
    if (tool === 'text') {
      const rect = canvas.getBoundingClientRect();
      const scaleX = canvas.width / rect.width;
      const scaleY = canvas.height / rect.height;
      
      textPosition = {
        x: (e.clientX - rect.left) * scaleX,
        y: (e.clientY - rect.top) * scaleY
      };
      return;
    }
    
    isDrawing = true;
    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    
    lastX = (e.clientX - rect.left) * scaleX;
    lastY = (e.clientY - rect.top) * scaleY;
    startX = lastX;
    startY = lastY;
    
    // 線條模式：在開始繪製前，保存當前狀態（包括已繪製的線條）
    if (tool === 'line') {
      savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    }
    
    // 畫筆模式：在開始點畫一個點
    if (tool === 'brush') {
      ctx.strokeStyle = color;
      ctx.fillStyle = color;
      ctx.lineWidth = lineWidth;
      ctx.lineCap = 'round';
      ctx.beginPath();
      ctx.arc(lastX, lastY, lineWidth / 2, 0, Math.PI * 2);
      ctx.fill();
    }
  }

  function draw(e) {
    if (!isDrawing || !ctx || !canvas) return;

    e.preventDefault();
    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    
    const currentX = (e.clientX - rect.left) * scaleX;
    const currentY = (e.clientY - rect.top) * scaleY;

    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';

    if (tool === 'brush') {
      // 畫筆模式：連續繪製
      ctx.beginPath();
      ctx.moveTo(lastX, lastY);
      ctx.lineTo(currentX, currentY);
      ctx.stroke();
      lastX = currentX;
      lastY = currentY;
    } else if (tool === 'line') {
      // 更新最後位置
      lastX = currentX;
      lastY = currentY;
      
      // 線條模式：預覽線條
      // 先恢復原始圖片和已繪製的內容（不包括當前預覽線條）
      if (savedImageData) {
        ctx.putImageData(savedImageData, 0, 0);
      } else if (image) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(image, 0, 0);
      }
      
      // 繪製預覽線條
      ctx.strokeStyle = color;
      ctx.lineWidth = lineWidth;
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';
      ctx.beginPath();
      ctx.moveTo(startX, startY);
      ctx.lineTo(currentX, currentY);
      ctx.stroke();
    }
  }

  function stopDrawing(e) {
    if (!isDrawing || !ctx || !canvas) {
      isDrawing = false;
      return;
    }
    
    if (tool === 'line') {
      // 線條模式：確定線條
      const rect = canvas.getBoundingClientRect();
      const scaleX = canvas.width / rect.width;
      const scaleY = canvas.height / rect.height;
      
      let endX, endY;
      if (e && e.clientX !== undefined && e.clientY !== undefined) {
        endX = (e.clientX - rect.left) * scaleX;
        endY = (e.clientY - rect.top) * scaleY;
      } else {
        // 使用最後的位置
        endX = lastX;
        endY = lastY;
      }
      
      // 恢復保存的狀態（包括已繪製的線條，不包括預覽線條）
      if (savedImageData) {
        ctx.putImageData(savedImageData, 0, 0);
      } else if (image) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(image, 0, 0);
      }
      
      // 繪製最終線條（永久保存）
      ctx.strokeStyle = color;
      ctx.lineWidth = lineWidth;
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';
      ctx.beginPath();
      ctx.moveTo(startX, startY);
      ctx.lineTo(endX, endY);
      ctx.stroke();
      
      // 立即更新保存的圖片數據（包含新繪製的線條）
      savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    } else if (tool === 'brush') {
      // 畫筆模式：更新保存的圖片數據
      savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    }
    
    isDrawing = false;
  }

  function clearCanvas() {
    if (!ctx || !canvas || !originalImageData) return;
    // 恢復到原始圖片（清除所有標註）
    ctx.putImageData(originalImageData, 0, 0);
    // 更新保存的圖片數據
    savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
  }

  function resetToOriginal() {
    if (!ctx || !canvas) return;
    
    // 如果有提供 originalImageSrc，則加載它
    if (originalImageSrc && originalImageSrc !== imageSrc) {
      const originalImage = new Image();
      originalImage.onload = () => {
        // 設置 canvas 尺寸為原始圖片尺寸
        canvas.width = originalImage.width;
        canvas.height = originalImage.height;
        
        // 繪製原始圖片
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(originalImage, 0, 0);
        
        // 更新保存的圖片數據
        originalImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
      };
      originalImage.src = originalImageSrc;
    } else if (originalImageData) {
      // 如果沒有 originalImageSrc，使用保存的 originalImageData
      ctx.putImageData(originalImageData, 0, 0);
      savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    }
  }

  function saveImage() {
    if (!canvas) return;
    
    const dataURL = canvas.toDataURL('image/png');
    if (onSave) {
      onSave(dataURL);
    }
  }

  function setColor(newColor) {
    color = newColor;
  }

  function setTool(newTool) {
    tool = newTool;
  }

  function setLineWidth(width) {
    lineWidth = width;
  }

  function addText() {
    if (!ctx || !canvas || !textPosition || !textInput.trim()) return;
    
    // 設置文字樣式
    ctx.fillStyle = color;
    ctx.font = `${lineWidth * 8}px Arial`; // 根據粗度調整文字大小
    ctx.textBaseline = 'top';
    
    // 繪製文字
    ctx.fillText(textInput, textPosition.x, textPosition.y);
    
    // 更新保存的圖片數據
    savedImageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    
    // 清空輸入和位置
    textInput = '';
    textPosition = null;
  }

  function cancelText() {
    textInput = '';
    textPosition = null;
  }

</script>

<div class="annotator-container">
  <div class="annotator-toolbar">
    <!-- 工具選擇 -->
    <div class="tool-group">
      <span class="tool-label">工具：</span>
      <div class="tool-buttons">
        <button 
          class="tool-btn" 
          class:active={tool === 'brush'}
          on:click={() => setTool('brush')}
          title="畫筆"
        >
          ✏️
        </button>
        <button 
          class="tool-btn" 
          class:active={tool === 'line'}
          on:click={() => setTool('line')}
          title="線條"
        >
          📏
        </button>
        <button 
          class="tool-btn" 
          class:active={tool === 'text'}
          on:click={() => setTool('text')}
          title="文字"
        >
          📝
        </button>
      </div>
    </div>

    <!-- 顏色選擇 -->
    <div class="tool-group">
      <span class="tool-label">顏色：</span>
      <div class="color-picker">
        {#each colors as c}
          <button
            class="color-btn"
            class:active={color === c}
            style="background-color: {c}; border-color: {c === '#ffffff' ? '#ccc' : c};"
            on:click={() => setColor(c)}
            title={c}
          />
        {/each}
        <input 
          type="color" 
          class="color-input"
          bind:value={color}
          title="自訂顏色"
        />
      </div>
    </div>

    <!-- 線條粗度 -->
    <div class="tool-group">
      <span class="tool-label">粗度：</span>
      <div class="line-width-selector">
        {#each lineWidths as w}
          <button
            class="width-btn"
            class:active={lineWidth === w}
            on:click={() => setLineWidth(w)}
            title="{w}px"
          >
            <span class="width-indicator" style="width: {w * 2}px; height: {w * 2}px; background: {color};"></span>
          </button>
        {/each}
      </div>
    </div>

    <!-- 操作按鈕 -->
    <div class="tool-group actions">
      <button class="action-btn reset" on:click={resetToOriginal} title="重置到原始圖片">
        🔄
      </button>
      <button class="action-btn clear" on:click={clearCanvas} title="清除所有標註">
        🗑️
      </button>
      <button class="action-btn save" on:click={saveImage} title="保存標註">
        💾
      </button>
    </div>
  </div>

  <!-- 文字輸入對話框 -->
  {#if textPosition}
    <div class="text-input-dialog">
      <input 
        type="text" 
        bind:value={textInput} 
        placeholder="輸入文字..."
        on:keydown={(e) => {
          if (e.key === 'Enter') addText();
          if (e.key === 'Escape') cancelText();
        }}
      />
      <div class="text-dialog-buttons">
        <button class="text-btn confirm" on:click={addText}>✓ 確定</button>
        <button class="text-btn cancel" on:click={cancelText}>✗ 取消</button>
      </div>
    </div>
  {/if}

  <div class="canvas-wrapper">
    <canvas
      bind:this={canvas}
      on:mousedown|preventDefault={startDrawing}
      on:mousemove|preventDefault={draw}
      on:mouseup|preventDefault={stopDrawing}
      on:mouseleave|preventDefault={stopDrawing}
      on:touchstart|preventDefault={(e) => {
        const touch = e.touches[0];
        const rect = canvas.getBoundingClientRect();
        const fakeEvent = {
          clientX: touch.clientX,
          clientY: touch.clientY,
          preventDefault: () => {}
        };
        startDrawing(fakeEvent);
      }}
      on:touchmove|preventDefault={(e) => {
        const touch = e.touches[0];
        const fakeEvent = {
          clientX: touch.clientX,
          clientY: touch.clientY,
          preventDefault: () => {}
        };
        draw(fakeEvent);
      }}
      on:touchend|preventDefault={(e) => {
        stopDrawing();
      }}
    />
  </div>
</div>

<style>
  .annotator-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 100%;
    max-width: 100%;
  }

  .annotator-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    padding: 1rem;
    background: #f7fafc;
    border-radius: 8px;
    align-items: center;
  }

  .tool-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .tool-label {
    font-size: 0.9rem;
    font-weight: 600;
    color: #4a5568;
    white-space: nowrap;
  }

  .tool-buttons {
    display: flex;
    gap: 0.5rem;
  }

  .tool-btn {
    padding: 0.5rem 1rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s ease;
  }

  .tool-btn:hover {
    border-color: #667eea;
    background: #edf2f7;
  }

  .tool-btn.active {
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  .color-picker {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .color-btn {
    width: 32px;
    height: 32px;
    border: 3px solid #cbd5e0;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    padding: 0;
  }

  .color-btn:hover {
    transform: scale(1.1);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }

  .color-btn.active {
    border-color: #2d3748;
    box-shadow: 0 0 0 2px #667eea;
  }

  .color-input {
    width: 32px;
    height: 32px;
    border: 3px solid #cbd5e0;
    border-radius: 6px;
    cursor: pointer;
    padding: 0;
    background: none;
  }

  .color-input::-webkit-color-swatch-wrapper {
    padding: 0;
  }

  .color-input::-webkit-color-swatch {
    border: none;
    border-radius: 3px;
  }

  .line-width-selector {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .width-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;
    padding: 0;
  }

  .width-btn:hover {
    border-color: #667eea;
    background: #edf2f7;
  }

  .width-btn.active {
    border-color: #667eea;
    background: #edf2f7;
  }

  .width-indicator {
    display: block;
    border-radius: 50%;
    transition: all 0.2s ease;
  }

  .actions {
    margin-left: auto;
  }

  .action-btn {
    padding: 0.5rem 1rem;
    border: 2px solid #cbd5e0;
    border-radius: 6px;
    background: white;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s ease;
  }

  .action-btn.reset {
    color: #2d3748;
    border-color: #cbd5e0;
  }

  .action-btn.reset:hover {
    background: #e2e8f0;
    border-color: #a0aec0;
  }

  .action-btn.clear {
    color: #e53e3e;
    border-color: #fc8181;
  }

  .action-btn.clear:hover {
    background: #fed7d7;
    border-color: #e53e3e;
  }

  .action-btn.save {
    color: #2d3748;
    border-color: #667eea;
    background: #667eea;
    color: white;
  }

  .action-btn.save:hover {
    background: #5568d3;
    border-color: #5568d3;
  }

  /* 文字輸入對話框 */
  .text-input-dialog {
    position: absolute;
    top: 80px;
    left: 50%;
    transform: translateX(-50%);
    background: white;
    padding: 1rem;
    border: 2px solid #667eea;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    z-index: 1000;
  }

  .text-input-dialog input {
    width: 300px;
    padding: 0.5rem;
    border: 2px solid #cbd5e0;
    border-radius: 4px;
    font-size: 1rem;
    margin-bottom: 0.5rem;
  }

  .text-input-dialog input:focus {
    outline: none;
    border-color: #667eea;
  }

  .text-dialog-buttons {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .text-btn {
    padding: 0.4rem 0.8rem;
    border: 2px solid #cbd5e0;
    border-radius: 4px;
    background: white;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s ease;
  }

  .text-btn.confirm {
    color: #2f855a;
    border-color: #68d391;
  }

  .text-btn.confirm:hover {
    background: #c6f6d5;
    border-color: #2f855a;
  }

  .text-btn.cancel {
    color: #c53030;
    border-color: #fc8181;
  }

  .text-btn.cancel:hover {
    background: #fed7d7;
    border-color: #c53030;
  }

  .canvas-wrapper {
    position: relative;
    width: 100%;
    max-width: 100%;
    overflow: auto;
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    background: #f7fafc;
  }

  canvas {
    display: block;
    max-width: 100%;
    height: auto;
    cursor: crosshair;
    touch-action: none;
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }
</style>

