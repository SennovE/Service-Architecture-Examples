package provider

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"producer/internal/config"
	"producer/internal/gen/api"
	"producer/internal/service"
)

type Provider struct {
	s *service.MoviesService
}

func New(cfg config.Config) (api.ServerInterface, error) {
	s, err := service.New(cfg)
	return &Provider{s: s}, err
}

func (p *Provider) GetHealth(c *gin.Context) {
	c.JSON(
		http.StatusOK,
		api.OkResponse{
			Status: "OK",
		},
	)
}

func (p *Provider) PostEvents(c *gin.Context) {
	var req api.PostEventsJSONRequestBody
	if !validateRequestBody(c, &req) {
		return
	}
	if !req.DeviceType.Valid() || !req.EventType.Valid() {
		c.JSON(
			http.StatusUnprocessableEntity,
			api.ErrorResponse{
				ErrorCode: "VALIDATIONERROR",
				Message:   "invalid event_type or device_type",
			},
		)
		return
	}
	if req.ProgressSeconds < 0 {
		c.JSON(
			http.StatusUnprocessableEntity,
			api.ErrorResponse{
				ErrorCode: "VALIDATIONERROR",
				Message:   "progress_seconds must not be less than 0",
			},
		)
		return
	}
	if err := p.s.PostEvent(c, req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			api.ErrorResponse{
				ErrorCode: "KAFKA_ERROR",
				Message:   err.Error(),
			},
		)
		return
	}
	c.JSON(
		http.StatusOK,
		api.PublishEventResponse{
			EventId: req.EventId,
		},
	)
}

func (p *Provider) PostGenerate(c *gin.Context, params api.PostGenerateParams) {
	if err := p.s.GenerateEvents(c, params.Sessions); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			api.ErrorResponse{
				ErrorCode: "INTERNAL_ERROR",
				Message:   err.Error(),
			},
		)
		return
	}
	c.JSON(
		http.StatusOK,
		api.OkResponse{
			Status: "OK",
		},
	)
}

func validateRequestBody[T any](ctx *gin.Context, req *T) bool {
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(
			http.StatusUnprocessableEntity,
			api.ErrorResponse{
				ErrorCode: "VALIDATIONERROR",
				Message:   "invalid body",
			},
		)
		return false
	}
	return true
}
