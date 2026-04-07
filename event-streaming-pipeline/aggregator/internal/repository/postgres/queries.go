package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type TopFilm struct {
	MetricDate time.Time `db:"metric_date"`
	MovieID    string    `db:"movie_id"`
	Views      int       `db:"views"`
}

type DailyMetric struct {
	MetricDate  time.Time `db:"metric_date"`
	MetricName  string    `db:"metric_name"`
	MetricValue float64   `db:"metric_value"`
}

type MoviesPostgres struct {
	conn *sqlx.DB
}

func New(host string, port int, user string, password string, dbName string) (*MoviesPostgres, error) {
	conn, err := Connect(host, port, user, password, dbName)
	return &MoviesPostgres{conn: conn}, err
}

func (m *MoviesPostgres) UpsertDailyMetrics(
	ctx context.Context, metric_date time.Time, metric_name string, metric_value float64) error {
	const query = `
		INSERT INTO daily_metrics (metric_date, metric_name, metric_value)
		VALUES ($1, $2, $3)
		ON CONFLICT (metric_date, metric_name)
		DO UPDATE SET metric_value = $3;
	`
	_, err := m.conn.ExecContext(ctx, query, metric_date, metric_name, metric_value)
	return err
}

func (m *MoviesPostgres) UpsertTopDailyFilms(
	ctx context.Context, metric_date time.Time, movie_id string, views int) error {
	const query = `
		INSERT INTO top_daily_films (metric_date, movie_id, views)
		VALUES ($1, $2, $3)
		ON CONFLICT (metric_date, movie_id)
		DO UPDATE SET views = $3;
	`
	_, err := m.conn.ExecContext(ctx, query, metric_date, movie_id, views)
	return err
}

func (m *MoviesPostgres) GetDailyMetrics(ctx context.Context, metric_date time.Time) ([]DailyMetric, error) {
	const query = `
		SELECT metric_date, metric_name, metric_value
		FROM daily_metrics
		WHERE metric_date = $1;
	`
	var metrics []DailyMetric
	err := m.conn.SelectContext(ctx, &metrics, query, metric_date)
	return metrics, err
}

func (m *MoviesPostgres) GetTopDailyFilms(ctx context.Context, metric_date time.Time) ([]TopFilm, error) {
	const query = `
		SELECT metric_date, movie_id, views
		FROM top_daily_films
		WHERE metric_date = $1
		ORDER BY views DESC
		LIMIT 10;
	`
	var films []TopFilm
	err := m.conn.SelectContext(ctx, &films, query, metric_date)
	return films, err
}
