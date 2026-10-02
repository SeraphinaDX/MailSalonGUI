// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

func TestFolderTransportUsesLiteralArgumentsAndJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "fake-sync")
	requestFile := filepath.Join(dir, "request")
	argsFile := filepath.Join(dir, "args")
	response := FolderResponse{Version: 1, Account: "sync-account", LocalRoot: dir, Folders: []RemoteFolder{{ID: "id:1", Name: "Work"}}}
	data, _ := json.Marshal(response)
	t.Setenv("FOLDER_TEST_REQUEST", requestFile)
	t.Setenv("FOLDER_TEST_ARGS", argsFile)
	t.Setenv("FOLDER_TEST_RESPONSE", string(data))
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FOLDER_TEST_ARGS\"\ncat > \"$FOLDER_TEST_REQUEST\"\nprintf '%s' \"$FOLDER_TEST_RESPONSE\"\n"
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	account := config.Account{Maildir: dir, SyncAccount: "sync-account", SyncExecutable: exe, SyncConfig: filepath.Join(dir, "config with spaces.toml")}
	name := "x'; $(touch SHOULD_NOT_EXIST)"
	_, err := ManageFolders(context.Background(), account, FolderRequest{Action: "create", Name: name})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(requestFile)
	var req FolderRequest
	if err := json.Unmarshal(b, &req); err != nil || req.Name != name || req.ExpectedLocalRoot != dir {
		t.Fatal(string(b), err)
	}
	b, _ = os.ReadFile(argsFile)
	if !strings.Contains(string(b), account.SyncConfig+"\nfolders\n-account\nsync-account\n-request-stdin\n") {
		t.Fatal(string(b))
	}
	for _, bad := range []FolderResponse{{Version: 2, Account: "sync-account", LocalRoot: dir}, {Version: 1, Account: "other", LocalRoot: dir}, {Version: 1, Account: "sync-account", LocalRoot: "/other"}, {Version: 1, Account: "sync-account", LocalRoot: dir, Folders: []RemoteFolder{{ID: "same", Name: "A"}, {ID: "same", Name: "B"}}}} {
		data, _ = json.Marshal(bad)
		t.Setenv("FOLDER_TEST_RESPONSE", string(data))
		if _, err := ManageFolders(context.Background(), account, FolderRequest{Action: "list"}); err == nil {
			t.Fatal("accepted mismatched response", bad)
		}
	}
}
func TestFolderTransportRequiresExplicitAccount(t *testing.T) {
	if _, err := ManageFolders(context.Background(), config.Account{}, FolderRequest{Action: "list"}); err == nil {
		t.Fatal("guessed account")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ManageFolders(ctx, config.Account{SyncAccount: "mail"}, FolderRequest{Action: "list"}); err == nil {
		t.Fatal("accepted cancelled command")
	}
}
