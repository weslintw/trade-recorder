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
        if (timeInMinutes >= europeanStart || timeInMinutes < 0) {
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
