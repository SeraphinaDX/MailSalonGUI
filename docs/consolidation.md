# One codebase, two clients

This repository now builds MailSalon (terminal) and MailSalonGUI (desktop).
The module path remains `github.com/SeraphinaDX/MailSalonGUI`, so the existing
desktop packaging and integrations retain their identity. There is one copy
of each shared backend package, with separate `internal/ui` and `internal/gui`
interfaces. MailSalonSync remains a separate sync/send tool.

## Build and move your checkout

```sh
git clone https://github.com/SeraphinaDX/MailSalonGUI.git
cd MailSalonGUI
make build-tui
./MailSalon -version
make build-gui
./MailSalonGUI -version
```

`make build` retains its previous GUI meaning; `make build-all` builds both.
The terminal target forces `CGO_ENABLED=0` and imports no graphics toolkit.
It can be built on a server without X11, OpenGL, Wayland or a C compiler.
The desktop target retains the documented native dependencies and zgo preference.

Use this checkout for future changes to either client after merging the
consolidation. Existing binaries keep working. The earlier terminal repository
is left intact; this change does not archive it, change its remote or overwrite
another checkout. Do not develop independent copies of the core in both repos.
MailSalon and MailSalonGUI keep separate version streams (0.9.0 and 0.3.0 for
this consolidation), declared together in `internal/version`.

## Configuration and local data

| Client | Default config beneath the OS configuration directory |
| --- | --- |
| MailSalon | `mailsalon/config.toml` |
| MailSalonGUI | `mailsalongui/config.toml` |

Linux normally uses `~/.config`; Windows uses the directory returned by Go's
`os.UserConfigDir`. Both accept `-config=PATH`, with the same shared TOML schema,
legacy account conversion, path expansion and validation. No configuration or
mail files need to be moved. You can deliberately point both executables at
one TOML file, or keep separate themes/interface preferences. GUI Settings edits
the file passed to that GUI invocation.

Maildir and collection locations stay in the account/collection settings.
Private saved GUI drafts retain their `mailsalongui/drafts` directory. Terminal
compositions keep their existing lifecycle; sharing a backend does not add GUI
window/draft behavior to the terminal interface. Each client retains its controls,
including terminal `[keybindings]` and the GUI's documented fixed shortcuts.

## Shared behavior and checks

Maildir mutations, MIME/HTML parsing, decoded invitations, contact operations,
calendar recurrence projection, command transport and OpenPGP now have one
implementation. Both interfaces retain their own worker/event handling.
Supported simple JMAP invitation conversion is shared code but remains exposed
through the terminal import chooser; the GUI retains its CalDAV chooser and
reviewed Accept/Decline drafts. Terminal import never replaces an existing UID;
GUI replacement still requires review and unchanged source bytes.

`make test-tui` checks the core and terminal without graphics dependencies.
`make test` / `make check` cover both interfaces with the Fyne CI driver. Tests
from both clients exercise shared calendar, MIME and configuration behavior.

## Source provenance

The terminal interface, tests, configuration fields, JMAP conversion and guides
were imported from `SeraphinaDX/MailSalon` main commit
`d619360` (terminal release 0.8.0), including merged calendar and invitation work.
The desktop baseline is `SeraphinaDX/MailSalonGUI` main commit `1c12b4b`, including
the message-list shortcuts. Imports were adapted to this module; duplicate MIME
and recurrence implementations were replaced by the shared core. Both clients
remain GPL-3.0-only, with third-party notices retained.
