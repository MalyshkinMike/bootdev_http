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
	 "github.com/MalyshkinMike/bootdev_http/internal/auth"
)


// structs 

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secret_string string
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
		UserID: row.UserID,
	}
}

func toUser(row database.User) User {
	return User {
		ID: row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Email: row.Email,
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

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirp_id := r.PathValue("id")
	chirp_uuid, err := uuid.Parse(chirp_id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error converting id to uuid: %v", err), err)
		return
	}
	sqlChirp, err := cfg.dbQueries.GetChirp(r.Context(), chirp_uuid)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Error getting chirp: %v", err), err)
		return
	}
	chirp := toChirp(sqlChirp)
	respondWithJson(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type chirpBody struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	JWTToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting bearer token: %v", err), err)
		return
	}

	userId, err := auth.ValidateJWT(JWTToken, cfg.secret_string)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Invalid JWT: %v", err), err)
		return
	}
	
	decoder := json.NewDecoder(r.Body)
	chirp := chirpBody{}
	err = decoder.Decode(&chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding json: %v", err), err)
		return
	}
	if !validateChirp(chirp.Body) {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Chirp is not valid"), nil)
		return
	}
	params := database.CreateChirpParams { Body: chirp.Body, UserID: userId }
	chrp, err := cfg.dbQueries.CreateChirp(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating chirp: %v", err), err)
		return
	}
	chirpToSend := toChirp(chrp)
	
	respondWithJson(w, http.StatusCreated, chirpToSend)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type userBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	user := userBody{}
	err := decoder.Decode(&user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %s", err), err)
		return
	}
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error hashing password: %v", err), err)
		return
	}
	userParams := database.CreateUserParams { 
		Email: user.Email, 
		HashedPassword: hashedPassword, 
	}
	usr, err := cfg.dbQueries.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating user: %s", err), err)
		return
	}
	userForSend := toUser(usr)
	respondWithJson(w, http.StatusCreated, userForSend)	
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type loginBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
		// ExpiresInSeconds int `json:"expires_in_seconds"`
	}
	type respondBody struct {
		ID uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Email string `json:"email"`
    Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	request_body := loginBody{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request_body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding json body: %v", err), err)
		return
	}
	// if request_body.ExpiresInSeconds > 60 {
	// 	request_body.ExpiresInSeconds = 60
	// }
	userSql, err := cfg.dbQueries.GetUserByEmail(r.Context(), request_body.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error fetching user from db: %v", err), err)
		return
	}
	match, err := auth.CheckPasswordHash(request_body.Password, userSql.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error comparing password: %v", err), err)
		return
	}
	if match {
		durationString := fmt.Sprintf("%vs", 60)
		duration, err := time.ParseDuration(durationString)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error parsing duration: %v", err), err)
			return
		}
		token, err := auth.MakeJWT(userSql.ID, cfg.secret_string, duration)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating token: %v", err), err)
			return
		}
		refresh_token := auth.MakeRefreshToken()
		create_token_params := database.CreateTokenParams {
			Token: refresh_token,
			UserID: userSql.ID,
			ExpiresAt: time.Now().AddDate(0, 0, 60),
		}
		_, err = cfg.dbQueries.CreateToken(r.Context(), create_token_params)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating refresh_token in db: %v", err), err)
			return
		}
		userFromSql := respondBody {
			ID: userSql.ID,
			CreatedAt: userSql.CreatedAt,
			UpdatedAt: userSql.UpdatedAt,
			Email: userSql.Email,
			Token: token,
			RefreshToken: refresh_token,
		}
		respondWithJson(w, http.StatusOK, userFromSql)
	} else {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
	}
}


func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type responceBody struct {
		Token string `json:"token"`
	}
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting bearer token: %v", err), err)
		return
	}
	sqltoken, err := cfg.dbQueries.GetToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Token not found: %v", err), err)
		return
	}
	// check expiration
	if sqltoken.ExpiresAt.Compare(time.Now()) <= 0 {
		respondWithError(w, http.StatusUnauthorized, "Token expired", nil)
		return
	}
	// check for revoking
	if sqltoken.RevokedAt.Valid && sqltoken.RevokedAt.Time.Compare(time.Now()) >= 0 {
		respondWithError(w, http.StatusUnauthorized, "Token revoked", nil)
		return
	}
	userId := sqltoken.UserID
	hour, err := time.ParseDuration("1h")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error parsing duration: %v", err), err)
		return
	}
	JWTToken, err := auth.MakeJWT(userId, cfg.secret_string, hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error making access token: %v", err), err)
		return
	}
	responce := responceBody{
		Token: JWTToken,
	}
	respondWithJson(w, http.StatusOK, responce)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	//revoke refresh token by setting revoked_at = currentTimestamp
	err = cfg.dbQueries.RevokeToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error revoking refresh_token: %v", err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	requestData := requestBody{}
	err := decoder.Decode(&requestData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %v", err), err)
		return
	}
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Error getting bearer token: %v", err), err)
		return
	}
	userId, err := auth.ValidateJWT(accessToken, cfg.secret_string)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Invalid JWT: %v", err), err)
		return
	}
	hashedPassword, err := auth.HashPassword(requestData.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error hashing new password: %v", err), err)
		return
	}
	params := database.UpdateUserParams {
		ID: userId,
		Email: requestData.Email,
		HashedPassword: hashedPassword,
	}
	usrSql, err := cfg.dbQueries.UpdateUser(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error updating user: %v", err), err)
		return
	}
	usrToSend := toUser(usrSql)
	respondWithJson(w, http.StatusOK, usrToSend)
}

// main func

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secret_string := os.Getenv("JWT_VERIFY_KEY")
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
	}
	port := "8080"
  filepath := "."
  mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlerGetHealth)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.handlerGetChirp)
  mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
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
