package herdr

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
)

func TestClientReadsSnapshotAndProcessInfo(t *testing.T) {
	responses := map[string]any{
		"session.snapshot": map[string]any{"type": "session_snapshot", "snapshot": map[string]any{
			"version": "0.8.0", "protocol": 1,
			"workspaces": []any{map[string]any{"workspace_id": "w1", "label": "demo", "agent_status": "idle"}},
			"tabs":       []any{map[string]any{"tab_id": "w1:t1", "workspace_id": "w1", "label": "API work", "number": 1}},
			"panes":      []any{map[string]any{"pane_id": "w1:p1", "tab_id": "w1:t1", "terminal_title_stripped": "Server", "foreground_cwd": "/repo"}},
			"agents":     []any{},
		}},
		"pane.process_info": map[string]any{"type": "pane_process_info", "process_info": map[string]any{"pane_id": "w1:p1", "shell_pid": 42, "foreground_processes": []any{}}},
	}
	done := make(chan struct{}, 2)
	client := &Client{Dial: func() (net.Conn, error) {
		clientSide, serverSide := net.Pipe()
		go func(connection net.Conn) {
			defer func() { done <- struct{}{} }()
			var request map[string]any
			_ = json.NewDecoder(bufio.NewReader(connection)).Decode(&request)
			_ = json.NewEncoder(connection).Encode(map[string]any{"id": request["id"], "result": responses[request["method"].(string)]})
			_ = connection.Close()
		}(serverSide)
		return clientSide, nil
	}}
	snapshot, err := client.SessionSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != "0.8.0" || snapshot.Workspaces[0].Label != "demo" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.Tabs[0].Label != "API work" || snapshot.Panes[0].TerminalTitle != "Server" || snapshot.Panes[0].ForegroundCWD != "/repo" {
		t.Fatalf("human pane metadata was not decoded: %+v", snapshot)
	}
	info, err := client.PaneProcessInfo("w1:p1")
	if err != nil {
		t.Fatal(err)
	}
	if info.ShellPID == nil || *info.ShellPID != 42 {
		t.Fatalf("unexpected process info: %+v", info)
	}
	<-done
	<-done
}
