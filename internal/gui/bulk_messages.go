// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/dialog"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func (a *App) selectedEntries() []maildir.Entry {
	var entries []maildir.Entry
	for _, id := range a.messageList.SelectedIDs() {
		entries = append(entries, a.messages[id])
	}
	return entries
}

func (a *App) updateSummary() {
	unread := 0
	for _, e := range a.all {
		if e.Unread {
			unread++
		}
	}
	a.summary.SetText(fmt.Sprintf("%d messages · %d unread · %d shown · %d selected", len(a.all), unread, len(a.messages), len(a.messageList.selection)))
}

// Resolve entries only after automatic mark-read finishes, because Maildir
// renames change their paths. A changed view or selection cancels queued work.
func (a *App) withSelection(fn func([]maildir.Entry)) {
	gen, revision := a.generation, a.messageList.revision
	a.afterReadChange(func() {
		if a.changing || gen != a.generation || revision != a.messageList.revision {
			return
		}
		if entries := a.selectedEntries(); len(entries) > 0 {
			fn(entries)
		}
	})
}

func (a *App) afterReadChange(fn func()) {
	if a.changingRead {
		a.pendingDelete = fn
		return
	}
	fn()
}

func (a *App) toggleRead() {
	a.withSelection(func(entries []maildir.Entry) {
		read := false
		for _, e := range entries {
			if e.Unread {
				read = true
				break
			}
		}
		a.changeSelectedRead(entries, read)
	})
}

func (a *App) markSelectedRead(read bool) {
	a.withSelection(func(entries []maildir.Entry) { a.changeSelectedRead(entries, read) })
}

func (a *App) changeSelectedRead(entries []maildir.Entry, read bool) {
	label := "Marked unread"
	if read {
		label = "Marked read"
	}
	a.mutateMessages(entries, label, false, func(e *maildir.Entry) error {
		if read {
			return maildir.MarkRead(e)
		}
		return maildir.MarkUnread(e)
	})
}

func (a *App) archive() {
	a.withSelection(func(entries []maildir.Entry) {
		acc := a.cfg.Accounts[a.account]
		folder, ok := maildir.FindFolder(a.folders, acc.ArchiveFolder)
		if !ok {
			a.fail(fmt.Errorf("create/sync the %s Maildir first", acc.ArchiveFolder))
			return
		}
		a.mutateMessages(entries, "Archived", true, func(e *maildir.Entry) error { return maildir.Archive(*e, folder) })
	})
}

func (a *App) deleteMessage() {
	a.withSelection(a.confirmDeleteMessages)
}

func (a *App) confirmDeleteMessages(entries []maildir.Entry) {
	acc := a.cfg.Accounts[a.account]
	trash, found := maildir.FindFolder(a.folders, acc.TrashFolder)
	if !found {
		if strings.ContainsAny(acc.TrashFolder, "/\\") || acc.TrashFolder == ".." {
			a.fail(fmt.Errorf("trash folder needs a simple name"))
			return
		}
		trash = maildir.Folder{Name: acc.TrashFolder, Path: filepath.Join(acc.Maildir, acc.TrashFolder)}
	}
	count := len(entries)
	text := fmt.Sprintf("Move %d selected messages to Trash?", count)
	if count == 1 {
		text = "Move this message to Trash?"
	}
	if filepath.Clean(filepath.Dir(filepath.Dir(entries[0].Path))) == filepath.Clean(trash.Path) {
		text = fmt.Sprintf("Permanently delete %d selected messages from Trash?", count)
		if count == 1 {
			text = "Permanently delete this message from Trash?"
		}
	}
	gen, revision := a.generation, a.messageList.revision
	dialog.ShowConfirm("Delete messages", text, func(yes bool) {
		if !yes || gen != a.generation || revision != a.messageList.revision {
			return
		}
		a.withSelection(func(current []maildir.Entry) {
			a.mutateMessages(current, "Deleted", true, func(e *maildir.Entry) error { return maildir.Delete(*e, trash) })
		})
	}, a.Window)
}

// One worker owns an immutable snapshot. Successful operations stay committed
// if another message fails; report all failures and refresh the actual Maildir.
func (a *App) mutateMessages(entries []maildir.Entry, label string, moved bool, fn func(*maildir.Entry) error) {
	if a.changing {
		return
	}
	a.cancelMailDrag()
	a.changing = true
	a.reading++ // Pending parses must not undo an explicit mark-unread operation.
	a.updateMessageMenu()
	a.status.SetText(fmt.Sprintf("Updating %d messages…", len(entries)))
	gen, account := a.generation, a.account
	go func() {
		updated := make(map[string]maildir.Entry)
		var failures []error
		for _, entry := range entries {
			if err := a.ctx.Err(); err != nil {
				return
			}
			old := entry.Path
			if err := fn(&entry); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", entry.Subject, err))
			} else {
				updated[old] = entry
			}
		}
		a.post(func() {
			a.changing = false
			if gen == a.generation {
				if moved || len(failures) > 0 {
					a.reload()
				} else {
					for i, e := range a.all {
						if replacement, ok := updated[e.Path]; ok {
							a.all[i] = replacement
						}
					}
					for i, e := range a.messages {
						if replacement, ok := updated[e.Path]; ok {
							a.messages[i] = replacement
						}
					}
					a.messageList.Refresh()
					a.updateSummary()
					if a.parsed == nil && a.messageList.selectedID >= 0 {
						a.loadMessage(a.messageList.selectedID, false)
					}
				}
				a.status.SetText(fmt.Sprintf("%s %d of %d messages", label, len(updated), len(entries)))
			} else if account == a.account {
				// A folder switch during the worker must keep that folder active.
				// It may be the batch destination, scanned before the moves finished.
				for _, folder := range a.folders {
					if folder.Path == a.folderPath {
						a.loadFolder(folder)
						break
					}
				}
			}
			a.updateMessageMenu()
			if len(failures) > 0 {
				a.fail(fmt.Errorf("%s %d of %d messages; %d failed:\n%w", label, len(updated), len(entries), len(failures), errors.Join(failures...)))
			}
		})
	}()
}
