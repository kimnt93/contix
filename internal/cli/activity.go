package cli

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// startActivity prints a stable status line when output is redirected and an
// animated spinner in an interactive terminal. The returned function stops and
// clears the spinner before the final result line is printed.
func startActivity(label string) func() {
	info, _ := os.Stdout.Stat()
	interactive := info != nil && info.Mode()&os.ModeCharDevice != 0
	if !interactive {
		fmt.Println(label + "...")
		return func() {}
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		frames := []string{"|", "/", "-", "\\"}
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		index := 0
		for {
			fmt.Printf("\r%s [%s]", label, frames[index%len(frames)])
			index++
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		close(done)
		wg.Wait()
		fmt.Printf("\r%-*s\r", len(label)+4, "")
	}
}
