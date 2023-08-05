// gtid.go
package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"database/sql"
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
		return nil, fmt.Errorf("Cannot parse OracleGtidSetEntry from %s", gtidRangeString)
	}
	if tokens[0] == "" {
		return nil, fmt.Errorf("Unexpected UUID: %s", tokens[0])
	}
	if tokens[1] == "" {
		return nil, fmt.Errorf("Unexpected GTID range: %s", tokens[1])
	}
	gtidRange := &OracleGtidSetEntry{UUID: tokens[0], Ranges: tokens[1]}
	return gtidRange, nil
}

// String returns a user-friendly string representation of this entry
func (this *OracleGtidSetEntry) String() string {
	return fmt.Sprintf("%s:%s", this.UUID, this.Ranges)
}

// String returns a user-friendly string representation of this entry
func (this *OracleGtidSetEntry) Explode() (result [](*OracleGtidSetEntry)) {
	intervals := strings.Split(this.Ranges, ":")
	for _, interval := range intervals {
		if submatch := multiValueInterval.FindStringSubmatch(interval); submatch != nil {
			intervalStart, _ := strconv.Atoi(submatch[1])
			intervalEnd, _ := strconv.Atoi(submatch[2])
			for i := intervalStart; i <= intervalEnd; i++ {
				result = append(result, &OracleGtidSetEntry{UUID: this.UUID, Ranges: fmt.Sprintf("%d", i)})
			}
		} else if submatch := singleValueInterval.FindStringSubmatch(interval); submatch != nil {
			result = append(result, &OracleGtidSetEntry{UUID: this.UUID, Ranges: interval})
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
func (this *OracleGtidSet) RemoveUUID(uuid string) (removed bool) {
	filteredEntries := [](*OracleGtidSetEntry){}
	for _, entry := range this.GtidEntries {
		if entry.UUID == uuid {
			removed = true
		} else {
			filteredEntries = append(filteredEntries, entry)
		}
	}
	if removed {
		this.GtidEntries = filteredEntries
	}
	return removed
}

// RetainUUID retains only entries that belong to given UUID.
func (this *OracleGtidSet) RetainUUID(uuid string) (anythingRemoved bool) {
	return this.RetainUUIDs([]string{uuid})
}

// RetainUUIDs retains only entries that belong to given UUIDs.
func (this *OracleGtidSet) RetainUUIDs(uuids []string) (anythingRemoved bool) {
	retainUUIDs := map[string]bool{}
	for _, uuid := range uuids {
		retainUUIDs[uuid] = true
	}
	filteredEntries := [](*OracleGtidSetEntry){}
	for _, entry := range this.GtidEntries {
		if retainUUIDs[entry.UUID] {
			filteredEntries = append(filteredEntries, entry)
		} else {
			anythingRemoved = true
		}
	}
	if anythingRemoved {
		this.GtidEntries = filteredEntries
	}
	return anythingRemoved
}

// SharedUUIDs returns UUIDs (range-less) that are shared between the two sets
func (this *OracleGtidSet) SharedUUIDs(other *OracleGtidSet) (shared []string) {
	thisUUIDs := map[string]bool{}
	for _, entry := range this.GtidEntries {
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
func (this *OracleGtidSet) Explode() (result [](*OracleGtidSetEntry)) {
	for _, entries := range this.GtidEntries {
		result = append(result, entries.Explode()...)
	}
	return result
}

func (this *OracleGtidSet) String() string {
	tokens := []string{}
	for _, entry := range this.GtidEntries {
		tokens = append(tokens, entry.String())
	}
	return strings.Join(tokens, ",")
}

func (this *OracleGtidSet) IsEmpty() bool {
	return len(this.GtidEntries) == 0
}

// My code starts here - Kurt Larsen - 2023-31-07
// function to check the GTID set subset issue and fix it if the -fix flag is set
func checkGtidSetSubset(db1 *sql.DB, db2 *sql.DB, source string, target string) {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string
	var sourceUUID, targetUUID string

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
			fmt.Println(green("[+]"), "Errant Transactions:", errantTransactions)
		} else {
			fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)

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
		if *fix {
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
