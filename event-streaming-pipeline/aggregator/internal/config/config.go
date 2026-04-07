package config

type Config struct {
	AppPort int `env:"APP_PORT"`

	CronIntervalSeconds int `env:"CRON_INTERVAL_SECONDS"`

	PostgresHost     string `env:"POSTGRES_HOST"`
	PostgresPort     int    `env:"POSTGRES_PORT"`
	PostgresUser     string `env:"POSTGRES_USER"`
	PostgresPassword string `env:"POSTGRES_PASSWORD"`
	PostgresDBName   string `env:"POSTGRES_DB_NAME"`

	ClickhouseHost     string `env:"CLICKHOUSE_HOST"`
	ClickhousePort     int    `env:"CLICKHOUSE_PORT"`
	ClickhouseUser     string `env:"CLICKHOUSE_USER"`
	ClickhousePassword string `env:"CLICKHOUSE_PASSWORD"`
	ClickhouseDBName   string `env:"CLICKHOUSE_DB_NAME"`

	MinIOHost          string `env:"MINIO_HOST"`
	MinIOPort          int    `env:"MINIO_PORT"`
	MinIOUser          string `env:"MINIO_USER"`
	MinIOPassword      string `env:"MINIO_PASSWORD"`
	MinIOMetricsBucket string `env:"MINIO_METRICS_BUCKET"`
}
