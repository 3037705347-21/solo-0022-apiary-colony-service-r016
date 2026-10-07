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

// Inspection 是一次现场巡检的固化结果。
//
// 量测采用双口径固化：Raw* 保存岗位提交时的原始读数与单位（出处留痕），
// ColonyStrength/MiteCount 保存换算到标准口径后的整数，范围校验与关注项派生
// 只基于标准口径。所有数值在保存时一次性换算完成，之后不再随换算规则变化改写。
type Inspection struct {
	ID                    string       `json:"id"`
	HiveID                string       `json:"hiveId"`
	ObservedAt            time.Time    `json:"observedAt"`
	ColonyStrength        int          `json:"colonyStrength"`
	ColonyStrengthUnit    Unit         `json:"colonyStrengthUnit"`
	RawColonyStrength     int          `json:"rawColonyStrength"`
	RawColonyStrengthUnit Unit         `json:"rawColonyStrengthUnit"`
	BroodPattern          BroodPattern `json:"broodPattern"`
	Temper                Temper       `json:"temper"`
	MiteCount             int          `json:"miteCount"`
	MiteCountUnit         Unit         `json:"miteCountUnit"`
	RawMiteCount          int          `json:"rawMiteCount"`
	RawMiteCountUnit      Unit         `json:"rawMiteCountUnit"`
	ConversionVersion     string       `json:"conversionVersion"`
	Note                  string       `json:"note"`
	Concerns              []string     `json:"concerns"`
}

const (
	MinColonyStrength = 1
	MaxColonyStrength = 5
	MinMiteCount      = 0
	MaxMiteCount      = 60
	MaxNoteRunes      = 200
)

// InspectionMeasurement 携带一次巡检两个可换算量测的原始读数与提交单位。
type InspectionMeasurement struct {
	Strength     int
	StrengthUnit string
	MiteCount    int
	MiteUnit     string
}

// ResolvedMeasurement 是单位解析与标准口径换算后的结果。
type ResolvedMeasurement struct {
	StrengthCanonical int
	StrengthUnit      Unit
	StrengthRaw       int
	StrengthRawUnit   Unit
	MiteCanonical     int
	MiteUnit          Unit
	MiteRaw           int
	MiteRawUnit       Unit
}

// ResolveMeasurement 解析群势与螨计数的单位并换算到标准口径。
// 未知或与维度不兼容的单位直接返回校验错误，不被静默接受。
func ResolveMeasurement(input InspectionMeasurement) (ResolvedMeasurement, error) {
	strengthSpec, err := ResolveUnit("colonyStrengthUnit", KindColonyStrength, input.StrengthUnit)
	if err != nil {
		return ResolvedMeasurement{}, err
	}
	miteSpec, err := ResolveUnit("miteCountUnit", KindMiteCount, input.MiteUnit)
	if err != nil {
		return ResolvedMeasurement{}, err
	}
	strengthCanonical, err := NormalizeMeasurement("colonyStrength", KindColonyStrength, strengthSpec, input.Strength)
	if err != nil {
		return ResolvedMeasurement{}, err
	}
	miteCanonical, err := NormalizeMeasurement("miteCount", KindMiteCount, miteSpec, input.MiteCount)
	if err != nil {
		return ResolvedMeasurement{}, err
	}
	return ResolvedMeasurement{
		StrengthCanonical: strengthCanonical,
		StrengthUnit:      CanonicalUnit(KindColonyStrength),
		StrengthRaw:       input.Strength,
		StrengthRawUnit:   strengthSpec.code,
		MiteCanonical:     miteCanonical,
		MiteUnit:          CanonicalUnit(KindMiteCount),
		MiteRaw:           input.MiteCount,
		MiteRawUnit:       miteSpec.code,
	}, nil
}

// NewInspection 接收已经换算到标准口径的量测，完成其余字段校验并固化巡检。
// displayStrengthUnit/displayMiteUnit 是该蜂箱钉住的展示口径：返回给岗位的读数
// 会从标准口径换算到该口径，保证同一蜂箱的巡检与汇总呈现始终使用同一口径。
func NewInspection(
	hiveID string,
	observedAt time.Time,
	measurement ResolvedMeasurement,
	displayStrengthUnit Unit,
	displayMiteUnit Unit,
	brood BroodPattern,
	temper Temper,
	note string,
) (Inspection, error) {
	if err := ValidateIdentifier("hiveId", hiveID, 3, 40); err != nil {
		return Inspection{}, err
	}
	if observedAt.IsZero() {
		return Inspection{}, Invalid("observedAt", "observedAt 不能为空")
	}
	if err := ValidateIntRange("colonyStrength", measurement.StrengthCanonical, MinColonyStrength, MaxColonyStrength); err != nil {
		return Inspection{}, err
	}
	if !ValidBroodPattern(brood) {
		return Inspection{}, Invalid("broodPattern", "broodPattern 必须是 %s、%s 或 %s", BroodSolid, BroodPatchy, BroodNone)
	}
	if !ValidTemper(temper) {
		return Inspection{}, Invalid("temper", "temper 必须是 %s、%s 或 %s", TemperCalm, TemperNervous, TemperAggressive)
	}
	if err := ValidateIntRange("miteCount", measurement.MiteCanonical, MinMiteCount, MaxMiteCount); err != nil {
		return Inspection{}, err
	}
	if err := ValidateText("note", note, 0, MaxNoteRunes); err != nil {
		return Inspection{}, err
	}
	return Inspection{
		HiveID:                TrimText(hiveID),
		ObservedAt:            observedAt.UTC(),
		ColonyStrength:        ConvertToUnit(KindColonyStrength, measurement.StrengthCanonical, displayStrengthUnit),
		ColonyStrengthUnit:    displayStrengthUnit,
		RawColonyStrength:     measurement.StrengthRaw,
		RawColonyStrengthUnit: measurement.StrengthRawUnit,
		BroodPattern:          brood,
		Temper:                temper,
		MiteCount:             ConvertToUnit(KindMiteCount, measurement.MiteCanonical, displayMiteUnit),
		MiteCountUnit:         displayMiteUnit,
		RawMiteCount:          measurement.MiteRaw,
		RawMiteCountUnit:      measurement.MiteRawUnit,
		ConversionVersion:     ConversionVersion,
		Note:                  TrimText(note),
		Concerns:              []string{},
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
