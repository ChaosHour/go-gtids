// db.go
package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db1 *sql.DB
	db2 *sql.DB
	err error
)

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() {
	file, err := ioutil.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "user") {
			os.Setenv("MYSQL_USER", strings.TrimSpace(line[5:]))
		}
		if strings.HasPrefix(line, "password") {
			os.Setenv("MYSQL_PASSWORD", strings.TrimSpace(line[9:]))
		}
	}
}

/*
// connet to the source database and the target Host(s) database at the same time. Show that you are connected to each source and target database
func connectToDatabase(source string, target string) {

	db1, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+source+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db1.Ping()
	if err != nil {
		log.Fatal(err)
	}
	db2, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+target+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db2.Ping()
	if err != nil {
		log.Fatal(err)
	}
}
*/

// connect to the source database and the target Host(s) database at the same time. Show that you are connected to each source and target database
func connectToDatabase(source string, targets string) {
	// connect to the source database
	db1, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+source+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db1.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// connect to the target Host(s) database
	targetHosts := strings.Split(targets, ",")
	for _, target := range targetHosts {
		db2, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+target+":3306)/")
		if err != nil {
			log.Fatal(err)
		}
		err = db2.Ping()
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(blue("[+]"), "Connected to target:", target)
	}
}
