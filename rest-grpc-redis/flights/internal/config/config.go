package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppPort int

	PostgresPort     int
	PostgresHost     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	ApiKey string

	RedisMasterName string
	RedisReplicas []string
}

func Load() (Config, error) {
	validation_errors := []error(nil)
	config := Config{
		AppPort: getEnvInt("APP_PORT", &validation_errors),

		PostgresPort:     getEnvInt("POSTGRES_PORT", &validation_errors),
		PostgresHost:     getEnvStr("POSTGRES_HOST", &validation_errors),
		PostgresUser:     getEnvStr("POSTGRES_USER", &validation_errors),
		PostgresPassword: getEnvStr("POSTGRES_PASSWORD", &validation_errors),
		PostgresDB:       getEnvStr("POSTGRES_DB", &validation_errors),

		ApiKey: getEnvStr("API_KEY", &validation_errors),

		RedisMasterName: getEnvStr("REDIS_MASTER_NAME", &validation_errors),
		RedisReplicas: *unmarshallFromEnv[[]string]("REDIS_REPLICAS", &validation_errors),
	}

	if len(validation_errors) > 0 {
		return config, errors.Join(validation_errors...)
	}

	return config, nil
}

func getEnvInt(key string, validation_errors *[]error) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		*validation_errors = append(*validation_errors, fmt.Errorf("EnvVar %s not found", key))
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		*validation_errors = append(*validation_errors, fmt.Errorf("EnvVar %s not an Integer", key))
		return 0
	}
	return n
}

func getEnvStr(key string, validation_errors *[]error) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		*validation_errors = append(*validation_errors, fmt.Errorf("EnvVar %s not found", key))
		return ""
	}
	return val
}

func unmarshallFromEnv[T any](key string, validation_errors *[]error) *T {
	var t T
	val, ok := os.LookupEnv(key)
	if !ok {
		*validation_errors = append(*validation_errors, fmt.Errorf("EnvVar %s not found", key))
		return nil
	}
	err := json.Unmarshal([]byte(val), &t)
	if err != nil {
		*validation_errors = append(*validation_errors, fmt.Errorf("Failed to validate %s", key))
		return nil
	}
	return &t
}
