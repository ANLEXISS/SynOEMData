package datasetengine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"synora/internal/intelligence/stateencoder"
)

const (
	maxHomes      = 256
	minHomes      = 6
	defaultStart  = "2026-01-01T00:00:00Z"
	observationNS = "occupancy"
)

var ErrInvalidConfig = errors.New("invalid dataset configuration")

type homePlan struct {
	groupID       string
	familyID      string
	familyIndex   int
	familyOrdinal int
	split         Split
	entities      []stateencoder.EntityRef
	rooms         []stateencoder.EntityRef
	sensors       []stateencoder.EntityRef
	topology      []stateencoder.Relation
	seedLineage   map[string]uint64
}

type exampleArtifacts struct {
	truth        GroundTruthArtifact
	sources      SourceArtifact
	availability AvailabilityArtifact
	input        InputArtifact
	targets      TargetArtifact
	index        ExampleIndex
}

func Generate(config Config) (Dataset, error) {
	if config.HomeCount == 0 {
		config.HomeCount = DefaultHomeCount
	}
	if config.HomeCount < minHomes || config.HomeCount > maxHomes {
		return Dataset{}, fmt.Errorf("%w: home_count must be between %d and %d", ErrInvalidConfig, minHomes, maxHomes)
	}
	if config.StartTime.IsZero() {
		parsed, _ := time.Parse(time.RFC3339, defaultStart)
		config.StartTime = parsed
	}
	if config.StartTime.Year() < 2000 || config.StartTime.Year() > 2200 {
		return Dataset{}, fmt.Errorf("%w: start_time year must be between 2000 and 2200", ErrInvalidConfig)
	}

	familyCount := config.HomeCount / 2
	if familyCount > 6 {
		familyCount = 6
	}
	if familyCount < 3 {
		familyCount = 3
	}
	familySplits := splitFamilies(familyCount, config.Seed)
	topologyTemplateSeed := streamSeed(config.Seed, "topology-templates")
	topologyTemplateOrder := rand.New(rand.NewSource(int64(topologyTemplateSeed))).Perm(6)
	dataset := Dataset{
		Manifest: DatasetManifest{
			SchemaVersion: DatasetSchemaVersion,
			Generator:     GeneratorVersion,
			Seed:          config.Seed,
			GeneratedAt:   config.StartTime,
			Config:        config,
			ScenarioIDs:   make([]ScenarioID, 0, len(catalog)),
			HomeCount:     config.HomeCount,
			Counts:        map[string]int{},
			SeedLineage:   map[string]uint64{"topology_templates": topologyTemplateSeed, "split": streamSeed(config.Seed, "split-families")},
		},
	}
	for _, scenario := range catalog {
		dataset.Manifest.ScenarioIDs = append(dataset.Manifest.ScenarioIDs, scenario.ID)
	}

	for homeIndex := 0; homeIndex < config.HomeCount; homeIndex++ {
		familyIndex := homeIndex % familyCount
		home := buildHome(config.Seed, homeIndex, familyIndex, topologyTemplateOrder[familyIndex], homeIndex/familyCount, familySplits[familyIndex])
		dataset.Homes = append(dataset.Homes, HomeArtifact{
			HomeGroupID: home.groupID, TopologyFamilyID: home.familyID, Split: home.split,
			Entities: cloneEntities(home.entities), Topology: cloneRelations(home.topology), SeedLineage: cloneSeeds(home.seedLineage),
		})
		dataset.Manifest.Counts["homes/"+string(home.split)]++
		scenarioOrder := rand.New(rand.NewSource(int64(streamSeed(config.Seed, "scenario-selection", home.groupID)))).Perm(len(catalog))
		for _, scenarioPosition := range scenarioOrder {
			scenario := catalog[scenarioPosition]
			if home.split == SplitTrain && scenario.HeldOut {
				continue
			}
			artifacts, err := buildExample(config, home, scenario)
			if err != nil {
				return Dataset{}, fmt.Errorf("generate %s for home %s: %w", scenario.ID, home.groupID, err)
			}
			dataset.GroundTruth = append(dataset.GroundTruth, artifacts.truth)
			dataset.SourceEmissions = append(dataset.SourceEmissions, artifacts.sources)
			dataset.Availability = append(dataset.Availability, artifacts.availability)
			dataset.Inputs = append(dataset.Inputs, artifacts.input)
			dataset.Targets = append(dataset.Targets, artifacts.targets)
			dataset.Index = append(dataset.Index, artifacts.index)
			countKey := string(home.split) + "/" + string(scenario.ID)
			dataset.Manifest.Counts[countKey]++
			for _, label := range artifacts.targets.Labels {
				labelCountKey := "label/" + string(home.split) + "/" + string(label.TaskID) + "/" + label.Value
				dataset.Manifest.Counts[labelCountKey]++
			}
		}
	}
	dataset.Manifest.ExampleCount = len(dataset.Index)
	sort.Slice(dataset.Index, func(i, j int) bool { return dataset.Index[i].ExampleID < dataset.Index[j].ExampleID })
	sort.Slice(dataset.Homes, func(i, j int) bool { return dataset.Homes[i].HomeGroupID < dataset.Homes[j].HomeGroupID })
	if err := dataset.Validate(); err != nil {
		return Dataset{}, fmt.Errorf("validate generated dataset: %w", err)
	}
	return dataset, nil
}

// Validate checks artifact separation, provenance links, causal cutoffs, and
// home/topology split boundaries before a dataset is consumed or persisted.
func (d Dataset) Validate() error {
	if d.Manifest.SchemaVersion != DatasetSchemaVersion || d.Manifest.Generator != GeneratorVersion {
		return fmt.Errorf("unsupported dataset or generator version")
	}
	if d.Manifest.HomeCount != len(d.Homes) || d.Manifest.ExampleCount != len(d.Index) {
		return fmt.Errorf("manifest counts do not match artifacts")
	}
	if len(d.Index) != len(d.GroundTruth) || len(d.Index) != len(d.SourceEmissions) || len(d.Index) != len(d.Availability) || len(d.Index) != len(d.Inputs) || len(d.Index) != len(d.Targets) {
		return fmt.Errorf("artifact collections have inconsistent example counts")
	}
	familySplit := map[string]Split{}
	homeSplit := map[string]Split{}
	for _, home := range d.Homes {
		if existing, ok := familySplit[home.TopologyFamilyID]; ok && existing != home.Split {
			return fmt.Errorf("topology family %s crosses home splits", home.TopologyFamilyID)
		}
		familySplit[home.TopologyFamilyID] = home.Split
		if existing, ok := homeSplit[home.HomeGroupID]; ok && existing != home.Split {
			return fmt.Errorf("home group %s crosses splits", home.HomeGroupID)
		}
		homeSplit[home.HomeGroupID] = home.Split
	}
	indexByID := make(map[string]ExampleIndex, len(d.Index))
	for _, index := range d.Index {
		if !isArtifactID(index.ExampleID) || index.ScenarioFamily == "" || index.HomeGroupID == "" {
			return fmt.Errorf("example index has an empty required identity")
		}
		if _, duplicate := indexByID[index.ExampleID]; duplicate {
			return fmt.Errorf("duplicate example id %s", index.ExampleID)
		}
		if homeSplit[index.HomeGroupID] != index.HomeSplit || familySplit[index.TopologyFamilyID] != index.HomeSplit {
			return fmt.Errorf("home/topology split mismatch for example %s", index.ExampleID)
		}
		if index.HomeSplit == SplitTrain && index.ScenarioTemplateSplit == ScenarioSplitHeldOut {
			return fmt.Errorf("held-out scenario template %s appears in training examples", index.ScenarioFamily)
		}
		indexByID[index.ExampleID] = index
	}
	truthByID := make(map[string]GroundTruthArtifact, len(d.GroundTruth))
	sourcesByID := make(map[string]SourceArtifact, len(d.SourceEmissions))
	availabilityByID := make(map[string]AvailabilityArtifact, len(d.Availability))
	inputsByID := make(map[string]InputArtifact, len(d.Inputs))
	targetsByID := make(map[string]TargetArtifact, len(d.Targets))
	for _, artifact := range d.GroundTruth {
		truthByID[artifact.Meta.ExampleID] = artifact
	}
	for _, artifact := range d.SourceEmissions {
		sourcesByID[artifact.Meta.ExampleID] = artifact
	}
	for _, artifact := range d.Availability {
		availabilityByID[artifact.Meta.ExampleID] = artifact
	}
	for _, artifact := range d.Inputs {
		inputsByID[artifact.Meta.ExampleID] = artifact
	}
	for _, artifact := range d.Targets {
		targetsByID[artifact.Meta.ExampleID] = artifact
	}
	for id, index := range indexByID {
		truth, hasTruth := truthByID[id]
		source, hasSource := sourcesByID[id]
		availabilityArtifact, hasAvailability := availabilityByID[id]
		input, hasInput := inputsByID[id]
		target, hasTarget := targetsByID[id]
		if !hasTruth || !hasSource || !hasAvailability || !hasInput || !hasTarget {
			return fmt.Errorf("example %s is missing an artifact stage", id)
		}
		if input.Batch.EventTime != index.Cutoff || input.Batch.ReceivedAt != index.Cutoff {
			return fmt.Errorf("input batch time differs from cutoff for %s", id)
		}
		if !availabilityArtifact.Cutoff.Equal(index.Cutoff) || digestJSON(availabilityArtifact.Records) != digestJSON(input.Availability) {
			return fmt.Errorf("availability stage differs from model input for %s", id)
		}
		availableIDs := make(map[string]struct{}, len(input.AvailableEmissionIDs))
		for _, emissionID := range input.AvailableEmissionIDs {
			if _, duplicate := availableIDs[emissionID]; duplicate {
				return fmt.Errorf("duplicate selected emission id %s", emissionID)
			}
			availableIDs[emissionID] = struct{}{}
		}
		sourceIDs := make(map[string]struct{}, len(source.Emissions))
		for _, emission := range source.Emissions {
			if _, duplicate := sourceIDs[emission.EmissionID]; duplicate {
				return fmt.Errorf("duplicate source emission id %s", emission.EmissionID)
			}
			sourceIDs[emission.EmissionID] = struct{}{}
			_, selected := availableIDs[emission.EmissionID]
			if selected != !emission.ReceivedAt.After(index.Cutoff) {
				return fmt.Errorf("availability cutoff mismatch for emission %s", emission.EmissionID)
			}
		}
		if len(input.Availability) != len(input.AvailableEmissionIDs) {
			return fmt.Errorf("availability metadata count mismatch for %s", id)
		}
		availabilityByID := make(map[string]AvailabilityRecord, len(input.Availability))
		for _, record := range input.Availability {
			if _, duplicate := availabilityByID[record.EmissionID]; duplicate {
				return fmt.Errorf("duplicate availability metadata for emission %s", record.EmissionID)
			}
			if _, ok := availableIDs[record.EmissionID]; !ok || record.ReceivedAt.After(index.Cutoff) {
				return fmt.Errorf("invalid availability metadata for emission %s", record.EmissionID)
			}
			availabilityByID[record.EmissionID] = record
		}
		for _, emissionID := range input.AvailableEmissionIDs {
			record, ok := availabilityByID[emissionID]
			if !ok {
				return fmt.Errorf("missing availability metadata for emission %s", emissionID)
			}
			found := false
			var matched SourceEmission
			for _, emission := range source.Emissions {
				if emission.EmissionID != emissionID {
					continue
				}
				matched = emission
				found = record.ReceivedAt.Equal(emission.ReceivedAt) && record.ObservedAt.Equal(emission.ObservedAt) &&
					record.Entity == emission.Entity && record.Attribute == emission.Attribute
				if found && record.Source != nil && emission.Source != nil {
					found = *record.Source == *emission.Source
				} else if found {
					found = record.Source == nil && emission.Source == nil
				}
				break
			}
			if !found {
				return fmt.Errorf("availability metadata differs from source emission %s: metadata=%+v source=%+v metadataSource=%v sourceSource=%v", emissionID, record, matched, record.Source, matched.Source)
			}
		}
		if len(availabilityArtifact.Meta.ParentDigests) != 1 || availabilityArtifact.Meta.ParentDigests[0] != source.Meta.Digest ||
			len(input.Meta.ParentDigests) != 1 || input.Meta.ParentDigests[0] != availabilityArtifact.Meta.Digest || len(target.Meta.ParentDigests) != 2 ||
			target.Meta.ParentDigests[0] != truth.Meta.Digest || target.Meta.ParentDigests[1] != source.Meta.Digest {
			return fmt.Errorf("artifact provenance mismatch for %s", id)
		}
		if truth.ScenarioID != index.ScenarioFamily || source.ScenarioID != index.ScenarioFamily {
			return fmt.Errorf("scenario provenance mismatch for %s", id)
		}
		if len(target.Labels) != len(allTasks()) {
			return fmt.Errorf("example %s has incomplete task labels", id)
		}
		seenTasks := make(map[TaskID]struct{}, len(target.Labels))
		for _, label := range target.Labels {
			if _, duplicate := seenTasks[label.TaskID]; duplicate {
				return fmt.Errorf("example %s has duplicate task label %s", id, label.TaskID)
			}
			seenTasks[label.TaskID] = struct{}{}
			if !validTaskValue(label.TaskID, label.Value) {
				return fmt.Errorf("example %s has invalid value %q for task %s", id, label.Value, label.TaskID)
			}
			if label.TargetTime.Before(index.Cutoff) || label.AvailableAfter.Before(label.TargetTime) {
				return fmt.Errorf("target %s has invalid horizon/availability for %s", label.TaskID, id)
			}
		}
		for _, task := range allTasks() {
			if _, exists := seenTasks[task]; !exists {
				return fmt.Errorf("example %s has no target for task %s", id, task)
			}
		}
		if artifactDigest(truth) != truth.Meta.Digest || artifactDigest(source) != source.Meta.Digest || artifactDigest(availabilityArtifact) != availabilityArtifact.Meta.Digest ||
			artifactDigest(target) != target.Meta.Digest || digestJSON(ModelInput{Batch: input.Batch, Encoded: input.Encoded, Availability: input.Availability}) != input.Meta.Digest {
			return fmt.Errorf("artifact digest mismatch for %s", id)
		}
	}
	return nil
}

func buildHome(master int64, index, familyIndex, topologyTemplateIndex, familyOrdinal int, split Split) homePlan {
	topologyKind, roomCount := topologyTemplate(topologyTemplateIndex)
	homeKey := shortHash(strconv.FormatInt(master, 10), "home", strconv.Itoa(index))
	familyKey := shortHash(strconv.FormatInt(master, 10), "topology-family", strconv.Itoa(topologyTemplateIndex))
	home := homePlan{
		groupID: "home:" + homeKey, familyID: "topology:" + familyKey, familyIndex: familyIndex, familyOrdinal: familyOrdinal, split: split,
		seedLineage: map[string]uint64{"topology_templates": streamSeed(master, "topology-templates")},
	}
	for room := 0; room < roomCount; room++ {
		ref := stateencoder.EntityRef{ID: "room:" + shortHash(homeKey, "room", strconv.Itoa(room)), Kind: "room"}
		home.rooms = append(home.rooms, ref)
		home.entities = append(home.entities, ref)
		sensor := stateencoder.EntityRef{ID: "sensor:" + shortHash(homeKey, "occupancy", strconv.Itoa(room)), Kind: "sensor"}
		home.sensors = append(home.sensors, sensor)
		home.entities = append(home.entities, sensor)
		home.topology = append(home.topology, stateencoder.Relation{Subject: ref.ID, Kind: "observed-by", Object: sensor.ID,
			Source: &stateencoder.SourceRef{ID: "source:" + shortHash(homeKey, "inventory"), Kind: "inventory"}})
	}
	for from, room := range home.rooms {
		var to int
		switch topologyKind {
		case 0: // directed ring
			to = (from + 1) % roomCount
		case 1: // directed chain, with the final room isolated from outgoing edges
			to = from + 1
			if to >= roomCount {
				continue
			}
		default: // directed star
			if from == 0 {
				continue
			}
			to = 0
		}
		home.topology = append(home.topology, stateencoder.Relation{Subject: room.ID, Kind: "adjacent-to", Object: home.rooms[to].ID,
			Source: &stateencoder.SourceRef{ID: "source:" + shortHash(homeKey, "floorplan"), Kind: "floorplan"}})
	}
	return home
}

// topologyTemplate gives each held-out family a distinct graph size or motif.
// Repeated homes in one family are graph-isomorphic but have disjoint IDs.
func topologyTemplate(familyIndex int) (kind, roomCount int) {
	switch familyIndex % 6 {
	case 0:
		return 0, 2 // ring
	case 1:
		return 0, 3 // ring
	case 2:
		return 1, 3 // chain
	case 3:
		return 1, 4 // chain
	case 4:
		return 2, 4 // star
	default:
		return 2, 5 // star
	}
}

func buildExample(config Config, home homePlan, scenario ScenarioDefinition) (exampleArtifacts, error) {
	cutoffHour := 1 + int(streamSeed(config.Seed, "cutoff", home.groupID, string(scenario.ID))%72)
	cutoff := config.StartTime.Add(time.Duration(cutoffHour) * time.Hour)
	exampleID := shortHash(strconv.FormatInt(config.Seed, 10), home.groupID, string(scenario.ID))
	lineage := map[string]uint64{
		"schedule":           streamSeed(config.Seed, "schedule", home.groupID, string(scenario.ID)),
		"source_behavior":    streamSeed(config.Seed, "source_behavior", home.groupID, string(scenario.ID)),
		"fault_injection":    streamSeed(config.Seed, "fault_injection", home.groupID, string(scenario.ID)),
		"clock_drift":        streamSeed(config.Seed, "clock_drift", home.groupID, string(scenario.ID)),
		"network_delay":      streamSeed(config.Seed, "network_delay", home.groupID, string(scenario.ID)),
		"scenario_selection": streamSeed(config.Seed, "scenario_selection", home.groupID, string(scenario.ID)),
	}
	scheduleRNG := rand.New(rand.NewSource(int64(lineage["schedule"])))
	sourceRNG := rand.New(rand.NewSource(int64(lineage["source_behavior"])))
	faultRNG := rand.New(rand.NewSource(int64(lineage["fault_injection"])))
	clockRNG := rand.New(rand.NewSource(int64(lineage["clock_drift"])))
	delayRNG := rand.New(rand.NewSource(int64(lineage["network_delay"])))
	room := home.rooms[scheduleRNG.Intn(len(home.rooms))]
	sensor := home.sensors[0]
	attribute := stateencoder.AttributeRef{Namespace: observationNS, Name: "present", ValueType: stateencoder.ValueBoolean}
	before := sourceRNG.Intn(2) == 1
	after := before
	faultDuration := time.Duration(30+faultRNG.Intn(90)) * time.Second
	if scenario.ID == ScenarioResidentArrival || scenario.ID == ScenarioTemporaryGuest || scenario.ID == ScenarioUnexpectedPresence {
		after = true
	} else if scenario.ID == ScenarioResidentDeparture {
		after = false
	} else if scenario.ID == ScenarioRoutineChange || scenario.ID == ScenarioOutOfOrder || scenario.ID == ScenarioClockDrift || scenario.ID == ScenarioDoorOccupancy || scenario.ID == ScenarioLighting {
		after = !before
	}
	observedAt := cutoff.Add(-10 * time.Second)
	receivedAt := cutoff.Add(-5 * time.Second)
	value := &stateencoder.Value{Boolean: boolPointer(before)}
	primary := SourceEmission{
		EmissionID: "emission:" + shortHash(home.groupID, sensor.ID, observedAt.Format(time.RFC3339Nano), "source-record"),
		Entity:     sensor, Attribute: attribute, Status: stateencoder.StatusObserved, Value: value,
		Source:     &stateencoder.SourceRef{ID: "source:" + shortHash(home.groupID, sensor.ID), Kind: "occupancy"},
		Provenance: "record:" + shortHash(sensor.ID, observedAt.Format(time.RFC3339Nano)), ObservedAt: observedAt, ReceivedAt: receivedAt,
	}
	emissions := []SourceEmission{primary}
	inputEntities := cloneEntities(home.entities)
	inputRelations := cloneRelations(home.topology)
	futureEdges := []stateencoder.Relation(nil)
	scenarioAnnotations := map[string]string{}
	consistent := true
	anomaly := scenario.ID == ScenarioRoutineChange || scenario.ID == ScenarioUnexpectedPresence
	if scenario.ID == ScenarioDailyRoutine {
		anomaly = home.familyOrdinal%2 == 0 // Balanced routine deviations across replicated homes in each topology family.
		scenarioAnnotations["routine_deviation"] = strconv.FormatBool(anomaly)
	}

	switch scenario.ID {
	case ScenarioDailyRoutine:
		baseline := primary
		baseline.ObservedAt = cutoff.Add(-5 * time.Minute)
		baseline.ReceivedAt = cutoff.Add(-4 * time.Minute)
		baseline.EmissionID = "emission:" + shortHash(home.groupID, sensor.ID, baseline.ObservedAt.Format(time.RFC3339Nano), "source-record")
		baseline.Provenance = "record:" + shortHash(sensor.ID, baseline.ObservedAt.Format(time.RFC3339Nano))
		primary.Value = &stateencoder.Value{Boolean: boolPointer(before != anomaly)}
		emissions = []SourceEmission{baseline, primary}
	case ScenarioMultipleOccupants:
		trueCount := 1 + sourceRNG.Intn(2)
		after = trueCount > 1
	case ScenarioSensorUnavailable:
		unavailable := primary
		unavailable.Status, unavailable.Value, unavailable.ObservedAt = stateencoder.StatusUnavailable, nil, time.Time{}
		emissions = []SourceEmission{unavailable, makeDoorEmission(home, cutoff, false)}
	case ScenarioSensorRecovery:
		scenarioAnnotations["outage_duration_seconds"] = strconv.FormatInt(int64(faultDuration/time.Second), 10)
		emissions = []SourceEmission{
			makeUnavailableEmission(primary, cutoff.Add(-faultDuration), cutoff.Add(-faultDuration+10*time.Second)),
			primary,
		}
	case ScenarioDelayedObservation:
		late := primary
		deliveryDelay := 30 + delayRNG.Intn(31)
		scenarioAnnotations["delivery_delay_seconds"] = strconv.Itoa(deliveryDelay)
		late.ReceivedAt = cutoff.Add(time.Duration(deliveryDelay) * time.Second)
		late.Provenance = "record:" + shortHash(sensor.ID, observedAt.Format(time.RFC3339Nano))
		emissions = []SourceEmission{makeDoorEmission(home, cutoff, before), late}
	case ScenarioOutOfOrder:
		older := primary
		older.ObservedAt = cutoff.Add(-30 * time.Second)
		older.ReceivedAt = cutoff.Add(-4 * time.Second)
		older.EmissionID = "emission:" + shortHash(home.groupID, sensor.ID, older.ObservedAt.Format(time.RFC3339Nano), "source-record")
		older.Provenance = "record:" + shortHash(sensor.ID, older.ObservedAt.Format(time.RFC3339Nano))
		newer := primary
		newer.ObservedAt = cutoff.Add(-20 * time.Second)
		newer.ReceivedAt = cutoff.Add(-8 * time.Second)
		newer.EmissionID = "emission:" + shortHash(home.groupID, sensor.ID, newer.ObservedAt.Format(time.RFC3339Nano), "source-record")
		newer.Value = &stateencoder.Value{Boolean: boolPointer(!before)}
		newer.Provenance = "record:" + shortHash(sensor.ID, newer.ObservedAt.Format(time.RFC3339Nano))
		emissions = []SourceEmission{newer, older}
	case ScenarioClockDrift:
		drift := time.Duration(1+clockRNG.Intn(60)) * time.Second
		scenarioAnnotations["source_clock_offset_seconds"] = strconv.FormatInt(int64(drift/time.Second), 10)
		primary.ObservedAt = cutoff.Add(drift)
		primary.Provenance = "record:" + shortHash(sensor.ID, observedAt.Format(time.RFC3339Nano))
		emissions = []SourceEmission{primary}
	case ScenarioConflictingSources:
		scenarioAnnotations["conflicting_source_count"] = "2"
		conflict := primary
		conflict.Source = &stateencoder.SourceRef{ID: "source:" + shortHash(home.groupID, sensor.ID, "secondary"), Kind: "occupancy"}
		conflict.EmissionID = "emission:" + shortHash(home.groupID, sensor.ID, conflict.Source.ID, observedAt.Format(time.RFC3339Nano), "source-record")
		conflict.Value = &stateencoder.Value{Boolean: boolPointer(!before)}
		conflict.Provenance = "record:" + shortHash(sensor.ID, "secondary", observedAt.Format(time.RFC3339Nano))
		emissions = append(emissions, conflict)
		consistent = false
	case ScenarioMissingEvent:
		emissions = []SourceEmission{makeDoorEmission(home, cutoff, before)}
	case ScenarioNewDevice:
		newDevice := stateencoder.EntityRef{ID: "device:" + shortHash(home.groupID, "device", "1"), Kind: "device"}
		newObservation := SourceEmission{
			EmissionID: "emission:" + shortHash(home.groupID, newDevice.ID, cutoff.Format(time.RFC3339Nano), "source-record"), Entity: newDevice,
			Attribute: stateencoder.AttributeRef{Namespace: "device", Name: "online", ValueType: stateencoder.ValueBoolean},
			Status:    stateencoder.StatusObserved, Value: &stateencoder.Value{Boolean: boolPointer(true)},
			Source:     &stateencoder.SourceRef{ID: "source:" + shortHash(newDevice.ID), Kind: "device"},
			Provenance: "record:" + shortHash(newDevice.ID, cutoff.Format(time.RFC3339Nano)), ObservedAt: cutoff.Add(5 * time.Second), ReceivedAt: cutoff.Add(10 * time.Second),
		}
		emissions = []SourceEmission{makeDoorEmission(home, cutoff, before), newObservation}
		inputEntities = withoutEntity(inputEntities, newDevice.ID)
	case ScenarioModifiedTopology:
		futureEdges = append(futureEdges, stateencoder.Relation{Subject: home.rooms[0].ID, Kind: "adjacent-to", Object: home.rooms[len(home.rooms)-1].ID,
			Source: &stateencoder.SourceRef{ID: "source:" + shortHash(home.groupID, "future-floorplan"), Kind: "floorplan"}, ValidFrom: cutoff.Add(time.Minute)})
		scenarioAnnotations["future_edge_count"] = strconv.Itoa(len(futureEdges))
		after = !before
	case ScenarioUnseenHomeTopology:
		// The topology-family holdout assignment is carried only in the example index.
		// No template label is added to the model-visible input.
	}

	if scenario.ID == ScenarioLighting {
		primary.Attribute = stateencoder.AttributeRef{Namespace: "lighting", Name: "on", ValueType: stateencoder.ValueBoolean}
		primary.Entity = stateencoder.EntityRef{ID: "device:" + shortHash(home.groupID, "light"), Kind: "device"}
		primary.Source = &stateencoder.SourceRef{ID: "source:" + shortHash(home.groupID, "light"), Kind: "lighting"}
		primary.Value = &stateencoder.Value{Boolean: boolPointer(before)}
		emissions[0] = primary
		inputEntities = append(inputEntities, primary.Entity)
	}
	if scenario.ID == ScenarioDoorOccupancy {
		emissions = append(emissions, makeDoorEmission(home, cutoff, !before))
	}
	if scenario.ID == ScenarioMultipleOccupants {
		after = sourceRNG.Intn(2) == 1
	}

	if scenario.ID == ScenarioSensorUnavailable || scenario.ID == ScenarioSensorRecovery {
		after = !before
	}
	if scenario.ID == ScenarioConflictingSources {
		after = before
	}
	if scenario.ID == ScenarioRoomMovement {
		after = !before
	}

	if scenario.ID == ScenarioLighting {
		room = stateencoder.EntityRef{ID: "device:" + shortHash(home.groupID, "light"), Kind: "device"}
		attribute = stateencoder.AttributeRef{Namespace: "lighting", Name: "on", ValueType: stateencoder.ValueBoolean}
	}
	truth := GroundTruthArtifact{
		Meta: artifactMeta(exampleID), ScenarioID: scenario.ID,
		Events: []TruthEvent{
			{At: cutoff.Add(-time.Minute), EntityID: room.ID, Attribute: string(attribute.Namespace) + "." + attribute.Name, Value: strconv.FormatBool(before), Kind: "state"},
			{At: cutoff.Add(time.Minute), EntityID: room.ID, Attribute: string(attribute.Namespace) + "." + attribute.Name, Value: strconv.FormatBool(after), Kind: "state"},
		},
		FutureEdges: futureEdges,
		Annotations: map[string]string{
			"scenario":                   string(scenario.ID),
			"anomaly_is_intrusion":       "false",
			"topology_unseen_from_train": strconv.FormatBool(scenario.ID == ScenarioUnseenHomeTopology && home.split == SplitTest),
		},
	}
	for key, value := range scenarioAnnotations {
		truth.Annotations[key] = value
	}
	truth.Meta.Digest = digestJSON(truth)

	sourceArtifact := SourceArtifact{Meta: artifactMeta(exampleID, truth.Meta.Digest), ScenarioID: scenario.ID, Emissions: emissions}
	sourceArtifact.Meta.Digest = digestJSON(sourceArtifact)

	available := SelectAvailable(emissions, cutoff)
	if len(available) == 0 {
		return exampleArtifacts{}, errors.New("no source emissions are available at prediction cutoff")
	}
	batchEntities := entitiesForAvailable(inputEntities, available, inputRelations)
	observations := make([]stateencoder.Observation, 0, len(available))
	emissionIDs := make([]string, 0, len(available))
	availability := make([]AvailabilityRecord, 0, len(available))
	for _, emission := range available {
		observations = append(observations, stateencoder.Observation{
			Entity: emission.Entity, Attribute: emission.Attribute, Status: emission.Status, Value: cloneValue(emission.Value),
			Source: cloneSource(emission.Source), Provenance: emission.Provenance, ObservedAt: emission.ObservedAt,
		})
		emissionIDs = append(emissionIDs, emission.EmissionID)
		availability = append(availability, AvailabilityRecord{EmissionID: emission.EmissionID, Entity: emission.Entity,
			Attribute: emission.Attribute, Source: cloneSource(emission.Source), ObservedAt: emission.ObservedAt, ReceivedAt: emission.ReceivedAt})
	}
	availabilityArtifact := AvailabilityArtifact{Meta: artifactMeta(exampleID, sourceArtifact.Meta.Digest), Cutoff: cutoff, Records: availability}
	availabilityArtifact.Meta.Digest = digestJSON(availabilityArtifact)
	batch := stateencoder.Batch{SchemaVersion: stateencoder.BatchSchemaVersion, EventTime: cutoff, ReceivedAt: cutoff,
		Entities: batchEntities, Observations: observations, Relations: inputRelations}
	encoded, err := stateencoder.Encode(batch)
	if err != nil {
		return exampleArtifacts{}, fmt.Errorf("encode cutoff input: %w", err)
	}
	input := InputArtifact{Meta: artifactMeta(exampleID, availabilityArtifact.Meta.Digest), AvailableEmissionIDs: emissionIDs, Batch: batch, Encoded: encoded}
	input.Meta.Digest = digestJSON(ModelInput{Batch: batch, Encoded: encoded, Availability: availability})
	input.Availability = availability

	transition := "steady"
	if !before && after {
		transition = "rising"
	} else if before && !after {
		transition = "falling"
	}
	topologyLabel := "unchanged"
	if len(futureEdges) > 0 {
		topologyLabel = "edge_added"
	}
	anomalyLabel := "false"
	if anomaly {
		anomalyLabel = "true"
	}
	consistencyLabel := "consistent"
	if !consistent {
		consistencyLabel = "inconsistent"
	}
	targets := TargetArtifact{Meta: artifactMeta(exampleID, truth.Meta.Digest, sourceArtifact.Meta.Digest), Labels: []TaskLabel{
		{TaskID: TaskNextState, Value: strconv.FormatBool(after), TargetTime: cutoff.Add(time.Minute), AvailableAfter: cutoff.Add(time.Minute), Definition: "physical occupancy/light state at the one-minute horizon"},
		{TaskID: TaskTemporalTransition, Value: transition, TargetTime: cutoff.Add(time.Minute), AvailableAfter: cutoff.Add(time.Minute), Definition: "transition from the hidden pre-cutoff state to the one-minute target state"},
		{TaskID: TaskTopologyTransition, Value: topologyLabel, TargetTime: cutoff.Add(time.Minute), AvailableAfter: cutoff.Add(time.Minute), Definition: "directed edge change in hidden topology over the one-minute horizon"},
		{TaskID: TaskBehaviorAnomaly, Value: anomalyLabel, TargetTime: cutoff.Add(time.Minute), AvailableAfter: cutoff.Add(time.Minute), Definition: "deviation from this synthetic routine; not intrusion, danger, or malfunction"},
		{TaskID: TaskObservationConsistency, Value: consistencyLabel, TargetTime: cutoff, AvailableAfter: cutoff, Definition: "whether generated contemporaneous source observations agree"},
	}}
	targets.Meta.Digest = digestJSON(targets)

	scenarioSplit := ScenarioSplitTrain
	if scenario.HeldOut {
		scenarioSplit = ScenarioSplitHeldOut
	}
	index := ExampleIndex{ExampleID: exampleID, HomeGroupID: home.groupID, TopologyFamilyID: home.familyID,
		ScenarioFamily: scenario.ID, HomeSplit: home.split, ScenarioTemplateSplit: scenarioSplit, Cutoff: cutoff, SeedLineage: lineage}
	return exampleArtifacts{truth: truth, sources: sourceArtifact, availability: availabilityArtifact, input: input, targets: targets, index: index}, nil
}

func validTaskValue(task TaskID, value string) bool {
	switch task {
	case TaskNextState, TaskBehaviorAnomaly:
		return value == "true" || value == "false"
	case TaskTemporalTransition:
		return value == "steady" || value == "rising" || value == "falling"
	case TaskTopologyTransition:
		return value == "unchanged" || value == "edge_added"
	case TaskObservationConsistency:
		return value == "consistent" || value == "inconsistent"
	default:
		return false
	}
}

func splitFamilies(count int, seed int64) map[int]Split {
	order := rand.New(rand.NewSource(int64(streamSeed(seed, "split-families")))).Perm(count)
	trainCount := (count*6 + 5) / 10
	validationCount := (count + 2) / 5
	if trainCount == 0 {
		trainCount = 1
	}
	if validationCount == 0 {
		validationCount = 1
	}
	if trainCount+validationCount >= count {
		trainCount, validationCount = 1, 1
	}
	result := make(map[int]Split, count)
	for position, family := range order {
		switch {
		case position < trainCount:
			result[family] = SplitTrain
		case position < trainCount+validationCount:
			result[family] = SplitValidation
		default:
			result[family] = SplitTest
		}
	}
	return result
}

func streamSeed(master int64, name string, parts ...string) uint64 {
	h := sha256.New()
	_, _ = h.Write([]byte(strconv.FormatInt(master, 10)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(name))
	for _, part := range parts {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(part))
	}
	return binary.LittleEndian.Uint64(h.Sum(nil)[:8])
}

func shortHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}

func isArtifactID(value string) bool {
	if len(value) != 16 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func artifactMeta(exampleID string, parentDigests ...string) ArtifactMeta {
	return ArtifactMeta{ExampleID: exampleID, DatasetVersion: DatasetSchemaVersion, GeneratorVersion: GeneratorVersion, ParentDigests: append([]string(nil), parentDigests...)}
}

func digestJSON(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func artifactDigest(value any) string {
	switch artifact := value.(type) {
	case GroundTruthArtifact:
		artifact.Meta.Digest = ""
		return digestJSON(artifact)
	case SourceArtifact:
		artifact.Meta.Digest = ""
		return digestJSON(artifact)
	case AvailabilityArtifact:
		artifact.Meta.Digest = ""
		return digestJSON(artifact)
	case TargetArtifact:
		artifact.Meta.Digest = ""
		return digestJSON(artifact)
	default:
		return ""
	}
}

// SelectAvailable returns only emissions received by the prediction cutoff.
// Event and observation timestamps do not substitute for actual availability.
func SelectAvailable(emissions []SourceEmission, cutoff time.Time) []SourceEmission {
	available := make([]SourceEmission, 0, len(emissions))
	for _, emission := range emissions {
		if !emission.ReceivedAt.After(cutoff) {
			copy := emission
			copy.Value = cloneValue(emission.Value)
			copy.Source = cloneSource(emission.Source)
			available = append(available, copy)
		}
	}
	return available
}

func boolPointer(value bool) *bool { return &value }

func cloneValue(value *stateencoder.Value) *stateencoder.Value {
	if value == nil {
		return nil
	}
	copy := *value
	if value.Number != nil {
		v := *value.Number
		copy.Number = &v
	}
	if value.Boolean != nil {
		v := *value.Boolean
		copy.Boolean = &v
	}
	if value.Category != nil {
		v := *value.Category
		copy.Category = &v
	}
	return &copy
}

func cloneSource(source *stateencoder.SourceRef) *stateencoder.SourceRef {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
}

func cloneEntities(values []stateencoder.EntityRef) []stateencoder.EntityRef {
	return append([]stateencoder.EntityRef(nil), values...)
}

func cloneRelations(values []stateencoder.Relation) []stateencoder.Relation {
	result := append([]stateencoder.Relation(nil), values...)
	for i := range result {
		result[i].Source = cloneSource(result[i].Source)
	}
	return result
}

func cloneSeeds(values map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func withoutEntity(entities []stateencoder.EntityRef, id string) []stateencoder.EntityRef {
	result := make([]stateencoder.EntityRef, 0, len(entities))
	for _, entity := range entities {
		if entity.ID != id {
			result = append(result, entity)
		}
	}
	return result
}

func entitiesForAvailable(entities []stateencoder.EntityRef, available []SourceEmission, relations []stateencoder.Relation) []stateencoder.EntityRef {
	needed := map[string]stateencoder.EntityRef{}
	for _, entity := range entities {
		needed[entity.ID] = entity
	}
	for _, emission := range available {
		needed[emission.Entity.ID] = emission.Entity
	}
	for _, relation := range relations {
		needed[relation.Subject] = stateencoder.EntityRef{ID: relation.Subject, Kind: entityKind(entities, relation.Subject)}
		needed[relation.Object] = stateencoder.EntityRef{ID: relation.Object, Kind: entityKind(entities, relation.Object)}
	}
	result := make([]stateencoder.EntityRef, 0, len(needed))
	for _, entity := range needed {
		result = append(result, entity)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func entityKind(entities []stateencoder.EntityRef, id string) string {
	for _, entity := range entities {
		if entity.ID == id {
			return entity.Kind
		}
	}
	return "room"
}

func makeDoorEmission(home homePlan, cutoff time.Time, value bool) SourceEmission {
	entity := stateencoder.EntityRef{ID: "sensor:" + shortHash(home.groupID, "door"), Kind: "sensor"}
	observed := cutoff.Add(-8 * time.Second)
	return SourceEmission{
		EmissionID: "emission:" + shortHash(home.groupID, entity.ID, observed.Format(time.RFC3339Nano)),
		Entity:     entity, Attribute: stateencoder.AttributeRef{Namespace: "access", Name: "door_open", ValueType: stateencoder.ValueBoolean},
		Status: stateencoder.StatusObserved, Value: &stateencoder.Value{Boolean: boolPointer(value)},
		Source:     &stateencoder.SourceRef{ID: "source:" + shortHash(home.groupID, entity.ID), Kind: "contact"},
		Provenance: "record:" + shortHash(entity.ID, observed.Format(time.RFC3339Nano)), ObservedAt: observed, ReceivedAt: cutoff.Add(-3 * time.Second),
	}
}

func makeUnavailableEmission(base SourceEmission, observedAt, receivedAt time.Time) SourceEmission {
	base.EmissionID = "emission:" + shortHash(base.Entity.ID, receivedAt.Format(time.RFC3339Nano), "source-record")
	base.Status, base.Value, base.ObservedAt, base.ReceivedAt = stateencoder.StatusUnavailable, nil, time.Time{}, receivedAt
	base.Provenance = "record:" + shortHash(base.Entity.ID, receivedAt.Format(time.RFC3339Nano))
	return base
}
