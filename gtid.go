// gtid.go
package main

import (
	"fmt"
	"log"
	"strings"

	"database/sql"
)

func checkGtidSetSubset(db1 *sql.DB, db2 *sql.DB, source string, target string) {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string

	// connect to the source database and run the query to get the gtid_executed value
	err = db1.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&sourceGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet)

	// connect to the target Host(s) database and run the query to get the gtid_executed value
	targetHosts := strings.Split(target, ",")
	for _, targetHost := range targetHosts {
		err = db2.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&targetGtidSet)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(yellow("[+]"), "Target ->", targetHost, "gtid_executed:", targetGtidSet)

		// using a select "select gtid_subtract get the source gtid_executed value and the target Host(s) gtid_executed value from the function checkGtidSetSubset()
		err = db2.QueryRow("select gtid_subtract('" + targetGtidSet + "','" + sourceGtidSet + "') as errant_transactions").Scan(&errantTransactions)
		if err != nil {
			log.Fatal(err)
		}
		if errantTransactions == "" {
			fmt.Println(green("[+]"), "Errant Transactions:", errantTransactions)
		} else {
			fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)

		}

	}
}
