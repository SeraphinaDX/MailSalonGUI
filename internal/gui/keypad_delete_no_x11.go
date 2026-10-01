//go:build (wayland || (!linux && !freebsd && !netbsd && !openbsd)) && !ci && !android && !ios && !mobile && !wasm && !test_web_driver && !tamago && !noos && !tinygo

// SPDX-License-Identifier: GPL-3.0-only

package gui

func x11KeypadDeleteScanCode() int { return -1 }
