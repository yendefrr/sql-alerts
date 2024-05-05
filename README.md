# SQL New rows monitoring

<img width="1266" alt="Arc-CRM@2x" src="https://github.com/yendefrr/sql-alerts/assets/91500932/785f5502-c2ed-453c-a55f-506beb220875">

### About

Just check new rows by specified queries and send notification to [ntfy](https://ntfy.sh/)

**Now support MySQL only**

### Installation
```bash
brew tap yendefrr/tap
brew install sql-alerts
```
For linux use `install.sh` or download from latest release

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
