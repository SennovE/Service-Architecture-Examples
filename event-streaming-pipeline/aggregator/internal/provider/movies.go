package provider

import (
	"aggregator/internal/config"
	"aggregator/internal/gen/api"
	"aggregator/internal/service"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Provider struct {
	s *service.MoviesService
}

func New(cfg config.Config) (*Provider, error) {
	s, err := service.New(cfg)
	return &Provider{s: s}, err
}

func (p *Provider) RunCron(ctx context.Context, interval int) {
	recomputeMetrics := time.NewTicker(time.Duration(interval) * time.Second)
	loadMinIO := time.NewTicker(time.Duration(24) * time.Hour)
	defer recomputeMetrics.Stop()
	defer loadMinIO.Stop()

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-recomputeMetrics.C:
			p.s.RecomputeMetrics(ctx, time.Now())
		case <-loadMinIO.C:
			p.s.UploadToMinIO(ctx, time.Now())
		}
	}
}

func (p *Provider) GetHealth(c *gin.Context) {
	c.JSON(
		http.StatusOK,
		api.OkResponse{
			Status: "OK",
		},
	)
}

func (p *Provider) PostExport(c *gin.Context, params api.PostExportParams) {
	m, err := p.s.ExportMetrics(c, params.Date.Time)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			ErrorCode: "INTERNAL_SERVER_ERROR",
			Message:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (p *Provider) PostRecompute(c *gin.Context, params api.PostRecomputeParams) {
	err := p.s.RecomputeMetrics(c, params.Date.Time)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			ErrorCode: "INTERNAL_SERVER_ERROR",
			Message:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, api.OkResponse{
		Status: "OK",
	})
}

func (p *Provider) PostLoad(c *gin.Context, params api.PostLoadParams) {
	err := p.s.UploadToMinIO(c, params.Date.Time)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			ErrorCode: "INTERNAL_SERVER_ERROR",
			Message:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, api.OkResponse{
		Status: "OK",
	})
}
