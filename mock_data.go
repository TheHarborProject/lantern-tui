package main

func mockComponents() []Component {
	return []Component{
		{
			CanonicalID: "ui/button", DisplayName: "Button", SourceFile: "src/components/ui/button.tsx", Status: StatusFail,
			States: []State{
				{ID: "button-default", Props: fields("size", "default", "variant", "default"), Checks: []Check{
					check("chk-button-keyboard", "lantern/keyboard-access", StatusFail, "excluded from the keyboard focus order", fields("tabIndex", "-1", "disabled", "false", "hidden", "false", "inert", "false"), "participates in sequential keyboard focus order", "excluded", "src/components/ui/button.tsx"),
					check("chk-button-name", "lantern/accessible-name", StatusPass, "visible text supplies an accessible name", fields("role", "button", "name", "Save changes"), "a non-empty accessible name", "Save changes", "src/components/ui/button.tsx"),
				}},
				{ID: "button-icon", Props: fields("size", "icon", "variant", "default"), Checks: []Check{
					check("chk-icon-name", "lantern/accessible-name", StatusReview, "icon-only control needs author confirmation", fields("role", "button", "aria-label", "Open menu"), "an intentional accessible name", "Open menu", "src/components/ui/button.tsx"),
					check("chk-icon-focus", "lantern/focus-visible", StatusPass, "focus indicator is rendered", fields("outlineWidth", "2px", "contrastRatio", "4.8:1"), "a visible keyboard focus indicator", "2px outline", "src/components/ui/button.tsx"),
				}},
				{ID: "button-disabled", Props: fields("size", "default", "variant", "secondary", "disabled", "true"), Checks: []Check{
					check("chk-disabled-semantics", "lantern/disabled-semantics", StatusPass, "native disabled state is exposed", fields("disabled", "true", "aria-disabled", "not set"), "disabled semantics exposed to assistive technology", "native disabled attribute", "src/components/ui/button.tsx"),
				}},
			},
		},
		{
			CanonicalID: "ui/label", DisplayName: "Label", SourceFile: "src/components/ui/label.tsx", Status: StatusPass,
			States: []State{{ID: "label-default", Props: fields("required", "false", "tone", "default"), Checks: []Check{
				check("chk-label-association", "lantern/label-association", StatusPass, "label references the form control", fields("htmlFor", "email", "controlId", "email"), "label and control identifiers match", "email → email", "src/components/ui/label.tsx"),
			}}},
		},
		{
			CanonicalID: "ui/calendar", DisplayName: "Calendar", SourceFile: "src/components/ui/calendar.tsx", Status: StatusReview,
			States: []State{
				{ID: "calendar-single", Props: fields("mode", "single", "month", "August 2026"), Checks: []Check{
					check("chk-calendar-grid", "lantern/grid-navigation", StatusReview, "roving focus behavior requires review", fields("role", "grid", "tabStops", "1", "arrowNavigation", "true"), "one tab stop with arrow-key navigation", "one active tab stop", "src/components/ui/calendar.tsx"),
				}},
				{ID: "calendar-empty", Props: fields("mode", "range", "disabledDays", "all"), Checks: nil},
			},
		},
		{CanonicalID: "ui/separator", DisplayName: "Separator", SourceFile: "src/components/ui/separator.tsx", Status: StatusPass, States: nil},
	}
}

func fields(values ...string) []EvidenceField {
	result := make([]EvidenceField, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		result = append(result, EvidenceField{Name: values[i], Value: values[i+1]})
	}
	return result
}

func check(id, ruleID string, status AuditStatus, reason string, evidence []EvidenceField, expected, observed, source string) Check {
	severity := "info"
	if status == StatusFail {
		severity = "serious"
	} else if status == StatusReview {
		severity = "moderate"
	}
	return Check{ID: id, RuleID: ruleID, Status: status, Severity: severity, Engine: "lantern-rendered-dom@1.0.0", OutcomeReason: reason, Evidence: evidence, Expected: expected, Observed: observed, Source: source}
}
