package gtids

import (
	"log"
	"os"
	"strings"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var (
	Db1 *sql.DB
	Db2 *sql.DB
	Err error
)

// ReadMyCnf reads the ~/.my.cnf file to get the database credentials
func ReadMyCnf() {
	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "user") {
			os.Setenv("MYSQL_USER", strings.TrimSpace(line[5:]))
		}
		if strings.HasPrefix(line, "password") {
			os.Setenv("MYSQL_PASSWORD", strings.TrimSpace(line[9:]))
		}
	}
}

// ConnectToDatabase connects to the source and target Host(s) database at the same time.
func ConnectToDatabase(source string, target string) {
	Db1, Err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+source+":3306)/mysql")
	if Err != nil {
		log.Fatal(Err)
	}
	Err = Db1.Ping()
	if Err != nil {
		log.Fatal(Err)
	}
	Db2, Err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+target+":3306)/mysql")
	if Err != nil {
		log.Fatal(Err)
	}
	Err = Db2.Ping()
	if Err != nil {
		log.Fatal(Err)
	}
}

// ConnectToDatabaseWithPort connects to the source and target Host(s) database with custom ports.
func ConnectToDatabaseWithPort(source string, sourcePort string, target string, targetPort string) {
	Db1, Err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+source+":"+sourcePort+")/mysql")
	if Err != nil {
		log.Fatal(Err)
	}
	Err = Db1.Ping()
	if Err != nil {
		log.Fatal(Err)
	}
	Db2, Err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+target+":"+targetPort+")/mysql")
	if Err != nil {
		log.Fatal(Err)
	}
	Err = Db2.Ping()
	if Err != nil {
		log.Fatal(Err)
	}
}
