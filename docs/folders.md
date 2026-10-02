# Creating folders and managing subscriptions

MailSalonGUI 0.4.0 manages remote mail folders through **MailSalonSync 0.7.0 or
newer**. It supports IMAP and JMAP. Set up the sync account normally first.

## Enable folder management

Add `sync_account` to the GUI's existing `[[accounts]]` block:

```toml
[[accounts]]
name = "personal"
maildir = "~/Maildir"
from = "Your Name <you@example.com>"
receive = "MailSalonSync -plain sync"
send = "MailSalonSync -plain jmap-send -account personal-jmap"
sync_account = "personal-jmap"
# sync_config = "~/.config/MailSalonSync/config.toml"
# sync_executable = "~/bin/MailSalonSync"
```

The GUI account name may differ from `sync_account`, which must match the name
in MailSalonSync. Its `local_root` must equal this GUI account's `maildir`; that
root is verified before server changes.

Omit `sync_config` for MailSalonSync's default config. If `receive` uses a custom
sync config, set `sync_config` to the same file. `sync_executable` is a binary
name or path, not a shell command. Credentials and server settings remain in
MailSalonSync. Save TOML settings and restart the GUI to apply changes.

Other sync tools continue to work with existing receive/send commands. Remote
folder management requires the MailSalonSync settings above; the GUI does not
guess account names from shell commands or email identities.

## Create a folder

Click **+** beside Folders, or **Manage folders… → New folder…**. Right-click
a folder for **New folder…** or **Manage subscriptions…**.
Enter one name and choose a parent, or leave it at Top level. Only parents
that can contain children are offered.

The optional Local folder field selects a relative Maildir path such as
`Projects/Work`. Otherwise MailSalonSync chooses `Folders/<remote-name>`,
encoding characters unsafe in paths. Existing paths and mapping collisions
are rejected; tracked mappings cannot be redirected.

Creation makes the server folder, subscribes, saves its sync mapping and
initializes its Maildir. It appears in the GUI's folder list immediately. Use
**Sync** to download its mail. You can drag selected emails into it; mapped
moves follow MailSalonSync's existing propagation settings.

## Subscriptions

**Manage folders…** lists remote folders, including ones not downloaded locally.
Each row shows server subscription and local sync status separately. Select
a folder and choose **Subscribe / sync locally** to enable both. A folder already
subscribed on the server can still be added to local syncing.

**Unsubscribe…** asks for confirmation, stops syncing this folder and keeps
existing local mail. The server folder and its messages remain. Cached folders
can remain visible in the main pane. Resubscribing reuses the original local
path, and the next sync reconciles it with the server.

The window names its account. Folder requests run in workers; controls disable
while busy, and sync/drag/bulk changes cannot overlap a folder request. Closing
the window cancels the request. Refresh after an interruption, since the server
may already have completed the operation.

## Storage and errors

MailSalonSync saves private `folders-<account-identity-hash>.toml` files in its
`state_dir`. The main sync config's comments and credentials stay unchanged.
Back up these files with the sync state.

Errors distinguish server rejection from a later mapping/local failure. If a
folder was created remotely but a later step failed, refresh and subscribe to
the existing folder to finish. Server failures keep unconfirmed mappings inactive.
Folder commands and sync share a process lock to prevent concurrent changes.

See [MailSalonSync's folder guide](https://github.com/SeraphinaDX/MailSalonSync/blob/main/docs/folders.md)
for CLI commands, the JSON interface and lock recovery. Rename/delete and
calendar/address-book subscription management are separate features.
