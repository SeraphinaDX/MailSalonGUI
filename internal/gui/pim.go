// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"net/mail"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

type collectionView struct {
	app             *App
	contacts        bool
	content         fyne.CanvasObject
	chooser         *widget.Select
	list            *widget.List
	search          *widget.Entry
	preview         *widget.Label
	info            *widget.Label
	collections     []config.Collection
	current         int
	items, filtered []pim.Item
	selected        int
	generation      uint64
	calendar        *calendarView
}

func (a *App) visibleCollections(contacts bool, account int) []config.Collection {
	var out []config.Collection
	name := a.cfg.Accounts[account].Name
	for _, c := range a.cfg.Collections {
		if pim.Contacts(c) == contacts && (c.Account == "" || c.Account == name) {
			out = append(out, c)
		}
	}
	return out
}
func (a *App) newCollectionView(contacts bool) *collectionView {
	v := &collectionView{app: a, contacts: contacts, current: -1, selected: -1}
	v.preview = widget.NewLabel("")
	v.preview.Wrapping = fyne.TextWrapWord
	v.info = widget.NewLabel("")
	v.info.Wrapping = fyne.TextWrapWord
	v.chooser = widget.NewSelect(nil, func(name string) {
		for i, c := range v.collections {
			if c.Name == name {
				v.current = i
				v.load()
				break
			}
		}
	})
	v.search = widget.NewEntry()
	v.search.SetPlaceHolder("Search names and details")
	v.search.OnChanged = func(string) { v.filter() }
	v.list = widget.NewList(func() int { return len(v.filtered) }, func() fyne.CanvasObject { l := widget.NewLabel(""); l.Truncation = fyne.TextTruncateEllipsis; return l }, func(id int, o fyne.CanvasObject) {
		if id < 0 || id >= len(v.filtered) {
			return
		}
		i := v.filtered[id]
		detail := i.Email
		if !v.contacts {
			detail = i.Start
		}
		o.(*widget.Label).SetText(i.Title + "  ·  " + detail)
	})
	v.list.OnSelected = func(id int) {
		if id < 0 || id >= len(v.filtered) {
			return
		}
		v.selected = id
		i := v.filtered[id]
		text := i.Title + "\n\nEmail: " + i.Email + "\nPhone: " + i.Phone + "\n\n" + i.Notes
		if !contacts {
			text = i.Title + "\n\nStart: " + i.Start + "\nEnd / duration: " + i.End + "\nTimezone: " + i.Zone + "\nLocation: " + i.Location + "\n\n" + i.Notes
			if i.Recurring {
				text += "\n\nRecurring event — rules are preserved in its source."
			}
		}
		v.preview.SetText(text)
	}
	tools := container.NewHBox(widget.NewButtonWithIcon("New", theme.ContentAddIcon(), v.create), widget.NewButton("Edit source", v.edit), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), v.remove), widget.NewButton("Reload", v.load))
	if contacts {
		tools.Add(widget.NewButton("Write email", func() {
			if item, ok := v.selectedItem(); ok {
				addresses := pim.ContactAddresses(item)
				if len(addresses) > 0 {
					a.compose(mimeutil.Draft{To: addresses[0].String()}, a.account, "")
				}
			}
		}))
	}
	panes := container.NewHSplit(container.NewBorder(container.NewVBox(v.chooser, v.search), v.info, nil, nil, v.list), container.NewBorder(tools, nil, nil, nil, container.NewVScroll(v.preview)))
	panes.Offset = 0.4
	v.content = panes
	if !contacts {
		v.calendar = v.newCalendarView()
		v.content = v.calendar.content
	}
	return v
}
func (v *collectionView) reloadCollections() {
	if v.calendar != nil {
		v.calendar.clear("Loading calendars…")
	}
	v.generation++
	previous := ""
	if v.current >= 0 && v.current < len(v.collections) {
		previous = v.collections[v.current].Name
	}
	v.collections = v.app.visibleCollections(v.contacts, v.app.account)
	v.current = -1
	v.items = nil
	v.filtered = nil
	v.selected = -1
	v.list.UnselectAll()
	v.list.Refresh()
	names := []string{}
	index := 0
	for i, c := range v.collections {
		names = append(names, c.Name)
		if c.Name == previous {
			index = i
		}
	}
	v.chooser.Options = names
	v.chooser.ClearSelected()
	v.chooser.Refresh()
	if len(names) == 0 {
		v.chooser.Disable()
		kind := "contacts"
		protocol := "carddav"
		if !v.contacts {
			kind = "calendar"
			protocol = "caldav"
		}
		v.info.SetText("No " + kind + " collections configured")
		v.preview.SetText("Add a collection in Settings to use this view:\n\n[[collections]]\nname = \"Personal " + kind + "\"\naccount = \"" + v.app.cfg.Accounts[v.app.account].Name + "\"\nprotocol = \"" + protocol + "\"\nlocal_dir = \"~/PIM/" + kind + "\"\n\nConfigure MailSalonSync to sync the same directory.")
		if v.calendar != nil {
			v.calendar.clear(v.preview.Text)
		}
		return
	}
	v.chooser.Enable()
	v.chooser.SetSelectedIndex(index)
}
func (v *collectionView) load() {
	if v.current < 0 || v.current >= len(v.collections) {
		return
	}
	c := v.collections[v.current]
	v.generation++
	gen := v.generation
	v.items = nil
	v.filtered = nil
	v.selected = -1
	v.list.UnselectAll()
	v.list.Refresh()
	v.info.SetText("Loading…")
	v.preview.SetText("Choose an item")
	if v.calendar != nil {
		v.calendar.clear("Loading events…")
	}
	go func() {
		items, err := pim.Load(c)
		v.app.post(func() {
			if gen != v.generation {
				return
			}
			if err != nil {
				v.info.SetText("Unable to read collection")
				v.app.fail(err)
				if v.calendar != nil {
					v.calendar.clear("Unable to read calendar: " + err.Error())
				}
				return
			}
			v.items = items
			v.filter()
		})
	}()
}
func (v *collectionView) filter() {
	q := strings.ToLower(v.search.Text)
	v.filtered = nil
	v.selected = -1
	v.list.UnselectAll()
	v.preview.SetText("Choose an item")
	for _, i := range v.items {
		if q == "" || strings.Contains(strings.ToLower(i.Title+" "+i.Email+" "+i.Phone+" "+i.Start+" "+i.Location+" "+i.Notes), q) {
			v.filtered = append(v.filtered, i)
		}
	}
	v.list.Refresh()
	v.info.SetText(fmt.Sprintf("%d items · %d shown", len(v.items), len(v.filtered)))
	if v.calendar != nil {
		v.calendar.render()
	}
}
func (v *collectionView) selectedItem() (pim.Item, bool) {
	if v.selected < 0 || v.selected >= len(v.filtered) {
		return pim.Item{}, false
	}
	return v.filtered[v.selected], true
}
func (v *collectionView) collection() (config.Collection, bool) {
	if v.current < 0 || v.current >= len(v.collections) {
		return config.Collection{}, false
	}
	return v.collections[v.current], true
}
func (v *collectionView) create() {
	c, ok := v.collection()
	if !ok {
		return
	}
	w := v.app.Fyne.NewWindow("New item — " + c.Name)
	w.Resize(fyne.NewSize(600, 440))
	labels := []string{"Name", "Email", "Phone"}
	if !v.contacts {
		labels = []string{"Title", "Start", "End (exclusive)", "Timezone", "Location"}
	}
	fields := []*widget.Entry{}
	form := widget.NewForm()
	for _, name := range labels {
		e := widget.NewEntry()
		fields = append(fields, e)
		form.Append(name, e)
	}
	if !v.contacts {
		fields[1].SetPlaceHolder("2026-10-01T10:00:00 or 2026-10-01")
		fields[2].SetPlaceHolder("2026-10-01T11:00:00 or 2026-10-02")
		fields[3].SetPlaceHolder("America/Toronto")
		if v.calendar != nil {
			day := v.calendar.cursor
			fields[1].SetText(day.Format("2006-01-02") + "T09:00:00")
			fields[2].SetText(day.Format("2006-01-02") + "T10:00:00")
			fields[3].SetText(v.calendar.zone.String())
		}
	}
	var save *widget.Button
	save = widget.NewButton("Save", func() {
		values := []string{}
		for _, f := range fields {
			values = append(values, f.Text)
		}
		data, uid, err := pim.New(c, values)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		save.Disable()
		go func() {
			err := pim.Save(c, nil, data, uid)
			v.app.post(func() {
				save.Enable()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				w.Close()
				v.reloadCollections()
			})
		}()
	})
	w.SetContent(container.NewBorder(nil, container.NewHBox(save, widget.NewButton("Cancel", w.Close)), nil, nil, form))
	showWindow(w)
}

func (v *collectionView) edit() {
	item, ok := v.selectedItem()
	if !ok {
		return
	}
	c, ok := v.collection()
	if !ok {
		return
	}
	w := v.app.Fyne.NewWindow("Edit source — " + item.Title)
	w.Resize(fyne.NewSize(850, 650))
	editor := widget.NewMultiLineEntry()
	editor.Wrapping = fyne.TextWrapOff
	editor.SetText(string(item.Data))
	busy := false
	var save *widget.Button
	save = widget.NewButton("Save", func() {
		if busy {
			return
		}
		data := []byte(editor.Text)
		if _, err := pim.Parse(c, data); err != nil {
			dialog.ShowError(err, w)
			return
		}
		busy = true
		save.Disable()
		editor.Disable()
		go func() {
			err := pim.Save(c, &item, data, item.UID)
			v.app.post(func() {
				busy = false
				save.Enable()
				editor.Enable()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				w.SetCloseIntercept(nil)
				w.Close()
				v.reloadCollections()
			})
		}()
	})
	w.SetCloseIntercept(func() {
		if busy {
			return
		}
		if editor.Text == string(item.Data) {
			w.SetCloseIntercept(nil)
			w.Close()
			return
		}
		dialog.ShowConfirm("Discard changes?", "Close without saving this item's changes?", func(ok bool) {
			if ok {
				w.SetCloseIntercept(nil)
				w.Close()
			}
		}, w)
	})
	w.SetContent(container.NewBorder(widget.NewLabel("Edit the original vCard, iCalendar or JSON. Keep its UID and any fields you need."), save, nil, nil, editor))
	showWindow(w)
}
func (v *collectionView) remove() {
	item, ok := v.selectedItem()
	if !ok {
		return
	}
	c, ok := v.collection()
	if !ok {
		return
	}
	prompt := "Remove “" + item.Title + "” locally? Your sync tool can propagate this deletion."
	if !v.contacts && item.Recurring {
		prompt += "\n\nThis removes the whole stored series, including all its occurrences."
	}
	dialog.ShowConfirm("Delete item?", prompt, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			err := pim.Delete(c, item)
			v.app.post(func() {
				if err != nil {
					v.app.fail(err)
					return
				}
				v.reloadCollections()
			})
		}()
	}, v.app.Window)
}
func (a *App) saveReplyContact(raw string, account int) {
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return
	}
	collections := a.visibleCollections(true, account)
	if len(collections) == 0 {
		return
	}
	c := collections[0]
	for _, candidate := range collections {
		if candidate.Account == a.cfg.Accounts[account].Name {
			c = candidate
			break
		}
	}
	go func() {
		added, err := pim.EnsureContact(c, *addr)
		a.post(func() {
			if err != nil {
				a.status.SetText("Reply opened; contact was not saved: " + err.Error())
			} else if added {
				a.status.SetText("Reply sender saved to " + c.Name)
			}
		})
	}()
}
func (a *App) pickContact(c *composer, destination *widget.Entry) {
	w := a.Fyne.NewWindow("Choose a contact")
	w.Resize(fyne.NewSize(600, 450))
	info := widget.NewLabel("Loading contacts…")
	search := widget.NewEntry()
	search.SetPlaceHolder("Search contacts")
	var addresses, filtered []mail.Address
	selected := -1
	alive := true
	list := widget.NewList(func() int { return len(filtered) }, func() fyne.CanvasObject { l := widget.NewLabel(""); l.Truncation = fyne.TextTruncateEllipsis; return l }, func(id int, o fyne.CanvasObject) { o.(*widget.Label).SetText(filtered[id].String()) })
	list.OnSelected = func(id int) { selected = id }
	filter := func() {
		filtered = nil
		selected = -1
		list.UnselectAll()
		for _, addr := range addresses {
			if strings.Contains(strings.ToLower(addr.String()), strings.ToLower(search.Text)) {
				filtered = append(filtered, addr)
			}
		}
		list.Refresh()
		info.SetText(fmt.Sprintf("%d contacts", len(filtered)))
	}
	search.OnChanged = func(string) { filter() }
	add := widget.NewButton("Add recipient", func() {
		if !c.sending && selected >= 0 && selected < len(filtered) {
			value := filtered[selected].String()
			if strings.TrimSpace(destination.Text) != "" {
				value = strings.TrimRight(destination.Text, " ,") + ", " + value
			}
			destination.SetText(value)
			w.Close()
		}
	})
	w.SetOnClosed(func() { alive = false })
	w.SetContent(container.NewBorder(container.NewVBox(search, info), add, nil, nil, list))
	showWindow(w)
	collections := a.visibleCollections(true, c.account)
	go func() {
		seen := map[string]bool{}
		var loaded []mail.Address
		var failures []string
		for _, collection := range collections {
			items, err := pim.Load(collection)
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			for _, item := range items {
				for _, addr := range pim.ContactAddresses(item) {
					key := strings.ToLower(addr.Address)
					if !seen[key] {
						seen[key] = true
						loaded = append(loaded, addr)
					}
				}
			}
		}
		a.post(func() {
			if !alive {
				return
			}
			addresses = loaded
			filter()
			if len(collections) == 0 {
				info.SetText("No contacts collections configured for this account.")
			}
			if len(failures) > 0 {
				info.SetText(strings.Join(failures, "\n"))
			}
		})
	}()
}
