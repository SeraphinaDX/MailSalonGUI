// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

type urlRecordingApp struct {
	fyne.App
	opened string
}

func (a *urlRecordingApp) OpenURL(u *url.URL) error { a.opened = u.String(); return nil }

func TestHTMLMailPreviewUsesHTMLAlternativeAndClickableLinks(t *testing.T) {
	a, q := demoApp(t)
	raw, err := os.ReadFile("../mimeutil/testdata/newsletter.eml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.messages[0].Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	a.selectMessage(0)
	waitForMessage(t, a, q)
	links := map[string]string{}
	bold, italic, code, heading := false, false, false, false
	var visible strings.Builder
	for _, segment := range a.body.Segments {
		visible.WriteString(segment.Textual())
		switch s := segment.(type) {
		case *widget.HyperlinkSegment:
			links[strings.TrimSpace(s.Text)] = s.URL.String()
		case *widget.TextSegment:
			bold = bold || s.Style.TextStyle.Bold
			italic = italic || s.Style.TextStyle.Italic
			code = code || s.Style.TextStyle.Monospace
			heading = heading || s.Style.SizeName == widget.RichTextStyleHeading.SizeName
		}
	}
	if !strings.Contains(visible.String(), "You're invited!") || strings.Contains(visible.String(), "Please view the formatted invitation") {
		t.Fatal("preview used the incomplete plain alternative")
	}
	if links["Reserve your place"] != "https://example.com/rsvp?event=go&source=mail" || links["Event details"] != "https://example.com/details" || links["Email the organiser"] != "mailto:community@example.com" {
		t.Fatalf("missing/wrong clickable links: %v", links)
	}
	if !bold || !italic || !code || !heading {
		t.Fatal("HTML styles were lost")
	}
	recorder := &urlRecordingApp{App: a.Fyne}
	fyne.SetCurrentApp(recorder)
	defer fyne.SetCurrentApp(a.Fyne)
	for _, segment := range a.body.Segments {
		if link, ok := segment.(*widget.HyperlinkSegment); ok && strings.TrimSpace(link.Text) == "Reserve your place" {
			test.Tap(link.Visual().(*fyne.Container).Objects[0].(*widget.Hyperlink))
		}
	}
	if recorder.opened != links["Reserve your place"] {
		t.Fatal("tapping the visible link did not open its destination")
	}
	screenshot(t, a.Window, "html-mail-preview")
}
func TestPlainMailClickableURLsPreserveOriginalText(t *testing.T) {
	p := &mimeutil.ParsedMessage{Body: "See (https://example.com/a_(b)).\nEmail mailto:person@example.com.\nLiteral *stars* and <tags>."}
	segments := messageBodySegments(p)
	var rebuilt strings.Builder
	var links []string
	for _, segment := range segments {
		rebuilt.WriteString(segment.Textual())
		if link, ok := segment.(*widget.HyperlinkSegment); ok {
			links = append(links, link.URL.String())
		}
	}
	if rebuilt.String() != p.Body {
		t.Fatalf("plain mail changed: %q", rebuilt.String())
	}
	if len(links) != 2 || links[0] != "https://example.com/a_(b)" || links[1] != "mailto:person@example.com" {
		t.Fatalf("incorrect plain links: %v", links)
	}
}

func TestBareTrackingLinkKeepsFullDestinationWithoutOversizedLabel(t *testing.T) {
	target := "https://example.com/track?token=" + strings.Repeat("a", 200)
	p := &mimeutil.ParsedMessage{HTMLPreview: []mimeutil.BodySpan{{Text: target, Link: target}}}
	segments := messageBodySegments(p)
	link, ok := segments[0].(*widget.HyperlinkSegment)
	if !ok || len(link.Text) >= len(target) || link.URL.String() != target || p.DisplayText() != target {
		t.Fatal("long-link destination was lost or preview label was not shortened")
	}
}
