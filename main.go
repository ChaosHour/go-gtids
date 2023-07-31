// main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Define flags
var (
	source = flag.String("s", "", "Source Host")
	target = flag.String("t", "", "Target Host")
	fix    = flag.Bool("fix", false, "fix the GTID set subset issue")
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

// print the help message
func printHelp() {
	fmt.Println("Usage: go-gtids -s <source> -t <target> [-fix]")
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
	connectToDatabase(*source, *target)
	checkGtidSetSubset(db1, db2, *source, *target)
	// check if the -fix flag is set
	//if *fix {
	// if the -fix flag is set, then fix the GTID set subset issue
	//	fixGtidSetSubset(db1, db2, *source, *target)
	//}
	// close the database connections

	defer db1.Close()
	defer db2.Close()
}
