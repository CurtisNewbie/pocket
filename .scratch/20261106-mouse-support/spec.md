# Mouse support

**Status:** ready-for-agent

## Problem Statement

Pocket answers only to the keyboard. A user who reaches for the mouse gets nothing back. Clicking a note card does nothing, clicking a menu entry does nothing, and clicking a button in a dialog does nothing. The user has to look up which key opens a note, which key returns to the list, and which key confirms a form, then find those keys on the keyboard, even when their hand is already on the mouse.

The obvious fix, switching mouse reporting on, makes the app worse than it was. Clicking a note card moves focus onto a primitive that no longer receives keys, so every shortcut on that page dies until the user clicks somewhere else. A click on the margin of a dialog reaches the page behind it and runs whichever menu entry sits under the pointer, including Delete and Exit. Clicking the title bar or the version line steals focus the same way. The mouse has to be wired in, not just switched on.

## Solution

Mouse reporting is on whenever Pocket runs. Clicking a control does what that control's key does. Clicking a note card opens the note, clicking an authenticator row copies its code, clicking a button in a dialog presses it, and clicking a menu entry runs it. Clicking something inert changes nothing, including the focus, so the keyboard keeps working exactly where it was. A dialog absorbs every pointer event outside its own box, so the page behind it never reacts to a stray click.

## User Stories

1. As a notebook user, I want to click a note card to open that note, so that I can browse with the mouse.
2. As a notebook user, I want the keyboard shortcuts on the detail page to keep working after I click a note, so that I do not have to click my way back to a working state.
3. As a notebook user, I want to click an entry in the left-hand options list to run it, so that the menu works with the mouse.
4. As a notebook user, I want to click an option that opens a dialog, so that I can work entirely with the mouse.
5. As a notebook user, I want to click Confirm, Save, Copy, or Cancel in a dialog, so that I do not have to remember which key confirms.
6. As a notebook user, I want clicks on the margin of a dialog to do nothing, so that a stray click cannot delete a note sitting behind it.
7. As a notebook user, I want to click a row in the authenticator table to select it and copy its code, so that copying a code needs one click.
8. As a notebook user, I want clicking the title bar, the search parameters, or the version line to leave focus where it was, so that the page shortcuts keep working.
9. As a notebook user, I want a dialog's text fields to ignore clicks, so that tabbing through a form and pressing Enter to save still works after I touch the mouse.
10. As a keyboard user, I want every shortcut to behave exactly as before, so that adding the mouse costs me nothing.
11. As a keyboard user, I want a terminal without mouse reporting to run Pocket as it always did, so that the feature cannot break my setup.
12. As a notebook user, I want the mouse never to move focus somewhere that ignores keys, so that I never end up in a page that looks selected but does nothing.
13. As a notebook user, I want the password prompt to accept a click on its button, so that unlocking works by mouse as well as keyboard.

## Implementation Decisions

- Mouse reporting is enabled when the application starts. There is no flag, environment variable, or settings entry, and no runtime toggle.
- Each widget keeps the library's native click behaviour, so a menu entry runs on a single click and a button fires on a single click. No confirm step is added to a menu entry.
- A note card gets its own mouse handling. A press on it moves focus to the card itself, never to the table inside it, and a click opens the note exactly as Enter does. The detail page then takes focus the way it does after Enter, so focus never sits on the record list while that list is no longer visible.
- A dialog's outermost primitive absorbs pointer events. Any event whose position falls outside the dialog's box is consumed there, so no page behind the dialog receives it.
- Inert areas drop the left-button-down action. A static text view, a non-selectable table, a title bar, a status line, and the version line all belong here. Dropping only the button-down is enough because focus changes on button-down, while a click action is derived later from the release.
- A dialog's input fields drop the left-button-down action. Buttons inside the same dialog keep theirs, so the dialog stays usable by mouse.
- A click on an authenticator table row selects the row and copies that entry's code, matching Enter. The copy goes through the substitutable clipboard writer the import and export flows already use.
- Wheel events are not part of this feature. A widget that scrolled with the wheel before still does, and no widget gains a new wheel behaviour. A note card's whole region absorbs pointer events it does not otherwise act on, so nothing behind the card scrolls. Content longer than the screen stays a keyboard job.
- Pointer movement is not handled. There is no hover highlight.
- Clicking and dragging text in the terminal becomes the terminal's mouse drag while Pocket runs; users hold the terminal's selection modifier to select text.

## Testing Decisions

A good test asserts what a user can observe after interacting: which page is in front, which control holds focus, which row or card is selected, what the status line says, and which text reached the clipboard. It never reaches into a widget's internals or asserts which handler ran.

Two seams carry the feature.

The first seam drives the real application through a simulation screen. A test builds the app the way the existing event-loop test does, feeds the mouse event pair for a press and a release through the event queue, waits for the event loop to consume it, then asserts the observable state above. Click coordinates come from the target control's rectangle rather than from hard-coded cells, so a layout change does not break the test. The clipboard writer is replaced with a stub for the authenticator test.

The second seam tests the dialog shield on its own. The shield is a primitive whose mouse handler is called directly with an event inside the dialog box and one outside it. The inside event is passed on and the outside event is reported as consumed. This needs no application and no screen.

Prior art sits in the existing UI and authenticator tests. The UI test starts a simulation screen, runs the application in a goroutine, and asserts that the event loop keeps moving. The authenticator tests substitute the clipboard writer and the schema initializer.

Every ticket adds its test first. Where the library already behaves correctly, the test pins that behaviour as a guard instead of failing first, and says so in the ticket.

## Out of Scope

- Wheel scrolling, hover highlighting, drag selection, double-click behaviour, and the middle and right mouse buttons.
- Clicking a text field inside a dialog to place the caret.
- Restyling the app, dimming the page behind a dialog, or hiding it.
- Changing how text is selected or copied in the terminal.
- Any behaviour on platforms whose terminals do not report mouse events.

## Further Notes

Terminal mouse reporting takes over drag selection. Users on macOS hold Option while dragging to select text, and on other terminals the usual modifier is Shift. Pocket's own copy paths shell out to the pasteboard and are unaffected.

## Contracts

### Data

N/A, no stored data changes.

### Interfaces

The mouse contract, by what sits under the pointer. Left button down means the press, left click means a press and release on the same cell.

| Pointer target | Left button down | Left click |
|---|---|---|
| Note card | Focus the card | Open the note |
| Menu entry | Focus the menu | Run the entry |
| Dialog button | Focus the button | Press the button |
| Dialog input field | Ignored | Ignored |
| Dialog box, outside any control | Focus the dialog's last focused control | Ignored |
| Dialog margin, outside the box | Consumed, reaches nothing | Consumed, reaches nothing |
| Static text view | Ignored | Ignored |
| Non-selectable table | Ignored | Ignored |
| Authenticator table row | Focus the table | Select the row and copy its code |

Every other pointer action is left to the target's default, except on a note card, whose region absorbs anything it does not act on.

### Config

None. The feature adds no key, default, or owner.

## Invariants

- After any mouse interaction, every key that the focused control handled before that interaction still works. No click may leave focus on a primitive that ignores keys.
- A pointer event outside the front dialog's box never reaches a page behind that dialog.
- A click on an inert area never changes the app's focus.
- A click on an interactive control has the same effect as the key that control answers to.
- Keyboard behaviour is identical whether or not the terminal reports mouse events.

## States and Failure

The state a click can change is the focused control and the front page. Focus moves to the control clicked, or stays put. The front page changes only where the clicked control's keyboard equivalent changes it, so a note card click moves to the detail page and a menu entry that opens a dialog adds the dialog.

There are no failure codes and no retries. The library derives a click only when a press and its release land on the same cell, so a press that drags away produces no click at all. A terminal that does not report mouse events delivers nothing; the app behaves as it did before this feature and never signals an error.

## Boundaries

The terminal emulator owns mouse reporting. Pocket switches reporting on at start and reads whatever arrives. The emulator decides whether a drag becomes a selection, and some emulators send no events under their own selection modifier. Neither side retries, orders, or deduplicates anything beyond the press and release pairing the library already does.