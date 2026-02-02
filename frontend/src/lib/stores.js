import { writable } from 'svelte/store';
import { SYMBOLS } from './constants';

// 預設帳號 ID 為 1
const storedAccount =
  typeof localStorage !== 'undefined' ? localStorage.getItem('selectedAccountId') : null;
export const selectedAccountId = writable(
  storedAccount && storedAccount !== 'NaN' ? parseInt(storedAccount) : null
);

// 當帳號改變時存入 localStorage 並清空快取
selectedAccountId.subscribe(value => {
  if (typeof localStorage !== 'undefined') {
    if (value !== null && !isNaN(value)) {
      localStorage.setItem('selectedAccountId', value.toString());
    } else {
      localStorage.removeItem('selectedAccountId');
    }
  }
  // 重要：切換帳號時立即清空快取，防止舊帳號資料殘留
  tradeDataCache.set({
    key: null,
    scope: null,
    plans: [],
    trades: [],
    summary: null,
    timestamp: null,
    stale: false,
  });
});

// 預設品種從 localStorage 讀取
const storedSymbol =
  typeof localStorage !== 'undefined' ? localStorage.getItem('selectedSymbol') : null;
export const selectedSymbol = writable(storedSymbol || SYMBOLS[0] || 'XAUUSD');

// 當品種改變時存入 localStorage
selectedSymbol.subscribe(value => {
  if (typeof localStorage !== 'undefined' && value) {
    localStorage.setItem('selectedSymbol', value);
  }
});

export const accounts = writable([]);

// 全局數據緩存，用於在頁面切換（如進入編輯頁再返回）時保留數據
// 這能解決「返回首頁時速度很慢」的問題
export const tradeDataCache = writable({
  key: null, // Cache key: `${accountId}_${symbol}_${startDate}_${endDate}`
  scope: null, // 'all' or 'partial'
  plans: [],
  trades: [],
  summary: null,
  timestamp: null,
  stale: false, // 如果為 true，代表數據可能過期（例如剛執行過編輯），需要背景更新
});

// 深色模式 Store
const storedTheme = typeof localStorage !== 'undefined' ? localStorage.getItem('theme') : null;
const initialDarkMode =
  storedTheme === 'dark' ||
  (!storedTheme &&
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-color-scheme: dark)').matches);

export const isDarkMode = writable(initialDarkMode);

// 訂閱 Store 變化並應用到 body 與 localStorage
isDarkMode.subscribe(value => {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('theme', value ? 'dark' : 'light');
  }
  if (typeof document !== 'undefined') {
    if (value) {
      document.body.classList.add('dark-mode');
    } else {
      document.body.classList.remove('dark-mode');
    }
  }
});
