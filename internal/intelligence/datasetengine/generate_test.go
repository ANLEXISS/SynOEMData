package datasetengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"synora/internal/intelligence/stateencoder"
)

func TestScenarioCatalogAndGenerationAreDeterministic(t *testing.T) {
	if got := len(ScenarioCatalog()); got != 20 {
		t.Fatalf("scenario catalog has %d entries, want 20", got)
	}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	config := Config{Seed: 8142, HomeCount: 6, StartTime: start}
	first, err := Generate(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("generated dataset failed validation: %v", err)
	}
	assertJSONEqual(t, first.Manifest, second.Manifest)
	assertJSONEqual(t, first.Homes, second.Homes)
	assertJSONEqual(t, first.GroundTruth, second.GroundTruth)
	assertJSONEqual(t, first.SourceEmissions, second.SourceEmissions)
	assertJSONEqual(t, first.Availability, second.Availability)
	assertJSONEqual(t, first.Inputs, second.Inputs)
	assertJSONEqual(t, first.Targets, second.Targets)
	assertJSONEqual(t, first.Index, second.Index)
	if first.Manifest.ExampleCount == 0 || first.Manifest.ExampleCount >= 6*len(ScenarioCatalog()) {
		t.Fatalf("scenario-template holdout was not reflected in generated examples: %d", first.Manifest.ExampleCount)
	}

	homeSplits := map[string]Split{}
	topologySplits := map[string]Split{}
	topologyShapes := map[string]Split{}
	for _, index := range first.Index {
		if split, exists := homeSplits[index.HomeGroupID]; exists && split != index.HomeSplit {
			t.Fatalf("home group %s crosses splits", index.HomeGroupID)
		}
		homeSplits[index.HomeGroupID] = index.HomeSplit
		if split, exists := topologySplits[index.TopologyFamilyID]; exists && split != index.HomeSplit {
			t.Fatalf("topology family %s crosses splits", index.TopologyFamilyID)
		}
		topologySplits[index.TopologyFamilyID] = index.HomeSplit
	}
	if len(topologySplits) < 3 {
		t.Fatalf("only %d topology families were generated", len(topologySplits))
	}
	for _, home := range first.Homes {
		shape := topologyShapeSignature(home)
		if split, exists := topologyShapes[shape]; exists && split != home.Split {
			t.Fatalf("isomorphic topology signature crosses splits: %s", shape)
		}
		topologyShapes[shape] = home.Split
	}
	for _, scenario := range ScenarioCatalog() {
		if !scenario.HeldOut {
			continue
		}
		found := false
		for _, index := range first.Index {
			if index.ScenarioFamily == scenario.ID && index.ScenarioTemplateSplit == ScenarioSplitHeldOut && index.HomeSplit != SplitTrain {
				found = true
			}
			if index.ScenarioFamily == scenario.ID && index.HomeSplit == SplitTrain {
				t.Fatalf("held-out scenario %s appears in train split", scenario.ID)
			}
		}
		if !found {
			t.Errorf("held-out scenario %s has no held-out examples", scenario.ID)
		}
	}
}

func TestCausalCutoffAndScenarioSemantics(t *testing.T) {
	dataset, err := Generate(Config{Seed: 73, HomeCount: 6, StartTime: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	truthByID := indexTruth(dataset.GroundTruth)
	sourceByID := indexSources(dataset.SourceEmissions)
	inputByID := indexInputs(dataset.Inputs)
	targetByID := indexTargets(dataset.Targets)

	for _, index := range dataset.Index {
		input := inputByID[index.ExampleID]
		truth := truthByID[index.ExampleID]
		source := sourceByID[index.ExampleID]
		target := targetByID[index.ExampleID]
		available := stringSet(input.AvailableEmissionIDs)
		for _, emission := range source.Emissions {
			selected := !emission.ReceivedAt.After(index.Cutoff)
			_, inInput := available[emission.EmissionID]
			if selected != inInput {
				t.Fatalf("cutoff selection mismatch for %s emission %s", index.ScenarioFamily, emission.EmissionID)
			}
		}
		modelInput, _ := json.Marshal(ModelInput{Batch: input.Batch, Encoded: input.Encoded, Availability: input.Availability})
		for _, scenario := range ScenarioCatalog() {
			if bytes.Contains(modelInput, []byte(scenario.ID)) {
				t.Fatalf("scenario ID %s leaked into model input", scenario.ID)
			}
		}
		if bytes.Contains(modelInput, []byte("ground_truth")) || bytes.Contains(modelInput, []byte("target")) {
			t.Fatalf("hidden artifact field leaked into model input for %s", index.ExampleID)
		}

		switch index.ScenarioFamily {
		case ScenarioDelayedObservation, ScenarioNewDevice:
			for _, emission := range source.Emissions {
				if emission.ReceivedAt.After(index.Cutoff) {
					if _, found := available[emission.EmissionID]; found {
						t.Fatalf("late emission entered input for %s", index.ScenarioFamily)
					}
				}
			}
		case ScenarioClockDrift:
			if len(input.Encoded.Observations) != 1 || input.Encoded.Observations[0].AgeSeconds == nil || *input.Encoded.Observations[0].AgeSeconds >= 0 {
				t.Fatalf("future source time was not preserved with negative age: %+v", input.Encoded.Observations)
			}
			driftSeconds, err := strconv.ParseInt(truth.Annotations["source_clock_offset_seconds"], 10, 64)
			if err != nil || int64(-*input.Encoded.Observations[0].AgeSeconds) != driftSeconds {
				t.Fatalf("hidden clock offset does not reproduce emitted timestamp: %q", truth.Annotations["source_clock_offset_seconds"])
			}
		case ScenarioConflictingSources:
			if len(input.Encoded.Observations) != 2 || labelValue(target, TaskObservationConsistency) != "inconsistent" {
				t.Fatalf("conflicting observations or task E target missing: input=%+v target=%+v", input.Encoded.Observations, target.Labels)
			}
		case ScenarioSensorUnavailable:
			found := false
			for _, observation := range input.Encoded.Observations {
				found = found || observation.Status == "unavailable"
			}
			if !found {
				t.Fatal("unavailable status was not preserved")
			}
		case ScenarioSensorRecovery:
			statuses := map[string]bool{}
			for _, observation := range input.Encoded.Observations {
				statuses[string(observation.Status)] = true
			}
			if !statuses["unavailable"] || !statuses["observed"] {
				t.Fatalf("recovery interval statuses missing: %v", statuses)
			}
		case ScenarioModifiedTopology:
			if len(truth.FutureEdges) == 0 {
				t.Fatal("future topology edge missing from ground truth")
			}
			for _, future := range truth.FutureEdges {
				for _, observed := range input.Batch.Relations {
					if future.Subject == observed.Subject && future.Object == observed.Object && future.Kind == observed.Kind && future.Source != nil && observed.Source != nil && future.Source.ID == observed.Source.ID {
						t.Fatal("future topology relation leaked into model input")
					}
				}
			}
		case ScenarioDailyRoutine:
			if len(input.Batch.Observations) != 2 || input.Batch.Observations[0].Value == nil || input.Batch.Observations[1].Value == nil ||
				input.Batch.Observations[0].Value.Boolean == nil || input.Batch.Observations[1].Value.Boolean == nil {
				t.Fatalf("daily routine needs two typed observations: %+v", input.Batch.Observations)
			}
			deviates := *input.Batch.Observations[0].Value.Boolean != *input.Batch.Observations[1].Value.Boolean
			if (labelValue(target, TaskBehaviorAnomaly) == "true") != deviates {
				t.Fatalf("daily-routine anomaly label does not match routine deviation")
			}
		}
		if len(input.Availability) != len(available) {
			t.Fatalf("available source timestamps are not preserved for %s", index.ScenarioFamily)
		}
	}

	positiveAnomaly, negativeAnomaly := false, false
	for _, target := range dataset.Targets {
		value := labelValue(target, TaskBehaviorAnomaly)
		positiveAnomaly = positiveAnomaly || value == "true"
		negativeAnomaly = negativeAnomaly || value == "false"
	}
	if !positiveAnomaly || !negativeAnomaly {
		t.Fatalf("anomaly labels need positive and negative examples: positive=%t negative=%t", positiveAnomaly, negativeAnomaly)
	}
	for _, split := range []Split{SplitTrain, SplitValidation, SplitTest} {
		for _, task := range allTasks() {
			count := 0
			for key, value := range dataset.Manifest.Counts {
				if strings.HasPrefix(key, "label/"+string(split)+"/"+string(task)+"/") {
					count += value
				}
			}
			if count == 0 {
				t.Fatalf("manifest lacks class distribution for %s/%s", split, task)
			}
		}
		for _, label := range []string{"D-behavior-anomaly/true", "D-behavior-anomaly/false", "C-topology-transition/edge_added", "E-observation-consistency/inconsistent"} {
			if dataset.Manifest.Counts["label/"+string(split)+"/"+label] == 0 {
				t.Fatalf("split %s lacks required class %s", split, label)
			}
		}
	}
}

func TestSelectAvailableUsesReceiptTimeAndCopiesValues(t *testing.T) {
	cutoff := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	oldSourceTime := cutoff.Add(-time.Hour)
	late := SourceEmission{EmissionID: "emission:late", ObservedAt: oldSourceTime, ReceivedAt: cutoff.Add(time.Second), Value: &stateencoder.Value{Boolean: boolPtr(true)}}
	available := SourceEmission{EmissionID: "emission:available", ObservedAt: cutoff.Add(time.Hour), ReceivedAt: cutoff, Value: &stateencoder.Value{Boolean: boolPtr(false)}}
	selected := SelectAvailable([]SourceEmission{late, available}, cutoff)
	if len(selected) != 1 || selected[0].EmissionID != available.EmissionID {
		t.Fatalf("availability selection used event/observation time instead of receipt: %+v", selected)
	}
	*selected[0].Value.Boolean = true
	if *available.Value.Boolean {
		t.Fatal("SelectAvailable did not copy nested typed values")
	}
}

func TestArtifactStorageSeparatesModelInputAndHiddenLabels(t *testing.T) {
	dataset, err := Generate(Config{Seed: 914, HomeCount: 6, StartTime: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := dataset.WriteArtifacts(root); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"ground-truth", "source-emissions", "availability", "model-inputs", "targets"} {
		entries, err := os.ReadDir(filepath.Join(root, directory))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != dataset.Manifest.ExampleCount {
			t.Fatalf("%s contains %d files, want %d", directory, len(entries), dataset.Manifest.ExampleCount)
		}
	}
	for _, entry := range mustReadDir(t, filepath.Join(root, "model-inputs")) {
		data, err := os.ReadFile(filepath.Join(root, "model-inputs", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var artifact ModelInputArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			t.Fatal(err)
		}
		if digestJSON(artifact.Data) != artifact.Meta.Digest || artifact.Meta.ExampleID == "" {
			t.Fatalf("model input digest/provenance invalid in %s", entry.Name())
		}
		text := string(data)
		for _, forbidden := range []string{"scenario_id", "ground_truth", "target_time", "labels", "held_out_templates"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("model input contains hidden field %q", forbidden)
			}
		}
	}
	manifest, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil || !json.Valid(manifest) {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
}

func TestInvalidGenerationConfig(t *testing.T) {
	if _, err := Generate(Config{HomeCount: 2}); err == nil {
		t.Fatal("too few homes accepted; group splits require at least six")
	}
	if _, err := Generate(Config{HomeCount: maxHomes + 1}); err == nil {
		t.Fatal("home count above resource cap accepted")
	}
	if _, err := Generate(Config{HomeCount: 6, StartTime: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)}); err == nil {
		t.Fatal("unsupported synthetic time range accepted")
	}
}

func assertJSONEqual(t *testing.T, left, right any) {
	t.Helper()
	a, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("same configuration and seeds generated different artifacts")
	}
}

func indexTruth(values []GroundTruthArtifact) map[string]GroundTruthArtifact {
	result := make(map[string]GroundTruthArtifact, len(values))
	for _, value := range values {
		result[value.Meta.ExampleID] = value
	}
	return result
}
func indexSources(values []SourceArtifact) map[string]SourceArtifact {
	result := make(map[string]SourceArtifact, len(values))
	for _, value := range values {
		result[value.Meta.ExampleID] = value
	}
	return result
}
func indexInputs(values []InputArtifact) map[string]InputArtifact {
	result := make(map[string]InputArtifact, len(values))
	for _, value := range values {
		result[value.Meta.ExampleID] = value
	}
	return result
}
func indexTargets(values []TargetArtifact) map[string]TargetArtifact {
	result := make(map[string]TargetArtifact, len(values))
	for _, value := range values {
		result[value.Meta.ExampleID] = value
	}
	return result
}
func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
func labelValue(target TargetArtifact, id TaskID) string {
	for _, label := range target.Labels {
		if label.TaskID == id {
			return label.Value
		}
	}
	return ""
}
func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
func boolPtr(value bool) *bool { return &value }

func topologyShapeSignature(home HomeArtifact) string {
	entityCount := 0
	for _, entity := range home.Entities {
		if entity.Kind == "room" {
			entityCount++
		}
	}
	outDegree, inDegree := map[string]int{}, map[string]int{}
	for _, relation := range home.Topology {
		if relation.Kind == "adjacent-to" {
			outDegree[relation.Subject]++
			inDegree[relation.Object]++
		}
	}
	out, in := make([]int, 0, entityCount), make([]int, 0, entityCount)
	for _, entity := range home.Entities {
		if entity.Kind != "room" {
			continue
		}
		out = append(out, outDegree[entity.ID])
		in = append(in, inDegree[entity.ID])
	}
	sort.Ints(out)
	sort.Ints(in)
	return fmt.Sprintf("rooms=%d/out=%v/in=%v", entityCount, out, in)
}
