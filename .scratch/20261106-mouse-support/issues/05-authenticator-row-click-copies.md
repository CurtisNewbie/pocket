# 05: Clicking an authenticator row copies its code

**What to build:** On the authenticator page, a click on a table row selects that row and copies the entry's current code, matching what Enter does on the focused row. The status line reports the copy the same way. Copying goes through the substitutable clipboard writer, so no test touches the system pasteboard.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A click on an authenticator row selects that row.
- [ ] The same click copies that entry's current code through the substitutable clipboard writer and reports it on the status line.
- [ ] A click inside the table but outside a row, and a click while the page shows no entries, copies nothing and reports nothing.
- [ ] A simulation-screen test stubs the clipboard writer, clicks a row at coordinates taken from the table, and asserts the copied text and the selected row; it fails before the change.
- [ ] Keyboard Enter on a focused row still copies as before.