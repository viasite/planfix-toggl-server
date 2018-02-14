package main

import (
	"os"
	"fmt"
	"log"

	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/popstas/go-toggl"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/rest"
)

var revision string

func main() {
	fmt.Printf("planfix-toggl %s\n", revision)

	cfg := config.GetConfig()
	if (cfg.SmtpSecure){
		err := "[ERR] Secure SMTP not implemented"
		log.Fatal(err)
		os.Exit(1)
	}

	sess := toggl.OpenSession(cfg.ApiToken)
	TogglClient := client.TogglClient{
		Session: sess,
		Config:  cfg,
	}

	log.SetFlags(log.Ldate | log.Ltime)
	if cfg.Debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	server := rest.Server{
		Version:     revision,
		TogglClient: TogglClient,
		Config:      cfg,
	}
	server.Run()
}
