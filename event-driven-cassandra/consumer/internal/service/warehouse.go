package service

import (
	"context"
	"log"

	"consumer/internal/models"
)

type Store interface {
	ApplyWarehouseEvent(ctx context.Context, event models.WarehouseEvent, meta models.KafkaMetadata) (models.ApplyResult, error)
	Health(ctx context.Context) error
}

type WarehouseService struct {
	store Store
}

func New(store Store) *WarehouseService {
	return &WarehouseService{store: store}
}

func (s *WarehouseService) Health(ctx context.Context) error {
	return s.store.Health(ctx)
}

func (s *WarehouseService) ProcessEvent(ctx context.Context, event models.WarehouseEvent, meta models.KafkaMetadata) error {
	result, err := s.store.ApplyWarehouseEvent(ctx, event, meta)
	if err != nil {
		return err
	}
	if result.Duplicate {
		log.Printf(
			"duplicate warehouse event skipped event_id=%s event_type=%s partition=%d offset=%d",
			event.EventID,
			event.EventType,
			meta.Partition,
			meta.Offset,
		)
		return nil
	}
	if result.Skipped {
		log.Printf(
			"out-of-order warehouse event skipped event_id=%s event_type=%s sequence_number=%d partition=%d offset=%d",
			event.EventID,
			event.EventType,
			event.SequenceNumber,
			meta.Partition,
			meta.Offset,
		)
		return nil
	}
	log.Printf(
		"warehouse event processed event_id=%s event_type=%s schema_version=%d sequence_number=%d partition=%d offset=%d",
		event.EventID,
		event.EventType,
		event.SchemaVersion,
		event.SequenceNumber,
		meta.Partition,
		meta.Offset,
	)
	return nil
}
