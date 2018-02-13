package rest

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/viasite/planfix-toggl-server/app/client"
)

// Server is a rest with store
type Server struct {
	Version        string
	TogglClient    client.TogglClient
}

//Run the lister and request's router, activate rest server
func (s Server) Run() {
	log.Printf("[INFO] activate rest server")

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	//router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	//router.Use(Limiter(10), AppInfo("secrets", s.Version), Ping)
	router.Use(AppInfo("planfix-toggl", s.Version), Ping)
	//router.Use(Rewrite("/show/(.*)", "/show/?$1"))

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(Logger())
		r.Get("/toggl/entries", s.getEntriesCtrl)
		r.Get("/params", s.getParamsCtrl)
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /api/")
	})

	s.fileServer(router, "/", http.Dir(filepath.Join(".", "docroot")))

	log.Fatal(http.ListenAndServe(":8096", router))
}

// GET /v1/toggl/entries
func (s Server) getEntriesCtrl(w http.ResponseWriter, r *http.Request) {
	entries, err := s.TogglClient.GetEntries("last")
	if err != nil {
		log.Printf("[WARN] failed to load entries")
	} else {
		//status = http.StatusOK
	}

	//render.Status(r, status)
	render.JSON(w, r, entries)
}

// GET /params
func (s Server) getParamsCtrl(w http.ResponseWriter, r *http.Request) {
	params := struct {
		PlanfixAccount string `json:"planfix_account"`
	}{
		PlanfixAccount: "tagilcity",
	}
	render.JSON(w, r, params)
}

// serves static files from ./docroot
func (s Server) fileServer(r chi.Router, path string, root http.FileSystem) {
	log.Printf("[INFO] run file server for %s", root)
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