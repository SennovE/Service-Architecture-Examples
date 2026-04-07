package kafka

import (
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avrov2"
)

type MoviesProducer struct {
	t string
	p *kafka.Producer
	s *avrov2.Serializer
}

func New(topic string, servers []string, serializerURL string) (*MoviesProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(servers, ","),
		"acks":               "all",
		"retries":            10,
		"retry.backoff.ms":   100,
		"enable.idempotence": true,
	})
	if err != nil {
		return nil, fmt.Errorf("producer error: %v", err)
	}

	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig(serializerURL))
	if err != nil {
		return nil, fmt.Errorf("schemaregistry error: %v", err)
	}
	sCfg := avrov2.NewSerializerConfig()
	sCfg.AutoRegisterSchemas = false
	sCfg.UseLatestVersion = true
	s, err := avrov2.NewSerializer(srClient, serde.ValueSerde, sCfg)
	if err != nil {
		return nil, fmt.Errorf("avro error: %v", err)
	}

	return &MoviesProducer{t: topic, p: p, s: s}, nil
}
