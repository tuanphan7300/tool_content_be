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
	// Google OAuth2 configuration
	GoogleClientID     string `envconfig:"GOOGLE_CLIENT_ID" default:""`
	GoogleClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET" default:""`
	GoogleRedirectURL  string `envconfig:"GOOGLE_REDIRECT_URL" default:"http://localhost:8080/auth/google/callback"`
	EmailImapHost      string `envconfig:"EMAIL_IMAP_HOST" default:""`
	EmailImapUser      string `envconfig:"EMAIL_IMAP_USER" default:""`
	EmailImapPassword  string `envconfig:"EMAIL_IMAP_PASS" default:""`
}

func (cfg *InfaConfig) LoadConfig() {
	err := envconfig.Process("", cfg)
	if err != nil {
		log.WithError(err).Error("load env err")
	}
}
