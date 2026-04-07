package service

import (
	"aggregator/internal/config"
	"aggregator/internal/gen/api"
	"aggregator/internal/repository/clickhouse"
	"aggregator/internal/repository/minio"
	"aggregator/internal/repository/postgres"
	"context"
	"fmt"
	"log"
	"time"
)

type MoviesService struct {
	p *postgres.MoviesPostgres
	c *clickhouse.MoviesClickhouse
	m *minio.MoviesMinIO
}

func New(cfg config.Config) (*MoviesService, error) {
	p, err := postgres.New(cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDBName)
	if err != nil {
		return nil, err
	}
	c, err := clickhouse.New(cfg.ClickhouseHost, cfg.ClickhousePort, cfg.ClickhouseUser, cfg.ClickhousePassword, cfg.ClickhouseDBName)
	if err != nil {
		return nil, err
	}
	m, err := minio.New(cfg.MinIOHost, cfg.MinIOPort, cfg.MinIOUser, cfg.MinIOPassword, cfg.MinIOMetricsBucket)
	if err != nil {
		return nil, err
	}
	return &MoviesService{p: p, c: c, m: m}, nil
}

func (s *MoviesService) RecomputeMetrics(ctx context.Context, date time.Time) error {
	startedAt := time.Now()
	log.Printf("metrics recompute started for %s", date.Format(time.DateOnly))
	m, err := s.c.RecomputeMetrics(ctx, date)
	if err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}
	totalRows, err := s.c.CountTotalRows(ctx, date)
	if err != nil {
		return err
	}
	err = s.p.UpsertDailyMetrics(ctx, date, "dau", float64(m.Dau))
	if err != nil {
		return err
	}
	err = s.p.UpsertDailyMetrics(ctx, date, "d1", m.D1)
	if err != nil {
		return err
	}
	err = s.p.UpsertDailyMetrics(ctx, date, "d7", m.D7)
	if err != nil {
		return err
	}
	err = s.p.UpsertDailyMetrics(ctx, date, "conversion_rate", m.ConversionRate)
	if err != nil {
		return err
	}
	err = s.p.UpsertDailyMetrics(ctx, date, "mean_watch_time", m.MeanWatchTime)
	if err != nil {
		return err
	}
	for _, film := range m.TopFilms {
		err = s.p.UpsertTopDailyFilms(ctx, date, film.MovieID, int(film.Views))
		if err != nil {
			return err
		}
	}
	log.Printf(
		"metrics recompute finished for %s | time spend %f s | rows scaned %d",
		date.Format(time.DateOnly),
		time.Since(startedAt).Seconds(),
		totalRows,
	)
	return nil
}

func (s *MoviesService) ExportMetrics(ctx context.Context, date time.Time) (api.MetricsResponse, error) {
	m, err := s.p.GetDailyMetrics(ctx, date)
	if err != nil {
		return api.MetricsResponse{}, err
	}
	films, err := s.p.GetTopDailyFilms(ctx, date)
	if err != nil {
		return api.MetricsResponse{}, err
	}
	var metrics api.MetricsResponse
	metrics.TopFilms = make([]api.TopFilm, len(films))
	for i, film := range films {
		metrics.TopFilms[i] = api.TopFilm{
			MovieId: film.MovieID,
			Views:   int(film.Views),
		}
	}
	for _, metric := range m {
		switch metric.MetricName {
		case "dau":
			metrics.Dau = int(metric.MetricValue)
		case "d1":
			metrics.D1 = float32(metric.MetricValue)
		case "d7":
			metrics.D7 = float32(metric.MetricValue)
		case "conversion_rate":
			metrics.ConversionRate = float32(metric.MetricValue)
		case "mean_watch_time":
			metrics.MeanWatchTime = float32(metric.MetricValue)
		}
	}
	return metrics, nil
}

func (s *MoviesService) UploadToMinIO(ctx context.Context, date time.Time) error {
	m, err := s.ExportMetrics(ctx, date)
	if err != nil {
		return err
	}
	topFilms := make([]minio.TopFilm, len(m.TopFilms))
	for i, f := range m.TopFilms {
		topFilms[i] = minio.TopFilm{
			MovieID: f.MovieId,
			Views:   f.Views,
		}
	}
	err = s.m.PutMetricsJSON(ctx, minio.Metrics{
		Dau:            m.Dau,
		D1:             m.D1,
		D7:             m.D7,
		MeanWatchTime:  m.MeanWatchTime,
		ConversionRate: m.ConversionRate,
		TopFilms:       topFilms,
	}, date)
	return err
}
