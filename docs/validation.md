# Validation

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
