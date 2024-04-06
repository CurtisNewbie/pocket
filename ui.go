package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cast"
)

const (
	PageList   = "list"
	PageDetail = "detail"
	PageSearch = "search"
	PageCreate = "create"
	PageEdit   = "edit"
)

// TODO: merges the view and options together as single value
type Pocket struct {
	App        *tview.Application
	Pages      *tview.Pages
	DetailPage *DetailPage
	ListPage   *ListPage
}

func (p *Pocket) ToPage(page string) {
	p.Pages.SwitchToPage(page)
}

func (p *Pocket) RemovePage(page string) {
	p.Pages.RemovePage(page)
}

func (p *Pocket) QueueCommand(f func()) {
	p.App.QueueUpdateDraw(f)
}

func (p *Pocket) Stop() {
	p.App.Stop()
}

type ListPage struct {
	*tview.Flex
	Options *tview.List
	View    *ListView
}

func (l *ListPage) SetPage(n int) {
	l.View.pageNum = n
}

func (l *ListPage) GetPage() int {
	return l.View.pageNum
}

func (l *ListPage) GetPageStr() string {
	return cast.ToString(l.View.pageNum)
}

func (l *ListPage) AddPage(delta int) int {
	l.View.pageNum += delta
	return l.View.pageNum
}

func NewListPage(pocket *Pocket) *ListPage {
	lp := new(ListPage)
	lv := NewListView(pocket)
	opt := NewOptionList().
		AddItem("Create Item", "", 'c', func() {
			PopCreateNotePage(pocket)
		}).
		AddItem("Search Param", "", '/', func() {
			PopEditSearchPage(lv, pocket.Pages)
		}).
		AddItem("Select Item", "", 's', func() {
			c := lv.content.GetItemCount()
			if c < 1 {
				pocket.App.SetFocus(lv.content)
			} else {
				pocket.App.SetFocus(lv.content.GetItem(0))
			}
		}).
		AddItem("Next Page", "", 'n', func() {
			pocket.ListPage.AddPage(1)
			lv.page.SetText(pocket.ListPage.GetPageStr())
		}).
		AddItem("Prev Page", "", 'N', func() {
			if pocket.ListPage.GetPage() > 1 {
				n := pocket.ListPage.AddPage(-1)
				lv.page.SetText(cast.ToString(n))
			}
		}).
		AddItem("Exit", "", 'q', func() { pocket.Stop() })

	cp := NewContentPlane(opt, lv.flex)

	// TODO: demo
	go func() {
		lv.name.SetText(`Goody`)
		lv.AddItem(ListItem{
			id:   1,
			name: "yo",
			desc: "yo it's me",
		})
		lv.AddItem(ListItem{
			id:   2,
			name: "yo",
			desc: "yo it's me",
		})
		lv.AddItem(ListItem{
			id:   3,
			name: "yo",
			desc: "yo it's me",
		})
	}()

	lp.Options = opt
	lp.View = lv
	lp.Flex = cp
	return lp
}

func PopEditSearchPage(liv *ListView, pages *tview.Pages) {
	closePopup := func() {
		pages.SwitchToPage(PageList)
		pages.RemovePage(PageSearch)
	}

	var tmpName string = ""

	form := NewForm()
	form.AddInputField("Name:", tmpName, 60, nil, func(t string) { tmpName = t })
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

	popup := createPopup(pages, form, 5, 70)
	pages.AddPage(PageSearch, popup, true, true)
}

func PopEditNotePage(pocket *Pocket, it ListItem) {
	closePopup := func() {
		pocket.ToPage(PageDetail)
		pocket.DetailPage.View.Display(it)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm()
	var tmpName string = it.name
	var tmpDesc string = it.desc
	var tmpContent string = it.content

	form.AddInputField("Name:", tmpName, 30, nil, func(t string) { tmpName = t })
	form.AddTextArea("Description:", tmpDesc, 100, 5, 250, func(t string) { tmpDesc = t })
	form.AddTextArea("Content:", tmpContent, 100, 20, 500, func(t string) { tmpContent = t })

	confirm := func() {
		pocket.DetailPage.View.Display(ListItem{
			name:    tmpName,
			desc:    tmpDesc,
			content: tmpContent,
			ctime:   it.ctime,
			utime:   time.Now(),
		})
		pocket.RemovePage(PageEdit)
	}
	form.AddButton("Confirm", confirm)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Edit Note ")

	popup := createPopup(pocket.Pages, form, 35, 100)
	pocket.Pages.AddPage(PageEdit, popup, true, true)
}

func PopCreateNotePage(pocket *Pocket) {
	closePopup := func() {
		pocket.ToPage(PageList)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm()
	var tmpName string = ""
	var tmpDesc string = ""
	var tmpContent string = ""

	form.AddInputField("Name:", tmpName, 30, nil, func(t string) { tmpName = t })
	form.AddTextArea("Description:", tmpDesc, 100, 5, 250, func(t string) { tmpDesc = t })
	form.AddTextArea("Content:", tmpContent, 100, 20, 500, func(t string) { tmpContent = t })

	confirm := func() {
		pocket.ToPage(PageDetail)
		ctime := time.Now()
		pocket.DetailPage.View.Display(ListItem{
			name:    tmpName,
			desc:    tmpDesc,
			content: tmpContent,
			ctime:   ctime,
			utime:   ctime,
		})
		pocket.RemovePage(PageCreate)
	}
	form.AddButton("Confirm", confirm)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Create Bootmark ")

	popup := createPopup(pocket.Pages, form, 35, 100)
	pocket.Pages.AddPage(PageCreate, popup, true, true)
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

type DetailPage struct {
	*tview.Flex
	Options *tview.List
	View    *DetailView
}

func NewDetailPage(pocket *Pocket) *DetailPage {
	dp := new(DetailPage)
	vw := NewDetailView(pocket)
	options := NewOptionList().
		AddItem("Edit", "", 'e', func() {
			PopEditNotePage(pocket, vw.Item)
		}).
		AddItem("Mask", "", 'm', func() {
		}).
		AddItem("Unmask", "", 'M', func() {
		}).
		AddItem("Exit", "", 'q', func() {
			pocket.Pages.SwitchToPage(PageList)
		})

	p := NewContentPlane(options, vw.flex)

	dp.Flex = p
	dp.Options = options
	dp.View = vw
	return dp
}

func NewApp() *tview.Application {
	app := tview.NewApplication()
	pages := tview.NewPages()
	pocket := &Pocket{
		App:   app,
		Pages: pages,
	}

	detailPage := NewDetailPage(pocket)
	pocket.DetailPage = detailPage

	listPage := NewListPage(pocket)
	pocket.ListPage = listPage

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

	Item ListItem
}

func (d *DetailView) Display(li ListItem) {
	d.name.SetText(li.name)
	d.desc.SetText(li.desc)
	d.content.SetText(li.content)
	d.ctime.SetText(FormatTime(li.ctime))
	d.utime.SetText(FormatTime(li.utime))
	d.Item = li
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
	id      int
	name    string
	desc    string
	content string
	ctime   time.Time
	utime   time.Time
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
			pocket.App.SetFocus(pocket.ListPage.Options)
			return nil
		}

		if evt.Key() == tcell.KeyEnter {
			j, ok := FindFocus(iv.content)
			if ok {
				itm := iv.content.GetItem(j)
				lip := itm.(*ListItemPrimitive)

				// TODO: should be a database lookup
				pocket.DetailPage.View.Display(ListItem{
					id:      lip.id,
					name:    lip.name,
					desc:    lip.desc,
					content: lip.content,
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

func FormatTime(t time.Time) string {
	return t.Format("2006/01/02 15:04:05")
}

// Form with grey background color, and only uses Shift+Tab to move cursor between inputs/buttons to support typing \t in textarea.
func NewForm() *tview.Form {
	form := tview.NewForm()
	form.SetFieldBackgroundColor(tcell.ColorDarkSlateGrey)
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {

		DebugLog(" %d %d - %d\n", ev.Key(), ev.Rune(), ev.Modifiers())

		if ev.Key() == tcell.KeyTAB && ev.Modifiers() == tcell.ModNone { // only TAB, append at the end \t
			ta, _, ok := FindFocusedTextArea(form)
			if ok {
				ta.SetText(ta.GetText()+"\t", true)
				return nil
			}
		}

		if (ev.Key() == tcell.KeyBacktab) || (ev.Key() == tcell.KeyTAB && ev.Modifiers() == tcell.ModNone) { // Shift+Tab, switch to next input
			return tcell.NewEventKey(tcell.KeyTAB, 0, tcell.ModNone)
		}

		return ev
	})
	return form
}

func FindFocusedTextArea(form *tview.Form) (*tview.TextArea, int, bool) {
	i, _ := form.GetFocusedItemIndex()
	if i > -1 {
		if v, ok := form.GetFormItem(i).(*tview.TextArea); ok {
			return v, i, true
		}
	}
	return nil, 0, false
}
