// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

func typeMailLetter(a *App, key fyne.KeyName, r rune) {
	// The desktop emits both events for printable keys. Actions must run once.
	a.Window.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: key})
	a.Window.Canvas().Focused().TypedRune(r)
}

func TestMailLetterReplyForwardAndCompose(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(2)
	typeMailLetter(a, fyne.KeyR, 'r')
	if len(a.composers) != 0 {
		t.Fatal("reply opened before preview loaded")
	}
	waitForMessage(t, a, q)
	a.parsed.Cc = "Pat <pat@example.com>, Demo User <demo@example.com>"
	for _, tc := range []struct {
		key          fyne.KeyName
		r            rune
		forward, all bool
	}{{fyne.KeyR, 'r', false, false}, {fyne.KeyR, 'R', false, true}, {fyne.KeyF, 'f', true, false}} {
		a.messageList.modifiers = func() fyne.KeyModifier {
			if tc.r == 'R' {
				return fyne.KeyModifierShift
			}
			return 0
		}
		want := replyDraft(a.parsed, tc.forward, tc.all, a.cfg.Accounts)
		typeMailLetter(a, tc.key, tc.r)
		if len(a.composers) != 1 {
			t.Fatalf("%c opened %d composers", tc.r, len(a.composers))
		}
		for c := range a.composers {
			if c.to.Text != want.To || c.cc.Text != want.Cc || c.subject.Text != want.Subject || c.body.Text != want.Body {
				t.Fatalf("%c did not use the selected message's draft", tc.r)
			}
			c.close()
		}
	}
	a.messageList.UnselectAll()
	typeMailLetter(a, fyne.KeyN, 'n')
	if len(a.composers) != 1 {
		t.Fatal("n requires a selected message")
	}
	for c := range a.composers {
		if c.to.Text != "" || c.subject.Text != "" {
			t.Fatal("new draft quoted mail")
		}
		c.close()
	}
}

func TestMailLetterBulkReadArchiveAndDelete(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.messageList.SelectWithModifiers(1, fyne.KeyModifierShift)
	waitForMessage(t, a, q)
	typeMailLetter(a, fyne.KeyU, 'u')
	pump(t, q, func() bool { return !a.changing })
	if a.messages[0].Unread || a.messages[1].Unread {
		t.Fatal("u did not mark both read")
	}
	typeMailLetter(a, fyne.KeyU, 'u')
	pump(t, q, func() bool { return !a.changing })
	if !a.messages[0].Unread || !a.messages[1].Unread {
		t.Fatal("u did not mark both unread")
	}
	want := map[string]bool{a.messages[0].Subject: true, a.messages[1].Subject: true}
	typeMailLetter(a, fyne.KeyE, 'e')
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 3 })
	archived, err := maildir.Scan(maildir.Folder{Name: "Archive", Path: filepath.Join(a.cfg.Accounts[0].Maildir, "Archive")})
	if err != nil || len(archived) != 2 {
		t.Fatalf("archive: %v %v", archived, err)
	}
	for _, e := range archived {
		if !want[e.Subject] {
			t.Fatal("archived a message outside selection")
		}
	}
	a.selectMessage(0)
	waitForMessage(t, a, q)
	path := a.messages[0].Path
	typeMailLetter(a, fyne.KeyD, 'd')
	if a.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("d did not ask to delete")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("d deleted without confirmation")
	}
	// Even a direct event delivered to the list must not act behind a dialog.
	a.messageList.TypedRune('e')
	if a.changing {
		t.Fatal("archived behind delete confirmation")
	}
	tapConfirmation(t, a.Window)
	pump(t, q, func() bool { return !a.changing && len(a.messages) == 2 })
}

func TestMailLetterFocusModifiersAndSelectionGuards(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	for _, mod := range []fyne.KeyModifier{fyne.KeyModifierControl, fyne.KeyModifierAlt, fyne.KeyModifierSuper} {
		a.messageList.modifiers = func() fyne.KeyModifier { return mod }
		a.messageList.TypedRune('e')
		a.messageList.TypedRune('r')
		if a.changing || len(a.composers) != 0 {
			t.Fatal("modified letter triggered mail action")
		}
	}
	a.messageList.modifiers = func() fyne.KeyModifier { return 0 }
	a.messageList.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl})
	if len(a.composers) != 1 {
		t.Fatal("Ctrl+R stopped working")
	}
	for c := range a.composers {
		c.window.Canvas().Focus(c.body)
		c.body.SetText("")
		for _, r := range "refund" {
			c.window.Canvas().Focused().TypedRune(r)
		}
		if c.body.Text != "refund" || a.changing {
			t.Fatal("draft typing invoked mail actions")
		}
		c.close()
	}
	a.Window.Canvas().Focus(a.search)
	a.messageList.TypedRune('e')
	for _, r := range "ref" {
		a.Window.Canvas().Focused().TypedRune(r)
	}
	if a.search.Text != "ref" || a.changing || len(a.composers) != 0 {
		t.Fatal("search letters invoked mail actions")
	}
	a.search.SetText("")
	a.selectMessage(0)
	waitForMessage(t, a, q)
	a.tabs.SelectIndex(1)
	a.Window.Canvas().Focus(a.messageList)
	a.messageList.TypedRune('e')
	if a.changing {
		t.Fatal("hidden list archived mail")
	}
	a.tabs.SelectIndex(0)
	a.messageList.SelectWithModifiers(1, fyne.KeyModifierShift)
	waitForMessage(t, a, q)
	before := len(a.Fyne.Driver().AllWindows())
	for _, r := range "rRfs" {
		a.messageList.TypedRune(r)
	}
	if len(a.composers) != 0 || len(a.Fyne.Driver().AllWindows()) != before {
		t.Fatal("single-message action accepted group selection")
	}
	a.messageList.UnselectAll()
	for _, r := range "erud" {
		a.messageList.TypedRune(r)
	}
	if a.changing || a.Window.Canvas().Overlays().Top() != nil {
		t.Fatal("mail action accepted no selection")
	}
}

func TestMailLetterNavigationSearchSourceAndHelp(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	typeMailLetter(a, fyne.KeyJ, 'j')
	waitForMessage(t, a, q)
	if a.selected != 1 {
		t.Fatal("j did not select next mail")
	}
	typeMailLetter(a, fyne.KeyK, 'k')
	waitForMessage(t, a, q)
	if a.selected != 0 {
		t.Fatal("k did not select previous mail")
	}
	before := len(a.Fyne.Driver().AllWindows())
	typeMailLetter(a, fyne.KeyS, 's')
	pump(t, q, func() bool { return len(a.Fyne.Driver().AllWindows()) > before })
	windows := a.Fyne.Driver().AllWindows()
	w := windows[len(windows)-1]
	if w.Title() != "Message source" {
		t.Fatal("s did not open source")
	}
	w.Close()
	typeMailLetter(a, fyne.KeySlash, '/')
	if a.Window.Canvas().Focused() != a.search {
		t.Fatal("/ did not focus search")
	}
	a.Window.Canvas().Focus(a.messageList)
	a.messageList.TypedRune('?')
	if a.Window.Canvas().Overlays().Top() == nil {
		t.Fatal("? did not show help")
	}
	// The menu exposes help even if the list has no selection or focus.
	a.Window.Canvas().Overlays().Top().Hide()
	for _, menu := range a.Window.MainMenu().Items {
		for _, item := range menu.Items {
			if strings.EqualFold(item.Label, "Keyboard shortcuts") {
				item.Action()
				if a.Window.Canvas().Overlays().Top() == nil {
					t.Fatal("menu did not show help")
				}
				screenshot(t, a.Window, "mail-shortcuts")
				return
			}
		}
	}
	t.Fatal("keyboard shortcuts menu missing")
}
