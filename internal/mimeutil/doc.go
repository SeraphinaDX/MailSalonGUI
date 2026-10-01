// SPDX-License-Identifier: GPL-3.0-only

// Package mimeutil parses incoming Internet mail and builds outgoing messages.
//
// Incoming messages provide decoded headers, a text body for quoting, threading
// headers, attachments and optional structured HTML preview spans. Multipart
// MIME is walked recursively; text charsets are decoded before HTML parsing.
// Plain alternatives are preserved for reply/forward quoting. The GUI displays
// a formatted HTML alternative when available. Scripts, styles and remote image
// loading are excluded; links retain supported destinations and image labels.
//
// Outgoing Draft values are serialized as RFC 5322/MIME messages. Attachments
// may come from files selected during composition or from in-memory attachments
// preserved while forwarding. Header values are validated before writing so
// user-entered text cannot inject additional mail headers.
package mimeutil
