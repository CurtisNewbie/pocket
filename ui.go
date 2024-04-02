package main

import (
	"fmt"

	"github.com/curtisnewbie/miso/miso"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewApp() (app *tview.Application) {
	app = tview.NewApplication()
	pages := tview.NewPages()
	infv := NewInfoView(app)
	options := NewOptionList().
		AddItem("Create", "", 'c', func() {
		}).
		AddItem("Edit", "", 'e', func() {
		}).
		AddItem("Search", "", '/', func() {
		}).
		AddItem("Unmask", "", 'u', func() {
		}).
		AddItem("Mask", "", 'm', func() {
		}).
		AddItem("Exit", "", 'q', func() {
			app.Stop()
		})

	cp := NewContentPlane(options, infv.flex)
	pages.AddPage("main", cp, true, true)
	app.SetRoot(pages, true) // TODO: page to validate username/password

	// TODO: demo
	go func() {
		infv.SetContent(`Hello World!!!!`)
		infv.SetName(`Goody`)
		infv.SetDesc(`Very good stuff Very good stuff Very good stuff Very good stuff`)
		infv.SetUTime(miso.Now().FormatClassic())
		infv.SetCTime(miso.Now().FormatClassic())
		infv.SetBar(`Very good stuff`)
	}()
	return app
}

func NewContentPlane(options tview.Primitive, content tview.Primitive) *tview.Flex {
	ctnp := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(options, 30, 1, true).
		AddItem(content, 0, 4, false)

	ver := tview.NewTextView()
	ver.SetBorder(true)
	ver.SetText(fmt.Sprintf("Pocket %v", Version))
	ver.SetTextAlign(tview.AlignCenter)
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ctnp, 0, 20, true).
		AddItem(ver, 3, 1, false)

	return layout
}

func NewOptionList() *tview.List {
	l := tview.NewList()
	l.SetBorder(true).SetTitle(" Options ")
	l.ShowSecondaryText(false)

	// capture hjkl
	l.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' {
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		if event.Rune() == 'j' {
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		}
		if event.Rune() == 'k' {
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		}
		if event.Rune() == 'l' {
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		}
		return event
	})
	return l
}

type InfoView struct {
	app  *tview.Application
	flex *tview.Flex

	bar     *tview.TextView
	name    *tview.TableCell
	ctime   *tview.TableCell
	utime   *tview.TableCell
	desc    *tview.TableCell
	content *tview.TextView
}

func (iv *InfoView) SetContent(s string) {
	QueueCommand(iv.app, func() { iv.content.SetText(s) })
}

func (iv *InfoView) SetBar(s string) {
	QueueCommand(iv.app, func() { iv.bar.SetText(s) })
}

func (iv *InfoView) SetName(s string) {
	QueueCommand(iv.app, func() { iv.name.SetText(s) })
}

func (iv *InfoView) SetCTime(s string) {
	QueueCommand(iv.app, func() { iv.ctime.SetText(s) })
}

func (iv *InfoView) SetUTime(s string) {
	QueueCommand(iv.app, func() { iv.utime.SetText(s) })
}

func (iv *InfoView) SetDesc(s string) {
	QueueCommand(iv.app, func() { iv.desc.SetText(s) })
}

func QueueCommand(app *tview.Application, f func()) {
	app.QueueUpdateDraw(f)
}

func NewInfoView(app *tview.Application) (iv *InfoView) {
	topFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	iv = &InfoView{}
	iv.app = app

	iv.bar = tview.NewTextView()
	iv.bar.SetBorder(true)
	iv.bar.SetText(" ")
	iv.bar.SetTextAlign(tview.AlignCenter)
	topFlex.AddItem(iv.bar, 3, 1, false)

	tb := tview.NewTable()
	tb.SetBorder(true).SetTitle(" Info ")

	tb.SetCellSimple(0, 1, "Name:")
	tb.GetCell(0, 1).SetAlign(tview.AlignRight)
	iv.name = tview.NewTableCell("")
	tb.SetCell(0, 2, iv.name)

	tb.SetCellSimple(1, 1, "Create Time:")
	tb.GetCell(1, 1).SetAlign(tview.AlignRight)
	iv.ctime = tview.NewTableCell("")
	tb.SetCell(1, 2, iv.ctime)

	tb.SetCellSimple(2, 1, "Update Time:")
	tb.GetCell(2, 1).SetAlign(tview.AlignRight)
	iv.utime = tview.NewTableCell("")
	tb.SetCell(2, 2, iv.utime)

	tb.SetCellSimple(3, 1, "Description:")
	tb.GetCell(3, 1).SetAlign(tview.AlignRight)
	iv.desc = tview.NewTableCell("")
	tb.SetCell(3, 2, iv.desc)

	infp := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tb, 0, 1, false)

	topFlex.AddItem(infp, 0, 1, false)

	iv.content = tview.NewTextView()
	iv.content.SetBorder(true).SetTitle(" Content ")
	iv.content.SetChangedFunc(func() { app.Draw() })

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 10, 1, true).
		AddItem(iv.content, 0, 1, false)

	iv.flex = mainFlex

	return iv
}
