# Working on MailSalonGUI

The entry points are `cmd/MailSalonGUI/main.go` (Fyne) and `cmd/MailSalon/main.go`
(gotui). `internal/gui` and `internal/ui` own their respective interfaces. Both
import the same backend packages in this module; neither executable requires
the other to be installed. See [Consolidation](consolidation.md) for provenance,
build boundaries and migration.

The shared core must not import Fyne, GLFW, gotui or tcell. Terminal builds and
tests use `CGO_ENABLED=0`; GUI builds retain their existing zgo/cgo path. `make
test-tui` checks core/terminal packages without the graphics driver; `make test`
and `make check` cover both interfaces with the Fyne software driver. Versions
live in `internal/version`; update GUI packaging metadata with the GUI version.

Calendar invitation decoding and bounded recurrence projection live in
`internal/pim`. MIME parsing populates the common decoded invitation model;
the GUI and terminal render it independently. JMAP import conversion retains
the terminal client's explicit unsupported-property checks. Terminal imports
use the non-replacement API; GUI imports retain reviewed, stale-file-guarded
replacement. Both use the same collection locking and atomic writer.

## Window and worker ownership

Fyne callbacks own GUI state. Scanning Maildirs, parsing messages, running
commands and loading collections happen in workers. Inputs are captured before
a worker starts. Results pass through `App.post`, which uses `fyne.Do` to return
to Fyne's event goroutine and ignores results after the application closes.

Folder scans and message parsing have separate generation counters. Changing a
folder, account or search invalidates pending results. A late parse completion
cannot replace a newer selection or mark an unrelated message read. An entry is
marked read only after the current selection has parsed successfully.

`messageList` owns a separate set of selected row IDs; Fyne's embedded List
tracks the active row for scrolling. Row highlights and bulk targets use that
set. Selection changes increment a revision so an open confirmation or queued
action cannot follow a different selection. Filtering remaps selected paths
onto visible rows, and folder/account reloads clear selection.

Message-list letters use `TypedRune`, while navigation/Delete use `TypedKey`.
Desktop drivers deliver both events for printable keys; handling letters once
avoids duplicate actions. `mail_shortcuts.go` checks list focus, the active Mail
tab, overlays and single-message eligibility before calling existing actions.
Ctrl/Alt/Super combinations stay on the shortcut path; Shift+r arrives as `R`.
Keep the in-app Help text and usage shortcut table in sync when adding keys.

Bulk operations capture immutable entries on the event goroutine, wait for any
automatic mark-read rename, then run sequentially in one worker. Explicit flag
changes invalidate pending parses so they cannot undo mark-unread. Flag changes
update renamed paths in both list models; moves and partial failures reload
the Maildir. Each failed message is reported without undoing successful work.

Synchronization is sequential across distinct receive command lines, with a
single in-progress flag. This prevents duplicate invocations when multiple
accounts share `MailSalonSync -plain sync`. The timer uses the same entry point
as the Sync button. Commands return their errors and captured output; the GUI
keeps compositions when delivery fails.

Composition windows capture a draft before sending, then disable fields during
delivery. Successful delivery is treated as final even if local draft cleanup
fails. The user sees a cleanup notice rather than being invited to resend.

## Local storage

Maildir state changes use its native filenames and directories. `internal/pim`
uses the same lock and local file formats as MailSalonSync. Source editors save
only if the original bytes still match, preserving server changes made after
an editor was opened.

Local composition drafts are private JSON files with atomic replacement. They
store file paths for newly attached files, and bytes for forwarded attachments.
They are separate from synchronized mail folders.

## Message body previews

MIME parsing keeps plain alternatives for quoting and structured HTML spans
for display. `multipart/alternative` chooses one display version; mixed and
related body sections stay in source order. HTML attachments remain attachments.
Text character sets are decoded before `golang.org/x/net/html` parses markup.

`internal/mimeutil/html.go` converts content into text/style/link spans with
structural line breaks and HTML whitespace rules. `internal/gui/message_body.go`
turns those spans into Fyne text and hyperlink segments. Raw message markup is
never interpreted as Markdown. Images are represented by alt text or linked
labels, and scripts/styles/hidden preheaders are excluded. Only HTTP, HTTPS and
mailto links are actionable. HTML CSS layouts and remote image rendering are
outside this simplified preview.

`ParsedMessage.DisplayText` includes link destinations for Copy body. Replies
and forwards use `Body`, preserving a meaningful plain alternative. For HTML-only
mail, `Body` contains the readable HTML conversion with its destinations.

## Icon and desktop packaging

`internal/assets/icon.png` is embedded through `internal/assets/assets.go`.
The GUI sets the application icon before creating any windows. Keep the app
ID, desktop-entry filename, installed icon name and Fyne metadata consistent.
The icon's generation prompt and license are in `internal/assets/README.md`.

Show top-level windows through `showWindow`; the main entry point calls
`App.Run`. Fyne creates native handles during `Show`, so X11 desktop identity
is applied afterwards. Both WM_CLASS fields and KDE's
`_KDE_NET_WM_DESKTOP_FILE` use `assets.AppID`, matching the launcher's
`StartupWMClass` and filename. This also groups auxiliary windows under the
same launcher. Other drivers retain Fyne's native behavior.

`make install` builds and installs the binary, icon and launcher under PREFIX,
which defaults to `~/.local`. `DATADIR` defaults to PREFIX/share; `DESTDIR` is a
staging root and is never included in the launcher's executable or icon paths. Prefixes
must be absolute and contain no line breaks, equals signs or percent signs.
Desktop Exec arguments use the freedesktop string/argument escaping rules.
The installer invokes no shell through the launcher.
The launcher's `Icon` is an absolute path with desktop-string escaping,
separate from `Exec` argument escaping. Install and uninstall refresh the
desktop database and KDE application cache when their tools are available;
staged installations skip cache updates. Cache failures are reported without
undoing the successfully installed files.

`cmd/MailSalonGUI/FyneApp.toml` supplies app identity and icon metadata to Fyne
packaging tools. Update its version alongside `gui.Version` on release.

## Validation

```sh
go test -tags ci ./...
go test -race -tags ci ./...
go vet -tags ci ./...
make build
```

The `ci` tag selects Fyne's software driver. Tests create temporary mailboxes
and use temporary shell commands for sync/send integration; no real mail is
sent. To regenerate the visual review screenshots:

```sh
MAILSALONGUI_SCREENSHOT_DIR=./docs/screenshots go test -tags ci ./internal/gui -run TestDemoMailComposeDraftAndCollections -count=1
```

The screenshot directory is relative to the package under `go test`; use an
absolute path if you want the root-level `docs/screenshots` folder.

The real GnuPG round trip requires an environment that allows Unix agent
sockets. Run it explicitly with:

```sh
MAILSALONGUI_TEST_GPG=1 go test ./internal/pgp
```

Fyne upstream API and threading guidance:
[Quick start](https://docs.fyne.io/started/quick/) and
[Using goroutines](https://docs.fyne.io/started/goroutines/).

Calendar mail parsing and guarded imports live in `internal/pim/invitations.go`;
event cards and draft creation live in `internal/gui/calendar_mail.go`. MIME
calendar alternatives are retained as attachments even without filenames. RSVP
drafts retain a fixed invited From address and a calendar response attachment.
Relevant protocol references are [iTIP RFC 5546 §3.2.3](https://www.rfc-editor.org/rfc/rfc5546.html#section-3.2.3),
[iMIP RFC 6047](https://www.rfc-editor.org/rfc/rfc6047.html), and
[CalDAV RFC 4791 §4.1](https://www.rfc-editor.org/rfc/rfc4791.html#section-4.1).

`internal/pim/occurrences.go` adapts local iCalendar/JSCalendar resources to
bounded display occurrences using `github.com/teambition/rrule-go` (MIT).
`internal/gui/calendar_view.go` controls date ranges, cancellable workers and
selection; `calendar_widgets.go` owns month cells and overlapping time-grid
layouts. Display occurrences retain their original source item, so source
editing never serializes a flattened recurrence. Tests cover timezone/DST and
exception semantics as well as stale navigation results. Relevant references:
[iCalendar RFC 5545](https://www.rfc-editor.org/rfc/rfc5545.html) and
[JSCalendar RFC 8984](https://www.rfc-editor.org/rfc/rfc8984.html).

Calendar screenshot fixtures can be regenerated with an absolute output path:

```sh
MAILSALONGUI_SCREENSHOT_DIR=/tmp/calendar-views go test -tags ci ./internal/gui -run TestCalendarViewsNavigationAndEventDetails -count=1
```
