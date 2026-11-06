package main

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

var authenticatorClickEntries = []TotpEntry{
	{
		Issuer:    "GitHub",
		Account:   "alice",
		Secret:    "JBSWY3DPEHPK3PXP",
		Digits:    6,
		Period:    3600,
		Algorithm: "SHA1",
	},
	{
		Issuer:    "GitLab",
		Account:   "bob",
		Secret:    "KRSXG5CTMVRXEZLU",
		Digits:    6,
		Period:    3600,
		Algorithm: "SHA1",
	},
}

// TestClickAuthenticatorRowSelectsAndCopiesCode drives the real application
// through a simulation screen: a click on the second table row selects that
// row, writes its code through the substitutable clipboard writer, and reports
// the copy on the status line, the same way Enter does on the focused row.
func TestClickAuthenticatorRowSelectsAndCopiesCode(t *testing.T) {
	copied := stubClipboardWriter(t)
	app := startAuthenticatorClickApp(t, authenticatorClickEntries)

	x, y := authenticatorRowPoint(t, app, 1)
	app.loop.clickScreen(t, app.screen, x, y)

	waitFor(t, "the clicked row to be selected", func() bool {
		var row, column int
		app.pocket.QueueUpdateDraw(func() { row, column = app.page.table.GetSelection() })
		return row == 1 && column == 0
	})
	waitFor(t, "the clicked row's code to be copied", func() bool {
		var n int
		app.pocket.QueueUpdateDraw(func() { n = len(*copied) })
		return n > 0
	})

	var got []string
	var selected, selectedColumn int
	var status, clickedCode, firstCode string
	app.pocket.QueueUpdateDraw(func() {
		got = append(got, *copied...)
		selected, selectedColumn = app.page.table.GetSelection()
		status = app.page.status.GetText(true)
		clickedCode = strings.ReplaceAll(app.page.table.GetCell(1, 3).Text, " ", "")
		firstCode = strings.ReplaceAll(app.page.table.GetCell(0, 3).Text, " ", "")
	})

	if selected != 1 {
		t.Fatalf("selected row after the click is %d, want 1", selected)
	}
	if selectedColumn != 0 {
		t.Fatalf("selected column after the click is %d, want 0", selectedColumn)
	}
	if len(got) != 1 {
		t.Fatalf("clipboard writes after the click = %d, want 1: %v", len(got), got)
	}
	if clickedCode == firstCode {
		t.Fatalf("both rows show the same code %q, cannot tell which row was copied", clickedCode)
	}
	if got[0] != clickedCode {
		t.Fatalf("copied code = %q, want the clicked row's code %q", got[0], clickedCode)
	}
	if want := "copied code for GitLab: bob"; status != want {
		t.Fatalf("status after the click = %q, want %q", status, want)
	}
}

// TestClickAuthenticatorTableOutsideRowsCopiesNothing clicks the empty space
// below the last row; the table's own click handler selects nothing there, so
// no code is copied and the status line keeps its hint.
func TestClickAuthenticatorTableOutsideRowsCopiesNothing(t *testing.T) {
	copied := stubClipboardWriter(t)
	app := startAuthenticatorClickApp(t, authenticatorClickEntries)

	var x, y, w, h int
	waitFor(t, "the authenticator table to be laid out", func() bool {
		app.pocket.QueueUpdateDraw(func() { x, y, w, h = app.page.table.GetRect() })
		return w > 0 && h > 0
	})
	app.loop.clickScreen(t, app.screen, x+3, y+h-2)

	var n int
	var status string
	app.pocket.QueueUpdateDraw(func() {
		n = len(*copied)
		status = app.page.status.GetText(true)
	})
	if n != 0 {
		t.Fatalf("clipboard writes after a click outside the rows = %d, want 0", n)
	}
	if status != statusHint {
		t.Fatalf("status after a click outside the rows = %q, want %q", status, statusHint)
	}
}

// TestClickAuthenticatorTableWithNoEntriesCopiesNothing clicks the page while
// it shows the empty table view; no entry exists to copy and the status line
// keeps its hint.
func TestClickAuthenticatorTableWithNoEntriesCopiesNothing(t *testing.T) {
	copied := stubClipboardWriter(t)
	app := startAuthenticatorClickApp(t, nil)

	var x, y, w, h int
	waitFor(t, "the empty authenticator view to be laid out", func() bool {
		app.pocket.QueueUpdateDraw(func() { x, y, w, h = app.page.empty.GetRect() })
		return w > 0 && h > 0
	})
	app.loop.clickScreen(t, app.screen, x+w/2, y+h/2)

	var n int
	var status string
	app.pocket.QueueUpdateDraw(func() {
		n = len(*copied)
		status = app.page.status.GetText(true)
	})
	if n != 0 {
		t.Fatalf("clipboard writes with no entries = %d, want 0", n)
	}
	if status != statusHint {
		t.Fatalf("status with no entries = %q, want %q", status, statusHint)
	}
}

// TestClickAuthenticatorEmptyTextKeepsFocus clicks the empty-state text. It is
// inert, so the click must leave the focus on the options list.
func TestClickAuthenticatorEmptyTextKeepsFocus(t *testing.T) {
	app := startAuthenticatorClickApp(t, nil)
	app.pocket.QueueUpdateDraw(func() { app.pocket.SetFocus(app.page.Options) })

	var x, y, w, h int
	waitFor(t, "the empty authenticator view to be laid out", func() bool {
		app.pocket.QueueUpdateDraw(func() { x, y, w, h = app.page.empty.GetRect() })
		return w > 0 && h > 0
	})
	app.loop.clickScreen(t, app.screen, x+w/2, y+h/2)

	if got := app.pocket.GetFocus(); got != app.page.Options {
		t.Fatalf("focus after clicking the empty text is %T, want the options list", got)
	}
}

// TestEnterStillCopiesAuthenticatorCode keeps the keyboard path honest: Enter
// on the focused table still copies the selected entry's code.
func TestEnterStillCopiesAuthenticatorCode(t *testing.T) {
	copied := stubClipboardWriter(t)
	app := startAuthenticatorClickApp(t, authenticatorClickEntries)

	app.screen.PostEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	app.loop.flushScreen(t, app.screen)

	waitFor(t, "Enter to copy the selected entry's code", func() bool {
		var n int
		app.pocket.QueueUpdateDraw(func() { n = len(*copied) })
		return n > 0
	})

	var got string
	var status, selectedCode string
	app.pocket.QueueUpdateDraw(func() {
		got = (*copied)[0]
		status = app.page.status.GetText(true)
		selectedCode = strings.ReplaceAll(app.page.table.GetCell(0, 3).Text, " ", "")
	})
	if got != selectedCode {
		t.Fatalf("copied code = %q, want the selected row's code %q", got, selectedCode)
	}
	if want := "copied code for GitHub: alice"; status != want {
		t.Fatalf("status after Enter = %q, want %q", status, want)
	}
}

// authenticatorTestApp bundles the app a click test drives with the loop that
// runs it.
type authenticatorTestApp struct {
	pocket *Pocket
	page   *AuthenticatorPage
	screen tcell.SimulationScreen
	loop   *mouseTestLoop
}

// startAuthenticatorClickApp builds the app the way the existing event-loop
// test does: a simulation screen instead of a terminal, the authenticator page
// in front, and the entry loop running in a goroutine.
func startAuthenticatorClickApp(t *testing.T, entries []TotpEntry) *authenticatorTestApp {
	t.Helper()
	pocket := NewApp()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(120, 50)
	pocket.SetScreen(screen)

	page := pocket.AuthenticatorPage
	page.entries = entries
	pocket.ToPage(PageAuthenticator)
	page.render()
	if len(entries) > 0 {
		page.focusTable(pocket)
	}

	app := &authenticatorTestApp{pocket: pocket, page: page, screen: screen}
	app.loop = startMouseTestLoop(t, pocket.Application)
	return app
}

// stubClipboardWriter replaces the substitutable clipboard writer, so no test
// reaches the system pasteboard, and records what the app copies.
func stubClipboardWriter(t *testing.T) *[]string {
	t.Helper()
	var copied []string
	original := StWriteClipboard
	StWriteClipboard = func(text string) error {
		copied = append(copied, text)
		return nil
	}
	t.Cleanup(func() { StWriteClipboard = original })
	return &copied
}

// authenticatorRowPoint returns the cell coordinates of a table row, taken from
// the table's rectangle so a layout change does not break the test. The table
// has an outer border and no grid, so each row takes one screen line.
func authenticatorRowPoint(t *testing.T, app *authenticatorTestApp, row int) (int, int) {
	t.Helper()
	var x, y, w, h int
	waitFor(t, "the authenticator table to be laid out", func() bool {
		app.pocket.QueueUpdateDraw(func() { x, y, w, h = app.page.table.GetRect() })
		return w > 0 && h > 0
	})
	return x + 3, y + 1 + row
}
