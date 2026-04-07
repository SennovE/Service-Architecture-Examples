package main

import (
	"flights/internal/config"
	"flights/internal/provider"
	"flights/internal/repository"
	"flights/internal/service"
	"fmt"
	"log"
	"net"
	"time"

	pb "flights/internal/gen/proto"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Validation Errors: \n%s", err)
	}

	dbc, err := repository.GetDBConnecion(
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)
	if err != nil {
		log.Fatalf("Error while connecting to DB: %s", err)
	}
	log.Printf("Postgres connected to %s:%d\n", cfg.PostgresHost, cfg.PostgresPort)

	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    cfg.RedisMasterName,
		SentinelAddrs: cfg.RedisReplicas,
		Password:      "",
		DB:            0,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
	})

	repo := repository.NewFlightsRepository(dbc)
	srvc := service.NewFlightsService(repo)
	prov := provider.NewServer(srvc, rdb)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.AppPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on 0.0.0.0:%d\n", cfg.AppPort)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(provider.APIKeyUnaryInterceptor(cfg.ApiKey)),
		grpc.StreamInterceptor(provider.APIKeyStreamInterceptor(cfg.ApiKey)),
	)
	pb.RegisterFlightsServer(grpcServer, prov)
	reflection.Register(grpcServer)
	grpcServer.Serve(lis)
}
