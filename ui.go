package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cast"
)

const (
	PagePassword = "delete"
	PageList     = "list"
	PageDetail   = "detail"
	PageSearch   = "search"
	PageCreate   = "create"
	PageEdit     = "edit"
	PageDelete   = "delete"
	PageMsg      = "message"
	PageExit     = "exit"
	PageConfirm  = "confirm"

	PageLimit = 10

	LabelName    = "Name:"
	LabelDesc    = "Description:"
	LabelContent = "Content:"
)

type Pocket struct {
	*tview.Application
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

type ListPage struct {
	*tview.Flex
	*ListView
	Options *tview.List
}

func (l *ListPage) SetPage(n int) {
	l.pageNum = n
}

func (l *ListPage) GetPage() int {
	return l.pageNum
}

func (l *ListPage) GetPageStr() string {
	return cast.ToString(l.pageNum)
}

func (l *ListPage) AddPage(delta int) int {
	l.pageNum += delta
	return l.pageNum
}

func (l *ListPage) FocusOne(pocket *Pocket) {
	c := l.ListView.content.GetItemCount()
	if c > 0 {
		pocket.SetFocus(l.ListView.content.GetItem(0))
	}
}

func NewListPage(pocket *Pocket) *ListPage {
	lp := new(ListPage)
	lv := NewListView(pocket, func(event *tcell.EventKey) (*tcell.EventKey, bool) {
		if event.Rune() == 'c' {
			PopCreateNotePage(pocket, func() {
				UIFetchNotes(pocket, 0)
			})
			return nil, true
		}
		return nil, false
	})

	extendedCap := func(event *tcell.EventKey) (*tcell.EventKey, bool) {
		if event.Key() == tcell.KeyESC {
			if lv.name.Text != "" {
				lv.name.SetText("")
			}
			UIFetchNotes(pocket, 0)
			return nil, true
		}
		if event.Rune() == 'l' || event.Key() == tcell.KeyRight {
			lp.FocusOne(pocket)
			return nil, true
		}
		return nil, false
	}
	opt := NewOptionList(extendedCap).
		AddItem("Create Item", "", 'c', func() {
			PopCreateNotePage(pocket, func() {
				UIFetchNotes(pocket, 0)
			})
		}).
		AddItem("Select Item", "", 'l', func() {
			lp.FocusOne(pocket)
		}).
		AddItem("Search Param", "", '/', func() {
			PopEditSearchPage(pocket)
		}).
		AddItem("Next Page", "", 'n', func() {
			lv.page.SetText(pocket.ListPage.GetPageStr())
			UIFetchNotes(pocket, 1)
		}).
		AddItem("Prev Page", "", 'N', func() {
			if pocket.ListPage.GetPage() > 1 {
				UIFetchNotes(pocket, -1)
			}
		}).
		AddItem("Exit", "", 'q', func() {
			PopExitPage(pocket)
		})

	cp := NewContentPlane(opt, lv.flex)
	lp.Options = opt
	lp.ListView = lv
	lp.Flex = cp

	return lp
}

func PopEditSearchPage(pocket *Pocket) {
	liv := pocket.ListPage.ListView
	pages := pocket.Pages
	closePopup := func() {
		pages.SwitchToPage(PageList)
		pages.RemovePage(PageSearch)
	}

	prevName := liv.name.Text
	var tmpName string = ""

	form := NewForm(false)
	form.AddInputField("Match (supports AND/OR):", tmpName, 80, nil, func(t string) { tmpName = t })
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Search Parameters ")
	form.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyEnter {
			liv.name.SetText(tmpName)
			closePopup()
			if prevName != tmpName {
				UIFetchNotes(pocket, 0)
			}
			return nil
		}
		return evt
	})

	popup := createPopup(pages, form, 5, 110)
	pages.AddPage(PageSearch, popup, true, true)
}

func PopEditNotePage(pocket *Pocket, it Note) {
	closePopup := func() {
		pocket.ToPage(PageDetail)
		pocket.DetailPage.Display(it)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm(true)
	var tmpName string = it.Name
	var tmpDesc string = it.Desc
	var tmpContent string = it.Content

	form.AddInputField(LabelName, tmpName, 30, nil, nil)
	form.AddTextArea(LabelDesc, tmpDesc, 100, 5, 250, nil)
	form.AddTextArea(LabelContent, tmpContent, 100, 20, 10000, nil)

	// this is so ugly :(, but it works
	ni := form.GetFormItemByLabel(LabelName).(*tview.InputField)
	ni.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpName, func(s string) {
				tmpName = s
				ni.SetText(tmpName)
			})
			return nil
		}
		return event
	})

	di := form.GetFormItemByLabel(LabelDesc).(*tview.TextArea)
	di.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpDesc, func(s string) {
				tmpDesc = s
				di.SetText(tmpDesc, true)
			})
			return nil
		}
		return event
	})

	ci := form.GetFormItemByLabel(LabelContent).(*tview.TextArea)
	ci.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpContent, func(s string) {
				tmpContent = s
				ci.SetText(tmpContent, true)
			})
			return nil
		}
		return event
	})

	loadInput := func() {
		tmpName = ni.GetText()
		tmpDesc = di.GetText()
		tmpContent = ci.GetText()
	}

	confirm := func() {
		loadInput()
		ni := Note{
			Id:      it.Id,
			Name:    tmpName,
			Desc:    tmpDesc,
			Content: tmpContent,
			Ctime:   it.Ctime,
			Utime:   Now(),
		}
		UIEditNote(pocket, ni, func(err error) {
			if err == nil {
				pocket.DetailPage.Display(ni)
			}
			pocket.RemovePage(PageEdit)
		})
	}

	form.AddButton("Confirm", confirm)
	form.AddButton("Close", closePopup)
	form.SetCancelFunc(func() {
		loadInput()
		if tmpName == it.Name && tmpDesc == it.Desc && tmpContent == it.Content {
			closePopup()
			return
		}
		PopConfirmDialog(pocket, closePopup, "Close Dialog?", 50, 15)
	})
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Edit Note (vim-based) ")

	popup := createPopup(pocket.Pages, form, 35, 100)
	pocket.Pages.AddPage(PageEdit, popup, true, true)
}

func PopCreateNotePage(pocket *Pocket, onConfirm func()) {
	closePopup := func() {
		pocket.ToPage(PageList)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm(true)
	var tmpName string = ""
	var tmpDesc string = ""
	var tmpContent string = ""

	form.AddInputField(LabelName, tmpName, 30, nil, nil)
	form.AddTextArea(LabelDesc, tmpDesc, 100, 5, 250, nil)
	form.AddTextArea(LabelContent, tmpContent, 100, 20, 10000, nil)

	// this is so ugly :(, but it works
	ni := form.GetFormItemByLabel(LabelName).(*tview.InputField)
	ni.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpName, func(s string) {
				tmpName = s
				ni.SetText(tmpName)
			})
			return nil
		}
		return event
	})

	di := form.GetFormItemByLabel(LabelDesc).(*tview.TextArea)
	di.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpDesc, func(s string) {
				tmpDesc = s
				di.SetText(tmpDesc, true)
			})
			return nil
		}
		return event
	})

	ci := form.GetFormItemByLabel(LabelContent).(*tview.TextArea)
	ci.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			VimEdit(pocket, tmpContent, func(s string) {
				tmpContent = s
				ci.SetText(tmpContent, true)
			})
			return nil
		}
		return event
	})

	loadInput := func() {
		tmpName = ni.GetText()
		tmpDesc = di.GetText()
		tmpContent = ci.GetText()
	}

	confirm := func() {
		loadInput()
		ctime := Now()
		note := Note{
			Name:    tmpName,
			Desc:    tmpDesc,
			Content: tmpContent,
			Ctime:   ctime,
			Utime:   ctime,
		}
		UICreateNote(pocket, note, func(nt Note, err error) {
			pocket.ToPage(PageList)
			if err == nil {
				pocket.ToPage(PageDetail)
				pocket.DetailPage.Display(nt)
				onConfirm()
			} else {
				pocket.RemovePage(PageCreate)
			}
		})
	}

	form.AddButton("Confirm", confirm)
	form.AddButton("Close", closePopup)
	form.SetCancelFunc(func() {
		loadInput()
		if tmpName == "" && tmpDesc == "" && tmpContent == "" {
			closePopup()
			return
		}
		PopConfirmDialog(pocket, closePopup, "Close Dialog?", 50, 15)
	})
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Create Note (vim-based) ")

	popup := createPopup(pocket.Pages, form, 35, 120)
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
	*DetailView
	Options *tview.List
}

func NewDetailPage(pocket *Pocket) *DetailPage {
	dp := new(DetailPage)
	vw := NewDetailView(pocket)
	extendedInputCap := func(e *tcell.EventKey) (*tcell.EventKey, bool) {
		if e.Rune() == 'h' || e.Key() == tcell.KeyESC || e.Key() == tcell.KeyLeft {
			pocket.Pages.SwitchToPage(PageList)
			UIFetchNotes(pocket, 0, func() { pocket.ListPage.FocusOne(pocket) })
			return nil, true
		}

		return nil, false
	}
	options := NewOptionList(extendedInputCap).
		AddItem("Edit", "", 'e', func() {
			PopEditNotePage(pocket, vw.Item)
		}).
		AddItem("Delete", "", 'd', func() {
			PopDeleteNotePage(pocket, vw.Item)
		}).
		AddItem("Mask/Unmask", "", 'm', vw.SwitchMasking).
		AddItem("Exit", "", 'q', func() {
			pocket.Pages.SwitchToPage(PageList)
			UIFetchNotes(pocket, 0, func() { pocket.ListPage.FocusOne(pocket) })
		})

	p := NewContentPlane(options, vw.flex)

	dp.Flex = p
	dp.Options = options
	dp.DetailView = vw
	return dp
}

func NewApp() *Pocket {
	app := tview.NewApplication()
	pages := tview.NewPages()
	pocket := &Pocket{
		Application: app,
		Pages:       pages,
	}

	listPage := NewListPage(pocket)
	pocket.ListPage = listPage

	detailPage := NewDetailPage(pocket)
	pocket.DetailPage = detailPage

	pages.AddPage(PageDetail, detailPage, true, true)
	pages.AddPage(PageList, listPage, true, true)
	PopPasswordPage(pocket)
	app.SetRoot(pages, true)

	return pocket
}

func NewContentPlane(options tview.Primitive, content tview.Primitive) *tview.Flex {
	ctnp := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(options, 30, 1, true).
		AddItem(content, 0, 4, false)

	ver := tview.NewTextView()
	ver.SetBorder(true)
	ver.SetText(fmt.Sprintf("Pocket %v by yongjie.zhuang", Version))
	ver.SetTextAlign(tview.AlignCenter)
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ctnp, 0, 20, true).
		AddItem(ver, 3, 1, false)

	return layout
}

func NewOptionList(extendedCap func(event *tcell.EventKey) (*tcell.EventKey, bool)) *tview.List {
	l := tview.NewList()
	l.SetBorder(true).SetTitle(" Options ")
	l.ShowSecondaryText(false)

	// capture hjkl
	l.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'j' {
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		}
		if event.Rune() == 'k' {
			return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		}
		if extendedCap != nil {
			if ev, ok := extendedCap(event); ok {
				return ev
			}
		}
		return event
	})
	return l
}

type DetailView struct {
	flex *tview.Flex

	bar     *tview.TextView
	id      *tview.TableCell
	name    *tview.TableCell
	ctime   *tview.TableCell
	utime   *tview.TableCell
	desc    *tview.TableCell
	content *tview.TextView

	Item   Note
	Masked bool
}

func (d *DetailView) MaskNote() {
	d.Masked = false
	d.SwitchMasking()
}

func (d *DetailView) SwitchMasking() {
	if d.Masked {
		d.content.SetText(d.Item.Content)
	} else {
		sb := strings.Builder{}
		rr := []rune(d.Item.Content)
		sb.Grow(len(rr))
		for _, r := range rr {
			if r == '\n' {
				sb.WriteRune(r)
			} else {
				sb.WriteRune('*')
			}
		}
		d.content.SetText(sb.String())
	}
	d.Masked = !d.Masked
}

func (d *DetailView) Display(nt Note) {
	Debugf("Display %#v", nt)
	d.id.SetText(cast.ToString(nt.Id))
	d.name.SetText(nt.Name)
	d.desc.SetText(nt.Desc)
	d.content.SetText(nt.Content)
	d.ctime.SetText(nt.Ctime.FormatClassic())
	d.utime.SetText(nt.Utime.FormatClassic())
	d.Item = nt

	d.MaskNote()
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

	r := 0
	tb.SetCellSimple(r, 1, "Id:")
	tb.GetCell(r, 1).SetAlign(tview.AlignRight)
	iv.id = tview.NewTableCell("")
	tb.SetCell(r, 2, iv.id)

	r += 1
	tb.SetCellSimple(r, 1, "Name:")
	tb.GetCell(r, 1).SetAlign(tview.AlignRight)
	iv.name = tview.NewTableCell("")
	tb.SetCell(r, 2, iv.name)

	r += 1
	tb.SetCellSimple(r, 1, "Create Time:")
	tb.GetCell(r, 1).SetAlign(tview.AlignRight)
	iv.ctime = tview.NewTableCell("")
	tb.SetCell(r, 2, iv.ctime)

	r += 1
	tb.SetCellSimple(r, 1, "Update Time:")
	tb.GetCell(r, 1).SetAlign(tview.AlignRight)
	iv.utime = tview.NewTableCell("")
	tb.SetCell(r, 2, iv.utime)

	r += 1
	tb.SetCellSimple(r, 1, "Description:")
	tb.GetCell(r, 1).SetAlign(tview.AlignRight)
	iv.desc = tview.NewTableCell("")
	tb.SetCell(r, 2, iv.desc)

	infp := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tb, 0, 1, false)

	topFlex.AddItem(infp, 0, 1, false)

	iv.content = tview.NewTextView()
	iv.content.SetBorder(true).SetTitle(" Content ")
	iv.content.SetChangedFunc(func() { pocket.Draw() })

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 10, 1, true).
		AddItem(iv.content, 0, 1, false)

	iv.flex = mainFlex

	return iv
}

type ListView struct {
	flex    *tview.Flex
	bar     *tview.TextView  // top bar
	name    *tview.TableCell // searched name
	page    *tview.TableCell // at page (1-based)
	content *tview.Flex      // content of the list view, contains N note items

	pageNum int
}

type ListItemPrimitive struct {
	*tview.Table
	Note
}

func (l *ListView) ClearNotes() {
	l.content.Clear()
}

func (l *ListView) AddNote(it Note) {
	tb := tview.NewTable()
	lip := new(ListItemPrimitive)
	lip.Table = tb
	lip.Note = it
	tb.SetBorder(true)

	tb.SetCellSimple(0, 0, "Id:")
	tb.GetCell(0, 0).SetAlign(tview.AlignRight)
	idc := tview.NewTableCell(cast.ToString(it.Id))
	tb.SetCell(0, 1, idc)

	tb.SetCellSimple(1, 0, "Name:")
	tb.GetCell(1, 0).SetAlign(tview.AlignRight)
	namec := tview.NewTableCell(it.Name)
	tb.SetCell(1, 1, namec)

	tb.SetCellSimple(2, 0, "Description:")
	tb.GetCell(2, 0).SetAlign(tview.AlignRight)
	descc := tview.NewTableCell(it.Desc)
	tb.SetCell(2, 1, descc)

	tb.SetCellSimple(3, 0, "Updated At:")
	tb.GetCell(3, 0).SetAlign(tview.AlignRight)
	utimec := tview.NewTableCell(it.Utime.FormatClassic())
	tb.SetCell(3, 1, utimec)

	lip.SetFocusFunc(func() { lip.SetBorderColor(tcell.ColorYellow) })
	lip.SetBlurFunc(func() { lip.SetBorderColor(tcell.ColorWhite) })
	l.content.AddItem(lip, 6, 1, false)
}

func NewListView(pocket *Pocket, extendCap func(event *tcell.EventKey) (*tcell.EventKey, bool)) (iv *ListView) {
	topFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	iv = new(ListView)
	iv.pageNum = 1
	iv.bar = tview.NewTextView()
	iv.bar.SetText(`Notes`)
	iv.bar.SetBorder(true)
	iv.bar.SetTextAlign(tview.AlignCenter)
	topFlex.AddItem(iv.bar, 3, 1, false)

	tb := tview.NewTable()
	tb.SetBorder(true).SetTitle(" Searching Parameters ")

	tb.SetCellSimple(0, 1, "Name:")
	tb.GetCell(0, 1).SetAlign(tview.AlignRight)
	iv.name = tview.NewTableCell("").SetTextColor(tview.Styles.SecondaryTextColor)
	tb.SetCell(0, 2, iv.name)

	tb.SetCellSimple(1, 1, "Page:")
	tb.GetCell(1, 1).SetAlign(tview.AlignRight)
	iv.page = tview.NewTableCell("1").SetTextColor(tview.Styles.SecondaryTextColor)
	tb.SetCell(1, 2, iv.page)

	infp := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tb, 0, 1, false)

	topFlex.AddItem(infp, 0, 2, false)

	iv.content = tview.NewFlex().SetDirection(tview.FlexRow)
	iv.content.SetBorder(true).SetTitle(" Records ")

	iv.content.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyESC || evt.Rune() == 'q' || evt.Rune() == 'h' || evt.Key() == tcell.KeyLeft {
			pocket.SetFocus(pocket.ListPage.Options)
			return nil
		}

		if evt.Rune() == 'G' || (evt.Rune() == 'g' && evt.Modifiers() == tcell.ModShift) {
			n := iv.content.GetItemCount()
			pocket.SetFocus(iv.content.GetItem(n - 1))
			return nil
		}

		if evt.Key() == tcell.KeyEnter || evt.Rune() == 'l' || evt.Key() == tcell.KeyRight {
			j, ok := FindFocus(iv.content)
			if ok {
				itm := iv.content.GetItem(j)
				lip := itm.(*ListItemPrimitive)
				pocket.DetailPage.Display(lip.Note)
				pocket.Pages.SwitchToPage(PageDetail)
			}
			return nil
		}

		r := evt.Rune()
		if r == 'j' || r == 'k' || evt.Key() == tcell.KeyUp || evt.Key() == tcell.KeyDown {
			l := iv.content.GetItemCount()
			i, ok := FindFocus(iv.content)
			if ok {
				if r == 'j' || evt.Key() == tcell.KeyDown {
					if i < l-1 {
						pocket.SetFocus(iv.content.GetItem(i + 1))
					}
				} else if r == 'k' || evt.Key() == tcell.KeyUp {
					if i > 0 {
						pocket.SetFocus(iv.content.GetItem(i - 1))
					}
				}
			} else if l > 0 {
				pocket.SetFocus(iv.content.GetItem(0))
			}
		}

		if t, ok := extendCap(evt); ok {
			return t
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

// Form with grey background color, and only uses Shift+Tab to move cursor between inputs/buttons to support typing \t in textarea.
func NewForm(vimBased bool) *tview.Form {
	form := tview.NewForm()
	form.SetFieldBackgroundColor(tcell.ColorNavy.TrueColor())

	if vimBased {
		form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			Debugf(" %d %d - %d\n", ev.Key(), ev.Rune(), ev.Modifiers())

			if ev.Key() == tcell.KeyTab || ev.Key() == tcell.KeyBacktab || ev.Key() == tcell.KeyEnter {
				return ev
			}

			if ev.Rune() == 'j' || ev.Key() == tcell.KeyDown {
				return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
			}
			if ev.Rune() == 'k' || ev.Key() == tcell.KeyUp {
				return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
			}
			if ev.Rune() == 'q' {
				return tcell.NewEventKey(tcell.KeyESC, 0, tcell.ModNone)
			}
			return nil
		})
	}

	// form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
	// 	Debugf(" %d %d - %d\n", ev.Key(), ev.Rune(), ev.Modifiers())

	// 	// TODO: Not very intuitive
	// 	// Shift+Tab, always append at the end \t
	// 	if (ev.Key() == tcell.KeyBacktab) || (ev.Key() == tcell.KeyTAB && ev.Modifiers() == tcell.ModShift) {
	// 		ta, _, ok := FindFocusedTextArea(form)
	// 		if ok {
	// 			ta.SetText(ta.GetText()+"\t", true)
	// 		}
	// 		return nil
	// 	}
	// 	return ev
	// })

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

func PopDeleteNotePage(pocket *Pocket, it Note) {
	form := NewForm(false)
	close := func() { pocket.RemovePage(PageDelete) }
	confirm := func() {
		UIDeleteNote(pocket, it, func(err error) {
			pocket.ToPage(PageList)
			if err == nil {
				UIFetchNotes(pocket, 0)
			}
		})
		close()
	}
	form.AddTextView("", fmt.Sprintf("Deleting %v", it.Name), 40, 5, false, true)
	form.AddButton("Confirm", confirm)
	form.SetCancelFunc(close)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Delete Note ")

	popup := createPopup(pocket.Pages, form, 15, 40)
	pocket.Pages.AddPage(PageDelete, popup, true, true)
}

func UIDeleteNote(pocket *Pocket, nt Note, callback func(err error)) {
	go func() {
		err := StDeleteNote(nt)
		callback(err)
	}()
}

func UIEditNote(pocket *Pocket, note Note, callback func(err error)) {
	go func() {
		err := StEditNote(note)
		callback(err)
	}()
}

func UICreateNote(pocket *Pocket, note Note, callback func(note Note, err error)) {
	go func() {
		note, err := StCreateNote(note)
		callback(note, err)
	}()
}

func UIFetchNotes(pocket *Pocket, pageDelta int, then ...func()) {
	name := pocket.ListPage.name.Text
	page := pocket.ListPage.pageNum
	page += pageDelta

	go func() {
		items, err := StFetchNotes(page, PageLimit, name)
		if err == nil {
			pocket.QueueUpdateDraw(func() {
				prev := pocket.ListPage.pageNum
				if prev != page {
					if page > prev && len(items) < 1 { // displyaing next page, but the page is empty
						return
					}

					pocket.ListPage.pageNum = page
					pocket.ListPage.page.SetText(cast.ToString(page))
				}
				pocket.ListPage.ClearNotes()
				for _, it := range items {
					pocket.ListPage.AddNote(it)
				}

				for _, th := range then {
					th()
				}
			})
		}
	}()
}

func PopPasswordPage(pocket *Pocket) {
	form := NewForm(false)

	var tmppw string = ""
	form.AddPasswordField("Password (8-32 english characters [0-9a-zA-Z-_!.]):", tmppw, 32, '*',
		func(t string) { tmppw = t })

	ni := form.GetFormItem(0).(*tview.InputField)
	resetPasswordField := func(msg string) {
		ni.SetText("")
		PopMsg(pocket, func() { pocket.SetFocus(ni) }, msg)
	}

	ni.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		if e.Key() == tcell.KeyEnter {
			if err := ValidatePassword(tmppw); err != nil {
				resetPasswordField(err.Error())
				return nil
			}
			InitPassword(tmppw)
			ok, err := StCheckPassword()
			if err != nil {
				resetPasswordField(err.Error())
				return nil
			}
			if !ok {
				resetPasswordField("password incorrect")
				return nil
			}

			if err := StInitSchema(); err != nil {
				PopMsg(pocket, nil, err.Error())
				return nil
			}
			pocket.RemovePage(PagePassword)
			pocket.ToPage(PageList)
			UIFetchNotes(pocket, 0)
			return nil
		}

		return e
	})

	form.SetCancelFunc(pocket.Stop)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Enter Password ")

	popup := createPopup(pocket.Pages, form, 5, 90)
	pocket.Pages.AddPage(PagePassword, popup, true, true)
}

func ValidatePassword(s string) error {
	n := 0
	for _, c := range s {
		n += 1
		if c >= '0' && c <= '9' {
			continue
		}
		if c >= 'A' && c <= 'z' {
			continue
		}
		if c == '-' || c == '_' || c == '!' {
			continue
		}
		return fmt.Errorf("contains illegal character '%s'\ncan only contains 8-32 english characters\n[0-9a-zA-Z-_!.]", string(c))
	}

	if n < 8 {
		return errors.New("password too short")
	}
	return nil
}

func PopMsg(pocket *Pocket, onClosed func(), pat string, args ...any) {
	form := NewForm(false)
	close := func() {
		pocket.RemovePage(PageMsg)
		if onClosed != nil {
			onClosed()
		}
	}
	form.AddTextView("", fmt.Sprintf(pat, args...), 50, 5, false, true)
	form.AddButton("Okay", close)
	form.SetCancelFunc(close)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Message ")
	popup := createPopup(pocket.Pages, form, 15, 50)
	pocket.Pages.AddPage(PageMsg, popup, true, true)
	pocket.SetFocus(form.GetButton(0))
}

func PopExitPage(pocket *Pocket) {
	PopConfirmDialog(pocket, func() { pocket.Stop() }, "Exit Pocket?", 40, 15)
}

func PopConfirmDialog(pocket *Pocket, confirm func(), msg string, width int, height int) {
	form := NewForm(false)
	close := func() { pocket.RemovePage(PageConfirm) }
	form.AddTextView("", msg, width, height-10, false, true)
	if confirm == nil {
		confirm = close
	}
	form.AddButton("Confirm", confirm)
	form.SetCancelFunc(close)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Message ")
	popup := createPopup(pocket.Pages, form, height, width)
	pocket.Pages.AddPage(PageConfirm, popup, true, true)
	pocket.SetFocus(form.GetButton(0))
}

func VimEdit(pocket *Pocket, content string, onClose func(s string)) {
	pocket.Suspend(func() {
		dir := os.TempDir()
		os.MkdirAll(dir, 0755)

		f, err := os.CreateTemp(dir, "pocket-*") // 0600
		if err != nil {
			PopMsg(pocket, nil, "Failed to create temp file, %v", err)
			return
		}
		defer f.Close()
		defer os.Remove(f.Name())

		if _, err := f.WriteString(content); err != nil {
			PopMsg(pocket, nil, "Failed to write to temp file, %v", err)
			return
		}
		cmd := exec.Command("vim", f.Name())

		// for term control
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			PopMsg(pocket, nil, "Failed to launch vim, %v", err)
			return
		}

		out, err := os.ReadFile(f.Name())
		if err != nil {
			PopMsg(pocket, nil, "Failed to read from temp file, %v", err)
			return
		}

		if out != nil {
			onClose(string(out))
		} else {
			onClose("")
		}
	})
}
