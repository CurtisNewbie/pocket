package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// boxedPrimitive is a tview primitive that exposes the widget box every tview
// widget carries, so the shield can ask whether a point is inside it.
type boxedPrimitive interface {
	tview.Primitive
	InRect(x, y int) bool
}

// mouseShield is the outermost primitive of a dialog page. It consumes every
// pointer event outside the dialog's box, so nothing on the page behind the
// dialog reacts to a stray click. Events inside the box reach the dialog as
// usual.
type mouseShield struct {
	*tview.Flex
	dialog boxedPrimitive
}

// newMouseShield wraps modal, the dialog's full-screen layout, in a shield
// around dialog, the dialog's own box.
func newMouseShield(modal *tview.Flex, dialog boxedPrimitive) *mouseShield {
	return &mouseShield{Flex: modal, dialog: dialog}
}

// MouseHandler hands events inside the dialog's box to the dialog and reports
// events outside it as consumed.
func (s *mouseShield) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	inner := s.Flex.MouseHandler()
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if !s.dialog.InRect(event.Position()) {
			return true, nil
		}
		return inner(action, event, setFocus)
	}
}
