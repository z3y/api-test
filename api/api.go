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
	router.HandleFunc("/register", a.handleRegister)

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

func badRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

type UserResponse struct {
	Username string    `json:"username"`
	Uuid     string    `json:"id"`
	JoinDate time.Time `json:"dateJoined"`
}

type DeleteRequest struct {
	Password string `json:"password"`
}

func (a *Api) handleUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	if r.Method == "GET" {

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

	} else if r.Method == "DELETE" {
		usr, err := a.storage.GetUserByUuid(uuid)
		if err != nil {
			fmt.Println(err)
			badRequest(w)
			return
		}

		req := DeleteRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w)
			return
		}

		if !usr.ValidatePassword(req.Password) {
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
