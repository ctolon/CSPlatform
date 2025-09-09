package security

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"
	"v0/internal/config"

	nldap "github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/shaj13/go-guardian/v2/auth/strategies/ldap"
	"github.com/shaj13/libcache"
	_ "github.com/shaj13/libcache/fifo"

	"v0/internal/utils"
)

var AuthStrategy auth.Strategy
var cacheObj libcache.Cache
var onceLdap sync.Once

// Initialize LDAP as singleton.
func InitLDAP(config *config.AppConfig, log zerolog.Logger) {
	onceLdap.Do(func() {
		uidField := config.AuthLdapUIDField
		filter := fmt.Sprintf("(%s=%%s)", uidField)
		searchAtributes := utils.ParseSearchAttr(config.AuthLdapSearchAttributes)
		cfg := &ldap.Config{
			BaseDN:       config.AuthLdapSearch,
			BindDN:       config.AuthLdapBindUser,
			BindPassword: config.AuthLdapBindPassword,
			Port:         config.AuthLdapPort,
			Host:         config.AuthLdapServer,
			Filter:       filter,
		}
		if searchAtributes != nil {
			cfg.Attributes = searchAtributes
		}
		// skip TLS for LDAPS
		if config.AuthLdapPort == "636" {
			cfg.TLS = &tls.Config{
				InsecureSkipVerify: !config.AuthLdapUseTLS,
			}
		}
		cacheObj = libcache.FIFO.New(0)
		cacheObj.SetTTL(time.Minute * 5)
		AuthStrategy = ldap.NewCached(cfg, cacheObj)
	})
}

func ParseMemberOfAll(memberOf []string, log zerolog.Logger) []string {
	var out []string
	for _, dnStr := range memberOf {
		parsedDN, err := nldap.ParseDN(dnStr)
		if err != nil {
			log.Error().Err(err).Msgf("Cannot parse DN: %s", dnStr)
			continue
		}

		var cn string
		for _, rdn := range parsedDN.RDNs {
			for _, ava := range rdn.Attributes {
				log.Info().Msgf("attr=%s val=%s", ava.Type, ava.Value)
				if strings.ToLower(ava.Type) == "cn" && cn == "" {
					cn = ava.Value
				}
			}
		}
		if cn != "" {
			out = append(out, cn)
		}
	}
	return out
}

func ParseMemberOf(memberOf []string, targetOU string, log zerolog.Logger) []string {
	if targetOU != "" {
		targetOU = strings.ToLower(targetOU)
	}

	var out []string
	for _, dnStr := range memberOf {
		parsedDN, err := nldap.ParseDN(dnStr)
		if err != nil {
			log.Error().Err(err).Msgf("Cannot parse DN: %s", dnStr)
			continue
		}

		var cn string
		hasTargetOU := targetOU == "" // eğer targetOU boşsa tüm gruplar geçerli

		for _, rdn := range parsedDN.RDNs {
			for _, ava := range rdn.Attributes {
				attrType := strings.ToLower(ava.Type)
				val := strings.ToLower(ava.Value)

				if attrType == "cn" && cn == "" {
					cn = ava.Value
				}
				if targetOU != "" && attrType == "ou" && val == targetOU {
					hasTargetOU = true
				}
			}
		}

		if cn != "" && hasTargetOU {
			out = append(out, cn)
		}
	}
	return out
}
