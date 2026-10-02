// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/transport"
)

type folderManager struct {
	app                                     *App
	window                                  fyne.Window
	account                                 int
	cfg                                     config.Account
	rows                                    []transport.RemoteFolder
	list                                    *widget.List
	status                                  *widget.Label
	create, subscribe, unsubscribe, refresh *widget.Button
	selected                                int
	closed                                  bool
	newAfterLoad                            bool
	cancel                                  context.CancelFunc
}

func (a *App) showFolderManager(create bool) {
	if a.folderManager != nil && a.folderManager.account != a.account {
		if a.folderBusy {
			dialog.ShowInformation("Folder operation in progress", "Wait for the other account's folder operation to finish.", a.Window)
			return
		}
		a.folderManager.window.Close()
	}
	if a.folderManager != nil && !a.folderManager.closed {
		a.folderManager.window.Show()
		a.folderManager.window.RequestFocus()
		if create && !a.folderBusy {
			a.folderManager.newFolder()
		}
		return
	}
	acc := a.cfg.Accounts[a.account]
	if acc.SyncAccount == "" {
		dialog.ShowInformation("Folder management setup", "Add sync_account to this account in TOML Settings. It must match the account name in MailSalonSync.\n\nOptional: sync_config for a custom sync config path and sync_executable for the binary path. Requires MailSalonSync 0.7.0 or newer.", a.Window)
		return
	}
	if a.syncing || a.changing || a.folderBusy {
		dialog.ShowInformation("Mail operation in progress", "Wait for the current mail operation to finish, then manage folders.", a.Window)
		return
	}
	a.cancelMailDrag()
	a.hideMessageMenu()
	m := &folderManager{app: a, account: a.account, cfg: acc, selected: -1, newAfterLoad: create}
	a.folderManager = m
	m.window = a.Fyne.NewWindow("Folders — " + acc.Name + " (" + acc.SyncAccount + ")")
	m.window.Resize(fyne.NewSize(760, 540))
	m.status = widget.NewLabel("Loading server folders…")
	m.status.Wrapping = fyne.TextWrapWord
	m.list = widget.NewList(func() int { return len(m.rows) }, func() fyne.CanvasObject { return widget.NewLabel("Folder\nSubscription / sync status") }, func(id int, o fyne.CanvasObject) {
		if id < 0 || id >= len(m.rows) {
			return
		}
		f := m.rows[id]
		sub := "not subscribed"
		if f.Subscribed {
			sub = "subscribed"
		}
		sync := "not syncing locally"
		if f.Syncing {
			sync = "syncing to " + f.Local
		}
		o.(*widget.Label).SetText(f.Name + "\n" + sub + " · " + sync)
	})
	m.list.OnSelected = func(id int) { m.selected = id; m.updateButtons() }
	m.create = widget.NewButton("New folder…", m.newFolder)
	m.subscribe = widget.NewButton("Subscribe / sync locally", func() { m.changeSubscription(true) })
	m.unsubscribe = widget.NewButton("Unsubscribe…", func() { m.changeSubscription(false) })
	m.refresh = widget.NewButton("Refresh", func() { m.request(transport.FolderRequest{Action: "list"}) })
	description := widget.NewLabel("Server subscriptions and local sync status are shown separately. Cached mail is kept when you unsubscribe.")
	description.Wrapping = fyne.TextWrapWord
	m.window.SetContent(container.NewBorder(container.NewVBox(description, widget.NewSeparator()), container.NewVBox(m.status, container.NewHBox(m.create, m.subscribe, m.unsubscribe, m.refresh)), nil, nil, m.list))
	m.window.SetOnClosed(func() {
		m.closed = true
		if m.cancel != nil {
			m.cancel()
		}
		if a.folderManager == m {
			a.folderManager = nil
		}
	})
	m.updateButtons()
	showWindow(m.window)
	m.request(transport.FolderRequest{Action: "list"})
}
func (m *folderManager) updateButtons() {
	busy := m.app.folderBusy || m.app.syncing || m.app.changing || m.closed
	for _, b := range []*widget.Button{m.create, m.refresh} {
		if busy {
			b.Disable()
		} else {
			b.Enable()
		}
	}
	m.subscribe.Disable()
	m.unsubscribe.Disable()
	if !busy && m.selected >= 0 && m.selected < len(m.rows) {
		f := m.rows[m.selected]
		if f.Selectable && (!f.Subscribed || !f.Syncing) {
			m.subscribe.Enable()
		}
		if f.Subscribed || f.Syncing {
			m.unsubscribe.Enable()
		}
	}
}
func (m *folderManager) request(req transport.FolderRequest) {
	a := m.app
	if m.closed {
		return
	}
	if a.folderBusy || a.syncing || a.changing {
		m.status.SetText("Wait for the current mail operation to finish, then retry.")
		return
	}
	a.folderBusy = true
	a.cancelMailDrag()
	m.updateButtons()
	m.status.SetText("Contacting MailSalonSync…")
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Minute)
	m.cancel = cancel
	run := a.folderCommand
	if run == nil {
		run = transport.ManageFolders
	}
	go func() {
		response, err := run(ctx, m.cfg, req)
		cancel()
		a.post(func() {
			a.folderBusy = false
			m.cancel = nil
			if a.folderManager != nil {
				a.folderManager.updateButtons()
			}
			// Reload even after an error: a remote operation may have succeeded before
			// a later mapping, Maildir or refresh step failed.
			if req.Action != "list" && a.account == m.account {
				a.reload()
			}
			if m.closed {
				return
			}
			if err != nil {
				m.newAfterLoad = false
				m.status.SetText("Folder operation failed — refresh to check server state.")
				dialog.ShowError(err, m.window)
				return
			}
			m.rows = response.Folders
			m.selected = -1
			m.list.UnselectAll()
			m.list.Refresh()
			m.updateButtons()
			if req.Action == "list" {
				m.status.SetText(fmt.Sprintf("%d server folders", len(m.rows)))
			} else if req.Action == "unsubscribe" {
				m.status.SetText("Unsubscribed; local mail retained and syncing stopped for this folder.")
			} else {
				m.status.SetText("Folder mapped locally. Use Sync to download its mail.")
			}
			if m.newAfterLoad {
				m.newAfterLoad = false
				m.newFolder()
			}
		})
	}()
}
func (m *folderManager) newFolder() {
	if m.closed || m.app.folderBusy || m.app.syncing || m.app.changing {
		return
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("Folder name")
	local := widget.NewEntry()
	local.SetPlaceHolder("Automatic; optionally Projects/Work")
	options := []string{"Top level"}
	parents := map[string]string{}
	for _, f := range m.rows {
		if f.CanCreateChild {
			label := f.Name
			for n := 2; parents[label] != ""; n++ {
				label = fmt.Sprintf("%s (%d)", f.Name, n)
			}
			options = append(options, label)
			parents[label] = f.ID
		}
	}
	parent := widget.NewSelect(options, nil)
	parent.SetSelected("Top level")
	if m.selected >= 0 && m.selected < len(m.rows) {
		f := m.rows[m.selected]
		if f.CanCreateChild {
			for _, label := range options {
				if parents[label] == f.ID {
					parent.SetSelected(label)
					break
				}
			}
		}
	}
	form := dialog.NewForm("New server folder", "Create", "Cancel", []*widget.FormItem{widget.NewFormItem("Name", name), widget.NewFormItem("Parent", parent), widget.NewFormItem("Local folder", local)}, func(ok bool) {
		if !ok || m.closed {
			return
		}
		if strings.TrimSpace(name.Text) == "" {
			dialog.ShowError(fmt.Errorf("enter a folder name"), m.window)
			return
		}
		m.request(transport.FolderRequest{Action: "create", Name: name.Text, Parent: parents[parent.Selected], Local: strings.TrimSpace(local.Text)})
	}, m.window)
	form.Resize(fyne.NewSize(620, 240))
	form.Show()
	m.window.Canvas().Focus(name)
}
func (m *folderManager) changeSubscription(on bool) {
	if m.selected < 0 || m.selected >= len(m.rows) || m.closed || m.app.folderBusy {
		return
	}
	folder := m.rows[m.selected]
	if on {
		m.request(transport.FolderRequest{Action: "subscribe", Mailbox: folder.ID})
		return
	}
	dialog.ShowConfirm("Unsubscribe from folder", "Unsubscribe from "+folder.Name+" and stop syncing it locally?\n\nExisting local mail will be kept. The server folder and its messages will remain.", func(ok bool) {
		if ok && !m.closed {
			m.request(transport.FolderRequest{Action: "unsubscribe", Mailbox: folder.ID})
		}
	}, m.window)
}
