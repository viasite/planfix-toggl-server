package main

import (
	"os"
	"fmt"
	"log"

	"github.com/jessevdk/go-flags"
	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/jason0x43/go-toggl"
)

var revision string

var opts struct {
	SentTag         string      `short:"t" long:"sent-tag" env:"SENT_TAG" description:"" default:"sent"`
	SmtpHost         string
	SmtpPort         int
	SmtpSecure      bool
	PlanfixAccount  string
	PlanfixAnaliticName  string

	ApiToken          string
	WorkspaceId       int
	SmtpLogin        string
	SmtpPassword      string
	EmailFrom       string
	PlanfixAuthorName  string
	Dbg            bool          `long:"dbg" description:"debug mode"`
}

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	fmt.Printf("planfix-toggl %s\n", revision)


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
	if opts.Dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	//params := messager.Params{MaxDuration: opts.MaxExpire, MaxPinAttempts: opts.MaxPinAttempts}
	server := rest.Server{
		Version:        revision,
		TogglClient:    TogglClient,
	}
	server.Run()
}
