# 03: Inert areas ignore clicks

**What to build:** Clicking an area that carries no interaction leaves focus where it was. The areas are the title bars on the list, detail, and authenticator pages, the status line, the version line, the search parameters table, and the read-only info table. After such a click, the page's shortcuts still work, so q, e, d, m, h, and the search and paging keys behave as before.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Clicking any inert area leaves the app's focus on the control that held it.
- [x] After such a click, the page's keys are still handled (returning to the options list, editing, searching, and paging all still work).
- [x] Interactive areas on the same pages keep their click behaviour.
- [x] A simulation-screen test clicks an inert area with nothing focused, then with a card focused, and asserts focus in both cases; it fails before the change.