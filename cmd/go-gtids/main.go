package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaosHour/go-gtids/pkg/gtids"
)

// Populated by goreleaser via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	source            = flag.String("s", "", "Source Host")
	target            = flag.String("t", "", "Target Host")
	sourcePort        = flag.String("source-port", "3306", "Source MySQL port")
	targetPort        = flag.String("target-port", "3306", "Target MySQL port")
	fix               = flag.Bool("fix", false, "fix the GTID set subset issue by applying to source")
	fixReplica        = flag.Bool("fix-replica", false, "fix the GTID set subset issue by applying to replica")
	fixMissingReplica = flag.Bool("fix-missing-replica", false, "fix missing GTIDs by applying dummy transactions to replica (WARNING: skips the transactions' data)")
	dryRun            = flag.Bool("dry-run", false, "print the statements a fix would execute without running them")
	assumeYes         = flag.Bool("yes", false, "skip the confirmation prompt before applying fixes")
	showVersion       = flag.Bool("version", false, "Print version and exit")
	help              = flag.Bool("h", false, "Print help")
)

func printHelp() {
	fmt.Println("Usage: go-gtids -s <source> -t <target> [-source-port <port>] [-target-port <port>] [-fix] [-fix-replica] [-fix-missing-replica] [-dry-run] [-yes]")
	flag.PrintDefaults()
	fmt.Println("Exit codes: 0 = in sync (or fix applied), 1 = error, 2 = errant/missing transactions remain")
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-gtids %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *source == "" || *target == "" {
		printHelp()
		os.Exit(1)
	}

	if *dryRun && !*fix && !*fixReplica && !*fixMissingReplica {
		fmt.Fprintln(os.Stderr, "Note: -dry-run has no effect without -fix, -fix-replica, or -fix-missing-replica")
	}

	// Ctrl-C / SIGTERM cancels in-flight work; fix cleanup still runs to completion.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db1, db2, err := gtids.ConnectToDatabases(ctx, *source, *sourcePort, *target, *targetPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to databases: %v\n", err)
		os.Exit(1)
	}
	defer db1.Close()
	defer db2.Close()

	unresolved, err := gtids.CheckGtidSetSubset(ctx, db1, db2, *source, *target, gtids.Options{
		Fix:               *fix,
		FixReplica:        *fixReplica,
		FixMissingReplica: *fixMissingReplica,
		DryRun:            *dryRun,
		AssumeYes:         *assumeYes,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking GTID set subset: %v\n", err)
		os.Exit(1)
	}
	if unresolved {
		os.Exit(2)
	}
}
