package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"flag"
	"github.com/getlantern/systray"
	"github.com/popstas/go-toggl"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/viasite/planfix-toggl-server/app/icon"
	"github.com/viasite/planfix-toggl-server/app/rest"
	"github.com/viasite/planfix-toggl-server/app/util"
	"runtime"
)

var version string
var trayMenu map[string] *systray.MenuItem

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
		//defer f.Close()
		// file should be first, when -ldflags -H=windowsgui build, Stdout absent and block log output
		mw := io.MultiWriter(f, os.Stdout)
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

func connectServices(cfg *config.Config, logger *log.Logger, togglClient *client.TogglClient) (err error) {
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

func initApp() {
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
		Config:  &cfg,
		Logger:  logger,
		SentLog: make(map[string]int),
		Opts:    map[string]string{"LastSent": ""},
	}
	togglClient.ReloadConfig()

	// get planfix and toggl user IDs, for early API check
	err := connectServices(&cfg, logger, &togglClient)
	if err != nil {
		isValid = false
		logger.Printf("[ERROR] %s", err.Error())
		util.Notify(err.Error())
	}

	trayMenu["web"].Enable()
	trayMenu["log"].Enable()

	if isValid {
		trayMenu["send"].Enable()
		togglClient.Run()
	} else {
		util.OpenBrowser(fmt.Sprintf("https://localhost:%d/#settings", cfg.PortSSL))
	}

	// update last sent on menuitem
	go func() {
		for {
			if togglClient.Opts["LastSent"] != "" {
				t := togglClient.Opts["LastSent"]
				trayMenu["send"].SetTitle(fmt.Sprintf("Sync (last at %s)", t))
			}
			time.Sleep(time.Duration(60 * time.Second))
		}
	}()

	go func() {
		// tray menu actions
		for {
			select {
			case <-trayMenu["send"].ClickedCh:
				err := togglClient.SendToPlanfix()
				t := togglClient.Opts["LastSent"]
				trayMenu["send"].SetTitle(fmt.Sprintf("Sync (last at %s)", t))
				if err != nil {
					logger.Println(err)
				}

			case <-trayMenu["web"].ClickedCh:
				cfg := config.GetConfig()
				util.OpenBrowser(fmt.Sprintf("https://localhost:%d", cfg.PortSSL))

			case <-trayMenu["log"].ClickedCh:
				cfg := config.GetConfig()
				systray.ShowAppWindow(fmt.Sprintf("https://localhost:%d/log", cfg.PortSSL))

			case <-trayMenu["quit"].ClickedCh:
				onExit()
			}
		}
	}()

	// start API server
	server := rest.Server{
		Version:     version,
		TogglClient: &togglClient,
		Config:      &cfg,
		Logger:      logger,
	}
	server.Run(cfg.PortSSL)
}

func onReady() {
	go initApp()

	// systray.EnableAppWindow("Lantern", 1024, 768) // in next systray versions
	systray.SetIcon(icon.Data)
	systray.SetTitle("planfix-toggl")
	systray.SetTooltip(fmt.Sprintf("planfix-toggl %s", version))

	trayMenu = make(map[string]*systray.MenuItem)
	trayMenu["send"] = systray.AddMenuItem("Sync", "")
	trayMenu["web"] = systray.AddMenuItem("Open web interface", "")
	trayMenu["log"] = systray.AddMenuItem("Open log", "")
	trayMenu["quit"] = systray.AddMenuItem("Quit", "Quit the whole app")

	trayMenu["send"].Disable()
	trayMenu["web"].Disable()
	trayMenu["log"].Disable()
}

func onExit() {
	systray.Quit()
	//os.Exit(0)
}

func main() {
	//systray.Run(onReady, onExit)
	systray.RunWithAppWindow("planfix-toggl", 1024, 768, onReady, onExit)
}
