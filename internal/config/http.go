package config

import (
	"strings"
)

type HTTPConfig struct {
	Addr string
}

func loadHTTP() (HTTPConfig, error) {
	val, err := requireEnv("HTTP_ADDR")

	if err != nil {
		return HTTPConfig{}, err
	}

	return HTTPConfig{
		Addr: strings.TrimSpace(val),
	}, nil
}
