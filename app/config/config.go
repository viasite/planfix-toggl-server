package config

import (
	"fmt"
	"github.com/jinzhu/configor"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"
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
	TogglUserID                int    `env:"TOGGL_USER_ID" yaml:"togglUserId"` // will get in runtime
	TogglSentTag               string `env:"TOGGL_SENT_TAG" yaml:"togglSentTag"`
	SMTPLogin                  string `env:"SMTP_LOGIN" yaml:"smtpLogin"`
	SMTPPassword               string `env:"SMTP_PASSWORD" yaml:"smtpPassword"`
	SMTPEmailFrom              string `env:"SMTP_EMAIL_FROM" yaml:"smtpEmailFrom"`
	PlanfixAuthorName          string `env:"PLANFIX_AUTHOR_NAME" yaml:"planfixAuthorName"`
	Debug                      bool   `env:"DEBUG" yaml:"debug"`
	DryRun                     bool   `env:"DRY_RUN" yaml:"dryRun"`
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

// - GetConfig читает конфиг из файлов и возвращает структуру
func GetConfig() (cfg Config) {
	configor.Load(&cfg, "config.yml", "config.default.yml")
	return cfg
}

// - SaveConfig пишет конфиг в файл
func (c *Config) SaveConfig() (cfg *Config, err error) {
	cfgString, err := yaml.Marshal(c)
	if err != nil {
		return cfg, err
	}
	if _, err := os.Stat("config.yml"); err == nil {
		os.Rename("config.yml", fmt.Sprintf("config-%s.yml", time.Now().Format("2006-02-01 15_04_05")))
		ioutil.WriteFile("config.yml", cfgString, 0644)
	}
	return c, nil
}

func (c *Config) Validate() (errors []string, isValid bool) {
	err := c.isGroupInvalid([]string{
		"TogglAPIToken",
		"TogglWorkspaceID",
	})
	if err != nil {
		errors = append(errors, "настройки Toggl неправильные")
	}

	err = c.isGroupInvalid([]string{
		"PlanfixAccount",
		"PlanfixApiKey",
		"PlanfixUserName",
		"PlanfixUserPassword",
	})
	if err != nil {
		err = c.isGroupInvalid([]string{
			"SMTPHost",
			"SMTPPort",
			"SMTPLogin",
			"SMTPPassword",
			"SMTPEmailFrom",
		})
		if err != nil {
			errors = append(errors, "настройки подключения к Планфиксу и настройки SMTP неправильные, нужно настроить что-то одно")
		}
	}

	err = c.isGroupInvalid([]string{
		"PlanfixAnaliticName",
		"PlanfixAnaliticTypeName",
		"PlanfixAnaliticTypeValue",
		"PlanfixAnaliticCountName",
		"PlanfixAnaliticCommentName",
		"PlanfixAnaliticDateName",
		"PlanfixAnaliticUsersName",
	})
	if err != nil {
		errors = append(errors, "настройки аналитики, отправляемой в Планфикс, неправильные")
	}

	if c.SMTPSecure {
		errors = append(errors, "Secure SMTP не поддерживается")
	}

	isValid = len(errors) == 0
	return errors, isValid
}

func isEmpty(s string) bool {
	return s == ""
}

func (c *Config) isGroupInvalid(fields []string) error {
	emptyFields := c.filterNotEmpty(fields)
	if len(emptyFields) == 0 {
		return nil
	}

	return fmt.Errorf("поля %s неправильные",
		strings.Join(emptyFields, ", "),
	)
}

func (c *Config) filterNotEmpty(fields []string) (emptyFields []string) {
	for _, fieldName := range fields {
		if isEmpty(c.getFieldByName(fieldName)) {
			emptyFields = append(emptyFields, fieldName)
		}
	}
	return emptyFields
}

func (c *Config) getFieldByName(field string) string {
	r := reflect.ValueOf(c)
	v := reflect.Indirect(r)
	f := v.FieldByName(field)
	return f.String()
}

func (c *Config) GetFields() (emptyFields []string) {
	s := reflect.ValueOf(c).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		emptyFields = append(emptyFields, typeOfT.Field(i).Name)
	}

	return emptyFields
}

/*func (c *Config) GetEmptyFields (emptyFields []string){
	r := reflect.ValueOf(c)

	fields := r.MapKeys()
	for _, field := range(fields){
		fieldValue := field.String()
	}
}*/
