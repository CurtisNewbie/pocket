# 04: Add from clipboard QR

**What to build:** Pressing the add key with an image on the clipboard decodes the QR code and feeds the same preview and save path as text. The screenshot never touches disk.

**Blocked by:** 02

**Status:** done

- [x] A QR image on the clipboard adds a TOTP entry through the same preview as text.
- [x] An image wins when the clipboard carries both an image and text.
- [x] An empty clipboard, a clipboard with neither image nor otpauth text, an image with no QR code, and a QR code that is not a TOTP URI each produce their message and save nothing.
- [x] The screenshot is never written to disk.
- [x] QR decoding is tested by encoding a URI to a PNG and decoding it back.
