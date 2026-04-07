package config

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	ApiPort int

	PostgresPort     int
	PostgresHost     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	AuthPrivateKey    *rsa.PrivateKey
	AuthPublicKey     *rsa.PublicKey
	JWTAccsessExpires time.Duration
	JWTRefreshExpires time.Duration
}

func Load() (Config, error) {
	config := Config{}
	validation_errors := []error(nil)

	config.ApiPort = getEnvInt("API_PORT", &validation_errors)

	config.PostgresPort = getEnvInt("POSTGRES_PORT", &validation_errors)
	config.PostgresHost = getEnvStr("POSTGRES_HOST", &validation_errors)
	config.PostgresUser = getEnvStr("POSTGRES_USER", &validation_errors)
	config.PostgresPassword = getEnvStr("POSTGRES_PASSWORD", &validation_errors)
	config.PostgresDB = getEnvStr("POSTGRES_DB", &validation_errors)

	authPrivateKeyPath := getEnvStr("AUTH_PRIVATE_KEY_PATH", &validation_errors)
	config.AuthPrivateKey = loadPrivateKey(authPrivateKeyPath, &validation_errors)
	authPublicKeyPath := getEnvStr("AUTH_PUBLIC_KEY_PATH", &validation_errors)
	config.AuthPublicKey = loadPublicKey(authPublicKeyPath, &validation_errors)

	config.JWTAccsessExpires = time.Duration(
		getEnvInt("JWT_EXPIRES_MINUTES", &validation_errors),
	) * time.Minute
	config.JWTRefreshExpires = time.Duration(
		getEnvInt("JWT_REFRESH_EXPIRES_DAYS", &validation_errors),
	) * 24 * time.Hour

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

func loadPublicKey(path string, validation_errors *[]error) *rsa.PublicKey {
	data, err := os.ReadFile(path)
	if err != nil {
		*validation_errors = append(*validation_errors, err)
		return nil
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		*validation_errors = append(*validation_errors, err)
		return nil
	}
	return key
}

func loadPrivateKey(path string, validation_errors *[]error) *rsa.PrivateKey {
	data, err := os.ReadFile(path)
	if err != nil {
		*validation_errors = append(*validation_errors, err)
		return nil
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		*validation_errors = append(*validation_errors, err)
		return nil
	}
	return key
}
