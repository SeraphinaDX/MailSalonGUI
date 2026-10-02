// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
)

type FolderRequest struct {
	Action            string `json:"action"`
	Mailbox           string `json:"mailbox,omitempty"`
	Name              string `json:"name,omitempty"`
	Parent            string `json:"parent,omitempty"`
	Local             string `json:"local,omitempty"`
	ExpectedLocalRoot string `json:"expected_local_root"`
}
type RemoteFolder struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Subscribed     bool   `json:"subscribed"`
	Syncing        bool   `json:"syncing"`
	Local          string `json:"local,omitempty"`
	Selectable     bool   `json:"selectable"`
	CanCreateChild bool   `json:"can_create_child"`
}
type FolderResponse struct {
	Version   int            `json:"version"`
	Account   string         `json:"account"`
	LocalRoot string         `json:"local_root"`
	Folders   []RemoteFolder `json:"folders"`
}

// Folder values travel as JSON on stdin, never interpolated into a shell.
// The account is explicit because UI identity names need not match sync names.
func ManageFolders(ctx context.Context, a config.Account, request FolderRequest) (FolderResponse, error) {
	if a.SyncAccount == "" {
		return FolderResponse{}, fmt.Errorf("set sync_account in this account's TOML settings to manage remote folders (requires MailSalonSync 0.7.0 or newer)")
	}
	executable := a.SyncExecutable
	if executable == "" {
		executable = "MailSalonSync"
	}
	args := []string{"-plain"}
	if a.SyncConfig != "" {
		args = append(args, "-config", a.SyncConfig)
	}
	args = append(args, "folders", "-account", a.SyncAccount, "-request-stdin")
	request.ExpectedLocalRoot = a.Maildir
	data, err := json.Marshal(request)
	if err != nil {
		return FolderResponse{}, err
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return FolderResponse{}, fmt.Errorf("MailSalonSync folder operation failed: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}
	var result FolderResponse
	if err = json.Unmarshal(out, &result); err != nil {
		return result, fmt.Errorf("invalid folder response from MailSalonSync (requires version 0.7.0 or newer): %w", err)
	}
	if result.Version != 1 || result.Account != a.SyncAccount || !sameFolderRoot(result.LocalRoot, a.Maildir) {
		return result, fmt.Errorf("MailSalonSync returned a different account, Maildir or unsupported folder response")
	}
	seen := map[string]bool{}
	for _, f := range result.Folders {
		if f.ID == "" || f.Name == "" || seen[f.ID] {
			return result, fmt.Errorf("MailSalonSync returned invalid or duplicate folders")
		}
		seen[f.ID] = true
	}
	return result, nil
}
func sameFolderRoot(a, b string) bool {
	normalize := func(s string) string {
		p, err := filepath.Abs(s)
		if err != nil {
			return ""
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		return filepath.Clean(p)
	}
	return normalize(a) == normalize(b)
}
