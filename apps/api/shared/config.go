package shared

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config encapsulates all application environment configurations with defaults and type-safety.
type Config struct {
	AppEnv        string `env:"APP_ENV" envDefault:"development"`
	AppURL        string `env:"APP_URL" envDefault:"http://localhost:8080"`
	FrontendURL   string `env:"FRONTEND_URL" envDefault:"http://localhost:5173"`
	APIPort       string `env:"API_PORT" envDefault:"8080"`
	DBURL         string `env:"DB_URL,required"`
	RedisAddr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	JWTSecret     string `env:"JWT_SECRET,required"`
	SMTPHost      string `env:"SMTP_HOST" envDefault:"localhost"`
	SMTPPort      int    `env:"SMTP_PORT" envDefault:"1025"`
	SMTPFrom      string `env:"SMTP_FROM" envDefault:"crackerstudio@digital"`
	TempoEndpoint string `env:"TEMPO_ENDPOINT" envDefault:"127.0.0.1:4317"`
	LogDir        string `env:"LOG_DIR"`
}

// LoadConfig loads variables from .env file (if present) and parses them into Config struct.
func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
