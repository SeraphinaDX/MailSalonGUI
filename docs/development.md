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

`messageList` owns a separate set of selected row IDs; Fyne's embedded List
tracks the active row for scrolling. Row highlights and bulk targets use that
set. Selection changes increment a revision so an open confirmation or queued
action cannot follow a different selection. Filtering remaps selected paths
onto visible rows, and folder/account reloads clear selection.

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
