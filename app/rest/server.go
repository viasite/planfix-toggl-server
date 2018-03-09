package rest

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"fmt"
	"github.com/viasite/planfix-toggl-server/app/client"
	"github.com/viasite/planfix-toggl-server/app/config"
	"time"
)

// Server is a rest with store
type Server struct {
	Version     string
	TogglClient client.TogglClient
	Config      config.Config
	Logger      *log.Logger
}

//Run the lister and request's router, activate rest server
func (s Server) Run() {
	//port := 8096
	portSSL := 8097
	//s.Logger.Printf("[INFO] запуск сервера на :%d", port)

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(AppInfo("planfix-toggl", s.Version), Ping)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(Logger())
		r.Get("/toggl/entries", s.getEntriesCtrl)
		r.Get("/toggl/planfix-task/{taskID}", s.getPlanfixTaskCtrl)
		r.Get("/toggl/planfix-task/{taskID}/last", s.getPlanfixTaskLastCtrl)
		r.Get("/params", s.getParamsCtrl)
		r.Get("/config/validate", s.getValidateConfigCtrl)
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /api/")
	})

	s.fileServer(router, "/", http.Dir(filepath.Join(".", "docroot")))

	s.Logger.Printf("[INFO] веб-интерфейс на https://localhost:%d", portSSL)
	s.Logger.Println(http.ListenAndServeTLS(
		fmt.Sprintf(":%d", portSSL),
		"certs/server.crt",
		"certs/server.key", router),
	)

	//s.Logger.Printf("[INFO] веб-интерфейс на http://localhost:%d", port)
	//s.Logger.Println(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

// GET /v1/toggl/entries
func (s Server) getEntriesCtrl(w http.ResponseWriter, r *http.Request) {
	var entries []client.TogglPlanfixEntry
	var err error
	w.Header().Set("Access-Control-Allow-Origin", "*")
	queryValues := r.URL.Query()
	t := queryValues.Get("type")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	if t == "today" {
		entries, err = s.TogglClient.GetEntries(
			s.Config.TogglWorkspaceID,
			time.Now().Format("2006-01-02"),
			tomorrow,
		)
	} else if t == "pending" {
		entries, err = s.TogglClient.GetPendingEntries()
	} else if t == "last" {
		entries, err = s.TogglClient.GetEntries(
			s.Config.TogglWorkspaceID,
			time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
			tomorrow,
		)
	}
	if err != nil {
		s.Logger.Printf("[WARN] failed to load entries")
	} else {
		//status = http.StatusOK
	}

	entries = s.TogglClient.SumEntriesGroup(s.TogglClient.GroupEntriesByTask(entries))

	//render.Status(r, status)
	render.JSON(w, r, entries)
}

// TODO
// GET /config/validate
func (s Server) getValidateConfigCtrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	errors, _ := s.Config.Validate()
	//getValidateConfigCtrl
	render.JSON(w, r, errors)
}

// GET /toggl/planfix-task/{taskID}
func (s Server) getPlanfixTaskCtrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	taskID := chi.URLParam(r, "taskID")
	entries, _ := s.TogglClient.GetEntriesByTag(taskID)
	render.JSON(w, r, entries)
}

// GET /toggl/planfix-task/{taskID}/last
func (s Server) getPlanfixTaskLastCtrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	taskID := chi.URLParam(r, "taskID")
	entries, _ := s.TogglClient.GetEntriesByTag(taskID)
	if len(entries) > 0 {
		render.JSON(w, r, entries[0])
	} else {
		render.JSON(w, r, entries)
	}
}

// GET /params
func (s Server) getParamsCtrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	params := struct {
		PlanfixAccount string `json:"planfix_account"`
	}{
		PlanfixAccount: s.Config.PlanfixAccount,
	}
	render.JSON(w, r, params)
}

// serves static files from ./docroot
func (s Server) fileServer(r chi.Router, path string, root http.FileSystem) {
	//s.Logger.Printf("[INFO] run file server for %s", root)
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != "/show/" {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}))
}
