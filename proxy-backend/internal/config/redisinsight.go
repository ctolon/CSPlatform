package config

// redisInsightConfig holds the configuration for the RedisInsight.
type redisInsightConfig struct {
	RedisInsightEnabled bool   `mapstructure:"REDIS_INSIGHT_ENABLED"`
	RedisInsightProto   string `mapstructure:"REDIS_INSIGHT_PROTO"`
	RedisInsightURL     string `mapstructure:"REDIS_INSIGHT_URL"`
}
