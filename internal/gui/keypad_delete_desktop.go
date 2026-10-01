//go:build !ci && !android && !ios && !mobile && !wasm && !test_web_driver && !tamago && !noos && !tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

import "github.com/go-gl/glfw/v3.3/glfw"

// Use the same GLFW backend as Fyne to resolve the native scan code. Hard-coded
// scan codes differ across operating systems and keyboard configurations.
// Called from keyboard handling after Fyne has initialized GLFW.
func keypadDeleteScanCode() int {
	return glfw.GetKeyScancode(glfw.KeyKPDecimal)
}
