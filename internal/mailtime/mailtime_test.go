// SPDX-License-Identifier: GPL-3.0-only

package mailtime

import (
	"strings"
	"testing"
	"time"
)

func TestDisplayHeaderDateUsesReaderTimezone(t *testing.T) {
	toronto, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skipf("IANA timezone data unavailable: %v", err)
	}

	for _, tt := range []struct {
		name, input, want string
	}{
		{
			name: "daylight saving",
			input: "Mon, 26 Oct 2026 15:30:00 +0000",
			want: "Mon, 26 Oct 2026 11:30:00 -0400",
		},
		{
			name: "standard time",
			input: "Sat, 10 Jan 2026 15:30:00 +0000",
			want: "Sat, 10 Jan 2026 10:30:00 -0500",
		},
		{
			name: "crosses date boundary",
			input: "Tue, 27 Oct 2026 01:00:00 +0000",
			want: "Mon, 26 Oct 2026 21:00:00 -0400",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayHeaderDate(tt.input, toronto); got != tt.want {
				t.Fatalf("DisplayHeaderDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDisplayHeaderDatePreservesMalformedHeader(t *testing.T) {
	for _, raw := range []string{"", "not a mail date", "Tue, 27 Oct 2026 25:61:00 +0000"} {
		if got := DisplayHeaderDate(raw, time.UTC); got != raw {
			t.Errorf("malformed Date header was altered: %q -> %q", raw, got)
		}
	}
}

func TestInLocationPreservesInstant(t *testing.T) {
	sent := time.Date(2026, 10, 27, 1, 30, 0, 0, time.UTC)
	other := InLocation(sent, time.FixedZone("EDT", -4*60*60))
	if !sent.Equal(other) || !strings.Contains(other.Format(time.RFC1123Z), "21:30:00 -0400") {
		t.Fatalf("time conversion changed instant or offset: %s -> %s", sent, other)
	}
}
