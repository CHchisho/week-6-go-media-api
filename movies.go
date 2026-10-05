package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Movie struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Director    string    `json:"director"`
	Genre       string    `json:"genre"`
	Year        int       `json:"year"`
	Rating      float64   `json:"rating"`
	Description string    `json:"description"`
	OwnerID     int       `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type movieInput struct {
	Title       string  `json:"title"`
	Director    string  `json:"director"`
	Genre       string  `json:"genre"`
	Year        int     `json:"year"`
	Rating      float64 `json:"rating"`
	Description string  `json:"description"`
}

type movieList struct {
	Items []Movie `json:"items"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int     `json:"total"`
}

var allowedGenres = map[string]bool{
	"Action":      true,
	"Comedy":      true,
	"Drama":       true,
	"Horror":      true,
	"Sci-Fi":      true,
	"Documentary": true,
}

const movieSelect = `SELECT id, title, director, genre, release_year, rating, description, owner_id, created_at FROM movies`

func (a *App) listMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	genre := strings.TrimSpace(query.Get("genre"))
	director := strings.TrimSpace(query.Get("director"))

	if genre != "" && !allowedGenres[genre] {
		writeError(w, http.StatusBadRequest, "genre must be one of: Action, Comedy, Drama, Horror, Sci-Fi, Documentary")
		return
	}

	year, err := queryInt(r, "year", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := queryInt(r, "page", 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := queryInt(r, "limit", 10)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if page < 1 || page > 10000 {
		writeError(w, http.StatusBadRequest, "page must be between 1 and 10000")
		return
	}
	if limit < 1 || limit > 50 {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 50")
		return
	}

	order, err := orderBy(query.Get("sort"), query.Get("order"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	where, args := movieFilters(genre, director, year)

	var total int
	countQuery := `SELECT COUNT(*) FROM movies` + where
	if err := a.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "could not list movies")
		return
	}

	listQuery := movieSelect + where + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, err := a.db.Query(listQuery, listArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list movies")
		return
	}
	defer rows.Close()

	items := []Movie{}
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not list movies")
			return
		}
		items = append(items, movie)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not list movies")
		return
	}

	writeJSON(w, http.StatusOK, movieList{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (a *App) getMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	movie, err := a.findMovie(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "movie not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load movie")
		return
	}
	writeJSON(w, http.StatusOK, movie)
}

func (a *App) createMovie(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	var input movieInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := cleanMovie(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := a.db.Exec(
		`INSERT INTO movies (title, director, genre, release_year, rating, description, owner_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Title, input.Director, input.Genre, input.Year, input.Rating, input.Description, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create movie")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create movie")
		return
	}
	movie, err := a.findMovie(int(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create movie")
		return
	}
	writeJSON(w, http.StatusCreated, movie)
}

func (a *App) updateMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if !a.allowOwner(w, r, id) {
		return
	}

	var input movieInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := cleanMovie(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err := a.db.Exec(
		`UPDATE movies SET title = ?, director = ?, genre = ?, release_year = ?, rating = ?, description = ? WHERE id = ?`,
		input.Title, input.Director, input.Genre, input.Year, input.Rating, input.Description, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update movie")
		return
	}
	movie, err := a.findMovie(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update movie")
		return
	}
	writeJSON(w, http.StatusOK, movie)
}

func (a *App) deleteMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if !a.allowOwner(w, r, id) {
		return
	}
	if _, err := a.db.Exec(`DELETE FROM movies WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete movie")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// allowOwner returns false after writing 401, 403, or 404.
// Only the user who created the movie can update or delete it.
func (a *App) allowOwner(w http.ResponseWriter, r *http.Request, movieID int) bool {
	userID, ok := currentUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return false
	}
	movie, err := a.findMovie(movieID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "movie not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load movie")
		return false
	}
	if movie.OwnerID != userID {
		writeError(w, http.StatusForbidden, "you can only change your own movies")
		return false
	}
	return true
}

func (a *App) findMovie(id int) (Movie, error) {
	row := a.db.QueryRow(movieSelect+` WHERE id = ?`, id)
	return scanMovie(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMovie(row scanner) (Movie, error) {
	var movie Movie
	err := row.Scan(
		&movie.ID, &movie.Title, &movie.Director, &movie.Genre,
		&movie.Year, &movie.Rating, &movie.Description, &movie.OwnerID, &movie.CreatedAt,
	)
	return movie, err
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive number")
		return 0, false
	}
	return id, true
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(key + " must be a number")
	}
	return value, nil
}

func movieFilters(genre, director string, year int) (string, []any) {
	where := ""
	args := []any{}
	if genre != "" {
		where += " AND genre = ?"
		args = append(args, genre)
	}
	if year != 0 {
		where += " AND release_year = ?"
		args = append(args, year)
	}
	if director != "" {
		where += " AND director LIKE ? ESCAPE '\\\\'"
		args = append(args, likeContains(director))
	}
	if where != "" {
		where = " WHERE " + strings.TrimPrefix(where, " AND ")
	}
	return where, args
}

func likeContains(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return "%" + value + "%"
}

func orderBy(sort, order string) (string, error) {
	if sort == "" {
		sort = "created"
	}
	columns := map[string]string{
		"title":   "title",
		"year":    "release_year",
		"rating":  "rating",
		"created": "created_at",
	}
	column, ok := columns[sort]
	if !ok {
		return "", errors.New("sort must be one of: title, year, rating, created")
	}
	if order == "" {
		order = "desc"
	}
	switch strings.ToLower(order) {
	case "asc":
		return column + " ASC", nil
	case "desc":
		return column + " DESC", nil
	default:
		return "", errors.New("order must be asc or desc")
	}
}

func cleanMovie(input *movieInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Director = strings.TrimSpace(input.Director)
	input.Genre = strings.TrimSpace(input.Genre)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" || len(input.Title) > 200 {
		return errors.New("title is required and must be at most 200 characters")
	}
	if input.Director == "" || len(input.Director) > 200 {
		return errors.New("director is required and must be at most 200 characters")
	}
	if !allowedGenres[input.Genre] {
		return errors.New("genre must be one of: Action, Comedy, Drama, Horror, Sci-Fi, Documentary")
	}
	maxYear := time.Now().Year() + 1
	if input.Year < 1888 || input.Year > maxYear {
		return errors.New("year is out of range")
	}
	if input.Rating < 0 || input.Rating > 10 {
		return errors.New("rating must be between 0 and 10")
	}
	if input.Description == "" || len(input.Description) > 2000 {
		return errors.New("description is required and must be at most 2000 characters")
	}
	return nil
}
