package models

type FeaturePreset string

const (
	AccountFeaturePresetPocket FeaturePreset = "pocket"
)

func (f FeaturePreset) String() string {
	return string(f)
}
