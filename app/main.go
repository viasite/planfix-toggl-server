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

	os.Exit(0)
	sess := toggl.OpenSession("")
	TogglClient := client.TogglClient{
		Session: sess,
	}
	/*sess := toggl.OpenSession("6bf19c7924f843c7ef870ed27f368ef8")
	account, err := sess.GetAccount()
	if err != nil {
		println("error:", err)
		return
	}
	data, err := json.MarshalIndent(&account, "", "    ")
	println("account:", string(data))

	report, err := sess.GetSummaryReport(308899, "2018-02-12", "2018-02-13")
	if err != nil {
		println("error:", err)
		return
	}
	reportJson, err := json.MarshalIndent(&report, "", "    ")
	println("report:", string(reportJson))*/

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
