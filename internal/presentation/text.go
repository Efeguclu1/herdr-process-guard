package presentation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/guard"
	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

func Report(report model.Report) string {
	var output strings.Builder
	fmt.Fprintf(&output, "Process Guard — %d workload trees, %d processes\n", report.Summary.Trees, report.Summary.Processes)
	fmt.Fprintf(&output, "active %d  idle %d  orphan candidates %d  suspicious %d  intentional %d\n\n",
		report.Summary.Active, report.Summary.Idle, report.Summary.OrphanCandidates, report.Summary.Suspicious, report.Summary.Intentional)
	if len(report.Trees) == 0 {
		output.WriteString("No attributed long-running workloads found.\n")
	}
	for _, tree := range report.Trees {
		output.WriteString(TreeAt(tree, report.ObservedAt))
		output.WriteByte('\n')
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(&output, "warning: %s\n", warning)
	}
	return output.String()
}

func Tree(tree model.Tree) string {
	return TreeAt(tree, time.Now().UTC())
}

func TreeAt(tree model.Tree, observedAt time.Time) string {
	var output strings.Builder
	fmt.Fprintf(&output, "%s  [related: %s · leftover: %s]\n", PaneLocation(tree.Attribution), RelationshipBadge(tree), LeftoverBadge(tree))
	fmt.Fprintf(&output, "  source: %s\n", PaneDetail(tree.Attribution))
	fmt.Fprintf(&output, "  relationship: %s\n", RelationshipExplanation(tree))
	if lineage := Lineage(tree); lineage != "" {
		fmt.Fprintf(&output, "  lineage: %s\n", lineage)
	}
	fmt.Fprintf(&output, "  leftover: %s\n", LeftoverExplanation(tree, observedAt))
	fmt.Fprintf(&output, "  activity: %s\n", ActivityExplanation(tree, observedAt))
	byPID := map[int]model.Process{}
	children := map[int][]int{}
	for _, process := range tree.Processes {
		byPID[process.Key.PID] = process
		children[process.PPID] = append(children[process.PPID], process.Key.PID)
	}
	for parent := range children {
		sort.Ints(children[parent])
	}
	writeNode(&output, tree.RootPID, "", true, byPID, children)
	if tree.Attribution.Agent != "" {
		fmt.Fprintf(&output, "  agent: %s (%s)\n", tree.Attribution.Agent, tree.Attribution.AgentStatus)
	}
	for _, evidence := range tree.Evidence {
		fmt.Fprintf(&output, "  • %s\n", evidence.Message)
	}
	return output.String()
}

func PaneLocation(attribution model.Attribution) string {
	workspace := attribution.WorkspaceName
	if workspace == "" {
		workspace = attribution.WorkspaceID
	}
	tab := strings.TrimSpace(attribution.TabLabel)
	title := strings.TrimSpace(attribution.PaneTitle)
	paneLabel := strings.TrimSpace(attribution.PaneLabel)
	if paneLabel != "" {
		title = paneLabel
	}
	switch {
	case tab != "" && isNumeric(tab) && title != "" && !strings.EqualFold(title, workspace):
		return fmt.Sprintf("%s › tab %s “%s”", workspace, tab, title)
	case tab != "" && !isNumeric(tab):
		return fmt.Sprintf("%s › tab “%s”", workspace, tab)
	case tab != "":
		return fmt.Sprintf("%s › tab %s", workspace, tab)
	case title != "" && !strings.EqualFold(title, workspace):
		return fmt.Sprintf("%s › “%s”", workspace, title)
	default:
		return workspace + " › pane " + attribution.PaneID
	}
}

func PaneDetail(attribution model.Attribution) string {
	parts := []string{PaneLocation(attribution)}
	if attribution.PaneID != "" {
		parts = append(parts, "internal "+attribution.PaneID)
	}
	if attribution.PaneCWD != "" {
		parts = append(parts, "pane cwd "+attribution.PaneCWD)
	}
	return strings.Join(parts, " · ")
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func RelationshipBadge(tree model.Tree) string {
	switch tree.Attribution.Relationship {
	case model.RelationshipCurrentAgentAncestry:
		return "YES"
	case model.RelationshipRecordedAgentWindow:
		return "LIKELY"
	case model.RelationshipSharedPaneSession:
		return "POSSIBLE"
	default:
		return "UNPROVEN"
	}
}

func LeftoverBadge(tree model.Tree) string {
	switch tree.LeftoverAssessment {
	case model.LeftoverLikely:
		return "LIKELY"
	case model.LeftoverIntentional:
		return "INTENTIONAL"
	case model.LeftoverAgentActive:
		return "NO — AGENT ACTIVE"
	case model.LeftoverRecentActivity:
		return "NOT YET — PROCESS ACTIVE"
	case model.LeftoverMonitoring:
		return "MONITORING"
	default:
		return "UNCLEAR"
	}
}

func RelationshipExplanation(tree model.Tree) string {
	agent := tree.Attribution.Agent
	if agent == "" {
		agent = "coding agent"
	}
	switch tree.Attribution.Relationship {
	case model.RelationshipCurrentAgentAncestry:
		if tree.Attribution.AgentPID > 0 {
			return fmt.Sprintf("YES — the current parent chain passes through %s PID %d.", agent, tree.Attribution.AgentPID)
		}
		return "YES — the current parent chain passes through the coding agent."
	case model.RelationshipRecordedAgentWindow:
		return fmt.Sprintf("LIKELY — it started during a recorded %s activity window in this pane.", agent)
	case model.RelationshipSharedPaneSession:
		return fmt.Sprintf("POSSIBLE — it shares the pane OS session where %s was observed, but no direct parent chain remains.", agent)
	default:
		return "UNPROVEN — it is in the pane, but Process Guard has no agent ancestry or recorded agent-window evidence."
	}
}

func Lineage(tree model.Tree) string {
	parts := make([]string, 0, len(tree.Attribution.Lineage))
	for _, process := range tree.Attribution.Lineage {
		name := process.Name
		if name == "" {
			name = "process"
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, process.PID))
	}
	return strings.Join(parts, " → ")
}

func LeftoverExplanation(tree model.Tree, observedAt time.Time) string {
	agentStatus := tree.Attribution.AgentStatus
	if agentStatus == "" {
		agentStatus = "not active"
	}
	switch tree.LeftoverAssessment {
	case model.LeftoverLikely:
		return "LIKELY LEFTOVER — the related agent is no longer working, the process is idle, and the observation window has matured."
	case model.LeftoverIntentional:
		return "INTENTIONAL — you marked this exact live process tree to be kept."
	case model.LeftoverAgentActive:
		return fmt.Sprintf("NOT A LEFTOVER — the related agent is currently %s.", agentStatus)
	case model.LeftoverRecentActivity:
		return fmt.Sprintf("NOT YET DETERMINED — the agent is %s, but the process still has recent activity.", agentStatus)
	case model.LeftoverMonitoring:
		if tree.HistoryMaturesAt != nil && observedAt.Before(*tree.HistoryMaturesAt) {
			return fmt.Sprintf("NOT YET DETERMINED — the agent is %s; %s more observation is required before it can become a leftover candidate.", agentStatus, shortDuration(tree.HistoryMaturesAt.Sub(observedAt)))
		}
		return fmt.Sprintf("NOT YET DETERMINED — the agent is %s; Process Guard is waiting for all leftover criteria.", agentStatus)
	default:
		return "UNDETERMINED — the relationship to an agent is not strong enough to call this a leftover."
	}
}

func ActivityExplanation(tree model.Tree, observedAt time.Time) string {
	if tree.Lifecycle == model.LifecycleOrphanCandidate || tree.Lifecycle == model.LifecycleIdle {
		return "IDLE — no qualifying CPU or connection activity was observed during the idle window."
	}
	for _, process := range tree.Processes {
		if process.HasEstablishedConnection() {
			return "ACTIVE — an established network connection is currently open."
		}
	}
	if tree.Attribution.AgentStatus == "working" || tree.Attribution.AgentStatus == "blocked" {
		return "ACTIVE — the related agent is currently " + tree.Attribution.AgentStatus + "."
	}
	if tree.LastActivity != nil {
		reason := tree.LastActivityReason
		if reason == "" {
			reason = "CPU or network activity was observed"
		}
		return fmt.Sprintf("ACTIVE — %s %s ago.", reason, shortDuration(observedAt.Sub(*tree.LastActivity)))
	}
	return "ACTIVE — the process started recently."
}

func shortDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	if value < time.Minute {
		return "less than 1m"
	}
	if value < time.Hour {
		return fmt.Sprintf("%dm", int(value.Round(time.Minute)/time.Minute))
	}
	hours := int(value / time.Hour)
	minutes := int(value.Round(time.Minute)/time.Minute) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func writeNode(output *strings.Builder, pid int, prefix string, last bool, processes map[int]model.Process, children map[int][]int) {
	process, ok := processes[pid]
	if !ok {
		return
	}
	branch := "├─"
	childPrefix := prefix + "│ "
	if last {
		branch = "└─"
		childPrefix = prefix + "  "
	}
	fmt.Fprintf(output, "%s%s %s  pid %d", prefix, branch, process.CommandSummary, pid)
	ports := []string{}
	for _, socket := range process.Sockets {
		if socket.Listen {
			ports = append(ports, socket.Local)
		}
	}
	if len(ports) > 0 {
		fmt.Fprintf(output, "  listen %s", strings.Join(ports, ", "))
	}
	if process.RSSBytes > 0 {
		fmt.Fprintf(output, "  rss %s", Bytes(process.RSSBytes))
	}
	if process.Detached {
		output.WriteString("  detached")
	}
	output.WriteByte('\n')
	items := children[pid]
	for index, child := range items {
		writeNode(output, child, childPrefix, index == len(items)-1, processes, children)
	}
}

func StopPlan(plan guard.StopPlan) string {
	var output strings.Builder
	fmt.Fprintf(&output, "Stop plan %s\n", plan.PlanID)
	fmt.Fprintf(&output, "workspace %s  pane %s  attribution %s\n", plan.Attribution.WorkspaceID, plan.Attribution.PaneID, plan.Attribution.Confidence)
	fmt.Fprintf(&output, "signal %s  affected processes %d\n", plan.Signal, len(plan.Targets))
	for _, target := range plan.Targets {
		fmt.Fprintf(&output, "  %s pid %d (started %s)\n", target.CommandSummary, target.Key.PID, time.UnixMilli(target.Key.StartUnixMS).Format(time.RFC3339))
	}
	if len(plan.PortsReleased) > 0 {
		fmt.Fprintf(&output, "ports released: %s\n", strings.Join(plan.PortsReleased, ", "))
	}
	for _, warning := range plan.Warnings {
		fmt.Fprintf(&output, "warning: %s\n", warning)
	}
	return output.String()
}

func Bytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor, exponent := uint64(unit), 0
	for quotient := value / unit; quotient >= unit; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}
