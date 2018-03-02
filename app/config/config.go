package config

import (
	"github.com/jinzhu/configor"
)

// Config - структура с конфигом приложения
type Config struct {
	SMTPHost                   string `env:"SMTP_HOST" yaml:"smtpHost"`
	SMTPPort                   int    `env:"SMTP_PORT" yaml:"smtpPort"`
	SMTPSecure                 bool   `env:"SMTP_SECURE" yaml:"smtpSecure"`
	PlanfixAccount             string `env:"PLANFIX_ACCOUNT" yaml:"planfixAccount"`
	SendInterval               int    `env:"SEND_INTERVAL" yaml:"sendInterval"`
	LogFile                    string `env:"LOG_FILE" yaml:"logFile"`
	NoConsole                  bool   `env:"NO_CONSOLE" yaml:"noConsole"`
	TogglAPIToken              string `env:"TOGGL_API_TOKEN" yaml:"togglApiToken"`
	TogglWorkspaceID           int    `env:"TOGGL_WORKSPACE_ID" yaml:"togglWorkspaceId"`
	TogglSentTag               string `env:"TOGGL_SENT_TAG" yaml:"togglSentTag"`
	SMTPLogin                  string `env:"SMTP_LOGIN" yaml:"smtpLogin"`
	SMTPPassword               string `env:"SMTP_PASSWORD" yaml:"smtpPassword"`
	SMTPEmailFrom              string `env:"SMTP_EMAIL_FROM" yaml:"smtpEmailFrom"`
	PlanfixAuthorName          string `env:"PLANFIX_AUTHOR_NAME" yaml:"planfixAuthorName"`
	Debug                      bool   `env:"DEBUG" yaml:"debug"`
	PlanfixAPIKey              string `env:"PLANFIX_API_KEY" yaml:"planfixApiKey"`
	PlanfixAPIUrl              string `env:"PLANFIX_API_URL" yaml:"planfixApiUrl"`
	PlanfixUserName            string `env:"PLANFIX_USER_NAME" yaml:"planfixUserName"`
	PlanfixUserPassword        string `env:"PLANFIX_USER_PASSWORD" yaml:"planfixUserPassword"`
	PlanfixUserID              int    `env:"PLANFIX_USER_ID" yaml:"planfixUserId"` // will get in runtime
	PlanfixAnaliticName        string `env:"PLANFIX_ANALITIC_NAME" yaml:"planfixAnaliticName"`
	PlanfixAnaliticTypeName    string `env:"PLANFIX_ANALITIC_TYPE_NAME" yaml:"planfixAnaliticTypeName"`
	PlanfixAnaliticTypeValue   string `env:"PLANFIX_ANALITIC_TYPE_VALUE" yaml:"planfixAnaliticTypeValue"`
	PlanfixAnaliticCountName   string `env:"PLANFIX_ANALITIC_COUNT_NAME" yaml:"planfixAnaliticCountName"`
	PlanfixAnaliticCommentName string `env:"PLANFIX_ANALITIC_COMMENT_NAME" yaml:"planfixAnaliticCommentName"`
	PlanfixAnaliticDateName    string `env:"PLANFIX_ANALITIC_DATE_NAME" yaml:"planfixAnaliticDateName"`
	PlanfixAnaliticUsersName   string `env:"PLANFIX_ANALITIC_USERS_NAME" yaml:"planfixAnaliticUsersName"`
}

// GetConfig читает конфиг из файлов и возвращает структуру
func GetConfig() (cfg Config) {
	configor.Load(&cfg, "config.yml", "config.default.yml")
	return cfg
}
