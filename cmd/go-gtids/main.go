package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChaosHour/go-gtids/pkg/gtids"
	"github.com/fatih/color"
)

var (
	source = flag.String("s", "", "Source Host")
	target = flag.String("t", "", "Target Host")
	fix    = flag.Bool("fix", false, "fix the GTID set subset issue")
	help   = flag.Bool("h", false, "Print help")
)

var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()

func init() {
	flag.Parse()
}

func printHelp() {
	fmt.Println("Usage: go-gtids -s <source> -t <target> [-fix]")
}

func main() {
	color.NoColor = false // Force color output.

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *source == "" || *target == "" {
		printHelp()
		os.Exit(1)
	}

	gtids.ReadMyCnf()
	gtids.ConnectToDatabase(*source, *target)
	gtids.CheckGtidSetSubset(gtids.Db1, gtids.Db2, *source, *target)
	defer gtids.Db1.Close()
	defer gtids.Db2.Close()
}
