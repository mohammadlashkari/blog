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
	ApiToken          string  `env:"API_TOKEN"`
}

func New() (*Config, error) {
	return &Config{
		Port:              "2026",
		DbPath:            "./blog.sqlite",
		LocalContentRepo:  "/var/lib/blog-content",
		RemoteContentRepo: "https://github.com/mohammadlashkari/blog-content.git",
		LimiterEnable:     false,
		WebhookSecret:     "secret",
	}, nil
}
