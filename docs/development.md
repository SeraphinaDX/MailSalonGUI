# Working on MailSalonGUI

Start at `cmd/MailSalonGUI/main.go`. It reads flags and TOML, constructs the Fyne
application and starts the GUI. The GUI imports local backend packages from this
project; installing the terminal MailSalon client is not required.

## Window and worker ownership

Fyne callbacks own GUI state. Scanning Maildirs, parsing messages, running
commands and loading collections happen in workers. Inputs are captured before
a worker starts. Results pass through `App.post`, which uses `fyne.Do` to return
to Fyne's event goroutine and ignores results after the application closes.

Folder scans and message parsing have separate generation counters. Changing a
folder, account or search invalidates pending results. A late parse completion
cannot replace a newer selection or mark an unrelated message read. An entry is
marked read only after the current selection has parsed successfully.

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
