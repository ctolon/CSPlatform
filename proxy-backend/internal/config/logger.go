package config

// loggerConfig holds the configuration for the logger.
type loggerConfig struct {
	LogLevel string `mapstructure:"APP_LOG_LEVEL"`
}
