package main

func lanternWireFixtures() []AuditWireDTO {
	engine := EngineWireDTO{ID: "lantern-rendered-dom", Name: "lantern-rendered-dom", Version: "1.0.0"}
	latest := AuditWireDTO{
		Version:   "1",
		Run:       RunWireDTO{ID: "run-2026-08-17T174200Z", StartedAt: "2026-08-17T17:42:00+02:00", FinishedAt: "2026-08-17T17:42:09.800+02:00", Status: "completed", DurationMS: 9800},
		Targeting: TargetingWireDTO{Mode: "source-root", Root: "src/components"},
		Engines:   []EngineWireDTO{engine},
		Config: ConfigWireDTO{
			Version: "lantern.config.v1", Fingerprint: "sha256:8b3fdd10a17c",
			Standards: []string{"wcag-2.2-aa", "aria-apg"},
			Rules: []ConfiguredRuleWireDTO{
				{RuleID: "lantern/keyboard-access", Severity: ConfigSeverityError, Enabled: true},
				{RuleID: "lantern/accessible-name", Severity: ConfigSeverityError, Enabled: true},
				{RuleID: "lantern/focus-visible", Severity: ConfigSeverityWarn, Enabled: true},
				{RuleID: "lantern/grid-navigation", Severity: ConfigSeverityWarn, Enabled: true},
			},
		},
		Diagnostics: []DiagnosticWireDTO{{Severity: DiagnosticSeverityInfo, Code: "TARGET_DISCOVERY_COMPLETE", Message: "Four canonical components discovered"}},
		Standards: StandardsWireDTO{Components: []ComponentWireDTO{
			{CanonicalID: "ui/button", DisplayName: "Button", Source: source("src/components/ui/button.tsx", 12), States: []StateWireDTO{
				{ID: "button-default", Props: props("size", "default", "variant", "default"), Checks: []CheckWireDTO{
					wireCheck("chk-button-keyboard", "lantern/keyboard-access", WireFail, CheckSeverityError, "EXCLUDED_FROM_FOCUS_ORDER", "excluded from the keyboard focus order", observation(map[string]string{"tabIndex": "-1", "disabled": "false", "hidden": "false", "inert": "false"}), expectation("participates in sequential keyboard focus order", "excluded"), element("button", `<button tabindex="-1">Save changes</button>`), sourceEvidence("src/components/ui/button.tsx", 24)),
					wireCheck("chk-button-name", "lantern/accessible-name", WirePass, CheckSeverityError, "ACCESSIBLE_NAME_PRESENT", "visible text supplies an accessible name", observation(map[string]string{"role": "button", "name": "Save changes"}), sourceEvidence("src/components/ui/button.tsx", 24)),
				}},
				{ID: "button-icon", Props: props("size", "icon", "variant", "default"), Checks: []CheckWireDTO{
					wireCheck("chk-icon-name", "lantern/accessible-name", WireReview, CheckSeverityError, "AUTHOR_CONFIRMATION_REQUIRED", "icon-only control needs author confirmation", observation(map[string]string{"role": "button", "aria-label": "Open menu"}), element("button", `<button aria-label="Open menu">…</button>`), sourceEvidence("src/components/ui/button.tsx", 41)),
					wireCheck("chk-icon-focus", "lantern/focus-visible", WirePass, CheckSeverityWarn, "FOCUS_INDICATOR_PRESENT", "focus indicator is rendered", observation(map[string]string{"outlineWidth": "2px", "contrastRatio": "4.8:1"}), expectation("a visible keyboard focus indicator", "2px outline"), sourceEvidence("src/components/ui/button.tsx", 41)),
				}},
				{ID: "button-disabled", Props: props("disabled", "true", "size", "default", "variant", "secondary"), Checks: []CheckWireDTO{
					wireCheck("chk-disabled-semantics", "lantern/disabled-semantics", WirePass, CheckSeverityWarn, "NATIVE_DISABLED_EXPOSED", "native disabled state is exposed", observation(map[string]string{"disabled": "true", "aria-disabled": "not set"}), sourceEvidence("src/components/ui/button.tsx", 56)),
				}},
			}},
			{CanonicalID: "ui/label", DisplayName: "Label", Source: source("src/components/ui/label.tsx", 8), States: []StateWireDTO{
				{ID: "label-default", Props: props("required", "false", "tone", "default"), Checks: []CheckWireDTO{
					wireCheck("chk-label-association", "lantern/label-association", WirePass, CheckSeverityError, "LABEL_CONTROL_MATCH", "label references the form control", observation(map[string]string{"htmlFor": "email", "controlId": "email"}), expectation("label and control identifiers match", "email → email"), sourceEvidence("src/components/ui/label.tsx", 17)),
				}},
			}},
			{CanonicalID: "ui/calendar", DisplayName: "Calendar", Source: source("src/components/ui/calendar.tsx", 16), States: []StateWireDTO{
				{ID: "calendar-single", Props: props("mode", "single", "month", "August 2026"), Checks: []CheckWireDTO{
					wireCheck("chk-calendar-grid", "lantern/grid-navigation", WireReview, CheckSeverityWarn, "INTERACTION_REVIEW_REQUIRED", "roving focus behavior requires review", observation(map[string]string{"role": "grid", "tabStops": "1", "arrowNavigation": "true"}), expectation("one tab stop with arrow-key navigation", "one active tab stop"), sourceEvidence("src/components/ui/calendar.tsx", 73)),
				}},
				{ID: "calendar-empty", Props: props("mode", "range", "disabledDays", "all"), Checks: []CheckWireDTO{}},
			}},
			{CanonicalID: "ui/separator", DisplayName: "Separator", Source: source("src/components/ui/separator.tsx", 5), States: []StateWireDTO{}},
		}},
		Summary: SummaryWireDTO{ComponentCount: 4, StateCount: 6, CheckCount: 7, PassCount: 4, FailCount: 1, ReviewCount: 2, SkippedCount: 0},
	}
	previous := latest
	previous.Run = RunWireDTO{ID: "run-2026-08-16T091500Z", StartedAt: "2026-08-16T09:15:00+02:00", FinishedAt: "2026-08-16T09:15:07.400+02:00", Status: "completed", DurationMS: 7400}
	return []AuditWireDTO{latest, previous}
}

func props(values ...string) map[string]string {
	result := map[string]string{}
	for i := 0; i+1 < len(values); i += 2 {
		result[values[i]] = values[i+1]
	}
	return result
}
func source(file string, line int) SourceWireDTO {
	return SourceWireDTO{File: file, Line: line, Column: 1}
}
func observation(facts map[string]string) EvidenceWireDTO {
	return EvidenceWireDTO{Kind: "observation", Observation: &ObservationEvidenceDTO{Facts: facts}}
}
func expectation(expected, observed string) EvidenceWireDTO {
	return EvidenceWireDTO{Kind: "expectation", Expectation: &ExpectationEvidenceDTO{Expected: expected, Observed: observed}}
}
func element(selector, html string) EvidenceWireDTO {
	return EvidenceWireDTO{Kind: "element", Element: &ElementEvidenceDTO{Selector: selector, HTML: html}}
}
func sourceEvidence(file string, line int) EvidenceWireDTO {
	value := source(file, line)
	return EvidenceWireDTO{Kind: "source", Source: &value}
}

func wireCheck(id, ruleID string, outcome WireOutcome, severity CheckSeverity, code, detail string, evidence ...EvidenceWireDTO) CheckWireDTO {
	checkSource := SourceWireDTO{}
	for _, record := range evidence {
		if record.Kind == "source" && record.Source != nil {
			checkSource = *record.Source
		}
	}
	return CheckWireDTO{ID: id, RuleID: ruleID, Outcome: outcome, Severity: severity, EngineID: "lantern-rendered-dom", OutcomeReason: OutcomeReasonWireDTO{Code: code, Detail: detail, Provenance: "lantern-rendered-dom"}, Evidence: evidence, Source: checkSource, DurationMS: 12}
}
