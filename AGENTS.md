# pocket

Terminal-based password-protected notebook. Go, tview + gorm/sqlite3.

## Agent skills

### Issue tracker

Local markdown: specs and issues live under `.scratch/<YYYYMMDD>-<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five roles, label string equal to the role name. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` at root, ADRs in `docs/adr/`. See `docs/agents/domain.md`.

### Database schema

No standalone schema file: DDL lives in `data.go`'s `InitSchema()`. See `docs/agents/schema.md`.

### Code review

Read before writing and when reviewing code. See `docs/agents/code-review.md`.