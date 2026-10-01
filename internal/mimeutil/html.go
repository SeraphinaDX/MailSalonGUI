// SPDX-License-Identifier: GPL-3.0-only

package mimeutil

import (
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// BodySpan is display content, never executable HTML. Links contain only
// supported destinations; images are represented by text without loading src.
type BodySpan struct {
	Text                    string
	Link                    string
	Bold, Italic, Monospace bool
	Heading                 int
}

// DisplayText includes HTML link destinations for copying. Reply quoting still
// uses the sender's plain-text alternative when present.
func (p *ParsedMessage) DisplayText() string {
	if len(p.HTMLPreview) == 0 {
		return p.Body
	}
	var out strings.Builder
	for i := 0; i < len(p.HTMLPreview); i++ {
		span := p.HTMLPreview[i]
		label := span.Text
		if span.Link != "" {
			for i+1 < len(p.HTMLPreview) && p.HTMLPreview[i+1].Link == span.Link {
				i++
				label += p.HTMLPreview[i].Text
			}
			out.WriteString(label)
			if strings.TrimSpace(label) != span.Link {
				out.WriteString(" (" + span.Link + ")")
			}
		} else {
			out.WriteString(label)
		}
	}
	return strings.Trim(out.String(), "\n")
}

type htmlText struct {
	spans []BodySpan
	space bool
	base  *url.URL
}

func (b *htmlText) trailingBreaks() int {
	count := 0
	for i := len(b.spans) - 1; i >= 0; i-- {
		text := b.spans[i].Text
		for j := len(text) - 1; j >= 0; j-- {
			if text[j] != '\n' {
				return count
			}
			count++
		}
	}
	return count
}
func (b *htmlText) append(text string, style BodySpan) {
	if text == "" {
		return
	}
	style.Text = ""
	if len(b.spans) > 0 {
		last := &b.spans[len(b.spans)-1]
		old := *last
		old.Text = ""
		if old == style {
			last.Text += text
			return
		}
	}
	style.Text = text
	b.spans = append(b.spans, style)
}
func (b *htmlText) text(text string, style BodySpan, pre bool) {
	if pre {
		b.append(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), style)
		return
	}
	var normalized strings.Builder
	for _, r := range text {
		// Source-code indentation is not a rendered line break in HTML.
		if strings.ContainsRune(" \t\r\n\f\u00a0", r) {
			b.space = true
			continue
		}
		if b.space && (normalized.Len() > 0 || (len(b.spans) > 0 && b.trailingBreaks() == 0)) {
			normalized.WriteByte(' ')
		}
		b.space = false
		normalized.WriteRune(r)
	}
	b.append(normalized.String(), style)
}
func (b *htmlText) line(count int) {
	b.space = false
	if len(b.spans) == 0 {
		return
	}
	if missing := count - b.trailingBreaks(); missing > 0 {
		b.append(strings.Repeat("\n", missing), BodySpan{})
	}
}

func attribute(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
func hiddenHTML(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "hidden" {
			return true
		}
	}
	for _, declaration := range strings.Split(strings.ToLower(attribute(n, "style")), ";") {
		key, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(strings.Split(value, "!")[0])
		if (key == "display" && value == "none") || (key == "visibility" && value == "hidden") || (key == "mso-hide" && value == "all") {
			return true
		}
	}
	return false
}

// SafeLink is shared with the GUI so plain mail cannot create executable/file
// links either. A protocol-relative web link uses HTTPS; relative HTML links
// are resolved only against an explicit, valid web base URL.
func SafeLink(raw string) *url.URL {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host != "" {
			return u
		}
	case "mailto":
		if u.Opaque != "" {
			return u
		}
	}
	return nil
}
func (b *htmlText) link(raw string) string {
	if u := SafeLink(raw); u != nil {
		return u.String()
	}
	if b.base != nil {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err == nil && u.Scheme == "" && raw != "" && !strings.HasPrefix(raw, "#") {
			if resolved := SafeLink(b.base.ResolveReference(u).String()); resolved != nil {
				return resolved.String()
			}
		}
	}
	return ""
}

func parseHTML(source string) []BodySpan {
	root, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil
	}
	b := &htmlText{}
	// Base is metadata only. The rest of head, including CSS, is not displayed.
	var findBase func(*html.Node)
	findBase = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "base" && b.base == nil {
			if u := SafeLink(attribute(n, "href")); u != nil && (u.Scheme == "http" || u.Scheme == "https") {
				b.base = u
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findBase(c)
		}
	}
	findBase(root)
	b.walk(root, BodySpan{}, false)
	// Strip leading/trailing layout breaks while preserving preformatted text.
	for len(b.spans) > 0 && strings.TrimSpace(b.spans[len(b.spans)-1].Text) == "" {
		b.spans = b.spans[:len(b.spans)-1]
	}
	if len(b.spans) > 0 {
		b.spans[len(b.spans)-1].Text = strings.TrimRight(b.spans[len(b.spans)-1].Text, "\n")
	}
	return b.spans
}

func (b *htmlText) walk(n *html.Node, style BodySpan, pre bool) {
	if n.Type == html.TextNode {
		b.text(n.Data, style, pre)
		return
	}
	if n.Type != html.ElementNode && n.Type != html.DocumentNode {
		return
	}
	if n.Type == html.ElementNode && hiddenHTML(n) {
		return
	}
	tag := n.Data
	block := 0
	switch tag {
	case "head", "script", "style", "template", "noscript", "iframe", "object", "svg":
		return
	case "br":
		b.append("\n", BodySpan{})
		b.space = false
		return
	case "hr":
		b.line(1)
		b.append("────────", BodySpan{})
		b.line(1)
		return
	case "p", "section", "article", "blockquote", "pre", "h1", "h2", "h3", "h4", "h5", "h6":
		block = 2
	case "div", "table", "tr", "ul", "ol", "li", "header", "footer", "dl", "dt", "dd":
		block = 1
	}
	if block > 0 {
		b.line(block)
	}
	switch tag {
	case "strong", "b":
		style.Bold = true
	case "em", "i", "blockquote":
		style.Italic = true
	case "pre":
		pre = true
		style.Monospace = true
	case "code", "tt":
		style.Monospace = true
	case "h1", "h2", "h3", "h4", "h5", "h6":
		style.Bold = true
		style.Heading, _ = strconv.Atoi(tag[1:])
	case "li":
		prefix := "* "
		if n.Parent != nil && n.Parent.Data == "ol" {
			number, _ := strconv.Atoi(attribute(n.Parent, "start"))
			if number == 0 {
				number = 1
			}
			for sibling := n.Parent.FirstChild; sibling != n; sibling = sibling.NextSibling {
				if sibling.Type == html.ElementNode && sibling.Data == "li" {
					if value, err := strconv.Atoi(attribute(sibling, "value")); err == nil {
						number = value
					}
					number++
				}
			}
			if value, err := strconv.Atoi(attribute(n, "value")); err == nil {
				number = value
			}
			prefix = strconv.Itoa(number) + ". "
		}
		b.append(prefix, BodySpan{})
	case "td", "th":
		if tag == "th" {
			style.Bold = true
		}
		for sibling := n.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
			if sibling.Type == html.ElementNode && (sibling.Data == "td" || sibling.Data == "th") {
				if b.trailingBreaks() == 0 {
					b.append(" | ", BodySpan{})
					b.space = false
				}
				break
			}
		}
	case "img":
		if attribute(n, "width") == "1" && attribute(n, "height") == "1" {
			return
		}
		b.text(attribute(n, "alt"), style, false)
		return
	case "a":
		style.Link = b.link(attribute(n, "href"))
	}
	before := len(b.spans)
	beforeText := ""
	if before > 0 {
		beforeText = b.spans[before-1].Text
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.walk(c, style, pre)
	}
	if tag == "a" && style.Link != "" && len(b.spans) == before && (before == 0 || b.spans[before-1].Text == beforeText) {
		label := attribute(n, "aria-label")
		if label == "" {
			label = attribute(n, "title")
		}
		if label == "" {
			label = style.Link
		}
		b.text(label, style, false)
	}
	if block > 0 {
		b.line(block)
	}
}
