package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChaosHour/go-gtids/pkg/gtids"
)

var (
	source     = flag.String("s", "", "Source Host")
	target     = flag.String("t", "", "Target Host")
	sourcePort = flag.String("source-port", "3306", "Source MySQL port")
	targetPort = flag.String("target-port", "3306", "Target MySQL port")
	fix        = flag.Bool("fix", false, "fix the GTID set subset issue by applying to source")
	fixReplica = flag.Bool("fix-replica", false, "fix the GTID set subset issue by applying to replica")
	help       = flag.Bool("h", false, "Print help")
)

func init() {
	flag.Parse()
}

func printHelp() {
	fmt.Println("Usage: go-gtids -s <source> -t <target> [-source-port <port>] [-target-port <port>] [-fix] [-fix-replica]")
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

	db1, db2, err := gtids.ConnectToDatabases(*source, *sourcePort, *target, *targetPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to databases: %v\n", err)
		os.Exit(1)
	}
	defer db1.Close()
	defer db2.Close()

	err = gtids.CheckGtidSetSubset(db1, db2, *source, *target, *fix, *fixReplica)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking GTID set subset: %v\n", err)
		os.Exit(1)
	}
}
