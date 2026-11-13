package main

import _ "github.com/lib/pq"

import (
   "net/http"
   "log"
	 "sync/atomic"
	 "strings"
	 "github.com/joho/godotenv"
   "os"
	 "database/sql"
	 "github.com/MalyshkinMike/bootdev_http/internal/database"
)

// structs 

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secret_string string
	polka_key string
}

// helper functions

func replaceProfaneWords(s string) string {
	output := strings.Split(s, " ")
	profaneWords := make(map[string]struct{})
	profaneWords["kerfuffle"] = struct{}{}
	profaneWords["sharbert"] = struct{}{}
	profaneWords["fornax"] = struct{}{}
	for idx, word := range output {
		if _, exists := profaneWords[strings.ToLower(word)]; exists {
			output[idx] = "****"
		}
	}
	return strings.Join(output, " ")
}

func validateChirp(body string) bool {
	if len(body) > 140 {
		return false
	}
	return true
}


// main func

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secret_string := os.Getenv("JWT_VERIFY_KEY")
	polka_key := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error creating database %v", err)
		return
	}
	queries := database.New(db)
	apiCfg := apiConfig{ 
		dbQueries: queries, 
		platform: platform,
		secret_string: secret_string,
		polka_key: polka_key,
	}
	port := "8080"
  filepath := "."
  mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlerHealthGet)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetricsGet)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerMetricsReset)
	mux.HandleFunc("POST /api/users", apiCfg.handlerUserCreate)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUserUpdate)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirpCreate)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsGet)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.handlerChirpGet)
	mux.HandleFunc("DELETE /api/chirps/{id}", apiCfg.handlerChirpDelete)
  mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.webhookPolka)
	// mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepath)))
  mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
  server := &http.Server{
		Addr: ":" + port,
			Handler: mux,
  }
  log.Printf("Serving files from %s on port: %s\n", filepath, port)
  log.Fatal(server.ListenAndServe())
}
