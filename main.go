// MySQL 8, find and fix errant gtid transaction sets
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
	target = flag.String("t", "", "Target Host")
	help   = flag.Bool("h", false, "Print help")
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
	db3 *sql.DB
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

// connet to the source database and the target database at the same time. Show that you are connected to each source and target database
func connectToDatabase() {

	db1, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*source+":3306)/")
	//checkGtidSetSubset()
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

	db3, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*target+":3306)/")
	if err != nil {
		log.Fatal(err)
	}
	err = db3.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

// run show replica status on the target database and only print the Retrieved_Gtid_Set: and Executed_Gtid_Set: values
func showReplicaStatus() {
	rows, err := db3.Query("SHOW REPLICA STATUS")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		var retrievedGtidSet string
		var executedGtidSet string
		for i, col := range columns {
			val := values[i]
			switch col {
			case "Retrieved_Gtid_Set":
				retrievedGtidSet = string(val.([]byte))
			case "Executed_Gtid_Set":
				executedGtidSet = string(val.([]byte))
			}
		}
		fmt.Println(yellow("[+]"), "Target Retrieved_Gtid_Set: "+retrievedGtidSet)
		fmt.Println(yellow("[+]"), "Target Executed_Gtid_Set: "+executedGtidSet)
		//defer db3.Close()
	}
}

// on the source and target database, get the gtid_executed
func checkGtidSetSubset() {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string

	// connect to the source database and run the query to get the gtid_executed value
	err = db1.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&sourceGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(blue("[+]"), "Source gtid_executed: "+sourceGtidSet)
	//defer db1.Close()
	// connect to the target database and run the query to get the gtid_executed value
	err = db2.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&targetGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(yellow("[+]"), "Target gtid_executed: "+targetGtidSet)
	//defer db2.Close()

	// using a select "select gtid_subtract get the source gtid_executed value and the target gtid_executed value from the function checkGtidSetSubset()
	// Example: err = db2.QueryRow("select gtid_subtract('1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1,c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33','c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33') as errant_transactions").Scan(&errantTransactions)
	err = db2.QueryRow("select gtid_subtract('" + targetGtidSet + "','" + sourceGtidSet + "') as errant_transactions").Scan(&errantTransactions)
	if err != nil {
		log.Fatal(err)
	}
	if errantTransactions == "" {
		fmt.Println(green("[+]"), "Errant Transactions: "+errantTransactions)
	} else {
		fmt.Println(red("[-]"), "Errant Transactions: "+errantTransactions)
	}
	//defer db2.Close()

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

	// make sure the source and target flags are set
	if *source == "" || *target == "" {
		printHelp()
		os.Exit(1)
	}

	readMyCnf()
	// show that each connection is working by printing the gtid_executed value
	connectToDatabase()
	checkGtidSetSubset()
	//showReplicaStatus()
	defer db1.Close()
	defer db2.Close()
	//defer db3.Close()
}
