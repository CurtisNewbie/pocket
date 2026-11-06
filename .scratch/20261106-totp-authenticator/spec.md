Status: ready-for-agent

# TOTP Authenticator

## Problem Statement

The user signs in to websites that require a one-time code from Google Authenticator. The codes only exist on the phone, while the user works in a terminal, so every login means picking up the phone. The phone is also a dead end. Google Authenticator has no export, so the seeds cannot be moved to another app or device later. Pocket already holds the user's passwords under one master password, but has no place for an MFA secret and cannot produce a code.

## Solution

An Authenticator page inside pocket. The user captures a QR screenshot to the macOS clipboard, presses one key on the page, and the app decodes the QR, previews the entry, and stores the secret encrypted in the notebook's database. The page lists every entry's live code with the seconds left in the period. Enter copies the code to the clipboard. An entry can later be exported as an otpauth URI or a QR PNG, so the user can move it to any other authenticator app or device. The feature does not change the note pages, and existing databases gain it on the next launch.

## User Stories

1. As the notebook's owner, I want to capture a QR screenshot from my clipboard when adding an MFA entry, so that I never type a secret by hand.
2. As the notebook's owner, I want to paste an otpauth URI as text when I cannot screenshot, so that import still works from a text-only source.
3. As the notebook's owner, I want to paste a bare Base32 secret and be prompted for issuer and account, so that entries from a non-QR source are still usable.
4. As the notebook's owner, I want to see the decoded issuer, account, and parameters before saving, so that a wrong screenshot does not become a bad entry.
5. As the notebook's owner, I want to attach a remark when the same site has several accounts, so that I can tell the rows apart later.
6. As the notebook's owner, I want two entries with the same issuer and account to coexist, so that I can keep separate secrets for the same login.
7. As the notebook's owner, I want the Authenticator list ordered newest first, so that a just-added entry appears where I expect.
8. As the notebook's owner, I want each row to show issuer, account, and remark, so that I can find the entry I need at a glance.
9. As the notebook's owner, I want each row to show the current code, so that I can read it without opening the entry.
10. As the notebook's owner, I want each row to show the seconds left in the period, so that I know whether a code will still be accepted when I type it.
11. As the notebook's owner, I want Enter on a row to copy its code to the clipboard, so that I can paste it into the website.
12. As the notebook's owner, I want the list to refresh every second without moving my focus or scroll position, so that navigation stays stable while codes roll over.
13. As the notebook's owner, I want to edit an entry's account and remark after creation, so that I can fix a mislabelled login or add context I skipped at import time.
14. As the notebook's owner, I want issuer, secret, and parameters to be immutable after creation, so that I cannot silently break a working entry.
15. As the notebook's owner, I want deleting an entry to require confirmation, so that a stray keypress does not lose a secret that cannot be re-imported without the phone.
16. As the notebook's owner, I want an entry's URI popup to show its full otpauth URI, so that I can copy it into another authenticator app.
17. As the notebook's owner, I want exporting a QR PNG to ask for a path and default to a known directory, so that I can scan the code with another device.
18. As the notebook's owner, I want a warning that an exported PNG contains the secret in plaintext, so that I decide where the file goes.
19. As the notebook's owner, I want an existing file to stop the export and ask for another path, so that I never overwrite an earlier export by accident.
20. As the notebook's owner, I want entries with SHA-256, SHA-512, 8 digits, or a non-default period to generate correct codes, so that non-standard sites still work.
21. As the notebook's owner, I want HOTP entries rejected with a clear message, so that I know why an import failed instead of getting a wrong code.
22. As the notebook's owner, I want the Authenticator page to appear in my existing database after an upgrade, so that I do not have to recreate the notebook.
23. As the notebook's owner, I want the codes to match the phone for the same secret, so that the website accepts them.
24. As the notebook's owner, I want a clear message when the clipboard is empty, holds no QR image, or holds text that is not an otpauth URI, so that I can fix the input and retry.
25. As the notebook's owner, I want the secret never written to the list, a log, or the database in plaintext, so that the notebook file alone does not expose my second factor.

## Implementation Decisions

### Page and access

- A new Authenticator page is reached from the existing left Options list on the notebook's main page. It uses a page name no existing page uses.
- The page is only reachable after the master password is accepted, like every other page.

### Storage

- A new sqlite table holds TOTP entries. One row is one TOTP entry.
- The secret is encrypted with the notebook's existing AES-GCM helper before it is written. Issuer, account, remark, digits, period, algorithm, and timestamps stay plaintext, matching how a Note stores its name and description.
- Entries are read ordered by creation time descending, with the row id as the tie-breaker.
- No uniqueness rule is enforced. Duplicate issuer and account pairs are legal, and no import ever updates or overwrites an existing row.

### Schema upgrade

- Table and index creation becomes idempotent and runs on every launch after the password check, so an existing database gains the new table without a manual migration.
- First-run initialisation of the configuration record stays separate and still runs only on a brand-new database. The new table's creation does not depend on it.

### Import

- One add action reads the macOS clipboard and decides by content. Image content is decoded as a QR code, and an image wins when the clipboard carries both. Text content is parsed as an otpauth URI or as a bare Base32 secret requiring issuer and account prompts.
- The image path asks the OS for a PNG rendering of the clipboard image, decodes it in memory, and never writes the screenshot to disk.
- The text path accepts only `otpauth://totp/...`. A bare Base32 secret is accepted with prompted issuer and account. Any other text is rejected.
- The parsed entry is previewed before anything is written. Saving happens only on confirmation. Cancelling discards the secret.
- Values absent from the URI take the TOTP defaults of 6 digits, a 30-second period, and SHA-1. Present values are stored and honoured.
- HOTP URIs are rejected; only TOTP is supported.

### List and code generation

- Codes are generated from the stored secret and parameters against the system clock, using standard TOTP.
- A row whose secret cannot be decrypted shows a decrypt failure in place of its code, and copying it reports the failure.
- The empty state names the add action so a first-time user is not stuck.

### Entry actions

- Account and remark are editable through a small form; the change is persisted immediately.
- Issuer, secret, digits, period, and algorithm are not editable. Replacing a secret means deleting the entry and importing again.
- Delete asks for confirmation, then removes the row.

### Export

- The URI popup shows the full otpauth URI built from the stored fields, including the decrypted secret, and copies it to the clipboard on Enter or y.
- QR PNG export asks for a path, defaulting to a directory under the user's home at `pocket-export/`, with the filename built from issuer and account, where `/`, a space, and `:` become `-`, leading and trailing dashes are trimmed, and an empty name becomes `totp`. The directory is created if absent.
- The export form states that the PNG contains the secret in plaintext before it writes anything.
- If the target file already exists, the export fails with a message and the user supplies another path.
- The PNG encodes the same otpauth URI that the URI popup shows.

### Interaction

The page keeps two focus contexts, the Options list on the left and the entry table on the right. The table returns focus to the page's Options list on Esc, h, Left, or q, and the Options list returns to the notebook list on b or Esc.

| Context | Key | Action |
|---|---|---|
| Options | j, k, Down, Up | Move the option |
| Options | Enter | Run the selected option |
| Options | a | Add from clipboard |
| Options | l | Move focus to the entry table |
| Options | u | Show the selected entry's otpauth URI |
| Options | x | Export the selected entry as a QR PNG |
| Options | b | Return to the notebook list |
| Options | q | Exit the app, after the existing exit confirmation |
| Options | Esc | Return to the notebook list |
| Table | j, k, Down, Up | Move the selection |
| Table | G | Jump to the last entry |
| Table | Enter, l, Right | Copy the selected entry's current code |
| Table | e | Edit the selected entry's account and remark |
| Table | d | Delete the selected entry, after confirmation |
| Table | u | Show the selected entry's otpauth URI |
| Table | x | Export the selected entry as a QR PNG |
| Table | a | Add from clipboard |
| Table | h, Left, Esc, q | Return focus to the Options list |
| Form | Tab, Shift-Tab | Move between fields and buttons |
| Form | Enter | Confirm |
| Form | Esc | Cancel |
| URI popup | Enter, y | Copy the URI |
| URI popup | Esc | Close |

The Options list offers six items, add from clipboard, select an entry, copy the URI, export a QR PNG, back to notes, and exit. The two export items act on the table's selected row, so they work without moving focus to the table. With no entries, both report "nothing to export, add an entry first".

A row shows issuer, account, remark, the current code, and the seconds left. Each identity column shows its own value, so a single empty side needs no fallback. When both are empty the issuer column shows `otpauth`. Long issuer and account values are truncated by rune count with an ellipsis. Six-digit codes display as two groups of three and eight-digit codes as two groups of four, while a copied code carries no grouping. The header shows the entry count and the refresh cadence. The empty state reads "no entries yet" and "press a to add from clipboard". The table scrolls natively and never paginates.

The entry list is built when the page opens and rebuilt only after an add, a delete, or an edit. A refresh tick runs once per second, reads the clock afresh instead of counting down, writes only the code and seconds cells whose text changed, and wraps the writes in one queued redraw. Because only cell text changes, the selection and the scroll offset survive. A tick is skipped while the page is not visible or a modal is open, and the next tick refreshes the page when it becomes visible again. The status line starts with the hint `enter copy  e edit  d delete  u uri  x qr`; a status message returns to that hint on the next keypress, not on a timer.

Messages appear in one of three places.

| Place | Message |
|---|---|
| Modal | clipboard is empty |
| Modal | clipboard isn't a qr image or otpauth:// uri |
| Modal | no qr code in clipboard image |
| Modal | qr code isn't a totp uri |
| Modal | hotp isn't supported, totp only |
| Modal | secret isn't valid base32 |
| Modal | digits must be 6 or 8 |
| Modal | period must be 1 to 3600 seconds |
| Form | this png contains your secret in plaintext |
| Form | file already exists: <name>, choose another name |
| Form | can't write to that folder |
| Form | couldn't encode the qr code |
| Status line | copied code for <issuer>: <account> |
| Status line | saved <issuer>: <account> |
| Status line | entry updated |
| Status line | deleted <issuer>: <account> |
| Status line | exported to <path> (plaintext) |
| Status line | reading clipboard... |
| Status line | copied uri |
| Status line | couldn't copy code |
| Status line | couldn't copy uri |
| Status line | secret can't be decrypted |

The preview form shows issuer, account, digits, period, algorithm, and an editable remark. Issuer and account are editable only when the import came from a bare Base32 secret, and all fields except remark are read-only otherwise. The edit form offers the account and remark; the account may be empty. The delete confirmation names the entry and includes its remark when one is set. The URI popup shows the full URI in a read-only field and copies it from there.

### Dependencies

- TOTP generation uses a maintained library rather than a hand-rolled HMAC implementation.
- QR decoding and QR encoding each use a maintained library; no parser is hand-written.
- Clipboard read and write go through the macOS pasteboard, so the feature is macOS-only, consistent with the app's existing dependence on Unix tools.

## Contract

### Data

**Table `pocket_totp`**

| Column | Type | Null | Default | Meaning |
|---|---|---|---|---|
| id | INTEGER | no | autoincrement | Row identity, also the sort tie-breaker |
| issuer | TEXT | no | none | Issuer from the otpauth URI |
| account | TEXT | no | none | Account from the otpauth URI |
| remark | TEXT | no | `''` | Free-text disambiguator, editable |
| secret | TEXT | no | none | Hex AES-GCM ciphertext of the Base32 secret |
| digits | INTEGER | no | 6 | 6 or 8 digits |
| period | INTEGER | no | 30 | Seconds per code, from 1 to 3600 |
| algorithm | TEXT | no | `SHA1` | `SHA1`, `SHA256`, or `SHA512` |
| ctime | DATETIME | no | none | Creation time |
| utime | DATETIME | no | none | Last account or remark change |

Index intent. Reading the list scans the table ordered by `ctime DESC, id DESC`. No additional index is introduced for a table of this size, and no uniqueness constraint exists on issuer or account.

### Interfaces

**Clipboard capture** returns one of the following.

| Field | Type | Meaning |
|---|---|---|
| kind | enum `image` / `text` | Which clipboard content was found |
| text | string, empty for image | Clipboard text when kind is text |
| png | bytes, empty for text | Clipboard image rendered as PNG when kind is image |

The failure codes are `clipboard-empty`, `clipboard-unsupported-content`, and `clipboard-read-failed`.

**QR decode** takes PNG bytes and returns the embedded text. Its failure codes are `qr-not-found` and `qr-decode-failed`.

**Import flow** reports `qr-not-totp-uri` when a decoded QR does not carry a TOTP URI. When the QR carries a TOTP-shaped URI with a defect, the parse failure survives as `hotp-unsupported`, `secret-invalid`, or `param-invalid`.

**Code generation** takes a stored entry and a time and returns the code. Its failure code is `secret-undecryptable`.

**otpauth parse** takes text and returns either a parsed TOTP entry or a bare Base32 secret.

| Field | Type | Null | Default | Meaning |
|---|---|---|---|---|
| scheme | enum `totp` / `hotp` | no | none | Non-TOTP schemes are rejected |
| issuer | string | yes | `''` | Empty when the URI omits it |
| account | string | yes | `''` | From the label part |
| secret | string | no | none | Base32, validated before acceptance |
| digits | integer | no | 6 | Must be 6 or 8 |
| period | integer | no | 30 | Must be between 1 and 3600 |
| algorithm | string | no | `SHA1` | `SHA1`, `SHA256`, `SHA512` |

The failure codes are `uri-invalid`, `hotp-unsupported`, `secret-invalid`, and `param-invalid`.

**otpauth build** takes a stored entry and the decrypted secret and returns a URI that round-trips through the parser. Its failure code is `secret-undecryptable`.

The storage functions (create, list, update the account and remark, delete) follow the notebook's existing storage-contract indirection so tests can substitute them. The failure codes are `db-write-failed`, `db-read-failed`, and `db-delete-failed`.

**Clipboard write** copies text. Its failure code is `clipboard-write-failed`.

**QR PNG encode** takes text and a destination path and writes a PNG. Its failure codes are `file-exists`, `file-write-failed`, and `encode-failed`.

### Config

| Key | Type | Domain | Default | Owner | Effect |
|---|---|---|---|---|---|
| export directory | string | any absolute path | `~/pocket-export/` | user, per export | Where the QR PNG defaults to; created when absent |

No new environment variables or config records.

### Invariants

- A secret is never stored, logged, listed, or displayed in plaintext; it leaves the process only through an explicit URI copy or a QR PNG export.
- A code is never persisted.
- Stored digits, period, and algorithm always drive generation; defaults apply only when the URI omits them.
- An import never modifies an existing entry.
- The list order is creation time descending.
- A copied code is the bare digits; only the display groups them.
- Export refuses to overwrite a file.
- The secret is never stored, logged, or listed in plaintext. It is decrypted in memory when entries are loaded, and for URI build and QR export.
- HOTP is not representable in the table and is rejected at import.

### States and failure

The list page is either empty or loaded. Empty shows the add hint. Loaded refreshes cell text every second.

The import flow runs through idle, reading the clipboard, decoding or parsing, the preview, and saving. The user can cancel at any point before saving. A failure returns to idle with a message and never persists a partial entry.

A confirmed delete removes the entry. A cancelled one leaves it unchanged.

The export flow leads from the path prompt to a written file, or to `file-exists` or `file-write-failed` with the prompt left open for a new path.

No operation retries automatically. A failed clipboard read or QR decode requires the user's next add action. A failed write surfaces the message and leaves the database unchanged.

### Boundaries

- **macOS pasteboard.** The app reads images as PNG and writes text, never images. There is no idempotency key and no ordering guarantee. When the pasteboard is unavailable, the add action fails with a message.
- **otpauth URI.** This is the interchange format for import and export, and the app is both reader and writer. Round-trip fidelity is an invariant.
- **SQLite database.** The app is the only writer. The table lives in the same file as notes, so backup and encryption expectations match the notes'.

## Testing Decisions

- Tests assert external behaviour. A known secret and timestamp produce the RFC 6238 vector code. A URI round-trips. An image produces the decoded entry. Storage CRUD on a temporary database returns what was written.
- The seams follow the repo's existing storage-contract pattern.
  1. Code generation is a pure function of secret, digits, period, algorithm, and time, tested against RFC 6238 vectors including SHA-256 and SHA-512.
  2. otpauth parse and build are pure functions, tested on standard URIs, defaults, a missing issuer, 8 digits, a non-default period, bad Base32, and HOTP rejection.
  3. The clipboard reader is a substitutable function returning the capture result, so the import flow can be tested without touching the OS pasteboard.
  4. QR decode is a pure function over PNG bytes, tested by encoding an otpauth URI to a QR PNG in the test and decoding it back.
  5. Storage goes through the existing storage-contract indirection, exercised against a temporary sqlite database. The prior art is the current single crypto test.
- The tview page itself is not unit-tested, consistent with the repo's current test surface; its logic lives in the seams above.

## Out of Scope

- HOTP, Steam Guard, Yubikey, and any non-TOTP second factor.
- Windows or Linux clipboard support.
- Rendering a QR code inside the terminal.
- Batch export, import, or migration from other authenticator apps.
- Editing a secret, issuer, or TOTP parameter after creation.
- Search, filtering, folders, or tags on the Authenticator page.
- Re-locking the app, auto-lock, or a second password prompt for the page.
- Changing the master password or its key derivation.
- Clock-skew detection or network time.
- Syncing entries between devices.

## Further Notes

- The notebook's AES key is the master password bytes padded to 32 bytes, with no key derivation. TOTP secrets inherit that strength. Strengthening key derivation is a separate change and is deliberately not part of this feature.
- Exporting a URI or a QR PNG is a deliberate plaintext disclosure; the UI warns before the PNG write, and the URI copy is an explicit user action.
- A deleted entry can only be recovered by scanning the original QR again; the delete confirmation is the only guard.
