# 01: Authenticator page and upgradable storage

**What to build:** The Authenticator page exists and opens from the notebook's Options list, showing its empty state. An existing database gains the TOTP entry table on the next launch, and TOTP entries can be created, listed, updated, and deleted with the secret encrypted at rest.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] A database created before this change opens the Authenticator page without a missing-table error, and a fresh database also gets the table.
- [x] The page is reachable from the Options list, shows the empty state with the add hint, and Esc returns focus to the Options list.
- [x] Creating a TOTP entry stores its secret encrypted; reading it back returns the decrypted secret, and the stored row never holds the plaintext secret.
- [x] Listing returns entries newest first, updating the remark persists, and deleting removes the row.
- [x] A failed write surfaces an error and leaves no partial row.
- [x] Tests cover the storage seam against a temporary database and cover creating the table on a database that lacks it.
