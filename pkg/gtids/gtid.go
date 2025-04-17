package gtids

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
)

var (
	singleValueInterval = regexp.MustCompile("^([0-9]+)$")
	multiValueInterval  = regexp.MustCompile("^([0-9]+)[-]([0-9]+)$")
)

type OracleGtidSetEntry struct {
	UUID   string
	Ranges string
}

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

func (oge *OracleGtidSetEntry) String() string {
	return fmt.Sprintf("%s:%s", oge.UUID, oge.Ranges)
}

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

type OracleGtidSet struct {
	GtidEntries [](*OracleGtidSetEntry)
}

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

func (ogs *OracleGtidSet) Explode() (result [](*OracleGtidSetEntry)) {
	for _, entries := range ogs.GtidEntries {
		result = append(result, entries.Explode()...)
	}
	return result
}

func CheckGtidSetSubset(db1 *sql.DB, db2 *sql.DB, source string, target string) {
	var sourceGtidSet string
	var targetGtidSet string
	var errantTransactions string
	var sourceUUID, targetUUID string

	err := db1.QueryRow("SELECT @@server_uuid").Scan(&sourceUUID)
	if err != nil {
		log.Fatal(err)
	}
	err = db2.QueryRow("SELECT @@server_uuid").Scan(&targetUUID)
	if err != nil {
		log.Fatal(err)
	}

	err = db1.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&sourceGtidSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(blue("[+]"), "Source ->", source, "gtid_executed:", sourceGtidSet)
	fmt.Println(blue("[+]"), "server_uuid:", sourceUUID)

	targetHosts := strings.Split(target, ",")
	for _, targetHost := range targetHosts {
		err = db2.QueryRow("SELECT @@GLOBAL.GTID_EXECUTED").Scan(&targetGtidSet)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(yellow("[+]"), "Target ->", targetHost, "gtid_executed:", targetGtidSet)
		fmt.Println(yellow("[+]"), "server_uuid:", targetUUID)

		err = db2.QueryRow("select gtid_subtract('" + targetGtidSet + "','" + sourceGtidSet + "') as errant_transactions").Scan(&errantTransactions)
		if err != nil {
			log.Fatal(err)
		}
		if errantTransactions == "" {
			fmt.Println(green("[+]"), "No Errant Transactions:", errantTransactions)
		} else {
			fmt.Println(red("[-]"), "Errant Transactions:", errantTransactions)
		}
	}
}
