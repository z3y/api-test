package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

	router := mux.NewRouter()

	router.HandleFunc("/users/{uuid}", a.handleUsers) // get user by id
	router.HandleFunc("/user", withJwt(a.handleUser)) // get current user
	router.HandleFunc("/register", a.handleRegister)  // register
	router.HandleFunc("/login", a.handleLogin)        // login and get a token

	http.ListenAndServe(a.listenAddr, router)
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

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	authId := r.Context().Value(ContextUserIdKey)
	if authId.(string) != uuid {
		unauthorized(w)
		return
	}

	user, err := a.storage.GetUserByUuid(uuid)
	if err != nil {
		fmt.Println(err)
		writeJson(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if r.Method == "GET" {

		resp := UserResponse{
			Username: user.username,
			JoinDate: user.dateJoined,
			Uuid:     user.uuid.String(),
		}

		writeJson(w, http.StatusOK, resp)

	} else if r.Method == "DELETE" {

		req := DeleteRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w)
			return
		}

		if !PasswordValid(user.encryptedPassword, req.Password) {
			badRequest(w)
			return
		}

		err = a.storage.DeleteUser(uuid)
		if err != nil {
			fmt.Println(err)
			badRequest(w)
			return
		}

	} else {
		badRequest(w)
		return
	}

}

func (a *Api) handleUser(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		badRequest(w)
		return
	}

	authId := r.Context().Value(ContextUserIdKey)

	user, err := a.storage.GetUserByUuid(authId.(string))
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

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *Api) handleRegister(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		badRequest(w)
		return
	}

	reg := RegisterRequest{}
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		badRequest(w)
		return
	}

	if taken, _ := a.storage.UsernameTaken(reg.Username); taken {
		writeJson(w, http.StatusConflict, ErrorResponse{Error: "username taken"})
		return
	}

	newUsr, err := a.storage.NewUser(reg.Username, reg.Password)
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

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (a *Api) handleLogin(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		badRequest(w)
		return
	}

	login := LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		badRequest(w)
		return
	}

	valid, id := a.storage.LoginValid(login.Username, login.Password)
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

func extractTokenFromHeader(r *http.Request) (string, error) {
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

		jwt, err := extractTokenFromHeader(r)
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
