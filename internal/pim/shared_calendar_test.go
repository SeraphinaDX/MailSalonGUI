// SPDX-License-Identifier: GPL-3.0-only
package pim

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

func TestDatedTaskProjectionPreservesNativeSource(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VTODO\nUID:task\nSUMMARY:Deadline\nDUE:20261002T120000Z\nEND:VTODO\nEND:VCALENDAR\n")
	from := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	out, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{{Data: data, UID: "task", Title: "Deadline"}}, from, from.AddDate(0, 0, 1))
	if len(issues) != 0 || len(out) != 1 || out[0].Start.Hour() != 12 || !out[0].End.Equal(out[0].Start) {
		t.Fatal(out, issues)
	}
	if string(out[0].Item.Data) != string(data) {
		t.Fatal("task source changed")
	}
}

func TestImportSeriesWithCancelledExceptionAndEscapedUID(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VEVENT\nUID:series\\,one\nDTSTART:20261002T120000Z\nRRULE:FREQ=WEEKLY;COUNT=2\nEND:VEVENT\nBEGIN:VEVENT\nUID:series\\,one\nRECURRENCE-ID:20261009T120000Z\nSTATUS:CANCELLED\nEND:VEVENT\nEND:VCALENDAR\n")
	events, err := ParseCalendar(data, "")
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	if events[0].UID != "series,one" {
		t.Fatal("UID wasn't decoded", events[0].UID)
	}
	c := config.Collection{Protocol: "caldav", LocalDir: t.TempDir()}
	added, err := ImportCalendarIfAbsent(c, events[0])
	if err != nil || !added {
		t.Fatal(added, err)
	}
	items, err := Load(c)
	if err != nil || len(items) != 1 || !strings.Contains(string(items[0].Data), "STATUS:CANCELLED") {
		t.Fatal(items, err)
	}
}
