package config

import (
	"errors"
	"time"
)

type DBConfig struct {
	DatabaseURL     string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
	PingTimeout     time.Duration
}

func loadDB() (DBConfig, error) {
	url, err := requireEnv("DATABASE_URL")
	if err != nil {
		return DBConfig{}, err
	}

	maxConns, err := requireInt("DATABASE_MAX_CONNS")
	if err != nil {
		return DBConfig{}, err
	}

	minConns, err := requireInt("DATABASE_MIN_CONNS")
	if err != nil {
		return DBConfig{}, err
	}

	if maxConns == 0 {
		return DBConfig{}, errors.New("max connections must be grater than zero")
	}

	if minConns > maxConns {
		return DBConfig{}, errors.New("min connections must be lower than max connections now")
	}

	maxLifetime, err := requireDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return DBConfig{}, err
	}

	connectTimeout, err := requireDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return DBConfig{}, err
	}

	queryTimeout, err := requireDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return DBConfig{}, err
	}

	pingTimeout, err := requireDuration("DATABASE_PING_TIMEOUT")
	if err != nil {
		return DBConfig{}, err
	}

	return DBConfig{
		DatabaseURL:     url,
		MaxConns:        maxConns,
		MinConns:        minConns,
		MaxConnLifetime: maxLifetime,
		ConnectTimeout:  connectTimeout,
		QueryTimeout:    queryTimeout,
		PingTimeout:     pingTimeout,
	}, nil
}
