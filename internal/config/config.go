package config

type Config struct {
	Port              string  `env:"PORT"`
	DbPath            string  `env:"DB_PATH"`
	LimiterRPS        float64 `env:"LIMITER_RPS"`
	LimiterBurst      int     `env:"LIMITER_BURST"`
	LimiterEnable     bool    `env:"LIMITER_ENABLE"`
	LocalContentRepo  string  `env:"LOCAL_CONTENT_REPO"`
	RemoteContentRepo string  `env:"REMOTE_CONTENT_REPO"`
	WebhookSecret     string  `env:"WEBHOOK_SECRET"`
}

func Web() (*Config, error) {
	return &Config{
		Port:             "2026",
		DbPath:           "./blog.sqlite",
		LocalContentRepo: "~/Documents/obsidian/mohammad/blog/",
	}, nil
}
