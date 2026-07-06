package config

import (
	"encoding/hex"
	"errors"
	"os"

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
	CookieSecret      []byte  `env:"COOKIE_SECRET,required"`
	AdminToken        string  `env:"ADMIN_TOKEN,required"`
	AdminUser         string  `env:"ADMIN_USER,required"`
	TokenVersion      int     `env:"TOKEN_VERSION,required"`
	ContentFilename   string  `env:"CONTENT_FILENAME,required"`
}

func Dev() (*Config, error) {
	s, _ := hex.DecodeString("13d6b4dff8f84a10851021ec8608f814570d562c92fe6b5ec4c9f595bcb3234b")
	return &Config{
		Env:               "develop",
		Port:              "2026",
		DBPath:            "blog.db",
		SiteURL:           "myblog",
		LocalContentPath:  "/home/mohammad/blog-content",
		RemoteContentPath: "https://github.com/mohammadlashkari/blog-content.git",
		LimiterEnable:     false,
		WebhookSecret:     "secret",
		CookieSecret:      s,
		MainBranchName:    "master",
		ContentFilename:   "index.md",
		AdminToken:        "secret",
		AdminUser:         "admin",
	}, nil
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
