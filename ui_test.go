package main

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"ascii fits", "GitHub", 18, "GitHub"},
		{"ascii exact", "abcdefghijklmnopqr", 18, "abcdefghijklmnopqr"},
		{"ascii long", "abcdefghijklmnopqrstuvwxyz", 18, "abcdefghijklmnopqr…"},
		{"multi-byte fits", "认证服务", 6, "认证服务"},
		{"multi-byte long", "认证服务名字很长", 4, "认证服务…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateRunes(c.in, c.n)
			if got != c.want {
				t.Fatalf("truncateRunes(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("truncateRunes(%q, %d) = %q, not valid UTF-8", c.in, c.n, got)
			}
		})
	}
}

func TestSecondsCellTextNonPositivePeriod(t *testing.T) {
	now := time.Unix(59, 0)
	for _, period := range []int{0, -30} {
		if got := secondsCellText(period, now); got != "" {
			t.Fatalf("secondsCellText(%d) = %q, want empty", period, got)
		}
	}
}

func TestDeleteConfirmTextNamesEntry(t *testing.T) {
	e := TotpEntry{Issuer: "GitHub", Account: "alice@example.com"}
	if got, want := deleteConfirmText(e), "Deleting GitHub: alice@example.com"; got != want {
		t.Fatalf("deleteConfirmText() = %q, want %q", got, want)
	}

	e.Remark = "work"
	if got, want := deleteConfirmText(e), "Deleting GitHub: alice@example.com (work)"; got != want {
		t.Fatalf("deleteConfirmText() with a remark = %q, want %q", got, want)
	}
}

func TestAuthenticatorRefreshDoesNotBlockEventLoop(t *testing.T) {
	app := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(80, 25)
	app.SetScreen(screen)

	pocket := &Pocket{Application: app, Pages: tview.NewPages()}
	a := NewAuthenticatorPage(pocket)
	pocket.AuthenticatorPage = a
	pocket.Pages.AddPage(PageAuthenticator, a, true, true)
	a.entries = []TotpEntry{{
		Issuer:    "GitHub",
		Account:   "alice",
		Secret:    "JBSWY3DPEHPK3PXP",
		Digits:    6,
		Period:    30,
		Algorithm: "SHA1",
	}}
	a.fillTable()
	// Force the next refresh to change a cell, which is the path that used to
	// call Draw from inside a queued update and deadlock the event loop.
	a.table.GetCell(0, 4).SetText("999s")

	app.SetRoot(pocket.Pages, true)
	go func() { _ = app.Run() }()
	defer func() { go app.Stop() }()

	done := make(chan struct{})
	go func() {
		a.refresh(pocket)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("refresh blocked the tview event loop")
	}
}
