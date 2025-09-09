package providers

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// ConfigType is a type for configuration file types
type ConfigType int

const (

	// JSON is a JSON configuration file type
	JSON ConfigType = iota

	// TOML is a TOML configuration file type
	TOML

	// YAML is a YAML configuration file type
	YAML

	// YML is a YML configuration file type
	YML

	// INI is an INI configuration file type
	INI

	// PROPERTIES is a PROPERTIES configuration file type
	PROPERTIES

	// PROPS is a PROPS configuration file type
	PROPS

	// ENV is an ENV configuration file type
	PROP

	// ENV is an ENV configuration file type
	ENV

	// DOTENV is a DOTENV configuration file type
	DOTENV
)

// String returns the string representation of the ConfigType
func (c ConfigType) String() string {
	return [...]string{"json", "toml", "yaml", "yml", "ini", "properties", "props", "prop", "env", "dotenv"}[c]
}

// ViperConfigProvider is a type for configuration providers
type ViperConfigProvider int

const (

	// LOCAL is a local configuration provider
	LOCAL ViperConfigProvider = iota

	// ETCD is an ETCD configuration provider
	ETCD

	// ETCD3 is an ETCD3 configuration provider
	ETCD3

	// CONSUL is a CONSUL configuration provider
	CONSUL

	// FIRESTORE is a FIRESTORE configuration provider
	FIRESTORE

	// NATS is a NATS configuration provider
	NATS
)

// String returns the string representation of the Provider
func (p ViperConfigProvider) String() string {
	return [...]string{"local", "etcd", "etcd3", "consul", "firestore", "nats"}[p]
}

// SecureRemoteConfigProvider is a struct for loading secure remote configuration
type SecureRemoteConfigProvider struct {
	Provider   ViperConfigProvider
	Endpoint   string
	ConfigType ConfigType
	ConfigPath string
	SecretKey  string
}

// RemoteConfigProvider is a struct for loading remote configuration
type RemoteConfigProvider struct {
	Provider   ViperConfigProvider
	Endpoint   string
	ConfigType ConfigType
	ConfigPath string
}

// LocalConfigProvider is a struct for loading local configuration
type LocalConfigProvider struct {
	Provider   ViperConfigProvider
	ConfigType ConfigType
	ConfigPath string
}

// LoadConfig loads configuration from the specified provider
func (r *SecureRemoteConfigProvider) LoadConfig() error {
	switch r.Provider {
	case ETCD:
		err := viper.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case ETCD3:
		err := viper.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case CONSUL:
		err := viper.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case FIRESTORE:
		err := viper.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case NATS:
		err := viper.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	default:
		panic("Invalid remote provider! remote provider must be one of etcd, etcd3, consul, firestore, nats")
	}
}

// LoadConfig loads configuration from the specified provider
func (r *SecureRemoteConfigProvider) LoadConfigWithNewViper(v *viper.Viper) (*viper.Viper, error) {
	switch r.Provider {
	case ETCD:
		err := v.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case ETCD3:
		err := v.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case CONSUL:
		err := v.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case FIRESTORE:
		err := v.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case NATS:
		err := v.AddSecureRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String(), r.SecretKey)
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	default:
		panic("Invalid remote provider! remote provider must be one of etcd, etcd3, consul, firestore, nats")
	}
}

// LoadConfig loads configuration from the specified provider
func (r *RemoteConfigProvider) LoadConfig() error {
	switch r.Provider {
	case ETCD:
		err := viper.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case ETCD3:
		err := viper.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case CONSUL:
		err := viper.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case FIRESTORE:
		err := viper.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	case NATS:
		err := viper.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		viper.SetConfigType(r.ConfigType.String())
		viper.ReadRemoteConfig()
		return err
	default:
		panic("Invalid remote provider! remote provider must be one of etcd, etcd3, consul, firestore, nats")
	}
}

// LoadConfigWithNewViper loads configuration from the specified provider with a new viper instance
func (r *RemoteConfigProvider) LoadConfigWithNewViper(v *viper.Viper) (*viper.Viper, error) {
	switch r.Provider {
	case ETCD:
		err := v.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case ETCD3:
		err := v.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case CONSUL:
		err := v.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case FIRESTORE:
		err := v.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	case NATS:
		err := v.AddRemoteProvider(r.Provider.String(), r.Endpoint, r.ConfigType.String())
		if err != nil {
			return nil, err
		}
		v.SetConfigType(r.ConfigType.String())
		v.ReadRemoteConfig()
		return v, nil
	default:
		panic("Invalid remote provider! remote provider must be one of etcd, etcd3, consul, firestore, nats")
	}
}

// LoadConfig loads configuration from the specified provider
func (l *LocalConfigProvider) LoadConfig() error {

	// TODO check with/without extension
	//_, err := os.Stat(l.ConfigPath)
	//if os.IsNotExist(err) {
	//	return err
	//}

	if l.ConfigPath != "" {
		viper.SetConfigFile(l.ConfigPath)
	} else {
		if l.ConfigType == ENV || l.ConfigType == DOTENV {
			// viper.SetConfigFile(fmt.Sprintf(".%s", l.ConfigType.String()))
			viper.SetConfigFile(".env")
			// viper.AutomaticEnv()
		} else {
			viper.SetConfigFile(fmt.Sprintf("config.%s", l.ConfigType.String()))
		}
	}
	viper.SetConfigType(l.ConfigType.String())

	return nil
}

// LoadConfigWithNewViper loads configuration from the specified provider with a new viper instance
func (l *LocalConfigProvider) LoadConfigWithNewViper(v *viper.Viper) (*viper.Viper, error) {

	// TODO check with/without extension
	_, err := os.Stat(l.ConfigPath)
	if os.IsNotExist(err) {
		return nil, err
	}

	v.SetConfigFile(l.ConfigPath)
	v.SetConfigType(l.ConfigType.String())
	return v, nil
}

// ConfigProvider is an interface for loading configuration
type ConfigProvider interface {

	// LoadConfig loads configuration from the specified provider
	LoadConfig() error

	// LoadConfigWithNewViper loads configuration from the specified provider with a new viper instance
	LoadConfigWithNewViper(v *viper.Viper) (*viper.Viper, error)
}

// NewConfigProvider creates a new configuration provider
func NewConfigProvider(provider ViperConfigProvider, endpoint string, configType ConfigType, configPath string, secretKey string) ConfigProvider {
	switch provider {
	case LOCAL:
		return &LocalConfigProvider{
			Provider:   provider,
			ConfigType: configType,
			ConfigPath: configPath,
		}
	case ETCD, ETCD3, CONSUL, FIRESTORE, NATS:
		if secretKey == "" {
			return &RemoteConfigProvider{
				Provider:   provider,
				Endpoint:   endpoint,
				ConfigType: configType,
				ConfigPath: configPath,
			}

		} else {
			return &SecureRemoteConfigProvider{
				Provider:   provider,
				Endpoint:   endpoint,
				ConfigType: configType,
				ConfigPath: configPath,
				SecretKey:  secretKey,
			}
		}

	default:
		panic("Invalid provider! provider must be one of local, etcd, etcd3, consul, firestore, nats")
	}
}
