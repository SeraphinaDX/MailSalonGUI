// SPDX-License-Identifier: GPL-3.0-only

package mimeutil

import (
	"strings"
	"testing"
)

func TestHTMLAlternativeKeepsFormattingAndImageButtonLinks(t *testing.T) {
	p, err := ParseFile("testdata/newsletter.eml")
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != "Please view the formatted invitation." {
		t.Fatal("plain alternative was lost for quoting")
	}
	display := p.DisplayText()
	for _, want := range []string{
		"You're invited!\n\nHello friend, join us for a relaxed afternoon at the library.",
		"Tea and coffee are provided.", "Day | Time\nThursday | 14:00",
		"Reserve your place (https://example.com/rsvp?event=go&source=mail)",
		"Event details (https://example.com/details)", "* Show a project\n* Ask questions",
		"Email the organiser (mailto:community@example.com)", "go build ./...\n  go test ./...",
	} {
		if !strings.Contains(display, want) {
			t.Fatalf("missing %q in:\n%s", want, display)
		}
	}
	for _, unwanted := range []string{"Invisible preheader", "Not body text", "alert(", "button.png", "p {"} {
		if strings.Contains(display, unwanted) {
			t.Fatalf("non-content leaked: %q", unwanted)
		}
	}
	if !strings.Contains(display, "friend") || len(p.HTMLPreview) == 0 {
		t.Fatal("missing HTML preview")
	}
}
func TestHTMLDOMWhitespaceMalformedMarkupAndFallbacks(t *testing.T) {
	cases := []struct{ html, want string }{
		{"<p>Hello\n    <b>world</b>!</p><p>Next paragraph", "Hello world!\n\nNext paragraph"},
		{"<div>One<div>Two</div>Three</div>", "One\nTwo\nThree"},
		{"<ol start=3><li>Third<li>Fourth</ol>", "3. Third\n4. Fourth"},
		{"<ol><li value=7>Seven<li>Eight</ol>", "7. Seven\n8. Eight"},
		{"<p>Keep &lt;angle brackets&gt; &amp; café.</p>", "Keep <angle brackets> & café."},
		{"<a href='https://example.com/x?a=1&amp;b=2'><img src='remote'></a>", "https://example.com/x?a=1&b=2"},
		{"<base href='https://example.com/events/'><a href='next'>Next</a>", "Next (https://example.com/events/next)"},
		{"<a href='//example.com/next'>Next</a>", "Next (https://example.com/next)"},
		{"<p>Keep</p><p hidden>Hide</p><p style='visibility: hidden'>Hide</p><script>Hide</script>", "Keep"},
		{"<p><a href='javascript:alert(1)'>Click</a> <a href='file:///tmp/x'>File</a></p>", "Click File"},
	}
	for _, tc := range cases {
		p := &ParsedMessage{HTMLPreview: parseHTML(tc.html)}
		if got := p.DisplayText(); got != tc.want {
			t.Errorf("HTML %q: got %q, want %q", tc.html, got, tc.want)
		}
	}
}
func TestHTMLFallbackWithEmptyPlainAlternativeAndCharset(t *testing.T) {
	raw := "MIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain\r\n\r\n \r\n--x\r\nContent-Type: text/html; charset=windows-1252\r\n\r\n<p>Caf\xe9 \x96 <a href=https://example.com>details</a></p>\r\n--x--\r\n"
	p, err := ParseBytes([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != "Café – details (https://example.com)" {
		t.Fatalf("bad HTML charset/fallback: %q", p.Body)
	}
	plain, err := ParseBytes([]byte("Content-Type: text/plain; charset=iso-8859-1\r\n\r\nCaf\xe9"))
	if err != nil || plain.Body != "Café" {
		t.Fatalf("plain charset: %v %#v", err, plain)
	}
}
func TestHTMLAttachmentsDoNotReplaceMessageBody(t *testing.T) {
	raw := "Content-Type: multipart/mixed; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain\r\n\r\nBody\r\n--x\r\nContent-Type: text/html; name=report.html\r\nContent-Disposition: attachment; filename=report.html\r\n\r\n<p>Attachment</p>\r\n--x--\r\n"
	p, err := ParseBytes([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if p.DisplayText() != "Body" || len(p.HTMLPreview) != 0 || len(p.Attachments) != 1 {
		t.Fatal("HTML attachment became body preview")
	}
}
func TestSupportedEmailLinks(t *testing.T) {
	for _, raw := range []string{"https://example.com", "http://example.com", "mailto:person@example.com", "//example.com/x"} {
		if SafeLink(raw) == nil {
			t.Errorf("supported URL rejected: %s", raw)
		}
	}
	for _, raw := range []string{"javascript:alert(1)", "file:///etc/passwd", "data:text/html,Hi", "#part", "/relative", "https:", "https://example.com/\x00"} {
		if SafeLink(raw) != nil {
			t.Errorf("unsupported URL accepted: %s", raw)
		}
	}
}

func TestUnknownCharsetStillDisplaysReadableMail(t *testing.T) {
	p, err := ParseBytes([]byte("Content-Type: text/html; charset=unknown-charset\r\n\r\n<p>Readable café</p>"))
	if err != nil || p.DisplayText() != "Readable café" {
		t.Fatalf("unknown charset hid content: %v %#v", err, p)
	}
}

func TestNestedAlternativeAndMixedBodySectionsKeepOrder(t *testing.T) {
	raw := `MIME-Version: 1.0
Content-Type: multipart/mixed; boundary=outer

--outer
Content-Type: text/plain; charset=utf-8

Intro https://example.com/intro
--outer
Content-Type: multipart/alternative; boundary=alt

--alt
Content-Type: text/plain; charset=utf-8

Plain quote version
--alt
Content-Type: multipart/related; boundary=related

--related
Content-Type: text/html; charset=utf-8

<h2>Formatted section</h2><p><a href="https://example.com/details">Details</a></p>
--related
Content-Type: image/png; name=logo.png
Content-Disposition: inline; filename=logo.png
Content-Transfer-Encoding: base64

aGVsbG8=
--related--
--alt--
--outer
Content-Type: text/plain; charset=utf-8

Footer
--outer--
`
	p, err := ParseBytes([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != "Intro https://example.com/intro\n\nPlain quote version\n\nFooter" {
		t.Fatalf("bad mixed quote body: %q", p.Body)
	}
	display := p.DisplayText()
	intro, middle, footer := strings.Index(display, "Intro"), strings.Index(display, "Formatted section"), strings.Index(display, "Footer")
	if intro < 0 || middle <= intro || footer <= middle || strings.Contains(display, "Plain quote version") {
		t.Fatalf("mixed sections dropped, duplicated or reordered: %q", display)
	}
	if !strings.Contains(display, "Details (https://example.com/details)") || len(p.Attachments) != 1 || string(p.Attachments[0].Data) != "hello" {
		t.Fatal("related HTML links or attachment were lost")
	}
}
