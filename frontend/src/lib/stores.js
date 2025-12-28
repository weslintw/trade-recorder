import { writable } from 'svelte/store';
import { SYMBOLS } from './constants';

export const selectedSymbol = writable(SYMBOLS[0] || 'XAUUSD');
