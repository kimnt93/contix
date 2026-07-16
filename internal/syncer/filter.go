package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ageSensitivePrefixes are directories that hold time-ordered session
// transcripts. Files under these can be pruned by age to keep bundles small.
// Everything else (config, memory, skills, rules) is always kept.
var ageSensitivePrefixes = []string{
	"sessions/",
	"archived_sessions/",
	"projects/",
}

// filterByAge drops files under session directories whose modification time is
// older than the given number of days. High-value state (config, memory,
// skills, rules) is never pruned.
func filterByAge(home string, rels []string, days int) []string {
	if days <= 0 {
		return rels
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	out := rels[:0:0]
	for _, rel := range rels {
		if !ageSensitive(rel) {
			out = append(out, rel)
			continue
		}
		abs := filepath.Join(home, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			// If we cannot stat it, keep it rather than silently dropping.
			out = append(out, rel)
			continue
		}
		if info.ModTime().After(cutoff) {
			out = append(out, rel)
		}
	}
	return out
}

// ageSensitive reports whether a relative path lives under a session directory.
func ageSensitive(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range ageSensitivePrefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}
