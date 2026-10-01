// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"golang.org/x/image/colornames"
)

type salonTheme struct {
	fyne.Theme
	colors config.Theme
}

func colorValue(s string, fallback color.Color) color.Color {
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		if n, err := strconv.ParseUint(s[1:], 16, 32); err == nil {
			return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}
		}
	}
	names := map[string]color.Color{"white": color.White, "black": color.Black, "grey": color.Gray{Y: 130}, "gray": color.Gray{Y: 130}, "pink": color.NRGBA{R: 255, G: 158, B: 203, A: 255}, "cyan": color.NRGBA{G: 216, B: 203, A: 255}, "red": color.NRGBA{R: 255, G: 107, B: 129, A: 255}, "green": color.NRGBA{R: 100, G: 216, B: 203, A: 255}, "yellow": color.NRGBA{R: 246, G: 193, B: 119, A: 255}}
	if c, ok := names[strings.ToLower(s)]; ok {
		return c
	}
	if c, ok := colornames.Map[strings.ToLower(s)]; ok {
		return c
	}
	return fallback
}
func (t salonTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	base := t.Theme.Color(n, v)
	switch n {
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return colorValue(t.colors.Title, base)
	case theme.ColorNameSelection:
		return colorValue(t.colors.SelectedBG, base)
	case theme.ColorNameBackground:
		return colorValue(t.colors.Background, base)
	case theme.ColorNameForeground:
		return colorValue(t.colors.Foreground, base)
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return colorValue(t.colors.Muted, base)
	case theme.ColorNameError:
		return colorValue(t.colors.Error, base)
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return colorValue(t.colors.Border, base)
	}
	return base
}
