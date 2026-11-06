package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// mouseDialog is a simulation-screen app showing one dialog.
type mouseDialog struct {
	pocket *Pocket
	loop   *mouseTestLoop
}

// startMouseDialogTest starts a simulation-screen app with the dialog added by
// addDialog and waits for the first draw so the dialog's primitives have their
// rectangles.
func startMouseDialogTest(t *testing.T, addDialog func(*Pocket)) *mouseDialog {
	t.Helper()

	// The tests click more than once in quick succession. Without this, tview
	// derives a double click from the second pair and no dialog handler answers
	// that action.
	doubleClickInterval := tview.DoubleClickInterval
	tview.DoubleClickInterval = 0
	t.Cleanup(func() { tview.DoubleClickInterval = doubleClickInterval })

	app := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	app.SetScreen(screen)
	screen.SetSize(120, 40)

	pocket := &Pocket{Application: app, Pages: tview.NewPages()}
	addDialog(pocket)
	app.SetRoot(pocket.Pages, true)

	loop := startMouseTestLoop(t, app)
	d := &mouseDialog{pocket: pocket, loop: loop}

	onMouseTestLoop(t, app, func() {})
	return d
}

// onMouseTestLoop runs f on the tview event loop and waits for it, so a pending
// redraw has finished before the test reads primitive state.
func onMouseTestLoop(t *testing.T, app *tview.Application, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(f)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tview event loop did not run the update")
	}
}

// sync waits until the event loop has handled every event queued before the
// call, then lets the pending redraw finish.
func (d *mouseDialog) sync(t *testing.T) {
	t.Helper()
	d.loop.flushQueued(t)
	onMouseTestLoop(t, d.pocket.Application, func() {})
}

// findDialogForm returns the form inside the front-most page of the app.
func findDialogForm(p tview.Primitive) *tview.Form {
	switch v := p.(type) {
	case *tview.Form:
		return v
	case *tview.Flex:
		for i := 0; i < v.GetItemCount(); i++ {
			if f := findDialogForm(v.GetItem(i)); f != nil {
				return f
			}
		}
	case *mouseShield:
		return findDialogForm(v.Flex)
	case *tview.Pages:
		_, item := v.GetFrontPage()
		return findDialogForm(item)
	}
	return nil
}

// click queues a left click at x, y and waits for it to be handled.
func (d *mouseDialog) click(t *testing.T, x, y int) {
	t.Helper()
	d.pocket.QueueEvent(tcell.NewEventMouse(x, y, tcell.ButtonPrimary, tcell.ModNone))
	d.pocket.QueueEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
	d.sync(t)
}

// clickField clicks the right-hand side of an input field, inside the field's
// internal text area.
func (d *mouseDialog) clickField(t *testing.T, field *tview.InputField) {
	t.Helper()
	x, y, w, h := field.GetRect()
	d.click(t, x+w-4, y+h/2)
}

// press queues a key and waits for it to be handled.
func (d *mouseDialog) press(t *testing.T, key tcell.Key) {
	t.Helper()
	d.pocket.QueueEvent(tcell.NewEventKey(key, 0, tcell.ModNone))
	d.sync(t)
}

// TestClickingPasswordFieldKeepsFocusAndEnter drives the real application
// through a simulation screen. Clicking the password field must leave the
// focus on the field, so Enter still validates the password, and the message
// dialog's button must still answer a click.
func TestClickingPasswordFieldKeepsFocusAndEnter(t *testing.T) {
	d := startMouseDialogTest(t, func(p *Pocket) { PopPasswordPage(p) })
	pocket, app := d.pocket, d.pocket.Application

	form := findDialogForm(pocket.Pages)
	if form == nil {
		t.Fatal("password dialog has no form")
	}
	field := form.GetFormItem(0).(*tview.InputField)
	if got := app.GetFocus(); got != field {
		t.Fatalf("password field does not start focused, focus = %T", got)
	}

	d.clickField(t, field)
	if got := app.GetFocus(); got != field {
		t.Fatalf("focus after clicking the password field = %T, want the field itself", got)
	}

	// The empty password fails validation, which opens the message dialog.
	d.press(t, tcell.KeyEnter)
	if !pocket.Pages.HasPage(PageMsg) {
		t.Fatal("Enter after clicking the password field did not validate the empty password")
	}

	// The message dialog's Okay button still acts on a click.
	msgForm := findDialogForm(pocket.Pages)
	if msgForm == nil {
		t.Fatal("message dialog has no form")
	}
	okay := msgForm.GetButton(0)
	x, y, w, h := okay.GetRect()
	d.click(t, x+w/2, y+h/2)
	if pocket.Pages.HasPage(PageMsg) {
		t.Fatal("clicking Okay did not close the message dialog")
	}
}

// TestClickingSearchFieldKeepsFocusAndEnter checks a text field whose confirm
// key is handled by the form rather than the field itself.
func TestClickingSearchFieldKeepsFocusAndEnter(t *testing.T) {
	d := startMouseDialogTest(t, func(p *Pocket) {
		p.ListPage = NewListPage(p)
		PopEditSearchPage(p)
	})
	pocket, app := d.pocket, d.pocket.Application

	form := findDialogForm(pocket.Pages)
	if form == nil {
		t.Fatal("search dialog has no form")
	}
	field := form.GetFormItem(0).(*tview.InputField)

	d.clickField(t, field)
	if got := app.GetFocus(); got != field {
		t.Fatalf("focus after clicking the search field = %T, want the field itself", got)
	}

	d.press(t, tcell.KeyEnter)
	if pocket.Pages.HasPage(PageSearch) {
		t.Fatal("Enter after clicking the search field did not close the search dialog")
	}
}

// TestClickingNoteTextAreaKeepsTabNavigation checks that a click on a dialog
// text area does not change where Tab moves the focus.
func TestClickingNoteTextAreaKeepsTabNavigation(t *testing.T) {
	d := startMouseDialogTest(t, func(p *Pocket) { PopCreateNotePage(p, func() {}) })
	app := d.pocket.Application

	form := findDialogForm(d.pocket.Pages)
	if form == nil {
		t.Fatal("create note dialog has no form")
	}
	name := form.GetFormItemByLabel(LabelName).(*tview.TextArea)
	desc := form.GetFormItemByLabel(LabelDesc).(*tview.TextArea)

	x, y, w, h := name.GetRect()
	d.click(t, x+w/2, y+h/2)
	if got := app.GetFocus(); got != name {
		t.Fatalf("focus after clicking the name area = %T, want the name area itself", got)
	}

	d.press(t, tcell.KeyTab)
	if got := app.GetFocus(); got != desc {
		t.Fatalf("after clicking the name area and pressing Tab, focus = %T, want the description area", got)
	}
}
