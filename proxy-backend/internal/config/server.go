package config

// serverConfig holds the configuration for the server.
type serverConfig struct {
	AppHost          string `mapstructure:"APP_HOST"`
	AppPort          int    `mapstructure:"APP_PORT"`
	AppWithTLS       bool   `mapstructure:"APP_WITH_TLS"`
	AppTLSCrt        string `mapstructure:"APP_TLS_CRT"`
	AppTLSKey        string `mapstructure:"APP_TLS_KEY"`
	AppSessionSecret string `mapstructure:"APP_SESSION_SECRET"`
	AppCORS          string `mapstructure:"APP_CORS"`
	AppSessionCookie string `mapstructure:"APP_SESSION_COOKIE"`
	AppAgentKey      string `mapstructure:"APP_AGENT_KEY"`
}
