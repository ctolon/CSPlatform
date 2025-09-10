package config

// pamConfig holds the configuration for the Config.
type pamConfig struct {
	PAMAPIUrl string `mapstructure:"PAM_API_URL"`
	PAMAuthAPIKey string `mapstructure:"PAM_AUTH_API_KEY"`
}
