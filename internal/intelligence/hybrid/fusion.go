package hybrid

import "fmt"

func NewGeneralFusion(event, topo BranchContribution) (FusionEnvelope, error) {
	envelope := FusionEnvelope{SchemaVersion: FusionSchemaVersion, Stage: FusionGeneral,
		Contributions: []BranchContribution{event, topo}}
	if err := envelope.Validate(); err != nil {
		return FusionEnvelope{}, err
	}
	if topo.State != ContributionAvailable && topo.State != ContributionPartial {
		return FusionEnvelope{}, fmt.Errorf("%w: required topo-temporal representation is unavailable", ErrComponentUnavailable)
	}
	if len(topo.Representations) != 1 {
		return FusionEnvelope{}, fmt.Errorf("%w: topo-temporal branch must provide one contextual representation", ErrIncompatible)
	}
	return envelope, nil
}

func NewCognitiveFusion(general FusionEnvelope, home BranchContribution) FusionEnvelope {
	copyOfGeneral := cloneFusion(general)
	return FusionEnvelope{SchemaVersion: FusionSchemaVersion, Stage: FusionCognitive,
		General: &copyOfGeneral, Contributions: []BranchContribution{home}}
}

func (i HomeAdaptationInput) Validate() error {
	if i.SchemaVersion != HomeInputSchemaVersion || i.Cutoff.IsZero() {
		return fmt.Errorf("%w: invalid home adaptation input schema", ErrIncompatible)
	}
	if err := i.General.Validate(); err != nil {
		return err
	}
	if i.General.Stage != FusionGeneral {
		return fmt.Errorf("%w: home adaptation requires general fusion", ErrIncompatible)
	}
	if i.Home.State == HomeAvailable && i.Home.TrainingCutoff.After(i.Cutoff) {
		return fmt.Errorf("%w: home adaptation training cutoff is in the future", ErrIncompatible)
	}
	return i.Home.Validate()
}
