# Code review rules: pocket

Mistakes this repo keeps making, each one a tell that shows up in a diff. Written by reviews;
empty until one finds a mistake here.

1. **Neutralized widget bypassed at a construction site.** A widget kind whose clicks must be
   swallowed, or whose mouse handler must be replaced, is safe only when every construction site
   goes through the wrapper. A fresh bare constructor silently reopens the hole the change closed.
   The tell is a new `tview.NewX()` call in a diff that also adds wrappers for the same kind of
   widget. Fix by routing every site through the shared constructor. Found at the detail content
   view and the authenticator empty view in the mouse-support review.