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


// structs 

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

type Chirp struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
}

// helper functions

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

func validateChirp(body string) bool {
	if len(body) > 140 {
		return false
	}
	return true
}

func toChirp(row database.Chirp) Chirp {
	return Chirp {
		ID: row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Body: row.Body,
		UserID: row.UserID
	}
}

func toUser(row database.User) User {
	return User {
		ID: row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Email: row.Email
	}
}

func toChirps(rows []database.Chirp) []Chirp {
	chirps := make([]Chirp, len(rows))
	for i, row := range rows {
		chirps[i] = toChirp(row)
	}
	return chirps
}

// middlewares

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// handlers

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

func handlerGetHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	
}


func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	sqlChirps, err := cfg.dbQueries.ListChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error listing chirps: %v", err), err)
		return
	}
	chirps := toChirps(sqlChirps)
	respondWithJson(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type chirpBody struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	
	decoder := json.NewDecoder(r.Body)
	chirp := chirpBody{}
	err := decoder.Decode(&chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding json: %v", err), err)
		return
	}
	if !validateChirp(chirp.Body) {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Chirp is not valid"), nil)
		return
	}
	params := database.CreateChirpParams { Body: chirp.Body, UserID: chirp.UserId }
	chrp, err := cfg.dbQueries.CreateChirp(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating chirp: %v", err), err)
		return
	}
	chirpToSend := toChirp(chrp)
	
	respondWithJson(w, http.StatusCreated, chirpToSend)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
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
	userForSend := toUser(usr)
	respondWithJson(w, http.StatusCreated, userForSend)	
}




// main func

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
	apiCfg := apiConfig{ dbQueries: queries, platform: platform }
	port := "8080"
  filepath := "."
  mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlerGetHealth)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
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
