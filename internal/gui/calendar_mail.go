// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

// Calendar cards share the attachment area, so inline calendar alternatives are
// visible even when the selected message body is HTML or ordinary text.
func (a *App) calendarAttachment(att mimeutil.Attachment, p *mimeutil.ParsedMessage, account int) fyne.CanvasObject {
	ctype, _, _ := mime.ParseMediaType(att.MIMEType)
	if ctype != "text/calendar" && !strings.EqualFold(filepath.Ext(att.Filename), ".ics") {
		return nil
	}
	events, err := pim.ParseCalendar(att.Data, att.CalendarMethod)
	if err != nil {
		label := widget.NewLabel("Calendar attachment: " + err.Error() + ". You can still save the file.")
		label.Wrapping = fyne.TextWrapWord
		return label
	}
	cards := container.NewVBox()
	for _, event := range events {
		e := event
		summary := fmt.Sprintf("Start: %s\nEnd / duration: %s\nTimezone: %s\nLocation: %s\nOrganizer: %s", calendarDate(e.Start), calendarDate(e.End), e.Zone, e.Location, strings.TrimPrefix(e.Organizer, "mailto:"))
		subtitle := "Calendar event"
		if e.Method == "REQUEST" {
			subtitle = "Meeting invitation"
		}
		if e.Method == "REPLY" {
			subtitle = "Invitation response"
		}
		if e.Recurring {
			summary += "\nRecurring event — recurrence rules and exceptions are preserved."
		}
		if e.Method == "CANCEL" || strings.EqualFold(e.Status, "CANCELLED") {
			summary += "\nCancelled event"
		}
		if e.Notes != "" {
			summary += "\n\n" + e.Notes
		}
		label := widget.NewLabel(summary)
		label.Wrapping = fyne.TextWrapWord
		add := widget.NewButton("Add to Calendar", func() { a.importCalendarEvent(e, account) })
		if _, err := e.ImportData(); err != nil {
			add.Disable()
		}
		accept := widget.NewButton("Accept…", func() { a.calendarReply(e, p, account, "ACCEPTED") })
		decline := widget.NewButton("Decline…", func() { a.calendarReply(e, p, account, "DECLINED") })
		help := "Accept/Decline opens a reply draft. Send it to notify the organizer. Add to Calendar saves locally."
		if _, _, _, err := e.Reply(a.cfg.Accounts[account].From, "ACCEPTED"); err != nil {
			accept.Disable()
			decline.Disable()
			help = "Accept/Decline unavailable: " + err.Error() + "."
		}
		if _, err := e.ImportData(); err != nil {
			help += " Import unavailable: " + err.Error() + "."
		}
		info := widget.NewLabel(help)
		info.Wrapping = fyne.TextWrapWord
		cards.Add(widget.NewCard(e.Title, subtitle, container.NewVBox(label, container.NewHBox(add, accept, decline), info)))
	}
	return cards
}

func calendarDate(value string) string {
	if len(value) > 10 && value[10] == 'T' {
		return value[:10] + " " + value[11:]
	}
	return value
}

func (a *App) calendarReply(e pim.CalendarEvent, message *mimeutil.ParsedMessage, account int, partstat string) *composer {
	data, organizer, attendee, err := e.Reply(a.cfg.Accounts[account].From, partstat)
	if err != nil {
		a.fail(err)
		return nil
	}
	status := "Accepted"
	if partstat == "DECLINED" {
		status = "Declined"
	}
	draft := mimeutil.Draft{To: organizer, Subject: status + ": " + e.Title,
		Body:                 status + ": " + e.Title + "\nStart: " + calendarDate(e.Start) + "\nTimezone: " + e.Zone,
		CalendarReplyAddress: attendee,
		MemoryAttachments:    []mimeutil.Attachment{{Filename: "reply.ics", MIMEType: "text/calendar; charset=utf-8; method=REPLY", Data: data}},
	}
	if message != nil {
		draft.InReplyTo = message.MessageID
		draft.References = strings.TrimSpace(message.References + " " + message.MessageID)
	}
	c := a.compose(draft, account, "")
	c.status.SetText("Calendar response draft — press Send to notify " + organizer + ". Add to Calendar is separate.")
	return c
}

func (a *App) importCalendarEvent(e pim.CalendarEvent, account int) {
	collections := a.visibleCollections(false, account)
	calendars := collections[:0]
	for _, c := range collections {
		if c.Protocol == "caldav" {
			calendars = append(calendars, c)
		}
	}
	if len(calendars) == 0 {
		dialog.ShowInformation("Choose a CalDAV calendar", "Add a CalDAV collection in Settings to import iCalendar events. MailSalonSync can upload its local directory. You can also save the .ics attachment for another calendar app. JMAP calendar conversion is not supported.", a.Window)
		return
	}
	w := a.Fyne.NewWindow("Add to Calendar — " + e.Title)
	w.Resize(fyne.NewSize(560, 300))
	names := []string{}
	for _, c := range calendars {
		names = append(names, c.Name)
	}
	chooser := widget.NewSelect(names, nil)
	chooser.SetSelectedIndex(0)
	info := widget.NewLabel("Save this event locally. Your sync tool can upload it. This does not send an RSVP.")
	info.Wrapping = fyne.TextWrapWord
	busy := false
	var add *widget.Button
	setBusy := func(value bool) {
		busy = value
		if value {
			add.Disable()
			chooser.Disable()
		} else {
			add.Enable()
			chooser.Enable()
		}
	}
	save := func(original *pim.Item) {
		index := chooser.SelectedIndex()
		if index < 0 {
			return
		}
		c := calendars[index]
		setBusy(true)
		info.SetText("Saving event…")
		go func() {
			err := pim.ImportCalendar(c, e, original)
			a.post(func() {
				setBusy(false)
				if err != nil {
					info.SetText(err.Error())
					return
				}
				w.SetCloseIntercept(nil)
				w.Close()
				a.calendar.reloadCollections()
				a.status.SetText("Saved “" + e.Title + "” to " + c.Name + ". Sync to upload it.")
			})
		}()
	}
	add = widget.NewButton("Add to Calendar", func() {
		if busy || chooser.SelectedIndex() < 0 {
			return
		}
		c := calendars[chooser.SelectedIndex()]
		setBusy(true)
		info.SetText("Checking for an existing event…")
		go func() {
			original, err := pim.CalendarTarget(c, e.UID)
			a.post(func() {
				if err != nil {
					setBusy(false)
					info.SetText(err.Error())
					return
				}
				if original == nil {
					save(nil)
					return
				}
				dialog.ShowConfirm("Replace existing event?", "This calendar already contains “"+original.Title+"” ("+original.Start+").\nReplace its source with the attached event “"+e.Title+"” ("+e.Start+")?", func(ok bool) {
					if ok {
						save(original)
					} else {
						setBusy(false)
						info.SetText("Existing event kept.")
					}
				}, w)
			})
		}()
	})
	w.SetCloseIntercept(func() {
		if !busy {
			w.SetCloseIntercept(nil)
			w.Close()
		}
	})
	w.SetContent(container.NewVBox(widget.NewLabel(e.Title), chooser, info, add))
	showWindow(w)
}
