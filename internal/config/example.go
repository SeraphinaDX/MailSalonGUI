// SPDX-License-Identifier: GPL-3.0-only

package config

import (
	_ "embed"
	"path/filepath"
)

// Example is also used by the first-run configuration editor and CLI initializer.
//
//go:embed config.toml.example
var Example string

// ResolvePath applies the same expansion as Load before a caller writes a file.
func ResolvePath(path string) (string, error) { return filepath.Abs(expandPath(path)) }
