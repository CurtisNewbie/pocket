package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cast"
)

const (
	PagePassword      = "delete"
	PageList          = "list"
	PageDetail        = "detail"
	PageSearch        = "search"
	PageCreate        = "create"
	PageEdit          = "edit"
	PageDelete        = "delete"
	PageMsg           = "message"
	PageExit          = "exit"
	PageConfirm       = "confirm"
	PageAuthenticator = "authenticator"

	PageAuthenticatorPreview = "authenticator-preview"
	PageAuthenticatorEdit    = "authenticator-edit"
	PageAuthenticatorUri     = "authenticator-uri"
	PageAuthenticatorExport  = "authenticator-export"
	PageAuthenticatorEmpty   = "authenticator-empty"
	PageAuthenticatorTable   = "authenticator-table"

	PageLimit = 5

	LabelName    = "Name:"
	LabelDesc    = "Description:"
	LabelContent = "Content:"

	statusHint = "enter copy  e edit  d delete  u uri  x qr"
)

var (
	KeyDownEvt    = tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	KeyUpEvt      = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	KeyTabEvt     = tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	KeyBackTabEvt = tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
	KeyEscEvt     = tcell.NewEventKey(tcell.KeyESC, 0, tcell.ModNone)
)

type Pocket struct {
	*tview.Application
	Pages             *tview.Pages
	DetailPage        *DetailPage
	ListPage          *ListPage
	AuthenticatorPage *AuthenticatorPage
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
	lv := NewListView(pocket, func(lv *ListView, event *tcell.EventKey) (*tcell.EventKey, bool) {
		if event.Rune() == 'c' {
			PopCreateNotePage(pocket, func() { UIFetchNotes(pocket, 0) })
			return nil, true
		}

		if event.Rune() == 'n' {
			UIFetchNotes(pocket, 1, func() { lp.FocusOne(pocket) })
			return nil, true
		}

		if event.Rune() == 'N' && pocket.ListPage.GetPage() > 1 {
			UIFetchNotes(pocket, -1, func() { lp.FocusOne(pocket) })
			return nil, true
		}

		if event.Rune() == '/' {
			PopEditSearchPage(pocket)
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
			UIFetchNotes(pocket, 1)
		}).
		AddItem("Prev Page", "", 'N', func() {
			if pocket.ListPage.GetPage() > 1 {
				UIFetchNotes(pocket, -1)
			}
		}).
		AddItem("Authenticator", "", 'a', func() {
			pocket.ToPage(PageAuthenticator)
			pocket.AuthenticatorPage.Load(pocket)
			pocket.SetFocus(pocket.AuthenticatorPage.Options)
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

type AuthenticatorPage struct {
	*tview.Flex
	Options *tview.List

	bar     *inertTextView
	content *tview.Pages
	table   *tview.Table
	empty   *inertTextView
	status  *inertTextView

	entries     []TotpEntry
	refreshStop chan struct{}
}

func NewAuthenticatorPage(pocket *Pocket) *AuthenticatorPage {
	ap := &AuthenticatorPage{}

	ap.bar = newInertTextView()
	ap.bar.SetBorder(true).SetTitle(" Authenticator ")
	ap.bar.SetTextAlign(tview.AlignCenter)

	ap.table = tview.NewTable().SetSelectable(true, false)
	ap.table.SetBorder(true).SetTitle(" Entries ")
	ap.table.SetInputCapture(ap.tableInputCapture(pocket))

	ap.empty = newInertTextView()
	ap.empty.SetBorder(true).SetTitle(" Entries ")
	ap.empty.SetTextAlign(tview.AlignCenter)
	ap.empty.SetText("no entries yet\npress a to add from clipboard")

	ap.content = tview.NewPages()
	ap.content.AddPage(PageAuthenticatorEmpty, ap.empty, true, true)
	ap.content.AddPage(PageAuthenticatorTable, ap.table, true, false)

	ap.status = newInertTextView()
	ap.status.SetBorder(true)
	ap.status.SetTextAlign(tview.AlignCenter)
	ap.status.SetText(statusHint)

	content := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ap.bar, 3, 1, false).
		AddItem(ap.content, 0, 1, true).
		AddItem(ap.status, 3, 1, false)

	ap.Options = tview.NewList()
	ap.Options.SetBorder(true).SetTitle(" Options ")
	ap.Options.ShowSecondaryText(false)
	ap.Options.SetInputCapture(ap.optionsInputCapture(pocket))
	ap.Options.AddItem("Add from clipboard", "", 'a', func() {
		ap.AddFromClipboard(pocket)
	})
	ap.Options.AddItem("Select entry", "", 'l', func() {
		ap.focusTable(pocket)
	})
	ap.Options.AddItem("Copy URI", "", 'u', func() {
		ap.showSelectedUri(pocket)
	})
	ap.Options.AddItem("Export QR PNG", "", 'x', func() {
		ap.exportSelected(pocket)
	})
	ap.Options.AddItem("Back to notes", "", 'b', func() {
		ap.backToNotes(pocket)
	})
	ap.Options.AddItem("Exit", "", 'q', func() {
		PopExitPage(pocket)
	})

	ap.Flex = NewContentPlane(ap.Options, content)
	return ap
}

func (a *AuthenticatorPage) optionsInputCapture(pocket *Pocket) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		a.clearStatus()
		if ev.Rune() == 'j' {
			return KeyDownEvt
		}
		if ev.Rune() == 'k' {
			return KeyUpEvt
		}
		switch {
		case ev.Key() == tcell.KeyESC:
			a.backToNotes(pocket)
			return nil
		case ev.Rune() == 'l':
			a.focusTable(pocket)
			return nil
		case ev.Rune() == 'q':
			PopExitPage(pocket)
			return nil
		}
		return ev
	}
}

func (a *AuthenticatorPage) tableInputCapture(pocket *Pocket) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		a.clearStatus()
		switch {
		case ev.Key() == tcell.KeyESC || ev.Key() == tcell.KeyLeft || ev.Rune() == 'h' || ev.Rune() == 'q':
			pocket.SetFocus(a.Options)
			return nil
		case ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRight || ev.Rune() == 'l':
			a.copySelectedCode()
			return nil
		case ev.Rune() == 'j':
			return KeyDownEvt
		case ev.Rune() == 'k':
			return KeyUpEvt
		case ev.Rune() == 'G':
			n := a.table.GetRowCount()
			if n > 0 {
				a.table.Select(n-1, 0)
				a.table.ScrollToEnd()
			}
			return nil
		case ev.Rune() == 'a':
			a.AddFromClipboard(pocket)
			return nil
		case ev.Rune() == 'e':
			a.editSelectedEntry(pocket)
			return nil
		case ev.Rune() == 'd':
			a.confirmDeleteSelected(pocket)
			return nil
		case ev.Rune() == 'u':
			a.showSelectedUri(pocket)
			return nil
		case ev.Rune() == 'x':
			a.exportSelected(pocket)
			return nil
		}
		return ev
	}
}

func (a *AuthenticatorPage) backToNotes(pocket *Pocket) {
	a.stopRefresh()
	pocket.ToPage(PageList)
	pocket.SetFocus(pocket.ListPage.Options)
}

func (a *AuthenticatorPage) focusTable(pocket *Pocket) {
	if len(a.entries) == 0 {
		return
	}
	a.content.SwitchToPage(PageAuthenticatorTable)
	if row, _ := a.table.GetSelection(); row < 0 || row >= a.table.GetRowCount() {
		a.table.Select(0, 0)
	}
	pocket.SetFocus(a.table)
}

func (a *AuthenticatorPage) Load(pocket *Pocket) {
	go func() {
		entries, err := StFetchTotpEntries()
		pocket.QueueUpdateDraw(func() {
			if err != nil {
				PopMsg(pocket, func() { pocket.SetFocus(a.Options) }, "%s", err.Error())
				return
			}
			a.entries = entries
			a.render()
			a.startRefresh(pocket)
		})
	}()
}

func (a *AuthenticatorPage) render() {
	a.fillTable()
	if len(a.entries) == 0 {
		a.content.SwitchToPage(PageAuthenticatorEmpty)
		return
	}
	a.content.SwitchToPage(PageAuthenticatorTable)
	a.table.Select(0, 0)
	a.table.ScrollToBeginning()
}

// renderKeepingSelection rebuilds the rows after an edit or a delete and
// selects row, clamped to the last entry. The scroll offset is left untouched,
// so the refresh does not jump the table back to the top.
func (a *AuthenticatorPage) renderKeepingSelection(row int) {
	a.fillTable()
	if len(a.entries) == 0 {
		a.content.SwitchToPage(PageAuthenticatorEmpty)
		return
	}
	a.content.SwitchToPage(PageAuthenticatorTable)
	if row < 0 {
		row = 0
	}
	if last := a.table.GetRowCount() - 1; row > last {
		row = last
	}
	a.table.Select(row, 0)
}

func (a *AuthenticatorPage) fillTable() {
	a.bar.SetText(authHeaderText(len(a.entries)))
	now := time.Now()
	a.table.Clear()
	for i, e := range a.entries {
		issuer := e.Issuer
		if issuer == "" && e.Account == "" {
			issuer = "otpauth"
		}
		a.table.SetCell(i, 0, a.authenticatorCell(truncateRunes(issuer, 18), i))
		a.table.SetCell(i, 1, a.authenticatorCell(truncateRunes(e.Account, 26), i))
		a.table.SetCell(i, 2, a.authenticatorCell(e.Remark, i))
		a.table.SetCell(i, 3, a.authenticatorCell(codeCellText(e, now), i))
		a.table.SetCell(i, 4, a.authenticatorCell(secondsCellText(e.Period, now), i))
	}
}

// startRefresh runs a one-second tick while the page is open. A tick is
// skipped when the page is not the front page, which also covers modals.
func (a *AuthenticatorPage) startRefresh(pocket *Pocket) {
	a.stopRefresh()
	stop := make(chan struct{})
	a.refreshStop = stop
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				a.refresh(pocket)
			}
		}
	}()
}

func (a *AuthenticatorPage) stopRefresh() {
	if a.refreshStop != nil {
		close(a.refreshStop)
		a.refreshStop = nil
	}
}

func (a *AuthenticatorPage) refresh(pocket *Pocket) {
	pocket.QueueUpdateDraw(func() {
		if name, _ := pocket.Pages.GetFrontPage(); name != PageAuthenticator {
			return
		}
		a.refreshCodes(time.Now())
	})
}

// refreshCodes updates only the code and seconds cells that changed, so the
// selection and scroll offset are untouched.
func (a *AuthenticatorPage) refreshCodes(now time.Time) bool {
	changed := false
	for i, e := range a.entries {
		if setCellTextIfChanged(a.table, i, 3, codeCellText(e, now)) {
			changed = true
		}
		if setCellTextIfChanged(a.table, i, 4, secondsCellText(e.Period, now)) {
			changed = true
		}
	}
	return changed
}

func setCellTextIfChanged(table *tview.Table, row, col int, text string) bool {
	cell := table.GetCell(row, col)
	if cell == nil || cell.Text == text {
		return false
	}
	cell.SetText(text)
	return true
}

func codeCellText(e TotpEntry, now time.Time) string {
	code, err := GenerateTotpCode(e.Secret, e.Digits, e.Period, e.Algorithm, now)
	if err != nil {
		return "decrypt failed"
	}
	return formatCode(code)
}

// truncateRunes shortens s to n runes and marks the cut with an ellipsis, so a
// multi-byte value is never split mid-rune.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func formatCode(code string) string {
	switch len(code) {
	case 6:
		return code[:3] + " " + code[3:]
	case 8:
		return code[:4] + " " + code[4:]
	default:
		return code
	}
}

func secondsCellText(period int, now time.Time) string {
	if period <= 0 {
		return ""
	}
	return fmt.Sprintf("%ds", period-int(now.Unix()%int64(period)))
}

// selectedRow returns the selected table row and whether it indexes an entry.
func (a *AuthenticatorPage) selectedRow() (int, bool) {
	row, _ := a.table.GetSelection()
	if row < 0 || row >= len(a.entries) {
		return 0, false
	}
	return row, true
}

func (a *AuthenticatorPage) copySelectedCode() {
	row, ok := a.selectedRow()
	if !ok {
		return
	}
	e := a.entries[row]
	code, err := GenerateTotpCode(e.Secret, e.Digits, e.Period, e.Algorithm, time.Now())
	if err == nil {
		err = StWriteClipboard(code)
	}
	if err != nil {
		a.setStatus("couldn't copy code")
		return
	}
	a.setStatus(fmt.Sprintf("copied code for %s: %s", e.Issuer, e.Account))
}

func (a *AuthenticatorPage) showSelectedUri(pocket *Pocket) {
	row, ok := a.selectedRow()
	if !ok {
		a.setStatus("nothing to export, add an entry first")
		return
	}
	uri, err := BuildTotpUri(a.entries[row])
	if err != nil {
		a.setStatus("secret can't be decrypted")
		return
	}
	returnFocus := a.returnFocus()
	PopAuthenticatorUriPage(pocket, a, uri, func() { pocket.SetFocus(returnFocus) })
}

func (a *AuthenticatorPage) exportSelected(pocket *Pocket) {
	row, ok := a.selectedRow()
	if !ok {
		a.setStatus("nothing to export, add an entry first")
		return
	}
	e := a.entries[row]
	uri, err := BuildTotpUri(e)
	if err != nil {
		a.setStatus("secret can't be decrypted")
		return
	}
	path, err := defaultExportPath(e)
	if err != nil {
		a.setStatus("can't write to that folder")
		return
	}
	returnFocus := a.returnFocus()
	PopAuthenticatorExportPage(pocket, a, uri, path, func() { pocket.SetFocus(returnFocus) })
}

// returnFocus reports the pane that invoked an entry action, so closing its
// popup returns focus there instead of always moving to the table.
func (a *AuthenticatorPage) returnFocus() tview.Primitive {
	if a.table.HasFocus() {
		return a.table
	}
	return a.Options
}

func (a *AuthenticatorPage) editSelectedEntry(pocket *Pocket) {
	row, ok := a.selectedRow()
	if !ok {
		return
	}
	restoreTable := func() { pocket.SetFocus(a.table) }
	PopAuthenticatorEditPage(pocket, a.entries[row], row, restoreTable)
}

func (a *AuthenticatorPage) confirmDeleteSelected(pocket *Pocket) {
	row, ok := a.selectedRow()
	if !ok {
		return
	}
	e := a.entries[row]
	restoreTable := func() { pocket.SetFocus(a.table) }
	PopConfirmDialog(pocket, func() {
		pocket.RemovePage(PageConfirm)
		restoreTable()
		go func() {
			if err := StDeleteTotpEntry(e); err != nil {
				pocket.QueueUpdateDraw(func() {
					PopMsg(pocket, restoreTable, "%s", err.Error())
				})
				return
			}
			entries, err := StFetchTotpEntries()
			pocket.QueueUpdateDraw(func() {
				if err != nil {
					PopMsg(pocket, restoreTable, "%s", err.Error())
					return
				}
				a.applyEntries(pocket, entries, row, fmt.Sprintf("deleted %s: %s", e.Issuer, e.Account))
			})
		}()
	}, deleteConfirmText(e), 60, 15, restoreTable)
}

// applyEntries replaces the list after an edit or a delete and keeps the
// selection at row, which is the same entry after an edit and the neighbour
// that took the deleted entry's place after a delete.
func (a *AuthenticatorPage) applyEntries(pocket *Pocket, entries []TotpEntry, row int, status string) {
	a.entries = entries
	a.setStatus(status)
	a.renderKeepingSelection(row)
	if len(a.entries) == 0 {
		pocket.SetFocus(a.Options)
		return
	}
	pocket.SetFocus(a.table)
}

func (a *AuthenticatorPage) setStatus(msg string) {
	a.status.SetText(msg)
}

func (a *AuthenticatorPage) clearStatus() {
	a.status.SetText(statusHint)
}

func entryCountText(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

func authHeaderText(n int) string {
	return entryCountText(n) + " (refresh every 1s)"
}

func deleteConfirmText(e TotpEntry) string {
	msg := fmt.Sprintf("Deleting %s: %s", e.Issuer, e.Account)
	if e.Remark != "" {
		msg += fmt.Sprintf(" (%s)", e.Remark)
	}
	return msg
}

func (a *AuthenticatorPage) AddFromClipboard(pocket *Pocket) {
	var returnFocus tview.Primitive = a.Options
	if a.table.HasFocus() {
		returnFocus = a.table
	}
	restoreFocus := func() { pocket.SetFocus(returnFocus) }

	a.setStatus("reading clipboard...")
	go func() {
		parsed, err := ReadTotpFromClipboard()
		pocket.QueueUpdateDraw(func() {
			if err != nil {
				a.clearStatus()
				PopMsg(pocket, restoreFocus, "%s", importErrorMessage(err))
				return
			}
			a.clearStatus()
			PopAuthenticatorPreviewPage(pocket, parsed, restoreFocus)
		})
	}()
}

func importErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrHotpUnsupported):
		return "hotp isn't supported, totp only"
	case errors.Is(err, ErrSecretInvalid):
		return "secret isn't valid base32"
	case errors.Is(err, ErrParamInvalid):
		return strings.TrimPrefix(err.Error(), ErrParamInvalid.Error()+": ")
	case errors.Is(err, ErrClipboardEmpty):
		return "clipboard is empty"
	case errors.Is(err, ErrQrNotFound), errors.Is(err, ErrQrDecodeFailed):
		return "no qr code in clipboard image"
	case errors.Is(err, ErrQrNotTotpUri):
		return "qr code isn't a totp uri"
	default:
		return "clipboard isn't a qr image or otpauth:// uri"
	}
}

func PopAuthenticatorPreviewPage(pocket *Pocket, parsed ParsedTotp, onClose func()) {
	const (
		labelIssuer    = "Issuer:"
		labelAccount   = "Account:"
		labelDigits    = "Digits:"
		labelPeriod    = "Period:"
		labelAlgorithm = "Algorithm:"
		labelRemark    = "Remark:"
	)

	form := NewForm(false)
	form.SetBorder(true).SetTitle(" Preview Entry ")

	var issuerItem, accountItem *tview.InputField
	if parsed.Bare {
		issuerItem = tview.NewInputField().SetLabel(labelIssuer).SetText(parsed.Issuer)
		accountItem = tview.NewInputField().SetLabel(labelAccount).SetText(parsed.Account)
		form.AddFormItem(issuerItem)
		form.AddFormItem(accountItem)
	} else {
		form.AddTextView(labelIssuer, parsed.Issuer, 50, 1, false, false)
		form.AddTextView(labelAccount, parsed.Account, 50, 1, false, false)
	}
	form.AddTextView(labelDigits, cast.ToString(parsed.Digits), 50, 1, false, false)
	form.AddTextView(labelPeriod, cast.ToString(parsed.Period), 50, 1, false, false)
	form.AddTextView(labelAlgorithm, parsed.Algorithm, 50, 1, false, false)
	remarkItem := tview.NewInputField().SetLabel(labelRemark)
	form.AddFormItem(remarkItem)

	closePopup := func() {
		pocket.RemovePage(PageAuthenticatorPreview)
		if onClose != nil {
			onClose()
		}
	}

	save := func() {
		issuer := parsed.Issuer
		if issuerItem != nil {
			issuer = issuerItem.GetText()
		}
		account := parsed.Account
		if accountItem != nil {
			account = accountItem.GetText()
		}
		entry := TotpEntry{
			Issuer:    issuer,
			Account:   account,
			Remark:    remarkItem.GetText(),
			Secret:    parsed.Secret,
			Digits:    parsed.Digits,
			Period:    parsed.Period,
			Algorithm: parsed.Algorithm,
			Ctime:     Now(),
			Utime:     Now(),
		}
		go func() {
			_, err := StCreateTotpEntry(entry)
			pocket.QueueUpdateDraw(func() {
				pocket.RemovePage(PageAuthenticatorPreview)
				if err != nil {
					PopMsg(pocket, onClose, "%s", err.Error())
					return
				}
				if onClose != nil {
					onClose()
				}
				pocket.AuthenticatorPage.setStatus(fmt.Sprintf("saved %s: %s", entry.Issuer, entry.Account))
				pocket.AuthenticatorPage.Load(pocket)
			})
		}()
	}

	form.AddButton("Save", save)
	form.AddButton("Cancel", closePopup)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if _, button := form.GetFocusedItemIndex(); ev.Key() == tcell.KeyEnter && button < 0 {
			save()
			return nil
		}
		return ev
	})
	form.SetFocus(form.GetFormItemIndex(labelRemark))

	popup := createPopup(form, 14, 80)
	pocket.Pages.AddPage(PageAuthenticatorPreview, popup, true, true)
	pocket.SetFocus(form)
}

func PopAuthenticatorEditPage(pocket *Pocket, entry TotpEntry, row int, onClose func()) {
	const (
		labelAccount = "Account:"
		labelRemark  = "Remark:"
	)

	form := NewForm(false)
	form.SetBorder(true).SetTitle(" Edit Entry ")

	accountItem := tview.NewInputField().SetLabel(labelAccount).SetText(entry.Account)
	form.AddFormItem(accountItem)
	remarkItem := tview.NewInputField().SetLabel(labelRemark).SetText(entry.Remark)
	form.AddFormItem(remarkItem)

	closePopup := func() {
		pocket.RemovePage(PageAuthenticatorEdit)
		if onClose != nil {
			onClose()
		}
	}

	save := func() {
		entry.Account = accountItem.GetText()
		entry.Remark = remarkItem.GetText()
		entry.Utime = Now()
		go func() {
			if err := StUpdateTotpAccountAndRemark(entry); err != nil {
				pocket.QueueUpdateDraw(func() {
					pocket.RemovePage(PageAuthenticatorEdit)
					PopMsg(pocket, onClose, "%s", err.Error())
				})
				return
			}
			entries, err := StFetchTotpEntries()
			pocket.QueueUpdateDraw(func() {
				closePopup()
				if err != nil {
					PopMsg(pocket, onClose, "%s", err.Error())
					return
				}
				pocket.AuthenticatorPage.applyEntries(pocket, entries, row, "entry updated")
			})
		}()
	}

	form.AddButton("Save", save)
	form.AddButton("Cancel", closePopup)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if _, button := form.GetFocusedItemIndex(); ev.Key() == tcell.KeyEnter && button < 0 {
			save()
			return nil
		}
		return ev
	})
	form.SetFocus(form.GetFormItemIndex(labelAccount))

	popup := createPopup(form, 9, 70)
	pocket.Pages.AddPage(PageAuthenticatorEdit, popup, true, true)
	pocket.SetFocus(form)
}

func PopAuthenticatorUriPage(pocket *Pocket, page *AuthenticatorPage, uri string, onClose func()) {
	form := NewForm(false)
	form.SetBorder(true).SetTitle(" otpauth URI ")

	closePopup := func() {
		pocket.RemovePage(PageAuthenticatorUri)
		if onClose != nil {
			onClose()
		}
	}

	copyUri := func() {
		if err := StWriteClipboard(uri); err != nil {
			page.setStatus("couldn't copy uri")
			return
		}
		closePopup()
		page.setStatus("copied uri")
	}

	uriItem := tview.NewInputField().SetLabel("URI:").SetText(uri)
	uriItem.SetDisabled(true)
	form.AddFormItem(uriItem)
	form.AddButton("Copy", copyUri)
	form.AddButton("Close", closePopup)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Rune() == 'y' {
			copyUri()
			return nil
		}
		return ev
	})

	popup := createPopup(form, 9, 120)
	pocket.Pages.AddPage(PageAuthenticatorUri, popup, true, true)
	pocket.SetFocus(form.GetButton(0))
}

func PopAuthenticatorExportPage(pocket *Pocket, page *AuthenticatorPage, uri string, path string, onClose func()) {
	const (
		warningText = "this png contains your secret in plaintext"
		labelPath   = "Path:"
	)

	form := NewForm(false)
	form.SetBorder(true).SetTitle(" Export QR PNG ")

	closePopup := func() {
		pocket.RemovePage(PageAuthenticatorExport)
		if onClose != nil {
			onClose()
		}
	}

	form.AddFormItem(tview.NewTextView().SetText(warningText).SetSize(1, 0))
	pathItem := tview.NewInputField().SetLabel(labelPath).SetText(path)
	form.AddFormItem(pathItem)
	errItem := tview.NewTextView().SetSize(1, 0)
	form.AddFormItem(errItem)

	export := func() {
		target := strings.TrimSpace(pathItem.GetText())
		err := EncodeQrPng(uri, target)
		if err == nil {
			closePopup()
			page.setStatus(fmt.Sprintf("exported to %s (plaintext)", target))
			return
		}
		if errors.Is(err, ErrFileExists) {
			errItem.SetText(fmt.Sprintf("file already exists: %s, choose another name", filepath.Base(target)))
			return
		}
		if errors.Is(err, ErrQrEncodeFailed) {
			errItem.SetText("couldn't encode the qr code")
			return
		}
		errItem.SetText("can't write to that folder")
	}

	form.AddButton("Export", export)
	form.AddButton("Cancel", closePopup)
	form.SetCancelFunc(closePopup)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if _, button := form.GetFocusedItemIndex(); ev.Key() == tcell.KeyEnter && button < 0 {
			export()
			return nil
		}
		return ev
	})

	popup := createPopup(form, 13, 90)
	pocket.Pages.AddPage(PageAuthenticatorExport, popup, true, true)
	pocket.SetFocus(pathItem)
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
			liv.pageNum = 1
			liv.page.SetText("1")
			closePopup()
			if prevName != tmpName {
				UIFetchNotes(pocket, 0)
			}
			return nil
		}
		return evt
	})

	popup := createPopup(form, 5, 110)
	pages.AddPage(PageSearch, popup, true, true)
}

func PopEditNotePage(pocket *Pocket, it Note) {
	closePopup := func() {
		pocket.ToPage(PageDetail)
		pocket.DetailPage.Display(it)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm(true)
	form.AddTextArea(LabelName, it.Name, 100, 2, 30, nil)
	form.AddTextArea(LabelDesc, it.Desc, 100, 5, 250, nil)
	form.AddTextArea(LabelContent, it.Content, 100, 20, 10000, nil)

	newInputCap := func(t *tview.TextArea) func(event *tcell.EventKey) *tcell.EventKey {
		return func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				VimEdit(pocket, t.GetText(), func(s string) { t.SetText(strings.TrimSpace(s), true) })
				return nil
			}
			return event
		}
	}

	// this is so ugly :(, but it works
	ni := form.GetFormItemByLabel(LabelName).(*tview.TextArea)
	di := form.GetFormItemByLabel(LabelDesc).(*tview.TextArea)
	ci := form.GetFormItemByLabel(LabelContent).(*tview.TextArea)

	ni.SetInputCapture(newInputCap(ni))
	di.SetInputCapture(newInputCap(di))
	ci.SetInputCapture(newInputCap(ci))

	confirm := func() {
		ni := Note{
			Id:      it.Id,
			Name:    ni.GetText(),
			Desc:    di.GetText(),
			Content: ci.GetText(),
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
		if ni.GetText() == it.Name && di.GetText() == it.Desc && ci.GetText() == it.Content {
			closePopup()
			return
		}
		PopConfirmDialog(pocket, closePopup, "Close Dialog?", 50, 15)
	})
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Edit Note (vim-based) ")

	popup := createPopup(form, 35, 100)
	pocket.Pages.AddPage(PageEdit, popup, true, true)
}

func PopCreateNotePage(pocket *Pocket, onConfirm func()) {
	closePopup := func() {
		pocket.ToPage(PageList)
		pocket.RemovePage(PageCreate)
	}

	form := NewForm(true)
	form.AddTextArea(LabelName, "", 100, 2, 30, nil)
	form.AddTextArea(LabelDesc, "", 100, 5, 250, nil)
	form.AddTextArea(LabelContent, "", 100, 20, 10000, nil)

	newInputCap := func(t *tview.TextArea) func(event *tcell.EventKey) *tcell.EventKey {
		return func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				VimEdit(pocket, t.GetText(), func(s string) { t.SetText(strings.TrimSpace(s), true) })
				return nil
			}
			return event
		}
	}

	// this is so ugly :(, but it works
	ni := form.GetFormItemByLabel(LabelName).(*tview.TextArea)
	di := form.GetFormItemByLabel(LabelDesc).(*tview.TextArea)
	ci := form.GetFormItemByLabel(LabelContent).(*tview.TextArea)

	ni.SetInputCapture(newInputCap(ni))
	di.SetInputCapture(newInputCap(di))
	ci.SetInputCapture(newInputCap(ci))

	confirm := func() {
		ctime := Now()
		note := Note{
			Name:    ni.GetText(),
			Desc:    di.GetText(),
			Content: ci.GetText(),
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
		if ni.GetText() == "" && di.GetText() == "" && ci.GetText() == "" {
			closePopup()
			return
		}
		PopConfirmDialog(pocket, closePopup, "Close Dialog?", 50, 15)
	})
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Create Note (vim-based) ")

	popup := createPopup(form, 35, 120)
	pocket.Pages.AddPage(PageCreate, popup, true, true)
}

func createPopup(form *tview.Form, height int, width int) tview.Primitive {
	ignoreDialogFieldClicks(form)
	modal := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
	return newMouseShield(modal, form)
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

	authenticatorPage := NewAuthenticatorPage(pocket)
	pocket.AuthenticatorPage = authenticatorPage

	pages.AddPage(PageDetail, detailPage, true, true)
	pages.AddPage(PageList, listPage, true, true)
	pages.AddPage(PageAuthenticator, authenticatorPage, true, false)
	PopPasswordPage(pocket)
	app.SetRoot(pages, true)
	app.EnableMouse(true)

	return pocket
}

func NewContentPlane(options tview.Primitive, content tview.Primitive) *tview.Flex {
	ctnp := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(options, 30, 1, true).
		AddItem(content, 0, 4, false)

	ver := newInertTextView()
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
			return KeyDownEvt
		}
		if event.Rune() == 'k' {
			return KeyUpEvt
		}
		if ev, ok := extendedCap(event); ok {
			return ev
		}
		return event
	})
	return l
}

type DetailView struct {
	flex *tview.Flex

	bar     *inertTextView
	id      *tview.TableCell
	name    *tview.TableCell
	ctime   *tview.TableCell
	utime   *tview.TableCell
	desc    *tview.TableCell
	content *inertTextView

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
	iv.bar = newInertTextView()
	iv.bar.SetBorder(true)
	iv.bar.SetText(" ")
	iv.bar.SetTextAlign(tview.AlignCenter)
	topFlex.AddItem(iv.bar, 3, 1, false)

	tb := newInertTable()
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

	iv.content = newInertTextView()
	iv.content.SetBorder(true).SetTitle(" Content ")
	iv.content.SetChangedFunc(func() { pocket.Draw() })

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 10, 1, true).
		AddItem(iv.content, 0, 1, false)

	iv.flex = mainFlex

	return iv
}

type ListView struct {
	flex     *tview.Flex
	bar      *inertTextView   // top bar
	name     *tview.TableCell // searched name
	page     *tview.TableCell // at page (1-based)
	total    *tview.TableCell // total
	content  *tview.Flex      // content of the list view, contains N note items
	openNote func(Note)       // opens a note, set when the view is built

	pageNum int
}

type ListItemPrimitive struct {
	*tview.Table
	Note
	onOpen func()
}

func (l *ListView) ClearNotes() {
	l.content.Clear()
}

func (l *ListView) AddNote(it Note) {
	tb := tview.NewTable()
	lip := new(ListItemPrimitive)
	lip.Table = tb
	lip.Note = it
	if l.openNote != nil {
		lip.onOpen = func() { l.openNote(it) }
	}
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

func NewListView(pocket *Pocket, extendCap func(lv *ListView, event *tcell.EventKey) (*tcell.EventKey, bool)) (iv *ListView) {
	topFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	iv = new(ListView)
	iv.pageNum = 1
	iv.openNote = func(it Note) { openNote(pocket, it) }
	iv.bar = newInertTextView()
	iv.bar.SetText(`Notes`)
	iv.bar.SetBorder(true)
	iv.bar.SetTextAlign(tview.AlignCenter)
	topFlex.AddItem(iv.bar, 3, 1, false)

	tb := newInertTable()
	tb.SetBorder(true).SetTitle(" Searching Parameters ")

	tb.SetCellSimple(0, 1, "Name:")
	tb.GetCell(0, 1).SetAlign(tview.AlignRight)
	iv.name = tview.NewTableCell("").SetTextColor(tview.Styles.SecondaryTextColor)
	tb.SetCell(0, 2, iv.name)

	tb.SetCellSimple(1, 1, "Page:")
	tb.GetCell(1, 1).SetAlign(tview.AlignRight)
	iv.page = tview.NewTableCell("1").SetTextColor(tview.Styles.SecondaryTextColor)
	tb.SetCell(1, 2, iv.page)

	tb.SetCellSimple(2, 1, "Total:")
	tb.GetCell(2, 1).SetAlign(tview.AlignRight)
	iv.total = tview.NewTableCell("0").SetTextColor(tview.Styles.SecondaryTextColor)
	tb.SetCell(2, 2, iv.total)

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
				openNote(pocket, lip.Note)
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
			return nil
		}

		if t, ok := extendCap(iv, evt); ok {
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

			if ev.Key() == tcell.KeyTab || ev.Key() == tcell.KeyBacktab || ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEsc {
				return ev
			}

			if ev.Rune() == 'j' || ev.Key() == tcell.KeyDown {
				return KeyTabEvt
			}
			if ev.Rune() == 'k' || ev.Key() == tcell.KeyUp {
				return KeyBackTabEvt
			}
			if ev.Rune() == 'q' {
				return KeyEscEvt
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

	popup := createPopup(form, 15, 40)
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
		total, items, err := StFetchNotes(page, PageLimit, name)
		if err == nil {
			pocket.QueueUpdateDraw(func() {
				pocket.ListPage.total.SetText(cast.ToString(total))

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

	popup := createPopup(form, 5, 90)
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
	popup := createPopup(form, 15, 50)
	pocket.Pages.AddPage(PageMsg, popup, true, true)
	pocket.SetFocus(form.GetButton(0))
}

func PopExitPage(pocket *Pocket) {
	PopConfirmDialog(pocket, func() { pocket.Stop() }, "Exit Pocket?", 40, 15)
}

func PopConfirmDialog(pocket *Pocket, confirm func(), msg string, width int, height int, onClosed ...func()) {
	form := NewForm(false)
	close := func() {
		pocket.RemovePage(PageConfirm)
		for _, f := range onClosed {
			f()
		}
	}
	form.AddTextView("", msg, width, height-10, false, true)
	if confirm == nil {
		confirm = close
	}
	form.AddButton("Confirm", confirm)
	form.SetCancelFunc(close)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true).SetTitle(" Message ")
	popup := createPopup(form, height, width)
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
