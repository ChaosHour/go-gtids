# go-gtids
A Go App To Check For Errant Transactions

## Usage
```Go
./go-gtids -h
Usage: go-gtids -s <source> -t <target> [-fix]


./go-gtids -help
 -fix
        fix the GTID set subset issue
  -h    Print help
  -s string
        Source Host
  -t string
        Target Host

```



## Credentials
```Go
The credentials are stored in the ~/.my.cnf file in the users home directory and are read by the app.

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() {
	file, err := ioutil.ReadFile(os.Getenv("HOME") + "/.my.cnf")
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


 cat ~/.my.cnf
[primary]
user=dba_util
password=xxxx
host=10.8.0.152
[replica]
user=dba_util
password=xxxx
host=10.8.0.153
[etlreplica]
user=dba_util
password=xxxx
host=10.8.0.154

```


## Example
```Go
Testing MySQL 8 & GTID's

mysql --version
mysql  Ver 8.0.32-24 for Linux on x86_64 (Percona Server (GPL), Release '24', Revision 'e5c6e9d2')


root@replica:~# mysql -e "select * from  book.million_words LIMIT 1"
+--------+---------+
| id     | word    |
+--------+---------+
| 212036 | a'irahS |
+--------+---------+


root@replica:~# mysql -e "delete from book.million_words where id = 212036 limit 1"



Checked for Errant Transactions:  -s = <primary> -t = <replica>

(data-sync) klarsen@Mac-Book-Pro2 data-sync % ./go-gtids -s 10.5.0.152 -t 10.5.0.153
[+] Source gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Target gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[-] Errant Transactions: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:2


Ran on the primary to resolve; other method is to reset master on the replica:
root@primary:~# mysql -e "SET GTID_NEXT='1d1fff5a-c9bc-11ed-9c19-02a36d996b94:2';BEGIN; COMMIT;SET GTID_NEXT='AUTOMATIC'"



Then checked again:

(data-sync) klarsen@Mac-Book-Pro2 data-sync % ./go-gtids -s 10.5.0.152 -t 10.5.0.153
[+] Source gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Target gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Errant Transactions: 



Validated data:
(data-sync) klarsen@Mac-Book-Pro2 data-sync % data-diff \
mysql://dba:xxxxx@10.5.0.152:3306/book million_words \
mysql://dba:xxxxx@10.5.0.153:3306/book million_words
- 212036

The Errant Transaction was resolved. You will still need to sync your data.
```

## Tools Used for Data validation:
- [Data-Diff](https://github.com/datafold/data-diff)




## Working on adding some more functionality to this Go App to make it more useful.
- Added code and the logic to check for Errant Transactions.
- Added a -fix flag to fix the errant transaction.
- Currently this only applies a dummy transaction to the Primary and that is replicated to the Replica's to fix the errant transaction.




## Screenshots

<img src="screenshots/Check_GTIDs.png" width="619" height="173" />











<img src="screenshots/Fix_GTIDs.png" width="634" height="275" />




## Added a show me which binlog file by name has the errant transaction(s) by default.

<img src="screenshots/Gtids_binlog.png" width="575" height="142" />








<img src="screenshots/Gtids_check_binlog.png" width="673" height="305" />








<img src="screenshots/Gtids_fix_check.png" width="614" height="343" />



```Go
To build:

go build -o go-gtids

FreeBSD:
env GOOS=freebsd GOARCH=amd64 go build .

On Mac:
env GOOS=darwin GOARCH=amd64 go build .

Linux:
env GOOS=linux GOARCH=amd64 go build .
```