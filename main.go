// MySQL find and fix errant gtid transaction sets
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source = flag.String("s", "", "Source Host")
	target = flag.String("t", "", "Target Host(s) (comma-separated)")
	//target = flag.String("t", "", "Target Host")
	help = flag.Bool("h", false, "Print help")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()

// parse flags
func init() {
	flag.Parse()
}

// global variables
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

// connet to the source database and the target Host(s) database at the same time. Show that you are connected to each source and target database
func connectToDatabase() {

	db1, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*source+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db1.Ping()
	if err != nil {
		log.Fatal(err)
	}
	db2, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*target+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db2.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

// on the source and target Host(s) database, get the gtid_executed
func checkGtidSetSubset() {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string

	// connect to the source database and run the query to get the gtid_executed value
	err = db1.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&sourceGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(blue("[+]"), "Source ->", *source, "gtid_executed:", sourceGtidSet)

	// connect to the target Host(s) database and run the query to get the gtid_executed value
	targetHosts := strings.Split(*target, ",")
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

// print the help message
func printHelp() {
	fmt.Println("Usage: ./go-gtids -s < source host> -t <target host>")
}

// main function
func main() {
	if *help {
		printHelp()
		os.Exit(0)
	}

	// make sure the source and target Host(s) flags are set
	if *source == "" || *target == "" {
		printHelp()
		os.Exit(1)
	}

	readMyCnf()
	// show that each connection is working by printing the gtid_executed value
	connectToDatabase()
	checkGtidSetSubset()
	defer db1.Close()
	defer db2.Close()
}
