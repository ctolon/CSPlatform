package config

// AuthConfig holds the configuration for the auth.
type authConfig struct {
	AuthBackend      string `mapstructure:"AUTH_BACKEND"`
	AuthRegularRoles string `mapstructure:"AUTH_REGULAR_ROLES"`
	AuthAdminRoles   string `mapstructure:"AUTH_ADMIN_ROLES"`
}
