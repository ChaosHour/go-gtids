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

	gtids.ReadMyCnf()
	gtids.ConnectToDatabaseWithPort(*source, *sourcePort, *target, *targetPort)
	gtids.CheckGtidSetSubset(gtids.Db1, gtids.Db2, *source, *target, *fix, *fixReplica)
	defer gtids.Db1.Close()
	defer gtids.Db2.Close()
}
