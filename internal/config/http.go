package config

import (
	"strings"
	"time"
)

type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func loadHTTP() (HTTPConfig, error) {
	addr, err := requireEnv("HTTP_ADDR")
	if err != nil {
		return HTTPConfig{}, err
	}
	readTimeout, err := requireDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return HTTPConfig{}, err
	}
	readHeaderTimeout, err := requireDuration("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return HTTPConfig{}, err
	}
	writeTimeout, err := requireDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return HTTPConfig{}, err
	}
	idleTimeout, err := requireDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return HTTPConfig{}, err
	}

	return HTTPConfig{
		Addr:              strings.TrimSpace(addr),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}, nil
}
