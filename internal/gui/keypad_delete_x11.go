//go:build (linux || freebsd || netbsd || openbsd) && !wayland && !ci && !android && !ios && !mobile && !wasm && !test_web_driver && !tamago && !noos && !tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/keysym.h>

static int mailsalon_keypad_delete(void *display) {
    return XKeysymToKeycode((Display *)display, XK_KP_Delete);
}
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Resolve KP_Delete itself instead of KP_Decimal. The standard X11 keymap has
// both KPDL (. / Del) and KPPT (decimal) mapped to GLFW_KEY_KP_DECIMAL.
func x11KeypadDeleteScanCode() int {
	display := glfw.GetX11Display()
	if display == nil {
		return -1
	}
	return int(C.mailsalon_keypad_delete(unsafe.Pointer(display)))
}
