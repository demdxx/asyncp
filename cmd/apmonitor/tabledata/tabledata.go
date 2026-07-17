// Package tabledata provides a thread-safe tview.TableContent implementation
// used by apmonitor to render the live metrics table.
package tabledata

import (
	"sync"

	"github.com/demdxx/gocast/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TableData is a thread-safe tview.TableContent implementation backed by a
// simple headers/rows/footer model.
type TableData struct {
	mx sync.RWMutex
	tview.TableContentReadOnly
	headers []string
	data    [][]string
	footer  []string
}

// NewTableData returns a new TableData populated with the given rows.
func NewTableData(data [][]string) *TableData {
	return &TableData{
		data: data,
	}
}

// SetHeaders replaces the table header row.
func (d *TableData) SetHeaders(headers []string) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.headers = headers
}

// SetData replaces all data rows.
func (d *TableData) SetData(data [][]string) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.data = data
}

// SetFooter replaces the table footer row.
func (d *TableData) SetFooter(footer []string) {
	d.mx.Lock()
	defer d.mx.Unlock()
	d.footer = footer
}

// GetCell implements tview.TableContent, returning the cell for the given
// row/column, accounting for the header (row 0) and footer (last row).
func (d *TableData) GetCell(row, column int) *tview.TableCell {
	d.mx.RLock()
	defer d.mx.RUnlock()

	cell := tview.NewTableCell("")
	if column >= len(d.headers) || column < 0 {
		return cell
	}

	columnName := d.headers[column]

	cell.
		SetAlign(gocast.IfThen(columnName == "task", tview.AlignLeft, tview.AlignRight)).
		SetTextColor(columnColorByName(columnName)).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorBlue).
			Foreground(tcell.ColorWhite))

	// Header
	if row == 0 {
		return cell.SetText(columnName).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
	}

	// Footer
	if row >= len(d.data)+1 {
		if column >= len(d.footer) {
			return cell
		}
		return cell.SetText(d.footer[column]).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false)
	}

	row-- // Adjust for header

	// Data rows
	if row >= len(d.data) || column >= len(d.data[row]) {
		return cell
	}

	return cell.SetText(d.data[row][column]).
		SetAttributes(columnAttrByName(columnName))
}

// GetRowCount returns the total number of rows, including the header and footer.
func (d *TableData) GetRowCount() int {
	d.mx.RLock()
	defer d.mx.RUnlock()
	return len(d.data) + 2 // +1 for header, +1 for footer
}

// GetColumnCount returns the number of columns, based on the headers set.
func (d *TableData) GetColumnCount() int {
	d.mx.RLock()
	defer d.mx.RUnlock()
	return len(d.headers)
}

func columnColorByName(name string) tcell.Color {
	switch name {
	case "task":
		return tcell.ColorYellow
	case "success":
		return tcell.ColorGreen
	case "skip":
		return tcell.ColorMaroon
	case "error":
		return tcell.ColorRed
	default:
		return tcell.ColorDefault
	}
}

func columnAttrByName(name string) tcell.AttrMask {
	switch name {
	case "task", "success", "skip", "error":
		return tcell.AttrBold
	}
	return tcell.AttrNone
}
