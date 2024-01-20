# SQL New rows monitoring

### About

Fow now it's just check new rows by specified queries and send notification by [ntfy](https://ntfy.sh/)

Support MySQL only

### Configuration

Rename config.example.json to config.json and fill your values

All queries must be `SELECT` type and query only one column that should be unique ID

### Usage

After configuration run

```
go build
./sql-alerts
```

### TODO

- [ ] Other SQL drivers
- [ ] Run in background
