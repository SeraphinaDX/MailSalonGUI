// SPDX-License-Identifier: GPL-3.0-only
package pim

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

func calendarTestItem(t *testing.T, body string) Item {
	t.Helper()
	data := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n" + body + "END:VCALENDAR\n")
	i, err := Parse(config.Collection{Protocol: "caldav"}, data)
	if err != nil {
		t.Fatal(err)
	}
	i.Path = "test.ics"
	return i
}
func testLocal(t *testing.T, zone, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatal(err)
	}
	dt, err := time.ParseInLocation("2006-01-02T15:04:05", value, loc)
	if err != nil {
		t.Fatal(err)
	}
	return dt
}

func TestRecurrenceExceptionsDSTAndDisplayTimezone(t *testing.T) {
	item := calendarTestItem(t, "BEGIN:VEVENT\nUID:meeting\nSUMMARY:Weekly meeting\nDTSTART;TZID=America/Toronto:20261025T090000\nDTEND;TZID=America/Toronto:20261025T100000\nRRULE:FREQ=WEEKLY;COUNT=4\nEXDATE;TZID=America/Toronto:20261108T090000\nRDATE;TZID=America/Toronto:20261103T090000\nEND:VEVENT\nBEGIN:VEVENT\nUID:meeting\nRECURRENCE-ID;TZID=America/Toronto:20261101T090000\nDTSTART;TZID=America/Toronto:20261102T130000\nDTEND;TZID=America/Toronto:20261102T143000\nSUMMARY:Moved meeting\nEND:VEVENT\n")
	from := testLocal(t, "UTC", "2026-10-25T00:00:00")
	to := from.AddDate(0, 1, 0)
	events, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, to)
	if len(issues) != 0 || len(events) != 4 {
		t.Fatalf("events=%d issues=%v", len(events), issues)
	}
	wants := []string{"2026-10-25T13:00:00Z", "2026-11-02T18:00:00Z", "2026-11-03T14:00:00Z", "2026-11-15T14:00:00Z"}
	for i, event := range events {
		if event.Start.Format(time.RFC3339) != wants[i] {
			t.Fatalf("event %d = %s want %s", i, event.Start, wants[i])
		}
		if event.Item.Path != item.Path || string(event.Item.Data) != string(item.Data) {
			t.Fatal("expansion changed source")
		}
	}
	if events[1].Title != "Moved meeting" || events[1].End.Sub(events[1].Start) != 90*time.Minute {
		t.Fatal("override details/duration lost")
	}
}

func TestAllDayDatesExclusiveEndAndNominalDuration(t *testing.T) {
	item := calendarTestItem(t, "BEGIN:VEVENT\nUID:holiday\nSUMMARY:Holiday\nDTSTART;VALUE=DATE:20261031\nDTEND;VALUE=DATE:20261103\nEND:VEVENT\n")
	from := testLocal(t, "America/Toronto", "2026-11-02T00:00:00")
	events, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 0, 1))
	if len(issues) != 0 || len(events) != 1 || !events[0].AllDay || events[0].Start.Day() != 31 || events[0].End.Day() != 3 {
		t.Fatalf("all-day overlap: %v %v", events, issues)
	}
	if events[0].End.Sub(events[0].Start) != 73*time.Hour {
		t.Fatal("all-day duration ignored DST")
	}
	events, _ = CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from.AddDate(0, 0, 1), from.AddDate(0, 0, 2))
	if len(events) != 0 {
		t.Fatal("exclusive all-day end occupied another day")
	}
	span, err := parseEventDuration("P1DT2H")
	if err != nil {
		t.Fatal(err)
	}
	start := testLocal(t, "America/Toronto", "2026-10-31T12:00:00")
	if span.end(start).Hour() != 14 || span.end(start).Sub(start) != 27*time.Hour {
		t.Fatal("nominal day duration lost DST")
	}
}

func TestMovedInstanceEntersRangeAndCancelledInstanceDisappears(t *testing.T) {
	item := calendarTestItem(t, "BEGIN:VEVENT\nUID:a\nSUMMARY:Series\nDTSTART:20260901T090000Z\nDTEND:20260901T100000Z\nRRULE:FREQ=DAILY;COUNT=3\nEND:VEVENT\nBEGIN:VEVENT\nUID:a\nRECURRENCE-ID:20260902T090000Z\nDTSTART:20261001T120000Z\nDTEND:20261001T130000Z\nSUMMARY:Moved into October\nEND:VEVENT\nBEGIN:VEVENT\nUID:a\nRECURRENCE-ID:20260903T090000Z\nSTATUS:CANCELLED\nEND:VEVENT\n")
	from := testLocal(t, "UTC", "2026-10-01T00:00:00")
	events, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 0, 1))
	if len(issues) != 0 || len(events) != 1 || events[0].Title != "Moved into October" {
		t.Fatalf("moved occurrence hidden: %v %v", events, issues)
	}
	from = testLocal(t, "UTC", "2026-09-01T00:00:00")
	events, issues = CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 0, 3))
	if len(issues) != 0 || len(events) != 1 {
		t.Fatal("moved/cancelled instance remained on original date")
	}
}

func TestJSONCalendarRulesAndOverrides(t *testing.T) {
	data := []byte(`{"@type":"Event","uid":"jmap","title":"Weekly","start":"2026-10-25T09:00:00","duration":"PT1H","timeZone":"America/Toronto","recurrenceRules":[{"frequency":"weekly","count":4,"byDay":[{"day":"su"}]}],"recurrenceOverrides":{"2026-11-01T09:00:00":{"start":"2026-11-02T11:00:00","duration":"PT2H","title":"Moved"},"2026-11-08T09:00:00":{"excluded":true},"2026-11-03T09:00:00":{}}}`)
	c := config.Collection{Protocol: "jmap-calendars"}
	item, err := Parse(c, data)
	if err != nil {
		t.Fatal(err)
	}
	from := testLocal(t, "UTC", "2026-10-25T00:00:00")
	events, issues := CalendarOccurrences(context.Background(), c, []Item{item}, from, from.AddDate(0, 1, 0))
	if len(issues) != 0 || len(events) != 4 {
		t.Fatalf("JSON recurrence: %v %v", events, issues)
	}
	if events[1].Start.Format(time.RFC3339) != "2026-11-02T16:00:00Z" || events[1].Title != "Moved" || events[1].End.Sub(events[1].Start) != 2*time.Hour {
		t.Fatal("JSON override lost")
	}
}

func TestUnsupportedAndDenseRecurrenceStaysInspectable(t *testing.T) {
	for _, extra := range []string{"DTSTART;TZID=Custom/Missing:20261001T100000\n", "DTSTART:20261001T100000Z\nRRULE:FREQ=SECONDLY\n", "DTSTART:20261001T100000Z\nRRULE:FREQ=DAILY;BYHOUR=0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23;BYMINUTE=0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59\n"} {
		item := calendarTestItem(t, "BEGIN:VEVENT\nUID:a\nSUMMARY:Inspect me\n"+extra+"END:VEVENT\n")
		from := testLocal(t, "UTC", "2026-10-01T00:00:00")
		events, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 0, 7))
		if len(events) != 0 || len(issues) != 1 || issues[0].Item.UID != "a" {
			t.Fatal("unsupported event vanished without inspection path")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	item := calendarTestItem(t, "BEGIN:VEVENT\nUID:a\nDTSTART:20261001T100000Z\nEND:VEVENT\n")
	events, _ := CalendarOccurrences(ctx, config.Collection{Protocol: "caldav"}, []Item{item}, time.Now(), time.Now().AddDate(0, 0, 30))
	if len(events) != 0 {
		t.Fatal("cancelled expansion returned stale events")
	}
	if !strings.Contains(item.Title, "a") {
		t.Fatal("missing title fallback")
	}
}

func TestNonexistentWallTimeAndMonthlyRules(t *testing.T) {
	from := testLocal(t, "America/Toronto", "2026-03-01T00:00:00")
	item := calendarTestItem(t, "BEGIN:VEVENT\nUID:gap\nDTSTART;TZID=America/Toronto:20260308T023000\nSUMMARY:Nonexistent time\nEND:VEVENT\n")
	events, issues := CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 1, 0))
	if len(events) != 0 || len(issues) != 1 {
		t.Fatal("nonexistent local time silently shifted to a different hour")
	}
	item = calendarTestItem(t, "BEGIN:VEVENT\nUID:monthly\nDTSTART:20260131T090000Z\nRRULE:FREQ=MONTHLY;COUNT=3\nSUMMARY:Month end\nEND:VEVENT\n")
	from = testLocal(t, "UTC", "2026-01-01T00:00:00")
	events, issues = CalendarOccurrences(context.Background(), config.Collection{Protocol: "caldav"}, []Item{item}, from, from.AddDate(0, 6, 0))
	if len(issues) != 0 || len(events) != 3 || events[1].Start.Month() != time.March || events[2].Start.Month() != time.May {
		t.Fatal("monthly rule invented a nonexistent month-end date")
	}
}
