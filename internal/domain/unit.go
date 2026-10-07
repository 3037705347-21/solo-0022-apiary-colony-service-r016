package domain

import "strings"

// MeasurementKind 标识一个量测所属的物理量维度。只有同一维度的单位才可以互相换算。
type MeasurementKind string

const (
	KindColonyStrength MeasurementKind = "colony_strength"
	KindMiteCount      MeasurementKind = "mite_count"
)

// Unit 是提交巡检时可携带的量测单位编码。
type Unit string

const (
	// 群势：五分制是标准口径，十分制按现场岗位习惯提交后换算。
	UnitStrengthScale5  Unit = "scale5"
	UnitStrengthScore10 Unit = "score10"
	// 螨计数：每百只成蜂是标准口径；按三百只样本计数的岗位提交后换算。
	UnitMitePer100 Unit = "per100"
	UnitMitePer300 Unit = "per300"
)

// ConversionVersion 标记保存巡检时所使用的换算规则版本。
// 记录固化后不随规则表演化而改写，保证换算规则变化不能追溯改写历史巡检。
const ConversionVersion = "unit-rules-v1"

// unitSpec 描述一个合规单位：所属维度、原始取值范围与到标准口径的整数换算。
// toCanonical 必须是确定性的纯函数；换算后的取值通过标准口径范围统一校验。
type unitSpec struct {
	code        Unit
	kind        MeasurementKind
	minRaw      int
	maxRaw      int
	toCanonical func(int) int
}

// unitRegistry 是唯一的合规单位来源；未注册的单位一律拒绝，不做静默兜底。
var unitRegistry = []unitSpec{
	{
		code:        UnitStrengthScale5,
		kind:        KindColonyStrength,
		minRaw:      MinColonyStrength,
		maxRaw:      MaxColonyStrength,
		toCanonical: func(v int) int { return v },
	},
	{
		code: UnitStrengthScore10,
		kind: KindColonyStrength,
		// 十分制为整数序数，1..10 全部有效；按半数向上收敛到五分制档级。
		minRaw:      1,
		maxRaw:      10,
		toCanonical: func(v int) int { return roundDiv(5*v, 10) },
	},
	{
		code:        UnitMitePer100,
		kind:        KindMiteCount,
		minRaw:      MinMiteCount,
		maxRaw:      MaxMiteCount,
		toCanonical: func(v int) int { return v },
	},
	{
		code: UnitMitePer300,
		kind: KindMiteCount,
		// 三百只样本上计数，标准口径上限 60 对应样本上限 180。
		minRaw:      0,
		maxRaw:      3 * MaxMiteCount,
		toCanonical: func(v int) int { return roundDiv(v, 3) },
	},
}

// CanonicalUnit 返回某一维度的标准口径单位，换算与阈值判定都以它为基准。
func CanonicalUnit(kind MeasurementKind) Unit {
	switch kind {
	case KindColonyStrength:
		return UnitStrengthScale5
	case KindMiteCount:
		return UnitMitePer100
	default:
		return ""
	}
}

// ResolveUnit 解析提交的单位编码：空串按标准口径处理；未注册或不属于该维度的
// 单位是不兼容单位，返回校验错误由入口拒绝，绝不静默当作标准口径接受。
func ResolveUnit(field string, kind MeasurementKind, raw string) (unitSpec, error) {
	code := Unit(strings.TrimSpace(raw))
	if code == "" {
		code = CanonicalUnit(kind)
	}
	for _, spec := range unitRegistry {
		if spec.code != code {
			continue
		}
		if spec.kind != kind {
			return unitSpec{}, Invalid(field, "%s 不能用于该量测项", code)
		}
		return spec, nil
	}
	return unitSpec{}, Invalid(field, "未知或不支持的量测单位 %q", code)
}

// NormalizeMeasurement 在接受巡检前把原始读数换算成标准口径：
// 先按提交单位自身的取值范围校验，再换算，最后按标准口径范围复核。
func NormalizeMeasurement(field string, kind MeasurementKind, spec unitSpec, raw int) (int, error) {
	if raw < spec.minRaw || raw > spec.maxRaw {
		return 0, Invalid(field, "%s 在单位 %q 下必须落在 %d 到 %d 之间", field, spec.code, spec.minRaw, spec.maxRaw)
	}
	canonical := spec.toCanonical(raw)
	minCanonical, maxCanonical := canonicalBounds(kind)
	if canonical < minCanonical || canonical > maxCanonical {
		return 0, Invalid(field, "%s 换算到标准口径后必须落在 %d 到 %d 之间", field, minCanonical, maxCanonical)
	}
	return canonical, nil
}

// ConvertToUnit 把标准口径读数换算到目标展示单位。标准口径到自身为恒等；
// 注册表中的换算只在保存巡检时使用一次，展示口径始终取自蜂箱首次巡检钉住的结果。
func ConvertToUnit(kind MeasurementKind, canonical int, target Unit) int {
	if target == CanonicalUnit(kind) {
		return canonical
	}
	switch kind {
	case KindColonyStrength:
		if target == UnitStrengthScore10 {
			return 2 * canonical
		}
	case KindMiteCount:
		if target == UnitMitePer300 {
			return 3 * canonical
		}
	}
	return canonical
}

func canonicalBounds(kind MeasurementKind) (int, int) {
	switch kind {
	case KindColonyStrength:
		return MinColonyStrength, MaxColonyStrength
	case KindMiteCount:
		return MinMiteCount, MaxMiteCount
	default:
		return 0, 0
	}
}

// roundDiv 实现非负整数的四舍五入除法（半数向上），用于离散档级之间的确定性换算。
func roundDiv(numerator, denominator int) int {
	if denominator <= 0 {
		return numerator
	}
	return (numerator + denominator/2) / denominator
}
