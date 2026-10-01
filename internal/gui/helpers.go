// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/SeraphinaDX/MailSalonGUI/internal/config"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pgp"
)

func pgpSettings(a config.Account) pgp.Settings {
	return pgp.Settings{Enabled: a.GPG.Enabled, Command: a.GPG.Command, HomeDir: a.GPG.HomeDir, SignKey: a.GPG.SignKey, EncryptToSelf: a.GPG.EncryptToSelf}
}
func prefix(subject, p string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), strings.ToLower(p)) {
		return subject
	}
	return p + " " + subject
}
func preferredAccount(accounts []config.Account, fallback int, p *mimeutil.ParsedMessage) int {
	// Parse each field separately: one malformed Cc must not hide a valid To.
	for _, field := range []string{p.To, p.Cc} {
		recipients := map[string]bool{}
		addrs, _ := mail.ParseAddressList(field)
		for _, addr := range addrs {
			recipients[strings.ToLower(addr.Address)] = true
		}
		for i, account := range accounts {
			addr, err := mail.ParseAddress(account.From)
			if err == nil && recipients[strings.ToLower(addr.Address)] {
				return i
			}
		}
	}
	return fallback
}
func replyDraft(p *mimeutil.ParsedMessage, forward, all bool, accounts []config.Account) mimeutil.Draft {
	d := mimeutil.Draft{}
	if forward {
		d.Subject = prefix(p.Subject, "Fwd:")
		d.Body = fmt.Sprintf("\n\n---------- Forwarded message ----------\nFrom: %s\nDate: %s\nSubject: %s\nTo: %s\n\n%s", p.From, p.Date, p.Subject, p.To, p.Body)
		d.MemoryAttachments = append([]mimeutil.Attachment(nil), p.Attachments...)
		return d
	}
	target := p.ReplyTo
	if strings.TrimSpace(target) == "" {
		target = p.From
	}
	d.To = target
	d.Subject = prefix(p.Subject, "Re:")
	d.InReplyTo = p.MessageID
	d.References = strings.TrimSpace(p.References + " " + p.MessageID)
	if all {
		self := map[string]bool{}
		for _, a := range accounts {
			addr, err := mail.ParseAddress(a.From)
			if err == nil {
				self[strings.ToLower(addr.Address)] = true
			}
		}
		seen := map[string]bool{}
		var to, cc []string
		appendAddresses := func(raw string, out *[]string) {
			addrs, _ := mail.ParseAddressList(raw)
			for _, a := range addrs {
				key := strings.ToLower(a.Address)
				if !seen[key] && !self[key] {
					seen[key] = true
					*out = append(*out, a.String())
				}
			}
		}
		appendAddresses(target, &to)
		appendAddresses(p.To, &cc)
		appendAddresses(p.Cc, &cc)
		d.To = strings.Join(to, ", ")
		d.Cc = strings.Join(cc, ", ")
	}
	var quoted []string
	for _, line := range strings.Split(p.Body, "\n") {
		quoted = append(quoted, "> "+line)
	}
	d.Body = fmt.Sprintf("\n\nOn %s, %s wrote:\n%s", p.Date, p.From, strings.Join(quoted, "\n"))
	return d
}
func applySignature(body, signature string) string {
	signature = strings.TrimRight(signature, "\r\n")
	if strings.TrimSpace(signature) == "" {
		return body
	}
	if !strings.HasPrefix(strings.TrimLeft(signature, "\t\r\n"), "-- ") {
		signature = "-- \n" + signature
	}
	cut := -1
	for _, marker := range []string{"\n\nOn ", "\n\n---------- Forwarded message ----------"} {
		if i := strings.Index(body, marker); i >= 0 && (cut < 0 || i < cut) {
			cut = i
		}
	}
	if cut >= 0 {
		return strings.TrimRight(body[:cut], " \t\r\n") + "\n\n" + signature + body[cut:]
	}
	return strings.TrimRight(body, " \t\r\n") + "\n\n" + signature
}
