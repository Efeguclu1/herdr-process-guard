package presentation

import (
	"strings"
	"testing"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

func TestTreeExplainsRelationshipLineageAndLeftoverState(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.UTC)
	matures := now.Add(20 * time.Minute)
	tree := model.Tree{
		RootPID:   30,
		Processes: []model.Process{{Key: model.ProcessKey{PID: 30}, PPID: 20, Name: "node", CommandSummary: "node server.js"}},
		Attribution: model.Attribution{
			WorkspaceName: "Klinika", PaneID: "w2:p22", TabID: "w2:t1F", TabLabel: "agent talking fix",
			PaneCWD: "/Users/example/Klinika", Agent: "codex", AgentStatus: "idle",
			Confidence: model.ConfidenceHigh, Relationship: model.RelationshipCurrentAgentAncestry,
			AgentPID: 20,
			Lineage: []model.LineageProcess{
				{PID: 10, Name: "zsh", Role: "pane_shell"},
				{PID: 20, Name: "codex", Role: "agent"},
				{PID: 30, Name: "node", Role: "workload"},
			},
		},
		Lifecycle: model.LifecycleIdle, Policy: model.PolicyUnreviewed,
		LeftoverAssessment: model.LeftoverMonitoring, HistoryMaturesAt: &matures,
	}

	got := TreeAt(tree, now)
	for _, expected := range []string{
		"Klinika › tab “agent talking fix”", "internal w2:p22", "related: YES", "current parent chain passes through codex PID 20",
		"zsh 10 → codex 20 → node 30", "20m more observation is required",
		"activity: IDLE",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q in:\n%s", expected, got)
		}
	}
}

func TestPaneLocationCombinesNumericTabWithTerminalTitle(t *testing.T) {
	attribution := model.Attribution{WorkspaceName: "Klinika", PaneID: "w2:p21", TabLabel: "6", PaneTitle: "Landing Page Redesign"}
	if got := PaneLocation(attribution); got != "Klinika › tab 6 “Landing Page Redesign”" {
		t.Fatalf("unexpected pane location: %q", got)
	}
}
