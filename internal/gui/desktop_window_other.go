//go:build (!linux && !freebsd && !netbsd && !openbsd) || wayland || ci || android || ios || mobile || wasm || test_web_driver || tamago || noos || tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

import "fyne.io/fyne/v2"

func setDesktopIdentity(fyne.Window) {}
