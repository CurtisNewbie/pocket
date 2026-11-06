# 04: Dialog fields ignore clicks, dialog buttons keep theirs

**What to build:** Clicking a text field inside a dialog does nothing, so the dialog keeps its own keyboard handling. After the click, Tab and Shift-Tab still move between fields and buttons, Enter still saves where Enter saved before, and the dialog's other form keys still work. Buttons inside the same dialog stay clickable. The dialogs covered are the password prompt, the search form, the create and edit note forms, and the authenticator create, edit, URI, and export forms.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Clicking an input field, password field, or text area inside a dialog leaves the dialog's keyboard handling intact (Tab, Enter, and the dialog's own keys behave as before the click).
- [x] Clicking a dialog button still presses it.
- [x] A simulation-screen test clicks a field in a dialog, then presses the dialog's save or confirm key, and asserts the effect. For the password and search fields the test fails before the change. The text area case is a guard, because the library already kept the form's focus there.