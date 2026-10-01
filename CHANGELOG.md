# Changelog

## 0.1.5

- Add a rose/plum envelope icon embedded in the app and inherited by its windows.
- Add user-local Linux desktop installation with a launcher and hicolor icon.
- Add Fyne packaging metadata for the application ID, icon and release version.

## 0.1.4

- Resolve the numpad Delete key directly through X11's KP_Delete mapping,
  avoiding GLFW reverse-lookup collisions with a second keypad decimal key.
- Add Help → Keyboard diagnostic to inspect and copy native key events in a
  separate window without deleting mail.

## 0.1.3

- Recognize the numpad . / Del key in the focused message list using its native
  scan code, including when Fyne reports an unknown key, period or comma.
- Ordinary punctuation keys and editable fields retain their existing behavior.

## 0.1.2

- Track keyboard selection when Fyne's built-in list items select a message.
- Accept Delete and Backspace in the focused message list.
- Preserve deletion requests and confirmations during automatic mark-read renames.

## 0.1.1

- Delete in the focused message list opens the existing deletion confirmation.
- Right-click messages for reply, forward, read/unread, archive, delete and source.
- Right-click selects the targeted email; stale menu actions are ignored after reloads.
- Arrow keys, Home and End select messages, following the clicked row.

## 0.1.0

- Independent MailSalonGUI codebase in Go and Fyne.
- Resizable mail panes, account selector, folder search and plain text previews.
- Compose, reply, reply all, forward with original attachments, and local drafts.
- Attachment saving, read/unread flags, archive, Trash and permanent deletion.
- TOML settings editor, periodic background sync and per-account send commands.
- Contacts and calendar browsing, creation, source editing and deletion.
- Optional GnuPG sign/encrypt, decrypt/verify and message security status.
- Disposable offline demo and documented zgo-aware build commands.
