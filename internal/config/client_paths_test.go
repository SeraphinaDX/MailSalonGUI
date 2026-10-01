// SPDX-License-Identifier: GPL-3.0-only
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientConfigurationPaths(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if TerminalPath() != filepath.Join(base, "mailsalon", "config.toml") {
		t.Fatal("terminal config path changed")
	}
	if GUIPath() != filepath.Join(base, "mailsalongui", "config.toml") || DefaultPath() != GUIPath() {
		t.Fatal("GUI config path changed")
	}
}
