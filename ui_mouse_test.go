package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// TestClickNoteCardOpensNote drives the real application through a simulation
// screen: a press on a note card focuses the card itself, and a click on it
// opens that note exactly as Enter does, down to where the focus lands.
func TestClickNoteCardOpensNote(t *testing.T) {
	pocket := NewApp()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(120, 50)
	pocket.SetScreen(screen)

	notes := []Note{
		{Id: 1, Name: "first", Desc: "first desc", Content: "first content"},
		{Id: 2, Name: "second", Desc: "second desc", Content: "second content"},
		{Id: 3, Name: "third", Desc: "third desc", Content: "third content"},
	}
	for i := range notes {
		notes[i].Ctime = Now()
		notes[i].Utime = Now()
		pocket.ListPage.AddNote(notes[i])
	}
	cardAt := func(i int) *ListItemPrimitive {
		return pocket.ListPage.ListView.content.GetItem(i).(*ListItemPrimitive)
	}
	pocket.ToPage(PageList)
	pocket.SetFocus(cardAt(0))

	// Returning to the list refetches the notes, which must not reach the
	// database.
	fetched := make(chan struct{}, 1)
	fetchNotes := StFetchNotes
	StFetchNotes = func(page int, limit int, name string) (int, []Note, error) {
		select {
		case fetched <- struct{}{}:
		default:
		}
		return len(notes), notes, nil
	}
	t.Cleanup(func() { StFetchNotes = fetchNotes })

	loop := startMouseTestLoop(t, pocket.Application)

	clicked := cardAt(1)

	// The coordinates come from the card's rectangle, so a layout change does
	// not break the test.
	var x, y, w, h int
	waitFor(t, "the note cards to be laid out", func() bool {
		pocket.QueueUpdateDraw(func() { x, y, w, h = clicked.GetRect() })
		return w > 0 && h > 0
	})
	cx, cy := x+w/2, y+h/2

	// A press without a release leaves the focus on the card itself, not on
	// the table inside it.
	screen.PostEvent(tcell.NewEventMouse(cx, cy, tcell.ButtonPrimary, tcell.ModNone))
	loop.flushScreen(t, screen)
	if got := pocket.GetFocus(); got != clicked {
		t.Fatalf("focus after the press is %T, want the clicked card", got)
	}

	screen.PostEvent(tcell.NewEventMouse(cx, cy, tcell.ButtonNone, tcell.ModNone))
	loop.flushScreen(t, screen)

	waitFor(t, "the clicked note to open on the detail page", func() bool {
		var page, name string
		pocket.QueueUpdateDraw(func() {
			page, _ = pocket.Pages.GetFrontPage()
			name = pocket.DetailPage.Item.Name
		})
		return page == PageDetail && name == "second"
	})

	// The page switch delegates the focus to the detail page's options list,
	// which is where Enter on the focused card leaves it too.
	if got := pocket.GetFocus(); got != pocket.DetailPage.Options {
		t.Fatalf("focus after the click is %T, want the detail page's options list", got)
	}

	// A key only the detail page handles still works after the click.
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
