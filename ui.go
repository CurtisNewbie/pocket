package main

import (
	"fmt"

	"github.com/curtisnewbie/miso/miso"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cast"
)

const (
	PageList   = "list"
	PageDetail = "detail"
	PageSearch = "search"
)

// TODO: merges the view and options together as single value
type Pocket struct {
	App           *tview.Application
	Pages         *tview.Pages
	ListView      *ListView
	ListOptions   *tview.List
	DetailView    *DetailView
	DetailOptions *tview.List
}

func (p *Pocket) ToPage(page string) {
	p.Pages.SwitchToPage(page)
}

func (p *Pocket) QueueCommand(f func()) {
	p.App.QueueUpdateDraw(f)
}

func (p *Pocket) Stop() {
	p.App.Stop()
}

func NewListPage(pocket *Pocket) tview.Primitive {
	liv := NewListView(pocket)
	options := NewOptionList().
		AddItem("Create", "", 'c', func() {
		}).
		AddItem("Search", "", '/', func() {
			EditSearchPage(liv, pocket.Pages)
		}).
		AddItem("Select", "", 's', func() {
			pocket.App.SetFocus(liv.content)
		}).
		AddItem("Next", "", 'n', func() {
			pocket.ListView.pageNum += 1
			liv.page.SetText(cast.ToString(pocket.ListView.pageNum))
		}).
		AddItem("Prev", "", 'N', func() {
			if pocket.ListView.pageNum > 1 {
				pocket.ListView.pageNum -= 1
				liv.page.SetText(cast.ToString(pocket.ListView.pageNum))
			}
		}).
		AddItem("Exit", "", 'q', func() { pocket.Stop() })

	pocket.ListView = liv
	pocket.ListOptions = options

	p := NewContentPlane(options, liv.flex)

	// TODO: demo
	go func() {
		liv.name.SetText(`Goody`)
		liv.AddItem(ListItem{
			id:   1,
			name: "yo",
			desc: "yo it's me",
		})
		liv.AddItem(ListItem{
			id:   2,
			name: "yo",
			desc: "yo it's me",
		})
		liv.AddItem(ListItem{
			id:   3,
			name: "yo",
			desc: "yo it's me",
		})
		// liv.desc.SetText(`Very good stuff`)
	}()
	return p
}

func EditSearchPage(liv *ListView, pages *tview.Pages) {
	closePopup := func() {
		pages.SwitchToPage(PageList)
		pages.RemovePage(PageSearch)
	}

	form := tview.NewForm()
	var tmpName string = liv.name.Text
	form.AddInputField("Name:", liv.name.Text, 30, nil, func(t string) { tmpName = t })
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Search Parameters ")
	form.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyEnter {
			liv.name.SetText(tmpName)
			closePopup()
			return nil
		}
		return evt
	})

	popup := createPopup(pages, form, 20, 60)
	pages.AddPage(PageSearch, popup, true, true)
}

func createPopup(pages *tview.Pages, form tview.Primitive, height int, width int) tview.Primitive {
	modal := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
	return modal
}

func NewDetailPage(pocket *Pocket) tview.Primitive {
	vw := NewDetailView(pocket)
	options := NewOptionList().
		AddItem("Edit", "", 'e', func() {
		}).
		AddItem("Mask", "", 'm', func() {
		}).
		AddItem("Unmask", "", 'u', func() {
		}).
		AddItem("Exit", "", 'q', func() {
			pocket.Pages.SwitchToPage(PageList)
		})

	pocket.DetailView = vw
	pocket.DetailOptions = options
	p := NewContentPlane(options, vw.flex)

	// TODO: demo
	go func() {
		pocket.QueueCommand(func() {
			vw.bar.SetText(`Hello World!!!!`)
			vw.name.SetText(`Goody`)
			vw.content.SetText(`Very good stuff Very good stuff Very good stuff Very good stuff`)
			vw.ctime.SetText(miso.Now().FormatClassic())
			vw.utime.SetText(miso.Now().FormatClassic())
			vw.desc.SetText(`Very good stuff`)
		})
	}()
	return p
}

func NewApp() *tview.Application {
	app := tview.NewApplication()
	pages := tview.NewPages()
	pocket := &Pocket{
		App:   app,
		Pages: pages,
	}

	detailPage := NewDetailPage(pocket)
	listPage := NewListPage(pocket)

	pages.AddPage(PageDetail, detailPage, true, true)
	pages.AddPage(PageList, listPage, true, true)
	pages.SwitchToPage(PageList)
	app.SetRoot(pages, true) // TODO: page to validate username/password
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

type DetailView struct {
	flex *tview.Flex

	bar     *tview.TextView
	name    *tview.TableCell
	ctime   *tview.TableCell
	utime   *tview.TableCell
	desc    *tview.TableCell
	content *tview.TextView
}

func (d *DetailView) Display(li ListItem) {
	d.name.SetText(li.name)
	d.desc.SetText(li.desc)
}

func NewDetailView(pocket *Pocket) (iv *DetailView) {
	topFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	iv = new(DetailView)
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
	iv.content.SetChangedFunc(func() { pocket.App.Draw() })

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 10, 1, true).
		AddItem(iv.content, 0, 1, false)

	iv.flex = mainFlex

	return iv
}

type ListView struct {
	flex    *tview.Flex
	bar     *tview.TextView
	name    *tview.TableCell
	page    *tview.TableCell
	content *tview.Flex

	pageNum int
}

type ListItem struct {
	id   int
	name string
	desc string
}

type ListItemPrimitive struct {
	*tview.Table
	ListItem
}

func (l *ListView) AddItem(itm ListItem) {
	tb := tview.NewTable()
	lip := new(ListItemPrimitive)
	lip.Table = tb
	lip.ListItem = itm
	tb.SetBorder(true)

	tb.SetCellSimple(0, 0, "Id:")
	tb.GetCell(0, 0).SetAlign(tview.AlignRight)
	idc := tview.NewTableCell(cast.ToString(itm.id))
	tb.SetCell(0, 1, idc)

	tb.SetCellSimple(1, 0, "Name:")
	tb.GetCell(1, 0).SetAlign(tview.AlignRight)
	namec := tview.NewTableCell(itm.name)
	tb.SetCell(1, 1, namec)

	tb.SetCellSimple(2, 0, "Description:")
	tb.GetCell(2, 0).SetAlign(tview.AlignRight)
	descc := tview.NewTableCell(itm.desc)
	tb.SetCell(2, 1, descc)
	lip.SetFocusFunc(func() { lip.SetBorderColor(tcell.ColorYellow) })
	lip.SetBlurFunc(func() { lip.SetBorderColor(tcell.ColorWhite) })
	l.content.AddItem(lip, 5, 1, false)
}

func NewListView(pocket *Pocket) (iv *ListView) {
	topFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	iv = new(ListView)
	iv.bar = tview.NewTextView()
	iv.bar.SetText(`List View`)
	iv.bar.SetBorder(true)
	iv.bar.SetTextAlign(tview.AlignCenter)
	topFlex.AddItem(iv.bar, 3, 1, false)

	tb := tview.NewTable()
	tb.SetBorder(true).SetTitle(" Searching Parameters ")

	tb.SetCellSimple(0, 1, "Name:")
	tb.GetCell(0, 1).SetAlign(tview.AlignRight)
	iv.name = tview.NewTableCell("")
	tb.SetCell(0, 2, iv.name)

	tb.SetCellSimple(1, 1, "Page:")
	tb.GetCell(1, 1).SetAlign(tview.AlignRight)
	iv.page = tview.NewTableCell("1")
	tb.SetCell(1, 2, iv.page)

	infp := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tb, 0, 1, false)

	topFlex.AddItem(infp, 0, 2, false)

	iv.content = tview.NewFlex().SetDirection(tview.FlexRow)
	iv.content.SetBorder(true).SetTitle(" Records ")

	iv.content.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyESC || evt.Rune() == 'q' {
			pocket.App.SetFocus(pocket.ListOptions)
			return nil
		}

		if evt.Key() == tcell.KeyEnter {
			j, ok := FindFocus(iv.content)
			if ok {
				itm := iv.content.GetItem(j)
				lip := itm.(*ListItemPrimitive)

				// TODO: should be a database lookup
				pocket.DetailView.Display(ListItem{
					id:   lip.id,
					name: lip.name,
					desc: lip.desc,
				})
				pocket.Pages.SwitchToPage(PageDetail)
			}
			return nil
		}

		r := evt.Rune()
		switch r {
		case 'j', 'k':
			l := iv.content.GetItemCount()
			i, ok := FindFocus(iv.content)
			if ok {
				switch r {
				case 'j':
					if i < l-1 {
						pocket.App.SetFocus(iv.content.GetItem(i + 1))
					}
				case 'k':
					if i > 0 {
						pocket.App.SetFocus(iv.content.GetItem(i - 1))
					}
				}
			} else if l > 0 {
				pocket.App.SetFocus(iv.content.GetItem(0))
			}
		}
		return evt
	})

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 10, 1, true).
		AddItem(iv.content, 0, 1, false)

	iv.flex = mainFlex

	return iv
}

func FindFocus(f *tview.Flex) (int, bool) {
	var i int = -1
	l := f.GetItemCount()
	for j := 0; j < l; j++ {
		it := f.GetItem(j)
		if it.HasFocus() {
			i = j
			break
		}
	}
	return i, i > -1
}
