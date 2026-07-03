package config

type Config struct {
	Env               string  `env:"ENV"`
	Port              string  `env:"PORT"`
	LimiterRPS        float64 `env:"LIMITER_RPS"`
	LimiterBurst      int     `env:"LIMITER_BURST"`
	LimiterEnable     bool    `env:"LIMITER_ENABLE"`
	LocalContentPath  string  `env:"LOCAL_CONTENT_PATH"`
	RemoteContentPath string  `env:"REMOTE_CONTENT_PATH"`
	MainBranchName    string  `env:"MAIN_BRANCH_NAME"`
	WebhookSecret     string  `env:"WEBHOOK_SECRET"`
	AdminToken        string  `env:"ADMIN_TOKEN"`
	AdminUser         string  `env:"ADMIN_USER"`
	ContentFilename   string  `env:"CONTENT_FILENAME"`
}

func New() (*Config, error) {
	return &Config{
		Env:               "develop",
		Port:              "2026",
		LocalContentPath:  "/home/mohammad/blog-content",
		RemoteContentPath: "https://github.com/mohammadlashkari/blog-content.git",
		LimiterEnable:     false,
		WebhookSecret:     "secret",
		MainBranchName:    "master",
		ContentFilename:   "index.md",
		AdminToken:        "secret",
		AdminUser:         "admin",
	}, nil
}
