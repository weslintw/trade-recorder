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

	var pnl sql.NullFloat64
	var pnlSeries sql.NullString

	db.QueryRow("SELECT pnl, pnl_series FROM trades WHERE id = 3113").Scan(&pnl, &pnlSeries)

	fmt.Printf("PnL: %v (Valid: %v)\n", pnl.Float64, pnl.Valid)
	if pnlSeries.Valid {
		fmt.Printf("PnL Series: %s\n", pnlSeries.String)
	} else {
		fmt.Printf("PnL Series: NULL\n")
	}
}
