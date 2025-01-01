# SQL New rows monitoring

<img width="1420" alt="sqlal" src="https://github.com/yendefrr/sql-alerts/assets/91500932/a614715d-74d5-411d-b152-7da2cad7b950">

### About

Just check new rows by specified queries and send notification to [ntfy](https://ntfy.sh/)

**Now support MySQL only**

### Installation
```bash
curl https://raw.githubusercontent.com/yendefrr/sql-alerts/refs/heads/main/install.sh | sh
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
- [x] DB configuration
- [ ] Validation
- [x] Run in background
