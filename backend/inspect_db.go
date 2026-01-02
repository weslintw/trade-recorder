package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA table_info(trades)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Columns in 'trades' table:")
	found := false
	for rows.Next() {
		var cid int
		var name string
		var typeName string
		var notNull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &dfltValue, &pk); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s (%s)\n", name, typeName)
		if name == "color_tag" {
			found = true
		}
	}

	if found {
		fmt.Println("\nSUCCESS: 'color_tag' column FOUND.")
	} else {
		fmt.Println("\nFAILURE: 'color_tag' column NOT FOUND.")
	}
}
