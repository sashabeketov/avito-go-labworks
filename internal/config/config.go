package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Database DBConfig
	Log      LogConfig
	App      AppConfig
}

func LoadConfig() (Config, error) {
	http, err := loadHTTP()

	if err != nil {
		return Config{}, err
	}

	database, err := loadDB()

	if err != nil {
		return Config{}, err
	}

	log, err := loadLog()

	if err != nil {
		return Config{}, err
	}

	app, err := loadApp()

	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP:     http,
		Database: database,
		Log:      log,
		App:      app,
	}, nil
}

func requireInt(key string) (int, error) {
	val, err := requireEnv(key)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, errors.New(key +
			" : " +
			"must be an integer")
	}

	if n < 0 {
		return 0, errors.New(key +
			" : " +
			" must not be negative")
	}

	return n, nil
}

func requireDuration(key string) (time.Duration, error) {
	val, err := requireEnv(key)

	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(val)

	if err != nil {
		return 0, errors.New(key +
			" : " +
			"invalid duration format (example: 10s)")
	}

	if duration <= 0 {
		return 0, errors.New(key +
			" : " +
			"duration must be greater than zero (example: 10s)")
	}

	return duration, nil
}

func requireEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	val = strings.TrimSpace(val)

	if !ok || val == "" {
		return "", errors.New(key + " is required")
	}

	return val, nil
}
