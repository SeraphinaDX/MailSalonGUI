// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/demo"
	"github.com/SeraphinaDX/MailSalonGUI/internal/drafts"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

func testApp(t *testing.T, cfg config.Config) (*App, chan func()) {
	t.Helper()
	f := test.NewApp()
	a := New(f, cfg, filepath.Join(t.TempDir(), "config.toml"), t.TempDir())
	queue := make(chan func(), 100)
	a.dispatch = func(fn func()) { queue <- fn }
	t.Cleanup(func() { a.cancel(); f.Quit() })
	return a, queue
}
func pump(t *testing.T, queue chan func(), done func() bool) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for !done() {
		select {
		case fn := <-queue:
			fn()
		case <-deadline.C:
			t.Fatal("timed out waiting for worker")
		}
	}
}
func screenshot(t *testing.T, w fyne.Window, name string) {
	t.Helper()
	dir := os.Getenv("MAILSALONGUI_SCREENSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	w.Show()
	w.Content().Refresh()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyViews(t *testing.T) {
	cfg := config.Default()
	cfg.Accounts[0].Maildir = t.TempDir()
	cfg.SyncInterval = 0
	cfg.StartupSync = false
	a, q := testApp(t, cfg)
	a.Start()
	pump(t, q, func() bool { return strings.Contains(a.status.Text, "populate") })
	a.folderList.Select(-1)
	a.messageList.Select(-1)
	a.tabs.SelectIndex(1)
	a.contacts.list.Select(-1)
	a.tabs.SelectIndex(2)
	a.calendar.list.Select(-1)
	if !a.contacts.chooser.Disabled() || !a.calendar.chooser.Disabled() {
		t.Fatal("unconfigured collections should be disabled")
	}
	if !strings.Contains(a.calendar.preview.Text, "No ") && !strings.Contains(a.calendar.preview.Text, "Add a collection") {
		t.Fatal("missing empty view guidance")
	}
	screenshot(t, a.Window, "empty-collections")
}
func TestDemoMailComposeDraftAndCollections(t *testing.T) {
	cfg, root, err := demo.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	a, q := testApp(t, cfg)
	a.Start()
	pump(t, q, func() bool { return len(a.messages) == 5 && len(a.contacts.items) == 2 && len(a.calendar.items) == 1 })
	index := 0
	for i, e := range a.messages {
		if strings.Contains(e.Subject, "project") {
			index = i
		}
	}
	a.messageList.Select(index)
	pump(t, q, func() bool { return a.parsed != nil && !a.changing && !a.messages[index].Unread })
	if len(a.parsed.Attachments) != 1 {
		t.Fatal("attachment not shown")
	}
	screenshot(t, a.Window, "mail")
	d := replyDraft(a.parsed, true, false, cfg.Accounts)
	c := a.compose(d, 0, "")
	c.to.SetText("Alex <alex@example.com>")
	if len(c.seed.MemoryAttachments) != 1 {
		t.Fatal("forward lost attachments")
	}
	if !c.save() {
		t.Fatal("draft save failed")
	}
	screenshot(t, c.window, "compose")
	saved, err := drafts.List(a.draftDir)
	if err != nil || len(saved) != 1 {
		t.Fatalf("saved drafts: %v %v", saved, err)
	}
	if saved[0].Draft.To != c.to.Text || len(saved[0].Draft.MemoryAttachments) != 1 {
		t.Fatal("saved draft lost fields")
	}
	c.close()
	reopened := a.compose(saved[0].Draft, 0, saved[0].ID)
	if reopened.subject.Text != d.Subject {
		t.Fatal("restored draft changed")
	}
	reopened.close()
	a.tabs.SelectIndex(1)
	pump(t, q, func() bool { return len(a.contacts.items) == 2 })
	a.contacts.list.Select(0)
	screenshot(t, a.Window, "contacts")
	a.tabs.SelectIndex(2)
	pump(t, q, func() bool { return len(a.calendar.items) == 1 })
	a.calendar.list.Select(0)
	screenshot(t, a.Window, "calendar")
	a.search.SetText("nothing matches this search")
	if len(a.messages) != 0 || a.selected != -1 || a.parsed != nil {
		t.Fatal("empty search left stale selection")
	}
}
func TestReplyIdentityAndRecipientHandling(t *testing.T) {
	accounts := []config.Account{{From: "Me <me@example.com>"}, {From: "Work <work@example.com>"}}
	p := &mimeutil.ParsedMessage{From: "Sender <sender@example.com>", ReplyTo: "List <list@example.com>", To: "Work <work@example.com>, Other <other@example.com>", Cc: "Sender <sender@example.com>, Me <me@example.com>", Subject: "Re: Hi", MessageID: "<original>", Body: "hello"}
	if preferredAccount(accounts, 0, p) != 1 {
		t.Fatal("reply identity not selected")
	}
	d := replyDraft(p, false, true, accounts)
	if d.To != "\"List\" <list@example.com>" {
		t.Fatal(d.To)
	}
	if strings.Contains(d.Cc, "me@example.com") || strings.Contains(d.Cc, "work@example.com") || !strings.Contains(d.Cc, "other@example.com") {
		t.Fatal(d.Cc)
	}
	if d.Subject != "Re: Hi" || d.InReplyTo != "<original>" || d.References != "<original>" {
		t.Fatal("reply headers changed")
	}
	p.Cc = "malformed address"
	if preferredAccount(accounts, 0, p) != 1 {
		t.Fatal("malformed Cc hid valid To identity")
	}
	if !strings.Contains(applySignature(d.Body, "signature"), "signature\n\nOn ") {
		t.Fatal("signature was not inserted before quotation")
	}
}
func TestConfigEditorDoesNotOverwriteValidFileWithInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := writeConfig(path, []byte(config.Example)); err != nil {
		t.Fatal(err)
	}
	if err := writeConfig(path, []byte("this is not TOML")); err == nil {
		t.Fatal("invalid TOML saved")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != config.Example {
		t.Fatal("invalid save replaced original")
	}
}
func TestStaleMessageDoesNotReplaceSearchResults(t *testing.T) {
	cfg, root, err := demo.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	a, q := testApp(t, cfg)
	a.Start()
	pump(t, q, func() bool { return len(a.messages) == 5 && len(a.contacts.items) == 2 && len(a.calendar.items) == 1 })
	a.messageList.Select(0)
	a.search.SetText("no such message")
	// Process the pending parse completion. Its selection token was invalidated.
	select {
	case fn := <-q:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("missing message result")
	}
	if a.parsed != nil || a.selected != -1 || len(a.messages) != 0 {
		t.Fatal("stale worker replaced the empty search")
	}
}

func tapConfirmation(t *testing.T, w fyne.Window) {
	t.Helper()
	var find func(fyne.CanvasObject) *widget.Button
	find = func(o fyne.CanvasObject) *widget.Button {
		if b, ok := o.(*widget.Button); ok && b.Text == "Yes" {
			return b
		}
		if popup, ok := o.(*widget.PopUp); ok {
			return find(popup.Content)
		}
		if box, ok := o.(*fyne.Container); ok {
			for _, child := range box.Objects {
				if b := find(child); b != nil {
					return b
				}
			}
		}
		return nil
	}
	if b := find(w.Canvas().Overlays().Top()); b != nil {
		test.Tap(b)
	} else {
		t.Fatal("confirmation button not found")
	}
}
func TestSendFailureRetainsCompositionAndSuccessRemovesDraft(t *testing.T) {
	cfg := config.Default()
	cfg.SyncInterval = 0
	cfg.Accounts[0].From = "Me <me@example.com>"
	cfg.Accounts[0].SendCommand = "printf 'delivery rejected' >&2; exit 1"
	a, q := testApp(t, cfg)
	c := a.compose(mimeutil.Draft{To: "friend@example.com", Subject: "Test", Body: "Hello"}, 0, "")
	if !c.save() {
		t.Fatal("save failed")
	}
	c.send()
	tapConfirmation(t, c.window)
	pump(t, q, func() bool { return !c.sending })
	if !a.composers[c] || !strings.Contains(c.status.Text, "failed") || c.body.Text != "Hello" {
		t.Fatal("send failure lost composition")
	}
	saved, err := drafts.List(a.draftDir)
	if err != nil || len(saved) != 1 {
		t.Fatal("send failure lost saved draft")
	}
	c.window.Canvas().Overlays().Top().Hide()
	delivered := filepath.Join(t.TempDir(), "message.eml")
	a.cfg.Accounts[0].SendCommand = "cat > '" + delivered + "'"
	c.send()
	tapConfirmation(t, c.window)
	pump(t, q, func() bool { return !c.sending })
	if a.composers[c] {
		t.Fatal("successful send left composer open")
	}
	raw, err := os.ReadFile(delivered)
	if err != nil {
		t.Fatal(err)
	}
	p, err := mimeutil.ParseBytes(raw)
	if err != nil || p.Body != "Hello" || p.Subject != "Test" {
		t.Fatal("send transport did not receive complete MIME message")
	}
	saved, err = drafts.List(a.draftDir)
	if err != nil || len(saved) != 0 {
		t.Fatal("successful send did not remove draft")
	}
}
func TestSyncDeduplicatesCommandsAndSkipsOverlappingRuns(t *testing.T) {
	cfg := config.Default()
	cfg.SyncInterval = 0
	cfg.Accounts[0].Maildir = t.TempDir()
	log := filepath.Join(t.TempDir(), "sync.log")
	cmd := "printf 'sync\n' >> '" + log + "'"
	cfg.Accounts[0].ReceiveCommand = cmd
	cfg.Accounts = append(cfg.Accounts, config.Account{Name: "second", From: "second@example.com", Maildir: t.TempDir(), ReceiveCommand: cmd})
	a, q := testApp(t, cfg)
	a.sync()
	a.sync()
	pump(t, q, func() bool { return !a.syncing })
	data, err := os.ReadFile(log)
	if err != nil || string(data) != "sync\n" {
		t.Fatalf("duplicated sync: %q %v", data, err)
	}
}
