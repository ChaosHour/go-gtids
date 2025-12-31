// gtid.go
package gtids

import (
	"fmt"
	"log"
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
	green               = color.New(color.FgGreen).SprintFunc()
	red                 = color.New(color.FgRed).SprintFunc()
	yellow              = color.New(color.FgYellow).SprintFunc()
	blue                = color.New(color.FgBlue).SprintFunc()
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

// String returns a user-friendly string representation of this entry
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

// String returns a user-friendly string representation of this entry
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

// function to check the GTID set subset issue and fix it if the -fix flag is set
// getServerInfo retrieves the server UUID and GTID_EXECUTED from a database connection
func getServerInfo(db *sql.DB) (uuid, gtidExecuted string, err error) {
	err = retryDatabaseOperation(func() error {
		return db.QueryRow("SELECT @@server_uuid").Scan(&uuid)
	}, 3)
	if err != nil {
		return "", "", fmt.Errorf("failed to get server UUID: %w", err)
	}

	err = retryDatabaseOperation(func() error {
		return db.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&gtidExecuted)
	}, 3)
	if err != nil {
		return "", "", fmt.Errorf("failed to get GTID_EXECUTED: %w", err)
	}

	return uuid, gtidExecuted, nil
}

// checkErrantTransactions uses gtid_subtract to find transactions that exist on target but not on source
func checkErrantTransactions(targetGtid, sourceGtid string, db *sql.DB) (errant string, err error) {
	err = retryDatabaseOperation(func() error {
		return db.QueryRow("select gtid_subtract('" + targetGtid + "','" + sourceGtid + "') as errant_transactions").Scan(&errant)
	}, 3)
	if err != nil {
		return "", fmt.Errorf("failed to check errant transactions: %w", err)
	}
	return errant, nil
}

// getBinaryLogInfo retrieves binary log information from SHOW MASTER STATUS
func getBinaryLogInfo(db *sql.DB) (logName string, executedGtidSet string, err error) {
	var pos int
	var Binlog_Do_DB string
	var Binlog_Ignore_DB string
	err = retryDatabaseOperation(func() error {
		return db.QueryRow("SHOW MASTER STATUS").Scan(&logName, &pos, &Binlog_Do_DB, &Binlog_Ignore_DB, &executedGtidSet)
	}, 3)
	if err != nil {
		return "", "", fmt.Errorf("failed to get binary log info: %w", err)
	}
	return logName, executedGtidSet, nil
}

// parseErrantTransactions explodes the errant GTID set into individual entries
func parseErrantTransactions(errant string) ([]string, error) {
	gtidSet := errant
	oracleGtidSet, err := NewOracleGtidSet(gtidSet)
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

// determineReplicationCommands determines the correct replication commands based on MySQL version
func determineReplicationCommands(db *sql.DB) (stopCmd, startCmd, statusCmd string, err error) {
	var version string
	err = retryDatabaseOperation(func() error {
		return db.QueryRow("SELECT VERSION()").Scan(&version)
	}, 3)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get MySQL version: %w", err)
	}

	// MySQL 8.0.22+ uses STOP REPLICA, older versions use STOP SLAVE
	if strings.Contains(version, "8.0") {
		// Extract version number to check if >= 8.0.22
		re := regexp.MustCompile(`8\.0\.(\d+)`)
		matches := re.FindStringSubmatch(version)
		if len(matches) > 1 {
			if minorVersion, err := strconv.Atoi(matches[1]); err == nil && minorVersion >= 22 {
				return "STOP REPLICA", "START REPLICA", "SHOW REPLICA STATUS", nil
			}
		}
	}
	return "STOP SLAVE", "START SLAVE", "SHOW SLAVE STATUS", nil
}

// applyGtidFixes applies the errant GTID entries to the specified database
func applyGtidFixes(db *sql.DB, entries []string, fixLocation string) error {
	for _, entry := range entries {
		err := retryDatabaseOperation(func() error {
			_, err := db.Exec("SET GTID_NEXT='" + entry + "'")
			return err
		}, 3)
		if err != nil {
			return fmt.Errorf("failed to set GTID_NEXT for %s: %w", entry, err)
		}

		err = retryDatabaseOperation(func() error {
			_, err := db.Exec("BEGIN")
			return err
		}, 3)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", entry, err)
		}

		err = retryDatabaseOperation(func() error {
			_, err := db.Exec("COMMIT")
			return err
		}, 3)
		if err != nil {
			return fmt.Errorf("failed to commit transaction for %s: %w", entry, err)
		}

		err = retryDatabaseOperation(func() error {
			_, err := db.Exec("SET GTID_NEXT='AUTOMATIC'")
			return err
		}, 3)
		if err != nil {
			return fmt.Errorf("failed to reset GTID_NEXT for %s: %w", entry, err)
		}

		fmt.Printf("Applied entry to %s: %s\n", fixLocation, entry)
	}
	return nil
}

// verifyReplicationStatus checks and reports the replication status after fixes
func verifyReplicationStatus(db *sql.DB, statusCmd string, fixLocation string, errantTransactions string) error {
	// Try a simpler query first to verify connection is working
	var testVar string
	err := retryDatabaseOperation(func() error {
		return db.QueryRow("SELECT @@version").Scan(&testVar)
	}, 3)
	if err != nil {
		fmt.Printf("Debug: Error querying version: %v\n", err)
	} else {
		fmt.Printf("Debug: Connected to MySQL version: %s\n", testVar)
	}

	// Enhanced replication status reporting with more robust handling
	var masterHost, retrievedGtidSet, executedGtidSet, sqlRunningState string
	var ioRunning, sqlRunning string
	var secondsBehind sql.NullInt64

	// Get column data as a map for easier access
	columns := make(map[string]string)

	// Use SHOW SLAVE STATUS directly - works across all versions
	fmt.Printf("Debug: Status command: %s\n", statusCmd)
	var rows *sql.Rows
	err = retryDatabaseOperation(func() error {
		var err error
		rows, err = db.Query(statusCmd)
		return err
	}, 3)
	if err != nil {
		return fmt.Errorf("failed to query replication status: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		// Get column names
		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		fmt.Printf("Debug: Found %d columns in status output\n", len(cols))
		fmt.Printf("Debug: Column names: %v\n", cols)

		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(cols))
		scanArgs := make([]interface{}, len(cols))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// Scan the result into the values slice
		err = rows.Scan(scanArgs...)
		if err != nil {
			return err
		}

		// Build a map of column name to string value
		for i, colName := range cols {
			var strVal string
			val := values[i]

			if val == nil {
				strVal = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					strVal = string(v)
				case string:
					strVal = v
				case int64:
					strVal = fmt.Sprintf("%d", v)
					if colName == "Seconds_Behind_Master" {
						secondsBehind.Int64 = v
						secondsBehind.Valid = true
					}
				default:
					strVal = fmt.Sprintf("%v", v)
				}
			}
			columns[colName] = strVal
		}

		// Extract the values we need - handle both MySQL 5.7 and 8.0+ column names
		if mh, ok := columns["Master_Host"]; ok {
			masterHost = mh
		} else if mh, ok := columns["Source_Host"]; ok {
			masterHost = mh
		}

		if io, ok := columns["Slave_IO_Running"]; ok {
			ioRunning = io
		} else if io, ok := columns["Replica_IO_Running"]; ok {
			ioRunning = io
		}

		if sql, ok := columns["Slave_SQL_Running"]; ok {
			sqlRunning = sql
		} else if sql, ok := columns["Replica_SQL_Running"]; ok {
			sqlRunning = sql
		}

		if sbm, ok := columns["Seconds_Behind_Master"]; ok && sbm != "NULL" {
			if sbmInt, err := strconv.ParseInt(sbm, 10, 64); err == nil {
				secondsBehind.Int64 = sbmInt
				secondsBehind.Valid = true
			}
		} else if sbm, ok := columns["Seconds_Behind_Source"]; ok && sbm != "NULL" {
			if sbmInt, err := strconv.ParseInt(sbm, 10, 64); err == nil {
				secondsBehind.Int64 = sbmInt
				secondsBehind.Valid = true
			}
		}

		if state, ok := columns["Slave_SQL_Running_State"]; ok {
			sqlRunningState = state
		} else if state, ok := columns["Replica_SQL_Running_State"]; ok {
			sqlRunningState = state
		}

		if rgs, ok := columns["Retrieved_Gtid_Set"]; ok {
			retrievedGtidSet = rgs
		}

		if egs, ok := columns["Executed_Gtid_Set"]; ok {
			executedGtidSet = egs
		}
	}

	// Print comprehensive replication status report
	fmt.Println("\nReplication Status:")
	fmt.Printf("                 Master_Host: %s\n", masterHost)
	fmt.Printf("            Slave_IO_Running: %s\n", ioRunning)
	fmt.Printf("           Slave_SQL_Running: %s\n", sqlRunning)
	if secondsBehind.Valid {
		fmt.Printf("       Seconds_Behind_Master: %d\n", secondsBehind.Int64)
	} else {
		fmt.Printf("       Seconds_Behind_Master: NULL\n")
	}
	fmt.Printf("     Slave_SQL_Running_State: %s\n", sqlRunningState)
	fmt.Printf("          Retrieved_Gtid_Set: %s\n", retrievedGtidSet)
	fmt.Printf("           Executed_Gtid_Set: %s\n", executedGtidSet)

	// Still keep the simple status indicator
	if ioRunning == "Yes" && sqlRunning == "Yes" {
		fmt.Printf("%s Replication is running on %s\n", green("[+]"), fixLocation)
		fmt.Printf("%s Note: Applied errant GTID %s to %s, but it will still show as errant until applied to source\n",
			blue("[i]"), errantTransactions, fixLocation)
	} else {
		fmt.Printf("%s Replication issue on %s\n", red("[-]"), fixLocation)
	}

	return nil
}

func CheckGtidSetSubset(db1 *sql.DB, db2 *sql.DB, source string, target string, fix bool, fixReplica bool) error {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string
	var sourceUUID, targetUUID string

	// Get server information for source
	sourceUUID, sourceGtidSet, err := getServerInfo(db1)
	if err != nil {
		return fmt.Errorf("failed to get source server info: %w", err)
	}

	// Get server information for target
	targetUUID, targetGtidSet, err = getServerInfo(db2)
	if err != nil {
		return fmt.Errorf("failed to get target server info: %w", err)
	}

	// Print source information
	fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet)
	fmt.Println(blue("[+]"), "server_uuid:", sourceUUID)

	// Process each target host (though currently only one db2 connection is used)
	targetHosts := strings.Split(target, ",")
	for _, targetHost := range targetHosts {
		// Re-get target GTID set (in case of multiple targets, but logic seems to assume single target)
		_, targetGtidSet, err = getServerInfo(db2)
		if err != nil {
			return fmt.Errorf("failed to get target server info for %s: %w", targetHost, err)
		}

		fmt.Println(yellow("[+]"), "Target ->", targetHost, "gtid_executed:", targetGtidSet)
		fmt.Println(yellow("[+]"), "server_uuid:", targetUUID)

		// Check for errant transactions
		errantTransactions, err = checkErrantTransactions(targetGtidSet, sourceGtidSet, db2)
		if err != nil {
			return fmt.Errorf("failed to check errant transactions: %w", err)
		}

		if errantTransactions == "" {
			fmt.Println(green("[+]"), "No Errant Transactions:", errantTransactions)
		} else {
			fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)

			// Get binary log information
			logName, executedGtidSet, err := getBinaryLogInfo(db2)
			if err != nil {
				return fmt.Errorf("failed to get binary log info: %w", err)
			}

			// Preserve the exact regex logic (even if it seems incorrect)
			re := regexp.MustCompile("Errant Transactions: ([0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}:[0-9]+)")
			matches := re.FindStringSubmatch(executedGtidSet)
			if len(matches) > 0 {
				errantTransactions = matches[1]
			}
			fmt.Println(yellow("[-]"), "Errant Transaction Found in Log Name:", logName)

			// Parse errant transactions into entries
			entries, err := parseErrantTransactions(errantTransactions)
			if err != nil {
				return fmt.Errorf("failed to parse errant transactions: %w", err)
			}

			// Only attempt fixes if there are errant transactions AND fix flag is set
			if (fix || fixReplica) && len(entries) > 0 {
				var targetDB *sql.DB
				var fixLocation string
				var stopCmd, startCmd, statusCmd string

				if fixReplica {
					targetDB = db2
					fixLocation = "replica"

					fmt.Printf("Debug: fixReplica is true, will apply to replica\n")
					fmt.Printf("Stopping replication on %s...\n", fixLocation)

					// Determine replication commands based on MySQL version
					stopCmd, startCmd, statusCmd, err = determineReplicationCommands(targetDB)
					if err != nil {
						return fmt.Errorf("failed to determine replication commands: %w", err)
					}

					_, err = targetDB.Exec(stopCmd)
					if err != nil {
						return fmt.Errorf("failed to stop replication on %s: %w", fixLocation, err)
					}
					fmt.Printf("Replication stopped on %s\n", fixLocation)

					// Make sure binary logging is disabled for this session
					fmt.Printf("Disabling binary logging on %s...\n", fixLocation)
					_, err = targetDB.Exec("SET sql_log_bin = 0")
					if err != nil {
						log.Printf("Warning: Failed to disable binary logging: %v\n", err)
						// Continue anyway, as this might be allowed in some setups
					}
				} else {
					targetDB = db1
					fixLocation = "source"
					if fix {
						fmt.Printf("Debug: fix is true, will apply to source\n")
					} else {
						fmt.Printf("Debug: Neither fix nor fixReplica is true\n")
					}
				}

				fmt.Printf("Applying errant GTIDs to %s...\n", fixLocation)

				// Actually disable binary logging AGAIN right before applying GTIDs if we're on replica
				if fixReplica {
					_, err := targetDB.Exec("SET SESSION sql_log_bin = 0")
					if err != nil {
						log.Printf("Warning: Failed to disable binary logging (retry): %v\n", err)
					}
				}

				// Apply the GTID fixes
				err = applyGtidFixes(targetDB, entries, fixLocation)
				if err != nil {
					return fmt.Errorf("failed to apply GTID fixes to %s: %w", fixLocation, err)
				}

				if fixReplica {
					// Re-enable binary logging
					fmt.Printf("Re-enabling binary logging on %s...\n", fixLocation)
					_, err = targetDB.Exec("SET sql_log_bin = 1")
					if err != nil {
						return fmt.Errorf("failed to re-enable binary logging on %s: %w", fixLocation, err)
					}

					// Start replication after applying GTIDs
					fmt.Printf("Starting replication on %s...\n", fixLocation)
					_, err = targetDB.Exec(startCmd)
					if err != nil {
						return fmt.Errorf("failed to start replication on %s: %w", fixLocation, err)
					}

					// Add a delay to allow replication to start up fully
					fmt.Printf("Waiting for replication to initialize...\n")
					time.Sleep(2 * time.Second)

					// Verify replication is working
					fmt.Printf("Verifying replication status on %s...\n", fixLocation)

					// Check replication status
					err = verifyReplicationStatus(targetDB, statusCmd, fixLocation, errantTransactions)
					if err != nil {
						return fmt.Errorf("failed to verify replication status on %s: %w", fixLocation, err)
					}
				}
			}
		}
	}
	return nil
}
