// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

// Letter shortcuts belong to the visible, focused mail list. In particular,
// a modal confirmation must not let an archive key act on mail behind it.
func (a *App) handleMessageRune(r rune) {
	if a.Window.Canvas().Focused() != a.messageList || a.tabs.SelectedIndex() != 0 || a.Window.Canvas().Overlays().Top() != nil {
		return
	}
	switch r {
	case 'n':
		a.compose(mimeutil.Draft{}, a.account, "")
	case '/':
		a.Window.Canvas().Focus(a.search)
	case '?':
		a.showKeyboardShortcuts()
	case 'j', 'k':
		key := fyne.KeyDown
		if r == 'k' {
			key = fyne.KeyUp
		}
		a.messageList.TypedKey(&fyne.KeyEvent{Name: key})
	case 'e':
		a.archive()
	case 'u':
		a.toggleRead()
	case 'd':
		a.deleteMessage()
	case 'r', 'R', 'f', 's':
		// Reply/forward/source target a single selected message, like the menu.
		if len(a.messageList.SelectedIDs()) != 1 || !a.messageList.selection[a.selected] || a.changing {
			return
		}
		if r == 's' {
			a.showSource()
		} else if a.parsed != nil {
			a.reply(r == 'f', r == 'R')
		}
	}
}

func (a *App) showKeyboardShortcuts() {
	dialog.ShowInformation("Keyboard shortcuts", "With the message list focused:\n"+
		"r — Reply                 Shift+r — Reply all\n"+
		"f — Forward              n — New message\n"+
		"e — Archive              u — Read / unread\n"+
		"d / Delete / Backspace / numpad Del — Confirm delete\n"+
		"s — View source         / — Search          ? — This help\n"+
		"j / k or Down / Up — Next / previous message\n"+
		"Home / End — First / last message\n"+
		"Ctrl-click — Toggle selection; Shift-click — Select range\n"+
		"Shift+Up/Down/Home/End — Extend selection\n"+
		"Ctrl+A — Select all; Ctrl+Space — Toggle active selection\n"+
		"Escape — Clear selection\n\n"+
		"Archive, read/unread and delete apply to the whole selection.\n"+
		"Reply, forward and source require one selected message.\n"+
		"Letters in text fields type normally.\n\n"+
		"Main window: Ctrl+N — Compose; Ctrl+R — Reply\n"+
		"Ctrl+F — Search; F5 — Sync\n"+
		"Compose: Ctrl+S — Save draft; Ctrl+Enter — Confirm send", a.Window)
}
