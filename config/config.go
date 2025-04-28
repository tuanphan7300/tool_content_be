package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var InfaCfg InfaConfig

type InfaConfig struct {
	ApiKey    string `envconfig:"API_KEY" default:""`
	GeminiKey string `envconfig:"GEMINI_KEY" default:""`
}

func (cfg *InfaConfig) LoadConfig() {
	err := envconfig.Process("", cfg)
	if err != nil {
		log.WithError(err).Error("load env err")
	}
}
