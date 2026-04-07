CREATE TABLE IF NOT EXISTS daily_metrics
(
    metric_date DATE,
    metric_name TEXT,
    metric_value DOUBLE PRECISION,

    PRIMARY KEY (metric_date, metric_name)
);

CREATE TABLE IF NOT EXISTS top_daily_films
(
    metric_date DATE,
    movie_id TEXT,
    views BIGINT,

    PRIMARY KEY (metric_date, movie_id)
);
