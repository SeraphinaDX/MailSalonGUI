// SPDX-License-Identifier: GPL-3.0-only

// Package gui owns the Fyne interface. Mutable interface state belongs to the
// Fyne event goroutine; workers capture inputs and return results through Do.
package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pgp"
	"github.com/SeraphinaDX/MailSalonGUI/internal/transport"
)

const Version = "0.1.2"

type App struct {
	Fyne                               fyne.App
	Window                             fyne.Window
	cfg                                config.Config
	configPath                         string
	ctx                                context.Context
	cancel                             context.CancelFunc
	account                            int
	folders                            []maildir.Folder
	folderList                         *widget.List
	messageList                        *messageList
	messageMenu                        *widget.PopUpMenu
	messageMenuItems                   map[string]*fyne.MenuItem
	all, messages                      []maildir.Entry
	folderPath                         string
	selected                           int
	parsed                             *mimeutil.ParsedMessage
	status, summary, headers, security *widget.Label
	body                               *widget.RichText
	attachments                        *fyne.Container
	search                             *widget.Entry
	tabs                               *container.AppTabs
	contacts, calendar                 *collectionView
	syncButton                         *widget.Button
	progress                           *widget.ProgressBarInfinite
	syncing, changing                  bool
	changingRead                       bool
	pendingDelete                      func()
	generation, reading                uint64
	composers                          map[*composer]bool
	log                                string
	draftDir                           string
	// dispatch is replaceable by synchronous dispatch in software-driver tests.
	dispatch func(func())
}

func New(f fyne.App, cfg config.Config, path, draftDir string) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{Fyne: f, cfg: cfg, configPath: path, ctx: ctx, cancel: cancel, selected: -1, composers: map[*composer]bool{}, draftDir: draftDir, dispatch: fyne.Do}
	a.account = cfg.AccountIndex(cfg.DefaultAccount)
	if a.account < 0 {
		a.account = 0
	}
	f.Settings().SetTheme(salonTheme{Theme: theme.DefaultTheme(), colors: cfg.Theme})
	a.Window = f.NewWindow("MailSalonGUI")
	a.Window.Resize(fyne.NewSize(1280, 820))
	a.status = widget.NewLabel("Ready")
	a.status.Truncation = fyne.TextTruncateEllipsis
	a.summary = widget.NewLabel("Choose a folder")
	a.headers = widget.NewLabel("Select a message to read it")
	a.headers.Wrapping = fyne.TextWrapWord
	a.security = widget.NewLabel("")
	a.security.Wrapping = fyne.TextWrapWord
	a.body = widget.NewRichTextWithText("")
	a.body.Wrapping = fyne.TextWrapWord
	a.attachments = container.NewVBox()
	a.search = widget.NewEntry()
	a.search.SetPlaceHolder("Search this folder — sender or subject")
	a.search.OnChanged = func(string) { a.filterMessages() }
	a.folderList = widget.NewList(func() int { return len(a.folders) }, func() fyne.CanvasObject { l := widget.NewLabel(""); l.Truncation = fyne.TextTruncateEllipsis; return l }, func(id widget.ListItemID, o fyne.CanvasObject) {
		if id >= 0 && id < len(a.folders) {
			o.(*widget.Label).SetText(a.folders[id].Name)
		}
	})
	a.folderList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(a.folders) {
			a.loadFolder(a.folders[id])
		}
	}
	a.messageList = newMessageList(func() int { return len(a.messages) }, func() fyne.CanvasObject {
		return newMessageRow(a.selectMessage, a.showMessageMenu)
	}, func(id int, o fyne.CanvasObject) {
		row := o.(*messageRow)
		row.id = -1
		if id < 0 || id >= len(a.messages) {
			return
		}
		row.id = id
		e := a.messages[id]
		row.subject.TextStyle = fyne.TextStyle{Bold: e.Unread}
		text := e.Subject
		if text == "" {
			text = "(no subject)"
		}
		if e.Unread {
			text = "●  " + text
		}
		row.subject.SetText(text)
		row.meta.SetText(e.From + "  ·  " + e.Date.Format("Jan 02 15:04"))
	}, a.deleteMessage)
	a.messageList.onSelected = a.openMessage
	readActions := container.NewHBox(
		widget.NewButtonWithIcon("Reply", theme.MailReplyIcon(), func() { a.reply(false, false) }),
		widget.NewButton("Reply all", func() { a.reply(false, true) }),
		widget.NewButtonWithIcon("Forward", theme.MailForwardIcon(), func() { a.reply(true, false) }),
		widget.NewButton("Read / unread", a.toggleRead),
	)
	manageActions := container.NewHBox(widget.NewButton("Archive", a.archive), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), a.deleteMessage), widget.NewButton("Copy body", func() {
		if a.parsed != nil {
			a.Window.Clipboard().SetContent(a.parsed.Body)
		}
	}), widget.NewButton("Source", a.showSource))
	preview := container.NewBorder(container.NewVBox(readActions, manageActions, widget.NewSeparator(), a.headers, a.security), container.NewVScroll(a.attachments), nil, nil, container.NewVScroll(a.body))
	reader := container.NewHSplit(container.NewBorder(a.search, a.summary, nil, nil, a.messageList), preview)
	reader.Offset = 0.40
	mailPane := container.NewHSplit(container.NewBorder(widget.NewLabelWithStyle("Folders", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, a.folderList), reader)
	mailPane.Offset = 0.17
	a.contacts = a.newCollectionView(true)
	a.calendar = a.newCollectionView(false)
	a.tabs = container.NewAppTabs(container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), mailPane), container.NewTabItemWithIcon("Contacts", theme.AccountIcon(), a.contacts.content), container.NewTabItemWithIcon("Calendar", theme.CalendarIcon(), a.calendar.content))
	a.tabs.OnSelected = func(item *container.TabItem) {
		switch item.Text {
		case "Contacts":
			a.contacts.reloadCollections()
		case "Calendar":
			a.calendar.reloadCollections()
		}
	}
	names := []string{}
	for _, acc := range cfg.Accounts {
		names = append(names, acc.Name)
	}
	accounts := widget.NewSelect(names, nil)
	accounts.SetSelectedIndex(a.account)
	accounts.OnChanged = func(name string) {
		idx := cfg.AccountIndex(name)
		if idx >= 0 && idx != a.account {
			a.account = idx
			a.reload()
			a.contacts.reloadCollections()
			a.calendar.reloadCollections()
		}
	}
	a.syncButton = widget.NewButtonWithIcon("Sync", theme.ViewRefreshIcon(), a.sync)
	brand := widget.NewLabelWithStyle("MailSalonGUI", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	toolbar := container.NewBorder(nil, nil, container.NewHBox(brand, accounts), nil, container.NewHBox(
		widget.NewButtonWithIcon("Compose", theme.MailComposeIcon(), func() { a.compose(mimeutil.Draft{}, a.account, "") }), a.syncButton,
		widget.NewButton("Reload", func() { a.reload(); a.contacts.reloadCollections(); a.calendar.reloadCollections() }), widget.NewButton("Drafts", a.showDrafts), widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), a.settings),
	))
	a.progress = widget.NewProgressBarInfinite()
	a.progress.Hide()
	a.Window.SetContent(container.NewBorder(container.NewVBox(toolbar, widget.NewSeparator()), container.NewVBox(a.progress, a.status), nil, nil, a.tabs))
	a.Window.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("File", fyne.NewMenuItem("Compose", func() { a.compose(mimeutil.Draft{}, a.account, "") }), fyne.NewMenuItem("Saved drafts", a.showDrafts), fyne.NewMenuItem("Settings", a.settings), fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Quit", a.requestClose)), fyne.NewMenu("Help", fyne.NewMenuItem("About", func() {
		dialog.ShowInformation("MailSalonGUI "+Version, "A Go/Fyne mail, contacts and calendar client.\nLocal Maildirs • External sync/send • GPL-3.0\n\nConfiguration: "+a.configPath, a.Window)
	}))))
	a.Window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { a.compose(mimeutil.Draft{}, a.account, "") })
	a.Window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { a.reply(false, false) })
	a.Window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { a.Window.Canvas().Focus(a.search); a.tabs.SelectIndex(0) })
	a.Window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF5}, func(fyne.Shortcut) { a.sync() })
	a.Window.SetCloseIntercept(a.requestClose)
	return a
}

func (a *App) Start() {
	a.reload()
	a.contacts.reloadCollections()
	a.calendar.reloadCollections()
	if a.cfg.StartupSync {
		a.sync()
	}
	if a.cfg.SyncInterval > 0 {
		go func() {
			ticker := time.NewTicker(a.cfg.SyncInterval)
			defer ticker.Stop()
			for {
				select {
				case <-a.ctx.Done():
					return
				case <-ticker.C:
					a.post(a.sync)
				}
			}
		}()
	}
}
func (a *App) post(fn func()) {
	if a.ctx.Err() == nil {
		a.dispatch(func() {
			if a.ctx.Err() == nil {
				fn()
			}
		})
	}
}
func (a *App) fail(err error) {
	if err != nil {
		a.status.SetText("Error: " + err.Error())
		dialog.ShowError(err, a.Window)
	}
}
func (a *App) clearPreview() {
	a.hideMessageMenu()
	a.reading++
	a.selected = -1
	a.parsed = nil
	a.headers.SetText("Select a message to read it")
	a.security.SetText("")
	a.body.ParseMarkdown("")
	a.attachments.Objects = nil
	a.attachments.Refresh()
}
func (a *App) reload() {
	a.generation++
	gen := a.generation
	acc := a.cfg.Accounts[a.account]
	previous := a.folderPath
	a.clearPreview()
	a.folders = nil
	a.all = nil
	a.messages = nil
	a.folderList.UnselectAll()
	a.messageList.UnselectAll()
	a.folderList.Refresh()
	a.messageList.Refresh()
	a.status.SetText("Loading folders…")
	go func() {
		folders, err := maildir.DiscoverFolders(acc.Maildir)
		a.post(func() {
			if gen != a.generation {
				return
			}
			if err != nil {
				a.status.SetText("Mailbox unavailable — open Settings to check " + acc.Maildir)
				return
			}
			a.folders = folders
			a.folderList.Refresh()
			if len(folders) == 0 {
				a.summary.SetText("No Maildir folders found")
				a.status.SetText("Run your sync tool to populate " + acc.Maildir)
				return
			}
			index := 0
			for i, f := range folders {
				if f.Path == previous {
					index = i
				}
			}
			a.folderList.Select(index)
		})
	}()
}
func (a *App) loadFolder(folder maildir.Folder) {
	a.generation++
	gen := a.generation
	a.folderPath = folder.Path
	a.clearPreview()
	a.all = nil
	a.messages = nil
	a.messageList.UnselectAll()
	a.messageList.Refresh()
	a.summary.SetText("Loading…")
	go func() {
		entries, err := maildir.Scan(folder)
		a.post(func() {
			if gen != a.generation {
				return
			}
			if err != nil {
				a.fail(err)
				a.summary.SetText("Unable to load folder")
				return
			}
			a.all = entries
			a.filterMessages()
			a.status.SetText("Ready — " + folder.Name)
		})
	}()
}
func (a *App) filterMessages() {
	oldPath := ""
	if a.selected >= 0 && a.selected < len(a.messages) {
		oldPath = a.messages[a.selected].Path
	}
	q := strings.ToLower(strings.TrimSpace(a.search.Text))
	a.messages = nil
	for _, e := range a.all {
		if q == "" || strings.Contains(strings.ToLower(e.From+" "+e.Subject), q) {
			a.messages = append(a.messages, e)
		}
	}
	a.messageList.UnselectAll()
	a.clearPreview()
	a.messageList.Refresh()
	unread := 0
	for _, e := range a.all {
		if e.Unread {
			unread++
		}
	}
	a.summary.SetText(fmt.Sprintf("%d messages · %d unread · %d shown", len(a.all), unread, len(a.messages)))
	for i, e := range a.messages {
		if oldPath != "" && e.Path == oldPath {
			a.messageList.Select(i)
			break
		}
	}
}
func (a *App) openMessage(id int) {
	if id < 0 || id >= len(a.messages) || a.changing {
		return
	}
	a.hideMessageMenu()
	a.reading++
	token := a.reading
	gen := a.generation
	acc := a.cfg.Accounts[a.account]
	e := a.messages[id]
	a.selected = id
	a.parsed = nil
	a.headers.SetText("Loading message…")
	a.body.Segments = nil
	a.body.Refresh()
	a.attachments.Objects = nil
	a.attachments.Refresh()
	// No automatic mutation while parsing: a stale result must not mark a message
	// read after the user has changed folders or selected another message.
	go func() {
		raw, err := os.ReadFile(e.Path)
		var p *mimeutil.ParsedMessage
		var info pgp.Info
		if err == nil {
			ctx, cancel := context.WithTimeout(a.ctx, 2*time.Minute)
			defer cancel()
			raw, info = pgp.ProcessIncoming(ctx, raw, pgpSettings(acc))
			p, err = mimeutil.ParseBytes(raw)
		}
		a.post(func() {
			if token != a.reading || gen != a.generation {
				return
			}
			if err != nil {
				a.fail(err)
				return
			}
			a.parsed = p
			a.headers.SetText(fmt.Sprintf("%s\nFrom: %s\nTo: %s\nCc: %s\nDate: %s", p.Subject, p.From, p.To, p.Cc, p.Date))
			a.security.SetText(info.Summary())
			a.body.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: p.Body}}
			a.body.Refresh()
			a.Window.Content().Refresh()
			for _, attachment := range p.Attachments {
				att := attachment
				a.attachments.Add(widget.NewButtonWithIcon(fmt.Sprintf("Save %s (%d bytes)", att.Filename, len(att.Data)), theme.DownloadIcon(), func() { a.saveAttachment(att, acc.DownloadDir) }))
			}
			if e.Unread {
				a.changeRead(e, true)
			}
			a.updateMessageMenu()
		})
	}()
}
func (a *App) changeRead(e maildir.Entry, read bool) {
	if a.changing {
		return
	}
	a.changing = true
	a.changingRead = true
	a.updateMessageMenu()
	gen := a.generation
	old := e.Path
	go func() {
		var err error
		if read {
			err = maildir.MarkRead(&e)
		} else {
			err = maildir.MarkUnread(&e)
		}
		a.post(func() {
			a.changing = false
			a.changingRead = false
			pending := a.pendingDelete
			a.pendingDelete = nil
			defer a.updateMessageMenu()
			if err != nil {
				a.fail(err)
				return
			}
			if gen != a.generation {
				return
			}
			for i := range a.all {
				if a.all[i].Path == old {
					a.all[i] = e
				}
			}
			for i := range a.messages {
				if a.messages[i].Path == old {
					a.messages[i] = e
				}
			}
			a.messageList.Refresh()
			unread := 0
			for _, x := range a.all {
				if x.Unread {
					unread++
				}
			}
			a.summary.SetText(fmt.Sprintf("%d messages · %d unread · %d shown", len(a.all), unread, len(a.messages)))
			if pending != nil {
				pending()
			}
		})
	}()
}
func (a *App) currentEntry() (maildir.Entry, bool) {
	if a.changing || a.selected < 0 || a.selected >= len(a.messages) {
		return maildir.Entry{}, false
	}
	return a.messages[a.selected], true
}
func (a *App) toggleRead() {
	if e, ok := a.currentEntry(); ok {
		a.changeRead(e, e.Unread)
	}
}
func (a *App) archive() {
	e, ok := a.currentEntry()
	if !ok {
		return
	}
	acc := a.cfg.Accounts[a.account]
	folder, ok := maildir.FindFolder(a.folders, acc.ArchiveFolder)
	if !ok {
		a.fail(fmt.Errorf("create/sync the %s Maildir first", acc.ArchiveFolder))
		return
	}
	a.mutateMessage(e, func() error { return maildir.Archive(e, folder) })
}
func (a *App) deleteMessage() {
	gen, reading, id := a.generation, a.reading, a.selected
	a.afterReadChange(func() {
		if gen == a.generation && reading == a.reading && id == a.selected && id == a.messageList.selectedID {
			a.confirmDeleteMessage()
		}
	})
}

// Reading unread mail renames it from new/ to cur/. Preserve a Delete request
// during that operation, then resolve the current path after the rename.
func (a *App) afterReadChange(fn func()) {
	if a.changingRead {
		a.pendingDelete = fn
		return
	}
	fn()
}

func (a *App) confirmDeleteMessage() {
	e, ok := a.currentEntry()
	if !ok {
		return
	}
	acc := a.cfg.Accounts[a.account]
	trash, found := maildir.FindFolder(a.folders, acc.TrashFolder)
	if !found {
		if strings.ContainsAny(acc.TrashFolder, "/\\") || acc.TrashFolder == ".." {
			a.fail(fmt.Errorf("trash folder needs a simple name"))
			return
		}
		trash = maildir.Folder{Name: acc.TrashFolder, Path: filepath.Join(acc.Maildir, acc.TrashFolder)}
	}
	text := "Move this message to Trash?"
	if filepath.Clean(filepath.Dir(filepath.Dir(e.Path))) == filepath.Clean(trash.Path) {
		text = "Permanently delete this message from Trash?"
	}
	gen, reading, id := a.generation, a.reading, a.selected
	dialog.ShowConfirm("Delete message", text, func(yes bool) {
		if !yes {
			return
		}
		a.afterReadChange(func() {
			if gen != a.generation || reading != a.reading || id != a.selected || id != a.messageList.selectedID {
				return
			}
			// The preview may rename the mail while confirmation is open.
			current, ok := a.currentEntry()
			if !ok {
				return
			}
			a.mutateMessage(current, func() error { return maildir.Delete(current, trash) })
		})
	}, a.Window)
}
func (a *App) mutateMessage(e maildir.Entry, fn func() error) {
	if a.changing {
		return
	}
	a.changing = true
	gen := a.generation
	go func() {
		err := fn()
		a.post(func() {
			a.changing = false
			if err != nil {
				a.fail(err)
				return
			}
			if gen == a.generation {
				a.reload()
			}
		})
	}()
}
func (a *App) reply(forward, all bool) {
	if a.parsed == nil {
		return
	}
	idx := a.account
	if !forward {
		idx = preferredAccount(a.cfg.Accounts, idx, a.parsed)
	}
	p := a.parsed
	a.compose(replyDraft(p, forward, all, a.cfg.Accounts), idx, "")
	if !forward && a.cfg.AutoAddReplyContacts {
		a.saveReplyContact(p.From, idx)
	}
}
func (a *App) sync() {
	if a.syncing {
		return
	}
	commands := []string{}
	seen := map[string]bool{}
	for _, acc := range a.cfg.Accounts {
		cmd := strings.TrimSpace(acc.ReceiveCommand)
		if cmd != "" && !seen[cmd] {
			seen[cmd] = true
			commands = append(commands, cmd)
		}
	}
	if len(commands) == 0 {
		a.status.SetText("No receive command configured — reloading local files")
		a.reload()
		return
	}
	a.syncing = true
	a.syncButton.Disable()
	a.progress.Show()
	a.progress.Start()
	a.status.SetText("Syncing all accounts…")
	go func() {
		var failures []string
		var logs []string
		for _, cmd := range commands {
			ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
			out, err := transport.Receive(ctx, cmd)
			cancel()
			logs = append(logs, cmd+"\n"+out)
			if err != nil {
				failures = append(failures, err.Error())
			}
			if a.ctx.Err() != nil {
				return
			}
		}
		a.post(func() {
			a.syncing = false
			a.syncButton.Enable()
			a.progress.Stop()
			a.progress.Hide()
			a.log = strings.Join(logs, "\n\n")
			a.reload()
			a.contacts.reloadCollections()
			a.calendar.reloadCollections()
			if len(failures) > 0 {
				dialog.ShowInformation("Sync failed", strings.Join(failures, "\n")+"\n\n"+a.log, a.Window)
			}
		})
	}()
}
func (a *App) requestClose() {
	for c := range a.composers {
		if c.sending {
			dialog.ShowInformation("Sending in progress", "Wait for delivery to finish before closing MailSalonGUI.", a.Window)
			return
		}
	}
	if len(a.composers) > 0 {
		dialog.ShowConfirm("Close MailSalonGUI?", "Save all open compositions as local drafts and quit?", func(ok bool) {
			if ok {
				for c := range a.composers {
					if !c.save() {
						return
					}
				}
				a.close()
			}
		}, a.Window)
		return
	}
	a.close()
}
func (a *App) close() {
	a.cancel()
	for c := range a.composers {
		c.window.SetCloseIntercept(nil)
		c.window.Close()
	}
	a.Window.SetCloseIntercept(nil)
	a.Window.Close()
}
func (a *App) saveAttachment(att mimeutil.Attachment, dir string) {
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			a.fail(err)
			return
		}
		if w == nil {
			return
		}
		go func() {
			_, err := w.Write(att.Data)
			ce := w.Close()
			if err == nil {
				err = ce
			}
			a.post(func() {
				if err != nil {
					a.fail(err)
				} else {
					a.status.SetText("Attachment saved")
				}
			})
		}()
	}, a.Window)
	name := filepath.Base(strings.ReplaceAll(att.Filename, "\\", "/"))
	if name == "." || name == "/" || name == "" {
		name = "attachment"
	}
	d.SetFileName(name)
	setDialogDirectory(d, dir)
	d.Show()
}
func (a *App) showSource() {
	e, ok := a.currentEntry()
	if !ok {
		return
	}
	go func() {
		raw, err := os.ReadFile(e.Path)
		a.post(func() {
			if err != nil {
				a.fail(err)
				return
			}
			w := a.Fyne.NewWindow("Message source")
			text := widget.NewMultiLineEntry()
			text.SetText(string(raw))
			text.Disable()
			w.SetContent(container.NewBorder(nil, widget.NewButton("Copy source", func() { w.Clipboard().SetContent(string(raw)) }), nil, nil, text))
			w.Resize(fyne.NewSize(800, 600))
			w.Show()
		})
	}()
}
