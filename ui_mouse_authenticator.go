package main

import "github.com/rivo/tview"

// authenticatorCell builds one cell of an authenticator table row. A left click
// on the cell selects its row and copies that entry's code, which is what Enter
// on the focused row does. Returning true suppresses the table's native
// selection of the clicked cell, so the row selection stands.
func (a *AuthenticatorPage) authenticatorCell(text string, row int) *tview.TableCell {
	return tview.NewTableCell(text).SetExpansion(1).SetClickedFunc(func() bool {
		a.table.Select(row, 0)
		a.copySelectedCode()
		return true
	})
}
