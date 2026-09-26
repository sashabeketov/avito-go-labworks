package config

type LogConfig struct {
	LogLevel string
}

// потом я добавлю допустимые уровни логирования, когда этап дойдет до логирования сервиса
func loadLog() (LogConfig, error) {
	val, err := requireEnv("LOG_LEVEL")

	if err != nil {
		return LogConfig{}, err
	}

	return LogConfig{
		LogLevel: val,
	}, nil
}
