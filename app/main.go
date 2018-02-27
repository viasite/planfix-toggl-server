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
	"github.com/popstas/planfix-go/planfix"
)

var revision string

func main() {
	fmt.Printf("planfix-toggl %s\n", revision)

	var err error
	cfg := config.GetConfig()
	if (cfg.SmtpSecure) {
		err := "[ERR] Secure SMTP not implemented"
		log.Fatal(err)
		os.Exit(1)
	}

	planfixApi := planfix.New(
		cfg.PlanfixApiUrl,
		cfg.PlanfixApiKey,
		cfg.PlanfixAccount,
		cfg.PlanfixUserName,
		cfg.PlanfixUserPassword,
	)
	planfixApi.UserAgent = "planfix-toggl"

	// get user id
	if cfg.PlanfixUserId == 0 {
		var user planfix.XmlResponseUserGet
		user, err = planfixApi.UserGet(0)
		if err != nil {
			log.Printf("[ERROR] ", err.Error())
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
	}

	// logging
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("[ERROR] No send interval, sending disabled", cfg.LogFile)
		}
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}
	toggl.DisableLog()
	if cfg.Debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	// start sender
	if cfg.SendInterval > 0 {
		go togglClient.RunSender()
	} else {
		log.Println("[INFO] No send interval, sending disabled")
	}

	// start server
	server := rest.Server{
		Version:     revision,
		TogglClient: togglClient,
		Config:      cfg,
	}
	server.Run()
}
