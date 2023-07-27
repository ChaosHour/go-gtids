// gtid.go
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"database/sql"
)

func fixGtidSet(db1 *sql.DB, errantTransactions string) error {
	// This needs to be ran on the primary database

	// set GTID_NEXT to the errantTransactions value
	_, err := db1.Exec("SET GTID_NEXT='" + errantTransactions + "'")
	if err != nil {
		return err
	}

	// execute a BEGIN statement
	_, err = db1.Exec("BEGIN")
	if err != nil {
		return err
	}

	// execute a COMMIT statement
	_, err = db1.Exec("COMMIT")
	if err != nil {
		return err
	}

	// set GTID_NEXT back to AUTOMATIC
	_, err = db1.Exec("SET GTID_NEXT='AUTOMATIC'")
	if err != nil {
		return err
	}

	return nil
}

// on the source and target Host(s) database, get the gtid_executed
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
			// ask the user if they want to fix the errant transactions
			fmt.Println(yellow("[!]"), "Do you want to fix the errant transactions? (y/n)")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			if answer == "y" {
				// fix the errant transactions
				err = fixGtidSet(db1, errantTransactions)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(green("[+]"), "Errant transactions fixed!")
			} else {
				fmt.Println(yellow("[!]"), "Errant transactions not fixed.")
			}
		}
	}
}
