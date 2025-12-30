import { writable } from 'svelte/store';
import axios from 'axios';
import { selectedAccountId } from './stores';

// API 基礎路徑
const API_BASE_URL = import.meta.env.DEV ? 'http://localhost:8080/api/v1' : '/api/v1';

// 初始化時從 localStorage 讀取
const initialToken = localStorage.getItem('token');
const initialUser = JSON.parse(localStorage.getItem('user') || 'null');

export const auth = writable({
    token: initialToken,
    user: initialUser,
    isAuthenticated: !!initialToken
});

// 當 auth 變動時同步到 localStorage
auth.subscribe(value => {
    if (value.token) {
        localStorage.setItem('token', value.token);
        localStorage.setItem('user', JSON.stringify(value.user));
    } else {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
    }
});

export const login = async (username, password) => {
    try {
        const response = await axios.post(`${API_BASE_URL}/auth/login`, { username, password });
        const { token, user } = response.data;
        localStorage.removeItem('selectedAccountId');
        selectedAccountId.set(null);
        auth.set({ token, user, isAuthenticated: true });
        return { success: true };
    } catch (error) {
        return { 
            success: false, 
            error: error.response?.data?.error || '登入失敗，請檢查網路連線' 
        };
    }
};

export const register = async (username, password) => {
    try {
        const response = await axios.post(`${API_BASE_URL}/auth/register`, { username, password });
        const { token, user } = response.data;
        localStorage.removeItem('selectedAccountId');
        selectedAccountId.set(null);
        auth.set({ token, user, isAuthenticated: true });
        return { success: true };
    } catch (error) {
        return { 
            success: false, 
            error: error.response?.data?.error || '註冊失敗' 
        };
    }
};

export const logout = () => {
    localStorage.removeItem('selectedAccountId');
    selectedAccountId.set(null);
    auth.set({ token: null, user: null, isAuthenticated: false });
    window.location.href = '/'; // 強制跳轉並重新載入
};

// 檢查 Token 是否有效
export const checkAuth = async () => {
    const token = localStorage.getItem('token');
    if (!token) return false;

    try {
        const response = await axios.get(`${API_BASE_URL}/auth/me`, {
            headers: { Authorization: `Bearer ${token}` }
        });
        auth.update(a => ({ ...a, user: response.data, isAuthenticated: true }));
        return true;
    } catch (error) {
        logout();
        return false;
    }
};
