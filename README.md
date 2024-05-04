# SQL New rows monitoring

### About

Fow now it's just check new rows by specified queries and send notification by [ntfy](https://ntfy.sh/)

Support MySQL only

In active development but almost ready for daily usage

### Installation
```bash
go install github.com/yendefrr/sql-alerts@latest
```

### Configuration

```bash
sqlal config # or edit `.config/sqlal/config.json
```

All queries must be `SELECT` type and query only one column that should be unique `ID`

For disable query provide `"disabled": true` parameter

### Usage

After configuration run

```bash
sqlal start
```
or
```bash
sqlal start --config <path-to-config> 
```

To stop service
```bash
sqlal stop
```

### TODO

- [ ] Other SQL drivers
- [ ] DB configuration
- [ ] Validation
- [x] Run in background
