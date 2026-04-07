package config

type Config struct {
	AppPort           int      `env:"APP_PORT"`
	KafkaServers      []string `env:"KAFKA_SERVERS"`
	SchemaRegistryURL string   `env:"SCHEMA_REGISTRY_URL"`
	Topic             string   `env:"MOVIE_TOPIC"`
}
