// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func visibleFolderRow(t *testing.T, a *App, name string) *folderRow {
	t.Helper()
	var find func(fyne.CanvasObject) *folderRow
	find = func(o fyne.CanvasObject) *folderRow {
		if !o.Visible() {
			return nil
		}
		if r, ok := o.(*folderRow); ok {
			if r.id >= 0 && r.label.Text == name {
				return r
			}
			return nil
		}
		var children []fyne.CanvasObject
		if box, ok := o.(*fyne.Container); ok {
			children = box.Objects
		} else if w, ok := o.(fyne.Widget); ok {
			children = test.WidgetRenderer(w).Objects()
		}
		for _, child := range children {
			if r := find(child); r != nil {
				return r
			}
		}
		return nil
	}
	if r := find(a.folderList); r != nil {
		return r
	}
	t.Fatalf("folder row %s missing", name)
	return nil
}
func rowCenter(a *App, r fyne.CanvasObject) fyne.Position {
	return a.Fyne.Driver().AbsolutePositionForObject(r).Add(fyne.NewPos(r.Size().Width/2, r.Size().Height/2))
}
func scanDemoFolder(t *testing.T, a *App, name string) []maildir.Entry {
	t.Helper()
	entries, err := maildir.Scan(maildir.Folder{Name: name, Path: filepath.Join(a.cfg.Accounts[0].Maildir, name)})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
func beginDrag(t *testing.T, a *App, id int, target string) (*messageRow, *folderRow) {
	t.Helper()
	r := visibleMessageRow(t, a, id)
	r.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: rowCenter(a, r)}})
	f := visibleFolderRow(t, a, target)
	pos := rowCenter(a, f)
	test.MoveMouse(a.Window.Canvas(), pos)
	r.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: pos}})
	return r, f
}

func TestFolderRowPrimaryClickAfterContextMenu(t *testing.T) {
	a, q := demoApp(t)
	sent := visibleFolderRow(t, a, "Sent")
	// Desktop Fyne hit testing picks the row because it handles secondary
	// clicks. TapCanvas only searches primary handlers and can incorrectly
	// fall through to the enclosing list item, hiding this regression.
	tappable, ok := any(sent).(fyne.Tappable)
	if !ok {
		t.Fatal("folder context-menu row must also handle primary clicks")
	}
	test.TapSecondary(sent)
	if filepath.Base(a.folderPath) != "INBOX" {
		t.Fatal("context menu changed the current folder")
	}
	a.Window.Canvas().Overlays().Top().Hide()
	test.Tap(tappable)
	pump(t, q, func() bool { return filepath.Base(a.folderPath) == "Sent" && a.status.Text == "Ready — Sent" })
	inbox := visibleFolderRow(t, a, "INBOX")
	test.Tap(any(inbox).(fyne.Tappable))
	pump(t, q, func() bool { return filepath.Base(a.folderPath) == "INBOX" && len(a.messages) == 5 })
}

func TestFolderRowClickRejectsStaleRow(t *testing.T) {
	a, _ := demoApp(t)
	row := visibleFolderRow(t, a, "Sent")
	tappable, ok := any(row).(fyne.Tappable)
	if !ok {
		t.Fatal("folder row must handle primary clicks")
	}
	row.path = "stale path"
	test.Tap(tappable)
	row.id = -1
	test.Tap(tappable)
	if filepath.Base(a.folderPath) != "INBOX" {
		t.Fatal("stale row changed folder")
	}
}

func TestDragCanvasMovesMailToFolder(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	want := a.messages[0].Subject
	r := visibleMessageRow(t, a, 0)
	destination := visibleFolderRow(t, a, "Sent")
	start, end := rowCenter(a, r), rowCenter(a, destination)
	// The desktop dispatches hover before the first drag event when a pointer
	// jumps directly to the target. Exercise that order with real canvas lookup.
	test.MoveMouse(a.Window.Canvas(), end)
	test.Drag(a.Window.Canvas(), start, end.X-start.X, end.Y-start.Y)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	sent := scanDemoFolder(t, a, "Sent")
	if len(sent) != 1 || sent[0].Subject != want {
		t.Fatal("drag moved wrong mail", sent)
	}
	if a.mailDrag != nil || destination.drop || a.parsed != nil || len(a.messageList.SelectedIDs()) != 0 {
		t.Fatal("move left stale UI state")
	}
	if filepath.Base(a.folderPath) != "INBOX" {
		t.Fatal("drop opened destination")
	}
	// Hover targets must retain ordinary folder-click behavior.
	destination = visibleFolderRow(t, a, "Sent")
	test.TapCanvas(a.Window.Canvas(), rowCenter(a, destination))
	pump(t, q, func() bool { return filepath.Base(a.folderPath) == "Sent" && len(a.messages) == 1 })
}

func TestDragMovesToDiscoveredNestedFolder(t *testing.T) {
	a, q := demoApp(t)
	path := filepath.Join(a.cfg.Accounts[0].Maildir, ".Projects.Client")
	if err := maildir.Ensure(path); err != nil {
		t.Fatal(err)
	}
	a.reload()
	pump(t, q, func() bool { return len(a.folders) == 5 && len(a.messages) == 5 })
	a.selectMessage(0)
	waitForMessage(t, a, q)
	want := a.messages[0].Subject
	r, _ := beginDrag(t, a, 0, "Projects/Client")
	r.DragEnd()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	entries, err := maildir.Scan(maildir.Folder{Path: path})
	if err != nil || len(entries) != 1 || entries[0].Subject != want {
		t.Fatal("nested folder move failed", entries, err)
	}
}

func TestDragKeepsGroupAndHighlightsTarget(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.messageList.SelectWithModifiers(1, fyne.KeyModifierShift)
	waitForMessage(t, a, q)
	want := map[string]bool{a.messages[0].Subject: true, a.messages[1].Subject: true}
	r, f := beginDrag(t, a, 0, "Archive")
	if a.mailDrag == nil || a.mailDrag.count != 2 || !f.drop || len(a.messageList.SelectedIDs()) != 2 {
		t.Fatal("drag lost group or target feedback")
	}
	screenshot(t, a.Window, "mail-drag")
	r.DragEnd()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 3 })
	entries := scanDemoFolder(t, a, "Archive")
	if len(entries) != 2 {
		t.Fatal(entries)
	}
	for _, e := range entries {
		if !want[e.Subject] {
			t.Fatal("moved mail outside selection")
		}
	}
}

func TestDragUnselectedRowMovesOnlyThatMail(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.messageList.SelectWithModifiers(1, fyne.KeyModifierShift)
	waitForMessage(t, a, q)
	want := a.messages[3].Subject
	r, _ := beginDrag(t, a, 3, "Archive")
	if a.mailDrag == nil || a.mailDrag.count != 1 || !a.messageList.selection[3] {
		t.Fatal("unselected drag reused old group")
	}
	r.DragEnd()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	entries := scanDemoFolder(t, a, "Archive")
	if len(entries) != 1 || entries[0].Subject != want {
		t.Fatal(entries)
	}
}

func TestDragWaitsForReadRenameAndResolvesCurrentPath(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	pump(t, q, func() bool { return a.changingRead })
	want := a.messages[0].Subject
	r, _ := beginDrag(t, a, 0, "Archive")
	r.DragEnd()
	if a.pendingDelete == nil {
		t.Fatal("drop during mark-read was discarded")
	}
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
	entries := scanDemoFolder(t, a, "Archive")
	if len(entries) != 1 || entries[0].Subject != want || entries[0].Unread {
		t.Fatal("drop used stale path", entries)
	}
}

func TestDragCancellationAndStaleViewGuards(t *testing.T) {
	for _, scenario := range []string{"outside", "same folder", "search", "selection", "reload", "folder", "account", "recycled target", "tab", "sync", "dialog", "escape"} {
		t.Run(scenario, func(t *testing.T) {
			a, q := demoApp(t)
			a.selectMessage(0)
			waitForMessage(t, a, q)
			r, _ := beginDrag(t, a, 0, "Archive")
			switch scenario {
			case "outside":
				pos := rowCenter(a, a.search)
				test.MoveMouse(a.Window.Canvas(), pos)
				r.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: pos}})
			case "same folder":
				pos := rowCenter(a, visibleFolderRow(t, a, "INBOX"))
				test.MoveMouse(a.Window.Canvas(), pos)
				r.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: pos}})
			case "search":
				a.search.SetText("Coffee")
			case "selection":
				a.messageList.Select(1)
			case "reload":
				a.reload()
			case "folder":
				a.loadFolder(maildir.Folder{Name: "Sent", Path: filepath.Join(a.cfg.Accounts[0].Maildir, "Sent")})
			case "account":
				a.account++ // Identity alone must invalidate an old gesture.
			case "recycled target":
				a.folderHover.path = filepath.Join(a.cfg.Accounts[0].Maildir, "Sent")
			case "tab":
				a.tabs.SelectIndex(1)
			case "sync":
				a.syncing = true
			case "dialog":
				dialog.ShowInformation("Test", "Modal", a.Window)
			case "escape":
				a.messageList.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			}
			r.DragEnd()
			if len(scanDemoFolder(t, a, "Archive")) != 0 || len(scanDemoFolder(t, a, "INBOX")) != 5 {
				t.Fatal("cancelled/stale drag moved mail")
			}
			if a.mailDrag != nil {
				t.Fatal("drag state survived cancellation")
			}
		})
	}
}

func TestRejectedDragCannotRestartBeforeRelease(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.syncing = true
	r, _ := beginDrag(t, a, 0, "Archive")
	a.syncing = false
	r.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: rowCenter(a, visibleFolderRow(t, a, "Archive"))}})
	r.DragEnd()
	if a.mailDrag != nil || len(scanDemoFolder(t, a, "Archive")) != 0 {
		t.Fatal("rejected gesture restarted after becoming idle")
	}
	// A fresh gesture is accepted normally.
	r, _ = beginDrag(t, a, 0, "Archive")
	r.DragEnd()
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 4 })
}

func TestFailedMoveReportsErrorAndKeepsSource(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	path := a.messages[0].Path
	r, f := beginDrag(t, a, 0, "Archive")
	if err := os.RemoveAll(f.path); err != nil {
		t.Fatal(err)
	}
	r.DragEnd()
	pump(t, q, func() bool { return !a.changing && a.Window.Canvas().Overlays().Top() != nil })
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed move lost source", err)
	}
}
