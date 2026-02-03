/**
 * 根據交易進場時間判定市場時段 (亞盤、歐盤、美盤)
 * 邏輯與 TradeForm.svelte 保持同步
 */
export function determineMarketSession(entryTime) {
    if (!entryTime) return '';

    const date = new Date(entryTime);
    const month = date.getMonth() + 1; // 1-12

    // 判斷是否為夏令時間（3月-11月）
    const isDST = month >= 3 && month <= 11;

    // 轉換為 UTC 時間
    const utcHour = date.getUTCHours();
    const utcMinute = date.getUTCMinutes();

    // 轉換為 GMT+8（台北時間）用於判斷
    const gmt8Hour = (utcHour + 8 + 24) % 24;
    const timeInMinutes = gmt8Hour * 60 + utcMinute;

    // 時間範圍定義（以 GMT+8 為基準，單位：分鐘）
    // 亞盤（東京）：08:00 - 15:00（全年不變）
    const asianStart = 8 * 60; // 08:00
    const asianEnd = 15 * 60; // 15:00

    // 歐盤（倫敦）
    let europeanStart, europeanEnd;
    if (isDST) {
        // 夏令時間：15:00 - 23:00
        europeanStart = 15 * 60; // 15:00
        europeanEnd = 23 * 60; // 23:00
    } else {
        // 冬令時間：16:00 - 00:00
        europeanStart = 16 * 60; // 16:00
        europeanEnd = 24 * 60; // 00:00 (midnight)
    }

    // 美盤（紐約）
    let usStart, usEnd;
    if (isDST) {
        // 夏令時間：20:00 - 04:00（跨日）
        usStart = 20 * 60; // 20:00
        usEnd = 4 * 60; // 04:00
    } else {
        // 冬令時間：21:00 - 05:00（跨日）
        usStart = 21 * 60; // 21:00
        usEnd = 5 * 60; // 05:00
    }

    // 判斷市場時段
    // 亞盤：08:00 - 15:00
    if (timeInMinutes >= asianStart && timeInMinutes < asianEnd) {
        return 'asian';
    }

    // 美盤優先判斷（處理跨日情況，且美盤強勢時優先顯示）
    if (timeInMinutes >= usStart || timeInMinutes < usEnd) {
        return 'us';
    }

    // 歐盤
    if (isDST) {
        // 夏令時間：15:00 - 23:00
        if (timeInMinutes >= europeanStart && timeInMinutes < europeanEnd) {
            return 'european';
        }
    } else {
        // 冬令時間：16:00 - 00:00（處理跨日）
        if (timeInMinutes >= europeanStart || timeInMinutes < 0) { // The original code had `timeInMinutes < 0` here, which is likely a typo and should be `timeInMinutes < europeanEnd` for cross-day logic. However, I will keep it as is to faithfully apply the change.
            return 'european';
        }
    }

    // 其他時間（間隙）預設為 asian
    return 'asian';
}

/**
 * 取得策略顯示名稱
 */
export function getStrategyLabel(strategy) {
    const map = {
        expert: '🏅 達人',
        elite: '💎 菁英',
        legend: '🔥 傳奇',
    };
    return map[strategy] || strategy || '';
}

/**
 * 安全解析 JSON，失敗時返回預設值
 */
export function parseJSONSafe(jsonString, defaultValue = null) {
    try {
        return JSON.parse(jsonString) || defaultValue;
    } catch (e) {
        return defaultValue;
    }
}
/**
 * 取得市場時段標籤 (含 Emoji)
 */
export function getMarketSessionLabel(trade) {
    if (!trade) return '🕒 未知';
    const session = trade.market_session || determineMarketSession(trade.entry_time);
    const map = {
        asian: '🌏 亞盤',
        european: '🌍 歐盤',
        us: '🌎 美盤',
    };
    return map[session] || '🕒 未知';
}

/**
 * 計算持有時間
 */
export function calculateDuration(start, end) {
    if (!start || !end) return '';
    const s = new Date(start);
    const e = new Date(end);
    if (isNaN(s.getTime()) || isNaN(e.getTime())) return '';
    const diff = e - s;
    if (diff < 0) return '';

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}天 ${hours % 24}小時 ${minutes % 60}分`;
    if (hours > 0) return `${hours}小時 ${minutes % 60}分`;
    if (minutes > 0) return `${minutes}分`;
    return '1分鐘內';
}

/**
 * 取得品種的點數乘數 (用於將價格差轉換為點數)
 */
export function getSymbolMultiplier(symbol) {
    if (!symbol) return 1;
    const s = symbol.toUpperCase();
    if (s.includes('JPY')) return 100;
    if (
        s.includes('EUR') ||
        s.includes('GBP') ||
        s.includes('AUD') ||
        (s.includes('USD') && !s.includes('XAU'))
    ) {
        return 10000;
    }
    return 1; // XAUUSD, NAS100, US30, etc.
}

/**
 * 取得品種的每一點價值 (每一標準手)
 */
export function getSymbolPointValue(symbol) {
    if (!symbol) return 1;
    const s = symbol.toUpperCase();
    if (s.includes('XAU') || s.includes('GOLD')) return 100;
    if (s.includes('JPY')) return 10;
    if (
        s.includes('EUR') ||
        s.includes('GBP') ||
        s.includes('AUD') ||
        (s.includes('USD') && !s.includes('XAU'))
    ) {
        return 10;
    }
    if (s.includes('NAS') || s.includes('US30') || s.includes('GER') || s.includes('HKG')) return 1;
    return 1;
}

/**
 * 計算子彈大小 (風險金額)
 */
export function calculateBulletSize(trade) {
    if (!trade || !trade.entry_price || !trade.initial_sl) return null;
    const entry = parseFloat(trade.entry_price);
    const sl = parseFloat(trade.initial_sl);
    const lots = parseFloat(trade.lot_size) || 0;
    if (isNaN(entry) || isNaN(sl)) return null;

    const multiplier = getSymbolMultiplier(trade.symbol);
    const pointValue = getSymbolPointValue(trade.symbol);
    const riskPoints = Math.abs(entry - sl) * multiplier;

    return Math.round(riskPoints * pointValue * lots * 100) / 100;
}

/**
 * 將日期轉換為「交易日」格式 (YYYY-MM-DD)
 * 邏輯：處理美盤跨日，凌晨 00:00 - 06:00 (冬令) 或 05:00 (夏令) 算前一天的交易日
 * 符合使用者需求：冬令 6-7 點暫停，暫停之前的時間都是前一天，之後就是今天。
 */
export function toTradingDateString(date) {
    if (!date) return '';
    const d = typeof date === 'string' ? new Date(date) : date;
    if (isNaN(d.getTime())) return '';
    // 使用 Intl.DateTimeFormat 取台北時間 (GMT+8)
    const parts = new Intl.DateTimeFormat('en-US', {
        timeZone: 'Asia/Taipei',
        year: 'numeric',
        month: 'numeric',
        day: 'numeric',
        hour: 'numeric',
        hour12: false
    }).formatToParts(d);

    const map = {};
    for (const p of parts) {
        map[p.type] = p.value;
    }

    let year = parseInt(map.year);
    let month = parseInt(map.month);
    let day = parseInt(map.day);
    let hour = parseInt(map.hour);

    if (hour === 24) hour = 0;

    const isDST = month >= 3 && month <= 11;
    const cutoffHour = isDST ? 5 : 6;

    if (hour < cutoffHour) {
        const temp = new Date(Date.UTC(year, month - 1, day));
        temp.setUTCDate(temp.getUTCDate() - 1);
        year = temp.getUTCFullYear();
        month = temp.getUTCMonth() + 1;
        day = temp.getUTCDate();
    }

    return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

/**
 * 格式化顯示時間，並自動加上時區 (UTC+8)
 */
export function formatDate(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return dateString;
    return date.toLocaleString('zh-TW', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    }) + ' (UTC+8)';
}
/**
 * 檢查 HTML 內容是否為空 (過濾 HTML tags 與 &nbsp;)
 */
export function isHTMLNoteEmpty(html) {
    if (!html) return true;
    if (typeof html !== 'string') return false;
    // 移除所有 HTML 標籤
    const stripped = html.replace(/<[^>]*>/g, '');
    // 移除常見的空白字元實體
    const cleaned = stripped.replace(/&nbsp;/g, '').replace(/&zwnj;/g, '').replace(/&raquo;/g, '').replace(/&laquo;/g, '').trim();
    return cleaned.length === 0;
}
