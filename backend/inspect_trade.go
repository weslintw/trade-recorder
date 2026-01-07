package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var accountId int64
	var symbol, side string
	var pnl sql.NullFloat64
	var exitPrice sql.NullFloat64
	var lotSizeInDB float64
	var entryPrice float64

	err = db.QueryRow("SELECT account_id, symbol, side, pnl, exit_price, lot_size, entry_price FROM trades WHERE ticket = 'ctrader-pos-257101309'").Scan(
		&accountId, &symbol, &side, &pnl, &exitPrice, &lotSizeInDB, &entryPrice)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Trade Details:\n")
	fmt.Printf("Account ID: %d\n", accountId)
	fmt.Printf("Symbol: '%s'\n", symbol)
	fmt.Printf("Side: '%s'\n", side)
	fmt.Printf("PnL in DB: %v\n", pnl)
	fmt.Printf("Exit Price Valid: %v\n", exitPrice.Valid)
	fmt.Printf("Lot Size (from DB col lot_size): %f\n", lotSizeInDB)
	fmt.Printf("Entry Price: %f\n", entryPrice)
}
