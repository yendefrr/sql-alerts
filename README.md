# SQL New rows monitoring

### About

Fow now it's just check new rows by specified queries and send notification by [ntfy](https://ntfy.sh/)

Support MySQL only

### Configuration

Rename config.example.json to config.json and fill your values

All queries must be `SELECT` type and query only one column that should be unique ID

For disable query provide `"disabled": true` parameter

### Usage

After configuration run

```
go install github.com/yendefrr/sql-alerts
sqlal --config <path-to-config> # or edit .config/sqlal/config.json
```

### TODO

- [ ] Other SQL drivers
- [ ] Run in background
