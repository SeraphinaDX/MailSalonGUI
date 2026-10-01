# Changelog

## 0.1.9

- Detect named iCalendar attachments and inline calendar MIME parts without
  filenames; show event details alongside the mail body.
- Import events into local CalDAV collections with UID duplicate confirmation,
  sync-lock/stale-file guards and preserved timezone/recurrence/source fields.
- Prepare Accept/Decline reply drafts for matching invited accounts, retaining
  event identifiers and sequence with correct calendar reply MIME parameters.
- Keep cancellations, unsupported calendar formats and delegated invitations
  visible without applying unsupported scheduling actions.

## 0.1.8

- Replace regex HTML stripping with HTML parsing and a formatted Fyne preview
  that preserves paragraphs, headings, emphasis, lists, table rows and code.
- Display HTML alternatives even when a plain alternative omits links; retain
  the plain alternative for reply/forward quoting.
- Make HTML and plain-text URLs clickable, retain image-button link labels,
  and include complete destinations when copying the displayed body.
- Decode text character sets, handle empty plain alternatives and malformed
  markup, and keep hidden content, scripts and remote images out of the preview.

## 0.1.7

- Add Ctrl-click, Shift-click, Shift navigation and Ctrl+A message selection,
  with highlighted rows and a visible selection count.
- Apply read/unread, archive and confirmed deletion to the selected group;
  preserve group selection when right-clicking a selected message.
- Guard bulk confirmations and queued actions against changed views/selections,
  resolve automatic-read renames before acting, and report partial failures.
- Keep text-field shortcuts and main-window shortcuts working from the list.

## 0.1.6

- Give X11/XWayland windows an explicit desktop-file identity and a consistent
  WM_CLASS so KDE associates main and auxiliary windows with the launcher.
- Use the installed icon's absolute path in the launcher to avoid icon-theme
  lookup failures, and refresh desktop/KDE caches on install and uninstall.
- Keep staging paths out of launcher fields and skip host cache updates when
  installing with DESTDIR.

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
