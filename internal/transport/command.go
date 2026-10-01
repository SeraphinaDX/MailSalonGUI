// SPDX-License-Identifier: GPL-3.0-only

package transport

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Receive runs the account's configured synchronization command and returns
// everything it wrote to stdout/stderr. The command is allowed to manage any
// protocol it wants; MailSalon only expects it to update the local Maildir.
func Receive(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("receive_command is not configured")
	}

	// Configuration stores a shell command line rather than an argv slice, so
	// invoke it through /bin/sh. This allows users to use normal shell quoting,
	// environment expansion, and wrappers in their TOML configuration.
	cmd := shellCommand(ctx, command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("receive command failed: %w", err)
	}
	return string(out), nil
}

// Send writes one complete RFC 5322 message to the configured send command's
// stdin. The external command is responsible for SMTP/JMAP delivery and may use
// message headers (for example with `msmtp -t`) to determine recipients.
func Send(ctx context.Context, command string, raw []byte) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("send_command is not configured")
	}
	cmd := shellCommand(ctx, command)
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("send command failed: %w", err)
	}
	return string(out), nil
}

// shellCommand follows the host shell rather than assuming /bin/sh exists.
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}
	// Bound pipe draining if a shell child retains stdout/stderr after the shell
	// is cancelled; otherwise a configured timeout can still leave Wait blocked.
	cmd.WaitDelay = 2 * time.Second
	return cmd
}
