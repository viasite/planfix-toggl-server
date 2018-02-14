package config

import (
	"github.com/jinzhu/configor"
)

type Config struct {
	SentTag             string `env:"SENT_TAG" yaml:"sentTag"`
	SmtpHost            string `env:"SMTP_HOST" yaml:"smtpHost"`
	SmtpPort            int    `env:"SMTP_PORT" yaml:"smtpPort"`
	SmtpSecure          bool   `env:"SMTP_SECURE" yaml:"smtpSecure"`
	PlanfixAccount      string `env:"PLANFIX_ACCOUNT" yaml:"planfixAccount"`
	PlanfixAnaliticName string `env:"PLANFIX_ANALITIC_NAME" yaml:"planfixAnaliticName"`
	ApiToken            string `env:"API_TOKEN" yaml:"apiToken"`
	WorkspaceId         int    `env:"WORKSPACE_ID" yaml:"workspaceId"`
	SmtpLogin           string `env:"SMTP_LOGIN" yaml:"smtpLogin"`
	SmtpPassword        string `env:"SMTP_PASSWORD" yaml:"smtpPassword"`
	EmailFrom           string `env:"EMAIL_FROM" yaml:"emailFrom"`
	PlanfixAuthorName   string `env:"PLANFIX_AUTHOR_NAME" yaml:"planfixAuthorName"`
	Debug               bool   `env:"DEBUG" yaml:"debug"`
}

func GetConfig() (cfg Config) {
	configor.Load(&cfg, "config.yml", "config.default.yml")
	return cfg
}
