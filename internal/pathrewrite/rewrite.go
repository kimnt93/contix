// Package pathrewrite rewrites absolute machine paths inside restored tool
// state so that sessions/projects resume correctly on a machine with a
// different username, home directory, or operating system.
package pathrewrite

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Mapping maps an old absolute path prefix to a new one.
type Mapping struct {
	Old string
	New string
}

// Rewriter applies a set of prefix mappings to file contents and directory
// names.
type Rewriter struct {
	maps     []Mapping
	targetOS string
}

// New builds a Rewriter. The auto mapping (sourceHome -> current home) is added
// first; explicit user mappings take precedence because they are appended and
// applied in order with longest-prefix-first sorting.
func New(sourceHome string, userMaps []Mapping) *Rewriter {
	r := &Rewriter{targetOS: runtime.GOOS}
	destHome, _ := os.UserHomeDir()
	if sourceHome != "" && destHome != "" && sourceHome != destHome {
		r.maps = append(r.maps, Mapping{Old: sourceHome, New: destHome})
	}
	r.maps = append(r.maps, userMaps...)
	// Longest Old first so more specific mappings win.
	sort.SliceStable(r.maps, func(i, j int) bool {
		return len(r.maps[i].Old) > len(r.maps[j].Old)
	})
	return r
}

// Active reports whether the rewriter would change anything.
func (r *Rewriter) Active() bool { return len(r.maps) > 0 }

// Mappings returns the effective mappings (for reporting).
func (r *Rewriter) Mappings() []Mapping { return r.maps }

// contentExts are the file types whose contents are rewritten.
var contentExts = map[string]bool{
	".jsonl": true, ".json": true, ".md": true,
	".toml": true, ".txt": true, ".yaml": true, ".yml": true,
}

// Apply rewrites directory names and file contents under root in place.
// Returns the number of files whose contents changed and dirs renamed.
func (r *Rewriter) Apply(root string) (filesChanged, dirsRenamed int, err error) {
	if !r.Active() {
		return 0, 0, nil
	}
	dirsRenamed, err = r.renameDirs(root)
	if err != nil {
		return 0, dirsRenamed, err
	}
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() || !contentExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		changed, e := r.rewriteFile(p)
		if e != nil {
			return e
		}
		if changed {
			filesChanged++
		}
		return nil
	})
	return filesChanged, dirsRenamed, err
}

// rewriteFile rewrites path prefixes inside a single file, preserving mode.
func (r *Rewriter) rewriteFile(p string) (bool, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return false, err
	}
	jsonMode := strings.HasSuffix(strings.ToLower(p), ".json") ||
		strings.HasSuffix(strings.ToLower(p), ".jsonl")
	out := r.Content(data, jsonMode)
	if string(out) == string(data) {
		return false, nil
	}
	info, err := os.Stat(p)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode().Perm()
	}
	return true, os.WriteFile(p, out, mode)
}

// Content applies all mappings to a byte slice. When jsonMode is set and the
// target OS is Windows, backslashes in the replacement are doubled so the JSON
// stays valid.
func (r *Rewriter) Content(data []byte, jsonMode bool) []byte {
	s := string(data)
	for _, m := range r.maps {
		newVal := m.New
		if jsonMode && r.targetOS == "windows" {
			newVal = strings.ReplaceAll(m.New, `\`, `\\`)
		}
		for _, ov := range variants(m.Old) {
			s = strings.ReplaceAll(s, ov, newVal)
		}
	}
	return []byte(s)
}

// variants returns the forms an old prefix may appear as across OSes: POSIX
// (forward slash), Windows (backslash), and JSON-escaped Windows (double
// backslash).
func variants(p string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	posix := filepath.ToSlash(p)
	win := strings.ReplaceAll(posix, "/", `\`)
	add(posix)
	add(win)
	add(strings.ReplaceAll(win, `\`, `\\`)) // JSON-escaped
	return out
}

// renameDirs renames directories whose basename embeds an old home path in
// Claude's lossy dash-encoded form (e.g. "-home-kim-proj" -> "-Users-kim-proj").
func (r *Rewriter) renameDirs(root string) (int, error) {
	type rename struct{ from, to string }
	var todo []rename

	err := filepath.WalkDir(root, func(p string, d os.DirEntry, werr error) error {
		if werr != nil || !d.IsDir() {
			return nil
		}
		base := d.Name()
		for _, m := range r.maps {
			oldEnc := encode(m.Old)
			if oldEnc != "" && strings.HasPrefix(base, oldEnc) {
				newBase := encode(m.New) + strings.TrimPrefix(base, oldEnc)
				todo = append(todo, rename{from: p, to: filepath.Join(filepath.Dir(p), newBase)})
				break
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	// Rename deepest paths first to keep parent paths valid.
	sort.Slice(todo, func(i, j int) bool {
		return strings.Count(todo[i].from, string(os.PathSeparator)) >
			strings.Count(todo[j].from, string(os.PathSeparator))
	})
	n := 0
	for _, rn := range todo {
		if rn.from == rn.to {
			continue
		}
		if _, err := os.Stat(rn.to); err == nil {
			// Destination exists; skip to avoid clobbering.
			continue
		}
		if err := os.Rename(rn.from, rn.to); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// encode reproduces Claude Code's lossy path->folder encoding: every character
// that is not an ASCII letter or digit becomes a dash.
func encode(p string) string {
	var b strings.Builder
	for _, r := range filepath.ToSlash(p) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// ParseMapping parses an "OLD=NEW" mapping string.
func ParseMapping(s string) (Mapping, bool) {
	i := strings.Index(s, "=")
	if i <= 0 || i == len(s)-1 {
		return Mapping{}, false
	}
	return Mapping{Old: s[:i], New: s[i+1:]}, true
}
