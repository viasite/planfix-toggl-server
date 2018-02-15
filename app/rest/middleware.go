package rest

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	//"github.com/go-chi/render"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// AppInfo adds custom app-info to the response header
func AppInfo(app string, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("App-Name", app)
			w.Header().Set("App-Version", version)
			if mhost := os.Getenv("MHOST"); mhost != "" {
				w.Header().Set("Host", mhost)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// Ping middleware response with pong to /ping. Stops chain if ping request detected
func Ping(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/ping") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("pong")); err != nil {
				log.Printf("[WARN] can't send pong, %s", err)
			}
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Recoverer is a middleware that recovers from panics, logs the panic and returns a HTTP 500 status if possible.
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				log.Printf("[WARN] request panic, %v", rvr)
				debug.PrintStack()
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll  LoggerFlag = iota
	LogBody
)
const maxBody = 1024

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(flags ...LoggerFlag) func(http.Handler) http.Handler {

	inFlags := func(f LoggerFlag) bool {
		for _, flg := range flags {
			if flg == LogAll || flg == f {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)

			body := func() (result string) {
				if inFlags(LogBody) {
					if content, err := ioutil.ReadAll(r.Body); err == nil {
						result = string(content)
						r.Body = ioutil.NopCloser(bytes.NewReader(content))

						if len(result) > 0 {
							result = strings.Replace(result, "\n", " ", -1)
							result = reMultWhtsp.ReplaceAllString(result, " ")
						}

						if len(result) > maxBody {
							result = result[:maxBody] + "..."
						}
					}
				}
				return result
			}()

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}
				// hide id and pin
				log.Printf("[INFO] REST %s - %s - %s - %d (%d) - %v %s",
					r.Method, q, strings.Split(r.RemoteAddr, ":")[0],
					ww.Status(), ww.BytesWritten(), t2.Sub(t1), body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}
