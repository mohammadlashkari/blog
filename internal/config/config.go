package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Env               string  `env:"ENV" envDefault:"develop"` // develop, production
	Port              string  `env:"PORT,required"`
	DBPath            string  `env:"DB_PATH,required"`
	SiteURL           string  `env:"SITE_URL,required"`
	LimiterRPS        float64 `env:"LIMITER_RPS,required"`
	LimiterBurst      int     `env:"LIMITER_BURST,required"`
	LimiterEnable     bool    `env:"LIMITER_ENABLE,required"`
	LocalContentPath  string  `env:"LOCAL_CONTENT_PATH,required"`
	RemoteContentPath string  `env:"REMOTE_CONTENT_PATH,required"`
	MainBranchName    string  `env:"MAIN_BRANCH_NAME,required"`
	WebhookSecret     string  `env:"WEBHOOK_SECRET,required"`
	CookieSecret      string  `env:"COOKIE_SECRET,required"`
	AdminToken        string  `env:"ADMIN_TOKEN,required"`
	AdminUser         string  `env:"ADMIN_USER,required"`
	TokenVersion      int     `env:"TOKEN_VERSION,required"`
	ContentFilename   string  `env:"CONTENT_FILENAME,required"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Resolve secrets from files: for any FOO_FILE var, read the file and set FOO.
	if err := resolveFileSecrets(); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func resolveFileSecrets() error {
	for _, e := range os.Environ() {
		key, path, ok := strings.Cut(e, "=")
		if !ok || !strings.HasSuffix(key, "_FILE") || path == "" {
			continue
		}
		base := strings.TrimSuffix(key, "_FILE")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read secret file for %s: %w", base, err)
		}
		if err := os.Setenv(base, strings.TrimSpace(string(data))); err != nil {
			return fmt.Errorf("set %s from file: %w", base, err)
		}
	}
	return nil
}
