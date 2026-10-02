// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
	"github.com/SeraphinaDX/MailSalonGUI/internal/transport"
	"path/filepath"
)

func folderTestResponse(acc config.Account) transport.FolderResponse {
	return transport.FolderResponse{Version: 1, Account: acc.SyncAccount, LocalRoot: acc.Maildir, Folders: []transport.RemoteFolder{{ID: "id:inbox", Name: "INBOX", Subscribed: true, Syncing: true, Local: "INBOX", Selectable: true, CanCreateChild: true}, {ID: "id:work", Name: "Projects/Work", Selectable: true, CanCreateChild: true}}}
}

func TestFolderManagerNewFolderFormSendsParentAndLiteralName(t *testing.T) {
	a, q := demoApp(t)
	a.cfg.Accounts[0].SyncAccount = "sync-mail"
	requests := make(chan transport.FolderRequest, 4)
	a.folderCommand = func(ctx context.Context, acc config.Account, req transport.FolderRequest) (transport.FolderResponse, error) {
		requests <- req
		return folderTestResponse(acc), nil
	}
	a.showFolderManager(true)
	m := a.folderManager
	pump(t, q, func() bool { return !a.folderBusy })
	<-requests
	var form *widget.Form
	var find func(fyne.CanvasObject)
	find = func(o fyne.CanvasObject) {
		if f, ok := o.(*widget.Form); ok {
			form = f
			return
		}
		if p, ok := o.(*widget.PopUp); ok {
			find(p.Content)
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				find(child)
			}
		}
	}
	find(m.window.Canvas().Overlays().Top())
	if form == nil {
		t.Fatal("new folder form missing")
	}
	form.Items[0].Widget.(*widget.Entry).SetText("Café & Work")
	form.Items[1].Widget.(*widget.Select).SetSelected("INBOX")
	form.Items[2].Widget.(*widget.Entry).SetText("Projects/Work")
	screenshot(t, m.window, "new-folder")
	button := calendarButton(m.window.Canvas().Overlays().Top().(*widget.PopUp).Content, "Create")
	if button == nil {
		t.Fatal("create button missing")
	}
	test.Tap(button)
	pump(t, q, func() bool { return !a.folderBusy })
	req := <-requests
	if req.Action != "create" || req.Name != "Café & Work" || req.Parent != "id:inbox" || req.Local != "Projects/Work" {
		t.Fatal(req)
	}
	m.window.Close()
}
func TestFolderManagerSubscribeAndUnsubscribeConfirmation(t *testing.T) {
	a, q := demoApp(t)
	a.cfg.Accounts[0].SyncAccount = "sync-mail"
	requests := make(chan transport.FolderRequest, 10)
	a.folderCommand = func(ctx context.Context, acc config.Account, req transport.FolderRequest) (transport.FolderResponse, error) {
		requests <- req
		response := folderTestResponse(acc)
		if req.Action == "subscribe" {
			response.Folders[1].Subscribed = true
			response.Folders[1].Syncing = true
			response.Folders[1].Local = "Folders/Projects/Work"
			if err := maildir.Ensure(filepath.Join(acc.Maildir, "Folders/Projects/Work")); err != nil {
				return response, err
			}
		}
		return response, nil
	}
	a.showFolderManager(false)
	m := a.folderManager
	pump(t, q, func() bool { return !a.folderBusy })
	if req := <-requests; req.Action != "list" {
		t.Fatal(req)
	}
	m.list.Select(1)
	if m.subscribe.Disabled() || !m.unsubscribe.Disabled() {
		t.Fatal("incorrect action eligibility")
	}
	screenshot(t, m.window, "folder-manager")
	test.Tap(m.subscribe)
	pump(t, q, func() bool { return !a.folderBusy })
	if req := <-requests; req.Action != "subscribe" || req.Mailbox != "id:work" {
		t.Fatal(req)
	}
	pump(t, q, func() bool { return len(a.folders) == 5 })
	m.list.Select(0)
	test.Tap(m.unsubscribe)
	if b := calendarButton(m.window.Canvas().Overlays().Top().(*widget.PopUp).Content, "No"); b != nil {
		test.Tap(b)
	} else {
		t.Fatal("missing confirmation cancel")
	}
	select {
	case req := <-requests:
		t.Fatal("cancel triggered command", req)
	default:
	}
	test.Tap(m.unsubscribe)
	tapConfirmation(t, m.window)
	pump(t, q, func() bool { return !a.folderBusy })
	if req := <-requests; req.Action != "unsubscribe" || req.Mailbox != "id:inbox" {
		t.Fatal(req)
	}
	m.window.Close()
}
func TestFolderManagerClosesDuringWorkerAndReportsFailures(t *testing.T) {
	a, q := demoApp(t)
	a.cfg.Accounts[0].SyncAccount = "sync-mail"
	started := make(chan struct{})
	released := make(chan struct{})
	a.folderCommand = func(ctx context.Context, acc config.Account, req transport.FolderRequest) (transport.FolderResponse, error) {
		close(started)
		<-released
		return transport.FolderResponse{}, errors.New("server refused")
	}
	a.showFolderManager(false)
	m := a.folderManager
	<-started
	if !m.create.Disabled() || !a.folderBusy {
		t.Fatal("operation did not disable controls")
	}
	a.sync()
	if a.syncing {
		t.Fatal("sync overlapped folder request")
	}
	m.window.Close()
	close(released)
	pump(t, q, func() bool { return !a.folderBusy })
	if a.folderManager != nil || m.window.Canvas().Overlays().Top() != nil {
		t.Fatal("completion touched closed window")
	}
	a.folderCommand = func(context.Context, config.Account, transport.FolderRequest) (transport.FolderResponse, error) {
		return transport.FolderResponse{}, errors.New("server refused")
	}
	a.showFolderManager(false)
	m = a.folderManager
	pump(t, q, func() bool { return !a.folderBusy })
	if m.window.Canvas().Overlays().Top() == nil {
		t.Fatal("failure not reported")
	}
	m.window.Close()
}
