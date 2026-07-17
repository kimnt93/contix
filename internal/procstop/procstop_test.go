package procstop

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestUniqueProcessNames(t *testing.T) {
	got := unique([]string{"codex", "", "codex", " cursor ", "bad/name"})
	want := []string{"codex", "cursor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unique() = %v, want %v", got, want)
	}
}

func TestCloseStopsMatchingProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process-name integration test")
	}
	const name = "ctix-stop-test"
	data, err := os.ReadFile("/bin/sleep")
	if err != nil {
		t.Skipf("sleep is unavailable: %v", err)
	}
	bin := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(bin, data, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(time.Second):
		}
	}()

	deadline := time.Now().Add(time.Second)
	for {
		running, err := unixRunning(name)
		if err != nil {
			t.Fatal(err)
		}
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("test process did not become visible")
		}
		time.Sleep(10 * time.Millisecond)
	}

	stopped, err := Close([]string{name})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stopped, []string{name}) {
		t.Fatalf("stopped = %v, want %s", stopped, name)
	}
	running, err := unixRunning(name)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("process is still running")
	}
}
