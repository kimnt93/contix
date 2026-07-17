// Package procstop closes applications before their live state is collected.
package procstop

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

const gracefulWait = 2 * time.Second

// Close stops every running executable in names and returns the executable
// names that were found. Names must be executable basenames, not paths.
func Close(names []string) ([]string, error) {
	names = unique(names)
	if runtime.GOOS == "windows" {
		return closeWindows(names)
	}
	return closeUnix(names)
}

func closeUnix(names []string) ([]string, error) {
	if _, err := exec.LookPath("pgrep"); err != nil {
		return nil, fmt.Errorf("force-close requires pgrep: %w", err)
	}
	if _, err := exec.LookPath("pkill"); err != nil {
		return nil, fmt.Errorf("force-close requires pkill: %w", err)
	}

	var stopped []string
	for _, name := range names {
		running, err := unixRunning(name)
		if err != nil {
			return stopped, err
		}
		if !running {
			continue
		}
		if err := exec.Command("pkill", "-TERM", "-x", name).Run(); err != nil {
			stillRunning, checkErr := unixRunning(name)
			if checkErr != nil || stillRunning {
				return stopped, fmt.Errorf("stop process %s: %w", name, err)
			}
		}
		stopped = append(stopped, name)
	}

	deadline := time.Now().Add(gracefulWait)
	for time.Now().Before(deadline) {
		anyRunning := false
		for _, name := range stopped {
			running, err := unixRunning(name)
			if err != nil {
				return stopped, err
			}
			anyRunning = anyRunning || running
		}
		if !anyRunning {
			return stopped, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	for _, name := range stopped {
		running, err := unixRunning(name)
		if err != nil {
			return stopped, err
		}
		if running {
			_ = exec.Command("pkill", "-KILL", "-x", name).Run()
		}
	}
	return stopped, nil
}

func unixRunning(name string) (bool, error) {
	err := exec.Command("pgrep", "-x", name).Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("find process %s: %w", name, err)
}

func closeWindows(names []string) ([]string, error) {
	if _, err := exec.LookPath("taskkill"); err != nil {
		return nil, fmt.Errorf("force-close requires taskkill: %w", err)
	}
	var stopped []string
	for _, name := range names {
		image := name
		if !strings.HasSuffix(strings.ToLower(image), ".exe") {
			image += ".exe"
		}
		err := exec.Command("taskkill", "/F", "/T", "/IM", image).Run()
		if err == nil {
			stopped = append(stopped, name)
			continue
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			continue // no matching process
		}
		return stopped, fmt.Errorf("stop process %s: %w", name, err)
	}
	return stopped, nil
}

func unique(names []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, `/\\`) || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
