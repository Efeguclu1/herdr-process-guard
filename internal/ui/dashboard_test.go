package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

func TestDashboardAnswersRelationshipAndLeftoverQuestions(t *testing.T) {
	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.UTC)
	matures := now.Add(15 * time.Minute)
	report := model.Report{
		ObservedAt: now,
		Summary:    model.Summary{Trees: 1, Processes: 1, Idle: 1},
		Trees: []model.Tree{{
			RootPID: 30,
			Processes: []model.Process{{
				Key: model.ProcessKey{PID: 30}, PPID: 20, Name: "Python",
				Command: "python -m http.server 4173", CommandSummary: "Python",
			}},
			Attribution: model.Attribution{
				WorkspaceName: "Klinika", PaneID: "w2:p1Y", TabLabel: "codex polish", PaneCWD: "/Users/example/Klinika",
				Agent: "codex", AgentStatus: "idle",
				Confidence: model.ConfidenceHigh, Relationship: model.RelationshipCurrentAgentAncestry,
				AgentPID: 20,
				Lineage: []model.LineageProcess{
					{PID: 10, Name: "zsh", Role: "pane_shell"},
					{PID: 20, Name: "codex", Role: "agent"},
					{PID: 30, Name: "Python", Role: "workload"},
				},
			},
			Lifecycle: model.LifecycleIdle, Policy: model.PolicyUnreviewed,
			LeftoverAssessment: model.LeftoverMonitoring, HistoryMaturesAt: &matures,
		}},
	}
	dashboard := Dashboard{report: report, rows: flatten(report), width: 220, height: 40}
	view := dashboard.View()
	for _, expected := range []string{
		"Klinika › tab “codex polish”",
		"source pane: Klinika › tab “codex polish” · internal w2:p1Y",
		"RELATED: YES · LEFTOVER: MONITORING",
		"relationship: YES — the current parent chain passes through codex PID 20",
		"lineage: zsh 10 → codex 20 → Python 30",
		"leftover: NOT YET DETERMINED",
		"activity: IDLE",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("dashboard missing %q in:\n%s", expected, view)
		}
	}
}
