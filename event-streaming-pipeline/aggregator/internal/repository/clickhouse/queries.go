package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type TopFilm struct {
	MovieID string `ch:"movie_id"`
	Views   uint64 `ch:"views"`
}

type Metrics struct {
	Dau            uint64
	D1             float64
	D7             float64
	MeanWatchTime  float64
	ConversionRate float64
	TopFilms       []TopFilm
}

type MoviesClickhouse struct {
	conn driver.Conn
}

func New(host string, port int, user string, password string, dbName string) (*MoviesClickhouse, error) {
	conn, err := Connect(host, port, user, password, dbName)
	return &MoviesClickhouse{conn: conn}, err
}

func (c *MoviesClickhouse) RecomputeMetrics(ctx context.Context, date time.Time) (*Metrics, error) {
	dau, err := c.UpdateDAU(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("while update DAU error: %w", err)
	}
	meanWatchTime, err := c.UpdateMeanWatchTime(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("while update UpdateMeanWatchTime error: %w", err)
	}
	topFilms, err := c.UpdateTopFilms(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("while update UpdateTopFilms error: %w", err)
	}
	conversionRate, err := c.UpdateConversionRate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("while update UpdateConversionRate error: %w", err)
	}
	retention := make([]float64, 8)
	for i := range len(retention) {
		r, err := c.UpdateRetention(ctx, i, date)
		if err != nil {
			return nil, err
		}
		retention[i] = r
	}

	return &Metrics{
		Dau:            dau,
		D1:             retention[1],
		D7:             retention[7],
		MeanWatchTime:  meanWatchTime,
		ConversionRate: conversionRate,
		TopFilms:       topFilms,
	}, nil
}

func (c *MoviesClickhouse) CountTotalRows(ctx context.Context, date time.Time) (uint64, error) {
	const query = `
		SELECT COUNT()
		FROM analytics.movie_events
		WHERE event_date = $1
	`
	var total uint64
	err := c.conn.QueryRow(ctx, query, date).Scan(&total)
	return total, err
}

func (c *MoviesClickhouse) UpsertDailyMetrics(
	ctx context.Context, date time.Time, value float64, metricName string) error {
	const query = `
		INSERT INTO analytics.daily_metrics (metric_date, metric_name, metric_value)
		VALUES ($1, $2, $3)
	`
	return c.conn.Exec(ctx, query, date, metricName, value)
}

func (c *MoviesClickhouse) UpsertTopFilms(ctx context.Context, date time.Time, topFilms []TopFilm) error {
	const query = "INSERT INTO analytics.top_daily_films (metric_date, movie_id, views)"

	batch, err := c.conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}

	for _, film := range topFilms {
		if err := batch.Append(
			date,
			film.MovieID,
			film.Views,
		); err != nil {
			return err
		}
	}

	return batch.Send()
}

func (c *MoviesClickhouse) UpdateDAU(ctx context.Context, date time.Time) (uint64, error) {
	const query = `
		SELECT uniqExact(user_id)
		FROM analytics.movie_events
		WHERE event_date = $1
	`
	var dau uint64
	err := c.conn.QueryRow(ctx, query, date).Scan(&dau)
	if err != nil {
		return dau, err
	}
	return dau, c.UpsertDailyMetrics(ctx, date, float64(dau), "dau")
}

func (c *MoviesClickhouse) UpdateMeanWatchTime(ctx context.Context, date time.Time) (float64, error) {
	const query = `
		SELECT avgOrDefault(progress_seconds)
		FROM analytics.movie_events
		WHERE event_date = $1 AND event_type = 'VIEW_FINISHED'
	`
	var meanWatchTime float64
	err := c.conn.QueryRow(ctx, query, date).Scan(&meanWatchTime)
	if err != nil {
		return meanWatchTime, err
	}
	return meanWatchTime, c.UpsertDailyMetrics(ctx, date, meanWatchTime, "mean_watch_time")
}

func (c *MoviesClickhouse) UpdateConversionRate(ctx context.Context, date time.Time) (float64, error) {
	const query = `
		SELECT
			if(started = 0, 0, finished / started) AS conversion_rate
		FROM
		(
			SELECT
				uniqExactIf(session_id, event_type = 'VIEW_STARTED') AS started,
				uniqExactIf(session_id, event_type = 'VIEW_FINISHED') AS finished
			FROM analytics.movie_events
			WHERE event_date = $1
		)
	`
	var conversionRate float64
	err := c.conn.QueryRow(ctx, query, date).Scan(&conversionRate)
	if err != nil {
		return conversionRate, err
	}
	return conversionRate, c.UpsertDailyMetrics(ctx, date, conversionRate, "conversion_rate")
}

func (c *MoviesClickhouse) UpdateTopFilms(ctx context.Context, date time.Time) ([]TopFilm, error) {
	const query = `
		SELECT
			movie_id,
			count() AS views
		FROM analytics.movie_events
		WHERE event_date = $1 AND event_type = 'VIEW_STARTED'
		GROUP BY movie_id
		ORDER BY views DESC
		LIMIT 10
	`
	top_films := make([]TopFilm, 0, 10)
	err := c.conn.Select(ctx, &top_films, query, date)
	if err != nil {
		return nil, err
	}
	return top_films, c.UpsertTopFilms(ctx, date, top_films)
}

func (c *MoviesClickhouse) UpdateRetention(ctx context.Context, retDay int, date time.Time) (float64, error) {
	const query = `
		WITH
			cohort_users AS (
				SELECT user_id
				FROM analytics.movie_events
				GROUP BY user_id
				HAVING min(event_date) = $1
			),
			cohort_size AS (
				SELECT count() AS size FROM cohort_users
			),
			retained_day AS (
				SELECT uniqExact(e.user_id) AS retained
				FROM cohort_users c
				INNER JOIN analytics.movie_events e ON e.user_id = c.user_id
				WHERE e.event_date = addDays(toDate($1), $2)
			)
		SELECT if(
			(SELECT size FROM cohort_size) = 0,
			0,
			(SELECT retained FROM retained_day) / (SELECT size FROM cohort_size)
		)
	`
	var retention float64
	err := c.conn.QueryRow(ctx, query, date, retDay).Scan(&retention)
	if err != nil {
		return retention, err
	}
	return retention, c.UpsertDailyMetrics(ctx, date, retention, fmt.Sprintf("d%d", retDay))
}
