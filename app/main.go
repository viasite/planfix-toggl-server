package main

import (
	"os"
	"fmt"
	"log"

	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/jason0x43/go-toggl"
	"github.com/jinzhu/configor"
)

var revision string

var Config struct {
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

func main() {
	configor.Load(&Config, "config.yml", "config.default.yml")
	fmt.Printf("planfix-toggl %s\n", revision)

	if (Config.SmtpSecure){
		err := "[ERR] Secure SMTP not implemented"
		log.Fatal(err)
		os.Exit(1)
	}

	sess := toggl.OpenSession(Config.ApiToken)
	TogglClient := client.TogglClient{
		Session: sess,
		Config: Config,
	}

	log.SetFlags(log.Ldate | log.Ltime)
	if Config.Debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	//params := messager.Params{MaxDuration: opts.MaxExpire, MaxPinAttempts: opts.MaxPinAttempts}
	server := rest.Server{
		Version:        revision,
		TogglClient:    TogglClient,
	}
	server.Run()
}
