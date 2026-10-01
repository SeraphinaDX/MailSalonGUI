// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/SeraphinaDX/MailSalonGUI/internal/assets"
	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/demo"
	"github.com/SeraphinaDX/MailSalonGUI/internal/drafts"
	"github.com/SeraphinaDX/MailSalonGUI/internal/gui"
)

func main() {
	path := flag.String("config", config.DefaultPath(), "TOML configuration path")
	showVersion := flag.Bool("version", false, "Print version and exit")
	initConfig := flag.Bool("init-config", false, "Write example TOML at -config path without overwriting an existing file")
	demoMode := flag.Bool("demo", false, "Explore a disposable offline demonstration mailbox")
	noStartup := flag.Bool("no-startup-sync", false, "Disable startup sync for this run")
	flag.Parse()
	if flag.NArg() > 0 {
		fatal(fmt.Errorf("unexpected arguments: %v", flag.Args()))
	}
	if *showVersion {
		fmt.Println("MailSalonGUI " + gui.Version)
		return
	}
	resolved, err := config.ResolvePath(*path)
	if err != nil {
		fatal(err)
	}
	*path = resolved
	if *initConfig {
		if err := os.MkdirAll(filepath.Dir(*path), 0700); err != nil {
			fatal(err)
		}
		f, err := os.OpenFile(*path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			fatal(err)
		}
		_, err = f.WriteString(config.Example)
		ce := f.Close()
		if err == nil {
			err = ce
		}
		if err != nil {
			fatal(err)
		}
		fmt.Println("Created " + *path)
		return
	}
	cfg := config.Default()
	if !*demoMode {
		cfg, err = config.Load(*path)
		if err != nil {
			fatal(err)
		}
	}
	draftDir := drafts.Directory()
	if *demoMode {
		var root string
		cfg, root, err = demo.Create()
		if err != nil {
			fatal(err)
		}
		defer os.RemoveAll(root)
		draftDir = filepath.Join(root, "drafts")
		*path = filepath.Join(root, "config.toml")
	}
	if *noStartup {
		cfg.StartupSync = false
	}
	f := app.NewWithID(assets.AppID)
	a := gui.New(f, cfg, *path, draftDir)
	a.Start()
	if !*demoMode {
		if _, err := os.Stat(*path); os.IsNotExist(err) {
			dialog.ShowInformation("Welcome to MailSalonGUI", "Open Settings to configure your Maildir and identity in TOML.\n\nOr run MailSalonGUI -demo to explore a temporary offline mailbox.\n\nConfiguration: "+*path, a.Window)
		}
	}
	a.Window.ShowAndRun()
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "MailSalonGUI:", err); os.Exit(1) }
