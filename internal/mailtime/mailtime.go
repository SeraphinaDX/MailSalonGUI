// SPDX-License-Identifier: GPL-3.0-only

// Package mailtime formats incoming message dates for the reader's timezone.
// Parsing and storage keep the original timestamps so message ordering and mail
// contents are not changed by display preferences.
package mailtime

import (
	"net/mail"
	"strings"
	"time"
)

// InLocation converts an instant for display without changing the instant.
// A nil location means the machine's local timezone.
func InLocation(t time.Time, location *time.Location) time.Time {
	if location == nil {
		location = time.Local
	}
	return t.In(location)
}

// DisplayHeaderDate shows a valid RFC 5322 Date header in the reader's timezone
// with an explicit numeric offset. Missing or malformed headers are left alone
// instead of presenting a possibly incorrect converted time.
func DisplayHeaderDate(header string, location *time.Location) string {
	if strings.TrimSpace(header) == "" {
		return header
	}
	date, err := (mail.Header{"Date": []string{header}}).Date()
	if err != nil {
		return header
	}
	return InLocation(date, location).Format(time.RFC1123Z)
}
