package kafka

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	appmetrics "consumer/internal/metrics"
	"consumer/internal/models"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	lagUpdateInterval   = 5 * time.Second
	kafkaQueryTimeoutMs = 5000
	kafkaConsumerPollMs = 1000
)

func (c *WarehouseConsumer) ListenMessages(ctx context.Context) {
	log.Printf("warehouse consumer subscribed topic=%s dlq_topic=%s", c.topic, c.dlqTopic)

	lagTicker := time.NewTicker(lagUpdateInterval)
	defer lagTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lagTicker.C:
			c.updateAssignedLags()
		default:
			ev := c.c.Poll(kafkaConsumerPollMs)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				if err := c.processMessage(ctx, e); err != nil {
					log.Printf(
						"message was not committed topic=%s partition=%d offset=%d error=%v",
						topicName(e),
						e.TopicPartition.Partition,
						e.TopicPartition.Offset,
						err,
					)
				}
			case kafka.Error:
				log.Printf("kafka error code=%v error=%v", e.Code(), e)
			default:
				log.Printf("ignored kafka event %T", e)
			}
		}
	}
}

func (c *WarehouseConsumer) processMessage(ctx context.Context, msg *kafka.Message) error {
	startedAt := time.Now()
	meta := models.KafkaMetadata{
		Topic:     topicName(msg),
		Partition: int(msg.TopicPartition.Partition),
		Offset:    int64(msg.TopicPartition.Offset),
	}

	var event models.WarehouseEvent
	if err := c.d.DeserializeInto(c.topic, msg.Value, &event); err != nil {
		return c.sendDLQAndCommit(
			ctx,
			msg,
			meta,
			nil,
			models.WithCode(models.ErrorCodeDeserialization, fmt.Errorf("deserialize avro: %w", err)),
		)
	}

	if err := c.h.ProcessEvent(ctx, event, meta); err != nil {
		if models.ErrorCode(err) == models.ErrorCodeCassandra {
			appmetrics.CassandraWriteErrorsTotal.Inc()
		}
		return c.sendDLQAndCommit(ctx, msg, meta, &event, err)
	}

	appmetrics.EventsProcessedTotal.WithLabelValues(event.EventType).Inc()
	appmetrics.EventProcessingDurationSeconds.WithLabelValues(event.EventType).Observe(time.Since(startedAt).Seconds())

	return c.commit(msg)
}

func (c *WarehouseConsumer) sendDLQAndCommit(
	ctx context.Context,
	msg *kafka.Message,
	meta models.KafkaMetadata,
	event *models.WarehouseEvent,
	cause error,
) error {
	if err := c.sendDLQ(ctx, msg, meta, event, cause); err != nil {
		return err
	}
	return c.commit(msg)
}

func (c *WarehouseConsumer) sendDLQ(
	ctx context.Context,
	msg *kafka.Message,
	meta models.KafkaMetadata,
	event *models.WarehouseEvent,
	cause error,
) error {
	originalEvent, err := originalEventPayload(msg.Value, event)
	if err != nil {
		return err
	}
	record := &models.WarehouseDLQEvent{
		OriginalEvent: originalEvent,
		ErrorCode:     models.ErrorCode(cause),
		ErrorReason:   cause.Error(),
		FailedAt:      time.Now().UTC(),
		KafkaMetadata: meta,
	}
	payload, err := c.s.Serialize(c.dlqTopic, record)
	if err != nil {
		return fmt.Errorf("serialize dlq: %w", err)
	}

	deliveryCh := make(chan kafka.Event, 1)
	key := fmt.Appendf(nil, "%s-%d-%d", meta.Topic, meta.Partition, meta.Offset)
	if event != nil && event.EventID != "" {
		key = []byte(event.EventID)
	}
	if err := c.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &c.dlqTopic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: payload,
	}, deliveryCh); err != nil {
		return fmt.Errorf("produce dlq: %w", err)
	}

	select {
	case e := <-deliveryCh:
		result := e.(*kafka.Message)
		if result.TopicPartition.Error != nil {
			return fmt.Errorf("deliver dlq: %w", result.TopicPartition.Error)
		}
		log.Printf(
			"warehouse event sent to dlq topic=%s partition=%d offset=%d reason=%s",
			meta.Topic,
			meta.Partition,
			meta.Offset,
			cause.Error(),
		)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *WarehouseConsumer) commit(msg *kafka.Message) error {
	partitions, err := c.c.CommitMessage(msg)
	if err != nil {
		return fmt.Errorf("commit offset: %w", err)
	}
	c.updateAssignedLags()
	log.Printf(
		"committed kafka offset topic=%s partition=%d offset=%d committed=%v",
		topicName(msg),
		msg.TopicPartition.Partition,
		msg.TopicPartition.Offset,
		partitions,
	)
	return nil
}

func (c *WarehouseConsumer) updateAssignedLags() {
	assignedPartitions, err := c.c.Assignment()
	if err != nil {
		log.Printf("consumer lag assignment failed error=%v", err)
		return
	}
	if len(assignedPartitions) == 0 {
		return
	}

	committedPartitions, err := c.c.Committed(assignedPartitions, kafkaQueryTimeoutMs)
	if err != nil {
		log.Printf("consumer lag committed offsets failed error=%v", err)
		return
	}

	for _, partition := range committedPartitions {
		if partition.Topic == nil {
			continue
		}

		topic := *partition.Topic
		low, high, err := c.c.QueryWatermarkOffsets(topic, partition.Partition, kafkaQueryTimeoutMs)
		if err != nil {
			log.Printf("consumer lag watermark failed topic=%s partition=%d error=%v", topic, partition.Partition, err)
			continue
		}

		committedOffset := int64(partition.Offset)
		if committedOffset < 0 {
			committedOffset = low
		}

		lag := high - committedOffset
		if lag < 0 {
			lag = 0
		}
		appmetrics.ConsumerLag.WithLabelValues(topic, strconv.Itoa(int(partition.Partition))).Set(float64(lag))
	}
}

func originalEventPayload(raw []byte, event *models.WarehouseEvent) (string, error) {
	if event == nil {
		return base64.StdEncoding.EncodeToString(raw), nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal original event: %w", err)
	}
	return string(payload), nil
}

func topicName(msg *kafka.Message) string {
	if msg.TopicPartition.Topic == nil {
		return ""
	}
	return *msg.TopicPartition.Topic
}
