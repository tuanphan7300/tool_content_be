package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var InfaCfg InfaConfig

type InfaConfig struct {
	ApiKey       string `envconfig:"API_KEY" default:""`
	GeminiKey    string `envconfig:"GEMINI_KEY" default:""`
	JWTACCESSKEY string `envconfig:"JWTACCESSKEY" default:""`
	DB_HOST      string `envconfig:"DB_HOST" default:""`
	DB_PORT      string `envconfig:"DB_PORT" default:""`
	DB_USER      string `envconfig:"DB_USER" default:""`
	DB_PASSWORD  string `envconfig:"DB_PASSWORD" default:""`
}

func (cfg *InfaConfig) LoadConfig() {
	err := envconfig.Process("", cfg)
	if err != nil {
		log.WithError(err).Error("load env err")
	}
}
