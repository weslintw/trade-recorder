<script>
  import { onMount, onDestroy } from 'svelte';
  import Quill from 'quill';
  import 'quill/dist/quill.snow.css';
  import { imagesAPI } from '../lib/api';

  export let value = '';
  export let placeholder = '輸入內容...';
  export let height = '200px';

  let editor;
  let quill;
  let editorContainer;

  onMount(() => {
    // 初始化 Quill 編輯器
    quill = new Quill(editor, {
      theme: 'snow',
      placeholder: placeholder,
      modules: {
        toolbar: [
          ['bold', 'italic', 'underline', 'strike'],
          [{ 'header': [1, 2, 3, false] }],
          [{ 'list': 'ordered'}, { 'list': 'bullet' }],
          [{ 'color': [] }, { 'background': [] }],
          ['link', 'image'],
          ['clean']
        ]
      }
    });

    // 設定初始內容
    if (value) {
      quill.root.innerHTML = value;
    }

    // 監聽內容變化
    quill.on('text-change', () => {
      value = quill.root.innerHTML;
    });

    // 處理圖片上傳
    const toolbar = quill.getModule('toolbar');
    toolbar.addHandler('image', selectLocalImage);

    // 處理貼上圖片
    quill.root.addEventListener('paste', handlePaste);
  });

  // 響應式更新：當 value 從外部變化時更新編輯器內容
  $: if (quill && value !== quill.root.innerHTML) {
    const currentSelection = quill.getSelection();
    quill.root.innerHTML = value || '';
    if (currentSelection) {
      quill.setSelection(currentSelection);
    }
  }

  onDestroy(() => {
    if (quill) {
      quill.root.removeEventListener('paste', handlePaste);
    }
  });

  // 選擇本地圖片
  function selectLocalImage() {
    const input = document.createElement('input');
    input.setAttribute('type', 'file');
    input.setAttribute('accept', 'image/*');
    input.click();

    input.onchange = async () => {
      const file = input.files[0];
      if (file) {
        await uploadImage(file);
      }
    };
  }

  // 處理貼上事件
  async function handlePaste(e) {
    const clipboardData = e.clipboardData || window.clipboardData;
    const items = clipboardData.items;

    for (let item of items) {
      if (item.type.indexOf('image') !== -1) {
        e.preventDefault();
        const file = item.getAsFile();
        await uploadImage(file);
        break;
      }
    }
  }

  // 上傳圖片
  async function uploadImage(file) {
    try {
      const formData = new FormData();
      formData.append('image', file);
      formData.append('symbol', 'note'); // 用於分類

      const response = await imagesAPI.upload(formData);
      const imageUrl = imagesAPI.getUrl(response.data.path);

      // 插入圖片到編輯器
      const range = quill.getSelection(true);
      quill.insertEmbed(range.index, 'image', imageUrl);
      quill.setSelection(range.index + 1);
    } catch (error) {
      console.error('圖片上傳失敗:', error);
      alert('圖片上傳失敗：' + (error.response?.data?.error || error.message));
    }
  }

  // 暴露方法讓父組件可以取得內容
  export function getContent() {
    return quill ? quill.root.innerHTML : value;
  }

  // 暴露方法讓父組件可以設定內容
  export function setContent(html) {
    if (quill) {
      quill.root.innerHTML = html;
      value = html;
    }
  }
</script>

<div class="rich-editor-wrapper" bind:this={editorContainer}>
  <div bind:this={editor} style="height: {height}"></div>
</div>

<style>
  .rich-editor-wrapper {
    border: 1px solid #cbd5e0;
    border-radius: 8px;
    overflow: hidden;
    background: white;
  }

  .rich-editor-wrapper :global(.ql-toolbar) {
    border: none;
    border-bottom: 1px solid #e2e8f0;
    background: #f7fafc;
  }

  .rich-editor-wrapper :global(.ql-container) {
    border: none;
    font-family: inherit;
    font-size: 0.95rem;
  }

  .rich-editor-wrapper :global(.ql-editor) {
    min-height: 150px;
    max-height: 400px;
    overflow-y: auto;
  }

  .rich-editor-wrapper :global(.ql-editor.ql-blank::before) {
    color: #a0aec0;
    font-style: normal;
  }

  .rich-editor-wrapper :global(.ql-editor img) {
    max-width: 100%;
    height: auto;
    display: block;
    margin: 0.5rem 0;
    border-radius: 4px;
  }

  .rich-editor-wrapper :global(.ql-snow .ql-picker) {
    font-size: 0.9rem;
  }
</style>

