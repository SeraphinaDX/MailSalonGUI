// SPDX-License-Identifier: GPL-3.0-only

// Package drafts stores private, local composition files independently of sync.
package drafts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SeraphinaDX/MailSalonGUI/internal/mimeutil"
)

type Saved struct {
	ID            string
	Account       string
	Draft         mimeutil.Draft
	Sign, Encrypt bool
	Updated       time.Time
}

func Directory() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, "mailsalongui", "drafts")
}

func Save(dir string, s Saved) (Saved, error) {
	if s.ID == "" {
		s.ID = time.Now().UTC().Format("20060102T150405.000000000")
	}
	s.Updated = time.Now()
	if filepath.Base(s.ID) != s.ID || strings.ContainsAny(s.ID, "/\\") {
		return s, os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return s, err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return s, err
	}
	f, err := os.CreateTemp(dir, ".draft-")
	if err != nil {
		return s, err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		return s, err
	}
	return s, os.Rename(f.Name(), filepath.Join(dir, s.ID+".json"))
}
func List(dir string) ([]Saved, error) {
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Saved
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		var s Saved
		if err = json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		s.ID = strings.TrimSuffix(f.Name(), ".json")
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}
func Delete(dir, id string) error {
	if id == "" {
		return nil
	}
	if filepath.Base(id) != id || strings.ContainsAny(id, "/\\") {
		return os.ErrInvalid
	}
	err := os.Remove(filepath.Join(dir, id+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
