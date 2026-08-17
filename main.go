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
	runs := NewListPanel("runs", "Runs", "", "", true)
	runDetails := NewTextPanel("run-details", "Run Details", "", "", true)
	queue := NewListPanel("queue", "Queue", "", "", true)
	config := NewListPanel("config", "Config", "", "", true)
	configQuickActions := NewListPanel("config-quick-actions", "Quick Actions", "", "", true)
	fixtures := lanternWireFixtures()
	controller := NewAuditController(adaptAuditComponents(fixtures[0]), components, states, checks, evidence)
	NewRunsController(fixtures, runs, runDetails)
	configStore, err := NewCWDConfigStore()
	if err != nil {
		log.Fatal(err)
	}
	authoredConfig, _, configLoadErr := configStore.Load()
	if authoredConfig == nil {
		authoredConfig = defaultAuthoredConfig()
	}
	configInput := NewInputModal("config-input", "Edit setting", ModalTypeInfo, 0, app)
	configSelect := NewChoiceModal("config-select", "Select value", false, app)
	configMultiSelect := NewChoiceModal("config-multi-select", "Select values", true, app)
	configReloadConfirm := NewYesNoModal("config-reload-confirm", "Reload Config", ModalTypeWarning, app)
	configController := NewConfigController(authoredConfig, config, configInput, configSelect, configMultiSelect)
	configController.SetQuickActionsPanel(configQuickActions)
	configController.SetPersistence(configStore, configLoadErr, configReloadConfirm)
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
	app.RegisterWorkspace(Workspace{ID: "runs", Name: "Runs", Components: []UIComponent{runs, runDetails}, Layout: &boxlayout.Box{Direction: boxlayout.ROW, Children: []*boxlayout.Box{{Window: runs.ID(), Weight: 3}, {Window: runDetails.ID(), Weight: 7}}}})
	app.RegisterWorkspace(Workspace{ID: "queue", Name: "Queue", Components: []UIComponent{queue}, Layout: &boxlayout.Box{Window: queue.ID()}})
	app.RegisterWorkspace(Workspace{
		ID:         "config",
		Name:       "Config",
		Components: []UIComponent{config, configQuickActions},
		Layout: &boxlayout.Box{Direction: boxlayout.ROW, Children: []*boxlayout.Box{
			{Window: config.ID(), Weight: 3},
			{Window: configQuickActions.ID(), Weight: 1},
		}},
	})
	for _, capability := range controller.FocusModeCapabilities() {
		app.RegisterFocusModeCapability(capability)
	}
	app.RegisterModal(configInput)
	app.RegisterModal(configSelect)
	app.RegisterModal(configMultiSelect)
	app.RegisterModal(configReloadConfirm)
	if err := queueController.RegisterBindings(); err != nil {
		log.Fatal(err)
	}
	if err := configController.RegisterBindings(app); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
