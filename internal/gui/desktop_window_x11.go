//go:build (linux || freebsd || netbsd || openbsd) && !wayland && !ci && !android && !ios && !mobile && !wasm && !test_web_driver && !tamago && !noos && !tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

/*
#cgo pkg-config: x11
#include <stdlib.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>

static void mailsalon_desktop_identity(void *display, unsigned long window, char *id) {
    Display *d = (Display *)display;
    XClassHint hint = {id, id};
    XSetClassHint(d, window, &hint);
    XChangeProperty(d, window, XInternAtom(d, "_KDE_NET_WM_DESKTOP_FILE", False),
                    XInternAtom(d, "UTF8_STRING", False), 8, PropModeReplace,
                    (const unsigned char *)id, (int)strlen(id));
    XFlush(d);
}
*/
import "C"

import (
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"github.com/SeraphinaDX/MailSalonGUI/internal/assets"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func setDesktopIdentity(w fyne.Window) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(context any) {
		x11, ok := context.(driver.X11WindowContext)
		if !ok || x11.WindowHandle == 0 {
			return
		}
		display := glfw.GetX11Display()
		if display == nil {
			return
		}
		// KDE expects the desktop-file basename without its .desktop extension.
		id := C.CString(assets.AppID)
		defer C.free(unsafe.Pointer(id))
		C.mailsalon_desktop_identity(unsafe.Pointer(display), C.ulong(x11.WindowHandle), id)
	})
}
