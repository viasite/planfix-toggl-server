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

func getLogger(cfg config.Config) *log.Logger {
	// logging
	logger := log.New(os.Stderr, "[planfix-toggl] ", log.LstdFlags)
	if cfg.Debug {
		logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	} else {
		toggl.DisableLog()
	}
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			logger.Fatalf("[ERROR] Failed to open log file: %s", cfg.LogFile)
		}
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		logger.SetOutput(mw)
	}
	return logger
}

func parseFlags(cfg *config.Config) {
	dryRun := flag.Bool("dry-run", false, "Don't actually change data")
	if runtime.GOOS == "windows" {
		// Allow user to hide the console window
		flag.BoolVar(&cfg.NoConsole, "no-console", false, "Hide console window")
	}
	flag.Parse()

	if *dryRun {
		cfg.DryRun = true
	}
}

func connectServices(cfg *config.Config, logger *log.Logger, togglClient client.TogglClient) (err error){
	// toggl
	logger.Println("[INFO] подключение к Toggl...")
	cfg.TogglUserID, err = togglClient.GetTogglUserID()
	if err != nil {
		return err
	}

	// planfix
	if cfg.PlanfixUserName != "" && cfg.PlanfixUserPassword != "" {
		logger.Println("[INFO] подключение к Планфиксу...")
		cfg.PlanfixUserID = togglClient.GetPlanfixUserID()
		logger.Println("[INFO] получение данных аналитики Планфикса...")
		_, err := togglClient.GetAnaliticData(
			cfg.PlanfixAnaliticName,
			cfg.PlanfixAnaliticTypeName,
			cfg.PlanfixAnaliticTypeValue,
			cfg.PlanfixAnaliticCountName,
			cfg.PlanfixAnaliticCommentName,
			cfg.PlanfixAnaliticDateName,
			cfg.PlanfixAnaliticUsersName,
		)
		if err != nil {
			return fmt.Errorf("Поля аналитики указаны неправильно: %s", err.Error())
		}
	}
	return nil
}

func main() {
	fmt.Printf("planfix-toggl %s\n", version)

	cfg := config.GetConfig()

	parseFlags(&cfg)

	logger := getLogger(cfg)

	errors, ok := cfg.Validate()
	if !ok {
		for _, e := range errors {
			log.Println(e)
		}
		os.Exit(1)
	}

	if cfg.NoConsole {
		util.HideConsole()
	}

	// create planfix client
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

	// create toggl client
	sess := toggl.OpenSession(cfg.TogglAPIToken)
	togglClient := client.TogglClient{
		Session:    &sess,
		Config:     &cfg,
		PlanfixAPI: planfixAPI,
		Logger:     logger,
	}

	// get planfix and toggl user IDs, for early API check
	err := connectServices(&cfg, logger, togglClient)
	if err != nil {
		logger.Fatalf("[ERROR] %s", err.Error())
	}

	// start tag cleaner
	go togglClient.RunTagCleaner()

	// start sender
	go togglClient.RunSender()

	// start API server
	server := rest.Server{
		Version:     version,
		TogglClient: togglClient,
		Config:      cfg,
		Logger:      logger,
	}
	server.Run()
}
