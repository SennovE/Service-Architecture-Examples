package main

import (
	"booking/internal/config"
	"booking/internal/gen/api"
	"booking/internal/gen/proto"
	"booking/internal/provider"
	"booking/internal/repository"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	ginmw "github.com/oapi-codegen/gin-middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type apiKeyCreds struct{ key string }

func (a apiKeyCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"x-api-key": a.key}, nil
}

func (a apiKeyCreds) RequireTransportSecurity() bool { return false }

var retryPolicy = `{
	"methodConfig": [{
	"name": [{"service": "flights"}],
	"retryPolicy": {
		"MaxAttempts": 3,
		"InitialBackoff": ".1s",
		"MaxBackoff": ".4s",
		"BackoffMultiplier": 2.0,
		"RetryableStatusCodes": [ "UNAVAILABLE", "DEADLINE_EXCEEDED" ]
	}
}]}`

func main() {
	router := gin.Default()

	swagger, _ := api.GetSwagger()
	swagger.Servers = nil
	router.StaticFile("/openapi.yaml", "./api/openapi.yaml")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/openapi.yaml"),
	))

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

	creds := apiKeyCreds{key: cfg.ApiKey}
	grcpConn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", cfg.GRPCHost, cfg.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(creds),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer grcpConn.Close()
	client := proto.NewFlightsClient(grcpConn)

	prov := provider.NewProvider(dbc, client)

	router.Use(ginmw.OapiRequestValidatorWithOptions(
		swagger, &ginmw.Options{
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
					return nil
				},
			},
			ErrorHandler: func(c *gin.Context, msg string, statusCode int) {
				c.JSON(statusCode, api.ErrorResponse{
					ErrorCode: "VALIDATION_ERROR",
					Message:   msg,
				})
			},
		},
	))

	router.Use(provider.CircuitBreakerMiddleware(10 * time.Second))
	api.RegisterHandlers(router, prov)

	router.Run(fmt.Sprintf(":%d", cfg.AppPort))
}
