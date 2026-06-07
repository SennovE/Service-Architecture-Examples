package main

import (
	"consumer/internal/config"
	"consumer/internal/provider/api"
	"consumer/internal/repository/cassandra"
	"consumer/internal/repository/kafka"
	"consumer/internal/service"
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/ilyakaznacheev/cleanenv"
)

func main() {
	var cfg config.Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	store, err := cassandra.New(cfg.CassandraContactPoints, cfg.CassandraKeyspace, cfg.CassandraLocalDC)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	srvc := service.New(store)
	consumer, err := kafka.New(
		cfg.WarehouseTopic,
		cfg.DLQTopic,
		cfg.KafkaConsumerGroup,
		cfg.KafkaServers,
		cfg.SchemaRegistryURL,
		srvc,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go consumer.ListenMessages(ctx)

	api.Run(srvc, consumer)
}
