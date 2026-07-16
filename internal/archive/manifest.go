// Package archive handles bundling a tool's state into a compressed archive
// with a fidelity manifest (SHA-256 per file, tool version, source machine
// identity), and restoring/verifying it on another machine.
package archive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// SchemaVersion is bumped when the manifest format changes incompatibly.
const SchemaVersion = 1

// FileEntry records one archived file for fidelity verification.
type FileEntry struct {
	Path   string `json:"path"`   // forward-slash path relative to tool home
	Size   int64  `json:"size"`   // bytes
	Mode   uint32 `json:"mode"`   // unix file mode bits
	SHA256 string `json:"sha256"` // hex digest of the file contents
}

// Manifest describes the contents and provenance of one tool's bundle.
type Manifest struct {
	Schema      int         `json:"schema"`
	Tool        string      `json:"tool"`
	ToolVersion string      `json:"tool_version,omitempty"`
	SourceOS    string      `json:"source_os"`   // linux | darwin | windows
	SourceHome  string      `json:"source_home"` // absolute home dir at push time
	SourceTool  string      `json:"source_tool"` // absolute tool dir at push time
	CreatedAt   time.Time   `json:"created_at"`
	Files       []FileEntry `json:"files"`
}

// ManifestName is the manifest filename stored alongside each bundle.
const ManifestName = "manifest.json"

// BundleName is the compressed archive filename stored per tool.
const BundleName = "bundle.tar.gz"

// NewManifest builds a manifest header (without files) for the current machine.
func NewManifest(toolName, toolVersion, toolHome string) Manifest {
	return Manifest{
		Schema:      SchemaVersion,
		Tool:        toolName,
		ToolVersion: toolVersion,
		SourceOS:    runtime.GOOS,
		SourceHome:  homeDir(),
		SourceTool:  toolHome,
		CreatedAt:   time.Now().UTC(),
	}
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

// WriteManifest writes a manifest as indented JSON.
func WriteManifest(path string, m Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ReadManifest loads a manifest from disk.
func ReadManifest(path string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	return m, err
}
