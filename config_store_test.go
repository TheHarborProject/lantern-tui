package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, directory, contents string) string {
	t.Helper()
	path := filepath.Join(directory, lanternConfigFilename)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfigStoreLoadsNestedTypedDocument(t *testing.T) {
	directory := t.TempDir()
	path := writeTestConfig(t, directory, `{
  "project": {"root": "packages/ui"},
  "runtime": {"enabled": false, "timeout": 9000},
  "extends": ["lantern:strict"],
  "rules": {"lantern/focus-visible": "error"},
  "future": {"enabled": true}
}`)
	config, exists, err := NewConfigStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if !exists || config.Project.Root != "packages/ui" || config.Runtime.Enabled || config.Runtime.Timeout != 9000 {
		t.Fatalf("loaded config = %#v", config)
	}
	if !reflect.DeepEqual(config.Extends, []string{"lantern:strict"}) || config.Rules["lantern/focus-visible"] != ConfigSeverityError {
		t.Fatal("nested array/rule settings did not load")
	}
	settings := config.Flatten()
	if settings[0].Path != "project.root" || settings[0].Value != "packages/ui" {
		t.Fatal("loaded settings did not flatten")
	}
	if config.IsDirty() {
		t.Fatal("loaded config started dirty")
	}
}

func TestConfigStoreSaveIsNestedTypedFormattedAndPreservesUnknown(t *testing.T) {
	directory := t.TempDir()
	path := writeTestConfig(t, directory, `{"project":{"root":".","future":"keep"},"runtime":{"enabled":true,"timeout":5000},"extends":["lantern:recommended"],"rules":{"lantern/focus-visible":"warn"},"unknown":{"answer":42}}`)
	store := NewConfigStore(path)
	config, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Edit("project.root", "/tmp/project"); err != nil {
		t.Fatal(err)
	}
	if err := config.Toggle("runtime.enabled"); err != nil {
		t.Fatal(err)
	}
	config.Runtime.Timeout = 10000
	if err := config.SetExtends([]string{"lantern:recommended", "lantern:strict"}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetRuleSeverity("rules.lantern/focus-visible", "error"); err != nil {
		t.Fatal(err)
	}
	if !config.IsDirty() {
		t.Fatal("edited config is not dirty")
	}
	if err := store.Save(config); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") || !strings.Contains(string(data), "\n  \"project\":") {
		t.Fatalf("config is not indented with trailing newline:\n%s", data)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if _, flattened := document["project.root"]; flattened {
		t.Fatal("save wrote flattened keys")
	}
	project := document["project"].(map[string]any)
	runtime := document["runtime"].(map[string]any)
	rules := document["rules"].(map[string]any)
	if project["root"] != "/tmp/project" || project["future"] != "keep" {
		t.Fatal("project values were not nested/preserved")
	}
	if runtime["enabled"] != false || runtime["timeout"] != float64(10000) {
		t.Fatal("boolean/number types were not preserved")
	}
	if _, ok := document["extends"].([]any); !ok {
		t.Fatal("extends was not saved as an array")
	}
	if rules["lantern/focus-visible"] != "error" {
		t.Fatal("rule was not saved as a string")
	}
	if document["unknown"].(map[string]any)["answer"] != float64(42) {
		t.Fatal("unknown setting was dropped")
	}
	if config.IsDirty() {
		t.Fatal("successful save did not clear dirty state")
	}
}

func TestConfigDirtyClearsWhenOriginalValueRestored(t *testing.T) {
	config := defaultAuthoredConfig()
	if err := config.Edit("project.root", "changed"); err != nil {
		t.Fatal(err)
	}
	if !config.IsDirty() {
		t.Fatal("edit did not set dirty")
	}
	if err := config.Edit("project.root", "."); err != nil {
		t.Fatal(err)
	}
	if config.IsDirty() {
		t.Fatal("restoring saved value did not clear dirty")
	}
}

func TestMissingConfigUsesUnsavedDefaults(t *testing.T) {
	config, exists, err := NewConfigStore(filepath.Join(t.TempDir(), lanternConfigFilename)).Load()
	if err != nil {
		t.Fatal(err)
	}
	if exists || !config.IsDirty() || config.Project.Root != "." {
		t.Fatal("missing config did not produce unsaved starter config")
	}
}

func TestInvalidExistingConfigCannotBeSilentlyOverwritten(t *testing.T) {
	directory := t.TempDir()
	path := writeTestConfig(t, directory, `{invalid`)
	store := NewConfigStore(path)
	_, _, loadErr := store.Load()
	if loadErr == nil {
		t.Fatal("invalid JSON was accepted")
	}
	fixture := newConfigFixture()
	fixture.controller.SetPersistence(store, loadErr, NewYesNoModal("reload", "Reload", ModalTypeWarning, fixture.app))
	fixture.controller.Save()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{invalid` {
		t.Fatal("invalid existing config was overwritten")
	}
	if !strings.Contains(fixture.panel.subtitle, "Failed to save") {
		t.Fatal("blocked save did not show a clear error")
	}
}

func TestConfigStoreWriteFailureLeavesDestinationIntact(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, lanternConfigFilename)
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "original")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := defaultAuthoredConfig()
	if err := NewConfigStore(destination).Save(config); err == nil {
		t.Fatal("save over directory unexpectedly succeeded")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "keep" {
		t.Fatal("failed save damaged existing destination")
	}
}

func TestConfigControllerReloadAndDirtyConfirmation(t *testing.T) {
	directory := t.TempDir()
	path := writeTestConfig(t, directory, `{"project":{"root":"disk"}}`)
	store := NewConfigStore(path)
	config, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	fixture := newConfigFixture()
	fixture.controller.config = config
	confirm := NewYesNoModal("reload", "Reload", ModalTypeWarning, fixture.app)
	fixture.controller.SetPersistence(store, nil, confirm)
	if err := config.Edit("project.root", "memory"); err != nil {
		t.Fatal(err)
	}
	fixture.controller.Refresh()
	fixture.controller.Reload()
	if !confirm.IsVisible() || fixture.controller.config.Project.Root != "memory" {
		t.Fatal("dirty reload did not wait for confirmation")
	}
	confirm.onYes()
	confirm.Hide()
	if fixture.controller.config.Project.Root != "disk" || fixture.controller.config.IsDirty() {
		t.Fatal("confirmed reload did not restore disk config")
	}

	if err := os.WriteFile(path, []byte(`{"project":{"root":"new-disk"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.controller.Reload()
	if fixture.controller.config.Project.Root != "new-disk" {
		t.Fatal("clean reload did not happen immediately")
	}
}

func TestConfigControllerSaveClearsDirtyAndShowsStatus(t *testing.T) {
	directory := t.TempDir()
	store := NewConfigStore(filepath.Join(directory, lanternConfigFilename))
	fixture := newConfigFixture()
	fixture.config.markNew()
	fixture.controller.SetPersistence(store, nil, NewYesNoModal("reload", "Reload", ModalTypeWarning, fixture.app))
	fixture.panel.SetSelectedIndex(2)
	fixture.controller.Save()
	if fixture.config.IsDirty() {
		t.Fatal("controller save did not clear dirty state")
	}
	if fixture.panel.GetSelectedIndex() != 2 {
		t.Fatal("controller save reset table selection")
	}
	if fixture.panel.subtitle != "Saved lantern.config.json" {
		t.Fatalf("save status = %q", fixture.panel.subtitle)
	}
	if strings.Contains(fixture.panel.title, "Config *") {
		t.Fatalf("dirty marker remained after save: %q", fixture.panel.title)
	}
}
