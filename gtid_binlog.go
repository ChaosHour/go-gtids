// gtid_binlog.go

package main

import (
	"fmt"

	binlog "./cmd/go-binlog"
)

// find where the binlogs are stored on the target Host(s) database
func findBinlogPath(target string) string {
	var binlogPath string
	rows, err := db2.Query("show variables like 'log_bin_basename';")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value string
		err = rows.Scan(&name, &value)
		if err != nil {
			fmt.Println(err)
		}
		binlogPath = value
	}
	return binlogPath
}

// find the binlog with the errant transaction and print to stdout the binlog name
func findErrantBinlog(target string, gtidSet *binlog.GtidSet, errantGTID string) {
	binlogPath := findBinlogPath(target)
	binlogReader, err := binlog.NewBinlogReader(binlogPath, 4)
	if err != nil {
		fmt.Println(err)
	}
	defer binlogReader.Close()
	for {
		event, err := binlogReader.GetEvent()
		if err != nil {
			fmt.Println(err)
			break
		}
		if event.Header.EventType == binlog.WRITE_ROWS_EVENTv1 || event.Header.EventType == binlog.WRITE_ROWS_EVENTv2 ||
			event.Header.EventType == binlog.UPDATE_ROWS_EVENTv1 || event.Header.EventType == binlog.UPDATE_ROWS_EVENTv2 ||
			event.Header.EventType == binlog.DELETE_ROWS_EVENTv1 || event.Header.EventType == binlog.DELETE_ROWS_EVENTv2 {
			tableID := event.TableID
			gtid, err := binlogReader.GetGTID()
			if err != nil {
				fmt.Println(err)
				break
			}
			if gtidSet.Contains(gtid) && gtid.String() == errantGTID {
				fmt.Printf("Errant transaction found in binlog %s\n", binlogReader.GetFilename())
				break
			}
		}
	}
}
