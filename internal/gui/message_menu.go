// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func (a *App) selectMessage(id int) {
	if a.changing && !a.changingRead {
		return
	}
	a.Window.Canvas().Focus(a.messageList)
	a.messageList.SelectWithModifiers(id, a.messageList.modifiers())
	if a.messageList.selectedID >= 0 && a.selected != a.messageList.selectedID {
		a.openMessage(a.messageList.selectedID)
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
	a.Window.Canvas().Focus(a.messageList)
	if !a.messageList.selection[id] {
		a.messageList.Select(id)
	}
	gen, revision := a.generation, a.messageList.revision
	// A timer reload or changed selection can invalidate an open popup. Guard
	// callbacks as well as hiding it, so recycled rows never target another mail.
	action := func(fn func()) func() {
		return func() {
			if gen == a.generation && revision == a.messageList.revision {
				fn()
			}
		}
	}
	items := map[string]*fyne.MenuItem{
		"reply":    fyne.NewMenuItem("Reply", action(func() { a.reply(false, false) })),
		"replyAll": fyne.NewMenuItem("Reply all", action(func() { a.reply(false, true) })),
		"forward":  fyne.NewMenuItem("Forward", action(func() { a.reply(true, false) })),
		"read": fyne.NewMenuItem("Mark as read", action(func() {
			a.markSelectedRead(true)
		})),
		"unread": fyne.NewMenuItem("Mark as unread", action(func() {
			a.markSelectedRead(false)
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
	entries := a.selectedEntries()
	single := len(entries) == 1 && a.messageList.selection[a.selected]
	for _, key := range []string{"reply", "replyAll", "forward"} {
		a.messageMenuItems[key].Disabled = !single || a.parsed == nil || a.changing
	}
	for _, key := range []string{"read", "unread", "archive", "delete"} {
		a.messageMenuItems[key].Disabled = a.changing || len(entries) == 0
	}
	a.messageMenuItems["source"].Disabled = !single || a.changing
	if !a.changing && len(entries) > 0 {
		anyRead, anyUnread := false, false
		for _, e := range entries {
			if e.Unread {
				anyUnread = true
			} else {
				anyRead = true
			}
		}
		a.messageMenuItems["read"].Disabled = !anyUnread
		a.messageMenuItems["unread"].Disabled = !anyRead
	}
	a.messageMenu.Refresh()
}
