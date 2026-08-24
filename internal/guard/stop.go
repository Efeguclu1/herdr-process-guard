package guard

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"syscall"
	"time"

	"github.com/Efeguclu1/herdr-process-guard/internal/model"
)

type StopPlan struct {
	PlanID        string            `json:"plan_id"`
	CreatedAt     time.Time         `json:"created_at"`
	Signal        string            `json:"signal"`
	SelectedPID   int               `json:"selected_pid"`
	Root          model.ProcessKey  `json:"root"`
	Targets       []StopTarget      `json:"targets"`
	Attribution   model.Attribution `json:"attribution"`
	PortsReleased []string          `json:"ports_released,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

type StopTarget struct {
	Key              model.ProcessKey `json:"key"`
	PPID             int              `json:"ppid"`
	SessionID        int              `json:"session_id"`
	Depth            int              `json:"depth"`
	Name             string           `json:"name"`
	Executable       string           `json:"executable,omitempty"`
	CommandSummary   string           `json:"command_summary"`
	CommandHash      string           `json:"command_hash"`
	Protected        bool             `json:"protected"`
	ProtectionReason string           `json:"protection_reason,omitempty"`
}

type StopResult struct {
	PlanID    string             `json:"plan_id"`
	Signal    string             `json:"signal"`
	Signaled  []model.ProcessKey `json:"signaled"`
	Survivors []model.ProcessKey `json:"survivors,omitempty"`
}

func (e *Engine) PlanStop(pid int, force bool) (StopPlan, error) {
	report, err := e.Scan("stop-preview")
	if err != nil {
		return StopPlan{}, err
	}
	tree, err := e.FindTree(report, pid)
	if err != nil {
		return StopPlan{}, err
	}
	selected := tree.Process(pid)
	if selected == nil {
		return StopPlan{}, fmt.Errorf("process %d disappeared", pid)
	}
	processes := map[int]model.Process{}
	for _, process := range tree.Processes {
		processes[process.Key.PID] = process
	}
	selectedProcesses := subtree(processes, pid)
	if len(selectedProcesses) == 0 {
		return StopPlan{}, errors.New("selected process tree is empty")
	}
	depths := processDepths(selectedProcesses, pid)
	targets := make([]StopTarget, 0, len(selectedProcesses))
	ports := []string{}
	for _, process := range selectedProcesses {
		if process.Protected || process.Infrastructure {
			return StopPlan{}, fmt.Errorf("refusing to stop protected process %d (%s): %s", process.Key.PID, process.Name, process.ProtectionReason)
		}
		targets = append(targets, StopTarget{Key: process.Key, PPID: process.PPID, SessionID: process.SessionID,
			Depth: depths[process.Key.PID], Name: process.Name, Executable: process.Executable,
			CommandSummary: process.CommandSummary, CommandHash: process.CommandHash,
			Protected: process.Protected, ProtectionReason: process.ProtectionReason})
		ports = append(ports, listeningPorts(process)...)
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Depth != targets[j].Depth {
			return targets[i].Depth < targets[j].Depth
		}
		return targets[i].Key.PID < targets[j].Key.PID
	})
	sort.Strings(ports)
	signal := "SIGTERM"
	if force {
		signal = "SIGKILL"
	}
	plan := StopPlan{CreatedAt: e.Now().UTC(), Signal: signal, SelectedPID: pid, Root: selected.Key, Targets: targets,
		Attribution: tree.Attribution, PortsReleased: uniqueStrings(ports)}
	plan.PlanID = hashPlan(plan)
	if tree.Attribution.Confidence == model.ConfidenceLow {
		plan.Warnings = append(plan.Warnings, "attribution confidence is low")
	}
	if force {
		state, err := e.Store.Load()
		if err != nil {
			return StopPlan{}, err
		}
		attempt, ok := state.TerminationAttempts[selected.Key.String()]
		if !ok || attemptExpired(e.Now().UTC(), attempt.AttemptedAt) {
			return StopPlan{}, errors.New("force-stop is allowed only after these exact processes survive a recent Process Guard SIGTERM")
		}
		if !sameAttemptTargets(attempt.Survivors, targets) {
			return StopPlan{}, errors.New("survivor identities changed since SIGTERM; gracefully stop the new tree first")
		}
	}
	return plan, nil
}

func attemptExpired(now, attempted time.Time) bool {
	return now.Before(attempted) || now.Sub(attempted) > 10*time.Minute
}

func sameAttemptTargets(left []model.ProcessIdentity, right []StopTarget) bool {
	if len(left) != len(right) {
		return false
	}
	set := map[string]model.ProcessIdentity{}
	for _, identity := range left {
		set[identity.Key.String()] = identity
	}
	for _, target := range right {
		identity, ok := set[target.Key.String()]
		if !ok || identity.SessionID != target.SessionID || identity.Executable != target.Executable || identity.CommandHash != target.CommandHash {
			return false
		}
	}
	return true
}

func processDepths(processes []model.Process, root int) map[int]int {
	byParent := map[int][]int{}
	for _, process := range processes {
		byParent[process.PPID] = append(byParent[process.PPID], process.Key.PID)
	}
	depths := map[int]int{root: 0}
	queue := []int{root}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, child := range byParent[pid] {
			depths[child] = depths[pid] + 1
			queue = append(queue, child)
		}
	}
	return depths
}

func hashPlan(plan StopPlan) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s:%d:%d:%s", plan.Signal, plan.Root.PID, plan.Root.StartUnixMS, plan.CreatedAt.Format(time.RFC3339Nano))
	for _, target := range plan.Targets {
		fmt.Fprintf(hash, ":%s", target.Key.String())
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := []string{values[0]}
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func (e *Engine) ExecuteStop(plan StopPlan) (StopResult, error) {
	force := plan.Signal == "SIGKILL"
	fresh, err := e.PlanStop(plan.SelectedPID, force)
	if err != nil {
		return StopResult{}, err
	}
	if !sameTargetSet(plan.Targets, fresh.Targets) || plan.Root != fresh.Root {
		return StopResult{}, errors.New("process tree changed after preview; review a new stop plan")
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	ordered := append([]StopTarget(nil), fresh.Targets...)
	// Signal the selected root first so wrappers begin shutdown, then descendants deepest-first.
	sort.SliceStable(ordered[1:], func(i, j int) bool { return ordered[i+1].Depth > ordered[j+1].Depth })
	result := StopResult{PlanID: plan.PlanID, Signal: plan.Signal}
	for _, target := range ordered {
		if err := syscall.Kill(target.Key.PID, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
			return result, fmt.Errorf("signal pid %d: %w", target.Key.PID, err)
		}
		result.Signaled = append(result.Signaled, target.Key)
	}
	deadline := time.Now().Add(e.Config.TerminateGrace)
	for time.Now().Before(deadline) {
		if survivors, _ := e.liveTargets(fresh.Targets); len(survivors) == 0 {
			e.recordTerminationAttempt(fresh, nil)
			return result, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	result.Survivors, _ = e.liveTargets(fresh.Targets)
	e.recordTerminationAttempt(fresh, result.Survivors)
	return result, nil
}

func (e *Engine) recordTerminationAttempt(plan StopPlan, survivors []model.ProcessKey) {
	_, _ = e.Store.Update(func(state *model.PersistedState) error {
		key := plan.Root.String()
		if plan.Signal == "SIGTERM" && len(survivors) > 0 {
			survivorSet := map[string]bool{}
			for _, survivor := range survivors {
				survivorSet[survivor.String()] = true
			}
			identities := []model.ProcessIdentity{}
			for _, target := range plan.Targets {
				if survivorSet[target.Key.String()] {
					identities = append(identities, model.ProcessIdentity{Key: target.Key, SessionID: target.SessionID, Executable: target.Executable, CommandHash: target.CommandHash})
				}
			}
			state.TerminationAttempts[key] = model.TerminationAttempt{Selected: plan.Root, Survivors: identities, AttemptedAt: e.Now().UTC()}
		} else {
			delete(state.TerminationAttempts, key)
		}
		return nil
	})
}

func sameTargetSet(left, right []StopTarget) bool {
	if len(left) != len(right) {
		return false
	}
	keys := map[string]StopTarget{}
	for _, target := range left {
		keys[target.Key.String()] = target
	}
	for _, target := range right {
		original, ok := keys[target.Key.String()]
		if !ok || original.CommandHash != target.CommandHash || original.Executable != target.Executable || original.SessionID != target.SessionID || original.PPID != target.PPID || original.Depth != target.Depth {
			return false
		}
	}
	return true
}

func (e *Engine) liveTargets(targets []StopTarget) ([]model.ProcessKey, error) {
	processes, _, err := e.Scanner.Processes()
	if err != nil {
		return nil, err
	}
	survivors := []model.ProcessKey{}
	for _, target := range targets {
		if process, ok := processes[target.Key.PID]; ok && process.Key == target.Key {
			survivors = append(survivors, target.Key)
		}
	}
	return survivors, nil
}
