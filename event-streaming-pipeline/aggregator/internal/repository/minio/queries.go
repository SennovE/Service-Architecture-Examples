package minio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

type TopFilm struct {
	MovieID string `json:"movie_id"`
	Views   int
}

type Metrics struct {
	Dau            int
	D1             float32
	D7             float32
	MeanWatchTime  float32   `json:"mean_watch_time"`
	ConversionRate float32   `json:"conversion_rate"`
	TopFilms       []TopFilm `json:"top_films"`
}

type MoviesMinIO struct {
	conn   *minio.Client
	bucket string
}

func New(host string, port int, user string, password string, bucket string) (*MoviesMinIO, error) {
	conn, err := Connect(host, port, user, password, bucket)
	return &MoviesMinIO{conn: conn, bucket: bucket}, err
}

func (m *MoviesMinIO) PutMetricsJSON(ctx context.Context, metrics Metrics, date time.Time) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(jsonData)
	path := fmt.Sprintf("daily/%s/aggregates.json", date.Format(time.DateOnly))
	_, err = m.conn.PutObject(
		ctx,
		m.bucket,
		path,
		reader,
		int64(len(jsonData)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		},
	)
	return err
}
