// SPDX-License-Identifier: GPL-3.0-only
package mimeutil

import (
	"bytes"
	"encoding/base64"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestCalendarMIMEParts(t *testing.T) {
	calendar := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:a\r\nSUMMARY:Café\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	for _, tc := range []struct{ name, headers, body string }{
		{"inline", "Content-Type: text/calendar; method=REQUEST", calendar},
		{"attachment", "Content-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=event.ICS", calendar},
		{"base64", "Content-Type: text/calendar\r\nContent-Transfer-Encoding: base64", base64.StdEncoding.EncodeToString([]byte(calendar))},
		{"charset", "Content-Type: text/calendar; charset=iso-8859-1", strings.ReplaceAll(calendar, "é", "\xe9")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseBytes([]byte("From: a@example.test\r\n" + tc.headers + "\r\n\r\n" + tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if p.Body != "" || len(p.Attachments) != 1 || string(p.Attachments[0].Data) != calendar {
				t.Fatalf("lost calendar part: %+v", p)
			}
			if p.Attachments[0].Filename == "" {
				t.Fatal("unnamed calendar cannot be saved as ICS")
			}
			if tc.name == "inline" && p.Attachments[0].CalendarMethod != "REQUEST" {
				t.Fatal("lost MIME method")
			}
		})
	}
	p, err := ParseFile("testdata/invitation.eml")
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != "Please join our community meeting." || len(p.Attachments) != 1 || !bytes.Contains(p.Attachments[0].Data, []byte("mailto:demo@example.com")) {
		t.Fatalf("wrong alternative selection: %+v", p)
	}
}

func TestCalendarReplyMIMEAndIdentity(t *testing.T) {
	d := Draft{From: "Demo <demo@example.com>", To: "organizer@example.test", Subject: "Accepted", Body: "Accepted", CalendarReplyAddress: "demo@example.com", MemoryAttachments: []Attachment{{Filename: "reply.ics", MIMEType: "text/calendar; charset=utf-8; method=REPLY", Data: []byte("METHOD:REPLY\r\n")}}}
	raw, err := Build(d)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(m.Header.Get("Content-Type"))
	r := multipart.NewReader(m.Body, params["boundary"])
	if _, err := r.NextPart(); err != nil {
		t.Fatal(err)
	}
	part, err := r.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	ctype, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err != nil || ctype != "text/calendar" || params["method"] != "REPLY" || params["name"] != "reply.ics" {
		t.Fatalf("invalid scheduling MIME: %s", part.Header.Get("Content-Type"))
	}
	d.From = "wrong@example.test"
	if _, err := Build(d); err == nil {
		t.Fatal("sent response from wrong identity")
	}
}
