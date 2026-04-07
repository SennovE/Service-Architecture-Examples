CREATE DATABASE IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.movie_events_queue
(
    event_id UUID,
    user_id String,
    movie_id String,
    event_type LowCardinality(String),
    timestamp DateTime64(3, 'UTC'),
    device_type LowCardinality(String),
    session_id String,
    progress_seconds Int32
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka1:9092,kafka2:9093',
    kafka_topic_list = 'movie-events',
    kafka_group_name = 'clickhouse-movie-events-consumer',
    kafka_format = 'AvroConfluent',
    kafka_num_consumers = 1,
    kafka_handle_error_mode = 'stream',
    kafka_commit_every_batch = 1,
    format_avro_schema_registry_url = 'http://schema-registry:8081';

CREATE TABLE IF NOT EXISTS analytics.movie_events
(
    event_id UUID,
    user_id String,
    movie_id String,
    event_type LowCardinality(String),
    timestamp DateTime64(3, 'UTC'),
    event_date Date DEFAULT toDate(timestamp),
    device_type LowCardinality(String),
    session_id String,
    progress_seconds UInt32
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, user_id, session_id, timestamp, event_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.movie_events_mv
TO analytics.movie_events
AS
SELECT
    event_id,
    user_id,
    movie_id,
    event_type,
    timestamp,
    toDate(timestamp) AS event_date,
    device_type,
    session_id,
    toUInt32(greatest(progress_seconds, 0)) AS progress_seconds
FROM analytics.movie_events_queue;

CREATE TABLE IF NOT EXISTS analytics.daily_metrics
(
    metric_date Date,
    metric_name LowCardinality(String),
    metric_value Float64,
    computed_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(computed_at)
PARTITION BY toYYYYMM(metric_date)
ORDER BY (metric_date, metric_name);

CREATE TABLE IF NOT EXISTS analytics.top_daily_films
(
    metric_date Date,
    movie_id String,
    views UInt64,
    computed_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(computed_at)
PARTITION BY toYYYYMM(metric_date)
ORDER BY (metric_date, movie_id);
