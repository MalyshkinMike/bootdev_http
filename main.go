package main

import (
   "net/http"
   "log"
	 "sync/atomic"
	 "fmt"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func healthcheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	
}

func main() {
	port := "8080"
  filepath := "."
  mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthcheck)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepath)))
	apiCfg := apiConfig{}
	mux.HandleFunc("GET /api/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /api/reset", apiCfg.handlerResetMetrics)
  mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
  server := &http.Server{
		Addr: ":" + port,
			Handler: mux,
  }
  log.Printf("Serving files from %s on port: %s\n", filepath, port)
  log.Fatal(server.ListenAndServe())
}
