package stateencoder

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTemporalDeltasAreSignedAndTimestampsArePreserved(t *testing.T) {
	event := time.Date(2026, 4, 5, 6, 7, 8, 0, time.FixedZone("declared", 5*60*60+30*60))
	base := minimalBatch(event, event.Add(10*time.Second))
	cases := []struct {
		name   string
		offset time.Duration
		want   float64
	}{
		{name: "before event", offset: -2 * time.Second, want: 2},
		{name: "at event", offset: 0, want: 0},
		{name: "after event", offset: 3 * time.Second, want: -3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			batch := cloneBatch(base)
			batch.Observations[0].ObservedAt = event.Add(tc.offset)
			encoded, err := Encode(batch)
			if err != nil {
				t.Fatalf("Encode rejected valid temporal data: %v", err)
			}
			observation := encoded.Observations[0]
			if observation.AgeSeconds == nil || *observation.AgeSeconds != tc.want {
				t.Fatalf("signed age = %v, want %v", observation.AgeSeconds, tc.want)
			}
			if observation.ObservedAt == nil || !observation.ObservedAt.Equal(batch.Observations[0].ObservedAt) {
				t.Fatalf("observed_at instant changed: %v", observation.ObservedAt)
			}
			_, gotOffset := observation.ObservedAt.Zone()
			_, wantOffset := batch.Observations[0].ObservedAt.Zone()
			if gotOffset != wantOffset {
				t.Fatalf("observed_at offset changed: got %d, want %d", gotOffset, wantOffset)
			}
			_, eventOffset := encoded.EventTime.Zone()
			_, inputEventOffset := batch.EventTime.Zone()
			if eventOffset != inputEventOffset {
				t.Fatalf("event_time offset changed: got %d, want %d", eventOffset, inputEventOffset)
			}
			if !encoded.ReceivedAt.Equal(batch.ReceivedAt) {
				t.Fatalf("received_at changed: got %v, want %v", encoded.ReceivedAt, batch.ReceivedAt)
			}
			if observation.Status != StatusObserved {
				t.Fatalf("encoder inferred status %q from temporal delta", observation.Status)
			}
		})
	}
}

func TestTemporalNanosecondsOutOfOrderAndReplay(t *testing.T) {
	event := time.Date(2026, 1, 1, 0, 0, 0, 2, time.UTC)
	batch := minimalBatch(event, event.Add(5*time.Second))
	batch.Observations[0].ObservedAt = event.Add(-3 * time.Nanosecond)
	first, err := Encode(batch)
	if err != nil {
		t.Fatal(err)
	}
	age := *first.Observations[0].AgeSeconds
	if math.Abs(age-3e-9) > 1e-15 {
		t.Fatalf("nanosecond age = %.18g, want approximately 3ns", age)
	}

	secondObservation := batch.Observations[0]
	secondObservation.ObservedAt = event.Add(-2 * time.Nanosecond)
	secondObservation.Value = &Value{Boolean: boolPointer(false)}
	batch.Observations = append(batch.Observations, secondObservation)
	// Reverse chronological input remains valid and is canonically ordered.
	encoded, err := Encode(batch)
	if err != nil {
		t.Fatalf("out-of-order observations rejected: %v", err)
	}
	if len(encoded.Observations) != 2 || encoded.Observations[0].ObservedAt.Equal(*encoded.Observations[1].ObservedAt) {
		t.Fatalf("out-of-order observations were not retained distinctly: %+v", encoded.Observations)
	}
	permuted := cloneBatch(batch)
	permuted.Observations[0], permuted.Observations[1] = permuted.Observations[1], permuted.Observations[0]
	permutedEncoded, err := Encode(permuted)
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := json.Marshal(encoded)
	permutedCanonical, _ := json.Marshal(permutedEncoded)
	if !bytes.Equal(canonical, permutedCanonical) {
		t.Fatal("permuting same-attribute temporal observations changed canonical output")
	}
	replayed, err := Encode(batch)
	if err != nil {
		t.Fatalf("stateless replay was rejected: %v", err)
	}
	left, _ := json.Marshal(encoded)
	right, _ := json.Marshal(replayed)
	if !bytes.Equal(left, right) {
		t.Fatal("replaying the same historical batch changed its encoded output")
	}
}

func TestDistinctSameAttributeObservationsArePreserved(t *testing.T) {
	event := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	batch := minimalBatch(event, event.Add(time.Minute))
	older := batch.Observations[0]
	older.Source = &SourceRef{ID: "sensor:second", Kind: "sensor"}
	older.ObservedAt = event.Add(-time.Minute)
	older.Value = &Value{Boolean: boolPointer(false)}
	sameSourceLater := batch.Observations[0]
	sameSourceLater.ObservedAt = event.Add(-30 * time.Second)
	sameSourceLater.Value = &Value{Boolean: boolPointer(false)}
	batch.Observations = []Observation{sameSourceLater, older, batch.Observations[0]}
	encoded, err := Encode(batch)
	if err != nil {
		t.Fatalf("distinct observations for one entity/attribute were rejected: %v", err)
	}
	if len(encoded.Observations) != 3 {
		t.Fatalf("preserved %d observations, want 3", len(encoded.Observations))
	}
}

func TestExactDuplicateRecordRejectedButDistinctRecordAllowed(t *testing.T) {
	event := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	batch := minimalBatch(event, event.Add(time.Second))
	batch.Observations = append(batch.Observations, batch.Observations[0])
	encoded, err := Encode(batch)
	if err == nil {
		t.Fatal("exact duplicate record was accepted")
	}
	if encoded.SchemaVersion != "" || len(encoded.Observations) != 0 || len(encoded.Entities) != 0 || len(encoded.Relations) != 0 {
		t.Fatalf("invalid batch returned partial encoded output: %+v", encoded)
	}
	batch.Observations[1].ObservedAt = batch.Observations[1].ObservedAt.Add(-time.Second)
	if _, err := Encode(batch); err != nil {
		t.Fatalf("same source and attribute at a distinct time was rejected: %v", err)
	}
}

func TestTemporalDeltaWideRangeAndOverflow(t *testing.T) {
	batch := minimalBatch(time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC))
	batch.Observations[0].ObservedAt = time.Date(1, 1, 1, 0, 0, 1, 0, time.UTC)
	encoded, err := Encode(batch)
	if err != nil {
		t.Fatalf("wide but representable timestamp span rejected: %v", err)
	}
	if math.IsInf(*encoded.Observations[0].AgeSeconds, 0) || math.IsNaN(*encoded.Observations[0].AgeSeconds) || *encoded.Observations[0].AgeSeconds < 3e11 {
		t.Fatalf("wide timestamp span was not represented finitely: %v", *encoded.Observations[0].AgeSeconds)
	}

	batch.EventTime = time.Unix(1<<63-1, 0)
	batch.Observations[0].ObservedAt = time.Unix(-1<<63, 0)
	if _, err := Encode(batch); err == nil {
		t.Fatal("unrepresentable signed timestamp delta was accepted")
	}
}

func TestAllDeclaredStatusesRemainInputOwned(t *testing.T) {
	statuses := []SemanticStatus{StatusObserved, StatusUnknown, StatusUnavailable, StatusUnsupported, StatusStale, StatusConflicting, StatusAbsent}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			event := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			batch := minimalBatch(event, event)
			observation := batch.Observations[0]
			observation.Status = status
			if status == StatusUnknown || status == StatusUnavailable || status == StatusUnsupported || status == StatusAbsent {
				observation.Value = nil
				observation.ObservedAt = time.Time{}
			} else {
				observation.Value = &Value{Boolean: boolPointer(true)}
				observation.ObservedAt = event.Add(time.Second)
			}
			batch.Observations[0] = observation
			encoded, err := Encode(batch)
			if err != nil {
				t.Fatalf("declared status %q rejected: %v", status, err)
			}
			result := encoded.Observations[0]
			if result.Status != status {
				t.Fatalf("status changed from %q to %q", status, result.Status)
			}
			if result.StatusBits[statusIndex(status)] != 1 {
				t.Fatalf("status bit missing for %q: %v", status, result.StatusBits)
			}
			if status == StatusObserved && (result.AgeSeconds == nil || *result.AgeSeconds != -1) {
				t.Fatalf("future observed status did not retain signed age: %v", result.AgeSeconds)
			}
		})
	}
}

func TestCyclicRelationsAndIsolatedEntitiesArePreserved(t *testing.T) {
	event := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	batch := minimalBatch(event, event.Add(time.Second))
	batch.Entities = []EntityRef{
		{ID: "room:a", Kind: "room"}, {ID: "room:b", Kind: "room"},
		{ID: "room:c", Kind: "room"}, {ID: "room:isolated", Kind: "room"},
	}
	for i := range batch.Observations {
		batch.Observations[i].Entity = batch.Entities[0]
	}
	validFrom := time.Date(2026, 4, 1, 0, 0, 1, 0, time.FixedZone("relation", -4*60*60))
	validTo := validFrom.Add(time.Hour)
	batch.Relations = []Relation{
		{Subject: "room:a", Kind: "connects", Object: "room:b", Source: &SourceRef{ID: "map:one", Kind: "map"}, ValidFrom: validFrom, ValidTo: validTo},
		{Subject: "room:b", Kind: "connects", Object: "room:c"},
		{Subject: "room:c", Kind: "connects", Object: "room:a"},
	}
	encoded, err := Encode(batch)
	if err != nil {
		t.Fatalf("cycle was rejected: %v", err)
	}
	if len(encoded.Entities) != 4 || encoded.Entities[3].ID != "room:isolated" || len(encoded.Relations) != 3 {
		t.Fatalf("entities or directed cycle changed: entities=%+v relations=%+v", encoded.Entities, encoded.Relations)
	}
	if encoded.Relations[0].Subject != "room:a" || encoded.Relations[0].Object != "room:b" {
		t.Fatalf("relation direction changed: %+v", encoded.Relations[0])
	}
	if encoded.Relations[0].ValidFrom == nil || !encoded.Relations[0].ValidFrom.Equal(validFrom) || encoded.Relations[0].ValidTo == nil || !encoded.Relations[0].ValidTo.Equal(validTo) || encoded.Relations[0].Source == nil || encoded.Relations[0].Source.ID != "map:one" {
		t.Fatalf("relation validity or provenance changed: %+v", encoded.Relations[0])
	}
	_, relationOffset := encoded.Relations[0].ValidFrom.Zone()
	_, inputRelationOffset := validFrom.Zone()
	if relationOffset != inputRelationOffset {
		t.Fatalf("relation validity offset changed: got %d, want %d", relationOffset, inputRelationOffset)
	}
	batch.Relations = batch.Relations[:2]
	changedTopology, err := Encode(batch)
	if err != nil {
		t.Fatal(err)
	}
	if len(changedTopology.Relations) == len(encoded.Relations) {
		t.Fatal("removing an explicit edge did not change encoded topology")
	}
}

func TestFloat32BoundariesAndUnderflow(t *testing.T) {
	for _, value := range []float64{0, -1.25, float64(math.MaxFloat32), float64(math.SmallestNonzeroFloat32)} {
		batch := numberBatch(value)
		encoded, err := Encode(batch)
		if err != nil {
			t.Fatalf("representable value %g rejected: %v", value, err)
		}
		if math.IsInf(float64(*encoded.Observations[0].Number), 0) || math.IsNaN(float64(*encoded.Observations[0].Number)) {
			t.Fatalf("finite input encoded non-finite: %g", value)
		}
	}
	for _, value := range []float64{math.Nextafter(float64(math.MaxFloat32), math.Inf(1)), math.SmallestNonzeroFloat64, math.Inf(1), math.NaN()} {
		if _, err := Encode(numberBatch(value)); err == nil {
			t.Fatalf("unrepresentable numeric value %g accepted", value)
		}
	}
	value := 1.0 + math.Ldexp(1, -25)
	encoded, err := Encode(numberBatch(value))
	if err != nil {
		t.Fatal(err)
	}
	if *encoded.Observations[0].Number != float32(value) {
		t.Fatalf("float32 conversion was not deterministic: %g", *encoded.Observations[0].Number)
	}
}

func TestExactResourceLimitsAcceptedAndOverflowRejected(t *testing.T) {
	event := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	batch := Batch{SchemaVersion: BatchSchemaVersion, EventTime: event, ReceivedAt: event.Add(time.Second)}
	for i := 0; i < MaxEntities; i++ {
		batch.Entities = append(batch.Entities, EntityRef{ID: "device:entity-" + strconv.Itoa(i), Kind: "device"})
	}
	for i := 0; i < MaxObservations; i++ {
		batch.Observations = append(batch.Observations, Observation{
			Entity:    batch.Entities[i%len(batch.Entities)],
			Attribute: AttributeRef{Namespace: "state", Name: "item-" + strconv.Itoa(i), ValueType: ValueBoolean},
			Status:    StatusUnknown,
		})
	}
	for i := 0; i < MaxRelations; i++ {
		batch.Relations = append(batch.Relations, Relation{Subject: batch.Entities[0].ID, Kind: "link-" + strconv.Itoa(i), Object: batch.Entities[1].ID})
	}
	if _, err := Encode(batch); err != nil {
		t.Fatalf("exact resource limits rejected: %v", err)
	}

	tooMany := batch
	tooMany.Entities = append(append([]EntityRef(nil), batch.Entities...), EntityRef{ID: "device:extra", Kind: "device"})
	if _, err := Encode(tooMany); err == nil {
		t.Fatal("entity overflow accepted")
	}
	tooMany = batch
	tooMany.Relations = append(append([]Relation(nil), batch.Relations...), Relation{Subject: batch.Entities[0].ID, Kind: "extra", Object: batch.Entities[1].ID})
	if _, err := Encode(tooMany); err == nil {
		t.Fatal("relation overflow accepted")
	}
}

func TestIdentifierAndEvidenceLimits(t *testing.T) {
	batch := minimalBatch(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 1, 0, 0, 1, 0, time.UTC))
	batch.Entities[0].ID = strings.Repeat("a", 63) + ":" + strings.Repeat("A", 128)
	batch.Observations[0].Entity = batch.Entities[0]
	batch.Observations[0].Source.ID = strings.Repeat("b", 63) + ":" + strings.Repeat("B", 128)
	batch.Observations[0].Provenance = strings.Repeat("p", MaxEvidenceLen)
	batch.Observations[0].EvidenceRef = strings.Repeat("e", MaxEvidenceLen)
	if _, err := Encode(batch); err != nil {
		t.Fatalf("exact identifier/evidence limits rejected: %v", err)
	}
	batch.Entities[0].ID += "A"
	batch.Observations[0].Entity = batch.Entities[0]
	if _, err := Encode(batch); err == nil {
		t.Fatal("identifier over 192 bytes accepted")
	}
	batch.Entities[0].ID = strings.Repeat("a", 63) + ":" + strings.Repeat("A", 128)
	batch.Observations[0].Entity = batch.Entities[0]
	batch.Observations[0].EvidenceRef += "e"
	if _, err := Encode(batch); err == nil {
		t.Fatal("evidence reference over limit accepted")
	}
}

func TestMalformedTimestampTextRejectedByStandardParser(t *testing.T) {
	var batch Batch
	if err := json.Unmarshal([]byte(`{"schema_version":"synora.observation-batch/v1","event_time":"2026-99-99T99:99:99Z"}`), &batch); err == nil {
		t.Fatal("malformed timestamp text parsed successfully")
	}
}

func minimalBatch(eventTime, receivedAt time.Time) Batch {
	observedAt := eventTime.Add(-time.Second)
	return Batch{
		SchemaVersion: BatchSchemaVersion,
		EventTime:     eventTime,
		ReceivedAt:    receivedAt,
		Entities:      []EntityRef{{ID: "device:one", Kind: "device"}},
		Observations: []Observation{{
			Entity:      EntityRef{ID: "device:one", Kind: "device"},
			Attribute:   AttributeRef{Namespace: "state", Name: "active", ValueType: ValueBoolean},
			Status:      StatusObserved,
			Value:       &Value{Boolean: boolPointer(true)},
			Source:      &SourceRef{ID: "sensor:one", Kind: "sensor"},
			Provenance:  "source-record-1",
			EvidenceRef: "evidence-1",
			ObservedAt:  observedAt,
		}},
	}
}

func numberBatch(value float64) Batch {
	event := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	batch := minimalBatch(event, event)
	batch.Observations[0].Attribute = AttributeRef{Namespace: "environment", Name: "temperature", ValueType: ValueNumber, Unit: "Cel"}
	batch.Observations[0].Value = &Value{Number: floatPointer(value)}
	return batch
}

func statusIndex(status SemanticStatus) int {
	for index, candidate := range statusOrder {
		if status == candidate {
			return index
		}
	}
	return -1
}
