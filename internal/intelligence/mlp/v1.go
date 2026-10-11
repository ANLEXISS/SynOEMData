package mlp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	core "synora/internal/cognitivecore"
)

const maxMLPActivePaths = 64

type Manifest struct {
	SchemaVersion  string              `json:"schema_version"`
	SnapshotSchema string              `json:"snapshot_schema"`
	EncoderSchema  string              `json:"encoder_schema"`
	EncoderVersion string              `json:"encoder_version"`
	InputDimension int                 `json:"input_dimension"`
	FeatureNames   []string            `json:"feature_names"`
	Labels         map[string][]string `json:"labels"`
	Heads          []string            `json:"heads"`
	WeightsSHA256  string              `json:"weights_sha256"`
	HeadVersions   map[string]string   `json:"head_versions"`
	Artifacts      map[string]Artifact `json:"artifacts"`
}

type Artifact struct {
	Version    string    `json:"version"`
	Artifact   string    `json:"artifact"`
	SHA256     string    `json:"sha256"`
	Labels     []string  `json:"labels"`
	Thresholds []float32 `json:"thresholds"`
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != core.BundleManifestSchema || m.SnapshotSchema != core.SnapshotSchemaVersion || m.EncoderSchema != core.EncoderSchemaVersion || m.EncoderVersion != "1.0.0" || m.InputDimension != core.CognitiveVectorSize {
		return fmt.Errorf("V1 MLP manifest does not match the contract")
	}
	if len(m.FeatureNames) != core.CognitiveVectorSize {
		return fmt.Errorf("V1 MLP feature order is missing")
	}
	for i, name := range core.CognitiveFeatureNames {
		if m.FeatureNames[i] != name {
			return fmt.Errorf("V1 MLP feature order mismatch at %d", i)
		}
	}
	if len(m.Heads) != len(core.HeadOrder) {
		return fmt.Errorf("V1 MLP manifest head count mismatch")
	}
	for i, head := range core.HeadOrder {
		if m.Heads[i] != head {
			return fmt.Errorf("V1 MLP head order mismatch at %d", i)
		}
		if len(m.Labels[head]) == 0 {
			return fmt.Errorf("V1 MLP labels missing for %s", head)
		}
		if m.HeadVersions != nil && m.HeadVersions[head] != "1.0.0" {
			return fmt.Errorf("V1 MLP head version mismatch for %s", head)
		}
		if m.Artifacts != nil && m.Artifacts[head].Artifact == "" {
			return fmt.Errorf("V1 MLP artifact missing for %s", head)
		}
	}
	return nil
}

func LoadManifest(path string) (Manifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", core.ErrModelUnavailable, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", core.ErrModelUnavailable, err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", core.ErrModelUnavailable, err)
	}
	return manifest, nil
}

type cpuLayer struct {
	InputSize  int       `json:"input_size"`
	OutputSize int       `json:"output_size"`
	Activation string    `json:"activation"`
	Weights    []float32 `json:"weights"`
	Bias       []float32 `json:"bias"`
}

type cpuHead struct {
	Schema       string     `json:"schema"`
	Head         string     `json:"head"`
	InputSize    int        `json:"input_size"`
	OutputSize   int        `json:"output_size"`
	Labels       []string   `json:"labels"`
	VectorSchema string     `json:"vector_schema"`
	Layers       []cpuLayer `json:"layers"`
}

// CPUBundleMLP is the small deterministic inference backend emitted by the
// V1 Python pipeline. It has no dependency on PyTorch model formats.
type CPUBundleMLP struct {
	Manifest Manifest
	Heads    map[string]cpuHead
}

func LoadCPUBundle(dir string) (*CPUBundleMLP, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: bundle directory is empty", core.ErrModelUnavailable)
	}
	manifest, err := LoadManifest(filepath.Join(dir, "MANIFEST.v1.json"))
	if err != nil {
		return nil, err
	}
	bundle := &CPUBundleMLP{Manifest: manifest, Heads: make(map[string]cpuHead, len(core.HeadOrder))}
	for _, head := range core.HeadOrder {
		artifact := manifest.Artifacts[head].Artifact
		if artifact == "" {
			artifact = head + ".cpu.json"
		}
		body, err := os.ReadFile(filepath.Join(dir, artifact))
		if err != nil {
			return nil, fmt.Errorf("%w: read %s: %v", core.ErrModelUnavailable, head, err)
		}
		var model cpuHead
		if err := json.Unmarshal(body, &model); err != nil {
			return nil, fmt.Errorf("%w: decode %s: %v", core.ErrModelUnavailable, head, err)
		}
		if err := validateCPUHead(model, head, manifest.Labels[head]); err != nil {
			return nil, fmt.Errorf("%w: %v", core.ErrModelUnavailable, err)
		}
		bundle.Heads[head] = model
	}
	return bundle, nil
}

func validateCPUHead(model cpuHead, expectedHead string, labels []string) error {
	if model.Schema != "synora.cognitive-cpu-mlp/v1" || model.Head != expectedHead || model.InputSize != core.CognitiveVectorSize || model.OutputSize != len(labels) || model.VectorSchema != core.EncoderSchemaVersion {
		return fmt.Errorf("invalid CPU head contract for %s", expectedHead)
	}
	if len(model.Labels) != len(labels) {
		return fmt.Errorf("CPU head labels mismatch for %s", expectedHead)
	}
	for i := range labels {
		if model.Labels[i] != labels[i] {
			return fmt.Errorf("CPU head label order mismatch for %s", expectedHead)
		}
	}
	if len(model.Layers) == 0 {
		return fmt.Errorf("CPU head has no layers: %s", expectedHead)
	}
	input := core.CognitiveVectorSize
	for _, layer := range model.Layers {
		if layer.InputSize != input || layer.OutputSize <= 0 || len(layer.Weights) != layer.InputSize*layer.OutputSize || len(layer.Bias) != layer.OutputSize || (layer.Activation != "relu" && layer.Activation != "identity") {
			return fmt.Errorf("invalid CPU layer for %s", expectedHead)
		}
		input = layer.OutputSize
	}
	if input != model.OutputSize {
		return fmt.Errorf("CPU head output mismatch for %s", expectedHead)
	}
	return nil
}

func (m *CPUBundleMLP) Run(ctx context.Context, encoded core.EncodedSnapshot, snapshot core.CognitiveSnapshot) (core.MLPOutput, map[string]float64, error) {
	if m == nil {
		return core.MLPOutput{}, nil, core.ErrModelUnavailable
	}
	if err := encoded.Validate(); err != nil {
		return core.MLPOutput{}, nil, err
	}
	values := encoded.Values[:]
	latencies := make(map[string]float64, len(core.HeadOrder))
	outputs := make(map[string][]float32, len(core.HeadOrder))
	trace := &core.MLPTrace{SchemaVersion: core.MLPTraceSchemaVersion, ModelVersion: m.modelVersion(), Topology: m.topology(), Activations: make([]core.MLPActivationSummary, 0, len(core.HeadOrder)*2), ActivePaths: make([]core.MLPActivePath, 0, len(core.HeadOrder)*4), Redacted: true}
	startedAll := time.Now()
	for _, head := range core.HeadOrder {
		if err := ctx.Err(); err != nil {
			return core.MLPOutput{}, latencies, err
		}
		started := time.Now()
		result, summaries, paths, err := m.runHead(head, values)
		latencies[head] = float64(time.Since(started).Microseconds()) / 1000.0
		if err != nil {
			return core.MLPOutput{}, latencies, err
		}
		outputs[head] = result
		trace.Activations = append(trace.Activations, summaries...)
		trace.ActivePaths = append(trace.ActivePaths, paths...)
	}
	trace.ActivePaths = selectBoundedActivePaths(trace.ActivePaths, maxMLPActivePaths)
	trace.DurationMS = float64(time.Since(startedAll).Microseconds()) / 1000.0
	dangerProb := softmax(outputs["danger"])
	dangerIndex := argmax(dangerProb)
	incidentProb := softmax(outputs["incident"])
	taskProb := softmax(outputs["task"])
	actionProb := softmax(outputs["action"])
	incidentIndex := argmax(incidentProb)
	taskIndex := argmax(taskProb)
	actionIndex := argmax(actionProb)
	return core.MLPOutput{
		DangerLabel:        m.Manifest.Labels["danger"][dangerIndex],
		DangerScore:        dangerProb[dangerIndex],
		IncidentConfidence: incidentProb[incidentIndex],
		TaskConfidence:     taskProb[taskIndex],
		ActionConfidence:   actionProb[actionIndex],
		Incidents:          []string{m.Manifest.Labels["incident"][incidentIndex]},
		Task:               m.Manifest.Labels["task"][taskIndex],
		Action:             core.ActionIntent{Action: m.Manifest.Labels["action"][actionIndex], Topology: snapshot.Topology, Capability: m.Manifest.Labels["action"][actionIndex]},
		Trace:              trace,
	}, latencies, nil
}

type activePathGroup struct {
	key   string
	head  string
	paths []core.MLPActivePath
}

// selectBoundedActivePaths keeps the existing redacted path strength as the
// only within-group ranking signal. It first reserves one path for every
// non-empty (head, layer transition) group, then allocates the remaining
// budget round-robin in stable head/transition order. No path is synthesized.
func selectBoundedActivePaths(paths []core.MLPActivePath, limit int) []core.MLPActivePath {
	if limit <= 0 || len(paths) == 0 {
		return nil
	}
	groupsByKey := make(map[string]*activePathGroup)
	for _, path := range paths {
		groupKey, head, _, _, ok := activePathGroupFor(path)
		if !ok {
			continue
		}
		group := groupsByKey[groupKey]
		if group == nil {
			group = &activePathGroup{key: groupKey, head: head}
			groupsByKey[groupKey] = group
		}
		group.paths = append(group.paths, path)
	}
	groups := make([]*activePathGroup, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		sort.SliceStable(group.paths, func(i, j int) bool {
			if group.paths[i].Strength != group.paths[j].Strength {
				return group.paths[i].Strength > group.paths[j].Strength
			}
			if group.paths[i].From != group.paths[j].From {
				return group.paths[i].From < group.paths[j].From
			}
			return group.paths[i].To < group.paths[j].To
		})
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		iHead, jHead := headOrderIndex(groups[i].head), headOrderIndex(groups[j].head)
		if iHead != jHead {
			return iHead < jHead
		}
		return groups[i].key < groups[j].key
	})

	selected := make([]core.MLPActivePath, 0, minInt(limit, len(paths)))
	positions := make([]int, len(groups))
	for index, group := range groups {
		if len(selected) >= limit {
			break
		}
		if len(group.paths) == 0 {
			continue
		}
		selected = append(selected, group.paths[0])
		positions[index] = 1
	}
	for len(selected) < limit {
		added := false
		for index, group := range groups {
			position := positions[index]
			if position >= len(group.paths) {
				continue
			}
			selected = append(selected, group.paths[position])
			positions[index]++
			added = true
			if len(selected) == limit {
				break
			}
		}
		if !added {
			break
		}
	}
	return selected
}

func activePathGroupFor(path core.MLPActivePath) (groupKey, head, fromLayer, toLayer string, ok bool) {
	fromHead, fromLayer, fromOK := mlpNodeHeadLayer(path.From)
	toHead, toLayer, toOK := mlpNodeHeadLayer(path.To)
	if !fromOK || !toOK || fromHead != toHead {
		return "", "", "", "", false
	}
	return fromHead + "\x00" + fromLayer + "->" + toLayer, fromHead, fromLayer, toLayer, true
}

func mlpNodeHeadLayer(nodeID string) (head, layer string, ok bool) {
	nodeMarker := strings.LastIndex(nodeID, ".node-")
	if nodeMarker <= 0 || nodeMarker+len(".node-") >= len(nodeID) {
		return "", "", false
	}
	headLayer := nodeID[:nodeMarker]
	separator := strings.LastIndexByte(headLayer, '.')
	if separator <= 0 || separator+1 >= len(headLayer) {
		return "", "", false
	}
	return headLayer[:separator], headLayer[separator+1:], true
}

func headOrderIndex(head string) int {
	for index, candidate := range core.HeadOrder {
		if candidate == head {
			return index
		}
	}
	return len(core.HeadOrder)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *CPUBundleMLP) runHead(head string, input []float32) ([]float32, []core.MLPActivationSummary, []core.MLPActivePath, error) {
	model, ok := m.Heads[head]
	if !ok {
		return nil, nil, nil, fmt.Errorf("head %s is unavailable", head)
	}
	current := append([]float32(nil), input...)
	var summaries []core.MLPActivationSummary
	var paths []core.MLPActivePath
	var previous []core.MLPActiveNode
	for layerIndex, layer := range model.Layers {
		next := make([]float32, layer.OutputSize)
		for output := range next {
			sum := layer.Bias[output]
			for index, value := range current {
				sum += layer.Weights[output*layer.InputSize+index] * value
			}
			if layer.Activation == "relu" && sum < 0 {
				sum = 0
			}
			next[output] = sum
		}
		layerID := fmt.Sprintf("%s.layer-%d", head, layerIndex+1)
		active := topActiveNodes(layerID, next)
		summaries = append(summaries, summarizeLayer(layerID, next, active))
		if layerIndex > 0 {
			for _, from := range previous {
				for _, to := range active {
					fromIndex := nodeIndex(from.NodeID)
					toIndex := nodeIndex(to.NodeID)
					if fromIndex >= 0 && fromIndex < layer.InputSize && toIndex >= 0 && toIndex < layer.OutputSize {
						strength := abs32(layer.Weights[toIndex*layer.InputSize+fromIndex]) * abs32(from.Activation)
						if strength > 0 {
							paths = append(paths, core.MLPActivePath{From: from.NodeID, To: to.NodeID, Strength: strength})
						}
					}
				}
			}
		}
		previous = active
		current = next
	}
	return current, summaries, paths, nil
}

func (m *CPUBundleMLP) modelVersion() string {
	hash := m.Manifest.WeightsSHA256
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return "cognitive-v1:" + hash
}

func (m *CPUBundleMLP) topology() core.MLPTopology {
	topology := core.MLPTopology{SchemaVersion: core.MLPTraceSchemaVersion, ModelVersion: m.modelVersion(), Heads: make([]core.MLPHeadTopology, 0, len(core.HeadOrder))}
	for _, head := range core.HeadOrder {
		model := m.Heads[head]
		item := core.MLPHeadTopology{Name: head, Layers: make([]core.MLPLayerTopology, 0, len(model.Layers))}
		for index, layer := range model.Layers {
			item.Layers = append(item.Layers, core.MLPLayerTopology{ID: fmt.Sprintf("%s.layer-%d", head, index+1), InputSize: layer.InputSize, OutputSize: layer.OutputSize, Activation: layer.Activation})
		}
		topology.Heads = append(topology.Heads, item)
	}
	return topology
}

func summarizeLayer(layerID string, values []float32, active []core.MLPActiveNode) core.MLPActivationSummary {
	var minimum, maximum, total float32
	if len(values) > 0 {
		minimum, maximum = values[0], values[0]
	}
	for _, value := range values {
		if value < minimum {
			minimum = value
		}
		if value > maximum {
			maximum = value
		}
		total += value
	}
	return core.MLPActivationSummary{LayerID: layerID, Count: len(values), Minimum: minimum, Maximum: maximum, Mean: total / float32(maxInt(1, len(values))), ActiveNodes: active}
}

func topActiveNodes(layerID string, values []float32) []core.MLPActiveNode {
	nodes := make([]core.MLPActiveNode, 0, len(values))
	for index, value := range values {
		nodes = append(nodes, core.MLPActiveNode{NodeID: fmt.Sprintf("%s.node-%d", layerID, index), Activation: value})
	}
	sort.Slice(nodes, func(i, j int) bool { return abs32(nodes[i].Activation) > abs32(nodes[j].Activation) })
	if len(nodes) > 5 {
		nodes = nodes[:5]
	}
	return nodes
}

func nodeIndex(id string) int {
	index := strings.LastIndex(id, "node-")
	if index < 0 {
		return -1
	}
	var value int
	if _, err := fmt.Sscanf(id[index+5:], "%d", &value); err != nil {
		return -1
	}
	return value
}
func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
func softmax(values []float32) []float32 {
	result := make([]float32, len(values))
	if len(values) == 0 {
		return result
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	var total float32
	for index, value := range values {
		result[index] = float32(math.Exp(float64(value - max)))
		total += result[index]
	}
	if total > 0 {
		for index := range result {
			result[index] /= total
		}
	}
	return result
}
func argmax(values []float32) int {
	index := 0
	for i := 1; i < len(values); i++ {
		if values[i] > values[index] {
			index = i
		}
	}
	return index
}
