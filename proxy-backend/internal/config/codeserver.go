package config

// codeServerConfig holds the configuration for the code-server.
type codeServerConfig struct {
	CodeServerBaseHost     string `mapstructure:"CODE_SERVER_BASE_HOST"`
	CodeServerBasePort     int    `mapstructure:"CODE_SERVER_BASE_PORT"`
	CodeServerWithUsername bool   `mapstructure:"CODE_SERVER_WITH_USERNAME"`
}
