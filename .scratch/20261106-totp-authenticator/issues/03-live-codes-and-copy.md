# 03: Live codes and copy

**What to build:** Every row shows the current code and the seconds left in its period, refreshed every second without moving the selection or the scroll. Enter copies the bare digits to the clipboard and reports the result.

**Blocked by:** 02

**Status:** done

- [x] Each row shows its current code and seconds left, and both update every second.
- [x] The selection and the scroll offset stay put across a code rollover.
- [x] Enter copies the bare digits to the clipboard and reports the copy in the status line; a clipboard write failure reports the failure instead.
- [x] RFC 6238 vectors pass for SHA-1, SHA-256, SHA-512, 6 digits, 8 digits, and a non-default period.
- [x] A row whose secret cannot be decrypted shows a failure in place of its code, and copying it reports that failure.
