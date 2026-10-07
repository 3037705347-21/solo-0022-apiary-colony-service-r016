package domain

import "strings"

// MeasurementRulesVersion 固化在每条巡检中的换算规则版本。
// 规则表只在接收时执行一次并把结果写入记录；后续即便规则升级，
// 历史巡检与其关注项仍按保存时的版本解读，不做追溯改写。
const MeasurementRulesVersion = "v1"

// MeasurementKind 标识一个量测口径所属的维度。不同维度的单位互不兼容。
type MeasurementKind string

const (
	KindStrength MeasurementKind = "strength"
	KindMite     MeasurementKind = "mite"
)

// Unit 是巡检提交时携带的量测单位。
type Unit string

const (
	// 群势：grade 为规范口径（1..5 级），frame 为现场满框数口径。
	UnitGrade Unit = "grade"
	UnitFrame Unit = "frame"
	// 螨计数：per100 为规范口径（每百只成蜂螨数），per300 为三百只抽样口径。
	UnitPer100 Unit = "per100"
	UnitPer300 Unit = "per300"
)

// Measurement 是一次量测在统一口径下的固化结果：原始记录与规范值同时保存。
type Measurement struct {
	// RawValue 是提交时按 RawUnit 记录的原始数值，不做追溯改写。
	RawValue int `json:"rawValue"`
	// RawUnit 是提交时携带的单位。
	RawUnit Unit `json:"rawUnit"`
	// Value 是接收时换算到规范单位后的数值，范围校验与关注项都基于它。
	Value int `json:"value"`
	// Unit 始终是该维度的规范单位，同一蜂箱的巡检与汇总呈现统一使用它。
	Unit Unit `json:"unit"`
	// RulesVersion 记录接收时生效的换算规则版本。
	RulesVersion string `json:"rulesVersion"`
}

// unitRule 描述一个单位所属维度、原始取值范围、是否规范单位以及到规范单位的换算系数。
// 系数以分子/分母表示，换算结果按远离零方向四舍五入；规范单位系数为 1/1。
type unitRule struct {
	kind        MeasurementKind
	minRaw      int
	maxRaw      int
	canonical   Unit
	scaleNum    int
	scaleDen    int
	isCanonical bool
}

// measurementRules 是 v1 版本唯一允许的单位表。新增或修改规则必须发布新版本，
// 不能改动本表中已保存记录所依赖的语义。
var measurementRules = map[Unit]unitRule{
	UnitGrade:  {kind: KindStrength, minRaw: 1, maxRaw: 5, canonical: UnitGrade, scaleNum: 1, scaleDen: 1, isCanonical: true},
	UnitFrame:  {kind: KindStrength, minRaw: 1, maxRaw: 10, canonical: UnitGrade, scaleNum: 1, scaleDen: 2},
	UnitPer100: {kind: KindMite, minRaw: 0, maxRaw: 60, canonical: UnitPer100, scaleNum: 1, scaleDen: 1, isCanonical: true},
	UnitPer300: {kind: KindMite, minRaw: 0, maxRaw: 180, canonical: UnitPer100, scaleNum: 1, scaleDen: 3},
}

// CanonicalBounds 返回维度规范口径的取值范围，用于换算后的二次范围校验。
func canonicalBounds(kind MeasurementKind) (int, int, bool) {
	for _, rule := range measurementRules {
		if rule.kind == kind && rule.isCanonical {
			return rule.minRaw, rule.maxRaw, true
		}
	}
	return 0, 0, false
}

// NormalizeMeasurement 校验单位合规性与维度兼容性，按 v1 规则换算到规范口径，
// 并对原始值与换算值分别做范围校验。未知单位、跨维度单位、越界值都显式报错，
// 不会被静默接受。单位问题归到 unitField，数值越界归到 valueField。
func NormalizeMeasurement(valueField, unitField string, value int, rawUnit string, kind MeasurementKind) (Measurement, error) {
	unit := Unit(strings.TrimSpace(rawUnit))
	if unit == "" {
		return Measurement{}, Invalid(unitField, "%s 必须携带量测单位", valueField)
	}
	rule, ok := measurementRules[unit]
	if !ok {
		return Measurement{}, Invalid(unitField, "%s 单位 %q 不受支持", valueField, unit)
	}
	if rule.kind != kind {
		return Measurement{}, Invalid(unitField, "%s 单位 %q 与该量测不兼容", valueField, unit)
	}
	if err := ValidateIntRange(valueField, value, rule.minRaw, rule.maxRaw); err != nil {
		return Measurement{}, err
	}
	canonicalValue := roundAwayFromZero(value*rule.scaleNum, rule.scaleDen)
	minValue, maxValue, _ := canonicalBounds(kind)
	if err := ValidateIntRange(valueField, canonicalValue, minValue, maxValue); err != nil {
		return Measurement{}, err
	}
	return Measurement{
		RawValue:     value,
		RawUnit:      unit,
		Value:        canonicalValue,
		Unit:         rule.canonical,
		RulesVersion: MeasurementRulesVersion,
	}, nil
}

// roundAwayFromZero 计算 num/den，并按远离零方向四舍五入。当前量测值均非负。
func roundAwayFromZero(num, den int) int {
	quotient, remainder := num/den, num%den
	if 2*remainder >= den {
		return quotient + 1
	}
	return quotient
}
