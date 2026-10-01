// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

func calendarDemo(t *testing.T) (*App, chan func()) {
	t.Helper()
	a, q := demoApp(t)
	v := a.calendar
	dir := t.TempDir()
	v.collections[v.current].LocalDir = dir
	for i := range a.cfg.Collections {
		if a.cfg.Collections[i].Name == v.collections[v.current].Name {
			a.cfg.Collections[i].LocalDir = dir
		}
	}
	events := []struct{ title, start, end, extra string }{
		{"Autumn break", "20261005", "20261007", ""},
		{"Team stand-up", "20261005T090000Z", "20261005T093000Z", "RRULE:FREQ=DAILY;COUNT=4\n"},
		{"Research review", "20261005T100000Z", "20261005T113000Z", ""},
		{"Release planning", "20261005T103000Z", "20261005T120000Z", ""},
		{"Design workshop", "20261005T111500Z", "20261005T123000Z", ""},
		{"Coffee with Alex", "20261005T150000Z", "20261005T160000Z", ""},
		{"Evening deployment", "20261004T230000Z", "20261005T010000Z", ""},
		{"Community meetup", "20261015T180000Z", "20261015T200000Z", ""},
	}
	for i, e := range events {
		key := ""
		if len(e.start) == 8 {
			key = ";VALUE=DATE"
		}
		data := []byte(fmt.Sprintf("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:demo-%d\nSUMMARY:%s\nDTSTART%s:%s\nDTEND%s:%s\nLOCATION:Community room\nDESCRIPTION:Discuss our next release.\n%sEND:VEVENT\nEND:VCALENDAR\n", i, e.title, key, e.start, key, e.end, e.extra))
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("event-%d.ics", i)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	c := v.calendar
	c.zone = time.UTC
	c.timezone.SetSelected("UTC")
	c.cursor = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	v.load()
	pump(t, q, func() bool { return c.ready && len(v.items) == len(events) })
	a.tabs.SelectIndex(2)
	pump(t, q, func() bool { return c.ready && len(v.items) == len(events) })
	return a, q
}

func TestBusyDayDateJumpTodayAndNewEvent(t *testing.T) {
	a, q := calendarDemo(t)
	c := a.calendar.calendar
	var more *calendarTile
	walkCalendar(c.board, func(o fyne.CanvasObject) {
		if tile, ok := o.(*calendarTile); ok && strings.HasPrefix(tile.title, "+") && !tile.Hidden {
			more = tile
		}
	})
	if more == nil {
		t.Fatal("busy day has no overflow link")
	}
	test.Tap(more)
	pump(t, q, func() bool { return c.ready })
	if c.mode != "Day" || c.cursor.Day() != 5 || len(c.occurrences) != 7 {
		t.Fatal("overflow did not open all events on the busy day")
	}
	// Date jumping acts through the actual form, not just the navigation helper.
	c.jumpToDate()
	overlay := a.Window.Canvas().Overlays().Top()
	walkCalendar(overlay, func(o fyne.CanvasObject) {
		if entry, ok := o.(*widget.Entry); ok {
			entry.SetText("2026-10-15")
		}
	})
	var goButton *widget.Button
	walkCalendar(overlay, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Go" {
			goButton = b
		}
	})
	if goButton == nil {
		t.Fatal("date jump form has no Go button")
	}
	test.Tap(goButton)
	pump(t, q, func() bool { return c.ready })
	if c.cursor.Day() != 15 || len(c.occurrences) != 1 || c.occurrences[0].Title != "Community meetup" {
		t.Fatal("date jump did not select requested date")
	}
	a.calendar.create()
	windows := a.Fyne.Driver().AllWindows()
	w := windows[len(windows)-1]
	form := w.Content().(*fyne.Container).Objects[0].(*widget.Form)
	if form.Items[1].Widget.(*widget.Entry).Text != "2026-10-15T09:00:00" || form.Items[3].Widget.(*widget.Entry).Text != "UTC" {
		t.Fatal("new event lost the selected date or display timezone")
	}
	w.Close()
	var tile *calendarTile
	walkCalendar(c.board, func(o fyne.CanvasObject) {
		if b, ok := o.(*calendarTile); ok && b.title == "Community meetup" {
			tile = b
		}
	})
	if tile == nil {
		t.Fatal("timed event missing")
	}
	tile.MouseIn(nil)
	if tile.tooltip == nil {
		t.Fatal("event hover did not show its full details")
	}
	tile.MouseOut()
	if tile.tooltip != nil {
		t.Fatal("event hover hint did not close")
	}
	var today *widget.Button
	walkCalendar(c.content, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Today" {
			today = b
		}
	})
	if today == nil {
		t.Fatal("Today button missing")
	}
	test.Tap(today)
	pump(t, q, func() bool { return c.ready })
	if !c.cursor.Equal(calendarMidnight(time.Now().In(c.zone))) {
		t.Fatal("Today did not return to current local date")
	}
}

func walkCalendar(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(o)
	if box, ok := o.(*fyne.Container); ok {
		for _, child := range box.Objects {
			walkCalendar(child, visit)
		}
	}
	if widget, ok := o.(fyne.Widget); ok {
		for _, child := range test.WidgetRenderer(widget).Objects() {
			walkCalendar(child, visit)
		}
	}
}
func TestCalendarViewsNavigationAndEventDetails(t *testing.T) {
	a, q := calendarDemo(t)
	c := a.calendar.calendar
	var cells []*calendarMonthCell
	walkCalendar(c.board, func(o fyne.CanvasObject) {
		if cell, ok := o.(*calendarMonthCell); ok {
			cells = append(cells, cell)
		}
	})
	if len(cells) != 42 || cells[0].day.Weekday() != time.Monday || cells[0].day.Format("2006-01-02") != "2026-09-28" {
		t.Fatal("month cells are not aligned to the visible month")
	}
	var clicked bool
	walkCalendar(c.board, func(o fyne.CanvasObject) {
		if tile, ok := o.(*calendarTile); ok && strings.Contains(tile.title, "Evening deployment") && !tile.Hidden && !clicked {
			pos := a.Fyne.Driver().AbsolutePositionForObject(tile)
			test.TapCanvas(a.Window.Canvas(), pos.Add(fyne.NewPos(tile.Size().Width/2, tile.Size().Height/2)))
			clicked = true
		}
	})
	if !clicked || !strings.Contains(a.calendar.preview.Text, "Evening deployment") || a.calendar.selected < 0 {
		t.Fatal("calendar event did not open source-linked details")
	}
	screenshot(t, a.Window, "calendar-month")
	for _, mode := range []string{"Week", "Day", "Agenda"} {
		c.view.SetSelected(mode)
		pump(t, q, func() bool { return c.ready })
		if len(c.occurrences) == 0 {
			t.Fatalf("%s dropped calendar events", mode)
		}
		screenshot(t, a.Window, "calendar-"+strings.ToLower(mode))
	}
	c.view.SetSelected("Month")
	pump(t, q, func() bool { return c.ready })
	c.cursor = time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	c.navigate(1)
	pump(t, q, func() bool { return c.ready })
	if c.cursor.Month() != time.February || !strings.Contains(c.rangeLabel.Text, "February") {
		t.Fatal("month navigation skipped February from Jan 31")
	}
	c.cursor = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	c.view.SetSelected("Week")
	pump(t, q, func() bool { return c.ready })
	c.navigate(1)
	pump(t, q, func() bool { return c.ready })
	if c.cursor.Day() != 12 {
		t.Fatal("week navigation did not advance seven days")
	}
	c.navigate(-1)
	pump(t, q, func() bool { return c.ready })
	a.calendar.search.SetText("Coffee")
	pump(t, q, func() bool { return c.ready })
	if len(c.occurrences) != 1 || c.occurrences[0].Title != "Coffee with Alex" {
		t.Fatal("calendar search did not filter all views")
	}
	a.calendar.search.SetText("no such event")
	pump(t, q, func() bool { return c.ready })
	if len(c.occurrences) != 0 || a.calendar.selected != -1 {
		t.Fatal("empty calendar search left stale details")
	}
}

func TestCalendarOverlapLanesMidnightAndAllDay(t *testing.T) {
	a, _ := calendarDemo(t)
	c := a.calendar.calendar
	from := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	segments := calendarSegments(from, 7, c.occurrences)
	foundOverlaps, foundNight := false, false
	for i, s := range segments {
		if s.event.AllDay {
			t.Fatal("all-day event entered timed grid")
		}
		if s.event.Title == "Evening deployment" {
			foundNight = true
			if s.top != 0 || s.bottom != calendarHourHeight {
				t.Fatal("overnight event was not clipped to the day")
			}
		}
		if s.event.Title == "Release planning" {
			foundOverlaps = true
			if s.lanes != 3 {
				t.Fatalf("overlap group has %d lanes", s.lanes)
			}
		}
		for _, other := range segments[i+1:] {
			if s.day == other.day && s.lane == other.lane && s.top < other.bottom && other.top < s.bottom {
				t.Fatal("overlapping events share a lane")
			}
		}
	}
	if !foundOverlaps || !foundNight {
		t.Fatal("timed fixtures missing")
	}
	day := from.AddDate(0, 0, 2)
	for _, o := range c.dayEvents(day) {
		if o.Title == "Autumn break" {
			t.Fatal("all-day event continued past its exclusive end")
		}
	}
}

func TestCalendarDropsStaleRenderAndKeepsUnknownEventsAccessible(t *testing.T) {
	a, q := calendarDemo(t)
	c := a.calendar.calendar
	c.cursor = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	c.view.SetSelected("Week")
	c.cursor = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	c.render()
	pump(t, q, func() bool { return c.ready })
	if len(c.occurrences) != 0 || !strings.Contains(c.rangeLabel.Text, "2027") {
		t.Fatal("stale worker overwrote newer date range")
	}
	collection := a.calendar.collections[a.calendar.current]
	data := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VEVENT\nUID:unknown\nSUMMARY:Custom timezone event\nDTSTART;TZID=Custom/Zone:20261005T100000\nEND:VEVENT\nEND:VCALENDAR\n")
	item, err := pim.Parse(collection, data)
	if err != nil {
		t.Fatal(err)
	}
	item.Path = filepath.Join(collection.LocalDir, "unknown.ics")
	a.calendar.items = append(a.calendar.items, item)
	c.cursor = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	a.calendar.filter()
	pump(t, q, func() bool { return c.ready })
	if len(c.issues) != 1 || c.issues[0].Item.Title != "Custom timezone event" {
		t.Fatal("unsupported timezone vanished without accessible source")
	}
	c.showIssues()
	windows := a.Fyne.Driver().AllWindows()
	w := windows[len(windows)-1]
	var clicked bool
	walkCalendar(w.Content(), func(o fyne.CanvasObject) {
		if b := calendarButton(o, "Show details"); b != nil && !clicked {
			test.Tap(b)
			clicked = true
		}
	})
	if !clicked || !strings.Contains(a.calendar.preview.Text, "Custom timezone event") {
		t.Fatal("unplaced event did not open original details")
	}
}
