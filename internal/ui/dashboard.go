package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/guard"
	"github.com/Efeguclu1/herdr-process-guard/internal/model"
	"github.com/Efeguclu1/herdr-process-guard/internal/presentation"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60"))
	orphanStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	intentionalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	suspiciousStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

type row struct {
	Tree  int
	PID   int
	Depth int
	Last  bool
}
type reportMessage struct{ Report model.Report }
type errorMessage struct{ Err error }
type tickMessage time.Time
type planMessage struct{ Plan guard.StopPlan }
type stopMessage struct {
	Result guard.StopResult
	Err    error
}
type mutationMessage struct {
	Err  error
	Text string
}

type Dashboard struct {
	engine       *guard.Engine
	report       model.Report
	rows         []row
	cursor       int
	width        int
	height       int
	err          error
	message      string
	confirm      *guard.StopPlan
	confirmInput string
}

func Run(engine *guard.Engine) error {
	program := tea.NewProgram(&Dashboard{engine: engine}, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (d *Dashboard) Init() tea.Cmd { return tea.Batch(d.scanCommand(), tickCommand()) }

func tickCommand() tea.Cmd {
	return tea.Tick(5*time.Second, func(value time.Time) tea.Msg { return tickMessage(value) })
}

func (d *Dashboard) scanCommand() tea.Cmd {
	return func() tea.Msg {
		report, err := d.engine.Scan("dashboard")
		if err != nil {
			return errorMessage{Err: err}
		}
		return reportMessage{Report: report}
	}
}

func (d *Dashboard) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch value := message.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = value.Width, value.Height
	case reportMessage:
		d.report, d.err, d.message = value.Report, nil, ""
		d.rows = flatten(value.Report)
		if d.cursor >= len(d.rows) {
			d.cursor = max(0, len(d.rows)-1)
		}
	case errorMessage:
		d.err = value.Err
	case tickMessage:
		if d.confirm == nil {
			return d, tea.Batch(d.scanCommand(), tickCommand())
		}
		return d, tickCommand()
	case planMessage:
		d.confirm = &value.Plan
		d.confirmInput = ""
	case stopMessage:
		d.confirm, d.confirmInput = nil, ""
		if value.Err != nil {
			d.err = value.Err
		} else if len(value.Result.Survivors) > 0 {
			d.message = fmt.Sprintf("%s sent; %d process(es) survived. Press F to review force-stop.", value.Result.Signal, len(value.Result.Survivors))
		} else {
			d.message = value.Result.Signal + " completed; all selected processes exited."
		}
		return d, d.scanCommand()
	case mutationMessage:
		if value.Err != nil {
			d.err = value.Err
		} else {
			d.message = value.Text
		}
		return d, d.scanCommand()
	case tea.KeyMsg:
		if d.confirm != nil {
			return d.updateConfirmation(value)
		}
		switch value.String() {
		case "q", "ctrl+c":
			return d, tea.Quit
		case "j", "down":
			if d.cursor < len(d.rows)-1 {
				d.cursor++
			}
		case "k", "up":
			if d.cursor > 0 {
				d.cursor--
			}
		case "r":
			return d, d.scanCommand()
		case "s":
			return d, d.planCommand(false)
		case "F":
			return d, d.planCommand(true)
		case "i":
			return d, d.markCommand(true)
		case "u":
			return d, d.markCommand(false)
		}
	}
	return d, nil
}

func (d *Dashboard) selectedPID() (int, bool) {
	if d.cursor < 0 || d.cursor >= len(d.rows) {
		return 0, false
	}
	return d.rows[d.cursor].PID, true
}

func (d *Dashboard) planCommand(force bool) tea.Cmd {
	pid, ok := d.selectedPID()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		plan, err := d.engine.PlanStop(pid, force)
		if err != nil {
			return errorMessage{Err: err}
		}
		return planMessage{Plan: plan}
	}
}

func (d *Dashboard) markCommand(mark bool) tea.Cmd {
	pid, ok := d.selectedPID()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		if mark {
			_, err := d.engine.MarkIntentional(pid)
			return mutationMessage{Err: err, Text: "Exact live tree marked intentional."}
		}
		return mutationMessage{Err: d.engine.UnmarkIntentional(pid), Text: "Intentional mark removed."}
	}
}

func (d *Dashboard) updateConfirmation(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	required := fmt.Sprintf("stop %d", d.confirm.SelectedPID)
	if d.confirm.Signal == "SIGKILL" {
		required = fmt.Sprintf("force %d", d.confirm.SelectedPID)
	}
	switch key.String() {
	case "esc", "ctrl+c":
		d.confirm, d.confirmInput = nil, ""
		return d, nil
	case "backspace":
		if len(d.confirmInput) > 0 {
			d.confirmInput = d.confirmInput[:len(d.confirmInput)-1]
		}
	case "enter":
		if d.confirmInput != required {
			d.err = fmt.Errorf("confirmation did not match %q", required)
			return d, nil
		}
		plan := *d.confirm
		return d, func() tea.Msg {
			result, err := d.engine.ExecuteStop(plan)
			return stopMessage{Result: result, Err: err}
		}
	default:
		if len(key.Runes) == 1 && key.Runes[0] >= 32 {
			d.confirmInput += string(key.Runes[0])
		}
	}
	return d, nil
}

func flatten(report model.Report) []row {
	rows := []row{}
	for treeIndex, tree := range report.Trees {
		children := map[int][]int{}
		for _, process := range tree.Processes {
			children[process.PPID] = append(children[process.PPID], process.Key.PID)
		}
		for parent := range children {
			sort.Ints(children[parent])
		}
		var visit func(int, int, bool)
		visit = func(pid, depth int, last bool) {
			rows = append(rows, row{Tree: treeIndex, PID: pid, Depth: depth, Last: last})
			for index, child := range children[pid] {
				visit(child, depth+1, index == len(children[pid])-1)
			}
		}
		visit(tree.RootPID, 0, true)
	}
	return rows
}

func (d *Dashboard) View() string {
	if d.confirm != nil {
		return d.confirmationView()
	}
	var output strings.Builder
	output.WriteString(titleStyle.Render("HERDR / PROCESS GUARD"))
	output.WriteString("\n")
	fmt.Fprintf(&output, "workloads %d   processes %d   active %d   idle %d   orphan %d   suspicious %d\n",
		d.report.Summary.Trees, d.report.Summary.Processes, d.report.Summary.Active, d.report.Summary.Idle,
		d.report.Summary.OrphanCandidates, d.report.Summary.Suspicious)
	if d.err != nil {
		output.WriteString(errorStyle.Render("error: " + d.err.Error()))
		output.WriteByte('\n')
	}
	if d.message != "" {
		output.WriteString(intentionalStyle.Render(d.message))
		output.WriteByte('\n')
	}
	if len(d.rows) == 0 && d.err == nil {
		output.WriteString("\nNo attributed long-running workloads found.\n")
	}
	available := max(3, d.height-17)
	start := max(0, d.cursor-available/2)
	end := min(len(d.rows), start+available)
	if end-start < available {
		start = max(0, end-available)
	}
	lastTree := -1
	for index := start; index < end; index++ {
		item := d.rows[index]
		tree := d.report.Trees[item.Tree]
		if item.Tree != lastTree {
			status := fmt.Sprintf("%s   RELATED: %s · LEFTOVER: %s", presentation.PaneLocation(tree.Attribution),
				presentation.RelationshipBadge(tree), presentation.LeftoverBadge(tree))
			switch tree.Policy {
			case model.PolicyIntentional:
				status = intentionalStyle.Render(status)
			case model.PolicySuspicious:
				status = suspiciousStyle.Render(status)
			}
			if tree.Lifecycle == model.LifecycleOrphanCandidate {
				status = orphanStyle.Render(status)
			}
			output.WriteString("\n" + status + "\n")
			lastTree = item.Tree
		}
		process := tree.Process(item.PID)
		if process == nil {
			continue
		}
		prefix := strings.Repeat("  ", item.Depth)
		line := fmt.Sprintf("%s%s  pid %-6d  %-20s", prefix, branch(item.Last), process.Key.PID, truncate(process.CommandSummary, 20))
		ports := []string{}
		for _, socket := range process.Sockets {
			if socket.Listen {
				ports = append(ports, socket.Local)
			}
		}
		if len(ports) > 0 {
			line += "  " + strings.Join(ports, ",")
		}
		if process.RSSBytes > 0 {
			line += "  " + presentation.Bytes(process.RSSBytes)
		}
		if index == d.cursor {
			line = selectedStyle.Render(fit(line, d.width))
		}
		output.WriteString(line + "\n")
	}
	if pid, ok := d.selectedPID(); ok {
		tree := d.report.Trees[d.rows[d.cursor].Tree]
		if process := tree.Process(pid); process != nil {
			output.WriteByte('\n')
			output.WriteString(detailLine("source pane: "+presentation.PaneDetail(tree.Attribution), d.width))
			output.WriteString(detailLine("relationship: "+presentation.RelationshipExplanation(tree), d.width))
			if lineage := presentation.Lineage(tree); lineage != "" {
				output.WriteString(detailLine("lineage: "+lineage, d.width))
			}
			output.WriteString(detailLine("leftover: "+presentation.LeftoverExplanation(tree, d.report.ObservedAt), d.width))
			output.WriteString(detailLine("activity: "+presentation.ActivityExplanation(tree, d.report.ObservedAt), d.width))
			if process.CWD != "" {
				output.WriteString(detailLine("process cwd: "+process.CWD, d.width))
			}
			output.WriteString(detailLine("command: "+process.Command, d.width))
		}
	}
	output.WriteString(dimStyle.Render("j/k move  r refresh  s graceful stop  F force stop  i intentional  u unmark  q close"))
	return output.String()
}

func detailLine(value string, width int) string {
	return dimStyle.Render(truncate(value, max(1, width))) + "\n"
}

func (d *Dashboard) confirmationView() string {
	var output strings.Builder
	output.WriteString(titleStyle.Render("PROCESS TERMINATION PREVIEW") + "\n\n")
	output.WriteString(presentation.StopPlan(*d.confirm) + "\n")
	required := fmt.Sprintf("stop %d", d.confirm.SelectedPID)
	if d.confirm.Signal == "SIGKILL" {
		required = fmt.Sprintf("force %d", d.confirm.SelectedPID)
	}
	output.WriteString(orphanStyle.Render("No automatic escalation or rollback is possible.") + "\n")
	fmt.Fprintf(&output, "Type %q to continue, or Esc to cancel:\n> %s", required, d.confirmInput)
	if d.err != nil {
		output.WriteString("\n" + errorStyle.Render(d.err.Error()))
	}
	return output.String()
}

func branch(last bool) string {
	if last {
		return "└─"
	}
	return "├─"
}
func truncate(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width < 2 {
		return value[:width]
	}
	return value[:width-1] + "…"
}
func fit(value string, width int) string {
	if width <= 0 {
		return value
	}
	if len(value) > width {
		return value[:width]
	}
	return value + strings.Repeat(" ", width-len(value))
}
