package stateencoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMixedInputFixturePreservesTypedStatusTimeAndRelations(t *testing.T) {
	batch := readFixture(t, "testdata/v1/mixed-input.json")
	encoded, err := Encode(batch)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.SchemaVersion != EncoderSchemaVersion || encoded.EncoderVersion != EncoderVersion {
		t.Fatalf("unexpected encoder version: %+v", encoded)
	}
	if encoded.EventTime.Equal(encoded.ReceivedAt) {
		t.Fatal("event_time and received_at were collapsed")
	}
	if encoded.FeatureNames != CanonicalFeatureNames() {
		t.Fatal("feature order changed")
	}
	if len(encoded.Entities) != 4 || encoded.Entities[0].ID != "building:home" {
		t.Fatalf("entities were not sorted canonically: %+v", encoded.Entities)
	}
	if len(encoded.Observations) != 9 || len(encoded.Relations) != 2 {
		t.Fatalf("unexpected encoded counts: observations=%d relations=%d", len(encoded.Observations), len(encoded.Relations))
	}

	byAttr := make(map[string][]EncodedObservation)
	for _, observation := range encoded.Observations {
		byAttr[observation.Attribute.Namespace+"."+observation.Attribute.Name] = append(byAttr[observation.Attribute.Namespace+"."+observation.Attribute.Name], observation)
	}
	observed := byAttr["environment.temperature"][0]
	if observed.Number == nil || *observed.Number != 21.5 || observed.Attribute.Unit != "Cel" || observed.StatusBits[0] != 1 || observed.TypeBits[0] != 1 {
		t.Fatalf("numeric observation was not represented explicitly: %+v", observed)
	}
	if observed.AgeSeconds == nil || *observed.AgeSeconds != 2 {
		t.Fatalf("age was not derived from event_time and observed_at: %+v", observed.AgeSeconds)
	}
	boolean := byAttr["motion.detected"][0]
	if boolean.Boolean == nil || !*boolean.Boolean || boolean.StatusBits[0] != 1 || boolean.TypeBits[1] != 1 {
		t.Fatalf("boolean observation was not represented explicitly: %+v", boolean)
	}
	if observed.ObservedAt == nil || !observed.ObservedAt.Equal(time.Date(2026, 1, 1, 11, 59, 58, 0, time.UTC)) {
		t.Fatalf("observed_at changed: %s", observed.ObservedAt)
	}
	for attribute, statusIndex := range map[string]int{
		"access.contact":       1,
		"power.battery":        2,
		"device.mode":          3,
		"environment.humidity": 4,
		"access.state":         5,
		"motion.occupancy":     6,
	} {
		values := byAttr[attribute]
		if len(values) == 0 {
			t.Fatalf("missing attribute %s", attribute)
		}
		for _, value := range values {
			if value.StatusBits[statusIndex] != 1 {
				t.Errorf("%s did not preserve status %q: %v", attribute, value.Status, value.StatusBits)
			}
		}
	}
	if len(byAttr["access.state"]) != 2 || byAttr["access.state"][0].Category == nil || byAttr["access.state"][1].Category == nil {
		t.Fatalf("source conflict candidates were not preserved: %+v", byAttr["access.state"])
	}
	if encoded.Relations[0].Subject != "room:entry" || encoded.Relations[0].Object != "device:door-front" {
		t.Fatalf("relation direction was changed: %+v", encoded.Relations[0])
	}
}

func TestEncodingIsDeterministicAndDoesNotMutateInput(t *testing.T) {
	batch := readFixture(t, "testdata/v1/mixed-input.json")
	before, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Encode(batch)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("Encode mutated the caller's input")
	}

	permuted := cloneBatch(batch)
	for left, right := 0, len(permuted.Entities)-1; left < right; left, right = left+1, right-1 {
		permuted.Entities[left], permuted.Entities[right] = permuted.Entities[right], permuted.Entities[left]
	}
	for left, right := 0, len(permuted.Observations)-1; left < right; left, right = left+1, right-1 {
		permuted.Observations[left], permuted.Observations[right] = permuted.Observations[right], permuted.Observations[left]
	}
	for left, right := 0, len(permuted.Relations)-1; left < right; left, right = left+1, right-1 {
		permuted.Relations[left], permuted.Relations[right] = permuted.Relations[right], permuted.Relations[left]
	}
	second, err := Encode(permuted)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("permuting input order changed canonical encoded output")
	}
}

func TestInvalidInputsFailClosed(t *testing.T) {
	invalidFixture := readFixture(t, "testdata/v1/missing-entity-id.json")
	cases := map[string]func(*Batch){
		"schema mismatch": func(batch *Batch) { batch.SchemaVersion = "synora.observation-batch/v2" },
		"unknown entity":  func(batch *Batch) { batch.Observations[0].Entity.ID = "device:missing" },
		"unsupported value type": func(batch *Batch) {
			batch.Observations[0].Attribute.ValueType = ValueType("vector")
		},
		"unknown relation node": func(batch *Batch) {
			batch.Relations = []Relation{{Subject: "room:entry", Kind: "contains", Object: "device:missing"}}
		},
		"observed without value": func(batch *Batch) { batch.Observations[0].Status = StatusObserved },
		"unknown never coerced to false": func(batch *Batch) {
			batch.Observations[0].Status = StatusUnknown
			batch.Observations[0].Value = &Value{Boolean: boolPointer(false)}
		},
		"nonfinite number": func(batch *Batch) {
			batch.Entities[0] = EntityRef{ID: "device:valid", Kind: "device"}
			batch.Observations[0].Entity = batch.Entities[0]
			batch.Observations[0].Attribute = AttributeRef{Namespace: "environment", Name: "temperature", ValueType: ValueNumber, Unit: "Cel"}
			batch.Observations[0].Status = StatusObserved
			batch.Observations[0].ObservedAt = batch.EventTime
			batch.Observations[0].Value = &Value{Number: floatPointer(math.NaN())}
		},
		"duplicate observation": func(batch *Batch) {
			batch.Entities[0] = EntityRef{ID: "device:valid", Kind: "device"}
			batch.Observations[0].Entity = batch.Entities[0]
			batch.Observations = append(batch.Observations, batch.Observations[0])
		},
		"bound overflow": func(batch *Batch) {
			batch.Entities = make([]EntityRef, MaxEntities+1)
			for i := range batch.Entities {
				batch.Entities[i] = EntityRef{ID: "device:item-" + string(rune('a'+i%26)) + ":" + string(rune('A'+i/26)), Kind: "device"}
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			batch := cloneBatch(invalidFixture)
			batch.Entities[0] = EntityRef{ID: "device:valid", Kind: "device"}
			batch.Observations[0].Entity = batch.Entities[0]
			batch.Observations[0].Attribute = AttributeRef{Namespace: "access", Name: "contact", ValueType: ValueBoolean}
			batch.Observations[0].Status = StatusUnknown
			mutate(&batch)
			if _, err := Encode(batch); !errors.Is(err, ErrInvalidBatch) {
				t.Fatalf("expected ErrInvalidBatch, got %v", err)
			}
		})
	}
	if _, err := Encode(invalidFixture); !errors.Is(err, ErrInvalidBatch) {
		t.Fatalf("namespaceless fixture id accepted: %v", err)
	}
}

func TestOverflowIsRejectedWithoutTruncation(t *testing.T) {
	batch := readFixture(t, "testdata/v1/mixed-input.json")
	for len(batch.Observations) <= MaxObservations {
		copy := batch.Observations[0]
		copy.Attribute.Name = "item-" + strconv.Itoa(len(batch.Observations))
		copy.Source.ID = "source:item-" + strconv.Itoa(len(batch.Observations))
		batch.Observations = append(batch.Observations, copy)
	}
	if _, err := Encode(batch); !errors.Is(err, ErrInvalidBatch) {
		t.Fatalf("overflow input was not rejected: %v", err)
	}
}

func readFixture(t *testing.T, path string) Batch {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var batch Batch
	if err := json.Unmarshal(data, &batch); err != nil {
		t.Fatal(err)
	}
	return batch
}

func cloneBatch(batch Batch) Batch {
	data, _ := json.Marshal(batch)
	var cloned Batch
	_ = json.Unmarshal(data, &cloned)
	return cloned
}

func boolPointer(value bool) *bool        { return &value }
func floatPointer(value float64) *float64 { return &value }
