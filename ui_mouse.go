package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// MouseHandler gives a note card its own pointer behaviour: a left button down
// focuses the card itself rather than the table inside it, and a left click
// opens the note, which is what Enter on the focused card does. The focus then
// follows the page switch exactly as it does for Enter. Every other pointer
// action on the card is consumed too, so a wheel or a move over a card cannot
// reach a page behind it.
func (l *ListItemPrimitive) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return l.Table.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if !l.InRect(event.Position()) {
			return false, nil
		}
		switch action {
		case tview.MouseLeftDown:
			setFocus(l)
			return true, nil
		case tview.MouseLeftClick:
			if l.onOpen != nil {
				l.onOpen()
			}
			return true, nil
		}
		return true, nil
	})
}

// openNote shows nt on the detail page, the same path Enter on a focused note
// card takes.
func openNote(pocket *Pocket, nt Note) {
	pocket.DetailPage.Display(nt)
	pocket.Pages.SwitchToPage(PageDetail)
}
