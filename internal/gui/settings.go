// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

func setDialogDirectory(d *dialog.FileDialog, path string) {
	uri, err := storage.ListerForURI(storage.NewFileURI(path))
	if err == nil {
		d.SetLocation(uri)
	}
}
func (a *App) settings() {
	w := a.Fyne.NewWindow("TOML settings — MailSalonGUI")
	w.Resize(fyne.NewSize(900, 700))
	editor := widget.NewMultiLineEntry()
	editor.Wrapping = fyne.TextWrapOff
	data, err := os.ReadFile(a.configPath)
	if os.IsNotExist(err) {
		data = []byte(config.Example)
	} else if err != nil {
		a.fail(err)
		return
	}
	editor.SetText(string(data))
	savedText := editor.Text
	save := widget.NewButton("Validate and save", func() {
		if err := writeConfig(a.configPath, []byte(editor.Text)); err != nil {
			dialog.ShowError(err, w)
			return
		}
		savedText = editor.Text
		dialog.ShowInformation("Configuration saved", "Restart MailSalonGUI to apply these settings.\n\n"+a.configPath, w)
	})
	save.Importance = widget.HighImportance
	w.SetContent(container.NewBorder(container.NewVBox(widget.NewLabel(a.configPath), widget.NewLabel("Use TOML for accounts, sync, signatures, collections and colors. Changes apply after restart.")), container.NewHBox(save, widget.NewButton("Close", w.Close)), nil, nil, editor))
	w.SetCloseIntercept(func() {
		if editor.Text == savedText {
			w.SetCloseIntercept(nil)
			w.Close()
			return
		}
		dialog.ShowConfirm("Discard changes?", "Close settings without saving?", func(ok bool) {
			if ok {
				w.SetCloseIntercept(nil)
				w.Close()
			}
		}, w)
	})
	showWindow(w)
}

// Validate against the same loader used on startup before replacing the file.
func writeConfig(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		return err
	}
	if _, err = config.Load(f.Name()); err != nil {
		return fmt.Errorf("configuration was not saved: %w", err)
	}
	return os.Rename(f.Name(), path)
}
