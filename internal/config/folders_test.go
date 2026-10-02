// SPDX-License-Identifier: GPL-3.0-only
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFolderSyncAccountSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	text := strings.Replace(Example, `# sync_account = "personal-jmap"`, `sync_account = "different-sync-name"`, 1)
	text = strings.Replace(text, `# sync_config = "~/.config/MailSalonSync/config.toml"`, `sync_config = "~/sync/config.toml"`, 1)
	text = strings.Replace(text, `# sync_executable = "MailSalonSync"`, `sync_executable = "~/bin/MailSalonSync"`, 1)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	a := cfg.Accounts[0]
	home, _ := os.UserHomeDir()
	if a.SyncAccount != "different-sync-name" || a.SyncConfig != filepath.Join(home, "sync/config.toml") || a.SyncExecutable != filepath.Join(home, "bin/MailSalonSync") {
		t.Fatal(a)
	}
}
