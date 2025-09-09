package config

// jwtConfig holds the configuration for the JWT.
type jwtConfig struct {
	JWTAccessSecret  string `mapstructure:"JWT_ACCESS_SECRET"`
	JWTRefreshSecret string `mapstructure:"JWT_REFRESH_SECRET"`
	JWTIssuer        string `mapstructure:"JWT_ISSUER"`
	JWTAudience      string `mapstructure:"JWT_AUDIENCE"`
}
