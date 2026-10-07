package inspection

import "example.com/apiary-colony-service/internal/domain"

const (
	ConcernMiteLoad    = "mite_load"
	ConcernBroodGap    = "brood_gap"
	ConcernTemperRisk  = "temper_risk"
	ConcernLowStrength = "low_strength"
)

const (
	MiteLoadThreshold  = 9
	LowStrengthCeiling = 2
)

// DeriveConcerns 按标准口径上的固定阈值从一次巡检派生出关注项，顺序稳定。
// 传入的 measurement 已经换算到标准口径，因此阈值与提交单位无关、口径唯一。
func DeriveConcerns(measurement domain.ResolvedMeasurement, brood domain.BroodPattern, temper domain.Temper) []string {
	concerns := make([]string, 0, 4)
	if measurement.MiteCanonical >= MiteLoadThreshold {
		concerns = append(concerns, ConcernMiteLoad)
	}
	if brood == domain.BroodNone {
		concerns = append(concerns, ConcernBroodGap)
	}
	if temper == domain.TemperAggressive {
		concerns = append(concerns, ConcernTemperRisk)
	}
	if measurement.StrengthCanonical <= LowStrengthCeiling {
		concerns = append(concerns, ConcernLowStrength)
	}
	return concerns
}
