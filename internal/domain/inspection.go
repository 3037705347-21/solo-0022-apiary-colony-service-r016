package domain

import "time"

type BroodPattern string

const (
	BroodSolid  BroodPattern = "solid"
	BroodPatchy BroodPattern = "patchy"
	BroodNone   BroodPattern = "none"
)

type Temper string

const (
	TemperCalm       Temper = "calm"
	TemperNervous    Temper = "nervous"
	TemperAggressive Temper = "aggressive"
)

// Inspection 是一次现场巡检的固化结果，关注项由量测派生。
// ColonyStrength 与 MiteCount 始终以规范口径保存：接收时已按提交单位换算完成，
// 之后不再重新换算，因此规则升级不会改写已保存的巡检与关注项。
type Inspection struct {
	ID             string       `json:"id"`
	HiveID         string       `json:"hiveId"`
	ObservedAt     time.Time    `json:"observedAt"`
	ColonyStrength int          `json:"colonyStrength"`
	StrengthUnit   Unit         `json:"strengthUnit"`
	BroodPattern   BroodPattern `json:"broodPattern"`
	Temper         Temper       `json:"temper"`
	MiteCount      int          `json:"miteCount"`
	MiteUnit       Unit         `json:"miteUnit"`
	// Strength 与 Mite 保留提交时的原始数值、单位与规范换算结果。
	Strength Measurement `json:"strength"`
	Mite     Measurement `json:"mite"`
	Note     string      `json:"note"`
	Concerns []string    `json:"concerns"`
}

const (
	MinColonyStrength = 1
	MaxColonyStrength = 5
	MinMiteCount      = 0
	MaxMiteCount      = 60
	MaxNoteRunes      = 200
)

func NewInspection(
	hiveID string,
	observedAt time.Time,
	strengthValue int,
	strengthUnit string,
	brood BroodPattern,
	temper Temper,
	miteValue int,
	miteUnit string,
	note string,
) (Inspection, error) {
	if err := ValidateIdentifier("hiveId", hiveID, 3, 40); err != nil {
		return Inspection{}, err
	}
	if observedAt.IsZero() {
		return Inspection{}, Invalid("observedAt", "observedAt 不能为空")
	}
	strength, err := NormalizeMeasurement("colonyStrength", "strengthUnit", strengthValue, strengthUnit, KindStrength)
	if err != nil {
		return Inspection{}, err
	}
	if !ValidBroodPattern(brood) {
		return Inspection{}, Invalid("broodPattern", "broodPattern 必须是 %s、%s 或 %s", BroodSolid, BroodPatchy, BroodNone)
	}
	if !ValidTemper(temper) {
		return Inspection{}, Invalid("temper", "temper 必须是 %s、%s 或 %s", TemperCalm, TemperNervous, TemperAggressive)
	}
	mite, err := NormalizeMeasurement("miteCount", "miteUnit", miteValue, miteUnit, KindMite)
	if err != nil {
		return Inspection{}, err
	}
	if err := ValidateText("note", note, 0, MaxNoteRunes); err != nil {
		return Inspection{}, err
	}
	return Inspection{
		HiveID:         TrimText(hiveID),
		ObservedAt:     observedAt.UTC(),
		ColonyStrength: strength.Value,
		StrengthUnit:   strength.Unit,
		BroodPattern:   brood,
		Temper:         temper,
		MiteCount:      mite.Value,
		MiteUnit:       mite.Unit,
		Strength:       strength,
		Mite:           mite,
		Note:           TrimText(note),
		Concerns:       []string{},
	}, nil
}

func ValidBroodPattern(value BroodPattern) bool {
	switch value {
	case BroodSolid, BroodPatchy, BroodNone:
		return true
	default:
		return false
	}
}

func ValidTemper(value Temper) bool {
	switch value {
	case TemperCalm, TemperNervous, TemperAggressive:
		return true
	default:
		return false
	}
}
