// SPDX-License-Identifier: GPL-3.0-only
package maildir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovePreservesFlagsContentsAndCollisionTarget(t *testing.T) {
	for _, sub := range []string{"new", "cur"} {
		t.Run(sub, func(t *testing.T) {
			source, destination := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
			if err := Ensure(source); err != nil {
				t.Fatal(err)
			}
			if err := Ensure(destination); err != nil {
				t.Fatal(err)
			}
			name := "message"
			if sub == "cur" {
				name += ":2,FRS"
			}
			path := filepath.Join(source, sub, name)
			if err := os.WriteFile(path, []byte(testMessage), 0600); err != nil {
				t.Fatal(err)
			}
			collision := filepath.Join(destination, sub, name)
			if err := os.WriteFile(collision, []byte("existing"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Move(Entry{Path: path}, Folder{Path: destination}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(collision)
			if err != nil || string(data) != "existing" {
				t.Fatal("overwrote destination")
			}
			files, err := os.ReadDir(filepath.Join(destination, sub))
			if err != nil || len(files) != 2 {
				t.Fatal(files, err)
			}
			for _, file := range files {
				if file.Name() == name {
					continue
				}
				data, err := os.ReadFile(filepath.Join(destination, sub, file.Name()))
				if err != nil || string(data) != testMessage {
					t.Fatal("message body changed")
				}
				if sub == "cur" && !strings.HasSuffix(file.Name(), ":2,FRS") {
					t.Fatal("collision changed flags", file.Name())
				}
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("source wasn't removed")
			}
		})
	}
}
func TestMoveInvalidDestinationAndSameFolder(t *testing.T) {
	source := t.TempDir()
	if err := Ensure(source); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "new", "message")
	if err := os.WriteFile(path, []byte(testMessage), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Move(Entry{Path: path}, Folder{Path: source}); err != nil {
		t.Fatal(err)
	}
	if err := Move(Entry{Path: path}, Folder{Path: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("accepted missing target")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("invalid move removed source")
	}
}
