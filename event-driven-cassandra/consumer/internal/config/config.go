package config

type Config struct {
	KafkaServers       []string `env:"KAFKA_SERVERS"`
	KafkaConsumerGroup string   `env:"KAFKA_CONSUMER_GROUP"`
	SchemaRegistryURL  string   `env:"SCHEMA_REGISTRY_URL"`
	WarehouseTopic     string   `env:"WAREHOUSE_TOPIC"`
	DLQTopic           string   `env:"DLQ_TOPIC"`

	CassandraContactPoints []string `env:"CASSANDRA_CONTACT_POINTS"`
	CassandraKeyspace      string   `env:"CASSANDRA_KEYSPACE"`
	CassandraLocalDC       string   `env:"CASSANDRA_LOCAL_DC"`
}
