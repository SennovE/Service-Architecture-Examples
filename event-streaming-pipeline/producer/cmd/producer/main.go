package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ilyakaznacheev/cleanenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"producer/internal/config"
	"producer/internal/gen/api"
	"producer/internal/provider"
)

func main() {
	router := gin.Default()

	swagger, _ := api.GetSwagger()
	swagger.Servers = nil
	router.StaticFile("/openapi.yaml", "./api/openapi.yaml")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/openapi.yaml"),
	))

	var cfg config.Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	app, err := provider.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	api.RegisterHandlers(router, app)
	router.Run(fmt.Sprintf(":%d", cfg.AppPort))
}
