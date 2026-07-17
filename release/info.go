// Package releaseinfo exposes release metadata embedded in the contix binary.
package releaseinfo

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionData string

//go:embed NOTES
var notesData string

// Version returns the release version compiled into this binary.
func Version() string { return strings.TrimSpace(versionData) }

// Notes returns the short release notes compiled into this binary.
func Notes() string { return strings.TrimSpace(notesData) }
