# Pocket

A terminal notebook whose contents are encrypted under one master password. This glossary covers the notebook itself and its TOTP authenticator.

## Language

**Note**:
A titled, encrypted record in the notebook, the app's original unit of storage.
_Avoid_: Item, entry

**TOTP entry**:
One account's TOTP secret together with its generation parameters and an optional remark, shown on the Authenticator page.
_Avoid_: Token, account, authenticator item

**Secret**:
The Base32 seed a TOTP entry's codes are derived from; stored encrypted and never displayed in the list.
_Avoid_: Key, seed, password

**Code**:
The short-lived numeric one-time password a TOTP entry produces. Never stored.
_Avoid_: Token, OTP, password

**Issuer**:
The service a TOTP entry belongs to, as named in its otpauth URI.
_Avoid_: Provider, site, service

**Account**:
The identity at an issuer that a TOTP entry authenticates.
_Avoid_: User, username, login

**Remark**:
Free-text disambiguator on a TOTP entry, editable after creation.
_Avoid_: Note, label, comment

**otpauth URI**:
The standard `otpauth://totp/...` text form of a TOTP entry, used for both import and export.
_Avoid_: QR code, link