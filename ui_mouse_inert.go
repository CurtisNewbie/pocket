package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// inertTextView is a text view that carries no interaction.
type inertTextView struct {
	*tview.TextView
}

// inertTable is a table whose cells carry no interaction.
type inertTable struct {
	*tview.Table
}

// ignoreMouseLeftDown returns a mouse handler that consumes only the
// left-button-down action on box, so the widget never takes the focus and the
// event does not reach a widget behind it. The wheel and the click action go to
// next untouched.
func ignoreMouseLeftDown(box *tview.Box, next func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive)) func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		if action == tview.MouseLeftDown && box.InRect(event.Position()) {
			return true, nil
		}
		return next(action, event, setFocus)
	}
}

// MouseHandler consumes the left-button-down action.
func (v *inertTextView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return ignoreMouseLeftDown(v.TextView.Box, v.TextView.MouseHandler())
}

// MouseHandler consumes the left-button-down action.
func (t *inertTable) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return ignoreMouseLeftDown(t.Table.Box, t.Table.MouseHandler())
}

// newInertTextView returns a text view that ignores the left-button-down
// action, so a click on it leaves the focus where it was.
func newInertTextView() *inertTextView {
	return &inertTextView{TextView: tview.NewTextView()}
}

// newInertTable returns a table that ignores the left-button-down action, so a
// click on it leaves the focus where it was.
func newInertTable() *inertTable {
	return &inertTable{Table: tview.NewTable()}
}
