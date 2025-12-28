import { writable } from 'svelte/store';
import { SYMBOLS } from './constants';

export const selectedSymbol = writable(SYMBOLS[0] || 'XAUUSD');

// 預設帳號 ID 為 1
const storedAccount = typeof localStorage !== 'undefined' ? localStorage.getItem('selectedAccountId') : null;
export const selectedAccountId = writable(storedAccount ? parseInt(storedAccount) : 1);

// 當帳號改變時存入 localStorage
selectedAccountId.subscribe(value => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem('selectedAccountId', value.toString());
    }
});

export const accounts = writable([]);
