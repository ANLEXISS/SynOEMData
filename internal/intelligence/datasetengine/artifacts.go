package datasetengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type manifestFile struct {
	Dataset DatasetManifest `json:"dataset"`
	Homes   []HomeArtifact  `json:"homes"`
	Index   []ExampleIndex  `json:"example_index"`
}

// WriteArtifacts persists the four visibility domains into separate trees.
// Model-input files contain only an integrity envelope and ModelInput data,
// never scenario labels, split metadata, late source records, or hidden truth.
func (d Dataset) WriteArtifacts(root string) error {
	if root == "" {
		return fmt.Errorf("artifact root is required")
	}
	if err := d.Validate(); err != nil {
		return fmt.Errorf("refuse to write invalid dataset: %w", err)
	}
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return fmt.Errorf("artifact output directory must be empty: %s", root)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect artifact output directory: %w", err)
	}
	directories := []string{"ground-truth", "source-emissions", "availability", "model-inputs", "targets"}
	for _, directory := range directories {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", directory, err)
		}
	}
	if err := writeJSON(filepath.Join(root, "manifest.json"), manifestFile{Dataset: d.Manifest, Homes: d.Homes, Index: d.Index}); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	for _, artifact := range d.GroundTruth {
		if err := writeJSON(artifactPath(root, "ground-truth", artifact.Meta.ExampleID), artifact); err != nil {
			return fmt.Errorf("write ground truth %s: %w", artifact.Meta.ExampleID, err)
		}
	}
	for _, artifact := range d.SourceEmissions {
		if err := writeJSON(artifactPath(root, "source-emissions", artifact.Meta.ExampleID), artifact); err != nil {
			return fmt.Errorf("write source emissions %s: %w", artifact.Meta.ExampleID, err)
		}
	}
	for _, artifact := range d.Availability {
		if err := writeJSON(artifactPath(root, "availability", artifact.Meta.ExampleID), artifact); err != nil {
			return fmt.Errorf("write availability %s: %w", artifact.Meta.ExampleID, err)
		}
	}
	for _, artifact := range d.Inputs {
		input := ModelInputArtifact{Meta: artifact.Meta, Data: ModelInput{Batch: artifact.Batch, Encoded: artifact.Encoded, Availability: artifact.Availability}}
		if err := writeJSON(artifactPath(root, "model-inputs", artifact.Meta.ExampleID), input); err != nil {
			return fmt.Errorf("write model input %s: %w", artifact.Meta.ExampleID, err)
		}
	}
	for _, artifact := range d.Targets {
		if err := writeJSON(artifactPath(root, "targets", artifact.Meta.ExampleID), artifact); err != nil {
			return fmt.Errorf("write targets %s: %w", artifact.Meta.ExampleID, err)
		}
	}
	return nil
}

func artifactPath(root, directory, exampleID string) string {
	return filepath.Join(root, directory, exampleID+".json")
}

func writeJSON(path string, value any) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".artifact-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
