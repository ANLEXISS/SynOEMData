package mlp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	core "synora/internal/cognitivecore"
)

type V3Manifest struct {
	SchemaVersion          string                       `json:"schema_version"`
	SnapshotSchema         string                       `json:"snapshot_schema"`
	EncoderSchema          string                       `json:"encoder_schema"`
	EncoderVersion         string                       `json:"encoder_version"`
	InputDimension         int                          `json:"input_dimension"`
	HiddenSize             int                          `json:"hidden_size"`
	FeatureNames           []string                     `json:"feature_names"`
	Heads                  []string                     `json:"heads"`
	Labels                 map[string][]string          `json:"labels"`
	Artifacts              map[string]map[string]string `json:"artifacts"`
	Hashes                 map[string]string            `json:"hashes"`
	DryRunRequired         bool                         `json:"dry_run_required"`
	PhysicalActionExecuted bool                         `json:"physical_action_executed"`
	RealVisionMetrics      bool                         `json:"real_vision_metrics"`
	Promoted               bool                         `json:"promoted"`
}

func (m V3Manifest) Validate() error {
	if m.SchemaVersion != "synora.cognitive-v3-manifest/v1" || m.SnapshotSchema != core.SnapshotSchemaVersionV3 || m.EncoderSchema != core.EncoderSchemaVersionV3 || m.EncoderVersion != "3.0.0" || m.InputDimension != core.CognitiveVectorSizeV3 || m.HiddenSize <= 0 || len(m.FeatureNames) != core.CognitiveVectorSizeV3 {
		return fmt.Errorf("V3 MLP manifest does not match the contract")
	}
	for i, name := range core.CognitiveFeatureNamesV3 {
		if m.FeatureNames[i] != name {
			return fmt.Errorf("V3 feature order mismatch at %d", i)
		}
	}
	if len(m.Heads) != len(core.HeadOrderV3) || !m.DryRunRequired || m.PhysicalActionExecuted || m.RealVisionMetrics || m.Promoted {
		return fmt.Errorf("invalid V3 head or dry-run contract")
	}
	for i, head := range core.HeadOrderV3 {
		if m.Heads[i] != head || len(m.Labels[head]) != len(core.LabelsV3[head]) {
			return fmt.Errorf("invalid V3 head %s", head)
		}
		for j, label := range core.LabelsV3[head] {
			if m.Labels[head][j] != label {
				return fmt.Errorf("V3 label order mismatch for %s", head)
			}
		}
	}
	return nil
}

type v3CPUHead struct {
	Schema     string     `json:"schema"`
	Head       string     `json:"head"`
	InputSize  int        `json:"input_size"`
	OutputSize int        `json:"output_size"`
	Labels     []string   `json:"labels"`
	Layers     []cpuLayer `json:"layers"`
}

type CPUBundleMLPV3 struct {
	Manifest V3Manifest
	Backbone []cpuLayer
	Heads    map[string]v3CPUHead
}

func LoadCPUBundleV3(dir string) (*CPUBundleMLPV3, error) {
	body, err := os.ReadFile(filepath.Join(dir, "MANIFEST.v3.json"))
	if err != nil {
		return nil, err
	}
	var manifest V3Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	readArtifact := func(name string) ([]byte, error) {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if expected := manifest.Hashes[name]; expected == "" || fmt.Sprintf("%x", sha256.Sum256(b)) != expected {
			return nil, fmt.Errorf("V3 artifact hash mismatch for %s", name)
		}
		return b, nil
	}
	readLayers := func(name string) ([]cpuLayer, error) {
		b, err := readArtifact(name)
		if err != nil {
			return nil, err
		}
		var value struct {
			Schema string     `json:"schema"`
			Layers []cpuLayer `json:"layers"`
		}
		if err := json.Unmarshal(b, &value); err != nil {
			return nil, err
		}
		if value.Schema != "synora.cognitive-cpu-mlp/v3-backbone" || len(value.Layers) == 0 {
			return nil, fmt.Errorf("invalid V3 backbone")
		}
		return value.Layers, nil
	}
	backbone, err := readLayers("backbone.cpu.json")
	if err != nil {
		return nil, err
	}
	if err := validateV3Layers(backbone, core.CognitiveVectorSizeV3, manifest.HiddenSize); err != nil {
		return nil, err
	}
	bundle := &CPUBundleMLPV3{Manifest: manifest, Backbone: backbone, Heads: make(map[string]v3CPUHead, len(core.HeadOrderV3))}
	for _, head := range core.HeadOrderV3 {
		name := manifest.Artifacts[head]["artifact"]
		if name == "" {
			name = head + ".cpu.json"
		}
		b, err := readArtifact(name)
		if err != nil {
			return nil, err
		}
		var model v3CPUHead
		if err := json.Unmarshal(b, &model); err != nil {
			return nil, err
		}
		if model.Schema != "synora.cognitive-cpu-mlp/v3-head" || model.Head != head || model.InputSize != manifest.HiddenSize || model.OutputSize != len(core.LabelsV3[head]) || len(model.Labels) != len(core.LabelsV3[head]) {
			return nil, fmt.Errorf("invalid V3 head contract for %s", head)
		}
		if err := validateV3Layers(model.Layers, manifest.HiddenSize, model.OutputSize); err != nil {
			return nil, err
		}
		bundle.Heads[head] = model
	}
	return bundle, nil
}

func validateV3Layers(layers []cpuLayer, input, output int) error {
	for _, layer := range layers {
		if layer.InputSize != input || layer.OutputSize <= 0 || len(layer.Weights) != layer.InputSize*layer.OutputSize || len(layer.Bias) != layer.OutputSize || (layer.Activation != "relu" && layer.Activation != "identity") {
			return fmt.Errorf("invalid V3 CPU layer")
		}
		input = layer.OutputSize
	}
	if input != output {
		return fmt.Errorf("V3 layer output mismatch")
	}
	return nil
}

func runCPULayersV3(layers []cpuLayer, input []float32) []float32 {
	current := append([]float32(nil), input...)
	for _, layer := range layers {
		next := make([]float32, layer.OutputSize)
		for o := range next {
			value := layer.Bias[o]
			for i, x := range current {
				value += layer.Weights[o*layer.InputSize+i] * x
			}
			if layer.Activation == "relu" && value < 0 {
				value = 0
			}
			next[o] = value
		}
		current = next
	}
	return current
}

func (m *CPUBundleMLPV3) RunV3(ctx context.Context, encoded core.EncodedSnapshotV3, snapshot core.CognitiveSnapshotV3) (core.MLPOutputV3, map[string]float64, error) {
	if m == nil {
		return core.MLPOutputV3{}, nil, fmt.Errorf("V3 CPU bundle unavailable")
	}
	if err := encoded.Validate(); err != nil {
		return core.MLPOutputV3{}, nil, err
	}
	if err := snapshot.Validate(); err != nil {
		return core.MLPOutputV3{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return core.MLPOutputV3{}, nil, err
	}
	started := time.Now()
	hidden := runCPULayersV3(m.Backbone, encoded.Values[:])
	logits := make(map[string][]float32, len(core.HeadOrderV3))
	latency := make(map[string]float64, len(core.HeadOrderV3))
	for _, head := range core.HeadOrderV3 {
		began := time.Now()
		logits[head] = runCPULayersV3(m.Heads[head].Layers, hidden)
		latency[head] = float64(time.Since(began).Microseconds()) / 1000
	}
	choose := func(head string) (string, float32) {
		probabilities := softmax(logits[head])
		index := argmax(probabilities)
		return core.LabelsV3[head][index], probabilities[index]
	}
	danger, dangerScore := choose("danger")
	incident, incidentScore := choose("incident")
	task, taskScore := choose("task")
	action, actionScore := choose("action")
	communication, communicationScore := choose("communication_intent")
	latency["shared_backbone"] = float64(time.Since(started).Microseconds()) / 1000
	return core.MLPOutputV3{DangerLabel: danger, DangerScore: dangerScore, Incident: incident, IncidentConfidence: incidentScore, Task: task, TaskConfidence: taskScore, Action: action, ActionConfidence: actionScore, Communication: core.CommunicationIntentV3{Intent: communication, TemplateID: core.TemplateIDV3(communication)}, CommunicationConfidence: communicationScore, ModelVersion: "cognitive-v3:" + m.Manifest.EncoderVersion}, latency, nil
}
