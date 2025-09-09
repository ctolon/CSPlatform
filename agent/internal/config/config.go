package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		LogLevel string `mapstructure:"log_level"`
		WithTLS  bool   `mapstructure:"with_tls"`
		Pem      string `mapstructure:"pem"`
		Key      string `mapstructure:"key"`
	} `mapstructure:"server"`

	ContainerTemplate struct {
		ImageName     string         `mapstructure:"image_name"`
		ContainerName string         `mapstructure:"container_name"`
		Restart       string         `mapstructure:"restart"`
		Environment   map[string]any `mapstructure:"environment"`
		Sysctls       map[string]any `mapstructure:"sysctls"`
		Expose        []int          `mapstructure:"expose"`
		MemLimit      string         `mapstructure:"mem_limit"`
		Cpus          int            `mapstructure:"cpus"`
		ExtraHost     []string       `mapstructure:"extra_host"`
		Volumes       []string       `mapstructure:"volumes"`
		Networks      map[string]any `mapstructure:"networks"`
		Ports         []string       `mapstructure:"ports"`
	} `mapstructure:"container_template"`

	AgentMetadata struct {
		InstanceID    string         `mapstructure:"instance_id"`
		ServiceName   string         `mapstructure:"service_name"`
		MainHostProto string         `mapstructure:"main_host_proto" default:"http"`
		MainHost      string         `mapstructure:"main_host"`
		HostPortProto string         `mapstructure:"host_port_proto" default:"http"`
		HostPort      string         `mapstructure:"host_port"`
		Version       string         `mapstructure:"version"`
		Region        string         `mapstructure:"region"`
		Tags          map[string]any `mapstructure:"tags"`
		ServerURL     string         `mapstructure:"server_url"`
		AgentKey      string         `mapstructure:"x_agent_key"`
	} `mapstructure:"agent_metadata"`

	Secrets struct {
		JWTAccessKey  string `mapstructure:"jwt_access_key"`
		JWTRefreshKey string `mapstructure:"jwt_refresh_key"`
		JWTIssuer     string `mapstructure:"jwt_issuer"`
		JWTAudience   string `mapstructure:"jwt_audience"`
		SessionSecret string `mapstructure:"session_secret"`
	} `mapstructure:"secrets"`

	Redis struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &cfg, nil
}
