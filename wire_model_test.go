package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRFC009FixtureRoundTripsAndPreservesStableIDs(t *testing.T) {
	fixtures := lanternWireFixtures()
	if len(fixtures) < 2 || fixtures[0].Version != "1" || fixtures[0].Run.Status != "completed" {
		t.Fatalf("wire fixtures are incomplete: %#v", fixtures)
	}
	encoded, err := json.Marshal(fixtures[0])
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var decoded AuditWireDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	button := decoded.Standards.Components[0]
	if button.CanonicalID != "ui/button" || button.States[0].ID != "button-default" || button.States[0].Checks[0].ID != "chk-button-keyboard" {
		t.Fatalf("stable IDs were not preserved: %#v", button)
	}
	if strings.Contains(string(encoded), "auditQueue") || strings.Contains(string(encoded), "queued") {
		t.Fatalf("transient queue state leaked into wire JSON: %s", encoded)
	}
}

func TestWireOutcomeMappingAndAdapter(t *testing.T) {
	components := adaptAuditComponents(lanternWireFixtures()[0])
	if components[0].Status != StatusFail || components[1].Status != StatusPass || components[2].Status != StatusReview {
		t.Fatalf("component outcome mapping = %q, %q, %q", components[0].Status, components[1].Status, components[2].Status)
	}
	if statusIcon(StatusSkipped) != "-" {
		t.Fatalf("skipped icon = %q, want -", statusIcon(StatusSkipped))
	}
}

func TestStructuredEvidenceRendersByKind(t *testing.T) {
	controller, _, _, _, evidence := newTestController()
	check := controller.currentCheck()
	if check.Evidence[0].Kind != "observation" || check.Evidence[0].Observation == nil {
		t.Fatalf("fixture evidence is not typed: %#v", check.Evidence)
	}
	for _, want := range []string{"observation", "tabIndex", "expectation", "expected", "element", "selector", "source", "line"} {
		if !strings.Contains(evidence.content, want) {
			t.Errorf("evidence rendering missing %q:\n%s", want, evidence.content)
		}
	}
	if !strings.Contains(evidence.content, "EXCLUDED_FROM_FOCUS_ORDER") || !strings.Contains(evidence.content, "Provenance: lantern-rendered-dom") {
		t.Fatalf("typed outcome reason was not rendered:\n%s", evidence.content)
	}
}

func TestWireDTOsContainNoUIRenderingFields(t *testing.T) {
	for _, value := range []any{AuditWireDTO{}, ComponentWireDTO{}, StateWireDTO{}, CheckWireDTO{}, EvidenceWireDTO{}} {
		typeOf := reflect.TypeOf(value)
		for i := 0; i < typeOf.NumField(); i++ {
			name := typeOf.Field(i).Name
			if name == "DisplayText" || name == "Footer" || name == "SelectedIndex" {
				t.Fatalf("wire type %s contains UI field %s", typeOf.Name(), name)
			}
		}
	}
}

func TestRunsAndConfigViewsUseWireFixture(t *testing.T) {
	fixtures := lanternWireFixtures()
	list := NewListPanel("runs", "Runs", "", "", true)
	details := NewTextPanel("details", "Run Details", "", "", true)
	NewRunsController(fixtures, list, details)
	if len(list.items) != len(fixtures) || !strings.Contains(list.items[0].DisplayText, "4 components") || !strings.Contains(list.items[0].DisplayText, "7 checks") {
		t.Fatalf("Runs rows do not use wire summary: %#v", list.items)
	}
	if !strings.Contains(details.content, fixtures[0].Run.ID) || !strings.Contains(details.content, "9800 ms") {
		t.Fatalf("Run details do not use wire run data: %q", details.content)
	}
	config := formatConfig(fixtures[0])
	for _, want := range []string{"wcag-2.2-aa", "lantern/keyboard-access", "error", "warn", "lantern-rendered-dom@1.0.0", "sha256:8b3fdd10a17c", "src/components"} {
		if !strings.Contains(config, want) {
			t.Errorf("Config view missing %q:\n%s", want, config)
		}
	}
}

func TestWireSeverityDomainsRejectInventedValues(t *testing.T) {
	for _, severity := range []CheckSeverity{CheckSeverityWarn, CheckSeverityError} {
		if !severity.Valid() {
			t.Errorf("valid check severity %q was rejected", severity)
		}
	}
	for _, severity := range []CheckSeverity{"off", "serious", "moderate", "warning", "info"} {
		if severity.Valid() {
			t.Errorf("invalid executed-check severity %q was accepted", severity)
		}
	}
	for _, severity := range []ConfigRuleSeverity{ConfigSeverityOff, ConfigSeverityWarn, ConfigSeverityError} {
		if !severity.Valid() {
			t.Errorf("valid config severity %q was rejected", severity)
		}
	}
	for _, severity := range []ConfigRuleSeverity{"serious", "moderate", "warning", "info"} {
		if severity.Valid() {
			t.Errorf("invalid config severity %q was accepted", severity)
		}
	}
	for _, severity := range []DiagnosticSeverity{DiagnosticSeverityInfo, DiagnosticSeverityWarning, DiagnosticSeverityError} {
		if !severity.Valid() {
			t.Errorf("valid diagnostic severity %q was rejected", severity)
		}
	}
	for _, severity := range []DiagnosticSeverity{"warn", "off", "serious", "moderate"} {
		if severity.Valid() {
			t.Errorf("invalid diagnostic severity %q was accepted", severity)
		}
	}
}

func TestFixtureUsesOnlyContractSeverities(t *testing.T) {
	fixture := lanternWireFixtures()[0]
	for _, diagnostic := range fixture.Diagnostics {
		if !diagnostic.Severity.Valid() {
			t.Errorf("invalid diagnostic severity %q", diagnostic.Severity)
		}
	}
	for _, rule := range fixture.Config.Rules {
		if !rule.Severity.Valid() {
			t.Errorf("invalid configured-rule severity %q", rule.Severity)
		}
	}
	for _, component := range fixture.Standards.Components {
		for _, state := range component.States {
			for _, check := range state.Checks {
				if !check.Severity.Valid() {
					t.Errorf("check %s has invalid severity %q", check.ID, check.Severity)
				}
			}
		}
	}
	encoded, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "serious") || strings.Contains(string(encoded), "moderate") {
		t.Fatalf("fixture contains invented impact terminology: %s", encoded)
	}
}
