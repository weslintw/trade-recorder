import axios from 'axios';

const API_BASE_URL = 'http://localhost:8080/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 交易紀錄相關
export const tradesAPI = {
  getAll: (params) => api.get('/trades', { params }),
  getOne: (id) => api.get(`/trades/${id}`),
  create: (data) => api.post('/trades', data),
  update: (id, data) => api.put(`/trades/${id}`, data),
  delete: (id) => api.delete(`/trades/${id}`),
};

// 圖片相關
export const imagesAPI = {
  upload: (formData) => api.post('/images/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  }),
  getUrl: (path) => {
    // path 格式: 2025-01/20250101-XAUUSD-abc123.jpg
    const filename = path.split('/').pop(); // 取得檔名
    return `${API_BASE_URL}/images/${filename}?path=${encodeURIComponent(path)}`;
  },
};

// 統計相關
export const statsAPI = {
  getSummary: (params) => api.get('/stats/summary', { params }),
  getEquityCurve: (params) => api.get('/stats/equity-curve', { params }),
  getBySymbol: (params) => api.get('/stats/by-symbol', { params }),
};

// 標籤相關
export const tagsAPI = {
  getAll: () => api.get('/tags'),
};

// 每日盤面規劃相關
export const dailyPlansAPI = {
  getAll: (params) => api.get('/daily-plans', { params }),
  getOne: (id) => api.get(`/daily-plans/${id}`),
  create: (data) => api.post('/daily-plans', data),
  update: (id, data) => api.put(`/daily-plans/${id}`, data),
  delete: (id) => api.delete(`/daily-plans/${id}`),
};

// 帳號管理相關
export const accountsAPI = {
  getAll: () => api.get('/accounts'),
  create: (data) => api.post('/accounts', data),
  update: (id, data) => api.put(`/accounts/${id}`, data),
  delete: (id) => api.delete(`/accounts/${id}`),
  sync: (id) => api.post(`/accounts/${id}/sync`),
};

export default api;

