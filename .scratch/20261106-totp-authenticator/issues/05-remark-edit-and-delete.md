# 05: Remark edit and delete

**What to build:** A selected entry's remark can be changed after creation, and the entry can be deleted after a confirmation that names it.

**Blocked by:** 02

**Status:** done

- [x] The remark key opens a form; saving updates the row and the list immediately, and cancelling changes nothing.
- [x] The delete key asks for confirmation naming the entry, including its remark when one is set.
- [x] Confirming deletes the row and cancelling keeps it.
- [x] After a delete, focus stays on the neighbouring entry and the header count updates.
- [x] Tests cover the storage update and delete paths.