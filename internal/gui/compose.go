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
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/drafts"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pgp"
	"github.com/SeraphinaDX/MailSalonGUI/internal/transport"
)

type composer struct {
	app                                  *App
	window                               fyne.Window
	from                                 *widget.Select
	to, cc, bcc, subject, body           *widget.Entry
	sign, encrypt                        *widget.Check
	status                               *widget.Label
	attachBox                            *fyne.Container
	sendButton, saveButton, attachButton *widget.Button
	seed                                 mimeutil.Draft
	account                              int
	draftID                              string
	sending                              bool
}

func (a *App) compose(d mimeutil.Draft, index int, id string) *composer {
	if id != "" {
		for c := range a.composers {
			if c.draftID == id {
				c.window.RequestFocus()
				return c
			}
		}
	}
	c := &composer{app: a, seed: d, account: index, draftID: id}
	c.window = a.Fyne.NewWindow("Compose — MailSalonGUI")
	c.window.Resize(fyne.NewSize(900, 730))
	a.composers[c] = true
	c.to = widget.NewEntry()
	c.to.SetText(d.To)
	c.to.SetPlaceHolder("name@example.com, Another Name <other@example.com>")
	c.cc = widget.NewEntry()
	c.cc.SetText(d.Cc)
	c.bcc = widget.NewEntry()
	c.bcc.SetText(d.Bcc)
	c.subject = widget.NewEntry()
	c.subject.SetText(d.Subject)
	c.body = widget.NewMultiLineEntry()
	c.body.Wrapping = fyne.TextWrapWord
	c.body.SetText(d.Body)
	c.status = widget.NewLabel("Ready")
	c.status.Wrapping = fyne.TextWrapWord
	c.sign = widget.NewCheck("Sign", nil)
	c.encrypt = widget.NewCheck("Encrypt", nil)
	names := []string{}
	for _, acc := range a.cfg.Accounts {
		names = append(names, acc.Name+" — "+acc.From)
	}
	c.from = widget.NewSelect(names, func(label string) {
		for i, name := range names {
			if name == label {
				c.account = i
				c.updatePGP()
				break
			}
		}
	})
	c.from.SetSelectedIndex(index)
	c.updatePGP()
	c.attachBox = container.NewVBox()
	c.refreshAttachments()
	recipient := func(label string, e *widget.Entry) *widget.FormItem {
		return widget.NewFormItem(label, container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.AccountIcon(), func() { a.pickContact(c, e) }), e))
	}
	fields := widget.NewForm(widget.NewFormItem("From", c.from), recipient("To", c.to), recipient("Cc", c.cc), recipient("Bcc", c.bcc), widget.NewFormItem("Subject", c.subject))
	c.sendButton = widget.NewButtonWithIcon("Send", theme.MailSendIcon(), c.send)
	c.sendButton.Importance = widget.HighImportance
	c.saveButton = widget.NewButtonWithIcon("Save draft", theme.DocumentSaveIcon(), func() { c.save() })
	c.attachButton = widget.NewButtonWithIcon("Attach file", theme.MailAttachmentIcon(), c.attach)
	tools := container.NewHBox(c.sendButton, c.saveButton, c.attachButton, c.sign, c.encrypt)
	c.window.SetContent(container.NewBorder(container.NewVBox(tools, fields), container.NewVBox(c.attachBox, c.status), nil, nil, c.body))
	c.window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
		if !c.sending {
			c.save()
		}
	})
	c.window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyReturn, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { c.send() })
	c.window.SetCloseIntercept(c.requestClose)
	showWindow(c.window)
	if d.InReplyTo != "" {
		c.window.Canvas().Focus(c.body)
	} else {
		c.window.Canvas().Focus(c.to)
	}
	return c
}
func (c *composer) updatePGP() {
	g := c.app.cfg.Accounts[c.account].GPG
	c.sign.SetChecked(g.Enabled && g.AutoSign)
	c.encrypt.SetChecked(g.Enabled && g.AutoEncrypt)
	if g.Enabled {
		c.sign.Enable()
		c.encrypt.Enable()
	} else {
		c.sign.Disable()
		c.encrypt.Disable()
	}
}
func (c *composer) snapshot() mimeutil.Draft {
	d := c.seed
	d.From = c.app.cfg.Accounts[c.account].From
	d.To = c.to.Text
	d.Cc = c.cc.Text
	d.Bcc = c.bcc.Text
	d.Subject = c.subject.Text
	d.Body = c.body.Text
	d.Attachments = append([]string(nil), c.seed.Attachments...)
	d.MemoryAttachments = append([]mimeutil.Attachment(nil), c.seed.MemoryAttachments...)
	return d
}
func (c *composer) save() bool {
	if c.sending {
		return false
	}
	s, err := drafts.Save(c.app.draftDir, drafts.Saved{ID: c.draftID, Account: c.app.cfg.Accounts[c.account].Name, Draft: c.snapshot(), Sign: c.sign.Checked, Encrypt: c.encrypt.Checked})
	if err != nil {
		dialog.ShowError(err, c.window)
		return false
	}
	c.draftID = s.ID
	c.status.SetText("Draft saved locally — " + s.Updated.Format("15:04:05"))
	return true
}
func (c *composer) requestClose() {
	if c.sending {
		dialog.ShowInformation("Sending in progress", "Wait for delivery to finish before closing this message.", c.window)
		return
	}
	var d dialog.Dialog
	close := func() { d.Hide(); c.close() }
	content := container.NewVBox(widget.NewLabel("Save this composition before closing?"), container.NewHBox(
		widget.NewButton("Save and close", func() {
			if c.save() {
				close()
			}
		}), widget.NewButton("Discard", func() {
			if c.draftID != "" {
				if err := drafts.Delete(c.app.draftDir, c.draftID); err != nil {
					dialog.ShowError(err, c.window)
					return
				}
			}
			close()
		}),
	))
	d = dialog.NewCustom("Close composition", "Keep editing", content, c.window)
	d.Show()
}
func (c *composer) close() {
	delete(c.app.composers, c)
	c.window.SetCloseIntercept(nil)
	c.window.Close()
}
func (c *composer) refreshAttachments() {
	c.attachBox.Objects = nil
	for i, path := range c.seed.Attachments {
		index := i
		c.attachBox.Add(container.NewBorder(nil, nil, nil, widget.NewButton("Remove", func() {
			if c.sending {
				return
			}
			c.seed.Attachments = append(c.seed.Attachments[:index], c.seed.Attachments[index+1:]...)
			c.refreshAttachments()
		}), widget.NewLabel(path)))
	}
	for i, att := range c.seed.MemoryAttachments {
		index := i
		c.attachBox.Add(container.NewBorder(nil, nil, nil, widget.NewButton("Remove", func() {
			if c.sending {
				return
			}
			c.seed.MemoryAttachments = append(c.seed.MemoryAttachments[:index], c.seed.MemoryAttachments[index+1:]...)
			c.refreshAttachments()
		}), widget.NewLabel(att.Filename+" (forwarded)")))
	}
	c.attachBox.Refresh()
}
func (c *composer) attach() {
	if c.sending {
		return
	}
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, c.window)
			return
		}
		if r == nil {
			return
		}
		path := r.URI().Path()
		r.Close()
		for _, p := range c.seed.Attachments {
			if p == path {
				return
			}
		}
		c.seed.Attachments = append(c.seed.Attachments, path)
		c.refreshAttachments()
	}, c.window)
	d.Show()
}
func (c *composer) setSending(b bool) {
	c.sending = b
	for _, e := range []*widget.Entry{c.to, c.cc, c.bcc, c.subject, c.body} {
		if b {
			e.Disable()
		} else {
			e.Enable()
		}
	}
	if b {
		c.from.Disable()
		c.sendButton.Disable()
		c.saveButton.Disable()
		c.attachButton.Disable()
		c.sign.Disable()
		c.encrypt.Disable()
	} else {
		c.from.Enable()
		c.sendButton.Enable()
		c.saveButton.Enable()
		c.attachButton.Enable()
		if c.app.cfg.Accounts[c.account].GPG.Enabled {
			c.sign.Enable()
			c.encrypt.Enable()
		}
	}
}
func (c *composer) send() {
	if c.sending {
		return
	}
	acc := c.app.cfg.Accounts[c.account]
	if strings.TrimSpace(acc.SendCommand) == "" {
		dialog.ShowInformation("Sending is not configured", "Set this account's send command in Settings first.", c.window)
		return
	}
	d := c.snapshot()
	sign, encrypt := c.sign.Checked, c.encrypt.Checked
	dialog.ShowConfirm("Send message?", "Send “"+d.Subject+"” from "+acc.From+"?", func(ok bool) {
		if !ok || c.sending {
			return
		}
		c.setSending(true)
		c.status.SetText("Sending… Please wait for delivery to finish.")
		go func() {
			signature, err := config.ReadSignature(acc)
			var raw []byte
			var out string
			if err == nil {
				d.Body = applySignature(d.Body, signature)
				raw, err = mimeutil.Build(d)
			}
			ctx, cancel := context.WithTimeout(c.app.ctx, 5*time.Minute)
			defer cancel()
			if err == nil && (sign || encrypt) {
				raw, err = pgp.ProtectOutgoing(ctx, raw, pgp.OutgoingOptions{Settings: pgpSettings(acc), Sign: sign, Encrypt: encrypt})
			}
			if err == nil {
				out, err = transport.Send(ctx, acc.SendCommand, raw)
			}
			c.app.post(func() {
				c.setSending(false)
				if err != nil {
					c.status.SetText("Delivery failed — composition kept")
					dialog.ShowError(fmt.Errorf("%w\n%s", err, out), c.window)
					return
				}
				// Delivery success is final even if local cleanup fails; never offer a
				// retry that could send the same message twice.
				if err := drafts.Delete(c.app.draftDir, c.draftID); err != nil {
					dialog.ShowInformation("Sent", "Message sent. Could not remove the saved draft: "+err.Error()+"\nDelete that draft without sending it again.", c.app.Window)
				}
				c.app.status.SetText("Message sent from " + acc.Name)
				c.close()
			})
		}()
	}, c.window)
}
func (a *App) showDrafts() {
	saved, err := drafts.List(a.draftDir)
	if err != nil {
		a.fail(err)
		return
	}
	w := a.Fyne.NewWindow("Local drafts — MailSalonGUI")
	w.Resize(fyne.NewSize(650, 430))
	selected := -1
	list := widget.NewList(func() int { return len(saved) }, func() fyne.CanvasObject { l := widget.NewLabel(""); l.Truncation = fyne.TextTruncateEllipsis; return l }, func(id int, o fyne.CanvasObject) {
		s := saved[id]
		subject := s.Draft.Subject
		if subject == "" {
			subject = "(no subject)"
		}
		o.(*widget.Label).SetText(s.Account + " · " + subject + " · " + s.Updated.Format("Jan 02 15:04"))
	})
	list.OnSelected = func(id int) { selected = id }
	open := widget.NewButton("Open", func() {
		if selected < 0 || selected >= len(saved) {
			return
		}
		s := saved[selected]
		idx := a.cfg.AccountIndex(s.Account)
		if idx < 0 {
			dialog.ShowInformation("Account unavailable", "The account for this draft is no longer configured.", w)
			return
		}
		c := a.compose(s.Draft, idx, s.ID)
		if a.cfg.Accounts[idx].GPG.Enabled {
			c.sign.SetChecked(s.Sign)
			c.encrypt.SetChecked(s.Encrypt)
		}
		w.Close()
	})
	remove := widget.NewButton("Delete", func() {
		if selected < 0 || selected >= len(saved) {
			return
		}
		s := saved[selected]
		for c := range a.composers {
			if c.draftID == s.ID {
				dialog.ShowInformation("Draft is open", "Close this composition before deleting its saved draft.", w)
				return
			}
		}
		dialog.ShowConfirm("Delete draft?", "Remove this saved local draft?", func(ok bool) {
			if !ok {
				return
			}
			if err := drafts.Delete(a.draftDir, s.ID); err != nil {
				dialog.ShowError(err, w)
				return
			}
			saved, err = drafts.List(a.draftDir)
			if err != nil {
				dialog.ShowError(err, w)
			}
			selected = -1
			list.UnselectAll()
			list.Refresh()
		}, w)
	})
	empty := widget.NewLabel("Saved drafts stay local until you send them.")
	if len(saved) == 0 {
		empty.SetText("No saved drafts yet.")
	}
	w.SetContent(container.NewBorder(empty, container.NewHBox(open, remove), nil, nil, list))
	showWindow(w)
}
