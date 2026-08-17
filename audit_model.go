package main

type AuditStatus string

const (
	StatusPass   AuditStatus = "pass"
	StatusFail   AuditStatus = "fail"
	StatusReview AuditStatus = "review"
)

type EvidenceField struct {
	Name  string
	Value string
}

type Check struct {
	ID            string
	RuleID        string
	Status        AuditStatus
	Severity      string
	Engine        string
	OutcomeReason string
	Evidence      []EvidenceField
	Expected      string
	Observed      string
	Source        string
}

type State struct {
	ID     string
	Props  []EvidenceField
	Checks []Check
}

type Component struct {
	CanonicalID string
	DisplayName string
	SourceFile  string
	Status      AuditStatus
	States      []State
}

func statusIcon(status AuditStatus) string {
	switch status {
	case StatusPass:
		return "✓"
	case StatusFail:
		return "✗"
	default:
		return "◌"
	}
}
