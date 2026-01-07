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

	fmt.Println("--- USERS ---")
	urows, _ := db.Query("SELECT id, username FROM users")
	for urows.Next() {
		var id int64
		var name string
		urows.Scan(&id, &name)
		fmt.Printf("User %d: %s\n", id, name)
	}

	fmt.Println("\n--- ACCOUNTS ---")
	arows, _ := db.Query("SELECT id, user_id, name FROM accounts")
	for arows.Next() {
		var id, uid int64
		var name string
		arows.Scan(&id, &uid, &name)
		fmt.Printf("Acc %d (User %d): %s\n", id, uid, name)
	}

	fmt.Println("\n--- TRADE COUNTS BY ACCOUNT ---")
	trows, _ := db.Query("SELECT account_id, count(*) FROM trades GROUP BY account_id")
	for trows.Next() {
		var aid int64
		var count int
		trows.Scan(&aid, &count)
		fmt.Printf("Acc %d: %d trades\n", aid, count)
	}

	fmt.Println("\n--- TOTAL TRADES ---")
	var total int
	db.QueryRow("SELECT count(*) FROM trades").Scan(&total)
	fmt.Printf("Total: %d\n", total)
}
