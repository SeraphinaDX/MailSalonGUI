# Everyday use

## Mail

Select an account and folder. Messages appear newest first. Unread messages
have a dot and bold subject. Selecting one message marks it read after successful
parsing. A group selection does not mark all its messages read just by selecting
them. Use **Read / unread** to change the selected messages' flags. Drag pane dividers
for a wider list or preview. Search matches sender and subject in this folder.

HTML mail uses a simplified formatted preview with headings, paragraphs, bold
and italic text, lists, table rows and clickable links. When a message has both
HTML and plain alternatives, the preview shows HTML while replies/forwards
retain the plain alternative for quoting. Layout follows the content structure;
complex CSS, columns and browser-specific layouts are not reproduced exactly.
Remote images are not loaded; image alt text and linked-button labels are kept.
Scripts and embedded web content do not run. Web links open through your default
browser, and mailto links through the system's default mail handler. URLs in
plain mail are clickable too. Very long bare URLs have shortened preview labels,
with their complete destinations retained for clicking and copying.
**Copy body** copies readable text including link destinations; **Source** shows and copies
the original MIME message. Each attachment has a Save button and a file dialog.
The dialog initially opens the account's download directory when it exists.

Select several messages with **Ctrl-click** to add/remove individual messages
or **Shift-click** to select a range from the last selection anchor. **Shift+Up/Down**
extends or shrinks that range; Shift+Home/End extends it to the folder's ends.
**Ctrl+A**, while the message list is focused, selects all messages currently
shown by the search. **Escape** clears selection. Selected rows are highlighted
and the list shows the selection count. Search keeps selected messages that
remain visible and drops hidden messages; changing folders/accounts or reloading
clears selection.

**Archive** moves all selected messages into the existing configured archive folder.
**Read / unread** marks the whole selection read if any selected message is
unread; otherwise it marks the selection unread. The context menu also offers
explicit **Mark as read** and **Mark as unread** actions.
**Delete** asks once for confirmation, showing the message count, and moves
the selected messages to Trash. Deleting a selection already in Trash asks
once before permanently removing it. Cancel leaves the messages untouched.
If the view or selection changes while confirmation is open, it is cancelled.
Operations continue if one message fails and report the failures; successful
changes are retained.
Press **Delete** or **Backspace** with the message list focused to open the same confirmation.
The numpad **. / Del** key also opens confirmation in the message list, with
Num Lock on or off. In text fields it keeps its normal editing behavior.
If a key does not work, open **Help → Keyboard diagnostic**, press the key in
that window, and choose **Copy report**. The report includes the app version,
received key name and scan code, expected keypad Delete scan code, and whether
it matches. The diagnostic window does not perform mail actions.
Right-click an email for reply/reply all, forward, mark read/unread, archive,
delete and source actions. Right-clicking within the selection preserves the
group; right-clicking outside it selects just that message. Reply, forward and
source menu actions require a single selection. Reply actions
become available after its preview loads; message changes briefly disable
conflicting actions. A reload or different selection closes the menu.
Sync with your configured tool to propagate local changes.

## Compose and drafts

**Compose** opens an independent window. Choose From, enter To/Cc/Bcc, subject
and body, and attach files. Recipient fields accept comma-separated RFC mail
addresses. Use the contact button next to a recipient field to find and insert
an address from that account's books or shared books.

Reply chooses the account addressed in To, then Cc, falling back to the active
account. It uses Reply-To when present. Reply all omits your own configured
identities and removes duplicate recipients. Forwards include original
attachments, which can be removed individually.

**Save draft** stores the current fields and attachment references locally.
Forwarded attachments are stored with the draft. Attached local files must
still exist when you send. The draft directory sits beside the GUI's default
configuration in `mailsalongui/drafts`; custom `-config` paths do not change it.
Drafts are private files, with no password encryption, and are not synced to a
remote Drafts folder. Saving does not send mail.

**Drafts** in the main window lets you reopen or delete saved drafts. A draft
whose account has been removed cannot be sent until that account is restored.
Closing a compose window offers Save and close, Discard, or Keep editing.
Quitting with open compositions prompts to save all of them as local drafts.

Sending asks for confirmation. Wait for completion; the window cannot be closed
while sending. Success removes the saved draft. Failure retains the composition
and displays the command error. A timeout can occur after a server accepted the
message, so check Sent before manually retrying an uncertain delivery.
MailSalonGUI does not add a second Sent copy; the sender/server owns sent-mail
storage. Bcc headers are supplied to the send tool so it can derive recipients;
that tool must remove Bcc before transmission (as `msmtp -t` does).

## Contacts and calendar

Select a collection. Search and click items for details. **New** provides a
simple contact or single-event form. Dates are ISO `YYYY-MM-DD`, or
`YYYY-MM-DDTHH:MM:SS` with an optional IANA timezone. An all-day end date is
exclusive, and must be later than the start.

**Edit source** opens the original vCard/iCalendar/JSON text. Keep the UID and
properties you want to retain. Saving validates the format and refuses to
overwrite an item changed since the editor opened. **Delete** retains a copy in
the sync tool's local `.mss-trash` directory. Calendar recurrence data is
preserved in the original resource while supported rules are expanded for display.

A view without configured collections explains how to add them in TOML and
leaves its list empty. It is safe to use mail without contacts or calendars.

### Calendar views

Choose **Month**, **Week**, **Day**, or **Agenda** above the calendar. Month shows
a six-week grid starting on Monday; a busy day has a **+N more…** link to Day
view. Click a day heading to open that date. Week and Day show a scrollable
24-hour schedule, starting at 08:00, with separate all-day rows. Overlapping
appointments use separate columns. Hover a truncated event for its full label,
or click it for details. Agenda lists the next 30 days, including continuations
of multi-day events.

Use the arrows to move by a month, week, day, or 30-day agenda period. **Today**
returns to the current date; **Go to date…** accepts `YYYY-MM-DD`. Calendar search
filters the selected collection in every view. **Local time / UTC** controls
the displayed clock; all-day dates and floating events stay on their civil dates.
IANA timezones are converted for display and recurrence retains the event's
timezone through daylight-saving changes.

Click an appointment to show its actual occurrence dates and details. **New
event** starts with the selected date at 09:00–10:00. **Edit source** and **Delete**
operate on the complete local resource; for a recurring event that means the
whole series. This release does not offer dragging/resizing, instance-only
editing, or reminders.

Common daily/weekly/monthly/yearly rules, additional dates, excluded dates and
moved/cancelled recurrence exceptions are displayed for iCalendar and JSCalendar
events. Expansion is limited to the visible interval and bounded so dense or
very old rules cannot occupy the interface indefinitely. **Unplaced events…**
shows sources whose timezone, duration, or recurrence cannot be displayed;
**Show details** selects their original resource for inspection/editing. Custom
non-IANA timezone definitions, sub-daily rules, iCalendar RDATE periods and
THISANDFUTURE updates are preserved but require source inspection. JSCalendar
non-Gregorian rules and nested location override patches also remain unplaced.
Rules needing skip adjustments or nonexistent local wall times are likewise
reported for inspection rather than silently shifted to another hour/date.

### Calendar invitations in mail

Named `.ics` attachments and inline `text/calendar` parts appear as event cards
below the mail body, even without a filename. Cards show the event title, start,
end/duration, timezone, location, organizer, description and recurrence status.
All-day end dates are exclusive. Timezone values stay as supplied by the sender;
custom timezone definitions and recurrence rules are preserved in the source.
Attachments remain available to save, including calendars that cannot be parsed.
Calendars containing multiple UIDs get a separate card for each event; recurrence
exceptions stay with their series.

**Add to Calendar** opens a chooser for CalDAV collections available to the
current mail account. Saving writes a local `.ics` resource for MailSalonSync
to upload on the next sync. It retains the UID, timezone definitions, recurrence
exceptions, alarms and event extensions, and removes the scheduling `METHOD`
property required only for emailed calendar messages. An existing UID asks
before replacing its source; a changed file or busy sync prevents replacement.
Instance-only updates cannot replace a full recurring series; merge those using
**Edit source**. JMAP calendar JSON conversion is not supported. Without a
CalDAV collection, save the attachment for use in another calendar application.

**Accept… / Decline…** opens an ordinary mail draft addressed to the calendar
organizer with a `METHOD:REPLY` attachment and the chosen participation status.
Review it and press **Send** to notify the organizer through the account's usual
send command. Nothing is sent by opening the draft. The From account is fixed
to the matching invited attendee, including after saving/reopening the draft.
Adding the event locally is a separate action and does not send an RSVP.
Responses require an active `REQUEST` invitation and a matching account From
address; delegated invitations are not handled. Cancellation messages are
shown but do not automatically remove events, and cannot be imported or replied
to with Accept/Decline. Replies from other attendees are previewed only.

## Shortcuts

| Context | Shortcut | Action |
| --- | --- | --- |
| Main window | Ctrl+N | New message |
| Main window | Ctrl+R | Reply to selected message |
| Main window | Ctrl+F | Focus mail search |
| Main window | F5 | Sync |
| Message list | Delete / Backspace / numpad . / Del | Confirm deleting selected mail |
| Message list | Up / Down / Home / End | Select a message |
| Message list | Ctrl-click | Add/remove a message from selection |
| Message list | Shift-click / Shift+Up/Down/Home/End | Select a range |
| Message list | Ctrl+A | Select all currently shown messages |
| Message list | Ctrl+Space | Toggle the active message's selection |
| Message list | Escape | Clear selection |
| Compose | Ctrl+S | Save local draft |
| Compose | Ctrl+Enter | Confirm sending |
| Editable fields | Ctrl+A/C/X/V | Select, copy, cut, paste |
| Widgets | Tab / Shift+Tab | Move input focus |

Fyne manages caret focus, so only the active editable field shows its cursor.
