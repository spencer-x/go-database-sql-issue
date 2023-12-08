# go-database-sql-issue

## Ways to reproduce the issue in linux machine
1. add packet latency to simulate the host to db cross region latency
```bash
sudo tc qdisc add dev lo root netem delay 60ms
```
2. run this code using
```bash
go run main.go
```
3. once things stabilize, check connection number from postgresql side with psql
```sql
SELECT sum(numbackends) FROM pg_stat_database;
```
4. inject packet loss between host and db to simulate network blip
```bash
sudo iptables -A OUTPUT -p tcp --dport 5432 -m statistic --mode random --probability 0.5 -j DROP
```
5. remove packet loss event in iptables supposing added rule index was 1
```bash
sudo iptables -D OUTPUT 1
```
6. observe that things never recover until restarted
7. check connection number from db side with psql again