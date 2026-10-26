// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func TestRestoreMatchUsesStableMailIdentity(t *testing.T) {
	old := maildir.Entry{Path: "/inbox/new/unique-123", MessageID: "<first@example.com>"}
	for _, tc := range []struct {
		name string
		path string
		id   string
	}{
		{"same path", old.Path, old.MessageID},
		{"moved to cur and marked seen", "/inbox/cur/unique-123:2,S", old.MessageID},
		{"flag change", "/inbox/cur/unique-123:2,FS", old.MessageID},
		{"fully renamed by sync tool", "/inbox/cur/renamed-456:2,S", old.MessageID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := []maildir.Entry{{Path: tc.path, MessageID: tc.id}}
			if got := restoreMatch(current, old, nil); got != 0 {
				t.Fatalf("restoreMatch() = %d, want 0", got)
			}
		})
	}
	if got := restoreMatch([]maildir.Entry{
		{Path: "/inbox/cur/one:2,S", MessageID: old.MessageID},
		{Path: "/inbox/cur/two:2,S", MessageID: old.MessageID},
	}, old, nil); got != -1 {
		t.Fatalf("ambiguous Message-ID picked wrong mail: %d", got)
	}
	if got := restoreMatch([]maildir.Entry{
		{Path: "/inbox/cur/else:2,S", MessageID: "<different@example.com>"},
	}, old, nil); got != -1 {
		t.Fatalf("missing message picked unrelated mail: %d", got)
	}
	if got := restoreMatch([]maildir.Entry{
		{Path: "/inbox/cur/unique-123:2,S", MessageID: old.MessageID},
	}, old, map[int]bool{0:true}); got != -1 {
		t.Fatalf("already-used message was matched twice: %d", got)
	}
}

func TestGUIRefreshRestoresMessageAfterMaildirRename(t *testing.T) {
	for _, tt := range []struct{
		name string
		rename func(string) string
	}{
		{
			name: "Maildir flags changed",
			rename: func(path string) string {
				return filepath.Join(filepath.Dir(path), maildirStem(path)+":2,FS")
			},
		},
		{
			name: "sync tool rewrote filename",
			rename: func(path string) string {
				return filepath.Join(filepath.Dir(path), "server-renamed-456:2,S")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, q := demoApp(t)
			a.messageList.Select(0)
			waitForMessage(t, a, q)
			old := a.messages[a.selected]
			if old.MessageID == "" {
				t.Fatal("demo mail missing Message-ID")
			}
			originalText := a.parsed.DisplayText()
			newPath := tt.rename(old.Path)
			if err := os.Rename(old.Path, newPath); err != nil {
				t.Fatalf("rename mail to simulate sync: %v", err)
			}
			a.reload()
			pump(t, q, func() bool {
				return a.parsed != nil && a.selected >= 0 &&
					a.messages[a.selected].Path == newPath && !a.changing
			})
			if a.parsed.DisplayText() != originalText || !a.messageList.selection[a.selected] {
				t.Fatal("reloaded message or selection was lost")
			}
		})
	}
}

func TestGUISyncPreservesActiveMessageAndMultipleSelection(t *testing.T) {
	a, q := demoApp(t)
	a.cfg.Accounts[0].ReceiveCommand = "echo synced"
	a.messageList.Select(0)
	waitForMessage(t, a, q)
	first := a.messages[a.selected].MessageID
	a.messageList.SelectWithModifiers(1, fyne.KeyModifierControl)
	waitForMessage(t, a, q)
	active := a.messages[a.selected].MessageID
	if len(a.messageList.SelectedIDs()) != 2 {
		t.Fatal("setup failed: need two selected messages")
	}
	a.sync()
	pump(t, q, func() bool {
		if a.syncing || a.parsed == nil || a.selected < 0 || a.messages[a.selected].MessageID != active {
			return false
		}
		return len(a.messageList.SelectedIDs()) == 2
	})
	seen := map[string]bool{}
	for _, e := range a.selectedEntries() {
		seen[e.MessageID] = true
	}
	if !seen[first] || !seen[active] {
		t.Fatalf("sync lost selection: %#v", seen)
	}
}

func TestGUIRefreshDoesNotSelectDifferentMessageIfDeleted(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	waitForMessage(t, a, q)
	path := a.messages[a.selected].Path
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	a.reload()
	pump(t, q, func() bool { return len(a.messages) == 4 })
	if a.parsed != nil || len(a.messageList.SelectedIDs()) != 0 || a.selected != -1 {
		t.Fatalf("deleted mail should not be replaced by unrelated mail: selected=%d", a.selected)
	}
}

func TestGUIRefreshMaintainsFolderAndReaderView(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	waitForMessage(t, a, q)
	folder, subject := a.folderPath, a.parsed.Subject
	a.formattedView.SetChecked(true)
	a.reload()
	pump(t, q, func() bool {
		return a.parsed != nil && a.parsed.Subject == subject &&
			a.selected >= 0 && len(a.messageList.SelectedIDs()) == 1
	})
	if a.folderPath != folder || !a.formattedView.Checked || !a.body.Visible() {
		t.Fatalf("sync did not preserve folder or formatted reader mode")
	}
	if !strings.Contains(a.headers.Text, subject) {
		t.Fatal("selected message preview was not restored")
	}
}

func TestGUIRefreshPreservesMessageWhenNewMailReordersList(t *testing.T) {
	a, q := demoApp(t)
	a.messageList.Select(0)
	waitForMessage(t, a, q)
	original := a.messages[a.selected]
	folder := a.folderPath

	// The new mail sorts above the currently selected message, so its old row
	// number would point to a different email after a scan.
	raw := "From: Other <other@example.com>\r\n" +
		"To: Me <me@example.com>\r\n" +
		"Subject: New after sync\r\n" +
		"Message-ID: <new-after-sync@example.com>\r\n" +
		"Date: " + time.Now().Add(24*time.Hour).Format(time.RFC1123Z) + "\r\n" +
		"\r\nnew message\r\n"
	path := filepath.Join(folder, "cur", "fresh-new-mail:2,S")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	a.reload()
	pump(t, q, func() bool {
		return len(a.messages) == 6 && a.parsed != nil && a.selected >= 0 &&
			a.messages[a.selected].MessageID == original.MessageID
	})
	if a.selected == 0 {
		t.Fatal("mail order did not change as expected")
	}
	if a.messages[a.selected].Path != original.Path || !a.messageList.selection[a.selected] {
		t.Fatal("refresh lost the original message after new mail was inserted")
	}
}
