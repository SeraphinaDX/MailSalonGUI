// SPDX-License-Identifier: GPL-3.0-only

package gui

import "fyne.io/fyne/v2"

// Fyne creates the native window during Show, so desktop identity must be set
// afterwards. Use this for every top-level window, including auxiliary ones.
func showWindow(w fyne.Window) {
	w.Show()
	setDesktopIdentity(w)
}

// Run shows the main window with its desktop identity and starts Fyne's loop.
func (a *App) Run() {
	showWindow(a.Window)
	a.Fyne.Run()
}
