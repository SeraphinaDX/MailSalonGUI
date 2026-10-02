# Validation

## MailSalonGUI 0.3.1

- The full software-driver test suite and vet passed for both interfaces and
  shared packages. GUI and Maildir race tests passed; terminal/core tests also
  passed with `CGO_ENABLED=0`.
- Both native Linux executables built successfully and report MailSalonGUI
  0.3.1 and MailSalon 0.9.0.
- Canvas drag tests moved one message to Sent and verified ordinary folder
  clicks still open folders. Local Maildir tests moved selected groups and an
  unselected message, including a discovered Maildir++ nested folder.
- A drop during automatic mark-read waited for the rename and used the current
  path. Cancelled, outside, same-folder and stale view/selection/account drops
  left messages in place. Tests reject recycled targets, sync/modal drops and
  gestures begun while busy, allowing a new gesture after release.
- Maildir tests verify message bytes, new/cur location, flags, collision-safe
  filenames, same-folder no-ops and invalid destinations. A GUI failure test
  verifies the error dialog and retained source when a destination disappears.
- The software-rendered bulk-drag screenshot was visually inspected for the
  destination highlight, selected rows and message-count status.
- No remote synchronization or native desktop pointer interaction was exercised
  in the headless environment.

## MailSalonGUI 0.3.0 / MailSalon 0.9.0 consolidation

- The full software-driver suite and vet passed for both commands, both
  interfaces and the shared core. GUI, terminal, PIM and MIME race tests passed.
- Terminal/core tests passed with `CGO_ENABLED=0`. Both native Linux builds
  passed and report their respective release versions; a Windows terminal
  executable also cross-compiled with cgo disabled.
- The terminal dependency graph contains no Fyne, OpenGL or GLFW packages.
  A real pseudo-terminal smoke run started with an empty disposable Maildir,
  rendered the interface and quit normally using `q`.
- Original terminal calendar/MIME/config tests run against the shared backend.
  They cover JMAP conversion, UID deduplication, sync locks, recurrence rules,
  exclusions, moved/cancelled events, all-day spans and timezone conversion.
- Additional checks preserve dated VTODO projection, escaped UIDs and cancelled
  exceptions during import. The GUI honors common calendar view/week-start
  settings. Both clients' config paths are checked explicitly.
- The existing GUI tests cover calendar and mail interaction, saved drafts,
  reviewed RSVP drafts, source editing, confirmations and stale worker results.
- Migrated terminal documentation links and Go formatting were checked. CI
  includes both-client tests and an isolated terminal build.
- No real mail was sent, remotely synchronized or migrated. Native GUI display
  interaction remains unavailable in the headless environment.

## 0.2.1

- Full software-driver tests and vet passed; GUI race tests passed. The native
  Linux build reports 0.2.1.
- Letter-key tests deliver both key and rune events and verify one reply,
  reply-all or forward draft for the selected mail, plus compose with no selection.
- Local Maildir tests verify bulk read/unread and archive through letter keys,
  and deletion only after the existing confirmation.
- Focus/selection tests cover normal search/draft typing, Ctrl/Alt/Super guards,
  existing Ctrl+R, hidden tabs, group selection, no selection and modal blocking.
- Navigation, source, search and both help entry points were exercised. The
  software-rendered shortcut help was inspected at the normal window size.
- No mail was sent or remotely synchronized; native KDE key input was not
  available in the headless environment.

## 0.2.0

- Full software-driver tests and vet passed; GUI/PIM race checks passed. The
  native Linux cgo/X11 build reports 0.2.0.
- Calendar UI tests exercised all four views, actual month-cell event taps,
  busy-day overflow, Go to date, Today, new-event date/timezone defaults,
  tooltips, search, month-end navigation and rejection of stale worker results.
- Layout checks verified separate overlap lanes, midnight clipping and
  exclusive all-day ends. Software-rendered month/week/day/agenda screenshots
  were inspected at the application's normal window size.
- Occurrence tests covered iCalendar and JSCalendar rules, exclusions, extra
  dates, moved/cancelled instances, timezone conversion, DST transitions,
  nominal duration and monthly recurrence over missing month-end dates. Source
  data stays intact. Unsupported/dense rules retain an inspection path.
- A separate source test refuses nonexistent local wall times instead of
  silently moving an appointment to a different hour.
- No remote calendar sync or native desktop interaction was exercised. The
  views use existing guarded collection CRUD; dragging/resizing appointments,
  occurrence-only editing and reminders are outside this update.

## 0.1.9

- Full software-driver tests and vet passed. GUI, PIM and MIME race checks
  passed; the native Linux cgo/X11 build reports 0.1.9.
- MIME checks covered unnamed inline calendar alternatives, named `.ICS` files,
  base64/quoted-printable transfer decoding, calendar charsets and reply MIME
  parameters. Ordinary mail body selection remains intact.
- A recurring invitation fixture exercised preview details, quoted parameter
  values, timezone definitions, sequence preservation, correct organizer/attendee
  addressing, Accept/Decline payloads and saved response draft identity.
- Local collection checks covered retained recurrence/alarms/extensions,
  scheduling METHOD removal, multiple UIDs, all-day dates, opaque sync filenames,
  UID path safety, duplicate imports, sync locks, changed-file confirmations and
  preventing instance-only updates from erasing a recurring series.
- Software-driver GUI checks opened the response draft and completed a real
  local import through the chooser. Invitation, response-draft and existing HTML
  preview screenshots were inspected.
- No real invitation replies were sent and no remote calendar server was used.
  JMAP conversion, delegated scheduling and automatic cancellations are outside
  this release. Native desktop interaction was not exercised headlessly.

## 0.1.8

- The full software-driver Go suite and vet passed. GUI/MIME tests with the
  race detector passed, and the native Linux cgo/X11 build reports 0.1.8.
- A multipart newsletter fixture verified formatted headings, emphasis,
  paragraph spacing, tables, lists, code, image-button labels and links absent
  from its plain alternative. Its software-rendered preview was inspected.
- A link-tap check reached the URL-opening app hook with the exact destination;
  launching a real browser was not exercised in this headless environment.
- MIME tests covered malformed markup, source indentation, hidden content,
  empty alternatives, text charsets, unsupported-charset fallback, HTML
  attachments and nested alternative/related/mixed body sections in order.
- URL tests covered entity-decoded destinations, HTML base URLs, protocol-relative
  links, unsupported schemes, prose punctuation and long tracking URLs. Complete
  destinations are retained for clicking/copying even when labels are shortened.
- Native desktop rendering and complex browser/CSS layouts were not tested;
  the app implements a simplified formatted preview rather than a web renderer.

## 0.1.7

- `go test -tags ci ./...`, `go vet -tags ci ./...`, and GUI tests with
  `-race -tags ci` passed. The native Linux cgo/X11 build reports 0.1.7.
- Software-driver mouse/keyboard checks covered Ctrl toggles, Shift ranges,
  Shift navigation, Ctrl+A, Ctrl+Space, Escape, group-preserving right click,
  visible selection counts and selection remapping through search.
- Real temporary Maildirs verified bulk read/unread, archive, confirmed
  move-to-Trash and permanent deletion, changed-selection confirmation guards,
  automatic-read path renames and continued processing after a file failure.
- A paused bulk worker confirmed that changing to the destination folder kept
  the original operation targets and refreshed that folder after completion.
- Explicit mark-unread survived pending preview loading. Ctrl+F/Ctrl+N still
  reached the main-window shortcuts, and Ctrl+A in the search field selected
  text rather than messages. Previous Delete/Backspace/numpad tests passed.
- Software-rendered selection and group-menu screenshots were visually
  inspected. Native modifier-key interaction still needs checking in a real
  desktop session.

## 0.1.6

- `go test -tags ci ./...` and `go vet -tags ci ./...` passed.
- The native Linux cgo/X11 build passed and reports version 0.1.6.
- Generated launchers passed `desktop-file-validate` 0.27. GIO resolved their
  icons to the correct absolute paths and launched the installed binary with
  normal paths, spaces, quotes, dollar signs, backticks and backslashes.
  Icon directories containing a percent sign or tab also worked.
- Install/uninstall checks used stand-in cache commands to verify KDE 6
  preference, KDE 5 fallback, nonfatal refresh failures and preservation of
  unrelated files. DESTDIR installs kept runtime executable/icon paths and
  invoked no host cache tools.
- Main and auxiliary window show sites now apply X11 desktop identity after
  the native handle is created. Actual X11 property changes and KDE taskbar
  rendering still require a graphical desktop session and were not exercised
  in this environment.

## 0.1.5

- Generated and inspected a rose/plum envelope icon with a transparent border,
  packaged as a 512 × 512 PNG and embedded in the executable.
- All package CI-driver tests, `go vet -tags ci ./...` and native Linux build
  passed; the binary reports 0.1.5.
- Installed desktop entries passed `desktop-file-validate` 0.27. GIO launched
  a temporary stand-in executable at plain, spaced and shell-special paths.
- Checked installed icon bytes, executable permissions, matching app/icon ID,
  Fyne metadata, staged `make install`/`make uninstall`, rejection of invalid
  desktop executable paths, and preservation of unrelated files on uninstall.
- Native taskbar rendering cannot be checked without a desktop session. Fyne's
  app icon is set before the first window, so its native window driver inherits
  the resource; the desktop launcher installs an icon for menu/pin lookup.

## 0.1.4

- Compiled the standard `evdev+us` XKB map offline and confirmed that KPDL
  (91) has KP_Delete/KP_Decimal while KPPT/I129 (129) also has KP_Decimal.
  GLFW's X11 reverse table keeps the last key for KP_DECIMAL, explaining why
  the previous lookup can miss the physical . / Del key.
- The X11 build now asks `XKeysymToKeycode` for KP_Delete on GLFW's display.
  No scan-code constants are used in the application.
- Tests check the diagnostic's native KeyDown capture, version and scan-code
  report, copy button, focus recovery and isolation from mail deletion.
- All package race tests, CI-driver vet and native Linux compilation passed.
- A native display could not be started because this environment forbids its
  listening sockets. Physical-key verification remains a desktop check; the
  Help menu diagnostic makes the received event available if it still fails.

## 0.1.3

- Keypad regression tests cover Fyne's unknown, period and comma events for
  the same physical . / Del key, confirming deletion of the selected email.
- Tests also cover ordinary punctuation, unrelated keys, missing scan codes,
  unavailable keypad mappings and search-field focus without mail deletion.
- The keypad scan code is resolved through Fyne's existing GLFW backend;
  no platform-specific scan-code constants are used.
- `go test -race -tags ci ./...`, `go vet -tags ci ./...`, `go mod tidy -diff`
  and the native Linux build passed with Go 1.24.9 and Fyne 2.7.0.
- Native physical keyboard input requires checking on a real desktop. The
  tests inject the native keypad lookup into Fyne's software test driver.

## 0.1.2

- New tests reproduce missing keyboard selection through Fyne's embedded
  `List.Select` and missing Backspace handling on the merged 0.1.1 code.
- Regression tests cover canvas clicks, native list selection, Delete/Backspace,
  search-field isolation, mark-read renames during requests and confirmation,
  and cancellation of pending actions when selection changes.
- All package tests with the race detector, `go vet -tags ci ./...`, and the
  native Linux build passed with Go 1.24.9 and Fyne 2.7.0.
- Physical keyboard input still needs checking on a real desktop; this
  environment provides Fyne's software test driver but no native GUI session.

## 0.1.0

Checked on Linux with Go 1.24.9 and Fyne 2.7.0.

- Native Linux desktop binary compiled with Fyne's cgo graphics driver.
- Software-driver unit/integration tests passed across all local packages.
- Race-detector suite passed.
- `go vet -tags ci ./...` passed.
- CLI version output and example TOML initialization checked.
- Rendered mail, compose, empty collections, contacts and calendar views inspected.
- GUI tests exercised empty selections, attachment forwarding, draft reopening,
  reply identity/Reply-To, stale parse results, failed sends and successful local
  test sends, plus shared-command synchronization deduplication.

The test send command writes to a temporary file; no live email was sent.
The actual GnuPG sign/encrypt/decrypt round trip was not run successfully in this
environment because it forbids the agent's Unix sockets. That test remains
available behind `MAILSALONGUI_TEST_GPG=1`. Windows/macOS builds and live server
synchronization/delivery were not checked. The desktop window was inspected
through Fyne's software renderer; a native graphical desktop session was not
available here.
