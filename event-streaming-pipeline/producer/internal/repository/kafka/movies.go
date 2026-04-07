package kafka

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"producer/internal/gen/api"
)

type AvroSchema struct {
	EventId         string    `avro:"event_id"`
	UserId          string    `avro:"user_id"`
	MovieId         string    `avro:"movie_id"`
	EventType       string    `avro:"event_type"`
	Timestamp       time.Time `avro:"timestamp"`
	DeviceType      string    `avro:"device_type"`
	SessionId       string    `avro:"session_id"`
	ProgressSeconds int       `avro:"progress_seconds"`
}

func (p *MoviesProducer) PostEvent(ctx context.Context, event api.MovieEvent) error {
	record := &AvroSchema{
		EventId:         event.EventId.String(),
		UserId:          event.UserId,
		MovieId:         event.MovieId,
		EventType:       string(event.EventType),
		Timestamp:       event.Timestamp,
		DeviceType:      string(event.DeviceType),
		SessionId:       event.SessionId,
		ProgressSeconds: event.ProgressSeconds,
	}

	payload, err := p.s.Serialize(p.t, record)
	if err != nil {
		return err
	}

	deliveryCh := make(chan kafka.Event, 1)
	err = p.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &p.t,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(event.UserId),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.EventId.String())},
			{Key: "event_type", Value: []byte(event.EventType)},
		},
	}, deliveryCh)
	if err != nil {
		return err
	}

	select {
	case e := <-deliveryCh:
		msg := e.(*kafka.Message)
		if msg.TopicPartition.Error != nil {
			return msg.TopicPartition.Error
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
