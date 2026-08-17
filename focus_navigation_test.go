package main

import "testing"

func TestAuditPanelFocusOrder(t *testing.T) {
	components := NewListPanel("components", "Components", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	app := &App{components: []UIComponent{components, checks, states, evidence}}

	focusables := app.getFocusableComponents()
	want := []string{"components", "checks", "states", "evidence"}
	if len(focusables) != len(want) {
		t.Fatalf("focusable count = %d, want %d", len(focusables), len(want))
	}
	for i, id := range want {
		if got := focusables[i].ID(); got != id {
			t.Errorf("focusables[%d] = %q, want %q", i, got, id)
		}
	}
}

func TestEvidencePanelIsFocusable(t *testing.T) {
	var component UIComponent = NewTextPanel("evidence", "Evidence", "", "", true)
	if _, ok := component.(Focusable); !ok {
		t.Fatal("evidence text panel must implement Focusable")
	}
}
