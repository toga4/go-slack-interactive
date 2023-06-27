package configs

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type EnvConfig struct {
	Port int `env:"PORT,default=8080"`

	SlackOauthToken    string `env:"SLACK_OAUTH_TOKEN,required"`
	SlackSigningSecret string `env:"SLACK_SIGNING_SECRET,required"`
	SlackChannelId     string `env:"SLACK_CHANNEL_ID,required"`
}

func New(ctx context.Context) (*EnvConfig, error) {
	configs := &EnvConfig{}
	if err := envconfig.Process(ctx, configs); err != nil {
		return nil, fmt.Errorf("envconfig.Process: %w", err)
	}
	return configs, nil
}
