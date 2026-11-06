# 02: Add from text with preview

**What to build:** Pressing the add key reads the clipboard's text. An otpauth URI or a bare Base32 secret becomes a previewed TOTP entry, and saving puts it at the top of the list. Nothing is written before confirmation.

**Blocked by:** 01

**Status:** done

- [x] A standard otpauth URI on the clipboard opens a preview showing issuer, account, digits, period, algorithm, and remark.
- [x] A URI carrying SHA-256, 8 digits, and a non-default period is saved with those values.
- [x] A bare Base32 secret prompts for issuer and account, then saves.
- [x] Invalid Base32, an HOTP URI, and text that is neither produce their message and save nothing.
- [x] Cancelling the preview saves nothing.
- [x] The saved entry appears first in the list with issuer, account, and remark, and an empty issuer or account falls back as the spec describes.
- [x] Tests cover otpauth parsing, its defaults, and every rejection at the parse seam.
