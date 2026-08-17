package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type AuditStatus = WireOutcome

const (
	StatusPass    = WirePass
	StatusFail    = WireFail
	StatusReview  = WireReview
	StatusSkipped = WireSkipped
)

type EvidenceField struct{ Name, Value string }

type Check struct {
	ID, RuleID    string
	Status        AuditStatus
	Severity      CheckSeverity
	Engine        string
	OutcomeReason OutcomeReasonWireDTO
	Evidence      []EvidenceWireDTO
	Source        SourceWireDTO
	DurationMS    int64
}

type State struct {
	ID     string
	Props  []EvidenceField
	Checks []Check
}
type Component struct {
	CanonicalID, DisplayName, SourceFile string
	Status                               AuditStatus
	States                               []State
}

func adaptAuditComponents(dto AuditWireDTO) []Component {
	engines := map[string]string{}
	for _, engine := range dto.Engines {
		engines[engine.ID] = engine.Name + "@" + engine.Version
	}
	components := make([]Component, 0, len(dto.Standards.Components))
	for _, wireComponent := range dto.Standards.Components {
		component := Component{CanonicalID: wireComponent.CanonicalID, DisplayName: wireComponent.DisplayName, SourceFile: wireComponent.Source.File, Status: StatusPass}
		for _, wireState := range wireComponent.States {
			state := State{ID: wireState.ID, Props: sortedFields(wireState.Props)}
			for _, wireCheck := range wireState.Checks {
				engine := engines[wireCheck.EngineID]
				if engine == "" {
					engine = "unsupported"
				}
				state.Checks = append(state.Checks, Check{ID: wireCheck.ID, RuleID: wireCheck.RuleID, Status: wireCheck.Outcome, Severity: wireCheck.Severity, Engine: engine, OutcomeReason: wireCheck.OutcomeReason, Evidence: wireCheck.Evidence, Source: wireCheck.Source, DurationMS: wireCheck.DurationMS})
			}
			component.States = append(component.States, state)
			component.Status = dominantOutcome(component.Status, stateStatus(&state))
		}
		components = append(components, component)
	}
	return components
}

func fixtureComponents() []Component { return adaptAuditComponents(lanternWireFixtures()[0]) }

func sortedFields(values map[string]string) []EvidenceField {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]EvidenceField, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, EvidenceField{Name: key, Value: values[key]})
	}
	return fields
}

func dominantOutcome(left, right AuditStatus) AuditStatus {
	rank := map[AuditStatus]int{StatusSkipped: 0, StatusPass: 1, StatusReview: 2, StatusFail: 3}
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func statusIcon(status AuditStatus) string {
	switch status {
	case StatusPass:
		return "✓"
	case StatusFail:
		return "✗"
	case StatusReview:
		return "◌"
	default:
		return "-"
	}
}

func formatRunRow(dto AuditWireDTO) string {
	started, _ := time.Parse(time.RFC3339, dto.Run.StartedAt)
	target := dto.Targeting.Root
	return fmt.Sprintf("%s %s  %s  %d components  %d checks  %.1fs", runStatusIcon(dto.Run.Status), started.Format("15:04"), target, dto.Summary.ComponentCount, dto.Summary.CheckCount, float64(dto.Run.DurationMS)/1000)
}

func formatRunDetails(dto AuditWireDTO) string {
	return fmt.Sprintf("Run\n%s\n\nStatus\n%s\n\nStarted\n%s\n\nFinished\n%s\n\nTarget\n%s · %s\n\nSummary\n%d components\n%d states\n%d checks\n%d pass · %d fail · %d review · %d skipped\n\nDuration\n%d ms\n", dto.Run.ID, strings.ToUpper(dto.Run.Status), dto.Run.StartedAt, dto.Run.FinishedAt, dto.Targeting.Mode, dto.Targeting.Root, dto.Summary.ComponentCount, dto.Summary.StateCount, dto.Summary.CheckCount, dto.Summary.PassCount, dto.Summary.FailCount, dto.Summary.ReviewCount, dto.Summary.SkippedCount, dto.Run.DurationMS)
}

func runStatusIcon(status string) string {
	if status == "completed" {
		return "✓"
	}
	if status == "failed" {
		return "✗"
	}
	return "◌"
}

func formatConfig(dto AuditWireDTO) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Snapshot\n%s\n\nFingerprint\n%s\n\nTargeting\n%s · %s\n\nStandards\n", dto.Config.Version, dto.Config.Fingerprint, dto.Targeting.Mode, dto.Targeting.Root)
	for _, standard := range dto.Config.Standards {
		fmt.Fprintf(&b, "- %s\n", standard)
	}
	b.WriteString("\nRules\n")
	for _, rule := range dto.Config.Rules {
		if rule.Enabled {
			fmt.Fprintf(&b, "%-34s %s\n", rule.RuleID, rule.Severity)
		}
	}
	b.WriteString("\nEngines\n")
	for _, engine := range dto.Engines {
		fmt.Fprintf(&b, "%s@%s  (%s)\n", engine.Name, engine.Version, engine.ID)
	}
	return b.String()
}
