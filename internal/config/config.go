package config

import "encoding/hex"

type Config struct {
	Env               string  `env:"ENV"`
	Port              string  `env:"PORT"`
	DBPath            string  `env:"DB_PATH"`
	SiteURL           string  `env:"SITE_URL"`
	LimiterRPS        float64 `env:"LIMITER_RPS"`
	LimiterBurst      int     `env:"LIMITER_BURST"`
	LimiterEnable     bool    `env:"LIMITER_ENABLE"`
	LocalContentPath  string  `env:"LOCAL_CONTENT_PATH"`
	RemoteContentPath string  `env:"REMOTE_CONTENT_PATH"`
	MainBranchName    string  `env:"MAIN_BRANCH_NAME"`
	WebhookSecret     string  `env:"WEBHOOK_SECRET"`
	CookieSecret      []byte  `env:"COOKIE_SECRET"`
	AdminToken        string  `env:"ADMIN_TOKEN"`
	AdminUser         string  `env:"ADMIN_USER"`
	TokenVersion      int     `env:"TOKEN_VERSION"`
	ContentFilename   string  `env:"CONTENT_FILENAME"`
}

func Load() (*Config, error) {
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
