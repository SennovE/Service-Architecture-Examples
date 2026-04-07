package main

import (
	"auth/internal/config"
	"auth/internal/gen"
	"auth/internal/middleware"
	"auth/internal/provider"
	"auth/internal/repository"
	"auth/internal/service"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	ginmw "github.com/oapi-codegen/gin-middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	router := gin.New()
	router.Use(gin.Recovery())

	swagger, _ := gen.GetSwagger()
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

	repo := repository.NewUserRepository(dbc)
	srvc := service.NewUserService(repo, &cfg)
	prov := provider.NewUserProvider(srvc)
	server := gen.ServerInterfaceWrapper{
		Handler: prov,
	}

	router.Use(middleware.RequestIDMIddleware())
	router.Use(middleware.LogRequest())
	router.Use(ginmw.OapiRequestValidatorWithOptions(
		swagger, &ginmw.Options{
			SilenceServersWarning: true,
			ErrorHandler: func(c *gin.Context, msg string, statusCode int) {
				c.JSON(statusCode, gen.ErrorResponse{
					ErrorCode: gen.VALIDATIONERROR,
					Message:   "validation error",
					Details:   &map[string]any{"error": msg},
				})
			},
		},
	))

	router.POST("/auth/login", server.Login)
	router.POST("/auth/refresh", server.RefreshToken)
	router.POST("/auth/register", server.RegisterUser)

	router.Run(fmt.Sprintf(":%d", cfg.ApiPort))
}
