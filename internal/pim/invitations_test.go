// SPDX-License-Identifier: GPL-3.0-only
package pim_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

func invitation(t *testing.T) pim.CalendarEvent {
	t.Helper()
	p, err := mimeutil.ParseFile("../mimeutil/testdata/invitation.eml")
	if err != nil {
		t.Fatal(err)
	}
	events, err := pim.ParseCalendar(p.Attachments[0].Data, p.Attachments[0].CalendarMethod)
	if err != nil || len(events) != 1 {
		t.Fatalf("parse invite: %v", err)
	}
	return events[0]
}

func TestInvitationPreviewAndReply(t *testing.T) {
	e := invitation(t)
	if e.Start != "2026-10-02T10:00:00" || e.End != "2026-10-02T11:00:00" || e.Zone != "America/Toronto" || !e.Recurring || !strings.Contains(e.Organizer, "organizer@example.test") {
		t.Fatalf("bad preview: %+v", e)
	}
	for _, state := range []string{"ACCEPTED", "DECLINED"} {
		data, to, from, err := e.Reply("Demo User <demo@example.com>", state)
		if err != nil || to != "organizer@example.test" || from != "demo@example.com" {
			t.Fatalf("bad reply %v", err)
		}
		s := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n ", ""), "\r\n", "\n")
		for _, want := range []string{"METHOD:REPLY", "SEQUENCE:2", "UID:" + e.UID, "PARTSTAT=" + state, "BEGIN:VTIMEZONE", "CN=\"Demo: User\"", "DTSTAMP:"} {
			if !strings.Contains(s, want) {
				t.Fatalf("missing %q in reply", want)
			}
		}
		if strings.Contains(s, "other@example.test") || strings.Contains(s, "BEGIN:VALARM") || strings.Contains(s, "RSVP=") || strings.Contains(s, "NEEDS-ACTION") {
			t.Fatal("reply retained another attendee, alarm or old RSVP state")
		}
		if _, err := pim.ParseCalendar(data, "REPLY"); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, _, err := e.Reply("wrong@example.test", "ACCEPTED"); err == nil {
		t.Fatal("wrong identity accepted")
	}
	for _, method := range []string{"CANCEL", "PUBLISH", "REPLY"} {
		e.Method = method
		if _, _, _, err := e.Reply("demo@example.com", "ACCEPTED"); err == nil {
			t.Fatal("wrong method replied")
		}
	}
}

func TestCalendarGroupsExceptionsAndAllDay(t *testing.T) {
	raw := "BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VEVENT\nUID:series\nRECURRENCE-ID:20261009T140000Z\nDTSTART:20261009T150000Z\nSUMMARY:Changed instance\nEND:VEVENT\nBEGIN:VEVENT\nUID:series\nDTSTART:20261002T140000Z\nSUMMARY:Series\nRRULE:FREQ=WEEKLY\nEND:VEVENT\nBEGIN:VEVENT\nUID:day\nDTSTART;VALUE=DATE:20261003\nDTEND;VALUE=DATE:20261004\nSUMMARY:Holiday\nEND:VEVENT\nEND:VCALENDAR\n"
	events, err := pim.ParseCalendar([]byte(raw), "")
	if err != nil || len(events) != 2 {
		t.Fatalf("events: %v", err)
	}
	if events[0].Title != "Series" || events[0].Zone != "UTC" || !events[0].Recurring || !events[1].AllDay || events[1].Start != "2026-10-03" {
		t.Fatalf("bad grouping: %+v", events)
	}
	data, err := events[0].ImportData()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "BEGIN:VEVENT") != 2 || strings.Contains(string(data), "UID:day") {
		t.Fatal("lost exception or combined different UIDs")
	}
	for _, raw := range []string{"broken", "BEGIN:VCALENDAR\nVERSION:2.0\nEND:VEVENT", "BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VEVENT\nSUMMARY:No UID\nEND:VEVENT\nEND:VCALENDAR"} {
		if _, err := pim.ParseCalendar([]byte(raw), ""); err == nil {
			t.Fatal("malformed input accepted")
		}
	}
	e := invitation(t)
	if _, err := pim.ParseCalendar(e.Data, "CANCEL"); err == nil {
		t.Fatal("conflicting MIME method accepted")
	}
}

func TestImportCalendarPreservesSourceAndGuardsUpdates(t *testing.T) {
	c := config.Collection{Protocol: "caldav", LocalDir: t.TempDir()}
	e := invitation(t)
	if err := pim.ImportCalendar(c, e, nil); err != nil {
		t.Fatal(err)
	}
	items, err := pim.Load(c)
	if err != nil || len(items) != 1 {
		t.Fatalf("load: %v", err)
	}
	data := string(items[0].Data)
	for _, want := range []string{"BEGIN:VTIMEZONE", "RRULE:", "BEGIN:VALARM", "X-EXAMPLE:preserve this", "UID:" + e.UID} {
		if !strings.Contains(data, want) {
			t.Fatalf("import lost %s", want)
		}
	}
	if strings.Contains(data, "METHOD:") {
		t.Fatal("CalDAV import retained METHOD")
	}
	if filepath.Dir(items[0].Path) != c.LocalDir {
		t.Fatal("unsafe UID escaped directory")
	}
	if err := pim.ImportCalendar(c, e, nil); err == nil {
		t.Fatal("duplicate UID silently imported")
	}
	// Sync may use an opaque filename; replacement must preserve it.
	opaque := filepath.Join(c.LocalDir, "server-resource.ics")
	if err := os.Rename(items[0].Path, opaque); err != nil {
		t.Fatal(err)
	}
	original, err := pim.CalendarTarget(c, e.UID)
	if err != nil || original == nil || original.Path != opaque {
		t.Fatalf("UID lookup: %v", err)
	}
	if err := pim.ImportCalendar(c, e, original); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opaque, []byte(strings.ReplaceAll(data, "Community room", "Remote room")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := pim.ImportCalendar(c, e, original); err == nil {
		t.Fatal("stale confirmation overwrote sync changes")
	}
	if err := os.WriteFile(filepath.Join(c.LocalDir, ".mss-lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := pim.ImportCalendar(c, e, nil); err == nil {
		t.Fatal("ignored sync lock")
	}
	if err := pim.ImportCalendar(config.Collection{Protocol: "jmap-calendars", LocalDir: t.TempDir()}, e, nil); err == nil {
		t.Fatal("lossy JMAP import accepted")
	}
}

func TestInstanceOnlyUpdateCannotEraseSeries(t *testing.T) {
	c := config.Collection{Protocol: "caldav", LocalDir: t.TempDir()}
	e := invitation(t)
	if err := pim.ImportCalendar(c, e, nil); err != nil {
		t.Fatal(err)
	}
	original, _ := pim.CalendarTarget(c, e.UID)
	data := strings.Replace(string(e.Data), "SUMMARY:MailSalon", "RECURRENCE-ID;TZID=America/Toronto:20261009T100000\r\nSUMMARY:MailSalon", 1)
	events, err := pim.ParseCalendar([]byte(data), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := pim.ImportCalendar(c, events[0], original); err == nil {
		t.Fatal("partial update erased full recurring series")
	}
	current, _ := os.ReadFile(original.Path)
	if string(current) != string(original.Data) {
		t.Fatal("failed import changed existing calendar")
	}
}

func TestRecurringReplyRetainsInstanceIdentity(t *testing.T) {
	e := invitation(t)
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(string(e.Data), "\r\n ", ""), "\r\n", "\n")), "\n")
	start, end := -1, -1
	for i, line := range lines {
		if line == "BEGIN:VEVENT" {
			start = i
		}
		if line == "END:VEVENT" {
			end = i
		}
	}
	exception := append([]string(nil), lines[start:end+1]...)
	exception = append(exception[:1], append([]string{"RECURRENCE-ID;TZID=America/Toronto:20261009T100000"}, exception[1:]...)...)
	lines = append(lines[:end+1], append(exception, lines[end+1:]...)...)
	events, err := pim.ParseCalendar([]byte(strings.Join(lines, "\r\n")+"\r\n"), "REQUEST")
	if err != nil {
		t.Fatal(err)
	}
	data, _, _, err := events[0].Reply("demo@example.com", "DECLINED")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n ", ""), "\r\n", "\n")
	if strings.Count(s, "BEGIN:VEVENT") != 2 || strings.Count(s, "PARTSTAT=DECLINED") != 2 || !strings.Contains(s, "RECURRENCE-ID;TZID=America/Toronto:20261009T100000") {
		t.Fatal("recurrence reply lost event identity or status")
	}
	if _, err := pim.ParseCalendar(append([]byte{0xef, 0xbb, 0xbf}, data...), "REPLY"); err != nil {
		t.Fatal(err)
	}
}
