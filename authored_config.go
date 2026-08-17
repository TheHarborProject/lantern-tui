package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type AuthoredConfig struct {
	Project struct {
		Root string
	}
	Runtime struct {
		Enabled bool
		Timeout int
	}
	Extends []string
	Rules   map[string]ConfigRuleSeverity

	document map[string]any
	baseline []byte
	newFile  bool
}

type ConfigEditorKind string

const (
	ConfigEditorToggle      ConfigEditorKind = "toggle"
	ConfigEditorText        ConfigEditorKind = "text"
	ConfigEditorNumber      ConfigEditorKind = "number"
	ConfigEditorSelect      ConfigEditorKind = "select"
	ConfigEditorMultiSelect ConfigEditorKind = "multi-select"
)

type AuthoredConfigSetting struct {
	Path       string
	Value      any
	EditorKind ConfigEditorKind
	Options    []string
}

var knownLanternPresets = []string{"lantern:recommended", "lantern:strict", "lantern:wcag-aa"}
var ruleSeverityOptions = []string{string(ConfigSeverityOff), string(ConfigSeverityWarn), string(ConfigSeverityError)}

func defaultAuthoredConfig() *AuthoredConfig {
	config := &AuthoredConfig{Extends: []string{"lantern:recommended"}, Rules: map[string]ConfigRuleSeverity{
		"lantern/keyboard-access": ConfigSeverityError,
		"lantern/accessible-name": ConfigSeverityError,
		"lantern/focus-visible":   ConfigSeverityWarn,
		"lantern/grid-navigation": ConfigSeverityWarn,
	}}
	config.Project.Root = "."
	config.Runtime.Enabled = true
	config.Runtime.Timeout = 5000
	config.document = map[string]any{}
	config.syncDocument()
	config.markSaved()
	return config
}

func authoredConfigFromDocument(document map[string]any) (*AuthoredConfig, error) {
	if document == nil {
		return nil, fmt.Errorf("configuration root must be a JSON object")
	}
	config := defaultAuthoredConfig()
	config.document = cloneDocument(document)
	if rawProject, exists := document["project"]; exists {
		project, ok := rawProject.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("project must be an object")
		}
		if root, exists := project["root"]; exists {
			value, ok := root.(string)
			if !ok {
				return nil, fmt.Errorf("project.root must be a string")
			}
			config.Project.Root = value
		}
	}
	if rawRuntime, exists := document["runtime"]; exists {
		runtime, ok := rawRuntime.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("runtime must be an object")
		}
		if enabled, exists := runtime["enabled"]; exists {
			value, ok := enabled.(bool)
			if !ok {
				return nil, fmt.Errorf("runtime.enabled must be a boolean")
			}
			config.Runtime.Enabled = value
		}
		if timeout, exists := runtime["timeout"]; exists {
			value, err := jsonNumberAsInt(timeout)
			if err != nil {
				return nil, fmt.Errorf("runtime.timeout must be a whole number")
			}
			config.Runtime.Timeout = value
		}
	}
	if raw, exists := document["extends"]; exists {
		values, err := stringSlice(raw)
		if err != nil {
			return nil, fmt.Errorf("extends must be a string array")
		}
		config.Extends = values
	}
	if rawRules, exists := document["rules"]; exists {
		rules, ok := rawRules.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rules must be an object")
		}
		for ruleID, raw := range rules {
			value, ok := raw.(string)
			if !ok || !ConfigRuleSeverity(value).Valid() {
				return nil, fmt.Errorf("rules.%s must be off, warn, or error", ruleID)
			}
			config.Rules[ruleID] = ConfigRuleSeverity(value)
		}
	}
	config.syncDocument()
	config.markSaved()
	return config, nil
}

func jsonNumberAsInt(value any) (int, error) {
	switch number := value.(type) {
	case json.Number:
		parsed, err := strconv.ParseInt(string(number), 10, 64)
		return int(parsed), err
	case float64:
		if number != float64(int(number)) {
			return 0, fmt.Errorf("not an integer")
		}
		return int(number), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func stringSlice(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		if strings, ok := value.([]string); ok {
			return append([]string(nil), strings...), nil
		}
		return nil, fmt.Errorf("not an array")
	}
	result := make([]string, len(items))
	for i, item := range items {
		stringValue, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("item %d is not a string", i)
		}
		result[i] = stringValue
	}
	return result, nil
}

func cloneDocument(document map[string]any) map[string]any {
	data, _ := json.Marshal(document)
	var clone map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	_ = decoder.Decode(&clone)
	return clone
}

func (c *AuthoredConfig) syncDocument() {
	if c.document == nil {
		c.document = map[string]any{}
	}
	project := ensureObject(c.document, "project")
	project["root"] = c.Project.Root
	runtime := ensureObject(c.document, "runtime")
	runtime["enabled"] = c.Runtime.Enabled
	runtime["timeout"] = c.Runtime.Timeout
	c.document["extends"] = append([]string(nil), c.Extends...)
	rules := ensureObject(c.document, "rules")
	for ruleID, severity := range c.Rules {
		rules[ruleID] = string(severity)
	}
}

func ensureObject(document map[string]any, key string) map[string]any {
	if object, ok := document[key].(map[string]any); ok {
		return object
	}
	object := map[string]any{}
	document[key] = object
	return object
}

func (c *AuthoredConfig) Document() map[string]any {
	c.syncDocument()
	return cloneDocument(c.document)
}

func (c *AuthoredConfig) canonical() []byte {
	c.syncDocument()
	data, _ := json.Marshal(c.document)
	return data
}

func (c *AuthoredConfig) markSaved() {
	c.baseline = append([]byte(nil), c.canonical()...)
	c.newFile = false
}
func (c *AuthoredConfig) markNew()      { c.baseline = nil; c.newFile = true }
func (c *AuthoredConfig) IsDirty() bool { return c.newFile || !bytes.Equal(c.canonical(), c.baseline) }

func (c *AuthoredConfig) ResetToBaseline() error {
	if len(c.baseline) == 0 {
		defaults := defaultAuthoredConfig()
		c.replaceCurrent(defaults)
		c.markSaved()
		return nil
	}
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(c.baseline))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("restore saved configuration: %w", err)
	}
	restored, err := authoredConfigFromDocument(document)
	if err != nil {
		return fmt.Errorf("restore saved configuration: %w", err)
	}
	c.replaceCurrent(restored)
	c.markSaved()
	return nil
}

func (c *AuthoredConfig) ApplyDefaults() {
	baseline := append([]byte(nil), c.baseline...)
	newFile := c.newFile
	c.replaceCurrent(defaultAuthoredConfig())
	c.baseline = baseline
	c.newFile = newFile
}

func (c *AuthoredConfig) replaceCurrent(source *AuthoredConfig) {
	c.Project = source.Project
	c.Runtime = source.Runtime
	c.Extends = append([]string(nil), source.Extends...)
	c.Rules = make(map[string]ConfigRuleSeverity, len(source.Rules))
	for ruleID, severity := range source.Rules {
		c.Rules[ruleID] = severity
	}
	c.document = cloneDocument(source.document)
}

func (c *AuthoredConfig) Flatten() []AuthoredConfigSetting {
	settings := []AuthoredConfigSetting{
		{Path: "project.root", EditorKind: ConfigEditorText, Value: c.Project.Root},
		{Path: "runtime.enabled", EditorKind: ConfigEditorToggle, Value: c.Runtime.Enabled},
		{Path: "runtime.timeout", EditorKind: ConfigEditorNumber, Value: c.Runtime.Timeout},
		{Path: "extends", EditorKind: ConfigEditorMultiSelect, Value: append([]string(nil), c.Extends...), Options: append([]string(nil), knownLanternPresets...)},
	}
	for _, ruleID := range []string{"lantern/keyboard-access", "lantern/accessible-name", "lantern/focus-visible", "lantern/grid-navigation"} {
		settings = append(settings, AuthoredConfigSetting{Path: "rules." + ruleID, EditorKind: ConfigEditorSelect, Value: c.Rules[ruleID], Options: append([]string(nil), ruleSeverityOptions...)})
	}
	return settings
}

func (c *AuthoredConfig) Edit(path, input string) error {
	switch path {
	case "project.root":
		c.Project.Root = input
	case "runtime.timeout":
		value, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("expected a whole number")
		}
		c.Runtime.Timeout = value
	default:
		return fmt.Errorf("setting %q is not edited as text", path)
	}
	return nil
}

func (c *AuthoredConfig) Toggle(path string) error {
	if path != "runtime.enabled" {
		return fmt.Errorf("setting %q is not boolean", path)
	}
	c.Runtime.Enabled = !c.Runtime.Enabled
	return nil
}

func (c *AuthoredConfig) SetExtends(values []string) error {
	known := map[string]bool{}
	for _, option := range knownLanternPresets {
		known[option] = true
	}
	for _, value := range values {
		if !known[value] {
			return fmt.Errorf("unknown Lantern preset %q", value)
		}
	}
	c.Extends = append([]string(nil), values...)
	return nil
}

func (c *AuthoredConfig) SetRuleSeverity(path, value string) error {
	const prefix = "rules."
	if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
		return fmt.Errorf("setting %q is not a rule", path)
	}
	severity := ConfigRuleSeverity(value)
	if !severity.Valid() {
		return fmt.Errorf("expected off, warn, or error")
	}
	c.Rules[path[len(prefix):]] = severity
	return nil
}

func formatAuthoredValue(setting AuthoredConfigSetting) string {
	if setting.EditorKind == ConfigEditorMultiSelect {
		encoded, _ := json.Marshal(setting.Value)
		return string(encoded)
	}
	return fmt.Sprint(setting.Value)
}
