package main

import (
	"os"
	"fmt"
	"log"
	"io"

	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/popstas/go-toggl"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/util"
	"github.com/popstas/planfix-go/planfix"
	"flag"
	"runtime"
	"io/ioutil"
)

var revision string

func main() {
	fmt.Printf("planfix-toggl %s\n", revision)

	var err error
	cfg := config.GetConfig()

	// logging
	dlog := log.New(os.Stderr, "[planfix-toggl] ", log.LstdFlags)
	if cfg.Debug {
		dlog.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	} else {
		toggl.DisableLog()
	}
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			dlog.Fatalf("[ERROR] No send interval, sending disabled", cfg.LogFile)
		}
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		dlog.SetOutput(mw)
	}

	if (cfg.SmtpSecure) {
		err := "[ERROR] Secure SMTP not implemented"
		dlog.Fatal(err)
		os.Exit(1)
	}

	if runtime.GOOS == "windows" {
		// Allow user to hide the console window
		flag.BoolVar(&cfg.NoConsole, "no-console", false, "Hide console window")
	}
	flag.Parse()

	if cfg.NoConsole {
		util.HideConsole()
	}

	planfixApi := planfix.New(
		cfg.PlanfixApiUrl,
		cfg.PlanfixApiKey,
		cfg.PlanfixAccount,
		cfg.PlanfixUserName,
		cfg.PlanfixUserPassword,
	)
	if !cfg.Debug {
		planfixApi.Logger.SetFlags(0)
		planfixApi.Logger.SetOutput(ioutil.Discard)
	}
	planfixApi.UserAgent = "planfix-toggl"

	// get user id
	if cfg.PlanfixUserId == 0 {
		var user planfix.XmlResponseUserGet
		user, err = planfixApi.UserGet(0)
		if err != nil {
			dlog.Printf("[ERROR] ", err.Error())
			os.Exit(1)
		}
		cfg.PlanfixUserId = user.User.Id
	}

	// create toggl client
	sess := toggl.OpenSession(cfg.TogglApiToken)
	togglClient := client.TogglClient{
		Session:    sess,
		Config:     cfg,
		PlanfixApi: planfixApi,
		Logger:      dlog,
	}

	// start tag cleaner
	go togglClient.RunTagCleaner()

	// start sender
	if cfg.SendInterval > 0 {
		go togglClient.RunSender()
	} else {
		dlog.Println("[INFO] No send interval, sending disabled")
	}

	// start server
	server := rest.Server{
		Version:     revision,
		TogglClient: togglClient,
		Config:      cfg,
		Logger:      dlog,
	}
	server.Run()
}
