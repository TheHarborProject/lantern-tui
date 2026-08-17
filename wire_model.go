package main

type WireOutcome string

type CheckSeverity string
type ConfigRuleSeverity string
type DiagnosticSeverity string

const (
	WirePass    WireOutcome = "pass"
	WireFail    WireOutcome = "fail"
	WireReview  WireOutcome = "review"
	WireSkipped WireOutcome = "skipped"
)

const (
	CheckSeverityWarn  CheckSeverity = "warn"
	CheckSeverityError CheckSeverity = "error"
)

const (
	ConfigSeverityOff   ConfigRuleSeverity = "off"
	ConfigSeverityWarn  ConfigRuleSeverity = "warn"
	ConfigSeverityError ConfigRuleSeverity = "error"
)

const (
	DiagnosticSeverityInfo    DiagnosticSeverity = "info"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityError   DiagnosticSeverity = "error"
)

func (s CheckSeverity) Valid() bool {
	return s == CheckSeverityWarn || s == CheckSeverityError
}

func (s ConfigRuleSeverity) Valid() bool {
	return s == ConfigSeverityOff || s == ConfigSeverityWarn || s == ConfigSeverityError
}

func (s DiagnosticSeverity) Valid() bool {
	return s == DiagnosticSeverityInfo || s == DiagnosticSeverityWarning || s == DiagnosticSeverityError
}

type AuditWireDTO struct {
	Version     string              `json:"version"`
	Run         RunWireDTO          `json:"run"`
	Targeting   TargetingWireDTO    `json:"targeting"`
	Engines     []EngineWireDTO     `json:"engines"`
	Config      ConfigWireDTO       `json:"config"`
	Diagnostics []DiagnosticWireDTO `json:"diagnostics"`
	Standards   StandardsWireDTO    `json:"standards"`
	Summary     SummaryWireDTO      `json:"summary"`
}

type RunWireDTO struct {
	ID         string `json:"id"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	Status     string `json:"status"`
	DurationMS int64  `json:"durationMs"`
}

type TargetingWireDTO struct {
	Mode string `json:"mode"`
	Root string `json:"root"`
}

type EngineWireDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ConfigWireDTO struct {
	Version     string                  `json:"version"`
	Fingerprint string                  `json:"fingerprint"`
	Standards   []string                `json:"standards"`
	Rules       []ConfiguredRuleWireDTO `json:"rules"`
}

type ConfiguredRuleWireDTO struct {
	RuleID   string             `json:"ruleId"`
	Severity ConfigRuleSeverity `json:"severity"`
	Enabled  bool               `json:"enabled"`
}

type DiagnosticWireDTO struct {
	Severity DiagnosticSeverity `json:"severity"`
	Code     string             `json:"code"`
	Message  string             `json:"message"`
}

type StandardsWireDTO struct {
	Components []ComponentWireDTO `json:"components"`
}

type ComponentWireDTO struct {
	CanonicalID string         `json:"canonicalId"`
	DisplayName string         `json:"displayName"`
	Source      SourceWireDTO  `json:"source"`
	States      []StateWireDTO `json:"states"`
}

type StateWireDTO struct {
	ID     string            `json:"id"`
	Props  map[string]string `json:"props"`
	Checks []CheckWireDTO    `json:"checks"`
}

type CheckWireDTO struct {
	ID            string               `json:"id"`
	RuleID        string               `json:"ruleId"`
	Outcome       WireOutcome          `json:"outcome"`
	Severity      CheckSeverity        `json:"severity"`
	EngineID      string               `json:"engineId"`
	OutcomeReason OutcomeReasonWireDTO `json:"outcomeReason"`
	Evidence      []EvidenceWireDTO    `json:"evidence"`
	Source        SourceWireDTO        `json:"source"`
	DurationMS    int64                `json:"durationMs"`
}

type OutcomeReasonWireDTO struct {
	Code       string `json:"code"`
	Detail     string `json:"detail"`
	Provenance string `json:"provenance"`
}

type EvidenceWireDTO struct {
	Kind        string                  `json:"kind"`
	Observation *ObservationEvidenceDTO `json:"observation,omitempty"`
	Expectation *ExpectationEvidenceDTO `json:"expectation,omitempty"`
	Element     *ElementEvidenceDTO     `json:"element,omitempty"`
	Source      *SourceWireDTO          `json:"source,omitempty"`
}

type ObservationEvidenceDTO struct {
	Facts map[string]string `json:"facts"`
}

type ExpectationEvidenceDTO struct {
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type ElementEvidenceDTO struct {
	Selector string `json:"selector"`
	HTML     string `json:"html"`
}

type SourceWireDTO struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type SummaryWireDTO struct {
	ComponentCount int `json:"componentCount"`
	StateCount     int `json:"stateCount"`
	CheckCount     int `json:"checkCount"`
	PassCount      int `json:"passCount"`
	FailCount      int `json:"failCount"`
	ReviewCount    int `json:"reviewCount"`
	SkippedCount   int `json:"skippedCount"`
}
