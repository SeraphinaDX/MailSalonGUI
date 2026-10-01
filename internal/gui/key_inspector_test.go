// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestKeyboardInspectorCapturesNativePressWithoutMailActions(t *testing.T) {
	a, q := demoApp(t)
	a.selectMessage(0)
	waitForMessage(t, a, q)
	w, inspector := newKeyboardInspectorWindow(a.Fyne, func() int { return 91 })
	t.Cleanup(w.Close)
	keyable, ok := w.Canvas().Focused().(desktop.Keyable)
	if !ok {
		t.Fatal("inspector is not focused for native key-down events")
	}
	keyable.KeyDown(&fyne.KeyEvent{Name: fyne.KeyUnknown, Physical: fyne.HardwareKey{ScanCode: 91}})
	for _, want := range []string{"MailSalonGUI " + Version, "Event: KeyDown", "Fyne key: \"\"", "Scan code: 91", "Keypad Delete scan code: 91", "Recognized as delete: true"} {
		if !strings.Contains(inspector.report, want) {
			t.Fatalf("missing %q in diagnostic: %s", want, inspector.report)
		}
	}
	screenshot(t, w, "keyboard-diagnostic")
	var tapCopy func(fyne.CanvasObject) bool
	tapCopy = func(o fyne.CanvasObject) bool {
		if b, ok := o.(*widget.Button); ok && b.Text == "Copy report" {
			test.Tap(b)
			return true
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if tapCopy(child) {
					return true
				}
			}
		}
		return false
	}
	if !tapCopy(w.Content()) || a.Fyne.Clipboard().Content() != inspector.report {
		t.Fatal("Copy report did not copy the keyboard diagnostic")
	}
	test.Tap(inspector)
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if len(a.messages) != 5 || a.Window.Canvas().Overlays().Top() != nil || w.Canvas().Overlays().Top() != nil {
		t.Fatal("diagnostic key opened mail deletion or changed the mailbox")
	}
	inspector.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPeriod, Physical: fyne.HardwareKey{ScanCode: 60}})
	if !strings.Contains(inspector.report, "Recognized as delete: false") {
		t.Fatal("ordinary period reported as Delete")
	}
}
