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
	source  = flag.String("s", "", "Source Host")
	target  = flag.String("t", "", "Target Host(s) (comma-separated)")
	fixgtid = flag.String("fix", "", "Fix GTID set")
	help    = flag.Bool("h", false, "Print help")
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
	connectToDatabase(*source, *target)
	checkGtidSetSubset(db1, db2, *source, *target)
	// check if the -fix flag is set

	defer db1.Close()
	defer db2.Close()
}
