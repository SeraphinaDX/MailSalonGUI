// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func (a *App) selectMessage(id int) {
	if a.changing {
		return
	}
	a.Window.Canvas().Focus(a.messageList)
	a.messageList.Select(id)
	if a.selected != id {
		a.openMessage(id)
	}
}
func (a *App) hideMessageMenu() {
	if a.messageMenu != nil {
		a.messageMenu.Hide()
		a.messageMenu = nil
	}
	a.messageMenuItems = nil
}
func (a *App) showMessageMenu(id int, position fyne.Position) {
	if a.changing || id < 0 || id >= len(a.messages) {
		return
	}
	a.hideMessageMenu()
	a.selectMessage(id)
	gen, reading := a.generation, a.reading
	// A timer reload or changed selection can invalidate an open popup. Guard
	// callbacks as well as hiding it, so recycled rows never target another mail.
	action := func(fn func()) func() {
		return func() {
			if gen == a.generation && reading == a.reading && a.selected == id {
				fn()
			}
		}
	}
	items := map[string]*fyne.MenuItem{
		"reply":    fyne.NewMenuItem("Reply", action(func() { a.reply(false, false) })),
		"replyAll": fyne.NewMenuItem("Reply all", action(func() { a.reply(false, true) })),
		"forward":  fyne.NewMenuItem("Forward", action(func() { a.reply(true, false) })),
		"read": fyne.NewMenuItem("Mark as read", action(func() {
			if e, ok := a.currentEntry(); ok {
				a.changeRead(e, true)
			}
		})),
		"unread": fyne.NewMenuItem("Mark as unread", action(func() {
			if e, ok := a.currentEntry(); ok {
				a.changeRead(e, false)
			}
		})),
		"archive": fyne.NewMenuItem("Archive", action(a.archive)),
		"delete":  fyne.NewMenuItem("Delete", action(a.deleteMessage)),
		"source":  fyne.NewMenuItem("View source", action(a.showSource)),
	}
	menu := fyne.NewMenu("", items["reply"], items["replyAll"], items["forward"], fyne.NewMenuItemSeparator(), items["read"], items["unread"], items["archive"], items["delete"], fyne.NewMenuItemSeparator(), items["source"])
	a.messageMenuItems = items
	a.messageMenu = widget.NewPopUpMenu(menu, a.Window.Canvas())
	a.updateMessageMenu()
	a.messageMenu.ShowAtPosition(position)
}
func (a *App) updateMessageMenu() {
	if a.messageMenu == nil {
		return
	}
	for _, key := range []string{"reply", "replyAll", "forward"} {
		a.messageMenuItems[key].Disabled = a.parsed == nil
	}
	e, ok := a.currentEntry()
	for _, key := range []string{"read", "unread", "archive", "delete", "source"} {
		a.messageMenuItems[key].Disabled = !ok
	}
	if ok {
		a.messageMenuItems["read"].Disabled = !e.Unread
		a.messageMenuItems["unread"].Disabled = e.Unread
	}
	a.messageMenu.Refresh()
}
