// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

var bodyURL = regexp.MustCompile(`(?i)(?:https?://|mailto:)[^\s<>]+`)

// Rich-text segments contain only parsed text and explicit links. Raw email
// markup is never passed to Markdown, a webview or an image downloader.
func messageBodySegments(p *mimeutil.ParsedMessage) []widget.RichTextSegment {
	spans := p.HTMLPreview
	if len(spans) == 0 {
		spans = plainBodySpans(p.Body)
	}
	var linkedSpans []mimeutil.BodySpan
	for _, span := range spans {
		if span.Link != "" {
			linkedSpans = append(linkedSpans, span)
			continue
		}
		// Plain sections of multipart mail and bare URLs inside HTML need links too.
		for _, part := range plainBodySpans(span.Text) {
			styled := span
			styled.Text, styled.Link = part.Text, part.Link
			linkedSpans = append(linkedSpans, styled)
		}
	}
	var segments []widget.RichTextSegment
	for _, span := range linkedSpans {
		if span.Text == "" {
			continue
		}
		if destination := mimeutil.SafeLink(span.Link); destination != nil {
			label := span.Text
			// Fyne hyperlinks do not wrap like TextSegments. Keep bare tracking
			// URLs from stretching the pane; their full destination stays in Copy.
			if strings.TrimSpace(label) == span.Link && utf8.RuneCountInString(label) > 70 {
				label = string([]rune(label)[:60]) + "…"
			}
			segments = append(segments, &widget.HyperlinkSegment{Text: label, URL: destination})
			continue
		}
		style := widget.RichTextStyleInline
		style.TextStyle = fyne.TextStyle{Bold: span.Bold, Italic: span.Italic, Monospace: span.Monospace}
		if span.Heading > 0 {
			if span.Heading <= 2 {
				style.SizeName = widget.RichTextStyleHeading.SizeName
			} else {
				style.SizeName = widget.RichTextStyleSubHeading.SizeName
			}
		}
		segments = append(segments, &widget.TextSegment{Text: span.Text, Style: style})
	}
	return segments
}

func plainBodySpans(body string) []mimeutil.BodySpan {
	var spans []mimeutil.BodySpan
	offset := 0
	for _, match := range bodyURL.FindAllStringIndex(body, -1) {
		start, end := match[0], match[1]
		candidate := strings.TrimRight(body[start:end], ".,;!?:\"'")
		// Trim unmatched prose parentheses while preserving balanced URL paths.
		for strings.HasSuffix(candidate, ")") && strings.Count(candidate, ")") > strings.Count(candidate, "(") {
			candidate = strings.TrimSuffix(candidate, ")")
		}
		end = start + len(candidate)
		if mimeutil.SafeLink(candidate) == nil {
			continue
		}
		if start > offset {
			spans = append(spans, mimeutil.BodySpan{Text: body[offset:start]})
		}
		spans = append(spans, mimeutil.BodySpan{Text: candidate, Link: candidate})
		offset = end
	}
	if offset < len(body) {
		spans = append(spans, mimeutil.BodySpan{Text: body[offset:]})
	}
	return spans
}
