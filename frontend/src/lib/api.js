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
  getSummary: () => api.get('/stats/summary'),
  getEquityCurve: () => api.get('/stats/equity-curve'),
  getBySymbol: () => api.get('/stats/by-symbol'),
};

// 標籤相關
export const tagsAPI = {
  getAll: () => api.get('/tags'),
};

export default api;

