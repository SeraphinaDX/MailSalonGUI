// SPDX-License-Identifier: GPL-3.0-only
package pim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/teambition/rrule-go"
)

// Occurrence is a displayed instance. Item always refers to the original local
// resource: inspecting a generated occurrence never rewrites a recurring series.
type Occurrence struct {
	Item                   Item
	Start, End             time.Time
	AllDay                 bool
	Title, Location, Notes string
}
type CalendarIssue struct {
	Item   Item
	Reason string
}
type eventSpan struct {
	days  int
	clock time.Duration
}

func (s eventSpan) end(t time.Time) time.Time { return t.AddDate(0, 0, s.days).Add(s.clock) }

type scheduledEvent struct {
	start                  time.Time
	span                   eventSpan
	allDay                 bool
	title, location, notes string
	rules, excludedRules   []string
	dates, excluded        []time.Time
	overrides              map[int64]*scheduledEvent // nil means cancelled/excluded.
}

// CalendarOccurrences expands only the requested interval, with cancellation
// and finite iteration/output limits. Errors are returned with their source
// item so the UI can offer details instead of silently hiding unsupported data.
func CalendarOccurrences(ctx context.Context, c config.Collection, items []Item, from, to time.Time) ([]Occurrence, []CalendarIssue) {
	var out []Occurrence
	var issues []CalendarIssue
	if !to.After(from) {
		return out, issues
	}
	for _, item := range items {
		if ctx.Err() != nil {
			return nil, nil
		}
		var events []*scheduledEvent
		var err error
		if c.Protocol == "caldav" {
			events, err = scheduleICS(item, from.Location())
		} else {
			events, err = scheduleJSON(item, from.Location())
		}
		if err != nil {
			issues = append(issues, CalendarIssue{item, err.Error()})
			continue
		}
		var instances []Occurrence
		for _, event := range events {
			x, e := expandEvent(ctx, item, event, from, to)
			if e != nil {
				err = e
				break
			}
			instances = append(instances, x...)
		}
		if err != nil {
			issues = append(issues, CalendarIssue{item, err.Error()})
			continue
		}
		out = append(out, instances...)
		if len(out) > 10000 {
			issues = append(issues, CalendarIssue{item, "visible occurrence limit reached; narrow the date range or search"})
			out = out[:10000]
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		if out[i].AllDay != out[j].AllDay {
			return out[i].AllDay
		}
		return out[i].Title < out[j].Title
	})
	return out, issues
}

func eventTime(p calendarProperty, fallback *time.Location) (time.Time, bool, error) {
	allDay := p.param("VALUE") == "DATE" || len(p.value) == 8
	zone := fallback
	if id := p.param("TZID"); id != "" && !allDay {
		var err error
		zone, err = time.LoadLocation(id)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("timezone %q cannot be placed in the grid; inspect its source", id)
		}
	}
	layout := "20060102T150405"
	if allDay {
		layout = "20060102"
	} else if strings.HasSuffix(p.value, "Z") {
		layout = "20060102T150405Z"
		zone = time.UTC
	}
	t, err := time.ParseInLocation(layout, p.value, zone)
	if err != nil || t.Format(layout) != p.value {
		return time.Time{}, false, fmt.Errorf("invalid event date %q", p.value)
	}
	return t, allDay, nil
}

var durationRE = regexp.MustCompile(`^P(?:(\d+)W|(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?)$`)

func parseEventDuration(value string) (eventSpan, error) {
	m := durationRE.FindStringSubmatch(value)
	if m == nil || value == "P" || value == "PT" {
		return eventSpan{}, errors.New("unsupported event duration")
	}
	numbers := make([]float64, 5)
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			n, err := strconv.ParseFloat(m[i], 64)
			if err != nil || n > 1e6 {
				return eventSpan{}, errors.New("event duration is too large")
			}
			numbers[i-1] = n
		}
	}
	return eventSpan{days: int(numbers[0]*7 + numbers[1]), clock: time.Duration((numbers[2]*3600 + numbers[3]*60 + numbers[4]) * float64(time.Second))}, nil
}
func dateDays(a, b time.Time) int {
	utcDate := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	return int(utcDate(b).Sub(utcDate(a)) / (24 * time.Hour))
}
func componentSchedule(c *calendarComponent, fallback *time.Location, inherited *scheduledEvent) (*scheduledEvent, error) {
	if strings.EqualFold(c.value("STATUS"), "CANCELLED") {
		return nil, nil
	}
	e := &scheduledEvent{overrides: map[int64]*scheduledEvent{}}
	if inherited != nil {
		*e = *inherited
		e.rules = nil
		e.dates = nil
		e.excluded = nil
		e.overrides = map[int64]*scheduledEvent{}
	}
	ps := c.values("DTSTART")
	if len(ps) == 0 {
		if inherited == nil {
			return nil, errors.New("event has no start date")
		}
	} else {
		var err error
		e.start, e.allDay, err = eventTime(ps[0], fallback)
		if err != nil {
			return nil, err
		}
	}
	if ps := c.values("DTEND"); len(ps) > 0 {
		end, allDay, err := eventTime(ps[0], e.start.Location())
		if err != nil {
			return nil, err
		}
		if allDay != e.allDay || end.Before(e.start) {
			return nil, errors.New("event end does not match its start")
		}
		if e.allDay {
			e.span = eventSpan{days: dateDays(e.start, end)}
		} else {
			e.span = eventSpan{clock: end.Sub(e.start)}
		}
	} else if d := c.value("DURATION"); d != "" {
		var err error
		e.span, err = parseEventDuration(d)
		if err != nil {
			return nil, err
		}
	} else if inherited == nil && e.allDay {
		e.span = eventSpan{days: 1}
	}
	if c.value("SUMMARY") != "" {
		e.title = unescape(c.value("SUMMARY"))
	}
	if c.value("LOCATION") != "" {
		e.location = unescape(c.value("LOCATION"))
	}
	if c.value("DESCRIPTION") != "" {
		e.notes = unescape(c.value("DESCRIPTION"))
	}
	for _, p := range c.values("RRULE") {
		e.rules = append(e.rules, p.value)
	}
	for _, key := range []string{"RDATE", "EXDATE"} {
		for _, p := range c.values(key) {
			if p.param("VALUE") == "PERIOD" {
				return nil, errors.New("RDATE periods need source inspection")
			}
			for _, value := range strings.Split(p.value, ",") {
				p.value = value
				t, _, err := eventTime(p, e.start.Location())
				if err != nil {
					return nil, err
				}
				if key == "RDATE" {
					e.dates = append(e.dates, t)
				} else {
					e.excluded = append(e.excluded, t)
				}
			}
		}
	}
	return e, nil
}

func scheduleICS(item Item, zone *time.Location) ([]*scheduledEvent, error) {
	calendars, err := ParseCalendar(item.Data, "")
	if err != nil {
		return nil, err
	}
	var result []*scheduledEvent
	for _, calendar := range calendars {
		if calendar.Method == "CANCEL" {
			continue
		}
		var master *calendarComponent
		var overrides []*calendarComponent
		for _, c := range calendar.root.children {
			if c.name != "VEVENT" {
				continue
			}
			if c.value("RECURRENCE-ID") == "" {
				master = c
			} else {
				overrides = append(overrides, c)
			}
		}
		var base *scheduledEvent
		if master != nil {
			base, err = componentSchedule(master, zone, nil)
			if err != nil {
				return nil, err
			}
			if base == nil {
				continue
			}
		}
		for _, c := range overrides {
			p := c.values("RECURRENCE-ID")[0]
			if p.param("RANGE") != "" {
				return nil, errors.New("THISANDFUTURE recurrence updates need source inspection")
			}
			fallback := zone
			if base != nil {
				fallback = base.start.Location()
			}
			id, _, e := eventTime(p, fallback)
			if e != nil {
				return nil, e
			}
			o, e := componentSchedule(c, fallback, base)
			if e != nil {
				return nil, e
			}
			if o != nil && c.value("DTSTART") == "" {
				o.start = id
			}
			if base != nil {
				base.overrides[id.Unix()] = o
			} else if o != nil {
				result = append(result, o)
			}
		}
		if base != nil {
			result = append(result, base)
		}
	}
	return result, nil
}

func expandRule(ctx context.Context, rule string, start, from, to time.Time) ([]time.Time, error) {
	option, err := rrule.StrToROptionInLocation(rule, start.Location())
	if err != nil {
		return nil, err
	}
	// Sub-daily rules can require millions of historical iterations. Keep them
	// inspectable through the source instead of blocking calendar navigation.
	if option.Freq > rrule.DAILY {
		return nil, errors.New("sub-daily recurrence needs source inspection")
	}
	if option.Interval < 0 || option.Count < 0 {
		return nil, errors.New("negative recurrence interval/count")
	}
	option.Dtstart = start
	if option.Until.IsZero() || option.Until.After(to) {
		option.Until = to
	}
	r, err := rrule.NewRRule(*option)
	if err != nil {
		return nil, err
	}
	next := r.Iterator()
	var dates []time.Time
	for steps := 0; steps < 100000; steps++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t, ok := next()
		if !ok || !t.Before(to) {
			return dates, nil
		}
		if !t.Before(from) {
			// The recurrence library can normalize a nonexistent spring-forward
			// wall time to another hour. Never display that as the requested time.
			hours := option.Byhour
			if len(hours) == 0 {
				hours = []int{start.Hour()}
			}
			validHour := false
			for _, hour := range hours {
				if t.Hour() == hour {
					validHour = true
					break
				}
			}
			if !validHour {
				return nil, errors.New("recurrence crosses a nonexistent local time; inspect its source")
			}
			dates = append(dates, t)
			if len(dates) > 2000 {
				return nil, errors.New("too many recurrences; narrow the visible range")
			}
		}
	}
	return nil, errors.New("recurrence expansion limit reached; inspect the source")
}

func expandEvent(ctx context.Context, item Item, e *scheduledEvent, from, to time.Time) ([]Occurrence, error) {
	// Include starts before the range whose duration overlaps its first day.
	lookback := from.AddDate(0, 0, -e.span.days).Add(-e.span.clock)
	starts := map[int64]time.Time{e.start.Unix(): e.start}
	for _, rule := range e.rules {
		dates, err := expandRule(ctx, rule, e.start, lookback, to)
		if err != nil {
			return nil, err
		}
		for _, t := range dates {
			starts[t.Unix()] = t
		}
	}
	for _, t := range e.dates {
		starts[t.Unix()] = t
	}
	for _, rule := range e.excludedRules {
		dates, err := expandRule(ctx, rule, e.start, lookback, to)
		if err != nil {
			return nil, err
		}
		for _, t := range dates {
			delete(starts, t.Unix())
		}
	}
	for _, t := range e.excluded {
		delete(starts, t.Unix())
	}
	var out []Occurrence
	appendEvent := func(event *scheduledEvent, start time.Time) {
		end := event.span.end(start)
		if start.Before(to) && (end.After(from) || end.Equal(start) && !start.Before(from)) {
			title := event.title
			if title == "" {
				title = item.Title
			}
			out = append(out, Occurrence{Item: item, Start: start.In(from.Location()), End: end.In(from.Location()), AllDay: event.allDay, Title: title, Location: event.location, Notes: event.notes})
		}
	}
	for key, start := range starts {
		if _, overridden := e.overrides[key]; !overridden {
			appendEvent(e, start)
		}
	}
	// Moved instances may enter the range even if their original date is outside.
	for _, event := range e.overrides {
		if event != nil {
			appendEvent(event, event.start)
		}
	}
	return out, nil
}

func jsonEvent(v map[string]any, zone *time.Location) (*scheduledEvent, error) {
	displayZone := zone
	if strings.EqualFold(str(v["status"]), "cancelled") {
		return nil, nil
	}
	if id := str(v["timeZone"]); id != "" {
		var err error
		zone, err = time.LoadLocation(id)
		if err != nil {
			return nil, fmt.Errorf("timezone %q cannot be placed in the grid", id)
		}
	}
	start, err := time.ParseInLocation("2006-01-02T15:04:05", str(v["start"]), zone)
	if err != nil || start.Format("2006-01-02T15:04:05") != str(v["start"]) {
		return nil, errors.New("invalid JSCalendar start")
	}
	span, err := parseEventDuration(str(v["duration"]))
	if err != nil {
		return nil, err
	}
	allDay, _ := v["showWithoutTime"].(bool)
	if allDay {
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, displayZone)
	}
	location := first(v["locations"])
	name := str(location["name"])
	if name == "" {
		name = str(location["title"])
	}
	return &scheduledEvent{start: start, span: span, allDay: allDay, title: str(v["title"]), location: name, notes: str(v["description"]), overrides: map[int64]*scheduledEvent{}}, nil
}

func jsonRule(rule map[string]any) (string, error) {
	if skip := str(rule["skip"]); skip != "" && skip != "omit" {
		return "", errors.New("recurrence skip adjustment needs source inspection")
	}
	if scale := str(rule["rscale"]); scale != "" && !strings.EqualFold(scale, "gregorian") {
		return "", errors.New("non-Gregorian recurrence needs source inspection")
	}
	parts := []string{"FREQ=" + strings.ToUpper(str(rule["frequency"]))}
	keys := []struct{ json, ical string }{{"interval", "INTERVAL"}, {"count", "COUNT"}, {"firstDayOfWeek", "WKST"}, {"byMonth", "BYMONTH"}, {"byMonthDay", "BYMONTHDAY"}, {"byYearDay", "BYYEARDAY"}, {"byWeekNo", "BYWEEKNO"}, {"byHour", "BYHOUR"}, {"byMinute", "BYMINUTE"}, {"bySecond", "BYSECOND"}, {"bySetPosition", "BYSETPOS"}}
	for _, key := range keys {
		value, ok := rule[key.json]
		if !ok {
			continue
		}
		values := []any{value}
		if xs, ok := value.([]any); ok {
			values = xs
		}
		var stringsOfValues []string
		for _, x := range values {
			stringsOfValues = append(stringsOfValues, strings.ToUpper(fmt.Sprint(x)))
		}
		parts = append(parts, key.ical+"="+strings.Join(stringsOfValues, ","))
	}
	if days, ok := rule["byDay"].([]any); ok {
		var values []string
		for _, day := range days {
			d, ok := day.(map[string]any)
			if !ok {
				return "", errors.New("invalid recurrence weekday")
			}
			prefix := ""
			if n, ok := d["nthOfPeriod"].(float64); ok {
				prefix = strconv.Itoa(int(n))
			}
			values = append(values, prefix+strings.ToUpper(str(d["day"])))
		}
		parts = append(parts, "BYDAY="+strings.Join(values, ","))
	}
	if until := str(rule["until"]); until != "" {
		t, err := time.Parse("2006-01-02T15:04:05", until)
		if err != nil {
			return "", errors.New("invalid recurrence until")
		}
		// JSCalendar UNTIL is in the event's local timezone; no trailing Z.
		parts = append(parts, "UNTIL="+t.Format("20060102T150405"))
	}
	return strings.Join(parts, ";"), nil
}

func scheduleJSON(item Item, zone *time.Location) ([]*scheduledEvent, error) {
	var v map[string]any
	if err := json.Unmarshal(item.Data, &v); err != nil {
		return nil, err
	}
	e, err := jsonEvent(v, zone)
	if err != nil || e == nil {
		return nil, err
	}
	for _, key := range []string{"recurrenceRules", "excludedRecurrenceRules"} {
		if rules, ok := v[key].([]any); ok {
			for _, rule := range rules {
				r, ok := rule.(map[string]any)
				if !ok {
					return nil, errors.New("invalid recurrence rule")
				}
				text, err := jsonRule(r)
				if err != nil {
					return nil, err
				}
				if key == "recurrenceRules" {
					e.rules = append(e.rules, text)
				} else {
					e.excludedRules = append(e.excludedRules, text)
				}
			}
		}
	}
	if overrides, ok := v["recurrenceOverrides"].(map[string]any); ok {
		for id, value := range overrides {
			t, err := time.ParseInLocation("2006-01-02T15:04:05", id, e.start.Location())
			if err != nil {
				return nil, errors.New("invalid recurrence override date")
			}
			patch, ok := value.(map[string]any)
			if !ok {
				return nil, errors.New("invalid recurrence override")
			}
			if patch["excluded"] == true {
				e.overrides[t.Unix()] = nil
				continue
			}
			copy := map[string]any{}
			for k, value := range v {
				copy[k] = value
			}
			copy["start"] = id
			for k, value := range patch {
				if strings.Contains(k, "/") {
					if strings.HasPrefix(k, "locations/") {
						return nil, errors.New("nested location override needs source inspection")
					}
					continue
				}
				copy[k] = value
			}
			o, err := jsonEvent(copy, zone)
			if err != nil {
				return nil, err
			}
			e.overrides[t.Unix()] = o
		}
	}
	return []*scheduledEvent{e}, nil
}
