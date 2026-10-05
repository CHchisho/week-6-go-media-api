package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type App struct {
	db        *sql.DB
	jwtSecret []byte
}

func main() {
	db, err := connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &App{
		db:        db,
		jwtSecret: []byte(env("JWT_SECRET", "JWT_SECRET_JWT_SECRET")),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.home)
	mux.HandleFunc("GET /api/health", app.health)
	mux.HandleFunc("GET /openapi.yaml", serveOpenAPI)

	mux.HandleFunc("POST /api/register", app.register)
	mux.HandleFunc("POST /api/login", app.login)

	mux.HandleFunc("GET /api/movies", app.listMovies)
	mux.HandleFunc("GET /api/movies/{id}", app.getMovie)
	mux.Handle("POST /api/movies", app.requireAuth(http.HandlerFunc(app.createMovie)))
	mux.Handle("PUT /api/movies/{id}", app.requireAuth(http.HandlerFunc(app.updateMovie)))
	mux.Handle("DELETE /api/movies/{id}", app.requireAuth(http.HandlerFunc(app.deleteMovie)))

	port := env("PORT", "8080")
	log.Printf("media API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, logRequests(mux)))
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"name": "week-6-go-media-api",
		"docs": "/openapi.yaml",
	})
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	if err := a.db.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "openapi.yaml")
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	return json.NewDecoder(r.Body).Decode(dst)
}
