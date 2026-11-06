# 06: URI popup and QR PNG export

**What to build:** A selected entry's full otpauth URI can be shown and copied, and the same URI can be exported as a QR PNG to a path the user chooses, with the plaintext warning and no overwrite.

**Blocked by:** 02

**Status:** ready-for-agent

- [ ] The URI key shows the full otpauth URI in a read-only field, and Enter or y copies it.
- [ ] A URI built from an entry with custom parameters round-trips through the parser unchanged.
- [ ] The export key opens a path form that warns the PNG holds the secret in plaintext and defaults to the export directory under the user's home, created when absent.
- [ ] An existing target file produces the collision message and writes nothing; a new path writes a PNG.
- [ ] The written PNG decodes back to the same otpauth URI.
- [ ] Tests cover the URI build round-trip and the encode and decode round-trip.
