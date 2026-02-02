package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./backend/data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Clean up Trades Notes
	fmt.Println("Cleaning up trade notes...")
	rows, err := db.Query("SELECT id, notes FROM trades")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Patterns to remove
	reMyfxbookSync := regexp.MustCompile(`\(Myfxbook Sync: \[sl [0-9.]+\d\]\)`)
	reMyfxbookTitle := regexp.MustCompile(`Myfxbook 同步: `)
	reFTMO := regexp.MustCompile(`FTMO CSV 匯入: Ticket \d+`)
	reTicket := regexp.MustCompile(`\(Ticket: \d+\)`)

	count := 0
	for rows.Next() {
		var id int64
		var notes string
		if err := rows.Scan(&id, &notes); err != nil {
			continue
		}

		orig := notes
		notes = reMyfxbookSync.ReplaceAllString(notes, "")
		notes = reMyfxbookTitle.ReplaceAllString(notes, "")
		notes = reFTMO.ReplaceAllString(notes, "")
		notes = reTicket.ReplaceAllString(notes, "")
		notes = strings.ReplaceAll(notes, "Myfxbook CSV 匯入", "")
		notes = strings.TrimSpace(notes)

		if notes != orig {
			_, err = db.Exec("UPDATE trades SET notes = ? WHERE id = ?", notes, id)
			if err == nil {
				count++
			}
		}
	}
	fmt.Printf(" Updated %d trade notes.\n", count)

	// 2. Clean up Account Names (if any)
	fmt.Println("Cleaning up account names...")
	aRows, err := db.Query("SELECT id, name FROM accounts")
	if err != nil {
		log.Fatal(err)
	}
	defer aRows.Close()

	acount := 0
	for aRows.Next() {
		var id int64
		var name string
		if err := aRows.Scan(&id, &name); err != nil {
			continue
		}

		orig := name
		name = reMyfxbookSync.ReplaceAllString(name, "")
		name = reMyfxbookTitle.ReplaceAllString(name, "")
		name = reFTMO.ReplaceAllString(name, "")
		name = strings.TrimSpace(name)

		if name != orig {
			_, err = db.Exec("UPDATE accounts SET name = ? WHERE id = ?", name, id)
			if err == nil {
				acount++
			}
		}
	}
	fmt.Printf(" Updated %d account names.\n", acount)

	fmt.Println("Database cleanup complete.")
}
