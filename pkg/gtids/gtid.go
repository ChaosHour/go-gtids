// gtid.go
package gtids

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"database/sql"

	"github.com/fatih/color"
)

/*
   Copyright 2015 Shlomi Noach, courtesy Booking.com

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// define variables
var (
	singleValueInterval = regexp.MustCompile("^([0-9]+)$")
	multiValueInterval  = regexp.MustCompile("^([0-9]+)[-]([0-9]+)$")
	// gtidEntryPattern matches a single exploded GTID entry (uuid:transaction-id).
	// SET GTID_NEXT cannot be parameterized, so entries are validated against this
	// before being interpolated into the statement.
	gtidEntryPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}:[0-9]+$`)
	versionPattern   = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	green            = color.New(color.FgGreen).SprintFunc()
	red              = color.New(color.FgRed).SprintFunc()
	yellow           = color.New(color.FgYellow).SprintFunc()
	blue             = color.New(color.FgBlue).SprintFunc()
)

// OracleGtidSetEntry represents an entry in a set of GTID ranges,
// for example, the entry: "316d193c-70e5-11e5-adb2-ecf4bb2262ff:1-8935:8984-6124596" (may include gaps)
type OracleGtidSetEntry struct {
	UUID   string
	Ranges string
}

// NewOracleGtidSetEntry parses a single entry text
func NewOracleGtidSetEntry(gtidRangeString string) (*OracleGtidSetEntry, error) {
	gtidRangeString = strings.TrimSpace(gtidRangeString)
	tokens := strings.SplitN(gtidRangeString, ":", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("cannot parse OracleGtidSetEntry from %s", gtidRangeString)
	}
	if tokens[0] == "" {
		return nil, fmt.Errorf("unexpected UUID: %s", tokens[0])
	}
	if tokens[1] == "" {
		return nil, fmt.Errorf("unexpected GTID range: %s", tokens[1])
	}
	gtidRange := &OracleGtidSetEntry{UUID: tokens[0], Ranges: tokens[1]}
	return gtidRange, nil
}

// String returns a user-friendly string representation of this entry
func (oge *OracleGtidSetEntry) String() string {
	return fmt.Sprintf("%s:%s", oge.UUID, oge.Ranges)
}

// Explode returns this entry as a list of single-transaction entries
func (oge *OracleGtidSetEntry) Explode() (result [](*OracleGtidSetEntry)) {
	intervals := strings.Split(oge.Ranges, ":")
	for _, interval := range intervals {
		if submatch := multiValueInterval.FindStringSubmatch(interval); submatch != nil {
			intervalStart, _ := strconv.Atoi(submatch[1])
			intervalEnd, _ := strconv.Atoi(submatch[2])
			for i := intervalStart; i <= intervalEnd; i++ {
				result = append(result, &OracleGtidSetEntry{UUID: oge.UUID, Ranges: fmt.Sprintf("%d", i)})
			}
		} else if submatch := singleValueInterval.FindStringSubmatch(interval); submatch != nil {
			result = append(result, &OracleGtidSetEntry{UUID: oge.UUID, Ranges: interval})
		}
	}
	return result
}

// OracleGtidSet represents a set of GTID ranges as depicted by Retrieved_Gtid_Set, Executed_Gtid_Set or @@gtid_purged.
type OracleGtidSet struct {
	GtidEntries [](*OracleGtidSetEntry)
}

// Example input:  `230ea8ea-81e3-11e4-972a-e25ec4bd140a:1-10539,
// 316d193c-70e5-11e5-adb2-ecf4bb2262ff:1-8935:8984-6124596,
// 321f5c0d-70e5-11e5-adb2-ecf4bb2262ff:1-56`
func NewOracleGtidSet(gtidSet string) (res *OracleGtidSet, err error) {
	res = &OracleGtidSet{}

	gtidSet = strings.TrimSpace(gtidSet)
	if gtidSet == "" {
		return res, nil
	}
	entries := strings.Split(gtidSet, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if gtidRange, err := NewOracleGtidSetEntry(entry); err == nil {
			res.GtidEntries = append(res.GtidEntries, gtidRange)
		} else {
			return res, err
		}
	}
	return res, nil
}

// RemoveUUID removes entries that belong to given UUID.
// By way of how this works there can only be one entry matching our UUID, but we generalize.
// We keep order of entries.
func (ogs *OracleGtidSet) RemoveUUID(uuid string) (removed bool) {
	filteredEntries := [](*OracleGtidSetEntry){}
	for _, entry := range ogs.GtidEntries {
		if entry.UUID == uuid {
			removed = true
		} else {
			filteredEntries = append(filteredEntries, entry)
		}
	}
	if removed {
		ogs.GtidEntries = filteredEntries
	}
	return removed
}

// RetainUUID retains only entries that belong to given UUID.
func (ogs *OracleGtidSet) RetainUUID(uuid string) (anythingRemoved bool) {
	return ogs.RetainUUIDs([]string{uuid})
}

// RetainUUIDs retains only entries that belong to given UUIDs.
func (ogs *OracleGtidSet) RetainUUIDs(uuids []string) (anythingRemoved bool) {
	retainUUIDs := map[string]bool{}
	for _, uuid := range uuids {
		retainUUIDs[uuid] = true
	}
	filteredEntries := [](*OracleGtidSetEntry){}
	for _, entry := range ogs.GtidEntries {
		if retainUUIDs[entry.UUID] {
			filteredEntries = append(filteredEntries, entry)
		} else {
			anythingRemoved = true
		}
	}
	if anythingRemoved {
		ogs.GtidEntries = filteredEntries
	}
	return anythingRemoved
}

// SharedUUIDs returns UUIDs (range-less) that are shared between the two sets
func (ogs *OracleGtidSet) SharedUUIDs(other *OracleGtidSet) (shared []string) {
	thisUUIDs := map[string]bool{}
	for _, entry := range ogs.GtidEntries {
		thisUUIDs[entry.UUID] = true
	}
	for _, entry := range other.GtidEntries {
		if thisUUIDs[entry.UUID] {
			shared = append(shared, entry.UUID)
		}
	}
	return shared
}

// Explode returns the set as a list of single-transaction entries
func (ogs *OracleGtidSet) Explode() (result [](*OracleGtidSetEntry)) {
	for _, entries := range ogs.GtidEntries {
		result = append(result, entries.Explode()...)
	}
	return result
}

func (ogs *OracleGtidSet) String() string {
	tokens := []string{}
	for _, entry := range ogs.GtidEntries {
		tokens = append(tokens, entry.String())
	}
	return strings.Join(tokens, ",")
}

func (ogs *OracleGtidSet) IsEmpty() bool {
	return len(ogs.GtidEntries) == 0
}

// getServerInfo retrieves the server UUID and GTID_EXECUTED from a database connection
func getServerInfo(ctx context.Context, db *sql.DB) (uuid, gtidExecuted string, err error) {
	err = retryDatabaseOperation(ctx, func() error {
		return db.QueryRowContext(ctx, "SELECT @@server_uuid").Scan(&uuid)
	}, 3)
	if err != nil {
		return "", "", fmt.Errorf("failed to get server UUID: %w", err)
	}

	err = retryDatabaseOperation(ctx, func() error {
		return db.QueryRowContext(ctx, "SELECT @@GLOBAL.GTID_EXECUTED").Scan(&gtidExecuted)
	}, 3)
	if err != nil {
		return "", "", fmt.Errorf("failed to get GTID_EXECUTED: %w", err)
	}

	return uuid, gtidExecuted, nil
}

// checkErrantTransactions uses gtid_subtract to find transactions that exist on target but not on source
func checkErrantTransactions(ctx context.Context, targetGtid, sourceGtid string, db *sql.DB) (errant string, err error) {
	err = retryDatabaseOperation(ctx, func() error {
		return db.QueryRowContext(ctx, "SELECT gtid_subtract(?, ?) AS errant_transactions", targetGtid, sourceGtid).Scan(&errant)
	}, 3)
	if err != nil {
		return "", fmt.Errorf("failed to check errant transactions: %w", err)
	}
	return errant, nil
}

// checkMissingTransactions uses gtid_subtract to find transactions that exist on source but not on target
func checkMissingTransactions(ctx context.Context, sourceGtid, targetGtid string, db *sql.DB) (missing string, err error) {
	err = retryDatabaseOperation(ctx, func() error {
		return db.QueryRowContext(ctx, "SELECT gtid_subtract(?, ?) AS missing_transactions", sourceGtid, targetGtid).Scan(&missing)
	}, 3)
	if err != nil {
		return "", fmt.Errorf("failed to check missing transactions: %w", err)
	}
	return missing, nil
}

// getBinaryLogInfo retrieves the current binary log file name.
// MySQL 8.4 removed SHOW MASTER STATUS in favor of SHOW BINARY LOG STATUS
// (available since 8.2), so try the new statement first and fall back.
func getBinaryLogInfo(ctx context.Context, db *sql.DB) (logName string, err error) {
	for _, stmt := range []string{"SHOW BINARY LOG STATUS", "SHOW MASTER STATUS"} {
		logName, err = queryColumnByName(ctx, db, stmt, "File")
		if err == nil {
			return logName, nil
		}
	}
	return "", fmt.Errorf("failed to get binary log info: %w", err)
}

// queryColumnByName runs a query and returns the named column from the first row,
// scanning dynamically so column count/order differences across versions don't matter.
func queryColumnByName(ctx context.Context, db *sql.DB, query, column string) (string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("%s returned no rows", query)
	}
	columns, err := scanRowAsMap(rows)
	if err != nil {
		return "", err
	}
	value, ok := columns[column]
	if !ok {
		return "", fmt.Errorf("column %s not found in %s output", column, query)
	}
	return value, nil
}

// scanRowAsMap scans the current row of rows into a column-name -> string map.
func scanRowAsMap(rows *sql.Rows) (map[string]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return nil, err
	}
	columns := make(map[string]string, len(cols))
	for i, colName := range cols {
		switch v := values[i].(type) {
		case nil:
			columns[colName] = "NULL"
		case []byte:
			columns[colName] = string(v)
		case string:
			columns[colName] = v
		default:
			columns[colName] = fmt.Sprintf("%v", v)
		}
	}
	return columns, nil
}

// parseErrantTransactions explodes the errant GTID set into individual entries
func parseErrantTransactions(errant string) ([]string, error) {
	oracleGtidSet, err := NewOracleGtidSet(errant)
	if err != nil {
		return nil, err
	}

	var entries []string
	gtidEntries := oracleGtidSet.Explode()
	for _, entry := range gtidEntries {
		entries = append(entries, fmt.Sprintf("%s:%s", entry.UUID, entry.Ranges))
	}
	return entries, nil
}

// replicationCommandsForVersion picks STOP/START SLAVE vs REPLICA statements.
// MySQL 8.0.22 introduced the REPLICA statements; 8.4 removed the SLAVE ones.
// MariaDB keeps the SLAVE statements (and has an incompatible GTID scheme anyway).
func replicationCommandsForVersion(version string) (stopCmd, startCmd, statusCmd string) {
	if !strings.Contains(strings.ToLower(version), "mariadb") {
		if m := versionPattern.FindStringSubmatch(version); m != nil {
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			patch, _ := strconv.Atoi(m[3])
			if major > 8 || (major == 8 && (minor > 0 || patch >= 22)) {
				return "STOP REPLICA", "START REPLICA", "SHOW REPLICA STATUS"
			}
		}
	}
	return "STOP SLAVE", "START SLAVE", "SHOW SLAVE STATUS"
}

// determineReplicationCommands determines the correct replication commands based on MySQL version
func determineReplicationCommands(ctx context.Context, db *sql.DB) (stopCmd, startCmd, statusCmd string, err error) {
	var version string
	err = retryDatabaseOperation(ctx, func() error {
		return db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	}, 3)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get MySQL version: %w", err)
	}
	stopCmd, startCmd, statusCmd = replicationCommandsForVersion(version)
	return stopCmd, startCmd, statusCmd, nil
}

// Options controls the fix behavior of CheckGtidSetSubset.
type Options struct {
	Fix               bool // apply errant GTIDs to the source
	FixReplica        bool // apply errant GTIDs to the replica
	FixMissingReplica bool // mark missing GTIDs as executed on the replica (skips their data)
	DryRun            bool // print the statements a fix would run without executing them
	AssumeYes         bool // skip the confirmation prompt
}

// confirmAction prompts on stdin before a destructive operation. Non-interactive
// runs (closed stdin, cron) hit EOF and abort — pass -yes to skip the prompt.
func confirmAction(prompt string, assumeYes bool) bool {
	if assumeYes {
		return true
	}
	fmt.Printf("%s %s Type 'yes' to continue (or pass -yes to skip this prompt): ", yellow("[?]"), prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Println("\nNo confirmation received — aborting.")
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// printGtidStatements prints the empty-transaction sequence a fix would execute.
func printGtidStatements(entries []string) {
	for _, entry := range entries {
		fmt.Printf("    SET GTID_NEXT='%s'; BEGIN; COMMIT;\n", entry)
	}
	fmt.Println("    SET GTID_NEXT='AUTOMATIC';")
}

// dryRunSourceFix prints what applyGtidsToSource would execute.
func dryRunSourceFix(entries []string) {
	fmt.Println(yellow("[dry-run]"), "Would execute on source (single pinned session):")
	printGtidStatements(entries)
}

// dryRunReplicaFix prints what applyGtidsToReplica would execute.
func dryRunReplicaFix(ctx context.Context, db *sql.DB, entries []string) error {
	stopCmd, startCmd, _, err := determineReplicationCommands(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to determine replication commands: %w", err)
	}
	fmt.Println(yellow("[dry-run]"), "Would execute on replica (single pinned session):")
	fmt.Printf("    %s;\n", stopCmd)
	fmt.Println("    SET SESSION sql_log_bin = 0;")
	printGtidStatements(entries)
	fmt.Println("    SET SESSION sql_log_bin = 1;")
	fmt.Printf("    %s;\n", startCmd)
	return nil
}

// applyGtidEntries injects an empty transaction for each GTID entry on a single
// pinned connection. GTID_NEXT is session-scoped, so every statement in the
// sequence must run on the same connection — never on the *sql.DB pool.
// GTID_NEXT is always reset to AUTOMATIC before returning, even on failure.
func applyGtidEntries(ctx context.Context, conn *sql.Conn, entries []string, fixLocation string) (err error) {
	defer func() {
		// Cleanup must run even if ctx was cancelled (e.g. Ctrl-C mid-apply).
		cleanupCtx := context.WithoutCancel(ctx)
		if _, resetErr := conn.ExecContext(cleanupCtx, "SET GTID_NEXT='AUTOMATIC'"); resetErr != nil && err == nil {
			err = fmt.Errorf("failed to reset GTID_NEXT to AUTOMATIC: %w", resetErr)
		}
	}()

	for _, entry := range entries {
		if !gtidEntryPattern.MatchString(entry) {
			return fmt.Errorf("refusing to apply invalid GTID entry %q", entry)
		}
		if _, err := conn.ExecContext(ctx, "SET GTID_NEXT='"+entry+"'"); err != nil {
			return fmt.Errorf("failed to set GTID_NEXT for %s: %w", entry, err)
		}
		if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", entry, err)
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return fmt.Errorf("failed to commit transaction for %s: %w", entry, err)
		}
		fmt.Printf("Applied entry to %s: %s\n", fixLocation, entry)
	}
	return nil
}

// applyGtidsToSource injects empty transactions on the source (binary logging
// stays on so the GTIDs replicate downstream, where they are auto-skipped).
func applyGtidsToSource(ctx context.Context, db *sql.DB, entries []string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Close()

	fmt.Println("Applying errant GTIDs to source...")
	return applyGtidEntries(ctx, conn, entries, "source")
}

// applyGtidsToReplica stops replication, injects empty transactions with binary
// logging disabled on the session, then restarts replication and verifies it.
// Replication is restarted even if applying the entries fails or ctx is cancelled.
func applyGtidsToReplica(ctx context.Context, db *sql.DB, entries []string, fixLocation string, errantTransactions string) error {
	stopCmd, startCmd, statusCmd, err := determineReplicationCommands(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to determine replication commands: %w", err)
	}

	// Cleanup must run even if ctx is cancelled mid-fix (e.g. Ctrl-C).
	cleanupCtx := context.WithoutCancel(ctx)

	fmt.Printf("Stopping replication on %s...\n", fixLocation)
	if _, err := db.ExecContext(ctx, stopCmd); err != nil {
		return fmt.Errorf("failed to stop replication on %s: %w", fixLocation, err)
	}

	applyErr := func() error {
		conn, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("failed to acquire connection: %w", err)
		}
		defer conn.Close()

		fmt.Printf("Disabling binary logging on %s...\n", fixLocation)
		if _, err := conn.ExecContext(ctx, "SET SESSION sql_log_bin = 0"); err != nil {
			log.Printf("Warning: failed to disable binary logging: %v", err)
		}
		defer func() {
			if _, err := conn.ExecContext(cleanupCtx, "SET SESSION sql_log_bin = 1"); err != nil {
				log.Printf("Warning: failed to re-enable binary logging: %v", err)
			}
		}()

		fmt.Printf("Applying GTIDs to %s...\n", fixLocation)
		return applyGtidEntries(ctx, conn, entries, fixLocation)
	}()

	fmt.Printf("Starting replication on %s...\n", fixLocation)
	if _, err := db.ExecContext(cleanupCtx, startCmd); err != nil {
		if applyErr != nil {
			return fmt.Errorf("applying GTIDs failed (%v) and replication could not be restarted on %s: %w", applyErr, fixLocation, err)
		}
		return fmt.Errorf("failed to start replication on %s: %w", fixLocation, err)
	}
	if applyErr != nil {
		return fmt.Errorf("failed to apply GTIDs to %s: %w", fixLocation, applyErr)
	}

	fmt.Println("Waiting for replication to initialize...")
	if err := sleepCtx(ctx, 2*time.Second); err != nil {
		return err
	}

	fmt.Printf("Verifying replication status on %s...\n", fixLocation)
	return verifyReplicationStatus(ctx, db, statusCmd, fixLocation, errantTransactions)
}

// verifyReplicationStatus checks and reports the replication status after fixes
func verifyReplicationStatus(ctx context.Context, db *sql.DB, statusCmd string, fixLocation string, errantTransactions string) error {
	var rows *sql.Rows
	err := retryDatabaseOperation(ctx, func() error {
		var err error
		rows, err = db.QueryContext(ctx, statusCmd)
		return err
	}, 3)
	if err != nil {
		return fmt.Errorf("failed to query replication status: %w", err)
	}
	defer rows.Close()

	columns := map[string]string{}
	if rows.Next() {
		if columns, err = scanRowAsMap(rows); err != nil {
			return err
		}
	}

	// Handle both MySQL 5.7/8.0 (Master/Slave) and 8.0.22+ (Source/Replica) column names.
	pick := func(names ...string) string {
		for _, name := range names {
			if v, ok := columns[name]; ok {
				return v
			}
		}
		return ""
	}
	masterHost := pick("Master_Host", "Source_Host")
	ioRunning := pick("Slave_IO_Running", "Replica_IO_Running")
	sqlRunning := pick("Slave_SQL_Running", "Replica_SQL_Running")
	secondsBehind := pick("Seconds_Behind_Master", "Seconds_Behind_Source")
	sqlRunningState := pick("Slave_SQL_Running_State", "Replica_SQL_Running_State")
	retrievedGtidSet := pick("Retrieved_Gtid_Set")
	executedGtidSet := pick("Executed_Gtid_Set")

	fmt.Println("\nReplication Status:")
	fmt.Printf("                 Master_Host: %s\n", masterHost)
	fmt.Printf("            Slave_IO_Running: %s\n", ioRunning)
	fmt.Printf("           Slave_SQL_Running: %s\n", sqlRunning)
	fmt.Printf("       Seconds_Behind_Master: %s\n", secondsBehind)
	fmt.Printf("     Slave_SQL_Running_State: %s\n", sqlRunningState)
	fmt.Printf("          Retrieved_Gtid_Set: %s\n", retrievedGtidSet)
	fmt.Printf("           Executed_Gtid_Set: %s\n", executedGtidSet)

	if ioRunning == "Yes" && sqlRunning == "Yes" {
		fmt.Printf("%s Replication is running on %s\n", green("[+]"), fixLocation)
		if errantTransactions != "" {
			fmt.Printf("%s Note: Applied errant GTID %s to %s, but it will still show as errant until applied to source\n",
				blue("[i]"), errantTransactions, fixLocation)
		}
	} else {
		fmt.Printf("%s Replication issue on %s\n", red("[-]"), fixLocation)
	}

	return nil
}

// CheckGtidSetSubset compares GTID_EXECUTED between source and target, reports
// errant/missing transactions, and optionally fixes them per opts.
// unresolved reports whether errant or missing transactions remain after the run
// (found in check-only mode, shown in dry-run, or left when a fix was declined).
func CheckGtidSetSubset(ctx context.Context, db1 *sql.DB, db2 *sql.DB, source string, target string, opts Options) (unresolved bool, err error) {
	sourceUUID, sourceGtidSet, err := getServerInfo(ctx, db1)
	if err != nil {
		return false, fmt.Errorf("failed to get source server info: %w", err)
	}

	targetUUID, targetGtidSet, err := getServerInfo(ctx, db2)
	if err != nil {
		return false, fmt.Errorf("failed to get target server info: %w", err)
	}

	fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet)
	fmt.Println(blue("[+]"), "server_uuid:", sourceUUID)
	fmt.Println(yellow("[+]"), "Target ->", target, "gtid_executed:", targetGtidSet)
	fmt.Println(yellow("[+]"), "server_uuid:", targetUUID)

	errantTransactions, err := checkErrantTransactions(ctx, targetGtidSet, sourceGtidSet, db2)
	if err != nil {
		return false, fmt.Errorf("failed to check errant transactions: %w", err)
	}

	if errantTransactions == "" {
		fmt.Println(green("[+]"), "No Errant Transactions:", errantTransactions)
	} else {
		unresolved = true
		fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)

		logName, err := getBinaryLogInfo(ctx, db2)
		if err != nil {
			return unresolved, fmt.Errorf("failed to get binary log info: %w", err)
		}
		fmt.Println(yellow("[-]"), "Errant Transaction Found in Log Name:", logName)

		if opts.Fix || opts.FixReplica {
			entries, err := parseErrantTransactions(errantTransactions)
			if err != nil {
				return unresolved, fmt.Errorf("failed to parse errant transactions: %w", err)
			}

			if len(entries) > 0 {
				switch {
				case opts.FixReplica && opts.DryRun:
					if err := dryRunReplicaFix(ctx, db2, entries); err != nil {
						return unresolved, err
					}
				case opts.FixReplica:
					prompt := fmt.Sprintf("About to apply %d empty transaction(s) on the REPLICA %s (replication will be stopped and restarted).", len(entries), target)
					if !confirmAction(prompt, opts.AssumeYes) {
						fmt.Println(yellow("[i]"), "Skipped applying errant GTIDs to replica.")
						break
					}
					if err := applyGtidsToReplica(ctx, db2, entries, "replica", errantTransactions); err != nil {
						return unresolved, err
					}
					unresolved = false
				case opts.DryRun:
					dryRunSourceFix(entries)
				default:
					prompt := fmt.Sprintf("About to apply %d empty transaction(s) on the SOURCE %s (they will replicate downstream).", len(entries), source)
					if !confirmAction(prompt, opts.AssumeYes) {
						fmt.Println(yellow("[i]"), "Skipped applying errant GTIDs to source.")
						break
					}
					if err := applyGtidsToSource(ctx, db1, entries); err != nil {
						return unresolved, err
					}
					unresolved = false
				}
			}
		}
	}

	if opts.FixMissingReplica {
		missingGtids, err := checkMissingTransactions(ctx, sourceGtidSet, targetGtidSet, db1)
		if err != nil {
			return unresolved, fmt.Errorf("failed to check missing transactions: %w", err)
		}
		if missingGtids != "" {
			fmt.Println(red("[-]"), "Missing GTIDs:", missingGtids)
			fmt.Println(red("[!]"), "WARNING: injecting empty transactions for missing GTIDs marks them as")
			fmt.Println(red("[!]"), "executed WITHOUT applying their data — the source will never resend them.")
			fmt.Println(red("[!]"), "The skipped transactions' data must be synced separately (e.g. data-diff).")

			entries, err := parseErrantTransactions(missingGtids)
			if err != nil {
				return unresolved, fmt.Errorf("failed to parse missing GTIDs: %w", err)
			}

			if opts.DryRun {
				unresolved = true
				if err := dryRunReplicaFix(ctx, db2, entries); err != nil {
					return unresolved, err
				}
			} else {
				prompt := fmt.Sprintf("About to mark %d missing transaction(s) as executed on the REPLICA %s WITHOUT applying their data.", len(entries), target)
				if !confirmAction(prompt, opts.AssumeYes) {
					fmt.Println(yellow("[i]"), "Skipped applying missing GTIDs to replica.")
					return true, nil
				}
				if err := applyGtidsToReplica(ctx, db2, entries, "replica", ""); err != nil {
					return unresolved, fmt.Errorf("failed to apply missing GTID fixes: %w", err)
				}
			}
		} else {
			fmt.Println(green("[+]"), "No Missing GTIDs")
		}
	}
	return unresolved, nil
}
