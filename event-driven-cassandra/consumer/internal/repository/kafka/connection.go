package kafka

import (
	"context"
	"fmt"
	"strings"

	"consumer/internal/models"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avrov2"
)

type Handler interface {
	ProcessEvent(ctx context.Context, event models.WarehouseEvent, meta models.KafkaMetadata) error
}

type WarehouseConsumer struct {
	topic    string
	dlqTopic string
	c        *kafka.Consumer
	p        *kafka.Producer
	d        *avrov2.Deserializer
	s        *avrov2.Serializer
	h        Handler
}

func New(topic string, dlqTopic string, groupID string, servers []string, serializerURL string, handler Handler) (*WarehouseConsumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(servers, ","),
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, fmt.Errorf("consumer: %w", err)
	}

	if err := c.SubscribeTopics([]string{topic}, nil); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("subscribe %s: %w", topic, err)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(servers, ","),
		"acks":               "all",
		"retries":            10,
		"retry.backoff.ms":   100,
		"enable.idempotence": true,
	})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("dlq producer: %w", err)
	}

	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig(serializerURL))
	if err != nil {
		_ = c.Close()
		p.Close()
		return nil, fmt.Errorf("schemaregistry: %w", err)
	}

	d, err := avrov2.NewDeserializer(srClient, serde.ValueSerde, avrov2.NewDeserializerConfig())
	if err != nil {
		_ = c.Close()
		p.Close()
		return nil, fmt.Errorf("avro deserializer: %w", err)
	}

	sCfg := avrov2.NewSerializerConfig()
	sCfg.AutoRegisterSchemas = false
	sCfg.UseLatestVersion = true
	s, err := avrov2.NewSerializer(srClient, serde.ValueSerde, sCfg)
	if err != nil {
		_ = c.Close()
		p.Close()
		return nil, fmt.Errorf("avro serializer: %w", err)
	}

	return &WarehouseConsumer{
		topic:    topic,
		dlqTopic: dlqTopic,
		c:        c,
		p:        p,
		d:        d,
		s:        s,
		h:        handler,
	}, nil
}

func (c *WarehouseConsumer) Close() {
	if c == nil {
		return
	}
	if c.p != nil {
		c.p.Flush(5000)
		c.p.Close()
	}
	if c.c != nil {
		_ = c.c.Close()
	}
}

func (c *WarehouseConsumer) Health(ctx context.Context) error {
	if c == nil || c.c == nil {
		return fmt.Errorf("kafka consumer is not initialized")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if _, err := c.c.GetMetadata(nil, false, 2000); err != nil {
		return fmt.Errorf("kafka metadata: %w", err)
	}
	return nil
}
