package calibration

// Fields is the projection of one boundary artifact: contract field name to text.
type Fields map[string]string

// Kind classifies how a case was obtained.
type Kind string

// Case kinds.
const (
	KindGolden    Kind = "golden"
	KindCandidate Kind = "candidate"
	KindNatural   Kind = "natural_negative"
	KindSynthetic Kind = "synthetic_negative"
	KindBoundary  Kind = "boundary"
	KindNeutral   Kind = "neutral"
)

// Expect is the verdict a case is built to receive.
type Expect string

// Expected verdicts. ExpectDrop is a relation: the case must score lower than
// its parent. A neutral perturbation expects accept (it must stay positive).
const (
	ExpectAccept Expect = "accept"
	ExpectReject Expect = "reject"
	ExpectDrop   Expect = "drop"
)

// Band is the severity of an injected defect.
type Band string

// Severity bands.
const (
	BandClean Band = "clean"
	BandMinor Band = "minor"
	BandMajor Band = "major"
)

// Case is one labeled item. Fields carry package content and exist only in
// memory: a persisted case is its descriptor, regenerated from source, operator
// and seed.
type Case struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parent_id,omitempty"`
	Kind     Kind     `json:"kind"`
	Operator string   `json:"operator,omitempty"`
	Stage    int      `json:"stage,omitempty"`
	Source   string   `json:"source"`
	Boundary string   `json:"boundary"`
	Expect   Expect   `json:"expect"`
	Criteria []string `json:"criteria,omitempty"`
	Band     Band     `json:"band,omitempty"`
	Level    float64  `json:"level,omitempty"`
	Seed     int64    `json:"seed,omitempty"`
	Golden   bool     `json:"golden,omitempty"`
	HeldOut  bool     `json:"held_out,omitempty"`
	Fields   Fields   `json:"-"`
	Digest   string   `json:"digest,omitempty"`
	Audited  bool     `json:"-"`
}

// Counted reports whether the case enters the golden metrics: candidates never
// do, and a stage 2 item only after it was audited.
func (c Case) Counted() bool {
	if c.Kind == KindCandidate {
		return false
	}
	return c.Stage < 2 || c.Audited
}
