package platform

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

func TestParsePSLine(t *testing.T) {
	line := fmt.Sprintf("42 7 42 7 %d Mon Aug 24 20:30:01 2026 S 2048 1:02.50 /opt/homebrew/bin/node node /work/server.js --token secret", os.Getuid())
	process, ok := parsePSLine(line)
	if !ok {
		t.Fatal("expected process line to parse")
	}
	if process.Key.PID != 42 || process.PPID != 7 || process.PGID != 42 || process.SessionID != 7 {
		t.Fatalf("unexpected identity: %+v", process)
	}
	if process.RSSBytes != 2048*1024 {
		t.Fatalf("unexpected rss: %d", process.RSSBytes)
	}
	if process.CPUTimeMillis != 62_500 {
		t.Fatalf("unexpected cpu time: %d", process.CPUTimeMillis)
	}
	if process.CommandSummary != "node server.js" {
		t.Fatalf("summary should be sanitized, got %q", process.CommandSummary)
	}
	if strings.Contains(process.CommandSummary, "secret") {
		t.Fatal("secret leaked into summary")
	}
}

func TestScannerAttachesCWDAndSockets(t *testing.T) {
	started := time.Now().Format("Mon Jan _2 15:04:05 2006")
	ps := fmt.Sprintf("42 7 42 7 %d %s S 2048 0:00.10 /usr/bin/node node server.js\n", os.Getuid(), started)
	runner := func(name string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case name == "/bin/ps":
			return []byte(ps), nil
		case strings.Contains(joined, "-d cwd"):
			return []byte("p42\nn/work/project\n"), nil
		case strings.Contains(joined, "-iTCP"):
			return []byte("p42\nPTCP\nn*:5173\nTST=LISTEN\n"), nil
		default:
			return nil, fmt.Errorf("unexpected command %s %s", name, joined)
		}
	}
	scanner := &Scanner{Run: runner, UID: os.Getuid()}
	processes, warnings, err := scanner.Processes()
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	process := processes[42]
	if process.CWD != "/work/project" {
		t.Fatalf("unexpected cwd: %q", process.CWD)
	}
	if !process.HasListener() || process.Sockets[0].Local != "*:5173" {
		t.Fatalf("listener not attached: %+v", process.Sockets)
	}
}

func TestParseCPUTimeWithDays(t *testing.T) {
	if got := parseCPUTime("2-01:02:03.50"); got != 176_523_500 {
		t.Fatalf("got %d", got)
	}
}

func TestAttachSocketsKeepsPartialLsofOutput(t *testing.T) {
	scanner := &Scanner{Run: func(name string, args ...string) ([]byte, error) {
		return []byte("p42\nPTCP\nn127.0.0.1:5173\nTST=LISTEN\n"), errors.New("lsof exited after a process disappeared")
	}}
	processes := map[int]model.Process{42: {Key: model.ProcessKey{PID: 42}}}
	if err := scanner.attachSockets(processes, []int{42}); err != nil {
		t.Fatalf("partial structured output should remain usable: %v", err)
	}
	if !processes[42].HasListener() {
		t.Fatalf("listener from partial output was discarded: %+v", processes[42])
	}
}

func TestParsePSLineUsesArgvWhenMacOSCommIsTruncated(t *testing.T) {
	line := fmt.Sprintf("42 7 42 0 %d Mon Aug 24 20:30:01 2026 S 2048 0:01.00 /opt/homebrew/Ce /opt/homebrew/Cellar/python/Contents/MacOS/Python -m http.server", os.Getuid())
	process, ok := parsePSLine(line)
	if !ok {
		t.Fatal("expected process line to parse")
	}
	if process.Name != "Python" || process.Executable != "/opt/homebrew/Cellar/python/Contents/MacOS/Python" {
		t.Fatalf("expected argv executable identity, got name=%q executable=%q", process.Name, process.Executable)
	}
}
