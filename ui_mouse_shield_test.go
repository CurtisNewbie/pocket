package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// shieldProbe is a primitive that counts the clicks which reach its own mouse
// handler, so a test can tell whether the shield passed an event on.
type shieldProbe struct {
	*tview.Box
	clicks int
}

func newShieldProbe() *shieldProbe {
	return &shieldProbe{Box: tview.NewBox()}
}

func (p *shieldProbe) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	return p.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if action == tview.MouseLeftClick && p.InRect(event.Position()) {
			p.clicks++
		}
		return false, nil
	})
}

// TestMouseShieldPassesOnlyEventsInsideTheDialogBox calls the shield's mouse
// handler directly: a click inside the dialog's box is passed on to the
// dialog, while a click outside the box is consumed and reaches nothing.
func TestMouseShieldPassesOnlyEventsInsideTheDialogBox(t *testing.T) {
	dialog := tview.NewFlex()
	dialog.SetRect(10, 5, 20, 8)
	probe := newShieldProbe()
	probe.SetRect(11, 6, 18, 6)
	dialog.AddItem(probe, 0, 1, true)

	modal := tview.NewFlex()
	modal.SetRect(0, 0, 60, 20)
	modal.AddItem(dialog, 0, 1, true)

	shield := newMouseShield(modal, dialog)
	shield.SetRect(0, 0, 60, 20)
	handler := shield.MouseHandler()
	setFocus := func(tview.Primitive) {}

	inside := tcell.NewEventMouse(12, 7, tcell.ButtonPrimary, tcell.ModNone)
	if consumed, _ := handler(tview.MouseLeftClick, inside, setFocus); consumed {
		t.Fatal("a click inside the dialog box was consumed by the shield")
	}
	if probe.clicks != 1 {
		t.Fatalf("a click inside the dialog box reached the dialog %d times, want 1", probe.clicks)
	}

	outside := tcell.NewEventMouse(2, 2, tcell.ButtonPrimary, tcell.ModNone)
	if consumed, _ := handler(tview.MouseLeftClick, outside, setFocus); !consumed {
		t.Fatal("a click outside the dialog box was not consumed")
	}
	if probe.clicks != 1 {
		t.Fatalf("a click outside the dialog box reached the dialog %d times, want 1", probe.clicks)
	}
}

// TestDialogAbsorbsClickOnMargin drives the real application through a
// simulation screen: with a dialog open, a click on the menu behind its box
// runs no entry and leaves the page behind untouched, while a click inside the
// box still reaches the dialog's button.
func TestDialogAbsorbsClickOnMargin(t *testing.T) {
	app := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(120, 50)
	app.SetScreen(screen)

	pocket := &Pocket{Application: app, Pages: tview.NewPages()}
	detailPage := NewDetailPage(pocket)
	pocket.DetailPage = detailPage
	pocket.Pages.AddPage(PageDetail, detailPage, true, true)
	note := Note{Id: 1, Name: "first", Desc: "first desc", Content: "first content"}
	note.Ctime, note.Utime = Now(), Now()
	detailPage.Display(note)
	pocket.SetFocus(detailPage.Options)

	PopMsg(pocket, nil, "hello")
	app.SetRoot(pocket.Pages, true)
	startMouseTestLoop(t, app)

	var marginX, marginY, okayX, okayY int
	waitFor(t, "the dialog and the menu behind it to be laid out", func() bool {
		var laidOut bool
		pocket.QueueUpdateDraw(func() {
			front, item := pocket.Pages.GetFrontPage()
			shield, ok := item.(*mouseShield)
			if !ok || front != PageMsg {
				return
			}
			form, ok := shield.dialog.(*tview.Form)
			if !ok || form.GetButtonCount() < 1 {
				return
			}
			menuX, menuY, menuW, menuH := detailPage.Options.GetRect()
			_, _, dialogW, dialogH := shield.dialog.GetRect()
			okayRectX, okayRectY, okayW, okayH := form.GetButton(0).GetRect()
			if menuW < 3 || menuH < 4 || dialogW < 1 || dialogH < 1 || okayW < 1 || okayH < 1 {
				return
			}
			// The third menu entry is Mask/Unmask, whose effect is visible on
			// the page the dialog covers.
			marginX, marginY = menuX+1, menuY+3
			if shield.dialog.InRect(marginX, marginY) {
				return
			}
			okayX, okayY = okayRectX+okayW/2, okayRectY+okayH/2
			laidOut = true
		})
		return laidOut
	})

	maskedBefore, menuItemBefore := false, 0
	pocket.QueueUpdateDraw(func() {
		maskedBefore = detailPage.Masked
		menuItemBefore = detailPage.Options.GetCurrentItem()
	})

	// A click on the menu entry behind the dialog's box.
	screen.PostEvent(tcell.NewEventMouse(marginX, marginY, tcell.ButtonPrimary, tcell.ModNone))
	screen.PostEvent(tcell.NewEventMouse(marginX, marginY, tcell.ButtonNone, tcell.ModNone))

	// The library treats two clicks closer than its double-click interval as a
	// double click, which a button does not act on. Wait it out so the next
	// click is a single click.
	time.Sleep(tview.DoubleClickInterval + 200*time.Millisecond)

	// Then a click on the dialog's own button. Events are handled in order, so
	// the dialog closing proves the margin click has been handled too, and the
	// button click proves an inside event still reaches the dialog.
	screen.PostEvent(tcell.NewEventMouse(okayX, okayY, tcell.ButtonPrimary, tcell.ModNone))
	screen.PostEvent(tcell.NewEventMouse(okayX, okayY, tcell.ButtonNone, tcell.ModNone))

	waitFor(t, "the dialog's button to close the dialog", func() bool {
		var front string
		pocket.QueueUpdateDraw(func() { front, _ = pocket.Pages.GetFrontPage() })
		return front == PageDetail
	})

	maskedAfter, menuItemAfter := false, 0
	pocket.QueueUpdateDraw(func() {
		maskedAfter = detailPage.Masked
		menuItemAfter = detailPage.Options.GetCurrentItem()
	})
	if maskedAfter != maskedBefore {
		t.Fatal("a click on the dialog's margin ran the Mask/Unmask menu entry behind it")
	}
	if menuItemAfter != menuItemBefore {
		t.Fatalf("a click on the dialog's margin selected menu entry %d, want %d", menuItemAfter, menuItemBefore)
	}
}
