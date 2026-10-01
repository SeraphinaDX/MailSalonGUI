// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"image/color"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// messageList keeps message actions local to the focused list. A global Delete
// shortcut would also intercept Delete while editing the search field.
type messageList struct {
	widget.List
	selectedID         int
	anchor             int
	selection          map[int]bool
	revision           uint64
	updatingSelection  bool
	onSelectionChanged func()
	modifiers          func() fyne.KeyModifier
	onShortcut         func(fyne.Shortcut)
	onDelete           func()
	onSelected         func(int)
	// Resolve lazily: GLFW is initialized only after a desktop window opens.
	keypadDeleteScanCode func() int
}

func newMessageList(length func() int, create func() fyne.CanvasObject, update func(int, fyne.CanvasObject), onDelete func()) *messageList {
	l := &messageList{selectedID: -1, anchor: -1, selection: map[int]bool{}, modifiers: currentModifiers, onDelete: onDelete, keypadDeleteScanCode: keypadDeleteScanCode}
	l.Length = length
	l.CreateItem = create
	l.UpdateItem = update
	// Fyne's native list items call the embedded List.Select directly, bypassing
	// our Select method. Track selection in the callback shared by both paths.
	l.List.OnSelected = func(id int) {
		l.selectedID = id
		if !l.updatingSelection {
			l.selection = map[int]bool{id: true}
			l.anchor = id
			l.selectionChanged()
		}
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

// The embedded List tracks only the active row for scrolling and focus. The
// separate selection set controls highlights and bulk-operation targets.
func currentModifiers() fyne.KeyModifier {
	if app := fyne.CurrentApp(); app != nil {
		if driver, ok := app.Driver().(desktop.Driver); ok {
			return driver.CurrentKeyModifiers()
		}
	}
	return 0
}
func (l *messageList) SelectedIDs() []int {
	ids := make([]int, 0, len(l.selection))
	for id := range l.selection {
		if id >= 0 && id < l.Length() {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
func (l *messageList) selectionChanged() {
	l.revision++
	l.Refresh()
	if l.onSelectionChanged != nil {
		l.onSelectionChanged()
	}
}
func (l *messageList) focusSelection(id int) {
	l.updatingSelection = true
	if id < 0 {
		l.selectedID = -1
		l.List.UnselectAll()
	} else {
		l.selectedID = id
		l.List.Select(id)
	}
	l.updatingSelection = false
	l.selectionChanged()
}
func (l *messageList) Select(id int) { l.SelectWithModifiers(id, 0) }
func (l *messageList) SelectWithModifiers(id int, modifiers fyne.KeyModifier) {
	if id < 0 || id >= l.Length() {
		return
	}
	switch {
	case modifiers&fyne.KeyModifierShift != 0:
		if l.anchor < 0 {
			l.anchor = id
		}
		if modifiers&fyne.KeyModifierControl == 0 {
			l.selection = map[int]bool{}
		}
		start, end := l.anchor, id
		if start > end {
			start, end = end, start
		}
		for i := start; i <= end; i++ {
			l.selection[i] = true
		}
	case modifiers&(fyne.KeyModifierControl|fyne.KeyModifierSuper) != 0:
		l.anchor = id
		if l.selection[id] {
			delete(l.selection, id)
			// Keep the preview on a selected message when the active row is removed.
			if id == l.selectedID {
				ids := l.SelectedIDs()
				id = -1
				if len(ids) > 0 {
					id = ids[0]
				}
			} else {
				id = l.selectedID
			}
		} else {
			l.selection[id] = true
		}
	default:
		l.selection = map[int]bool{id: true}
		l.anchor = id
	}
	l.focusSelection(id)
}
func (l *messageList) SelectAll() {
	if l.Length() == 0 {
		return
	}
	l.selection = map[int]bool{}
	for i := 0; i < l.Length(); i++ {
		l.selection[i] = true
	}
	id := l.selectedID
	if id < 0 {
		id = 0
	}
	l.anchor = id
	l.focusSelection(id)
}
func (l *messageList) UnselectAll() {
	l.selection = map[int]bool{}
	l.anchor = -1
	l.focusSelection(-1)
}
func (l *messageList) TypedShortcut(shortcut fyne.Shortcut) {
	switch s := shortcut.(type) {
	case *fyne.ShortcutSelectAll:
		l.SelectAll()
		return
	case *desktop.CustomShortcut:
		if s.KeyName == fyne.KeySpace && s.Modifier == fyne.KeyModifierControl {
			l.SelectWithModifiers(l.selectedID, s.Modifier)
			return
		}
	}
	// Fyne sends shortcuts to the focused widget before canvas shortcuts.
	// Forward unhandled ones so Ctrl+N/R/F still work from the message list.
	if l.onShortcut != nil {
		l.onShortcut(shortcut)
	}
}

// Rows handle both mouse buttons themselves. Selection is therefore also the
// keyboard navigation position, rather than List's private pointer focus.
func (l *messageList) FocusGained() { l.Refresh() }
func (l *messageList) FocusLost()   { l.Refresh() }
func (l *messageList) TypedKey(event *fyne.KeyEvent) {
	if l.Length() == 0 {
		return
	}
	id := l.selectedID
	if l.isDeleteKey(event) {
		if id >= 0 && l.onDelete != nil {
			l.onDelete()
		}
		return
	}
	switch event.Name {
	case fyne.KeyEscape:
		l.UnselectAll()
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
	l.SelectWithModifiers(id, l.modifiers())
}

func (l *messageList) isDeleteKey(event *fyne.KeyEvent) bool {
	if event.Name == fyne.KeyDelete || event.Name == fyne.KeyBackspace {
		return true
	}
	// Fyne 2.7 does not map GLFW's KP_DECIMAL key. It can arrive as an
	// unknown key, period or comma depending on keyboard layout and platform.
	// Match the physical keypad key, not an ordinary punctuation key.
	switch event.Name {
	case fyne.KeyUnknown, fyne.KeyPeriod, fyne.KeyComma:
		if event.Physical.ScanCode > 0 && l.keypadDeleteScanCode != nil {
			scanCode := l.keypadDeleteScanCode()
			return scanCode > 0 && event.Physical.ScanCode == scanCode
		}
	}
	return false
}

type messageRow struct {
	widget.BaseWidget
	subject, meta *widget.Label
	id            int
	onSelect      func(int)
	onContext     func(int, fyne.Position)
	background    *canvas.Rectangle
	selected      bool
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
	r.background = canvas.NewRectangle(color.Transparent)
	if r.selected {
		r.background.FillColor = theme.Color(theme.ColorNameSelection)
	}
	return widget.NewSimpleRenderer(container.NewStack(r.background, container.NewVBox(r.subject, r.meta)))
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

func (r *messageRow) Refresh() {
	if r.background != nil {
		r.background.FillColor = color.Transparent
		if r.selected {
			r.background.FillColor = theme.Color(theme.ColorNameSelection)
		}
		r.background.Refresh()
	}
	r.BaseWidget.Refresh()
}
