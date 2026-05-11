package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	// App
	AppEnv    string `mapstructure:"app_env"`
	AppPort   string `mapstructure:"app_port"`
	GRPCPort  string `mapstructure:"grpc_port"`
	WSPort    string `mapstructure:"ws_port"`

	// JWT
	JWTSecret     string `mapstructure:"jwt_secret"`
	JWTExpireHours int   `mapstructure:"jwt_expire_hours"`

	// Redis
	RedisAddr     string `mapstructure:"redis_addr"`
	RedisPassword string `mapstructure:"redis_password"`
	RedisDB       int    `mapstructure:"redis_db"`

	// MySQL
	MySQLDSN string `mapstructure:"mysql_dsn"`

	// OTel
	OtelEndpoint string `mapstructure:"otel_endpoint"`
	OtelInsecure bool   `mapstructure:"otel_insecure"`

	// Log
	LogLevel string `mapstructure:"log_level"`

	// DB Log
	DBLogLevel string `mapstructure:"db_log_level"`

	// WebSocket
	WSAllowedOrigin string `mapstructure:"ws_allowed_origin"`
	WSReadLimit     int    `mapstructure:"ws_read_limit"`

	// Tunnel
	CloudflaredPath string `mapstructure:"cloudflared_path"`
}

func Load() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")

	_ = viper.ReadInConfig()

	setDefaults()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("app_port", "8080")
	viper.SetDefault("grpc_port", "9090")
	viper.SetDefault("ws_port", "8081")
	viper.SetDefault("app_env", "development")
	viper.SetDefault("jwt_expire_hours", 24)
	viper.SetDefault("redis_addr", "localhost:6379")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("ws_read_limit", 512)
	viper.SetDefault("mysql_dsn", "")
	viper.SetDefault("otel_endpoint", "")
	viper.SetDefault("otel_insecure", true)
	viper.SetDefault("db_log_level", "warn")
	viper.SetDefault("cloudflared_path", "")
}

func validate(cfg *Config) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required and must not be empty")
	}
	if len(cfg.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return nil
}
