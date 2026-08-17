package main

import (
	"log"

	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

func main() {
	app, err := NewApp(AppConfig{
		DebugMode: false,
		AppName:   "lantern-tui",
		Version:   "0.1.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	components := NewListPanel("components", "Components", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	runs := NewTextPanel("runs", "Runs", "", "", true)
	queue := NewListPanel("queue", "Queue", "", "", true)
	config := NewTextPanel("config", "Config", "", "", true)
	runs.SetContent("No audit runs loaded.\n")
	config.SetContent("No Lantern configuration loaded.\n")

	controller := NewAuditController(mockComponents(), components, states, checks, evidence)
	queueController := NewQueueController(controller, app, queue)

	auditLayout := &boxlayout.Box{
		Direction: boxlayout.COLUMN,
		Children: []*boxlayout.Box{
			{
				Direction: boxlayout.ROW,
				Weight:    1,
				Children: []*boxlayout.Box{
					{Window: components.ID(), Weight: 3},
					{Window: states.ID(), Weight: 7},
				},
			},
			{
				Direction: boxlayout.ROW,
				Weight:    1,
				Children: []*boxlayout.Box{
					{Window: checks.ID(), Weight: 3},
					{Window: evidence.ID(), Weight: 7},
				},
			},
		},
	}
	app.RegisterWorkspace(Workspace{ID: "audit", Name: "Audit", Components: []UIComponent{components, checks, states, evidence}, Layout: auditLayout})
	app.RegisterWorkspace(Workspace{ID: "runs", Name: "Runs", Components: []UIComponent{runs}, Layout: &boxlayout.Box{Window: runs.ID()}})
	app.RegisterWorkspace(Workspace{ID: "queue", Name: "Queue", Components: []UIComponent{queue}, Layout: &boxlayout.Box{Window: queue.ID()}})
	app.RegisterWorkspace(Workspace{ID: "config", Name: "Config", Components: []UIComponent{config}, Layout: &boxlayout.Box{Window: config.ID()}})
	for _, capability := range controller.FocusModeCapabilities() {
		app.RegisterFocusModeCapability(capability)
	}
	if err := queueController.RegisterBindings(); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
