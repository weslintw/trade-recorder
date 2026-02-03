import axios from 'axios';

const API_BASE_URL =
  import.meta.env.VITE_API_URL ||
  (import.meta.env.DEV ? 'http://localhost:8080/api/v1' : '/api/v1');

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 60000, // 增加到 60 秒，避免緩慢網路導致 ECONNABORTED
  headers: {
    'Content-Type': 'application/json',
  },
});

// 請求去重佇列 (Request Deduplication Queue)
const pendingRequests = new Map();

api.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // 針對 GET 請求加上隨機參數，強制繞過可能堵塞的舊連線
    if (config.method === 'get') {
      const separator = config.url.includes('?') ? '&' : '?';
      config.url = `${config.url}${separator}_t=${Date.now()}`;
    }

    config.metadata = { startTime: performance.now() };
    return config;
  },
  error => Promise.reject(error)
);

// 回應攔截器：統一記錄 Log 並清理佇列
api.interceptors.response.use(
  response => {
    const duration = performance.now() - response.config.metadata.startTime;
    console.log(`✅ [API] ${response.config.method.toUpperCase()} ${response.config.url} - ${duration.toFixed(2)}ms`);
    return response;
  },
  error => {
    const duration = error.config?.metadata ? (performance.now() - error.config.metadata.startTime).toFixed(2) : 'unknown';
    console.error(`❌ [API Error] ${error.config?.url} - ${duration}ms - ${error.message}`);
    return Promise.reject(error);
  }
);

// 包裝特定 API 實現真正去重
const originalGet = api.get;
api.get = function (url, config) {
  // 只針對特定的高頻資源進行去重 (如 /accounts)
  if (url === '/accounts' || url.startsWith('/accounts?')) {
    if (pendingRequests.has(url)) {
      console.log(`🛡️ [Dedupe] Sharing existing request for ${url}`);
      return pendingRequests.get(url);
    }

    const request = originalGet.call(this, url, config).finally(() => {
      pendingRequests.delete(url);
    });

    pendingRequests.set(url, request);
    return request;
  }
  return originalGet.call(this, url, config);
};

// 認證相關
export const authAPI = {
  changePassword: data => api.post('/auth/change-password', data),
};

// 交易紀錄相關
export const tradesAPI = {
  getAll: (params, signal) => api.get('/trades', { params, signal }),
  getOne: (id, signal) => api.get(`/trades/${id}`, { signal }),
  create: data => api.post('/trades', data),
  update: (id, data) => api.put(`/trades/${id}`, data),
  delete: id => api.delete(`/trades/${id}`),
  sync: id => api.post(`/trades/${id}/sync`),
  getUsedSymbols: (params, signal) => api.get('/trades/symbols', { params, signal }),
  getChartData: (id, period, signal) => api.get(`/trades/${id}/chart`, { params: { period }, signal }),
  saveTrendlines: (id, data) => api.put(`/trades/${id}/trendlines`, data),
  getTrendlines: (id, signal) => api.get(`/trades/${id}/trendlines`, { signal }),
};

// 圖片相關
export const imagesAPI = {
  upload: formData =>
    api.post('/images/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    }),
  getUrl: path => {
    // path 格式: 2025-01/20250101-XAUUSD-abc123.jpg
    const filename = path.split('/').pop(); // 取得檔名
    return `${API_BASE_URL}/images/${filename}?path=${encodeURIComponent(path)}`;
  },
};

// 統計相關
export const statsAPI = {
  getSummary: (params, signal) => api.get('/stats/summary', { params, signal }),
  getEquityCurve: (params, signal) =>
    api.get('/stats/equity-curve', { params, signal }),
  getBySymbol: (params, signal) => api.get('/stats/by-symbol', { params, signal }),
  getByStrategy: (params, signal) =>
    api.get('/stats/by-strategy', { params, signal }),
  getByColorTag: (params, signal) =>
    api.get('/stats/by-color', { params, signal }),
};

// 標籤相關
export const tagsAPI = {
  getAll: signal => api.get('/tags', { signal }),
};

// 每日盤面規劃相關
export const dailyPlansAPI = {
  getAll: (params, signal) => api.get('/daily-plans', { params, signal }),
  getOne: (id, signal) => api.get(`/daily-plans/${id}`, { signal }),
  create: data => api.post('/daily-plans', data),
  update: (id, data) => api.put(`/daily-plans/${id}`, data),
  delete: id => api.delete(`/daily-plans/${id}`),
};

// 帳號管理相關
export const accountsAPI = {
  getAll: signal => api.get('/accounts', { signal }),
  create: data => api.post('/accounts', data),
  update: (id, data) => api.put(`/accounts/${id}`, data),
  delete: id => api.delete(`/accounts/${id}`),
  sync: (id, data) => api.post(`/accounts/${id}/sync`, data),
  importCSV: (id, formData) =>
    api.post(`/accounts/${id}/import-csv`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    }),
  clearData: id => api.delete(`/accounts/${id}/data`),
  getCTraderAuthURL: params => api.get('/auth/ctrader/url', { params }),
};

// 分享相關
export const sharesAPI = {
  create: data => api.post('/shares', data),
  getPublic: (token, signal) => api.get(`/shares/public/${token}`, { signal }),
  getChartData: (token, tradeId, period, signal) => api.get(`/shares/public/${token}/chart/${tradeId}`, { params: { period }, signal }),
  getTrendlines: (token, tradeId, signal) => api.get(`/shares/public/${token}/trendlines/${tradeId}`, { signal }),
};

// 管理員相關
export const adminAPI = {
  getUsage: signal => api.get('/admin/usage', { signal }),
};

export default api;
