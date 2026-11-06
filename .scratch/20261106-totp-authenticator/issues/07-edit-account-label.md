# 07: Edit an entry's account label

**What to build:** The selected entry's account label and remark are editable after creation. Saving updates the row and the otpauth URI that export builds; the code, issuer, secret, digits, period, and algorithm do not change.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] The edit key opens a form with the current account and remark prefilled.
- [x] Saving a new account updates the row and the exported otpauth URI label, and the generated code is unchanged.
- [x] Cancelling changes nothing.
- [x] The storage update writes the account, remark, and update time, and leaves the secret and parameters untouched; a test covers the storage seam.
- [x] The status line reports "entry updated".
