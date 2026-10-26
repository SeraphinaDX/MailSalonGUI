// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

// messageSelectionSnapshot survives a folder rescan. List row numbers and
// Maildir paths are not durable: new mail changes row numbers and mail flags
// may rename files between cur/ and new/.
type messageSelectionSnapshot struct {
	folderPath string
	entries    []maildir.Entry
	active     int // index into entries; -1 when the preview was not active
	offset     fyne.Position
}

func (a *App) snapshotMessageSelection() *messageSelectionSnapshot {
	if a.messageList == nil {
		return nil
	}
	ids := a.messageList.SelectedIDs()
	if len(ids) == 0 {
		return nil
	}
	snap := &messageSelectionSnapshot{
		folderPath: a.folderPath,
		active:     -1,
	}
	for _, id := range ids {
		snap.entries = append(snap.entries, a.messages[id])
		if id == a.selected {
			snap.active = len(snap.entries) - 1
		}
	}
	// The current list focus is useful if a message is still being parsed.
	if snap.active < 0 {
		for i, id := range ids {
			if id == a.messageList.selectedID {
				snap.active = i
				break
			}
		}
	}
	if a.previewScroll != nil {
		snap.offset = a.previewScroll.Offset
	}
	return snap
}

// maildirStem ignores the Maildir seen/flag suffix; the unique filename is
// preserved when mail is moved between new/ and cur/ or flags are changed.
func maildirStem(path string) string {
	name := filepath.Base(path)
	if i := strings.Index(name, ":2,"); i >= 0 {
		return name[:i]
	}
	return name
}

// restoreMatch chooses an unambiguous match. Prefer exact paths, then Maildir
// filenames (stable across read/flag updates), then a unique Message-ID if a
// sync tool has renamed the file. Never guess from subject or row position.
func restoreMatch(messages []maildir.Entry, old maildir.Entry, used map[int]bool) int {
	for i, current := range messages {
		if !used[i] && current.Path == old.Path {
			return i
		}
	}

	stem := maildirStem(old.Path)
	match, count := -1, 0
	for i, current := range messages {
		if !used[i] && maildirStem(current.Path) == stem {
			match, count = i, count+1
		}
	}
	if count == 1 {
		return match
	}
	// Ambiguous duplicate filenames should not fall through to a guessed ID.
	if count > 1 {
		return -1
	}

	if id := strings.TrimSpace(old.MessageID); id != "" {
		match, count = -1, 0
		for i, current := range messages {
			if !used[i] && strings.TrimSpace(current.MessageID) == id {
				match, count = i, count+1
			}
		}
		if count == 1 {
			return match
		}
	}
	return -1
}

func (a *App) restoreMessageSelection(snap *messageSelectionSnapshot) {
	if snap == nil || snap.folderPath != a.folderPath {
		return
	}
	used := make(map[int]bool, len(snap.entries))
	active := -1
	for i, old := range snap.entries {
		id := restoreMatch(a.messages, old, used)
		if id < 0 {
			continue // Message was deleted, moved out or filtered away.
		}
		used[id] = true
		a.messageList.selection[id] = true
		if i == snap.active {
			active = id
		}
	}
	if len(used) == 0 {
		return // Do not select a different message when the old one is gone.
	}
	if active == -1 {
		// The previously active mail disappeared; keep surviving selections
		// without restoring its scroll offset onto an unrelated message.
		for id := range used {
			if active < 0 || id < active {
				active = id
			}
		}
	} else {
		offset := snap.offset
		a.pendingPreviewOffset = &offset
	}
	a.messageList.anchor = active
	a.messageList.focusSelection(active) // Triggers the normal message preview.
	a.pendingPreviewOffset = nil
}
