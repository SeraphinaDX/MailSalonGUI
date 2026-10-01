// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/drafts"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

func calendarButton(o fyne.CanvasObject, title string) *widget.Button {
	if button, ok := o.(*widget.Button); ok && button.Text == title {
		return button
	}
	if box, ok := o.(*fyne.Container); ok {
		for _, child := range box.Objects {
			if b := calendarButton(child, title); b != nil {
				return b
			}
		}
	}
	if card, ok := o.(*widget.Card); ok {
		return calendarButton(card.Content, title)
	}
	return nil
}

func TestCalendarMailPreviewReplyAndImport(t *testing.T) {
	a, q := demoApp(t)
	a.cfg.Accounts[0].From = "Demo User <demo@example.com>"
	raw, err := os.ReadFile("../mimeutil/testdata/invitation.eml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.messages[0].Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	a.selectMessage(0)
	waitForMessage(t, a, q)
	accept := calendarButton(a.attachments, "Accept…")
	if accept == nil || accept.Disabled() {
		t.Fatal("inline invitation has no enabled Accept button")
	}
	if len(a.attachments.Objects) != 2 {
		t.Fatal("calendar event card or Save attachment missing")
	}
	screenshot(t, a.Window, "calendar-invitation")
	test.Tap(accept)
	if len(a.composers) != 1 {
		t.Fatal("Accept did not open reply draft")
	}
	var c *composer
	for composer := range a.composers {
		c = composer
	}
	if c.to.Text != "organizer@example.test" || !c.from.Disabled() || !strings.HasPrefix(c.subject.Text, "Accepted:") || c.sending {
		t.Fatal("wrong reply recipient, identity or delivery state")
	}
	d := c.snapshot()
	if d.InReplyTo != "<meeting-request@example.test>" || len(d.MemoryAttachments) != 1 || !strings.Contains(string(d.MemoryAttachments[0].Data), "PARTSTAT=ACCEPTED") {
		t.Fatal("missing reply headers or RSVP payload")
	}
	if _, err := mimeutil.Build(d); err != nil {
		t.Fatal(err)
	}
	if !c.save() {
		t.Fatal("calendar reply draft could not be saved")
	}
	savedDrafts, err := drafts.List(a.draftDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(savedDrafts) != 1 {
		t.Fatal("saved RSVP draft missing")
	}
	saved := savedDrafts[0]
	if saved.Draft.CalendarReplyAddress != "demo@example.com" || saved.Draft.MemoryAttachments[0].MIMEType != "text/calendar; charset=utf-8; method=REPLY" {
		t.Fatal("saved draft lost RSVP identity or MIME method")
	}
	c.setSending(true)
	c.setSending(false)
	if !c.from.Disabled() {
		t.Fatal("failed send enabled RSVP identity switching")
	}
	screenshot(t, c.window, "calendar-response-draft")
	// Exercise the real calendar chooser and asynchronous local save.
	before := len(a.Fyne.Driver().AllWindows())
	test.Tap(calendarButton(a.attachments, "Add to Calendar"))
	windows := a.Fyne.Driver().AllWindows()
	if len(windows) != before+1 {
		t.Fatal("import chooser did not open")
	}
	w := windows[len(windows)-1]
	button := calendarButton(w.Content(), "Add to Calendar")
	if button == nil {
		t.Fatal("no import button")
	}
	test.Tap(button)
	pump(t, q, func() bool { return strings.Contains(a.status.Text, "Saved “MailSalon community meeting”") })
	collections := a.visibleCollections(false, 0)
	var target *pim.Item
	for _, collection := range collections {
		if collection.Protocol != "caldav" {
			continue
		}
		item, err := pim.CalendarTarget(collection, "community/meeting@example.test")
		if err != nil {
			t.Fatal(err)
		}
		if item != nil {
			target = item
		}
	}
	if target == nil || strings.Contains(string(target.Data), "METHOD:") {
		t.Fatal("calendar event not imported correctly")
	}
}

func TestCalendarCardsDisableUnsupportedActions(t *testing.T) {
	a, _ := demoApp(t)
	a.cfg.Accounts[0].From = "wrong@example.test"
	p, err := mimeutil.ParseFile("../mimeutil/testdata/invitation.eml")
	if err != nil {
		t.Fatal(err)
	}
	card := a.calendarAttachment(p.Attachments[0], p, 0)
	if !calendarButton(card, "Accept…").Disabled() || calendarButton(card, "Add to Calendar").Disabled() {
		t.Fatal("wrong account allowed RSVP or prevented independent import")
	}
	att := p.Attachments[0]
	att.Data = []byte(strings.ReplaceAll(string(att.Data), "METHOD:REQUEST", "METHOD:CANCEL"))
	att.CalendarMethod = "CANCEL"
	card = a.calendarAttachment(att, p, 0)
	if !calendarButton(card, "Accept…").Disabled() || !calendarButton(card, "Add to Calendar").Disabled() {
		t.Fatal("cancellation allowed RSVP or import")
	}
	att.Data = []byte("broken calendar")
	if _, ok := a.calendarAttachment(att, p, 0).(*widget.Label); !ok {
		t.Fatal("malformed attachment did not retain readable error")
	}
}
