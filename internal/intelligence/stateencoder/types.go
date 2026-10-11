// Package stateencoder validates and deterministically encodes structured
// observations for Intelligence-owned research and future model adapters.
// It is intentionally separate from the frozen CognitiveSnapshot encoders.
package stateencoder

import "time"

const (
	BatchSchemaVersion   = "synora.observation-batch/v1"
	EncoderSchemaVersion = "synora.state-encoder/v1"
	EncoderVersion       = "1.0.0"

	MaxEntities      = 512
	MaxObservations  = 2048
	MaxRelations     = 2048
	MaxIdentifierLen = 192
	MaxEvidenceLen   = 256
)

type ValueType string

const (
	ValueNumber   ValueType = "number"
	ValueBoolean  ValueType = "boolean"
	ValueCategory ValueType = "category"
)

type SemanticStatus string

const (
	StatusObserved    SemanticStatus = "observed"
	StatusUnknown     SemanticStatus = "unknown"
	StatusUnavailable SemanticStatus = "unavailable"
	StatusUnsupported SemanticStatus = "unsupported"
	StatusStale       SemanticStatus = "stale"
	StatusConflicting SemanticStatus = "conflicting"
	StatusAbsent      SemanticStatus = "absent"
)

var statusOrder = [...]SemanticStatus{
	StatusObserved,
	StatusUnknown,
	StatusUnavailable,
	StatusUnsupported,
	StatusStale,
	StatusConflicting,
	StatusAbsent,
}

var valueTypeOrder = [...]ValueType{ValueNumber, ValueBoolean, ValueCategory}

// FeatureNames gives the frozen order of the status and value-type channels.
// Typed values and temporal age are carried as separate named fields rather
// than coerced into one dimensionless scalar.
var featureNames = [...]string{
	"status.observed",
	"status.unknown",
	"status.unavailable",
	"status.unsupported",
	"status.stale",
	"status.conflicting",
	"status.absent",
	"value_type.number",
	"value_type.boolean",
	"value_type.category",
	"value.number_in_declared_unit",
	"value.boolean",
	"value.category_symbol",
	"time.age_seconds",
}

type EntityRef struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type AttributeRef struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	ValueType ValueType `json:"value_type"`
	Unit      string    `json:"unit,omitempty"`
}

type SourceRef struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
}

// Value is a tagged union. Exactly one payload pointer must be set, and it
// must agree with AttributeRef.ValueType when status is observed.
type Value struct {
	Number   *float64 `json:"number,omitempty"`
	Boolean  *bool    `json:"boolean,omitempty"`
	Category *string  `json:"category,omitempty"`
}

type Observation struct {
	Entity      EntityRef      `json:"entity"`
	Attribute   AttributeRef   `json:"attribute"`
	Status      SemanticStatus `json:"status"`
	Value       *Value         `json:"value,omitempty"`
	Source      *SourceRef     `json:"source,omitempty"`
	Provenance  string         `json:"provenance,omitempty"`
	EvidenceRef string         `json:"evidence_ref,omitempty"`
	ObservedAt  time.Time      `json:"observed_at,omitempty"`
}

type Relation struct {
	Subject   string     `json:"subject"`
	Kind      string     `json:"kind"`
	Object    string     `json:"object"`
	Source    *SourceRef `json:"source,omitempty"`
	ValidFrom time.Time  `json:"valid_from,omitempty"`
	ValidTo   time.Time  `json:"valid_to,omitempty"`
}

type EncodedRelation struct {
	Subject   string     `json:"subject"`
	Kind      string     `json:"kind"`
	Object    string     `json:"object"`
	Source    *SourceRef `json:"source,omitempty"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
}

// Batch carries event time separately from reception time. The encoder does
// not infer freshness status or resolve conflicting sources.
type Batch struct {
	SchemaVersion string        `json:"schema_version"`
	EventTime     time.Time     `json:"event_time"`
	ReceivedAt    time.Time     `json:"received_at"`
	Entities      []EntityRef   `json:"entities"`
	Observations  []Observation `json:"observations"`
	Relations     []Relation    `json:"relations,omitempty"`
}

type EncodedObservation struct {
	Entity      EntityRef      `json:"entity"`
	Attribute   AttributeRef   `json:"attribute"`
	Status      SemanticStatus `json:"status"`
	StatusBits  [7]float32     `json:"status_bits"`
	TypeBits    [3]float32     `json:"type_bits"`
	Number      *float32       `json:"number,omitempty"`
	Boolean     *bool          `json:"boolean,omitempty"`
	Category    *string        `json:"category,omitempty"`
	Source      *SourceRef     `json:"source,omitempty"`
	Provenance  string         `json:"provenance,omitempty"`
	EvidenceRef string         `json:"evidence_ref,omitempty"`
	ObservedAt  *time.Time     `json:"observed_at,omitempty"`
	// AgeSeconds is signed event_time - observed_at; negative values are valid.
	AgeSeconds *float64 `json:"age_seconds,omitempty"`
}

type EncodedBatch struct {
	SchemaVersion  string                    `json:"schema_version"`
	EncoderVersion string                    `json:"encoder_version"`
	EventTime      time.Time                 `json:"event_time"`
	ReceivedAt     time.Time                 `json:"received_at"`
	FeatureNames   [len(featureNames)]string `json:"feature_names"`
	Entities       []EntityRef               `json:"entities"`
	Observations   []EncodedObservation      `json:"observations"`
	Relations      []EncodedRelation         `json:"relations,omitempty"`
}

func CanonicalFeatureNames() [len(featureNames)]string { return featureNames }
