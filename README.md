# go-gtids

A Go App To Check For Errant Transactions

## Usage

```Go
./bin/go-gtids -h
Usage: go-gtids -s <source> -t <target> [-source-port <port>] [-target-port <port>] [-fix] [-fix-replica]

./bin/go-gtids -help
Usage of ./bin/go-gtids-macos:
  -fix
        fix the GTID set subset issue by applying to source
  -fix-replica
        fix the GTID set subset issue by applying to replica
  -h    Print help
  -s string
        Source Host
  -source-port string
        Source MySQL port (default "3306")
  -t string
        Target Host
  -target-port string
        Target MySQL port (default "3306")

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
[client]
user=root
password=s3cr3t

[client_primary1]
host=192.168.50.50
port=3306

[client_replica1]
host=192.168.50.50
port=3307
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

## Tools Used for Data validation

- [Data-Diff](https://github.com/datafold/data-diff)

## Working on adding some more functionality to this Go App to make it more useful

- Added code and the logic to check for Errant Transactions.
- Added a -fix flag to fix the errant transaction by applying dummy transactions to the Primary.
- Added a -fix-replica flag to fix the errant transaction by applying dummy transactions directly to the Replica.
- The -fix-replica method safely stops replication, disables binary logging, applies GTIDs, re-enables logging, and restarts replication.

## Screenshots

![Check GTIDs screenshot](screenshots/Check_GTIDs.png)

![Fix GTIDs screenshot](screenshots/Fix_GTIDs.png)

## Added a show me which binlog file by name has the errant transaction(s) by default

![GTIDs binlog screenshot](screenshots/Gtids_binlog.png)

![GTIDs check binlog screenshot](screenshots/Gtids_check_binlog.png)

![GTIDs fix check screenshot](screenshots/Gtids_fix_check.png)

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

## I added a couple of Dockerfiles and a Makefile to help build for testing

```bash
make all

docker run --network mysql57-docker-gtids_db-network go-gtids -s 172.25.0.3 -t 172.25.0.2
[+] Source -> 172.25.0.3 gtid_executed: 874d7e24-2292-11ef-bc48-0242ac190003:1-3966,
c04d4193-2292-11ef-bcdd-0242ac190002:1-4
[+] server_uuid: 874d7e24-2292-11ef-bc48-0242ac190003
[+] Target -> 172.25.0.2 gtid_executed: 874d7e24-2292-11ef-bc48-0242ac190003:1-3966,
c04d4193-2292-11ef-bcdd-0242ac190002:1-4
[+] server_uuid: c04d4193-2292-11ef-bcdd-0242ac190002
[+] No Errant Transactions:



docker run --network mysql57-docker-gtids_db-network pt-slave-restart -h 172.25.0.2

docker run --network mysql57-docker-gtids_db-network pt-slave-restart --master-uuid b33e4e58-21de-11ef-a136-0242ac190003 -h 172.25.0.2

```

## Test using go-gtids and pt-slave-restart

```bash
Replication borked on purpose ...

              Last_SQL_Errno: 1007
               Last_SQL_Error: Error 'Can't create database 'chaos'; database exists' on query. Default database: 'chaos'. Query: 'create database `chaos`'
  Replicate_Ignore_Server_Ids:
             Master_Server_Id: 1
                  Master_UUID: b33e4e58-21de-11ef-a136-0242ac190003
             Master_Info_File: mysql.slave_master_info
                    SQL_Delay: 0
          SQL_Remaining_Delay: NULL
      Slave_SQL_Running_State:
           Master_Retry_Count: 86400
                  Master_Bind:
      Last_IO_Error_Timestamp:
     Last_SQL_Error_Timestamp: 240604 15:48:53
               Master_SSL_Crl:
           Master_SSL_Crlpath:
           Retrieved_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-5967,
ec45394f-21de-11ef-a23d-0242ac190002:5-7
            Executed_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-3976,
ec45394f-21de-11ef-a23d-0242ac190002:1-7
                Auto_Position: 1





Does not register as an Errant transaction, hmm:

docker run --network mysql57-docker-gtids_db-network go-gtids -s 172.25.0.3 -t 172.25.0.2
[+] Source -> 172.25.0.3 gtid_executed: b33e4e58-21de-11ef-a136-0242ac190003:1-5967,
ec45394f-21de-11ef-a23d-0242ac190002:1-7
[+] server_uuid: b33e4e58-21de-11ef-a136-0242ac190003
[+] Target -> 172.25.0.2 gtid_executed: b33e4e58-21de-11ef-a136-0242ac190003:1-3976,
ec45394f-21de-11ef-a23d-0242ac190002:1-7
[+] server_uuid: ec45394f-21de-11ef-a23d-0242ac190002
[+] No Errant Transactions:




mysql --defaults-group-suffix=_replica1 -e "show slave status\G" | awk -v RS='\n ' '
{
    if ($1 ~ /Master_Host|Slave_IO_Running|Slave_SQL_Running|Seconds_Behind_Master|Retrieved_Gtid_Set|Executed_Gtid_Set/) {
        split($0, a, ": ");
        print a[1] ": " substr($0, index($0, a[2]));
    }
}'
                 Master_Host: 172.25.0.3
            Slave_IO_Running: Yes
           Slave_SQL_Running: No
       Seconds_Behind_Master: NULL
Master_SSL_Verify_Server_Cert: No
     Slave_SQL_Running_State:      Slave_SQL_Running_State:
          Retrieved_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-5967,
ec45394f-21de-11ef-a23d-0242ac190002:5-7
           Executed_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-3976,
ec45394f-21de-11ef-a23d-0242ac190002:1-7





Do you have to set --master-uuid? No it still worked without adding that argument.

docker run --network mysql57-docker-gtids_db-network pt-slave-restart --master-uuid b33e4e58-21de-11ef-a136-0242ac190003 -h 172.25.0.2
2024-06-04T16:15:06 h=172.25.0.2 relay-log-bin.000005         658 1007




mysql --defaults-group-suffix=_replica1 -e "show slave status\G" | awk -v RS='\n ' '
{
    if ($1 ~ /Master_Host|Slave_IO_Running|Slave_SQL_Running|Seconds_Behind_Master|Retrieved_Gtid_Set|Executed_Gtid_Set/) {
        split($0, a, ": ");
        print a[1] ": " substr($0, index($0, a[2]));
    }
}'
                 Master_Host: 172.25.0.3
            Slave_IO_Running: Yes
           Slave_SQL_Running: Yes
       Seconds_Behind_Master: 28684
Master_SSL_Verify_Server_Cert: No
     Slave_SQL_Running_State: creating table
          Retrieved_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-5967,
ec45394f-21de-11ef-a23d-0242ac190002:5-7
           Executed_Gtid_Set: b33e4e58-21de-11ef-a136-0242ac190003:1-4856,
ec45394f-21de-11ef-a23d-0242ac190002:1-7
```

## Example with Errant Transactions in a Docker Container

```Go
./bin/go-gtids -s 192.168.50.50 -source-port 3306 -t 192.168.50.50 -target-port 3307
[+] Source -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39
[+] server_uuid: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc
[+] Target -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39,
35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[+] server_uuid: 35e31a1e-1c26-11f0-b708-f6db8cba6bab
[-] Errant Transactions: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[-] Errant Transaction Found in Log Name: binlog.000002



mysql --defaults-group-suffix=_replica1 -e "show slave status\G" | awk -v RS='\n ' '
{
    if ($1 ~ /Master_Host|Slave_IO_Running|Slave_SQL_Running|Seconds_Behind_Master|Retrieved_Gtid_Set|Executed_Gtid_Set/) {
        split($0, a, ": ");
        print a[1] ": " substr($0, index($0, a[2]));
    }
}'
                 Master_Host: 172.20.0.3
            Slave_IO_Running: Yes
           Slave_SQL_Running: Yes
       Seconds_Behind_Master: 0
Master_SSL_Verify_Server_Cert: No
     Slave_SQL_Running_State: Replica has read all relay log; waiting for more updates
          Retrieved_Gtid_Set: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39
           Executed_Gtid_Set: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39,
35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11

./bin/go-gtids -s 192.168.50.50 -source-port 3306 -t 192.168.50.50 -target-port 3307 -fix
[+] Source -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39
[+] server_uuid: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc
[+] Target -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39,
35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[+] server_uuid: 35e31a1e-1c26-11f0-b708-f6db8cba6bab
[-] Errant Transactions: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[-] Errant Transaction Found in Log Name: binlog.000002
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:1
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:2
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:3
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:4
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:5
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:6
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:7
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:8
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:9
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:10
Applied entry: 35e31a1e-1c26-11f0-b708-f6db8cba6bab:11


./bin/go-gtids -s 192.168.50.50 -source-port 3306 -t 192.168.50.50 -target-port 3307
[+] Source -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39,
35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[+] server_uuid: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc
[+] Target -> 192.168.50.50 gtid_executed: 35dacfbe-1c26-11f0-ab3a-3eaa1b6dc9dc:1-39,
35e31a1e-1c26-11f0-b708-f6db8cba6bab:1-11
[+] server_uuid: 35e31a1e-1c26-11f0-b708-f6db8cba6bab
[+] No Errant Transactions: 
```
