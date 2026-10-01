//go:build !ci && !android && !ios && !mobile && !wasm && !test_web_driver && !tamago && !noos && !tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

import "github.com/go-gl/glfw/v3.3/glfw"

// Called from keyboard handling after Fyne has initialized GLFW. X11 can map
// multiple physical keys to GLFW's KP_DECIMAL token; its reverse lookup then
// returns only the last one, which may be KPPT instead of the . / Del key.
func keypadDeleteScanCode() int {
	if code := x11KeypadDeleteScanCode(); code > 0 {
		return code
	}
	return glfw.GetKeyScancode(glfw.KeyKPDecimal)
}
