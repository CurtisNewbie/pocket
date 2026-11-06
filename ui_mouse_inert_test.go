package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestClickOnInertAreaKeepsFocus drives the real application through a
// simulation screen. A click on the list page's title bar must not move the
// focus, whether nothing is focused or a note card is, so the list page's keys
// keep working afterwards.
func TestClickOnInertAreaKeepsFocus(t *testing.T) {
	// The test clicks the same inert cell twice in quick succession. tview
	// would otherwise derive a double click from the second pair, and no handler
	// on the title bar answers that action.
	doubleClickInterval := tview.DoubleClickInterval
	tview.DoubleClickInterval = 0
	t.Cleanup(func() { tview.DoubleClickInterval = doubleClickInterval })

	pocket := &Pocket{Application: tview.NewApplication(), Pages: tview.NewPages()}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(120, 40)
	pocket.SetScreen(screen)

	list := NewListPage(pocket)
	pocket.ListPage = list
	pocket.Pages.AddPage(PageList, list, true, true)
	list.ListView.AddNote(Note{Id: 1, Name: "first"})
	list.ListView.AddNote(Note{Id: 2, Name: "second"})

	pocket.SetRoot(pocket.Pages, true)

	loop := startMouseTestLoop(t, pocket.Application)

	// The coordinates come from the title bar's rectangle, so a layout change
	// does not break the test.
	bar := list.ListView.bar
	var x, y, w, h int
	waitFor(t, "the list title bar to be laid out", func() bool {
		pocket.QueueUpdateDraw(func() { x, y, w, h = bar.GetRect() })
		return w > 0 && h > 0
	})
	click := func() {
		loop.clickScreen(t, screen, x+w/2, y+h/2)
	}

	// With nothing focused the click must leave the focus empty.
	pocket.QueueUpdateDraw(func() { pocket.SetFocus(nil) })
	click()
	if got := pocket.GetFocus(); got != nil {
		t.Fatalf("focus after clicking the title bar is %T, want nothing focused", got)
	}

	// With a note card focused the click must leave the focus on that card.
	card := list.ListView.content.GetItem(0).(*ListItemPrimitive)
	pocket.QueueUpdateDraw(func() { pocket.SetFocus(card) })
	click()
	if got := pocket.GetFocus(); got != card {
		t.Fatalf("focus after clicking the title bar moved to %T, want the focused note card", got)
	}

	// With the card still focused, the list page's keys must still be handled.
	screen.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))
	loop.flushScreen(t, screen)
	waitFor(t, "'q' to return the focus to the options list", func() bool {
		return pocket.GetFocus() == list.Options
	})
}

// TestClickOnNoteBodyKeepsDetailFocus drives the real application through a
// simulation screen. The note body is inert, so clicking it leaves the focus on
// the detail page's options list and the page's keys keep working.
func TestClickOnNoteBodyKeepsDetailFocus(t *testing.T) {
	pocket := &Pocket{Application: tview.NewApplication(), Pages: tview.NewPages()}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(120, 40)
	pocket.SetScreen(screen)

	list := NewListPage(pocket)
	pocket.ListPage = list
	pocket.Pages.AddPage(PageList, list, true, false)

	detail := NewDetailPage(pocket)
	pocket.DetailPage = detail
	pocket.Pages.AddPage(PageDetail, detail, true, true)
	note := Note{Id: 1, Name: "first", Desc: "first desc", Content: "first content"}
	note.Ctime, note.Utime = Now(), Now()
	detail.Display(note)
	pocket.SetRoot(pocket.Pages, true)
	pocket.SetFocus(detail.Options)

	// Returning to the list refetches the notes, which must not reach the
	// database.
	fetched := make(chan struct{}, 1)
	fetchNotes := StFetchNotes
	StFetchNotes = func(page int, limit int, name string) (int, []Note, error) {
		select {
		case fetched <- struct{}{}:
		default:
		}
		return 0, nil, nil
	}
	t.Cleanup(func() { StFetchNotes = fetchNotes })

	loop := startMouseTestLoop(t, pocket.Application)

	var x, y, w, h int
	waitFor(t, "the note body to be laid out", func() bool {
		pocket.QueueUpdateDraw(func() { x, y, w, h = detail.content.GetRect() })
		return w > 0 && h > 0
	})
	loop.clickScreen(t, screen, x+w/2, y+h/2)
	if got := pocket.GetFocus(); got != detail.Options {
		t.Fatalf("focus after clicking the note body is %T, want the detail page's options list", got)
	}

	// The detail page's keys still work after the click.
	screen.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone))
	loop.flushScreen(t, screen)
	waitFor(t, "'h' to return to the note list", func() bool {
		var page string
		pocket.QueueUpdateDraw(func() { page, _ = pocket.Pages.GetFrontPage() })
		return page == PageList
	})
	select {
	case <-fetched:
	case <-time.After(3 * time.Second):
		t.Fatal("returning to the list did not refetch the notes")
	}
}
