package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChaosHour/go-gtids/pkg/gtids"
)

var (
	source = flag.String("s", "", "Source Host")
	target = flag.String("t", "", "Target Host")
	fix    = flag.Bool("fix", false, "fix the GTID set subset issue")
	help   = flag.Bool("h", false, "Print help")
)

func init() {
	flag.Parse()
}

func printHelp() {
	fmt.Println("Usage: go-gtids -s <source> -t <target> [-fix]")
}

func main() {
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
	gtids.CheckGtidSetSubset(gtids.Db1, gtids.Db2, *source, *target, *fix)
	defer gtids.Db1.Close()
	defer gtids.Db2.Close()
}
