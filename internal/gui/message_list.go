// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// messageList keeps message actions local to the focused list. A global Delete
// shortcut would also intercept Delete while editing the search field.
type messageList struct {
	widget.List
	selectedID int
	onDelete   func()
	onSelected func(int)
}

func newMessageList(length func() int, create func() fyne.CanvasObject, update func(int, fyne.CanvasObject), onDelete func()) *messageList {
	l := &messageList{selectedID: -1, onDelete: onDelete}
	l.Length = length
	l.CreateItem = create
	l.UpdateItem = update
	// Fyne's native list items call the embedded List.Select directly, bypassing
	// our Select method. Track selection in the callback shared by both paths.
	l.List.OnSelected = func(id int) {
		l.selectedID = id
		if l.onSelected != nil {
			l.onSelected(id)
		}
	}
	l.List.OnUnselected = func(id int) {
		if l.selectedID == id {
			l.selectedID = -1
		}
	}
	// Extend before the List renderer is created so mouse/keyboard focus belongs
	// to this widget and reaches its TypedKey override.
	l.ExtendBaseWidget(l)
	return l
}
func (l *messageList) Select(id int) {
	if id < 0 || id >= l.Length() {
		return
	}
	l.List.Select(id)
}
func (l *messageList) UnselectAll() { l.selectedID = -1; l.List.UnselectAll() }

// Rows handle both mouse buttons themselves. Selection is therefore also the
// keyboard navigation position, rather than List's private pointer focus.
func (l *messageList) FocusGained() { l.Refresh() }
func (l *messageList) FocusLost()   { l.Refresh() }
func (l *messageList) TypedKey(event *fyne.KeyEvent) {
	if l.Length() == 0 {
		return
	}
	id := l.selectedID
	switch event.Name {
	case fyne.KeyDelete, fyne.KeyBackspace:
		if id >= 0 && l.onDelete != nil {
			l.onDelete()
		}
		return
	case fyne.KeyDown:
		if id < 0 {
			id = 0
		} else if id < l.Length()-1 {
			id++
		}
	case fyne.KeyUp:
		if id < 0 {
			id = 0
		} else if id > 0 {
			id--
		}
	case fyne.KeyHome:
		id = 0
	case fyne.KeyEnd:
		id = l.Length() - 1
	case fyne.KeySpace, fyne.KeyReturn:
		if id < 0 {
			id = 0
		}
	default:
		return
	}
	l.Select(id)
}

type messageRow struct {
	widget.BaseWidget
	subject, meta *widget.Label
	id            int
	onSelect      func(int)
	onContext     func(int, fyne.Position)
}

func newMessageRow(onSelect func(int), onContext func(int, fyne.Position)) *messageRow {
	r := &messageRow{id: -1, onSelect: onSelect, onContext: onContext}
	r.subject = widget.NewLabel("")
	r.subject.Truncation = fyne.TextTruncateEllipsis
	r.meta = widget.NewLabel("")
	r.meta.Truncation = fyne.TextTruncateEllipsis
	r.ExtendBaseWidget(r)
	return r
}
func (r *messageRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewVBox(r.subject, r.meta))
}
func (r *messageRow) Tapped(*fyne.PointEvent) {
	if r.id >= 0 && r.onSelect != nil {
		r.onSelect(r.id)
	}
}
func (r *messageRow) TappedSecondary(event *fyne.PointEvent) {
	if r.id >= 0 && r.onContext != nil {
		r.onContext(r.id, event.AbsolutePosition)
	}
}
