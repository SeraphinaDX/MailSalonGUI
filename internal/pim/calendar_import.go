// SPDX-License-Identifier: GPL-3.0-only

package pim

import (
	"crypto/sha256"
	"fmt"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"os"
	"path/filepath"
	"strings"
)

// ICalendar makes a local copy that suppresses server-side scheduling. The
// original emailed invitation, including its participant parameters, is intact.
func (e CalendarEvent) ICalendar() ([]byte, error) {
	data, err := e.ImportData()
	if err != nil {
		return nil, err
	}
	lines := unfold(data)
	for i, line := range lines {
		p, err := calendarProp(line)
		if err != nil {
			return nil, err
		}
		if p.name != "ATTENDEE" && p.name != "ORGANIZER" {
			continue
		}
		params := []string{}
		for _, param := range p.params {
			key, _, _ := strings.Cut(param, "=")
			if !strings.EqualFold(key, "SCHEDULE-AGENT") {
				params = append(params, param)
			}
		}
		params = append(params, "SCHEDULE-AGENT=NONE")
		lines[i] = p.name + ";" + strings.Join(params, ";") + ":" + p.value
	}
	return serialize(lines), nil
}

// ImportCalendarIfAbsent keeps the terminal client's non-replacement policy.
// Both formats use the same collection lock, UID lookup and atomic writer as
// the GUI's reviewed replacement workflow.
func ImportCalendarIfAbsent(c config.Collection, event CalendarEvent) (bool, error) {
	var data []byte
	var err error
	switch c.Protocol {
	case "caldav":
		data, err = event.ICalendar()
	case "jmap-calendars":
		data, err = event.JSCalendar()
	default:
		return false, fmt.Errorf("destination is not a calendar")
	}
	if err != nil {
		return false, err
	}
	if _, err := Parse(c, data); err != nil {
		return false, err
	}
	release, err := lock(c)
	if err != nil {
		return false, err
	}
	defer release()
	current, err := calendarTarget(c, event.UID)
	if err != nil {
		return false, err
	}
	if current != nil {
		return false, nil
	}
	sum := sha256.Sum256([]byte(event.UID))
	path := filepath.Join(c.LocalDir, fmt.Sprintf("%x%s", sum, Extension(c)))
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return false, fmt.Errorf("import destination already exists")
	}
	if err := write(path, data); err != nil {
		return false, err
	}
	return true, nil
}
