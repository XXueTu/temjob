package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Server   ServerConfig   `yaml:"server"`
	Worker   WorkerConfig   `yaml:"worker"`
	Engine   EngineConfig   `yaml:"engine"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
}

type MySQLConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parse_time"`
	Loc       string `yaml:"loc"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

type WorkerConfig struct {
	Concurrency int           `yaml:"concurrency"`
	Timeout     time.Duration `yaml:"timeout"`
	RetryDelay  time.Duration `yaml:"retry_delay"`
}

type EngineConfig struct {
	MonitorInterval     time.Duration `yaml:"monitor_interval"`
	MaxWorkflowTimeout  time.Duration `yaml:"max_workflow_timeout"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	setDefaults(&config)

	return &config, nil
}

func setDefaults(config *Config) {
	if config.Database.MySQL.Charset == "" {
		config.Database.MySQL.Charset = "utf8mb4"
	}
	if config.Database.MySQL.Loc == "" {
		config.Database.MySQL.Loc = "Local"
	}
	if config.Redis.PoolSize == 0 {
		config.Redis.PoolSize = 10
	}
	if config.Server.Mode == "" {
		config.Server.Mode = "debug"
	}
	if config.Worker.Concurrency == 0 {
		config.Worker.Concurrency = 10
	}
	if config.Worker.Timeout == 0 {
		config.Worker.Timeout = 30 * time.Minute
	}
	if config.Worker.RetryDelay == 0 {
		config.Worker.RetryDelay = 5 * time.Second
	}
	if config.Engine.MonitorInterval == 0 {
		config.Engine.MonitorInterval = 10 * time.Second
	}
	if config.Engine.MaxWorkflowTimeout == 0 {
		config.Engine.MaxWorkflowTimeout = 24 * time.Hour
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}
}

func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset, c.ParseTime, c.Loc)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}