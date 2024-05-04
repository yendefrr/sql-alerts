# SQL New rows monitoring

### About

Fow now it's just check new rows by specified queries and send notification by [ntfy](https://ntfy.sh/)

Support MySQL only

### Configuration

Edit `.config/sqlal/config.json` or run `sqlal config`

All queries must be `SELECT` type and query only one column that should be unique ID

For disable query provide `"disabled": true` parameter

### Installation
```bash
go install github.com/yendefrr/sql-alerts@latest
```

### Usage

After configuration run

```bash
sqlal start --config <path-to-config> 
sqlal start # or edit .config/sqlal/config.json and run it
```

To stop service
```bash
sqlal stop
```

### TODO

- [ ] Other SQL drivers
- [x] Run in background
