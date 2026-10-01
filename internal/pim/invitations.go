// SPDX-License-Identifier: GPL-3.0-only

package pim

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

// CalendarEvent contains one UID, including its recurrence exceptions and
// original timezone definitions. Source properties are kept for lossless import.
type CalendarEvent struct {
	Item
	Method, Organizer, Status string
	AllDay                    bool
	root                      *calendarComponent
}

type calendarProperty struct {
	line, name, value string
	params            []string
}
type calendarComponent struct {
	name     string
	props    []calendarProperty
	children []*calendarComponent
}

// splitCalendar ignores separators inside quoted parameter values (e.g. CN).
func splitCalendar(s string, sep byte) []string {
	var parts []string
	start := 0
	quoted := false
	for i := range s {
		if s[i] == '"' {
			quoted = !quoted
		}
		if s[i] == sep && !quoted {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

func calendarProp(line string) (calendarProperty, error) {
	parts := splitCalendar(line, ':')
	if len(parts) < 2 {
		return calendarProperty{}, errors.New("malformed calendar content line")
	}
	head := splitCalendar(parts[0], ';')
	return calendarProperty{line: line, name: strings.ToUpper(head[0]), value: strings.Join(parts[1:], ":"), params: head[1:]}, nil
}
func (p calendarProperty) param(key string) string {
	for _, param := range p.params {
		k, value, _ := strings.Cut(param, "=")
		if strings.EqualFold(k, key) {
			return strings.Trim(value, "\"")
		}
	}
	return ""
}
func (c *calendarComponent) values(key string) []calendarProperty {
	var out []calendarProperty
	for _, p := range c.props {
		if p.name == key {
			out = append(out, p)
		}
	}
	return out
}
func (c *calendarComponent) value(key string) string {
	ps := c.values(key)
	if len(ps) == 0 {
		return ""
	}
	return ps[0].value
}
func (c *calendarComponent) lines() []string {
	out := []string{"BEGIN:" + c.name}
	for _, p := range c.props {
		out = append(out, p.line)
	}
	for _, child := range c.children {
		out = append(out, child.lines()...)
	}
	return append(out, "END:"+c.name)
}

// ParseCalendar splits multiple events by UID while retaining recurrence
// exceptions. A malformed calendar does not prevent saving the mail attachment.
func ParseCalendar(data []byte, mimeMethod string) ([]CalendarEvent, error) {
	var root *calendarComponent
	var stack []*calendarComponent
	for _, line := range unfold(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})) {
		p, err := calendarProp(line)
		if err != nil {
			return nil, err
		}
		switch p.name {
		case "BEGIN":
			if len(stack) >= 32 {
				return nil, errors.New("calendar nesting is too deep")
			}
			c := &calendarComponent{name: strings.ToUpper(p.value)}
			if len(stack) == 0 {
				if root != nil || c.name != "VCALENDAR" {
					return nil, errors.New("expected one VCALENDAR")
				}
				root = c
			} else {
				parent := stack[len(stack)-1].name
				if c.name == "VCALENDAR" || ((c.name == "VEVENT" || c.name == "VTIMEZONE") && parent != "VCALENDAR") {
					return nil, errors.New("calendar component is in the wrong container")
				}
				stack[len(stack)-1].children = append(stack[len(stack)-1].children, c)
			}
			stack = append(stack, c)
		case "END":
			if len(stack) == 0 || stack[len(stack)-1].name != strings.ToUpper(p.value) {
				return nil, errors.New("unbalanced calendar components")
			}
			stack = stack[:len(stack)-1]
		default:
			if len(stack) == 0 {
				return nil, errors.New("calendar property outside component")
			}
			stack[len(stack)-1].props = append(stack[len(stack)-1].props, p)
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, errors.New("incomplete calendar")
	}
	if root.value("VERSION") != "2.0" || len(root.values("VERSION")) != 1 {
		return nil, errors.New("calendar needs VERSION:2.0")
	}
	if len(root.values("METHOD")) > 1 {
		return nil, errors.New("calendar has multiple methods")
	}
	method := strings.ToUpper(root.value("METHOD"))
	if mimeMethod != "" && method != "" && !strings.EqualFold(method, mimeMethod) {
		return nil, errors.New("MIME and calendar methods disagree")
	}
	if method == "" {
		method = strings.ToUpper(mimeMethod)
	}
	var ids []string
	groups := map[string][]*calendarComponent{}
	for _, c := range root.children {
		if c.name != "VEVENT" {
			continue
		}
		uid := c.value("UID")
		if strings.TrimSpace(uid) == "" || len(c.values("UID")) != 1 {
			return nil, errors.New("event needs one UID")
		}
		for _, key := range []string{"DTSTART", "DTEND", "DURATION", "ORGANIZER", "SEQUENCE", "RECURRENCE-ID", "SUMMARY", "STATUS", "RRULE"} {
			if len(c.values(key)) > 1 {
				return nil, fmt.Errorf("event has multiple %s properties", key)
			}
		}
		if c.value("DTEND") != "" && c.value("DURATION") != "" {
			return nil, errors.New("event has both end time and duration")
		}
		if seq := c.value("SEQUENCE"); seq != "" {
			n, err := strconv.Atoi(seq)
			if err != nil || n < 0 {
				return nil, errors.New("invalid event sequence")
			}
		}
		for _, existing := range groups[uid] {
			if existing.value("RECURRENCE-ID") == c.value("RECURRENCE-ID") {
				return nil, errors.New("duplicate event recurrence instance")
			}
		}
		if _, ok := groups[uid]; !ok {
			ids = append(ids, uid)
		}
		groups[uid] = append(groups[uid], c)
	}
	if len(ids) == 0 {
		return nil, errors.New("no VEVENT in calendar")
	}
	var out []CalendarEvent
	for _, uid := range ids {
		group := &calendarComponent{name: "VCALENDAR", props: root.props}
		for _, c := range root.children {
			if c.name == "VTIMEZONE" {
				group.children = append(group.children, c)
			}
		}
		group.children = append(group.children, groups[uid]...)
		data := serialize(group.lines())
		item, err := Parse(config.Collection{Protocol: "caldav"}, data)
		if err != nil {
			return nil, err
		}
		main := groups[uid][0]
		for _, c := range groups[uid] {
			if c.value("RECURRENCE-ID") == "" {
				main = c
				break
			}
		}
		// Main events can occur after their exceptions in the source.
		item.Title = unescape(main.value("SUMMARY"))
		if item.Title == "" {
			item.Title = uid
		}
		item.Start = readableDate(main.value("DTSTART"))
		item.End = readableDate(main.value("DTEND"))
		if item.End == "" {
			item.End = main.value("DURATION")
		}
		item.Location = unescape(main.value("LOCATION"))
		item.Notes = unescape(main.value("DESCRIPTION"))
		for _, c := range groups[uid] {
			if c.value("RRULE") != "" || c.value("RDATE") != "" || c.value("RECURRENCE-ID") != "" {
				item.Recurring = true
			}
		}
		item.Zone = "Floating local time"
		allDay := false
		if ps := main.values("DTSTART"); len(ps) > 0 {
			allDay = strings.EqualFold(ps[0].param("VALUE"), "DATE") || len(ps[0].value) == 8
			if ps[0].param("TZID") != "" {
				item.Zone = ps[0].param("TZID")
			}
			if strings.HasSuffix(ps[0].value, "Z") {
				item.Zone = "UTC"
			}
			if allDay {
				item.Zone = "All day (end date exclusive)"
			}
		}
		out = append(out, CalendarEvent{Item: item, Method: method, Organizer: main.value("ORGANIZER"), Status: main.value("STATUS"), AllDay: allDay, root: group})
	}
	return out, nil
}

func calendarEmail(value string) (string, error) {
	if len(value) < 7 || !strings.EqualFold(value[:7], "mailto:") {
		return "", errors.New("calendar address must use mailto:")
	}
	address := value[7:]
	a, err := mail.ParseAddress(address)
	if err != nil || a.Address != address || strings.ContainsAny(address, "\r\n") {
		return "", errors.New("invalid calendar email address")
	}
	return address, nil
}

// Reply generates an iTIP REPLY for the invited account only. The original UID,
// sequence and recurrence identifiers are retained; other attendees and alarms
// are excluded from the response. The caller presents it in a normal mail draft.
func (e CalendarEvent) Reply(from, partstat string) (data []byte, organizer, attendee string, err error) {
	if e.root == nil {
		return nil, "", "", errors.New("no calendar event")
	}
	if e.Method != "REQUEST" || strings.EqualFold(e.Status, "CANCELLED") {
		return nil, "", "", errors.New("Accept/Decline is available for active REQUEST invitations")
	}
	if partstat != "ACCEPTED" && partstat != "DECLINED" {
		return nil, "", "", errors.New("unsupported participation status")
	}
	a, err := mail.ParseAddress(from)
	if err != nil {
		return nil, "", "", errors.New("configure a valid From address")
	}
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//MailSalonGUI//Calendar//EN", "METHOD:REPLY"}
	for _, c := range e.root.children {
		if c.name == "VTIMEZONE" {
			lines = append(lines, c.lines()...)
		}
	}
	for _, c := range e.root.children {
		if c.name != "VEVENT" {
			continue
		}
		if strings.EqualFold(c.value("STATUS"), "CANCELLED") {
			return nil, "", "", errors.New("invitation includes a cancelled instance")
		}
		if len(c.values("ORGANIZER")) != 1 {
			return nil, "", "", errors.New("invitation needs one organizer")
		}
		address, err := calendarEmail(c.value("ORGANIZER"))
		if err != nil {
			return nil, "", "", err
		}
		if organizer != "" && !strings.EqualFold(organizer, address) {
			return nil, "", "", errors.New("event organizers disagree")
		}
		organizer = address
		var matches []calendarProperty
		for _, p := range c.values("ATTENDEE") {
			address, err := calendarEmail(p.value)
			if err == nil && strings.EqualFold(address, a.Address) {
				matches = append(matches, p)
			}
		}
		if len(matches) != 1 {
			return nil, "", "", errors.New("this account's From address must match one invited attendee")
		}
		p := matches[0]
		if p.param("DELEGATED-TO") != "" || p.param("DELEGATED-FROM") != "" || p.param("SENT-BY") != "" {
			return nil, "", "", errors.New("delegated invitations need an external calendar client")
		}
		attendee = a.Address
		lines = append(lines, "BEGIN:VEVENT", "DTSTAMP:"+time.Now().UTC().Format("20060102T150405Z"))
		for _, p := range c.props {
			switch p.name {
			case "UID", "ORGANIZER", "SEQUENCE", "RECURRENCE-ID", "DTSTART", "DTEND", "DURATION", "SUMMARY", "LOCATION", "RRULE", "RDATE", "EXDATE":
				lines = append(lines, p.line)
			}
		}
		// Preserve descriptive attendee parameters, but replace prior RSVP state.
		head := "ATTENDEE"
		for _, param := range p.params {
			key, _, _ := strings.Cut(param, "=")
			if !strings.EqualFold(key, "PARTSTAT") && !strings.EqualFold(key, "RSVP") {
				head += ";" + param
			}
		}
		lines = append(lines, head+";PARTSTAT="+partstat+":"+p.value, "END:VEVENT")
	}
	lines = append(lines, "END:VCALENDAR")
	return serialize(lines), organizer, attendee, nil
}

func (e CalendarEvent) ImportData() ([]byte, error) {
	if e.root == nil {
		return nil, errors.New("no calendar event")
	}
	if e.Method != "" && e.Method != "PUBLISH" && e.Method != "REQUEST" {
		return nil, errors.New("only published events and active invitations can be imported")
	}
	if strings.EqualFold(e.Status, "CANCELLED") {
		return nil, errors.New("this event is cancelled")
	}
	for _, c := range e.root.children {
		if c.name != "VEVENT" {
			continue
		}
		start := c.value("DTSTART")
		valid := false
		for _, layout := range []string{"20060102", "20060102T150405", "20060102T150405Z"} {
			if _, err := time.Parse(layout, start); err == nil {
				valid = true
				break
			}
		}
		if !valid {
			return nil, errors.New("event needs a valid DTSTART before import")
		}
	}
	root := &calendarComponent{name: "VCALENDAR", children: e.root.children}
	for _, p := range e.root.props {
		if p.name != "METHOD" {
			root.props = append(root.props, p)
		}
	}
	if root.value("PRODID") == "" {
		p, _ := calendarProp("PRODID:-//MailSalonGUI//Calendar//EN")
		root.props = append(root.props, p)
	}
	// CalDAV stored resources cannot contain scheduling METHOD properties.
	return serialize(root.lines()), nil
}

func calendarTarget(c config.Collection, uid string) (*Item, error) {
	items, err := Load(c)
	if err != nil {
		return nil, err
	}
	var target *Item
	for _, item := range items {
		if item.UID != uid {
			continue
		}
		if target != nil {
			return nil, errors.New("collection contains multiple files with this UID")
		}
		copy := item
		target = &copy
	}
	return target, nil
}

// CalendarTarget finds an existing UID, regardless of the sync tool's filename.
func CalendarTarget(c config.Collection, uid string) (*Item, error) {
	if c.Protocol != "caldav" {
		return nil, errors.New("iCalendar import requires a CalDAV collection")
	}
	return calendarTarget(c, uid)
}

// ImportCalendar checks for duplicates and stale confirmations under the shared
// sync lock. UID-derived filenames use a hash, allowing UIDs with slashes safely.
func ImportCalendar(c config.Collection, e CalendarEvent, original *Item) error {
	if c.Protocol != "caldav" {
		return errors.New("iCalendar import requires a CalDAV collection")
	}
	data, err := e.ImportData()
	if err != nil {
		return err
	}
	release, err := lock(c)
	if err != nil {
		return err
	}
	defer release()
	current, err := calendarTarget(c, e.UID)
	if err != nil {
		return err
	}
	if original != nil {
		if current == nil || current.Path != original.Path || !bytes.Equal(current.Data, original.Data) {
			return errors.New("event changed; reopen the import before replacing it")
		}
		// An instance-only update must never erase the rest of an existing series.
		old, err := ParseCalendar(original.Data, "")
		if err != nil {
			return err
		}
		if len(old) != 1 {
			return errors.New("existing calendar resource must contain one UID")
		}
		incomingMain, existingMain := false, false
		for _, c := range e.root.children {
			if c.name == "VEVENT" && c.value("RECURRENCE-ID") == "" {
				incomingMain = true
			}
		}
		for _, c := range old[0].root.children {
			if c.name == "VEVENT" && c.value("RECURRENCE-ID") == "" {
				existingMain = true
			}
		}
		if !incomingMain && existingMain {
			return errors.New("instance-only update cannot replace a full series; use Edit source to merge it")
		}
		if !incomingMain {
			if !sameCalendarInstances(e.root, old[0].root) {
				return errors.New("instance-only update cannot remove other instances; use Edit source to merge it")
			}
		}
		return saveLocked(c, original, data, e.UID)
	}
	if current != nil {
		return errors.New("event already exists; reopen the import to review its replacement")
	}
	if _, err := Parse(c, data); err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(e.UID))
	path := filepath.Join(c.LocalDir, fmt.Sprintf("%x.ics", hash))
	// Do not overwrite an unrelated file with an unexpectedly matching filename.
	// The collection lock also excludes competing imports and MailSalonSync.
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return errors.New("calendar import filename already exists or cannot be checked")
	}
	return write(path, data)
}

func sameCalendarInstances(a, b *calendarComponent) bool {
	ids := map[string]bool{}
	for _, c := range a.children {
		if c.name == "VEVENT" {
			ids[c.value("RECURRENCE-ID")] = true
		}
	}
	count := 0
	for _, c := range b.children {
		if c.name == "VEVENT" {
			count++
			if !ids[c.value("RECURRENCE-ID")] {
				return false
			}
		}
	}
	return count == len(ids)
}
