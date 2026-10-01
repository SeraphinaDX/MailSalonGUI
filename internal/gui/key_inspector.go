// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// The inspector owns focus in a separate window. Captured keys never invoke
// mail actions, and the report contains only keyboard and version information.
type keyInspector struct {
	widget.BaseWidget
	label   *widget.Label
	resolve func() int
	report  string
}

var _ desktop.Keyable = (*keyInspector)(nil)

func newKeyboardInspectorWindow(f fyne.App, resolve func() int) (fyne.Window, *keyInspector) {
	w := f.NewWindow("Keyboard diagnostic")
	k := &keyInspector{label: widget.NewLabel("Click here, then press your numpad . / Del key."), resolve: resolve}
	k.label.Wrapping = fyne.TextWrapWord
	k.ExtendBaseWidget(k)
	copy := widget.NewButton("Copy report", func() { f.Clipboard().SetContent(k.report) })
	w.SetContent(container.NewBorder(widget.NewLabel("Press a key below. This window does not delete messages."),
		container.NewHBox(copy, widget.NewButton("Close", w.Close)), nil, nil, k))
	w.Resize(fyne.NewSize(570, 300))
	w.Show()
	w.Canvas().Focus(k)
	return w, k
}

func (k *keyInspector) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(k.label))
}
func (k *keyInspector) FocusGained()   {}
func (k *keyInspector) FocusLost()     {}
func (k *keyInspector) TypedRune(rune) {}
func (k *keyInspector) Tapped(*fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(k); c != nil {
		c.Focus(k)
	}
}
func (k *keyInspector) KeyDown(event *fyne.KeyEvent) { k.capture("KeyDown", event) }
func (k *keyInspector) KeyUp(*fyne.KeyEvent)         {}
func (k *keyInspector) TypedKey(event *fyne.KeyEvent) {
	k.capture("TypedKey", event)
}
func (k *keyInspector) capture(source string, event *fyne.KeyEvent) {
	scanCode := k.resolve()
	matcher := &messageList{keypadDeleteScanCode: func() int { return scanCode }}
	k.report = fmt.Sprintf("MailSalonGUI %s keyboard diagnostic (%s)\nEvent: %s\nFyne key: %q\nScan code: %d\nKeypad Delete scan code: %d\nRecognized as delete: %t",
		Version, runtime.GOOS, source, event.Name, event.Physical.ScanCode, scanCode, matcher.isDeleteKey(event))
	k.label.SetText(k.report)
}
