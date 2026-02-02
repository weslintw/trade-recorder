package ctrader

import (
	"strings"
)

func getMultiplier(symbol string) float64 {
	if strings.Contains(symbol, "JPY") {
		return 100.0
	} // 2 decimals JPY = 100 pips per $1
	if strings.Contains(symbol, "XAU") || strings.Contains(symbol, "GOLD") || strings.Contains(symbol, "XPT") {
		return 1.0
	} // Gold: 1.0 = 1 point
	if strings.Contains(symbol, "NAS") || strings.Contains(symbol, "US30") || strings.Contains(symbol, "SPD") || strings.Contains(symbol, "HSI") {
		return 1.0
	} // Indices: 1.0 = 1 point
	// Forex pairs: 4 decimals = 10000 points (Pips)
	if len(symbol) >= 6 && (strings.Contains(symbol, "USD") || strings.Contains(symbol, "EUR") || strings.Contains(symbol, "GBP")) {
		// Exclude Cryptos
		if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
			return 1.0
		}
		return 10000.0
	}
	return 1.0
}

func getPointValue(symbol string) float64 {
	symbol = strings.ToUpper(symbol)
	if strings.Contains(symbol, "XAU") || strings.Contains(symbol, "GOLD") {
		return 100.0
	}
	if strings.Contains(symbol, "JPY") {
		return 10.0 // 這裡簡化，假設 1 pip = 10 USD (實際上受匯率影響)
	}
	if strings.Contains(symbol, "NAS") || strings.Contains(symbol, "US30") || strings.Contains(symbol, "GER") || strings.Contains(symbol, "HKG") || strings.Contains(symbol, "HSI") {
		return 1.0
	}
	// Forex pairs
	if len(symbol) >= 6 && (strings.Contains(symbol, "USD") || strings.Contains(symbol, "EUR") || strings.Contains(symbol, "GBP") || strings.Contains(symbol, "AUD")) {
		// Exclude Cryptos
		if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
			return 1.0
		}
		return 10.0 // 1 pip = 10 USD per lot
	}
	return 1.0
}

func getDigits(symbol string) int {
	symbol = strings.ToUpper(symbol)
	if strings.Contains(symbol, "JPY") {
		return 3
	}
	if strings.Contains(symbol, "XAU") || strings.Contains(symbol, "GOLD") {
		return 2
	}
	if strings.Contains(symbol, "EUR") || strings.Contains(symbol, "GBP") || strings.Contains(symbol, "AUD") || (strings.Contains(symbol, "USD") && !strings.Contains(symbol, "XAU")) {
		return 5
	}
	return 2
}
