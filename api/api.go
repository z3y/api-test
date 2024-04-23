package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Api struct {
	listenAddr string
	storage    *Storage
}

func NewApi(listenAddr string, storage *Storage) *Api {
	return &Api{
		listenAddr: listenAddr,
		storage:    storage,
	}
}

func (a *Api) Run() {

	http.HandleFunc("GET /users/{uuid}", a.handleUsers) // get user by id
	http.HandleFunc("/user", withJwt(a.handleUser))     // get or delete current user
	http.HandleFunc("POST /register", a.handleRegister) // register
	http.HandleFunc("POST /login", a.handleLogin)       // login and get a token
	http.HandleFunc("GET /exists", a.handleExists)      // login and get a token

	log.Fatal(http.ListenAndServe(a.listenAddr, nil))
}

func writeJson(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

func unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

type UserResponse struct {
	Username string    `json:"username"`
	Uuid     string    `json:"id"`
	JoinDate time.Time `json:"dateJoined"`
}

type DeleteRequest struct {
	Password string `json:"password"`
}

func (a *Api) handleUsers(w http.ResponseWriter, r *http.Request) {
	uuid := r.PathValue("uuid")

	user, err := a.storage.GetUserByUuid(uuid)
	if err != nil {
		fmt.Println(err)
		writeJson(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	resp := UserResponse{
		Username: user.username,
		JoinDate: user.dateJoined,
		Uuid:     user.uuid.String(),
	}

	writeJson(w, http.StatusOK, resp)
}

func (a *Api) handleUser(w http.ResponseWriter, r *http.Request) {

	authId := r.Context().Value(ContextUserIdKey)

	if authId == nil || authId.(string) == "" {
		unauthorized(w)
		return
	}

	user, err := a.storage.GetUserByUuid(authId.(string))
	if err != nil {
		fmt.Println(err)
		writeJson(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if r.Method == http.MethodGet {

		resp := UserResponse{
			Username: user.username,
			JoinDate: user.dateJoined,
			Uuid:     user.uuid.String(),
		}

		writeJson(w, http.StatusOK, resp)
	} else if r.Method == http.MethodDelete {

		err = a.storage.DeleteUser(user.uuid.String())
		if err != nil {
			fmt.Println(err)
			badRequest(w)
			return
		}
	}
}

func (a *Api) handleRegister(w http.ResponseWriter, r *http.Request) {

	authorization := r.Header.Get("Authorization")
	username, password, err := praseBasicAuthentication(authorization)
	if err != nil {
		writeJson(w, http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	if taken, _ := a.storage.UsernameTaken(username); taken {
		writeJson(w, http.StatusConflict, ErrorResponse{Error: "username taken"})
		return
	}

	newUsr, err := a.storage.NewUser(username, password)
	if err != nil {
		badRequest(w)
		return
	}

	resp := UserResponse{
		Username: newUsr.username,
		JoinDate: newUsr.dateJoined,
		Uuid:     newUsr.uuid.String(),
	}

	writeJson(w, http.StatusOK, resp)
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (a *Api) handleLogin(w http.ResponseWriter, r *http.Request) {

	authorization := r.Header.Get("Authorization")
	username, password, err := praseBasicAuthentication(authorization)
	if err != nil {
		writeJson(w, http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	valid, id := a.storage.LoginValid(string(username), string(password))
	if !valid {
		writeJson(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
		return
	}

	token, err := CreateJwt(id)
	if err != nil {
		fmt.Println(err)
		badRequest(w)
		return
	}

	resp := LoginResponse{
		Token: token,
	}

	writeJson(w, http.StatusOK, resp)
}

func extractAuthFromHeader(r *http.Request) (string, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return "", fmt.Errorf("token not found")
	}

	// The Authorization header should be in the format "Bearer <token>"
	tokenParts := strings.Split(authorizationHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fmt.Errorf("invalid token")
	}

	return tokenParts[1], nil
}

type ContextKey string

const ContextUserIdKey ContextKey = "id"

func withJwt(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		jwt, err := extractAuthFromHeader(r)
		if err != nil {
			unauthorized(w)
			return
		}

		claims, err := ValidateAndParseJwt(jwt)
		if err != nil {
			unauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), ContextUserIdKey, claims["id"])

		f(w, r.WithContext(ctx))
	}
}

func praseBasicAuthentication(authorization string) (username, password string, err error) {
	const prefix = "Basic "
	if !strings.HasPrefix(authorization, prefix) {
		return "", "", fmt.Errorf("authorization has no prefix 'Basic '")
	}

	basicB64 := strings.TrimPrefix(authorization, prefix)

	basic, err := base64.URLEncoding.DecodeString(basicB64)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode from base64")
	}

	authParts := strings.SplitN(string(basic), ":", 2)
	if len(authParts) != 2 {
		return "", "", fmt.Errorf("failed to split username:password")
	}

	return authParts[0], authParts[1], nil
}

func (a *Api) handleExists(w http.ResponseWriter, r *http.Request) {

	username := r.URL.Query().Get("username")
	if username == "" {
		writeJson(w, http.StatusNotFound, ErrorResponse{Error: "no username query"})
		return
	}

	exists, err := a.storage.UsernameTaken(username)
	if err != nil {
		badRequest(w)
		return
	}

	writeJson(w, http.StatusOK, map[string]bool{"user_exists": exists})
}
