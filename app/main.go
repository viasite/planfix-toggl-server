package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"flag"
	"github.com/popstas/go-toggl"
	"github.com/popstas/planfix-go/planfix"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/util"
	"io/ioutil"
	"runtime"
)

var version string

func main() {
	fmt.Printf("planfix-toggl %s\n", version)

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

	if cfg.SMTPSecure {
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

	planfixAPI := planfix.New(
		cfg.PlanfixAPIUrl,
		cfg.PlanfixAPIKey,
		cfg.PlanfixAccount,
		cfg.PlanfixUserName,
		cfg.PlanfixUserPassword,
	)
	if !cfg.Debug {
		planfixAPI.Logger.SetFlags(0)
		planfixAPI.Logger.SetOutput(ioutil.Discard)
	}
	planfixAPI.UserAgent = "planfix-toggl"

	// get user id
	if cfg.PlanfixUserID == 0 {
		var user planfix.XMLResponseUserGet
		user, err = planfixAPI.UserGet(0)
		if err != nil {
			dlog.Printf("[ERROR] ", err.Error())
			os.Exit(1)
		}
		cfg.PlanfixUserID = user.User.ID
	}

	// create toggl client
	sess := toggl.OpenSession(cfg.TogglAPIToken)
	togglClient := client.TogglClient{
		Session:    sess,
		Config:     cfg,
		PlanfixAPI: planfixAPI,
		Logger:     dlog,
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
		Version:     version,
		TogglClient: togglClient,
		Config:      cfg,
		Logger:      dlog,
	}
	server.Run()
}
