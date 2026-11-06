# Database schema: where it lives

The schema has exactly one home: `data.go`.

- `createSchema()` runs the DDL, and `InitSchema()` calls it on every launch: `pocket_config`
  (`CREATE TABLE` + `key_idx` index), `pocket_note` as an `fts4` virtual table, and `pocket_totp`
  for TOTP entries.
- `Note` in `data.go` is the gorm model for `pocket_note`.

There is no `schema.sql`, no `migrations/` directory, and no separate snapshot. Never stand a
second record beside this one: a table change edits `data.go` directly, and a ticket that alters
tables is not done until `data.go` matches.