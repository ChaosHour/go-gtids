# go-gtids
A Go App To Check For Errant Transactions

## Usage
```
./go-gtids -s < source host> -t <target host>


./go-gtids -help
  -h	Print help
  -s string
    	Source Host
  -t string
    	Target Host

```


## Example
```Testing MySQL 8 & GTID's

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

(data-sync) klarsen@Mac-Book-Pro2 data-sync % go run . -s 10.8.0.152 -t 10.8.0.153
[+] Source gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Target gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[-] Errant Transactions: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:2


Ran on the primary to resolve; other method is to reset master on the replica:
root@primary:~# mysql -e "SET GTID_NEXT='1d1fff5a-c9bc-11ed-9c19-02a36d996b94:2';BEGIN; COMMIT;SET GTID_NEXT='AUTOMATIC'"



Then checked again:

(data-sync) klarsen@Mac-Book-Pro2 data-sync % go run . -s 10.8.0.152 -t 10.8.0.153
[+] Source gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Target gtid_executed: 1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1-2,
c4709bcc-c9bb-11ed-8d19-02a36d996b94:1-33
[+] Errant Transactions: 



Validated data:
(data-sync) klarsen@Mac-Book-Pro2 data-sync % data-diff \
mysql://dba:xxxxx@10.8.0.152:3306/book million_words \
mysql://dba:xxxxx@10.8.0.153:3306/book million_words
- 212036

The Errant Transaction was resolved. You will still need to sync your data.
```