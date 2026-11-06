package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// mouseTestLoop owns the tview application a simulation-screen test drives.
// Start it after installing any package-level stub: its cleanup stops the loop
// and waits for Run to return, and cleanups run last-in-first-out, so the
// stop happens before the stub's own cleanup restores the global that the loop
// may still read.
type mouseTestLoop struct {
	app     *tview.Application
	done    chan struct{}
	flushed chan struct{}
}

// startMouseTestLoop starts app's event loop on a goroutine and registers a
// cleanup that waits for it to stop.
func startMouseTestLoop(t *testing.T, app *tview.Application) *mouseTestLoop {
	t.Helper()

	loop := &mouseTestLoop{
		app:     app,
		done:    make(chan struct{}),
		flushed: make(chan struct{}, 1),
	}
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF24 {
			select {
			case loop.flushed <- struct{}{}:
			default:
			}
			return nil
		}
		return event
	})

	go func() {
		defer close(loop.done)
		_ = app.Run()
	}()
	t.Cleanup(func() { loop.stop(t) })
	return loop
}

// stop stops the event loop and waits until Run has returned, so a later
// cleanup can safely restore a global the loop may still read.
func (l *mouseTestLoop) stop(t *testing.T) {
	t.Helper()
	l.app.Stop()
	select {
	case <-l.done:
	case <-time.After(3 * time.Second):
		t.Error("tview event loop did not stop")
	}
}

// clickScreen posts a left click at x, y and waits until the event loop has
// handled it.
func (l *mouseTestLoop) clickScreen(t *testing.T, screen tcell.SimulationScreen, x, y int) {
	t.Helper()
	screen.PostEvent(tcell.NewEventMouse(x, y, tcell.ButtonPrimary, tcell.ModNone))
	screen.PostEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
	l.flushScreen(t, screen)
}

// flushScreen waits until the event loop has handled every screen event posted
// before the call.
func (l *mouseTestLoop) flushScreen(t *testing.T, screen tcell.SimulationScreen) {
	t.Helper()
	screen.PostEvent(tcell.NewEventKey(tcell.KeyF24, 0, tcell.ModNone))
	l.waitFlushed(t)
}

// flushQueued waits until the event loop has handled every event queued with
// Application.QueueEvent before the call.
func (l *mouseTestLoop) flushQueued(t *testing.T) {
	t.Helper()
	l.app.QueueEvent(tcell.NewEventKey(tcell.KeyF24, 0, tcell.ModNone))
	l.waitFlushed(t)
}

func (l *mouseTestLoop) waitFlushed(t *testing.T) {
	t.Helper()
	select {
	case <-l.flushed:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the event loop to flush posted events")
	}
}

// waitFor polls cond until it holds or the deadline passes, so a test does not
// depend on how long the event loop needs to consume a posted event.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
