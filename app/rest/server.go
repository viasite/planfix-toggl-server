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
	"encoding/json"
	"github.com/popstas/go-toggl"
)

// Server is a rest with store
type Server struct {
	Version     string
	TogglClient *client.TogglClient
	Config      *config.Config
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
	router.Use(CORS)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(Logger())

		// toggl
		r.Route("/toggl", func(r chi.Router) {
			r.Get("/entries", s.getEntriesCtrl)
			r.Get("/entries/planfix/{taskID}", s.getPlanfixTaskCtrl)
			r.Get("/entries/planfix/{taskID}/last", s.getPlanfixTaskLastCtrl)
			r.Get("/user", s.getTogglUser)
			r.Get("/workspaces", s.getTogglWorkspaces)
		})

		// config
		r.Route("/config", func(r chi.Router) {
			r.Get("/", s.getConfigCtrl)
			r.Options("/", s.updateConfigCtrl)
			r.Post("/", s.updateConfigCtrl)
			r.Post("/reload", s.reloadConfigCtrl)
		})

		// planfix
		r.Route("/planfix", func(r chi.Router) {
			r.Get("/user", s.getPlanfixUser)
		})

		// validate
		r.Route("/validate", func(r chi.Router) {
			r.Get("/config", s.validateConfig)
			r.Get("/planfix/user", s.validatePlanfixUser)
			r.Get("/planfix/analitic", s.validatePlanfixAnalitic)
			r.Get("/toggl/user", s.validateTogglUser)
			r.Get("/toggl/workspace", s.validateTogglWorkspace)
		})
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /api/")
	})

	s.fileServer(router, "/", http.Dir(filepath.Join(".", "docroot")))

	//go http.ListenAndServe(fmt.Sprintf(":%d", port), router)
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

// GET /v1/config
func (s Server) getConfigCtrl(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, config.GetConfig())
}

// POST /v1/config
func (s Server) updateConfigCtrl(w http.ResponseWriter, r *http.Request) {
	// answer to OPTIONS request for content-type
	if r.Method == "OPTIONS" {
		if r.Header.Get("Access-Control-Request-Method") == "content-type" {
			w.Header().Set("Content-Type", "application/json")
		}
		return
	}

	newConfig := config.GetConfig()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newConfig)
	if err != nil {
		s.Logger.Printf("[ERROR] Cannot decode %v", err.Error())
	}

	errors, _ := newConfig.Validate()
	if len(errors) == 0 {
		newConfig.SaveConfig()
	}
	render.JSON(w, r, errors)
}

// POST /v1/config/reload
func (s *Server) reloadConfigCtrl(w http.ResponseWriter, r *http.Request) {
	newConfig := config.GetConfig()
	s.Config = &newConfig
	s.TogglClient.Config = &newConfig
	s.TogglClient.ReloadConfig()
}

type ValidatorStatus struct {
	Ok     bool        `json:"ok"`
	Errors []string    `json:"errors"`
	Data   interface{} `json:"data"`
}

// GET /api/v1/validate/config
func (s Server) validateConfig(w http.ResponseWriter, r *http.Request) {
	v := client.ConfigValidator{s.Config}
	render.JSON(w, r, client.StatusFromCheck(v.Check()))
}

// GET /api/v1/validate/planfix/user
func (s Server) validatePlanfixUser(w http.ResponseWriter, r *http.Request) {
	v := client.PlanfixUserValidator{s.TogglClient}
	render.JSON(w, r, client.StatusFromCheck(v.Check()))
}

// GET /api/v1/validate/planfix/analitic
func (s Server) validatePlanfixAnalitic(w http.ResponseWriter, r *http.Request) {
	v := client.PlanfixAnaliticValidator{s.TogglClient}
	render.JSON(w, r, client.StatusFromCheck(v.Check()))
}

// GET /api/v1/validate/toggl/user
func (s Server) validateTogglUser(w http.ResponseWriter, r *http.Request) {
	v := client.TogglUserValidator{s.TogglClient}
	render.JSON(w, r, client.StatusFromCheck(v.Check()))
}

// GET /api/v1/validate/toggl/workspace
func (s Server) validateTogglWorkspace(w http.ResponseWriter, r *http.Request) {
	v := client.TogglWorkspaceValidator{s.TogglClient}
	render.JSON(w, r, client.StatusFromCheck(v.Check()))
}

// GET /api/v1/planfix/user
func (s Server) getPlanfixUser(w http.ResponseWriter, r *http.Request) {
	v := client.PlanfixUserValidator{s.TogglClient}
	errors, ok, data := v.Check()
	render.JSON(w, r, ValidatorStatus{ok, errors, data})
}

// GET /api/v1/toggl/user
func (s Server) getTogglUser(w http.ResponseWriter, r *http.Request) {
	var user toggl.Account
	var errors []string;
	user, err := s.TogglClient.Session.GetAccount()
	if err != nil {
		msg := "Не удалось получить Toggl UserID, проверьте TogglAPIToken, %s"
		errors = append(errors, fmt.Sprintf(msg, err.Error()))
	}

	render.JSON(w, r, ValidatorStatus{err == nil, errors, user.Data})
}

// GET /api/v1/toggl/workspaces
func (s Server) getTogglWorkspaces(w http.ResponseWriter, r *http.Request) {
	var workspaces []toggl.Workspace
	var errors []string;
	workspaces, err := s.TogglClient.Session.GetWorkspaces()
	if err != nil {
		msg := "Не удалось получить Toggl workspaces, проверьте TogglAPIToken, %s"
		errors = append(errors, fmt.Sprintf(msg, err.Error()))
	}

	render.JSON(w, r, ValidatorStatus{err == nil, errors, workspaces})
}

// GET /toggl/entries/planfix/{taskID}
func (s Server) getPlanfixTaskCtrl(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	entries, _ := s.TogglClient.GetEntriesByTag(taskID)
	render.JSON(w, r, entries)
}

// GET /toggl/entries/planfix/{taskID}/last
func (s Server) getPlanfixTaskLastCtrl(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	entries, _ := s.TogglClient.GetEntriesByTag(taskID)
	if len(entries) > 0 {
		render.JSON(w, r, entries[0])
	} else {
		render.JSON(w, r, entries)
	}
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
