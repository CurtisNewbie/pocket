package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// dialogFieldMouseCapture is the part of a form item this file needs: the
// ability to intercept its pointer events before its own handler sees them.
type dialogFieldMouseCapture interface {
	SetMouseCapture(func(tview.MouseAction, *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse)) *tview.Box
}

// ignoreDialogFieldClicks drops the left-button-down action on a dialog form's
// input fields and text areas. Focus stays on the control the dialog already
// had, so its Tab, Enter, and form keys keep working after a click. Buttons are
// not touched and stay clickable.
func ignoreDialogFieldClicks(form *tview.Form) {
	for i := 0; i < form.GetFormItemCount(); i++ {
		item := form.GetFormItem(i)
		switch item.(type) {
		case *tview.InputField, *tview.TextArea:
			dropDialogFieldLeftDown(item.(dialogFieldMouseCapture))
		}
	}
}

// dropDialogFieldLeftDown turns the left-button-down action off for p. The
// click action is derived from the release, so the press alone is what moves
// the focus.
func dropDialogFieldLeftDown(p dialogFieldMouseCapture) {
	p.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDown {
			return action, nil
		}
		return action, event
	})
}
