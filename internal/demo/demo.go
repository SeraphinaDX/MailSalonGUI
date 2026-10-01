// SPDX-License-Identifier: GPL-3.0-only

// Package demo creates an isolated, disposable mailbox for exploring the GUI.
package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

func Create() (config.Config, string, error) {
	cfg := config.Default()
	root, err := os.MkdirTemp("", "mailsalongui-demo-")
	if err != nil {
		return cfg, "", err
	}
	fail := func(err error) (config.Config, string, error) { os.RemoveAll(root); return cfg, "", err }
	mailbox := filepath.Join(root, "Maildir")
	for _, name := range []string{"INBOX", "Sent", "Archive", "Trash"} {
		if err := maildir.Ensure(filepath.Join(mailbox, name)); err != nil {
			return fail(err)
		}
	}
	cfg.Accounts = []config.Account{{Name: "demo", Maildir: mailbox, From: "Demo User <demo@example.com>", DownloadDir: filepath.Join(root, "Downloads"), TrashFolder: "Trash", ArchiveFolder: "Archive"}}
	cfg.DefaultAccount = "demo"
	cfg.StartupSync = false
	cfg.SyncInterval = 0
	cfg.AutoAddReplyContacts = false
	cfg.Theme.Background = "#14101c"
	cfg.Theme.Foreground = "#f2e5ee"
	cfg.Theme.Muted = "#b39aad"
	cfg.Theme.Title = "#f5a7ce"
	cfg.Theme.SelectedBG = "#725068"
	cfg.Theme.Border = "#594355"
	cfg.Collections = []config.Collection{{Name: "Friends", Account: "demo", Protocol: "carddav", LocalDir: filepath.Join(root, "contacts")}, {Name: "Personal calendar", Account: "demo", Protocol: "caldav", LocalDir: filepath.Join(root, "calendar")}}
	messages := []struct{ from, subject, body string }{
		{"MailSalonGUI <hello@example.com>", "Welcome to your new mail salon", "Welcome to MailSalonGUI!\n\nYour folders are on the left, messages in the middle, and the reading pane on the right. Drag either divider to make the space yours.\n\nCompose, reply, forward, manage attachments and save local drafts. Contacts and Calendar use the same local files as MailSalonSync.\n\nThis demo is completely offline. Its messages and collections live in a temporary directory and disappear when you quit.\n\nTo connect real mail, open Settings or pass your existing TOML file with -config=PATH.\n\nEnjoy your new salon!"},
		{"Alex Rivera <alex@example.com>", "Coffee and a catch-up?", "Hey!\n\nWould Thursday work for coffee? I found a lovely little place near the library.\n\nLet me know what time works.\n\nAlex"},
		{"Morgan Lee <morgan@example.com>", "A few notes for our next project", "Hi there,\n\nI attached the project notes. Could you take a look when you have a moment?\n\nThanks,\nMorgan"},
		{"Dev Community <community@example.com>", "This week's Go meetup", "Our next meetup is all about desktop applications in Go. Bring a project and share what you are building!\n\nSee you there."},
		{"Garden Club <garden@example.com>", "Autumn planting ideas", "A little inspiration for the cooler days: bulbs, herbs, and a fresh notebook for next spring's plans."},
	}
	for i, m := range messages {
		d := mimeutil.Draft{From: m.from, To: cfg.Accounts[0].From, Subject: m.subject, Body: m.body}
		if i == 2 {
			d.MemoryAttachments = []mimeutil.Attachment{{Filename: "project-notes.txt", MIMEType: "text/plain", Data: []byte("Project notes\n- Keep the interface friendly\n- Document every workflow\n")}}
		}
		raw, err := mimeutil.Build(d)
		if err != nil {
			return fail(err)
		}
		path := filepath.Join(mailbox, "INBOX", "new", fmt.Sprintf("demo-%03d", i))
		if err = os.WriteFile(path, raw, 0600); err != nil {
			return fail(err)
		}
		_ = os.Chtimes(path, time.Now().Add(-time.Duration(i)*time.Hour), time.Now().Add(-time.Duration(i)*time.Hour))
	}
	for _, fields := range [][]string{{"Alex Rivera", "alex@example.com", "+1 555 0101"}, {"Morgan Lee", "morgan@example.com", "+1 555 0102"}} {
		data, uid, err := pim.New(cfg.Collections[0], fields)
		if err != nil {
			return fail(err)
		}
		if err = pim.Save(cfg.Collections[0], nil, data, uid); err != nil {
			return fail(err)
		}
	}
	day := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	data, uid, err := pim.New(cfg.Collections[1], []string{"Coffee with Alex", day + "T10:00:00", day + "T11:00:00", "America/Toronto", "The library café"})
	if err != nil {
		return fail(err)
	}
	if err = pim.Save(cfg.Collections[1], nil, data, uid); err != nil {
		return fail(err)
	}
	return cfg, root, nil
}
