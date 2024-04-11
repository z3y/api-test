package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (a *Api) Run() error {

	router := mux.NewRouter()

	router.HandleFunc("/user/{uuid}", a.handleUser)

	http.ListenAndServe(a.listenAddr, router)
	return nil
}

func writeJson(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, error string) {
	writeJson(w, http.StatusPreconditionFailed, ErrorResponse{Error: error})
}

type UserResponse struct {
	Uuid     string    `json:"id"`
	Username string    `json:"username"`
	JoinDate time.Time `json:"dateJoined"`
}

func (a *Api) handleUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	user, err := a.storage.GetUserByUuid(uuid)
	if err != nil {
		fmt.Println(err)
		writeError(w, "invalid user")
		return
	}

	resp := UserResponse{
		Username: user.username,
		JoinDate: user.dateJoined,
		Uuid:     user.uuid.String(),
	}

	writeJson(w, http.StatusPreconditionFailed, resp)
}
