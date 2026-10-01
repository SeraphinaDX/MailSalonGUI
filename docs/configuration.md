# TOML configuration

MailSalonGUI accepts MailSalon account/collection settings. It defaults to its
own `mailsalongui/config.toml` and will not automatically change the terminal
client's configuration. Passing the terminal client's path with `-config=PATH`
uses that file directly, including if you save changes through Settings.
Both executables now share this parser; the terminal client's full settings
and keybindings are documented in [Terminal configuration](terminal/configuration.md).

`calendar_default_view` (`month`, `week`, `day`, `agenda`) and
`calendar_week_start` (`monday`, `sunday`) in `[options]` apply to both calendars.
GUI keybindings remain as listed in its usage guide; `[keybindings]` controls
the terminal interface.

## One account

```toml
[[accounts]]
name = "personal"
maildir = "~/Maildir"
from = "Your Name <you@example.com>"
signature_file = "~/.signature"
trash_folder = "Trash"
archive_folder = "Archive"
download_dir = "~/Downloads"
receive = "MailSalonSync -plain sync"
send = "MailSalonSync -plain jmap-send -account personal-jmap"

[options]
default_account = "personal"
startup_sync = false
sync_interval = "5m"
auto_add_reply_contacts = true
```

Remove `signature_file` if you do not have one. Change `personal-jmap` to the
account name in your MailSalonSync configuration. A signature is read at send
time and added before quoted text; changing the From account selects that
account's identity, transport command, signature and PGP settings.

`sync_interval` accepts Go durations such as `30s`, `5m`, or `1h`; use `"0"` to
disable the timer. Nonzero intervals must be at least one second. Startup sync
is independent of the timer; `-no-startup-sync` affects only startup.

Every manual/timer sync runs all distinct receive commands sequentially.
Overlapping sync attempts are skipped while one is running. Commands have a
10-minute timeout each. Send commands have a five-minute timeout. The GUI stays
responsive while they run. Configure a shell wrapper when your workflow needs
special invocation behavior.

`maildir` can be the actual INBOX Maildir or a directory containing separate
INBOX/Sent/etc. Maildir folders. Archive must already exist as a Maildir. Trash
is created locally when needed; create/sync its server counterpart if your sync
tool requires one. Local flags, moves and deletions are propagated according to
that tool's configuration.

Use separate `[[accounts]]` blocks with unique names for multiple accounts.
Legacy `[mail]`, `[identity]` and `[commands]` single-account sections also load.

## Contacts and calendars

```toml
[[collections]]
name = "Personal contacts"
account = "personal"
protocol = "carddav"
local_dir = "~/PIM/contacts/personal"

[[collections]]
name = "Personal calendar"
account = "personal"
protocol = "caldav"
local_dir = "~/PIM/calendars/personal"
```

Protocols: `carddav`, `caldav`, `jmap-contacts`, `jmap-calendars`. The directories
must match MailSalonSync. Collection names are unique. Omit `account` for an
intentionally shared collection. The view and contact picker include only the
selected account's collections and shared collections.

Opening a reply saves a missing sender to an account-specific contact collection
or, if unavailable, a shared one. Disable this with
`auto_add_reply_contacts = false`. No contacts are created if no address book is
configured. Local edits check the original bytes and use the MailSalonSync
collection lock so a stale editor cannot overwrite a newer synced version.

## OpenPGP

Place this subsection immediately after its account, before another account:

```toml
[accounts.gpg]
enabled = true
command = "gpg"
homedir = "~/.gnupg"
sign_key = "you@example.com"
auto_sign = false
auto_encrypt = false
encrypt_to_self = true
```

`command` is an executable, not a shell command line. The compose window has
Sign/Encrypt controls. Incoming messages show verification/decryption status.
Configure keys and a working GnuPG/pinentry environment before enabling it.
A verification error is shown; it does not turn an unverified message into a
trusted one. Unsupported inline PGP is not automatically converted.

## Theme and shortcuts

`[theme]` supports the original MailSalon colors. The GUI maps background,
foreground, muted, border, title, selected_bg and error onto Fyne's palette.
The example uses plum and rose colors. Other terminal-only color settings are
accepted but do not change every Fyne widget.

The original `[keybindings]` section is accepted and validated for compatibility;
the GUI uses its own shortcuts, listed in [Usage](usage.md).

See [config.toml.example](../config.toml.example) for a complete starting file.
