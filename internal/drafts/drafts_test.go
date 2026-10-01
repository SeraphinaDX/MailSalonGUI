// SPDX-License-Identifier: GPL-3.0-only

package drafts

import (
	"os"
	"testing"

	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

func TestDraftRoundTripAndSafeIDs(t *testing.T) {
	dir := t.TempDir()
	s, err := Save(dir, Saved{Account: "work", Draft: mimeutil.Draft{To: "friend@example.com", Subject: "A draft", MemoryAttachments: []mimeutil.Attachment{{Filename: "file.txt", Data: []byte("content")}}}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := List(dir)
	if err != nil || len(out) != 1 || string(out[0].Draft.MemoryAttachments[0].Data) != "content" {
		t.Fatalf("draft round trip: %v %v", out, err)
	}
	files, _ := os.ReadDir(dir)
	info, _ := files[0].Info()
	if info.Mode().Perm() != 0600 {
		t.Fatal("draft is not private")
	}
	if _, err := Save(dir, Saved{ID: "../bad"}); err == nil {
		t.Fatal("unsafe ID accepted")
	}
	if err := Delete(dir, "../bad"); err == nil {
		t.Fatal("unsafe delete accepted")
	}
	if err := Delete(dir, s.ID); err != nil {
		t.Fatal(err)
	}
	out, err = List(dir)
	if err != nil || len(out) != 0 {
		t.Fatal("delete failed")
	}
}
