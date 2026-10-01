// SPDX-License-Identifier: GPL-3.0-only

// Package assets embeds resources needed by the installed application.
package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// AppID also names the Linux desktop entry and installed icon.
const AppID = "ca.cerberusgames.mailsalongui"

//go:embed icon.png
var iconPNG []byte

// Icon is embedded so launching outside the source directory needs no files.
var Icon fyne.Resource = fyne.NewStaticResource("mailsalongui.png", iconPNG)
