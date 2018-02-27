package config

import (
	"github.com/jinzhu/configor"
)

type Config struct {
	SmtpHost            string `env:"SMTP_HOST" yaml:"smtpHost"`
	SmtpPort            int    `env:"SMTP_PORT" yaml:"smtpPort"`
	SmtpSecure          bool   `env:"SMTP_SECURE" yaml:"smtpSecure"`
	PlanfixAccount      string `env:"PLANFIX_ACCOUNT" yaml:"planfixAccount"`
	PlanfixAnaliticName string `env:"PLANFIX_ANALITIC_NAME" yaml:"planfixAnaliticName"`
	SendInterval        int    `env:"SEND_INTERVAL" yaml:"sendInterval"`
	LogFile             string `env:"LOG_FILE" yaml:"logFile"`
	TogglApiToken       string `env:"TOGGL_API_TOKEN" yaml:"togglApiToken"`
	TogglWorkspaceId    int    `env:"TOGGL_WORKSPACE_ID" yaml:"togglWorkspaceId"`
	TogglSentTag        string `env:"TOGGL_SENT_TAG" yaml:"togglSentTag"`
	SmtpLogin           string `env:"SMTP_LOGIN" yaml:"smtpLogin"`
	SmtpPassword        string `env:"SMTP_PASSWORD" yaml:"smtpPassword"`
	SmtpEmailFrom       string `env:"SMTP_EMAIL_FROM" yaml:"smtpEmailFrom"`
	PlanfixAuthorName   string `env:"PLANFIX_AUTHOR_NAME" yaml:"planfixAuthorName"`
	Debug               bool   `env:"DEBUG" yaml:"debug"`
	PlanfixApiKey       string `env:"PLANFIX_API_KEY" yaml:"planfixApiKey"`
	PlanfixApiUrl       string `env:"PLANFIX_API_URL" yaml:"planfixApiUrl"`
	PlanfixUserName     string `env:"PLANFIX_USER_NAME" yaml:"planfixUserName"`
	PlanfixUserPassword string `env:"PLANFIX_USER_PASSWORD" yaml:"planfixUserPassword"`
	PlanfixUserId       int    `env:"PLANFIX_USER_ID" yaml:"planfixUserId"`
}

func GetConfig() (cfg Config) {
	configor.Load(&cfg, "config.yml", "config.default.yml")
	return cfg
}
