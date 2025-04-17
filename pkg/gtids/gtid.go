// gtid.go
package gtids

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

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

// My code starts here - Kurt Larsen - 2023-31-07
// function to check the GTID set subset issue and fix it if the -fix flag is set
func CheckGtidSetSubset(db1 *sql.DB, db2 *sql.DB, source string, target string, fix bool) {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string
	var sourceUUID, targetUUID string
	//var binaryLogFile string

	// connect to the source and target Host(s) database and run the query to get the SELECT @@server_uuid value
	err := db1.QueryRow("SELECT @@server_uuid").Scan(&sourceUUID)
	if err != nil {
		log.Fatal(err)
	}
	err = db2.QueryRow("SELECT @@server_uuid").Scan(&targetUUID)
	if err != nil {
		log.Fatal(err)
	}

	// connect to the source database and run the query to get the gtid_executed value
	err = db1.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&sourceGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	// add the source @@server_uuid to the output as well
	fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet)
	fmt.Println(blue("[+]"), "server_uuid:", sourceUUID)
	//fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet, "server_uuid:", sourceGtidSet)

	// connect to the target Host(s) database and run the query to get the gtid_executed value
	targetHosts := strings.Split(target, ",")
	for _, targetHost := range targetHosts {
		err = db2.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&targetGtidSet)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(yellow("[+]"), "Target ->", targetHost, "gtid_executed:", targetGtidSet)
		// add the target Host(s) @@server_uuid to the output as well and it to separate line just the server_uuid
		fmt.Println(yellow("[+]"), "server_uuid:", targetUUID)
		// fmt.Println(yellow("[+]"), "Target ->", targetHost, "gtid_executed:", targetGtidSet, "server_uuid:", targetGtidSet)

		// using a select "select gtid_subtract get the source gtid_executed value and the target Host(s) gtid_executed value from the function checkGtidSetSubset()
		err = db2.QueryRow("select gtid_subtract('" + targetGtidSet + "','" + sourceGtidSet + "') as errant_transactions").Scan(&errantTransactions)
		if err != nil {
			log.Fatal(err)
		}
		if errantTransactions == "" {
			fmt.Println(green("[+]"), "No Errant Transactions:", errantTransactions)
		} else {
			fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)

		}
		if errantTransactions != "" {
			// New code to get the current binary log file name and executed GTID set - Kurt Larsen 2023-08-20
			// use SHOW MASTER STATUS to get the name of the most recent binary log file and the executed GTID set
			var logName string
			var pos int
			var Binlog_Do_DB string
			var Binlog_Ignore_DB string
			var executedGtidSet string
			err = db2.QueryRow("SHOW MASTER STATUS").Scan(&logName, &pos, &Binlog_Do_DB, &Binlog_Ignore_DB, &executedGtidSet)
			if err != nil {
				log.Fatal(err)
			}
			err = db2.Ping()
			if err != nil {
				log.Fatal(err)
			}
			//fmt.Println(red("[-]"), "Errant Transaction Found in Log Name:", logName)
			//fmt.Println("Executed GTID Set:", executedGtidSet)
			//fmt.Println("Errant Transaction:", errantTransactions)

			/*
				// use regexp to search for the errant transactions in the executed GTID set
				re := regexp.MustCompile(errantTransactions)
				if re.MatchString(executedGtidSet) {
					//fmt.Println("Executed GTID Set:", executedGtidSet)
					fmt.Println(yellow("[-]"), "Errant Transaction Found in Log Name:", logName)
					//} else {
					//fmt.Println(green("[+]"), "No Errant Transaction Found in Log Name:", logName)
				}
			*/
			re := regexp.MustCompile("Errant Transactions: ([0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}:[0-9]+)")
			matches := re.FindStringSubmatch(executedGtidSet)
			if len(matches) > 0 {
				errantTransactions = matches[1]
			}
			fmt.Println(yellow("[-]"), "Errant Transaction Found in Log Name:", logName)

		}

		/*
			// function to explode the errant transactions and print them out, then apply them to the source database to replicate the errant transactions down to the target Host(s)
			gtidSet := errantTransactions
			oracleGtidSet, err := NewOracleGtidSet(gtidSet)
			if err != nil {
				log.Fatal(err)
			}

			gtidEntries := oracleGtidSet.Explode()
			for _, entry := range gtidEntries {
				fmt.Printf("%s:%s\n", entry.UUID, entry.Ranges)
			}
		*/

		// function to explode the errant transactions and print them out, then apply them to the source database to replicate the errant transactions down to the target Host(s)
		gtidSet := errantTransactions
		oracleGtidSet, err := NewOracleGtidSet(gtidSet)
		if err != nil {
			log.Fatal(err)
		}

		// Store UUID and Ranges in a slice
		var entries []string
		gtidEntries := oracleGtidSet.Explode()
		for _, entry := range gtidEntries {
			entries = append(entries, fmt.Sprintf("%s:%s", entry.UUID, entry.Ranges))
		}
		if fix {
			// Apply the UUID and Ranges to the database
			for _, entry := range entries {
				//fmt.Printf("Applying entry: %s\n", entry)
				// Apply the errant transaction
				_, err := db1.Exec("SET GTID_NEXT='" + entry + "'")
				if err != nil {
					log.Fatal(err)
				}

				// execute a BEGIN statement
				_, err = db1.Exec("BEGIN")
				if err != nil {
					log.Fatal(err)
				}

				// execute a COMMIT statement
				_, err = db1.Exec("COMMIT")
				if err != nil {
					log.Fatal(err)
				}

				// set GTID_NEXT back to AUTOMATIC
				_, err = db1.Exec("SET GTID_NEXT='AUTOMATIC'")
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Applied entry: %s\n", entry)
			}

		}
	}
}
