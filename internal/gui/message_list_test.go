// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/SeraphinaDX/MailSalonGUI/internal/demo"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func demoApp(t *testing.T) (*App, chan func()) {
	t.Helper()
	cfg, root, err := demo.Create()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	a, q := testApp(t, cfg)
	a.Start()
	pump(t, q, func() bool { return len(a.messages) == 5 && len(a.contacts.items) == 2 && len(a.calendar.items) == 1 })
	a.Window.Show()
	return a, q
}
func waitForMessage(t *testing.T, a *App, q chan func()) {
	t.Helper()
	pump(t, q, func() bool { return a.parsed != nil && !a.changing })
}
func visibleMessageRow(t *testing.T, a *App, id int) *messageRow {
	t.Helper()
	var find func(fyne.CanvasObject) *messageRow
	find = func(o fyne.CanvasObject) *messageRow {
		if r, ok := o.(*messageRow); ok {
			if r.id == id {
				return r
			}
			return nil
		}
		if box, ok := o.(*fyne.Container); ok {
			for _, child := range box.Objects {
				if r := find(child); r != nil {
					return r
				}
			}
		}
		if w, ok := o.(fyne.Widget); ok {
			for _, child := range test.WidgetRenderer(w).Objects() {
				if r := find(child); r != nil {
					return r
				}
			}
		}
		return nil
	}
	if row := find(a.messageList); row != nil {
		return row
	}
	t.Fatalf("visible message row %d not found", id)
	return nil
}
func TestDeleteKeyUsesConfirmationAndCurrentMaildirPath(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	original := a.messages[0].Path
	// Delete before parsing completes. Parsing can mark the mail read and rename
	// it while the confirmation is open; confirming must still delete that mail.
	a.Window.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if a.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("Delete did not show confirmation")
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatal("mail deleted before confirmation")
	}
	waitForMessage(t, a, q)
	subject := a.messages[0].Subject
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	trash, _ := maildir.Scan(maildir.Folder{Name: "Trash", Path: filepath.Join(a.cfg.Accounts[0].Maildir, "Trash")})
	if len(trash) != 1 || trash[0].Subject != subject {
		t.Fatal("Delete key did not move selected mail to Trash")
	}
}
func TestDeleteOnlyAppliesToMessageList(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.Window.Canvas().Focus(a.search)
	a.Window.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if len(a.messages) != 5 || a.Window.Canvas().Overlays().Top() != nil {
		t.Fatal("Delete outside the list deleted mail")
	}
	a.messageList.UnselectAll()
	a.clearPreview()
	a.Window.Canvas().Focus(a.messageList)
	a.Window.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if a.Window.Canvas().Overlays().Top() != nil {
		t.Fatal("Delete with no selection showed confirmation")
	}
	a.search.SetText("no messages match")
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	a.showMessageMenu(0, fyne.NewPos(300, 200))
	if a.messageMenu != nil {
		t.Fatal("empty list showed a context menu")
	}
}
func TestRightClickSelectsClickedMailAndEnablesReadyActions(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	row := visibleMessageRow(t, a, 2)
	subject := a.messages[2].Subject
	test.TapSecondary(row)
	if a.selected != 2 || a.messageMenu == nil {
		t.Fatal("right-click did not select clicked mail and show menu")
	}
	if !a.messageMenuItems["reply"].Disabled {
		t.Fatal("reply enabled before clicked mail parsed")
	}
	waitForMessage(t, a, q)
	if a.parsed.Subject != subject || a.messageMenuItems["reply"].Disabled || a.messageMenuItems["delete"].Disabled {
		t.Fatal("context actions did not follow parsed clicked mail")
	}
	screenshot(t, a.Window, "message-menu")
	a.messageMenuItems["unread"].Action()
	pump(t, q, func() bool { return !a.changing })
	if !a.messages[2].Unread {
		t.Fatal("context action marked a different mail unread")
	}
	a.messageMenuItems["delete"].Action()
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	trash, _ := maildir.Scan(maildir.Folder{Name: "Trash", Path: filepath.Join(a.cfg.Accounts[0].Maildir, "Trash")})
	if len(trash) != 1 || trash[0].Subject != subject {
		t.Fatal("context Delete targeted the previously selected email")
	}
}
func TestStaleContextActionCannotDeleteNewSelection(t *testing.T) {
	a, q := demoApp(t)
	a.showMessageMenu(0, fyne.NewPos(300, 200))
	waitForMessage(t, a, q)
	staleDelete := a.messageMenuItems["delete"].Action
	a.search.SetText("Coffee")
	if a.messageMenu != nil {
		t.Fatal("search did not close context menu")
	}
	staleDelete()
	if a.Window.Canvas().Overlays().Top() != nil {
		t.Fatal("stale menu opened a deletion confirmation")
	}
}
func TestRowClickAndKeyboardNavigationUseSameSelection(t *testing.T) {
	a, q := demoApp(t)
	test.Tap(visibleMessageRow(t, a, 2))
	waitForMessage(t, a, q)
	if a.Window.Canvas().Focused() != a.messageList || a.selected != 2 {
		t.Fatal("row tap did not focus and select mail")
	}
	a.Window.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	waitForMessage(t, a, q)
	if a.selected != 3 {
		t.Fatal("Down did not follow clicked row")
	}
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	waitForMessage(t, a, q)
	if a.selected != 0 {
		t.Fatal("Home did not select first row")
	}
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	waitForMessage(t, a, q)
	if a.selected != 4 {
		t.Fatal("End did not select last row")
	}
}

func TestKeyboardDeletionCanBeCancelledAndConfirmsPermanentTrashDeletion(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	path := a.messages[0].Path
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	a.Window.Canvas().Overlays().Top().Hide()
	if _, err := os.Stat(path); err != nil {
		t.Fatal("cancelling deletion removed email")
	}
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	for i, f := range a.folders {
		if f.Name == "Trash" {
			a.folderList.Select(i)
			break
		}
	}
	pump(t, q, func() bool { return len(a.messages) == 1 })
	a.selectMessage(0)
	waitForMessage(t, a, q)
	path = a.messages[0].Path
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if _, err := os.Stat(path); err != nil {
		t.Fatal("Trash email removed before confirmation")
	}
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 0 })
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("confirmed Trash deletion did not remove file")
	}
}
