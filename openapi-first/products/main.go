package main

import (
	"context"
	"fmt"
	"log"
	"products/internal/config"
	"products/internal/gen"
	"products/internal/middleware"
	"products/internal/provider"
	"products/internal/repository"
	"products/internal/service"

	"github.com/getkin/kin-openapi/openapi3filter"
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

	productsRepo := repository.NewProductsRepository(dbc)
	productsSrvc := service.NewProductsService(productsRepo)

	promoCodesRepo := repository.NewPromoCodesRepository(dbc)
	promoCodesSrvc := service.NewPromoCodesService(promoCodesRepo)

	ordersRepo := repository.NewOrdersRepository(dbc)
	ordersSrvc := service.NewOrdersService(ordersRepo, &cfg)

	prov := provider.NewProvider(productsSrvc, promoCodesSrvc, ordersSrvc)
	server := gen.ServerInterfaceWrapper{Handler: prov}

	router.Use(middleware.RequestIDMIddleware())
	router.Use(middleware.LogRequest())
	router.Use(ginmw.OapiRequestValidatorWithOptions(
		swagger, &ginmw.Options{
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
					return nil
				},
			},
			ErrorHandler: func(c *gin.Context, msg string, statusCode int) {
				c.JSON(statusCode, gen.ErrorResponse{
					ErrorCode: gen.VALIDATIONERROR,
					Message:   "validation error",
					Details:   &map[string]any{"error": msg},
				})
			},
		},
	))

	router.GET(
		"/products",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "SELLER", "ADMIN"}),
		server.ListProducts,
	)
	router.POST(
		"/products",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"SELLER", "ADMIN"}),
		server.CreateProduct,
	)
	router.DELETE(
		"/products/:id",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"SELLER", "ADMIN"}),
		server.ArchiveProduct,
	)
	router.GET(
		"/products/:id",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "SELLER", "ADMIN"}),
		server.GetProductById)
	router.PUT(
		"/products/:id",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"SELLER", "ADMIN"}),
		server.UpdateProduct)

	router.POST(
		"/promo-codes",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"SELLER", "ADMIN"}),
		server.CreatePromoCode,
	)

	router.GET(
		"/orders/:id",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "ADMIN"}),
		server.GetOrderById,
	)
	router.POST(
		"/orders",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "ADMIN"}),
		server.CreateOrder,
	)
	router.POST(
		"/orders/:id/cancel",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "ADMIN"}),
		server.CancelOrder,
	)
	router.PUT(
		"/orders/:id",
		middleware.AuthMiddleware(cfg.AuthPublicKey, []string{"USER", "ADMIN"}),
		server.UpdateOrder,
	)

	router.Run(fmt.Sprintf(":%d", cfg.ApiPort))
}
