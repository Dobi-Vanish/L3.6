package config

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"time"

	"github.com/wb-go/wbf/config/cleanenv-port"
)

type Config struct {
	PostgresDSN     string        `env:"POSTGRES_DSN" env-required:"true"`
	HTTPServerPort  string        `env:"HTTP_SERVER_PORT" env-default:"8081"`
	LogLevel        string        `env:"LOG_LEVEL" env-default:"info"`
	AppName         string        `env:"APP_NAME" env-default:"L3.6"`
	Env             string        `env:"ENV" env-default:"development"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

var cfg Config

func Load() (*Config, error) {
	configPath := fetchConfigPath()
	if configPath != "" {
		if err := cleanenvport.LoadPath(configPath, &cfg); err != nil {
			return nil, err
		}
	} else {
		if err := loadFromEnv(); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func fetchConfigPath() string {
	var path string
	if f := flag.Lookup("config"); f != nil {
		path = f.Value.String()
	} else {
		flag.StringVar(&path, "config", "", "path to config file")
		flag.Parse()
	}
	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}
	return path
}

func loadFromEnv() error {
	_ = godotenv.Load("configs/config.env")
	cfg.PostgresDSN = os.Getenv("POSTGRES_DSN")
	cfg.HTTPServerPort = os.Getenv("HTTP_SERVER_PORT")
	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	cfg.AppName = os.Getenv("APP_NAME")
	cfg.Env = os.Getenv("ENV")
	if cfg.PostgresDSN == "" {
		return fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.HTTPServerPort == "" {
		cfg.HTTPServerPort = "8081"
	}
	return nil
}
