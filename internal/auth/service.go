package auth

import (
	"blog/internal/config"
	"context"
)

type AuthService struct {
	cfg *config.Config
}

func New(ctx context.Context, cfg *config.Config) *AuthService {
	return &AuthService{
		cfg: cfg,
	}
}
