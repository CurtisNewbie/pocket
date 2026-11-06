# 01: Mouse reporting on, note card click opens the note

**What to build:** The app turns mouse reporting on when it starts, and a click on a note card opens that note exactly as Enter does. After the click, focus sits on the card itself, so the record list's keys (j, k, h, l, q, Enter, G, c, n, N, /) keep working without the user clicking anywhere else.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Starting the app enables terminal mouse reporting, and the app runs with mouse reporting unavailable exactly as before.
- [ ] A click on a note card opens that note on the detail page, the same result Enter gives on the focused card.
- [ ] After that click, focus is where Enter leaves it (the detail page's options list), so the detail page keys work; focus never lands on the card's inner table.
- [ ] A press on a card, with no click following, leaves focus on the card itself rather than on its inner table.
- [ ] A test on the simulation-screen seam clicks a note card at coordinates taken from the card's rectangle and asserts the front page and the focused control; it fails before the change.
- [ ] The existing test suite passes unchanged.