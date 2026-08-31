package main

import _ "github.com/lib/pq"

import (
   "net/http"
   "log"
	 "sync/atomic"
	 "fmt"
	 "encoding/json"
	 "strings"
	 "github.com/joho/godotenv"
	 "github.com/google/uuid"
	 "time"
   "os"
	 "database/sql"
	 "github.com/MalyshkinMike/bootdev_http/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	if cfg.platform == "dev" {
		err := cfg.dbQueries.TruncateUsers(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error deleting users: %v", err), err)
			return
		}
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
	}
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func healthcheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	type errorResponse struct {
		Message string `json: "error"`
	}
	if err != nil {
		log.Printf("Error occured, %s", err)
	}
	resp := errorResponse{ Message: msg }
	respondWithJson(w, code, resp)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(body)
}

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

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Body string `json:"body"`
	}
	type regularResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}
	decoder := json.NewDecoder(r.Body)
	chirp := requestBody{}
	err := decoder.Decode(&chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %s", err), err)
		return
	}
	if len(chirp.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}
	resp := regularResponse{ CleanedBody: replaceProfaneWords(chirp.Body) }
	respondWithJson(w, http.StatusOK, resp)

}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type emailBody struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	email := emailBody{}
	err := decoder.Decode(&email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %s", err), err)
		return
	}
	
	usr, err := cfg.dbQueries.CreateUser(r.Context(), email.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating user: %s", err), err)
		return
	}
	userForSend := User {
		ID: usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email: usr.Email }
	respondWithJson(w, http.StatusCreated, userForSend)	
}


func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error creating database %v", err)
		return
	}
	queries := database.New(db)
	port := "8080"
  filepath := "."
  mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthcheck)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepath)))
	apiCfg := apiConfig{ dbQueries: queries, platform: platform }
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
  mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
  server := &http.Server{
		Addr: ":" + port,
			Handler: mux,
  }
  log.Printf("Serving files from %s on port: %s\n", filepath, port)
  log.Fatal(server.ListenAndServe())
}
