# Validation

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
