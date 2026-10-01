// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func assertSelection(t *testing.T, a *App, ids ...int) {
	t.Helper()
	got := a.messageList.SelectedIDs()
	if len(got) != len(ids) || (len(ids) > 0 && !reflect.DeepEqual(got, ids)) {
		t.Fatalf("selection = %v, want %v", got, ids)
	}
	for _, id := range ids {
		row := visibleMessageRow(t, a, id)
		if !row.selected || row.background == nil || row.background.FillColor == nil {
			t.Fatalf("row %d is not highlighted", id)
		}
	}
}
func scanFolder(t *testing.T, a *App, name string) []maildir.Entry {
	t.Helper()
	entries, err := maildir.Scan(maildir.Folder{Name: name, Path: filepath.Join(a.cfg.Accounts[0].Maildir, name)})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
func clickMessage(t *testing.T, a *App, id int, modifiers fyne.KeyModifier) {
	t.Helper()
	a.messageList.modifiers = func() fyne.KeyModifier { return modifiers }
	test.Tap(visibleMessageRow(t, a, id))
	a.messageList.modifiers = func() fyne.KeyModifier { return 0 }
}
func TestMultiSelectionMouseKeyboardAndContext(t *testing.T) {
	a, q := demoApp(t)
	clickMessage(t, a, 1, 0)
	clickMessage(t, a, 3, fyne.KeyModifierControl)
	assertSelection(t, a, 1, 3)
	clickMessage(t, a, 1, fyne.KeyModifierControl)
	assertSelection(t, a, 3)
	clickMessage(t, a, 1, fyne.KeyModifierShift)
	assertSelection(t, a, 1) // Ctrl-click reset the range anchor to 1.
	clickMessage(t, a, 4, fyne.KeyModifierShift)
	assertSelection(t, a, 1, 2, 3, 4)
	a.messageList.modifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	assertSelection(t, a, 1, 2, 3)
	a.messageList.modifiers = func() fyne.KeyModifier { return 0 }
	waitForMessage(t, a, q)
	revision := a.messageList.revision
	test.TapSecondary(visibleMessageRow(t, a, 2))
	assertSelection(t, a, 1, 2, 3)
	if a.messageList.revision != revision || a.messageMenu == nil {
		t.Fatal("right click replaced group selection")
	}
	if !a.messageMenuItems["reply"].Disabled || !a.messageMenuItems["source"].Disabled {
		t.Fatal("single-message menu actions enabled for a group")
	}
	screenshot(t, a.Window, "bulk-message-menu")
	test.TapSecondary(visibleMessageRow(t, a, 0))
	assertSelection(t, a, 0)
	waitForMessage(t, a, q)
	a.messageList.TypedShortcut(&fyne.ShortcutSelectAll{})
	assertSelection(t, a, 0, 1, 2, 3, 4)
	if !strings.Contains(a.summary.Text, "5 selected") {
		t.Fatal("selection count not shown")
	}
	screenshot(t, a.Window, "bulk-selection")
	a.messageList.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeySpace, Modifier: fyne.KeyModifierControl})
	assertSelection(t, a, 1, 2, 3, 4)
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assertSelection(t, a)
	if a.selected != -1 || a.parsed != nil {
		t.Fatal("clearing selection retained preview")
	}
}
func TestBulkSelectionFilteringDropsHiddenMessages(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.SelectAll()
	waitForMessage(t, a, q)
	a.search.SetText("Coffee")
	assertSelection(t, a, 0)
	waitForMessage(t, a, q)
	a.search.SetText("")
	assertSelection(t, a, 1)
	a.search.SetText("nothing matches")
	assertSelection(t, a)
	a.messageList.TypedShortcut(&fyne.ShortcutSelectAll{})
	assertSelection(t, a)
}

func TestSelectionShortcutsKeepMainAndTextFieldShortcutsWorking(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.messageList.TypedShortcut(&fyne.ShortcutSelectAll{})
	a.messageList.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl})
	if a.Window.Canvas().Focused() != a.search {
		t.Fatal("Ctrl+F from list did not reach the main window")
	}
	a.search.SetText("notes")
	a.search.TypedShortcut(&fyne.ShortcutSelectAll{})
	if a.search.SelectedText() != "notes" {
		t.Fatal("Ctrl+A in search did not select text")
	}
	assertSelection(t, a, 0)
	a.search.SetText("")
	a.Window.Canvas().Focus(a.messageList)
	a.messageList.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: fyne.KeyModifierControl})
	if len(a.composers) != 1 {
		t.Fatal("Ctrl+N from list did not open compose")
	}
}
func TestBulkDeleteUsesOneConfirmationAndExactSelection(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	a.messageList.SelectWithModifiers(2, fyne.KeyModifierControl)
	subjects := []string{a.messages[0].Subject, a.messages[2].Subject}
	a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if a.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("bulk delete did not ask for confirmation")
	}
	waitForMessage(t, a, q)
	if len(scanFolder(t, a, "INBOX")) != 5 {
		t.Fatal("bulk deletion happened before confirmation")
	}
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 3 })
	trash := scanFolder(t, a, "Trash")
	if len(trash) != 2 || trash[0].Subject != subjects[0] || trash[1].Subject != subjects[1] {
		t.Fatalf("wrong bulk deletion targets: %v", trash)
	}
	for i, folder := range a.folders {
		if folder.Name == "Trash" {
			a.folderList.Select(i)
			break
		}
	}
	pump(t, q, func() bool { return len(a.messages) == 2 })
	a.messageList.SelectAll()
	waitForMessage(t, a, q)
	a.deleteMessage()
	if len(scanFolder(t, a, "Trash")) != 2 {
		t.Fatal("trash was emptied before confirmation")
	}
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 0 })
	if len(scanFolder(t, a, "Trash")) != 0 {
		t.Fatal("bulk permanent deletion did not empty Trash")
	}
}
func TestBulkDeleteConfirmationCannotFollowChangedGroup(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	a.messageList.SelectWithModifiers(2, fyne.KeyModifierControl)
	waitForMessage(t, a, q)
	a.deleteMessage()
	a.messageList.SelectWithModifiers(3, fyne.KeyModifierControl)
	tapConfirmation(t, a.Window)
	if a.changing || len(scanFolder(t, a, "INBOX")) != 5 {
		t.Fatal("stale confirmation deleted a changed selection")
	}
}
func TestBulkArchiveAndReadFlags(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	waitForMessage(t, a, q)
	a.messageList.SelectWithModifiers(2, fyne.KeyModifierControl)
	waitForMessage(t, a, q)
	a.toggleRead() // A mixed selection becomes read as a group.
	pump(t, q, func() bool { return !a.changing })
	if a.messages[0].Unread || a.messages[2].Unread || !a.messages[1].Unread {
		t.Fatal("bulk mark read changed wrong flags")
	}
	a.toggleRead()
	pump(t, q, func() bool { return !a.changing })
	if !a.messages[0].Unread || !a.messages[2].Unread {
		t.Fatal("bulk toggle did not mark group unread")
	}
	assertSelection(t, a, 0, 2)
	subjects := []string{a.messages[0].Subject, a.messages[2].Subject}
	a.showMessageMenu(0, fyne.NewPos(300, 200))
	a.messageMenuItems["archive"].Action()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 3 })
	archived := scanFolder(t, a, "Archive")
	if len(archived) != 2 || archived[0].Subject != subjects[0] || archived[1].Subject != subjects[1] {
		t.Fatalf("wrong archive targets: %v", archived)
	}
}
func TestBulkMarkUnreadSurvivesPendingPreviewAndRetainsSelection(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.TypedShortcut(&fyne.ShortcutSelectAll{})
	a.markSelectedRead(true)
	pump(t, q, func() bool { return !a.changing && a.parsed != nil })
	a.markSelectedRead(false)
	pump(t, q, func() bool { return !a.changing })
	assertSelection(t, a, 0, 1, 2, 3, 4)
	for _, e := range scanFolder(t, a, "INBOX") {
		if !e.Unread {
			t.Fatalf("explicit unread flag reverted for %s", e.Subject)
		}
	}
}
func TestBulkActionWaitsForAutomaticReadRename(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	pump(t, q, func() bool { return a.changingRead })
	a.messageList.SelectWithModifiers(2, fyne.KeyModifierControl)
	a.archive()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 3 })
	if len(scanFolder(t, a, "Archive")) != 2 {
		t.Fatal("bulk archive lost automatic-read rename")
	}
}
func TestBulkActionReportsPartialFailureAndContinues(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.SelectAll()
	waitForMessage(t, a, q)
	if err := os.Remove(a.messages[1].Path); err != nil {
		t.Fatal(err)
	}
	a.archive()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 0 })
	if len(scanFolder(t, a, "Archive")) != 4 {
		t.Fatal("batch stopped at the first failure")
	}
	if a.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("partial failure was not reported")
	}
}

func TestBulkWorkerKeepsSnapshotAndRefreshesNewlyViewedDestination(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	a.messageList.SelectWithModifiers(2, fyne.KeyModifierControl)
	waitForMessage(t, a, q)
	entries := a.selectedEntries()
	archive, ok := maildir.FindFolder(a.folders, "Archive")
	if !ok {
		t.Fatal("missing demo archive")
	}
	started, resume := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-resume:
		default:
			close(resume)
		}
	}()
	first := true
	a.mutateMessages(entries, "Archived", true, func(e *maildir.Entry) error {
		if first {
			first = false
			close(started)
			<-resume
		}
		return maildir.Archive(*e, archive)
	})
	<-started
	a.loadFolder(archive)
	pump(t, q, func() bool { return strings.Contains(a.status.Text, "Ready — Archive") })
	if len(a.messages) != 0 {
		t.Fatal("archive populated before the worker resumed")
	}
	close(resume)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 2 })
	if a.folderPath != archive.Path || len(scanFolder(t, a, "INBOX")) != 3 {
		t.Fatal("batch followed the new view instead of its snapshot")
	}
}
