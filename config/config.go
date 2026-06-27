package config

import "github.com/spf13/viper"

type Config struct {
	ServerConfig ServerConfig `mapstructure:"server"`
	AuthConfig   AuthConfig   `mapstructure:"auth"`
	CORSConfig   CORSConfig   `mapstructure:"cors"`
	LoggerConfig LoggerConfig `mapstructure:"logger"`
	MongoConfig  MongoConfig  `mapstructure:"mongo"`
	RedisConfig  RedisConfig  `mapstructure:"redis"`
}

type AuthConfig struct {
	InternalToken string `mapstructure:"internal_token"`
}

type ServerConfig struct {
	Port                 int  `mapstructure:"port"`
	ReadTimeout          int  `mapstructure:"read_timeout"`           // seconds
	WriteTimeout         int  `mapstructure:"write_timeout"`          // seconds
	IdleTimeout          int  `mapstructure:"idle_timeout"`           // seconds
	EnableRequestLogging bool `mapstructure:"enable_request_logging"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type MongoConfig struct {
	URI    string `mapstructure:"uri"`
	DBName string `mapstructure:"db_name"`
}

type RedisConfig struct {
	Addr          string `mapstructure:"addr"`
	Password      string `mapstructure:"password"`
	DB            int    `mapstructure:"db"`
	MaxMemory     string `mapstructure:"max_memory"`    // e.g. "256mb"; empty = leave to operator
	CacheTTLHours int    `mapstructure:"cache_ttl_hours"`
}

type LoggerConfig struct {
	EnableConsoleLogger bool             `mapstructure:"enable_console_logger"`
	EnableFileLogger    bool             `mapstructure:"enable_file_logger"`
	FileLoggerConfig    FileLoggerConfig `mapstructure:"file"`
}

type FileLoggerConfig struct {
	FileName string `mapstructure:"file_name"`
	Path     string `mapstructure:"path"`
}

// Load reads the default "default.toml" config from standard search paths.
// Searches: ../conf/, ../../conf/, ./, ./conf/
func Load() (Config, error) {
	return LoadFrom("default")
}

// LoadFrom reads a config file by name from standard search paths.
// Use this when you need a specific config file (e.g. "production", "test").
func LoadFrom(fileName string) (Config, error) {
	viper.SetConfigName(fileName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("../conf/")
	viper.AddConfigPath("../../conf/")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./conf/")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
