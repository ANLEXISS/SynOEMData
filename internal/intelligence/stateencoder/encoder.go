package stateencoder

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	namespacePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	namePattern      = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	kindPattern      = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	qualifiedID      = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}:[A-Za-z0-9][A-Za-z0-9._~:/@+-]{0,127}$`)
	tokenPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@+-]{0,127}$`)
)

var ErrInvalidBatch = errors.New("invalid observation batch")

// Encode validates a batch and returns a canonical, bounded representation.
// It never truncates inputs, invents missing timestamps, hashes identifiers,
// classifies values as stale, or picks a winning source.
func Encode(batch Batch) (EncodedBatch, error) {
	if err := batch.Validate(); err != nil {
		return EncodedBatch{}, err
	}
	out := EncodedBatch{
		SchemaVersion:  EncoderSchemaVersion,
		EncoderVersion: EncoderVersion,
		EventTime:      batch.EventTime,
		ReceivedAt:     batch.ReceivedAt,
		FeatureNames:   CanonicalFeatureNames(),
		Entities:       append([]EntityRef(nil), batch.Entities...),
		Observations:   make([]EncodedObservation, 0, len(batch.Observations)),
		Relations:      make([]EncodedRelation, 0, len(batch.Relations)),
	}
	sort.Slice(out.Entities, func(i, j int) bool {
		if out.Entities[i].ID != out.Entities[j].ID {
			return out.Entities[i].ID < out.Entities[j].ID
		}
		return out.Entities[i].Kind < out.Entities[j].Kind
	})
	relations := append([]Relation(nil), batch.Relations...)
	sort.Slice(relations, func(i, j int) bool { return relationKey(relations[i]) < relationKey(relations[j]) })
	for _, relation := range relations {
		out.Relations = append(out.Relations, EncodedRelation{
			Subject:   relation.Subject,
			Kind:      relation.Kind,
			Object:    relation.Object,
			Source:    cloneSource(relation.Source),
			ValidFrom: timePointer(relation.ValidFrom),
			ValidTo:   timePointer(relation.ValidTo),
		})
	}

	observations := append([]Observation(nil), batch.Observations...)
	sort.Slice(observations, func(i, j int) bool {
		return observationIdentityKey(observations[i]) < observationIdentityKey(observations[j])
	})
	for _, observation := range observations {
		encoded := EncodedObservation{
			Entity:      observation.Entity,
			Attribute:   observation.Attribute,
			Status:      observation.Status,
			Source:      cloneSource(observation.Source),
			Provenance:  observation.Provenance,
			EvidenceRef: observation.EvidenceRef,
			ObservedAt:  timePointer(observation.ObservedAt),
		}
		for i, status := range statusOrder {
			if observation.Status == status {
				encoded.StatusBits[i] = 1
			}
		}
		for i, valueType := range valueTypeOrder {
			if observation.Attribute.ValueType == valueType {
				encoded.TypeBits[i] = 1
			}
		}
		if observation.Value != nil {
			switch observation.Attribute.ValueType {
			case ValueNumber:
				value := float32(*observation.Value.Number)
				encoded.Number = &value
			case ValueBoolean:
				value := *observation.Value.Boolean
				encoded.Boolean = &value
			case ValueCategory:
				value := *observation.Value.Category
				encoded.Category = &value
			}
		}
		if !observation.ObservedAt.IsZero() {
			age, _ := signedAgeSeconds(batch.EventTime, observation.ObservedAt)
			encoded.AgeSeconds = &age
		}
		out.Observations = append(out.Observations, encoded)
	}
	return out, nil
}

func (b Batch) Validate() error {
	if b.SchemaVersion != BatchSchemaVersion {
		return invalid("schema_version", "must be %q", BatchSchemaVersion)
	}
	if b.EventTime.IsZero() || b.ReceivedAt.IsZero() {
		return invalid("timestamps", "event_time and received_at are required and remain distinct")
	}
	if len(b.Entities) == 0 || len(b.Entities) > MaxEntities {
		return invalid("entities", "count must be between 1 and %d", MaxEntities)
	}
	if len(b.Observations) == 0 || len(b.Observations) > MaxObservations {
		return invalid("observations", "count must be between 1 and %d", MaxObservations)
	}
	if len(b.Relations) > MaxRelations {
		return invalid("relations", "count must not exceed %d", MaxRelations)
	}

	entities := make(map[string]string, len(b.Entities))
	for i, entity := range b.Entities {
		if !validQualifiedID(entity.ID) || !kindPattern.MatchString(entity.Kind) {
			return invalid(fmt.Sprintf("entities[%d]", i), "requires a namespaced id and supported kind")
		}
		if _, exists := entities[entity.ID]; exists {
			return invalid(fmt.Sprintf("entities[%d].id", i), "duplicate entity id %q", entity.ID)
		}
		entities[entity.ID] = entity.Kind
	}

	seenObservations := make(map[string]struct{}, len(b.Observations))
	for i, observation := range b.Observations {
		path := fmt.Sprintf("observations[%d]", i)
		entityKind, exists := entities[observation.Entity.ID]
		if !exists || entityKind != observation.Entity.Kind {
			return invalid(path+".entity", "must reference a declared entity with matching kind")
		}
		if !namespacePattern.MatchString(observation.Attribute.Namespace) || !namePattern.MatchString(observation.Attribute.Name) {
			return invalid(path+".attribute", "namespace and name must use the v1 identifier form")
		}
		if !validValueType(observation.Attribute.ValueType) {
			return invalid(path+".attribute.value_type", "unsupported value type %q", observation.Attribute.ValueType)
		}
		if observation.Attribute.ValueType == ValueNumber {
			if !tokenPattern.MatchString(observation.Attribute.Unit) {
				return invalid(path+".attribute.unit", "numeric attributes require an explicit unit token")
			}
		} else if observation.Attribute.Unit != "" {
			return invalid(path+".attribute.unit", "units are only valid for numeric attributes")
		}
		if !validStatus(observation.Status) {
			return invalid(path+".status", "unsupported semantic status %q", observation.Status)
		}
		if err := validateSource(observation.Source, path+".source"); err != nil {
			return err
		}
		if len(observation.Provenance) > MaxEvidenceLen || len(observation.EvidenceRef) > MaxEvidenceLen {
			return invalid(path, "provenance and evidence_ref must not exceed %d bytes", MaxEvidenceLen)
		}
		if err := validateObservationValue(observation, path); err != nil {
			return err
		}
		if !observation.ObservedAt.IsZero() {
			if _, ok := signedAgeSeconds(b.EventTime, observation.ObservedAt); !ok {
				return invalid(path+".observed_at", "age is not representable")
			}
		}
		key := observationIdentityKey(observation)
		if _, duplicate := seenObservations[key]; duplicate {
			return invalid(path, "duplicate entity/attribute/source record; represent source conflicts explicitly")
		}
		seenObservations[key] = struct{}{}
	}

	seenRelations := make(map[string]struct{}, len(b.Relations))
	for i, relation := range b.Relations {
		path := fmt.Sprintf("relations[%d]", i)
		if _, exists := entities[relation.Subject]; !exists {
			return invalid(path+".subject", "unknown entity id %q", relation.Subject)
		}
		if _, exists := entities[relation.Object]; !exists {
			return invalid(path+".object", "unknown entity id %q", relation.Object)
		}
		if relation.Subject == relation.Object || !validQualifiedID(relation.Subject) || !validQualifiedID(relation.Object) {
			return invalid(path, "requires distinct, valid entity references")
		}
		if !tokenPattern.MatchString(relation.Kind) {
			return invalid(path+".kind", "requires a non-empty relation token")
		}
		if err := validateSource(relation.Source, path+".source"); err != nil {
			return err
		}
		if !relation.ValidFrom.IsZero() && !relation.ValidTo.IsZero() && !relation.ValidTo.After(relation.ValidFrom) {
			return invalid(path, "valid_to must be after valid_from")
		}
		key := relationKey(relation)
		if _, duplicate := seenRelations[key]; duplicate {
			return invalid(path, "duplicate relation")
		}
		seenRelations[key] = struct{}{}
	}
	return nil
}

func validateObservationValue(observation Observation, path string) error {
	status := observation.Status
	valueAllowed := status == StatusObserved || status == StatusStale || status == StatusConflicting
	if observation.Value != nil && !valueAllowed {
		return invalid(path+".value", "status %q cannot carry a value", status)
	}
	if status == StatusObserved && observation.Value == nil {
		return invalid(path+".value", "observed status requires a value")
	}
	if observation.Value != nil && observation.ObservedAt.IsZero() {
		return invalid(path+".observed_at", "a retained value requires its observation time")
	}
	if observation.Value == nil && (status == StatusObserved || !observation.ObservedAt.IsZero()) {
		return invalid(path+".value", "a value timestamp requires a value")
	}
	if observation.Value == nil {
		return nil
	}
	value := observation.Value
	count := 0
	if value.Number != nil {
		count++
	}
	if value.Boolean != nil {
		count++
	}
	if value.Category != nil {
		count++
	}
	if count != 1 {
		return invalid(path+".value", "exactly one typed value must be set")
	}
	switch observation.Attribute.ValueType {
	case ValueNumber:
		if value.Number == nil || math.IsNaN(*value.Number) || math.IsInf(*value.Number, 0) || math.Abs(*value.Number) > math.MaxFloat32 {
			return invalid(path+".value.number", "must be finite and representable as float32")
		}
		if *value.Number != 0 && float32(*value.Number) == 0 {
			return invalid(path+".value.number", "non-zero value underflows float32 representation")
		}
	case ValueBoolean:
		if value.Boolean == nil {
			return invalid(path+".value.boolean", "must match the declared boolean type")
		}
	case ValueCategory:
		if value.Category == nil || strings.TrimSpace(*value.Category) == "" || len(*value.Category) > 128 {
			return invalid(path+".value.category", "must be non-empty and at most 128 bytes")
		}
	}
	return nil
}

func validateSource(source *SourceRef, path string) error {
	if source == nil {
		return nil
	}
	if !validQualifiedID(source.ID) || !kindPattern.MatchString(source.Kind) {
		return invalid(path, "source id and kind must both use the v1 identifier form")
	}
	return nil
}

func validQualifiedID(value string) bool {
	return len(value) <= MaxIdentifierLen && qualifiedID.MatchString(value)
}

func validValueType(value ValueType) bool {
	for _, supported := range valueTypeOrder {
		if value == supported {
			return true
		}
	}
	return false
}

func validStatus(value SemanticStatus) bool {
	for _, supported := range statusOrder {
		if value == supported {
			return true
		}
	}
	return false
}

func invalid(path, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidBatch, path, fmt.Sprintf(format, args...))
}

func observationIdentityKey(value Observation) string {
	if !value.ObservedAt.IsZero() {
		value.ObservedAt = value.ObservedAt.UTC()
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func relationKey(value Relation) string {
	if !value.ValidFrom.IsZero() {
		value.ValidFrom = value.ValidFrom.UTC()
	}
	if !value.ValidTo.IsZero() {
		value.ValidTo = value.ValidTo.UTC()
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func signedAgeSeconds(eventTime, observedAt time.Time) (float64, bool) {
	eventSeconds := eventTime.Unix()
	observedSeconds := observedAt.Unix()
	const maxInt64 = int64(1<<63 - 1)
	const minInt64 = -1 << 63
	if (observedSeconds > 0 && eventSeconds < minInt64+observedSeconds) ||
		(observedSeconds < 0 && eventSeconds > maxInt64+observedSeconds) {
		return 0, false
	}
	seconds := eventSeconds - observedSeconds
	nanoseconds := int64(eventTime.Nanosecond() - observedAt.Nanosecond())
	age := float64(seconds) + float64(nanoseconds)/1e9
	return age, !math.IsInf(age, 0) && !math.IsNaN(age)
}

func cloneSource(value *SourceRef) *SourceRef {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func timePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
