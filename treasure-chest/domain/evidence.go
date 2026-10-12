package domain

// Evidence classes are part of the Treasure Chest interchange contract. They
// intentionally live in this package instead of the host's internal domain so
// the provider can be built and moved without importing Strategist internals.
const (
	EvidenceClassExplicit              = "explicit"
	EvidenceClassCorroboratedInference = "corroborated_inference"
	EvidenceClassWeakInference         = "weak_inference"
	EvidenceClassUnknown               = "unknown"
)

var orderedEvidenceClasses = []string{
	EvidenceClassExplicit,
	EvidenceClassCorroboratedInference,
	EvidenceClassWeakInference,
	EvidenceClassUnknown,
}

// EvidenceClasses returns the accepted evidence classes in descending
// strength order.
func EvidenceClasses() []string {
	return append([]string(nil), orderedEvidenceClasses...)
}

// Confidence levels are part of the Treasure Chest interchange contract.
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

var orderedConfidenceLevels = []string{
	ConfidenceLow,
	ConfidenceMedium,
	ConfidenceHigh,
}

// ConfidenceLevels returns the accepted confidence levels in ascending
// confidence order.
func ConfidenceLevels() []string {
	return append([]string(nil), orderedConfidenceLevels...)
}
