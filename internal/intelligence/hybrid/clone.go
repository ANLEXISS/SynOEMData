package hybrid

import (
	"time"

	"synora/internal/intelligence/stateencoder"
)

func cloneEventInput(input EventInput) EventInput {
	input.Observations = append([]AvailableObservation(nil), input.Observations...)
	for index := range input.Observations {
		observation := &input.Observations[index].Observation
		if observation.Number != nil {
			value := *observation.Number
			observation.Number = &value
		}
		if observation.Boolean != nil {
			value := *observation.Boolean
			observation.Boolean = &value
		}
		if observation.Category != nil {
			value := *observation.Category
			observation.Category = &value
		}
		observation.Source = cloneSource(observation.Source)
		observation.ObservedAt = cloneTime(observation.ObservedAt)
		observation.AgeSeconds = cloneFloat64(observation.AgeSeconds)
	}
	return input
}

func cloneEventRepresentations(events []EventRepresentation) []EventRepresentation {
	cloned := append([]EventRepresentation(nil), events...)
	for index := range cloned {
		cloned[index].Source = cloneSource(cloned[index].Source)
		cloned[index].ObservedAt = cloneTime(cloned[index].ObservedAt)
		cloned[index].Representation = cloneRepresentation(cloned[index].Representation)
	}
	return cloned
}

func cloneRepresentation(representation Representation) Representation {
	representation.Values = append([]float32(nil), representation.Values...)
	representation.ValidityMask = append([]bool(nil), representation.ValidityMask...)
	representation.EvidenceRefs = append([]string(nil), representation.EvidenceRefs...)
	return representation
}

func cloneTopoInput(input TopoTemporalInput) TopoTemporalInput {
	input.Events = cloneEventRepresentations(input.Events)
	input.Entities = append([]AvailableEntity(nil), input.Entities...)
	input.Relations = append([]AvailableRelation(nil), input.Relations...)
	for index := range input.Relations {
		input.Relations[index].Relation.Source = cloneSource(input.Relations[index].Relation.Source)
		input.Relations[index].Relation.ValidFrom = cloneTime(input.Relations[index].Relation.ValidFrom)
		input.Relations[index].Relation.ValidTo = cloneTime(input.Relations[index].Relation.ValidTo)
	}
	input.Locations = append([]EventLocationAssociation(nil), input.Locations...)
	for index := range input.Locations {
		if input.Locations[index].Entity != nil {
			entity := *input.Locations[index].Entity
			input.Locations[index].Entity = &entity
		}
	}
	input.HistoryStart = cloneTime(input.HistoryStart)
	input.HistoryEnd = cloneTime(input.HistoryEnd)
	return input
}

func cloneHomeInput(input HomeAdaptationInput) HomeAdaptationInput {
	input.General = cloneFusion(input.General)
	return input
}

func cloneFusion(fusion FusionEnvelope) FusionEnvelope {
	if fusion.General != nil {
		general := cloneFusion(*fusion.General)
		fusion.General = &general
	}
	fusion.Contributions = append([]BranchContribution(nil), fusion.Contributions...)
	for index := range fusion.Contributions {
		contribution := &fusion.Contributions[index]
		if contribution.Component != nil {
			component := *contribution.Component
			contribution.Component = &component
		}
		contribution.Events = cloneEventRepresentations(contribution.Events)
		contribution.Representations = append([]Representation(nil), contribution.Representations...)
		for representationIndex := range contribution.Representations {
			contribution.Representations[representationIndex] = cloneRepresentation(contribution.Representations[representationIndex])
		}
	}
	return fusion
}

func cloneSource(source *stateencoder.SourceRef) *stateencoder.SourceRef {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
