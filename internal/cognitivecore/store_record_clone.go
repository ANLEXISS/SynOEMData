package cognitivecore

// Clone methods define the ownership boundary between cognitive callers and
// Foundation statestore. They copy only the mutable reference fields in the
// persisted cognitive records.
func (d Decision) Clone() Decision {
	d.HeadOrder = append([]string(nil), d.HeadOrder...)
	d.Incidents = append([]string(nil), d.Incidents...)
	d.HeadLatencyMS = cloneFloatMap(d.HeadLatencyMS)
	d.Action.Reasons = append([]string(nil), d.Action.Reasons...)
	d.Trace = cloneMLPTrace(d.Trace)
	return d
}

func (d DecisionV3) Clone() DecisionV3 {
	d.HeadOrder = append([]string(nil), d.HeadOrder...)
	d.HeadLatencyMS = cloneFloatMap(d.HeadLatencyMS)
	d.Action.Reasons = append([]string(nil), d.Action.Reasons...)
	d.Communication.Reasons = append([]string(nil), d.Communication.Reasons...)
	return d
}

func cloneFloatMap(input map[string]float64) map[string]float64 {
	if input == nil {
		return nil
	}
	result := make(map[string]float64, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneMLPTrace(input *MLPTrace) *MLPTrace {
	if input == nil {
		return nil
	}
	result := *input
	result.Topology.Heads = append([]MLPHeadTopology(nil), input.Topology.Heads...)
	for i := range result.Topology.Heads {
		result.Topology.Heads[i].Layers = append([]MLPLayerTopology(nil), input.Topology.Heads[i].Layers...)
	}
	result.Activations = append([]MLPActivationSummary(nil), input.Activations...)
	for i := range result.Activations {
		result.Activations[i].ActiveNodes = append([]MLPActiveNode(nil), input.Activations[i].ActiveNodes...)
	}
	result.ActivePaths = append([]MLPActivePath(nil), input.ActivePaths...)
	return &result
}
