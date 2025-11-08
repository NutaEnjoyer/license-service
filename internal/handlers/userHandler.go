package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"license-service/internal/models"
	"license-service/internal/utils"
)

type RegisterLoginRequest struct {
	Login     string  `json:"login"`
	Password  string  `json:"password"`
}

type RegisterLoginResponse struct {
	Ok          bool    `json:"ok"`
	Message     string  `json:"message"`
	AccessToken string  `json:"access_token"`
}

var Logger = log.Default()


func (h *Handler) Register (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and Password are required", http.StatusBadRequest)
		return
	}

	if len(req.Login) < 6 || len(req.Password) < 8 {
		http.Error(w, "Login and/or Password invalid", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Println("Error: ", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user := models.User{
		Login: req.Login,
		PasswordHash: string(hash),
	}

	free, err := h.storage.IsLoginFree(user.Login)

	if err != nil {
		Logger.Println("Error: ", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !free {
		http.Error(w, "Login is already taken", http.StatusBadRequest)
		return
	}

	err = h.storage.CreateUser(user)

	if err != nil {
		log.Println("Error: ", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.Login)

	if err != nil {
		log.Println("Error: ", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	} 

	response := RegisterLoginResponse{
		Ok: true,
		Message: "User successfully created",
		AccessToken: accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Login (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and Password are required", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUser(req.Login)

	if err != nil {
		http.Error(w, "Failed to login", http.StatusBadRequest)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))

	if err != nil {
		http.Error(w, "Failed to login. Incorrect login or password", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.Login)

	if err != nil {
		http.Error(w, "Failed to login", http.StatusBadRequest)
		return
	}

	response := RegisterLoginResponse{
		Ok: true,
		Message: "User successfully logined",
		AccessToken: accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
