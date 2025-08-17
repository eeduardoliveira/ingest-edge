// internal/httpserver/server.go
package httpserver

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"ingest-edge/internal/ingest"
	"ingest-edge/internal/mw"
	"ingest-edge/internal/store"
)

func Start() error {
	st, err := store.NewRedisStore()
	if err != nil {
		return err
	}
	rps := envInt("RATE_LIMIT_RPS", 2)
	burst := envInt("RATE_LIMIT_BURST", 4)
	rl := mw.NewRateLimiter(st.Rdb, rps, burst)

	hdl := ingest.NewHandler(st, rl)

	r := mux.NewRouter()
	ing := r.PathPrefix("/ingest").Subrouter()
	ing.Use(mw.AuthMiddleware)
	ing.Use(rl.Middleware)
	ing.HandleFunc("/location", hdl.Ingest).Methods("POST")

	// health
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")

	port := ":" + env("PORT", "8080")
	log.Println("Ingest Edge listening on", port)
	return http.ListenAndServe(port, r)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
