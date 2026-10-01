// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

type calendarView struct {
	owner              *collectionView
	content            fyne.CanvasObject
	board              *fyne.Container
	view, timezone     *widget.Select
	rangeLabel, status *widget.Label
	cursor             time.Time
	zone               *time.Location
	mode               string
	weekStart          time.Weekday
	occurrences        []pim.Occurrence
	issues             []pim.CalendarIssue
	revision           uint64
	cancel             context.CancelFunc
	ready              bool
}

func calendarMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
func calendarMonday(t time.Time) time.Time {
	return calendarMidnight(t).AddDate(0, 0, -(int(t.Weekday())+6)%7)
}
func calendarBounds(mode string, date time.Time, starts ...time.Weekday) (time.Time, time.Time) {
	firstDay := time.Monday
	if len(starts) > 0 {
		firstDay = starts[0]
	}
	weekBeginning := func(t time.Time) time.Time {
		return calendarMidnight(t).AddDate(0, 0, -(int(t.Weekday())-int(firstDay)+7)%7)
	}
	date = calendarMidnight(date)
	switch mode {
	case "Month":
		first := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		start := weekBeginning(first)
		return start, start.AddDate(0, 0, 42)
	case "Week":
		start := weekBeginning(date)
		return start, start.AddDate(0, 0, 7)
	case "Agenda":
		return date, date.AddDate(0, 0, 30)
	default:
		return date, date.AddDate(0, 0, 1)
	}
}

func (v *collectionView) newCalendarView() *calendarView {
	c := &calendarView{owner: v, zone: time.Local, cursor: calendarMidnight(time.Now()), mode: "Month", weekStart: time.Monday}
	if v.app.cfg.CalendarWeekStart == "sunday" {
		c.weekStart = time.Sunday
	}
	for _, mode := range []string{"Month", "Week", "Day", "Agenda"} {
		if strings.EqualFold(mode, v.app.cfg.CalendarDefaultView) {
			c.mode = mode
		}
	}
	c.board = container.NewStack(widget.NewLabel("Loading calendar…"))
	c.rangeLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	c.status = widget.NewLabel("")
	c.status.Truncation = fyne.TextTruncateEllipsis
	c.view = widget.NewSelect([]string{"Month", "Week", "Day", "Agenda"}, func(value string) {
		if c.mode != value {
			c.mode = value
			c.render()
		}
	})
	c.view.SetSelected(c.mode)
	c.timezone = widget.NewSelect([]string{"Local time", "UTC"}, func(value string) {
		if value == "UTC" {
			c.zone = time.UTC
		} else {
			c.zone = time.Local
		}
		c.cursor = time.Date(c.cursor.Year(), c.cursor.Month(), c.cursor.Day(), 0, 0, 0, 0, c.zone)
		c.render()
	})
	c.timezone.SetSelected("Local time")
	previous := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { c.navigate(-1) })
	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { c.navigate(1) })
	today := widget.NewButton("Today", func() { c.cursor = calendarMidnight(time.Now().In(c.zone)); c.render() })
	jump := widget.NewButton("Go to date…", c.jumpToDate)
	nav := container.NewBorder(nil, nil, container.NewHBox(previous, today, next, jump), c.timezone, c.rangeLabel)
	v.search.SetPlaceHolder("Search calendar events")
	selector := container.NewBorder(nil, nil, container.NewHBox(v.chooser, c.view), widget.NewButton("Reload", v.load), v.search)
	tools := container.NewHBox(widget.NewButtonWithIcon("New event", theme.ContentAddIcon(), v.create), widget.NewButton("Edit source", v.edit), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), v.remove))
	details := container.NewBorder(widget.NewLabelWithStyle("Event details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), tools, nil, nil, container.NewVScroll(v.preview))
	panes := container.NewHSplit(c.board, details)
	panes.Offset = 0.76
	inspect := widget.NewButton("Unplaced events…", c.showIssues)
	c.content = container.NewBorder(container.NewVBox(selector, nav), container.NewBorder(nil, nil, nil, inspect, c.status), nil, nil, panes)
	return c
}

func (c *calendarView) clear(message string) {
	if c.cancel != nil {
		c.cancel()
	}
	c.revision++
	c.ready = false
	c.occurrences = nil
	c.issues = nil
	l := widget.NewLabel(message)
	l.Wrapping = fyne.TextWrapWord
	c.board.Objects = []fyne.CanvasObject{l}
	c.board.Refresh()
	c.status.SetText(message)
}
func (c *calendarView) navigate(direction int) {
	switch c.mode {
	case "Month":
		c.cursor = time.Date(c.cursor.Year(), c.cursor.Month()+time.Month(direction), 1, 0, 0, 0, 0, c.zone)
	case "Week":
		c.cursor = c.cursor.AddDate(0, 0, 7*direction)
	case "Agenda":
		c.cursor = c.cursor.AddDate(0, 0, 30*direction)
	default:
		c.cursor = c.cursor.AddDate(0, 0, direction)
	}
	c.render()
}
func (c *calendarView) jumpToDate() {
	e := widget.NewEntry()
	e.SetText(c.cursor.Format("2006-01-02"))
	e.SetPlaceHolder("YYYY-MM-DD")
	dialog.ShowForm("Go to date", "Go", "Cancel", []*widget.FormItem{widget.NewFormItem("Date", e)}, func(ok bool) {
		if !ok {
			return
		}
		day, err := time.ParseInLocation("2006-01-02", e.Text, c.zone)
		if err != nil {
			dialog.ShowError(fmt.Errorf("use a date such as 2026-10-01"), c.owner.app.Window)
			return
		}
		c.cursor = day
		c.render()
	}, c.owner.app.Window)
}

func (c *calendarView) render() {
	v := c.owner
	collection, ok := v.collection()
	if !ok {
		return
	}
	if c.cancel != nil {
		c.cancel()
	}
	ctx, cancel := context.WithCancel(v.app.ctx)
	c.cancel = cancel
	c.revision++
	revision, gen := c.revision, v.generation
	items := append([]pim.Item(nil), v.filtered...)
	mode, cursor := c.mode, c.cursor
	from, to := calendarBounds(mode, cursor, c.weekStart)
	if mode == "Month" {
		c.rangeLabel.SetText(cursor.Format("January 2006"))
	} else if mode == "Day" {
		c.rangeLabel.SetText(from.Format("Monday, January 2, 2006"))
	} else {
		c.rangeLabel.SetText(from.Format("Jan 2, 2006") + " – " + to.AddDate(0, 0, -1).Format("Jan 2, 2006"))
	}
	c.ready = false
	c.issues = nil
	c.occurrences = nil
	c.board.Objects = []fyne.CanvasObject{widget.NewLabel("Loading calendar view…")}
	c.board.Refresh()
	c.status.SetText("Loading calendar view…")
	go func() {
		events, issues := pim.CalendarOccurrences(ctx, collection, items, from, to)
		if ctx.Err() != nil {
			return
		}
		v.app.post(func() {
			if revision != c.revision || gen != v.generation {
				return
			}
			c.occurrences, c.issues = events, issues
			var board fyne.CanvasObject
			switch mode {
			case "Month":
				board = c.monthBoard(from, cursor)
			case "Week", "Day":
				board = c.timeBoard(from, to)
			default:
				board = c.agendaBoard(from, to)
			}
			c.board.Objects = []fyne.CanvasObject{board}
			c.board.Refresh()
			c.ready = true
			c.status.SetText(fmt.Sprintf("%d events in view · %d source items · %d unplaced · %s", len(events), len(items), len(issues), c.timezone.Selected))
		})
	}()
}

func occurrenceOnDay(o pim.Occurrence, day time.Time) bool {
	end := day.AddDate(0, 0, 1)
	return o.Start.Before(end) && (o.End.After(day) || o.End.Equal(o.Start) && !o.Start.Before(day))
}
func (c *calendarView) dayEvents(day time.Time) []pim.Occurrence {
	var events []pim.Occurrence
	for _, o := range c.occurrences {
		if occurrenceOnDay(o, day) {
			events = append(events, o)
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].AllDay != events[j].AllDay {
			return events[i].AllDay
		}
		return events[i].Start.Before(events[j].Start)
	})
	return events
}
func (c *calendarView) openDay(day time.Time) {
	c.cursor = day
	if c.mode == "Day" {
		c.render()
	} else {
		c.view.SetSelected("Day")
	}
}
func (c *calendarView) selectOccurrence(o pim.Occurrence) {
	v := c.owner
	v.selected = -1
	for i, item := range v.filtered {
		if item.Path == o.Item.Path {
			v.selected = i
			break
		}
	}
	text := o.Title + "\n\nStart: " + o.Start.Format("Mon, Jan 2, 2006 15:04 MST -07:00") + "\nEnd: " + o.End.Format("Mon, Jan 2, 2006 15:04 MST -07:00")
	if o.AllDay {
		text = o.Title + "\n\nAll day: " + o.Start.Format("Mon, Jan 2, 2006") + "\nEnd (exclusive): " + o.End.Format("Mon, Jan 2, 2006")
	}
	text += "\nLocation: " + o.Location + "\n\n" + o.Notes
	if o.Item.Recurring {
		text += "\n\nRecurring event. Edit source and Delete apply to the whole stored series."
	}
	v.preview.SetText(text)
}
func (c *calendarView) showIssues() {
	if len(c.issues) == 0 {
		dialog.ShowInformation("Unplaced events", "All loaded events can be placed in the calendar grid.", c.owner.app.Window)
		return
	}
	w := c.owner.app.Fyne.NewWindow("Unplaced calendar events")
	w.Resize(fyne.NewSize(660, 500))
	box := container.NewVBox()
	for _, issue := range c.issues {
		x := issue
		l := widget.NewLabel(x.Reason)
		l.Wrapping = fyne.TextWrapWord
		box.Add(widget.NewCard(x.Item.Title, "", container.NewVBox(l, widget.NewButton("Show details", func() {
			v := c.owner
			for i, item := range v.filtered {
				if item.Path == x.Item.Path {
					v.list.Select(i)
					break
				}
			}
			w.Close()
		}))))
	}
	w.SetContent(container.NewVScroll(box))
	showWindow(w)
}

func (c *calendarView) monthBoard(from, cursor time.Time) fyne.CanvasObject {
	headers := container.NewGridWithColumns(7)
	for i := 0; i < 7; i++ {
		headers.Add(widget.NewLabelWithStyle(from.AddDate(0, 0, i).Weekday().String(), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	}
	cells := container.NewGridWithColumns(7)
	for i := 0; i < 42; i++ {
		day := from.AddDate(0, 0, i)
		cells.Add(newCalendarMonthCell(day, day.Month() == cursor.Month(), c.dayEvents(day), func() { c.openDay(day) }, c.selectOccurrence))
	}
	return container.NewBorder(headers, nil, nil, nil, cells)
}

func (c *calendarView) agendaBoard(from, to time.Time) fyne.CanvasObject {
	box := container.NewVBox()
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		events := c.dayEvents(day)
		if len(events) == 0 {
			continue
		}
		box.Add(widget.NewLabelWithStyle(day.Format("Monday, January 2"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, event := range events {
			o := event
			detail := o.Start.Format("15:04") + " – " + o.End.Format("15:04")
			if o.AllDay {
				detail = "All day"
			}
			if calendarMidnight(o.Start).Before(day) {
				detail = "Continues · " + detail
			}
			box.Add(newCalendarTile(o.Title, detail, false, func() { c.selectOccurrence(o) }))
		}
	}
	if len(box.Objects) == 0 {
		box.Add(widget.NewLabel("No events in this date range."))
	}
	return container.NewVScroll(box)
}

func (c *calendarView) timeBoard(from, to time.Time) fyne.CanvasObject {
	days := 0
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		days++
	}
	headers := container.NewGridWithColumns(days)
	allDay := container.NewGridWithColumns(days)
	for i := 0; i < days; i++ {
		day := from.AddDate(0, 0, i)
		headers.Add(newCalendarTile(day.Format("Mon 2"), "", true, func() { c.openDay(day) }))
		box := container.NewVBox()
		for _, event := range c.dayEvents(day) {
			if !event.AllDay {
				continue
			}
			o := event
			box.Add(newCalendarTile(o.Title, "", true, func() { c.selectOccurrence(o) }))
		}
		if len(box.Objects) == 0 {
			box.Add(widget.NewLabel(""))
		}
		allDay.Add(box)
	}
	top := container.NewVBox(container.NewBorder(nil, nil, widget.NewLabel("Time    "), nil, headers), container.NewBorder(nil, nil, widget.NewLabel("All day"), nil, allDay))
	grid := newCalendarTimeGrid(from, days, c.occurrences, c.selectOccurrence)
	scroll := container.NewVScroll(grid)
	scroll.Offset = fyne.NewPos(0, 8*calendarHourHeight)
	return container.NewBorder(top, nil, nil, nil, scroll)
}

func calendarEventText(o pim.Occurrence) string {
	if o.AllDay {
		return o.Title
	}
	return strings.TrimSpace(o.Start.Format("15:04") + " " + o.Title)
}
func calendarMoreLabel(n int) string { return fmt.Sprintf("+%d more…", n) }
func calendarHourLabel(n int) string { return fmt.Sprintf("%02d:00", n) }
