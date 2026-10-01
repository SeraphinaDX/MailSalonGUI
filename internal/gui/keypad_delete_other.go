//go:build ci || android || ios || mobile || wasm || test_web_driver || tamago || noos || tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

// Other Fyne drivers have no GLFW scan-code lookup. Named Delete/Backspace
// events still work; software-driver tests inject a keypad scan-code resolver.
func keypadDeleteScanCode() int { return -1 }
