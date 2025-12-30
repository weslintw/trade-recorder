import { writable } from 'svelte/store';
import { SYMBOLS } from './constants';



// 預設帳號 ID 為 1
const storedAccount = typeof localStorage !== 'undefined' ? localStorage.getItem('selectedAccountId') : null;
export const selectedAccountId = writable(storedAccount && storedAccount !== 'NaN' ? parseInt(storedAccount) : null);

// 當帳號改變時存入 localStorage
selectedAccountId.subscribe(value => {
    if (typeof localStorage !== 'undefined') {
        if (value !== null && !isNaN(value)) {
            localStorage.setItem('selectedAccountId', value.toString());
        } else {
            localStorage.removeItem('selectedAccountId');
        }
    }
});

// 預設品種從 localStorage 讀取
const storedSymbol = typeof localStorage !== 'undefined' ? localStorage.getItem('selectedSymbol') : null;
export const selectedSymbol = writable(storedSymbol || SYMBOLS[0] || 'XAUUSD');

// 當品種改變時存入 localStorage
selectedSymbol.subscribe(value => {
    if (typeof localStorage !== 'undefined' && value) {
        localStorage.setItem('selectedSymbol', value);
    }
});

export const accounts = writable([]);
