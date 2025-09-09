package config

// ldapAuthConfig holds the configuration for the ldap.
type ldapAuthConfig struct {
	AuthLdapServer           string `mapstructure:"AUTH_LDAP_SERVER"`
	AuthLdapPort             string `mapstructure:"AUTH_LDAP_PORT"`
	AuthLdapSearch           string `mapstructure:"AUTH_LDAP_SEARCH"`
	AuthLdapBindUser         string `mapstructure:"AUTH_LDAP_BIND_USER"`
	AuthLdapBindPassword     string `mapstructure:"AUTH_LDAP_BIND_PASSWORD"`
	AuthLdapUIDField         string `mapstructure:"AUTH_LDAP_UID_FIELD"`
	AuthLdapUseTLS           bool   `mapstructure:"AUTH_LDAP_USE_TLS"`
	AuthLdapAllowSelfSigned  bool   `mapstructure:"AUTH_LDAP_ALLOW_SELF_SIGNED"`
	AuthLdapTlsCacertFile    string `mapstructure:"AUTH_LDAP_TLS_CACERTFILE"`
	AuthLdapSearchAttributes string `mapstructure:"AUTH_LDAP_SEARCH_ATTRIBUTES"`
}
