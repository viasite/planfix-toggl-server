package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"flag"
	"github.com/popstas/go-toggl"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/util"
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

func connectServices(cfg *config.Config, logger *log.Logger, togglClient client.TogglClient) (err error) {
	// toggl
	logger.Println("[INFO] подключение к Toggl...")
	account, err := togglClient.GetTogglUser()
	cfg.TogglUserID = account.Data.ID
	if err != nil {
		return err
	}

	ok, err := togglClient.IsWorkspaceExists(cfg.TogglWorkspaceID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Toggl workspace ID %d не найден", cfg.TogglWorkspaceID)
	}

	// planfix
	if cfg.PlanfixUserName != "" && cfg.PlanfixUserPassword != "" {
		logger.Println("[INFO] подключение к Планфиксу...")
		user, err := togglClient.GetPlanfixUser()
		cfg.PlanfixUserID = user.ID
		if err != nil {
			return err
		}

		logger.Println("[INFO] получение данных аналитики Планфикса...")
		_, err = togglClient.GetAnaliticDataCached(
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

	errors, isValid := cfg.Validate()
	if !isValid {
		for _, e := range errors {
			log.Println(e)
		}
	}

	if cfg.NoConsole {
		util.HideConsole()
	}

	togglClient := client.TogglClient{
		Config: &cfg,
		Logger: logger,
	}
	togglClient.ReloadConfig()

	// get planfix and toggl user IDs, for early API check
	err := connectServices(&cfg, logger, togglClient)
	if err != nil {
		isValid = false
		logger.Printf("[ERROR] %s", err.Error())
	}

	if isValid {
		togglClient.Run()
	} else {
		util.OpenBrowser("https://localhost:8097")
	}

	// start API server
	server := rest.Server{
		Version:     version,
		TogglClient: &togglClient,
		Config:      &cfg,
		Logger:      logger,
	}
	server.Run()
}
