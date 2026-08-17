package main

import (
	"fmt"
	"strings"
)

type AuditController struct {
	components   []Component
	auditQueue   map[string]bool
	queueChanged []func()

	selectedComponent int
	selectedState     int
	selectedCheck     int

	componentsPanel *ListPanel
	statesPanel     *ListPanel
	checksPanel     *ListPanel
	evidencePanel   *TextPanel
}

func NewAuditController(components []Component, componentsPanel, statesPanel, checksPanel *ListPanel, evidencePanel *TextPanel) *AuditController {
	c := &AuditController{
		components:      components,
		auditQueue:      map[string]bool{"ui/button": true, "ui/calendar": true},
		componentsPanel: componentsPanel, statesPanel: statesPanel,
		checksPanel: checksPanel, evidencePanel: evidencePanel,
	}
	componentsPanel.SetEmptyMessage("No components found")
	statesPanel.SetEmptyMessage("No states for this component")
	checksPanel.SetEmptyMessage("No checks for this state")
	componentsPanel.SetOnSelectionChanged(c.selectComponent)
	statesPanel.SetOnSelectionChanged(c.selectState)
	checksPanel.SetOnSelectionChanged(c.selectCheck)
	c.refreshComponents()
	c.refreshStates()
	return c
}

func (c *AuditController) FocusModeCapabilities() []FocusModeCapability {
	return []FocusModeCapability{
		{PanelID: c.componentsPanel.ID(), StatusHints: "q: quit | Esc: back | ↑/↓: navigate | Space: toggle", OnEnter: c.refreshFocusedComponents, OnExit: c.refreshComponents, OnKey: c.handleComponentsFocusKey},
		{PanelID: c.statesPanel.ID(), StatusHints: "q: quit | Esc: back | ↑/↓: navigate", OnEnter: c.refreshFocusedStates, OnExit: c.refreshStates},
		{PanelID: c.checksPanel.ID(), StatusHints: "q: quit | Esc: back | ↑/↓: navigate", OnEnter: c.refreshFocusedChecks, OnExit: c.refreshChecks},
		{PanelID: c.evidencePanel.ID(), StatusHints: "q: quit | Esc: back | ↑/↓: scroll"},
	}
}

func (c *AuditController) refreshFocusedComponents() {
	items := make([]ListItem[any], 0, len(c.components))
	for i := range c.components {
		component := &c.components[i]
		marker := "[ ]"
		if c.auditQueue[component.CanonicalID] {
			marker = "[x]"
		}
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%s %s  %s  %s", marker, component.DisplayName, component.CanonicalID, component.SourceFile), Object: component})
	}
	c.componentsPanel.SetItems(items)
	c.componentsPanel.SetSelectedIndex(c.selectedComponent)
}

func (c *AuditController) handleComponentsFocusKey(key interface{}) bool {
	if key != ' ' {
		return false
	}
	component := c.currentComponent()
	if component == nil {
		return true
	}
	c.ToggleQueued(component.CanonicalID)
	c.refreshFocusedComponents()
	return true
}

func (c *AuditController) Components() []Component {
	return c.components
}

func (c *AuditController) IsQueued(componentID string) bool {
	return c.auditQueue[componentID]
}

func (c *AuditController) ToggleQueued(componentID string) bool {
	for i := range c.components {
		if c.components[i].CanonicalID == componentID {
			c.auditQueue[componentID] = !c.auditQueue[componentID]
			c.notifyQueueChanged()
			return true
		}
	}
	return false
}

func (c *AuditController) SelectAllQueued() {
	for i := range c.components {
		c.auditQueue[c.components[i].CanonicalID] = true
	}
	c.notifyQueueChanged()
}

func (c *AuditController) SelectNoQueued() {
	for i := range c.components {
		c.auditQueue[c.components[i].CanonicalID] = false
	}
	c.notifyQueueChanged()
}

func (c *AuditController) QueuedComponents() []Component {
	var queued []Component
	for i := range c.components {
		if c.auditQueue[c.components[i].CanonicalID] {
			queued = append(queued, c.components[i])
		}
	}
	return queued
}

func (c *AuditController) OnQueueChanged(handler func()) {
	c.queueChanged = append(c.queueChanged, handler)
}

func (c *AuditController) notifyQueueChanged() {
	for _, handler := range c.queueChanged {
		handler()
	}
}

func (c *AuditController) SelectComponentByCanonicalID(componentID string) bool {
	for i := range c.components {
		if c.components[i].CanonicalID != componentID {
			continue
		}
		c.componentsPanel.SetSelectedIndex(i)
		// SetSelectedIndex intentionally does not notify when the same row is
		// selected, so refresh explicitly in that case.
		if c.selectedComponent == i {
			c.refreshStates()
		}
		return true
	}
	return false
}

func (c *AuditController) refreshFocusedStates() {
	component := c.currentComponent()
	if component == nil || len(component.States) == 0 {
		c.statesPanel.SetItems(nil)
		return
	}
	items := make([]ListItem[any], 0, len(component.States))
	for i := range component.States {
		state := &component.States[i]
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%s %s  %s  %s", statusIcon(stateStatus(state)), stateLabel(state), state.ID, checkSummary(state)), Object: state})
	}
	c.statesPanel.SetItems(items)
	c.statesPanel.SetSelectedIndex(c.selectedState)
}

func (c *AuditController) refreshFocusedChecks() {
	state := c.currentState()
	if state == nil || len(state.Checks) == 0 {
		c.checksPanel.SetItems(nil)
		return
	}
	items := make([]ListItem[any], 0, len(state.Checks))
	for i := range state.Checks {
		check := &state.Checks[i]
		engine := check.Engine
		if engine == "" {
			engine = "unsupported"
		}
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%s %s  %s  %s  %s", statusIcon(check.Status), check.RuleID, check.Severity, engine, check.OutcomeReason), Object: check})
	}
	c.checksPanel.SetItems(items)
	c.checksPanel.SetSelectedIndex(c.selectedCheck)
}

func (c *AuditController) selectComponent(index int) {
	c.selectedComponent = index
	c.selectedState = 0
	c.selectedCheck = 0
	c.refreshStates()
}

func (c *AuditController) selectState(index int) {
	c.selectedState = index
	c.selectedCheck = 0
	c.refreshChecks()
}

func (c *AuditController) selectCheck(index int) {
	c.selectedCheck = index
	c.refreshEvidence()
}

func (c *AuditController) refreshComponents() {
	items := make([]ListItem[any], 0, len(c.components))
	for i := range c.components {
		component := &c.components[i]
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%s %s", statusIcon(component.Status), component.DisplayName), Object: component})
	}
	c.componentsPanel.SetItems(items)
}

func (c *AuditController) refreshStates() {
	component := c.currentComponent()
	if component == nil || len(component.States) == 0 {
		c.statesPanel.SetItems(nil)
		c.checksPanel.SetItems(nil)
		c.evidencePanel.SetContent("No evidence: select a state with checks.\n")
		return
	}
	c.selectedState = clamp(c.selectedState, len(component.States))
	items := make([]ListItem[any], 0, len(component.States))
	for i := range component.States {
		state := &component.States[i]
		items = append(items, ListItem[any]{DisplayText: stateLabel(state), Object: state})
	}
	c.statesPanel.SetItems(items)
	c.statesPanel.SetSelectedIndex(c.selectedState)
	c.refreshChecks()
}

func (c *AuditController) refreshChecks() {
	state := c.currentState()
	if state == nil || len(state.Checks) == 0 {
		c.checksPanel.SetItems(nil)
		c.evidencePanel.SetContent("No evidence: this state has no checks.\n")
		return
	}
	c.selectedCheck = clamp(c.selectedCheck, len(state.Checks))
	items := make([]ListItem[any], 0, len(state.Checks))
	for i := range state.Checks {
		check := &state.Checks[i]
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%s %s", statusIcon(check.Status), shortRuleID(check.RuleID)), Object: check})
	}
	c.checksPanel.SetItems(items)
	c.checksPanel.SetSelectedIndex(c.selectedCheck)
	c.refreshEvidence()
}

func (c *AuditController) refreshEvidence() {
	state, check := c.currentState(), c.currentCheck()
	if state == nil || check == nil {
		c.evidencePanel.SetContent("No evidence available.\n")
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Rule\n%s\n\nStatus\n%s\n\nEngine\n%s\n\nState\n%s\n\nOutcome\n%s\n\nEvidence\n", check.RuleID, strings.ToUpper(string(check.Status)), check.Engine, stateLabel(state), check.OutcomeReason)
	for _, field := range check.Evidence {
		fmt.Fprintf(&b, "%-12s %s\n", field.Name, field.Value)
	}
	fmt.Fprintf(&b, "\nExpected\n%s\n\nObserved\n%s\n\nSource\n%s\n", check.Expected, check.Observed, check.Source)
	c.evidencePanel.ScrollToTop()
	c.evidencePanel.SetContent(b.String())
}

func (c *AuditController) currentComponent() *Component {
	if len(c.components) == 0 || c.selectedComponent < 0 || c.selectedComponent >= len(c.components) {
		return nil
	}
	return &c.components[c.selectedComponent]
}

func (c *AuditController) currentState() *State {
	component := c.currentComponent()
	if component == nil || len(component.States) == 0 || c.selectedState < 0 || c.selectedState >= len(component.States) {
		return nil
	}
	return &component.States[c.selectedState]
}

func (c *AuditController) currentCheck() *Check {
	state := c.currentState()
	if state == nil || len(state.Checks) == 0 || c.selectedCheck < 0 || c.selectedCheck >= len(state.Checks) {
		return nil
	}
	return &state.Checks[c.selectedCheck]
}

func stateLabel(state *State) string {
	if state == nil || len(state.Props) == 0 {
		return "default"
	}
	parts := make([]string, 0, len(state.Props))
	for _, prop := range state.Props {
		parts = append(parts, prop.Name+"="+prop.Value)
	}
	return strings.Join(parts, "  ")
}

func shortRuleID(ruleID string) string {
	if _, suffix, found := strings.Cut(ruleID, "/"); found {
		return suffix
	}
	return ruleID
}

func clamp(index, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func stateStatus(state *State) AuditStatus {
	status := StatusPass
	for _, check := range state.Checks {
		if check.Status == StatusFail {
			return StatusFail
		}
		if check.Status == StatusReview {
			status = StatusReview
		}
	}
	return status
}

func checkSummary(state *State) string {
	counts := map[AuditStatus]int{}
	for _, check := range state.Checks {
		counts[check.Status]++
	}
	return fmt.Sprintf("%d fail · %d review · %d pass", counts[StatusFail], counts[StatusReview], counts[StatusPass])
}
