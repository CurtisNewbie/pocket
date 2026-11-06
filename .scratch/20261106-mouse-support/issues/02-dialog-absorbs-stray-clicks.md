# 02: A dialog absorbs stray clicks

**What to build:** Every dialog in the app has a shield around its box. A pointer event outside the dialog's box is consumed there and reaches nothing behind it, so a click on a dialog's margin cannot select a note card, scroll a table, or run a menu entry sitting under the pointer. Events inside the dialog's box pass through to the dialog as usual.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] A click on a dialog's margin leaves the front page, the page behind, and every selection unchanged; a menu entry under the pointer does not run.
- [x] A click inside the dialog's box still reaches the dialog, so a click on its Confirm, Cancel, Copy, or Save button still acts.
- [x] The shield applies to every dialog the app opens, the password prompt included.
- [x] A unit test calls the shield's mouse handler with one event inside the dialog box and one outside, and asserts that only the inside event is passed on; it fails before the change.
- [x] A simulation-screen test opens a dialog, clicks its margin, and asserts that the page behind kept its state.