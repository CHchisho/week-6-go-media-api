package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userIDKey contextKey = "userID"

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type publicUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  publicUser `json:"user"`
}

type claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var body credentials
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if !usernamePattern.MatchString(body.Username) {
		writeError(w, http.StatusBadRequest, "username must be 3-30 characters: letters, numbers, underscore")
		return
	}
	if len(body.Password) < 6 || len(body.Password) > 72 {
		writeError(w, http.StatusBadRequest, "password must be 6-72 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	result, err := a.db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		body.Username, string(hash),
	)
	if isDuplicate(err) {
		writeError(w, http.StatusConflict, "username is already taken")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	token, err := a.signToken(int(id), body.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  publicUser{ID: int(id), Username: body.Username},
	})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var body credentials
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)

	var user publicUser
	var passwordHash string
	err := a.db.QueryRow(
		`SELECT id, username, password_hash FROM users WHERE username = ?`,
		body.Username,
	).Scan(&user.ID, &user.Username, &passwordHash)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(body.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := a.signToken(user.ID, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (a *App) signToken(userID int, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	return token.SignedString(a.jwtSecret)
}

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || raw == "" || raw == header {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		parsed, err := jwt.ParseWithClaims(raw, &claims{}, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return a.jwtSecret, nil
		})
		if err != nil || !parsed.Valid {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		userClaims, ok := parsed.Claims.(*claims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userClaims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUserID(r *http.Request) (int, bool) {
	id, ok := r.Context().Value(userIDKey).(int)
	return id, ok
}

func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
