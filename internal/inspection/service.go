package inspection

import (
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

// Service 接收巡检、完成单位解析与标准口径换算、范围校验并派生关注项。
type Service struct {
	store *store.Memory
	clock func() time.Time
}

func New(memory *store.Memory, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: memory, clock: clock}
}

type RecordInput struct {
	ObservedAt     time.Time
	ColonyStrength int
	// StrengthUnit 为群势读数的单位编码，缺省按标准口径 scale5 处理。
	StrengthUnit string
	BroodPattern string
	Temper       string
	MiteCount    int
	// MiteUnit 为螨计数读数的单位编码，缺省按标准口径 per100 处理。
	MiteUnit string
	Note     string
}

func (s *Service) Record(hiveID string, input RecordInput) (domain.Inspection, error) {
	observedAt := input.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.clock()
	}
	brood := domain.BroodPattern(domain.TrimText(input.BroodPattern))
	temper := domain.Temper(domain.TrimText(input.Temper))

	// 接受前先按同一口径换算：未知/不兼容单位与越界读数在此被拒绝，绝不静默接受。
	measurement, err := domain.ResolveMeasurement(domain.InspectionMeasurement{
		Strength:     input.ColonyStrength,
		StrengthUnit: input.StrengthUnit,
		MiteCount:    input.MiteCount,
		MiteUnit:     input.MiteUnit,
	})
	if err != nil {
		return domain.Inspection{}, err
	}

	// 关注项只基于标准口径的固定阈值判定，并在保存时固化到记录上，
	// 之后换算规则或阈值调整都不会追溯改写已经保存的关注项。
	concerns := DeriveConcerns(measurement, brood, temper)

	draft := store.InspectionDraft{
		ObservedAt:   observedAt,
		Measurement:  measurement,
		BroodPattern: brood,
		Temper:       temper,
		Note:         input.Note,
		Concerns:     concerns,
	}
	saved, err := s.store.AddInspection(hiveID, draft)
	if err != nil {
		return domain.Inspection{}, err
	}
	return saved, nil
}

func (s *Service) List(hiveID string) ([]domain.Inspection, error) {
	return s.store.ListInspections(hiveID)
}
