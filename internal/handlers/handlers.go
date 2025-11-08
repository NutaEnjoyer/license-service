package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"license-service/internal/middlewares"
	"license-service/internal/models"
	"license-service/internal/storage"
)

type Handler struct {
	storage storage.Storage
	logger  log.Logger
}

// constructor
func NewHandler(storage storage.Storage) *Handler {
	return &Handler{storage: storage, logger: *log.Default()}
}

type HealthResponse struct {
	Health bool `json:"health"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{Health: true}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type AddLicenseRequest struct {
	Owner      string `json:"owner"`
	Product    string `json:"product"`
	OneTime    bool   `json:"one_time"`
	ExpireTime int64  `json:"expire_time"`
}

func (h *Handler) AddLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	login, ok := r.Context().Value(middlewares.UserLoginKey).(string)
	if !ok || login == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req AddLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ExpireTime < 5 {
		http.Error(w, "Expire time must be more than 5 minutes", http.StatusBadRequest)
		return
	}

	license := models.License{
		Owner:      login,
		OneTime:    req.OneTime,
		Project:    req.Product,
		ExpireTime: time.Now().Add(time.Duration(req.ExpireTime) * time.Minute),
		CreatedAt:  time.Now(),
	}

	key, err := h.storage.Add(license)
	if err != nil {
		h.logger.Printf("Error adding license: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "License added successfully",
		"key":     key,
	})
}

type CheckLicenseResponse struct {
	Valid      bool      `json:"valid"`
	ExpireTime time.Time `json:"expire_time"`
	Message    string    `json:"message"`
}

func (h *Handler) CheckLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	valid, err := h.storage.IsValid(key)

	response := CheckLicenseResponse{}

	if err != nil {
		h.logger.Printf("Error %s", err)
		response.Valid = false
		response.Message = "Invalid license key"
	} else {
		response.Valid = valid
		if valid {
			license, err := h.storage.Get(key)
			if err != nil {
				response.Valid = false
				response.Message = "Invalid license key 2"
			} else {
				response.Valid = true
				response.ExpireTime = license.ExpireTime
				response.Message = "License is valid"

				if time.Now().After(response.ExpireTime) {
					response.Valid = false
					response.Message = "License expired"
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type InvalidLicenseResponse struct {
	Key     string `json:"key"`
	Success bool   `json:"success"`
}

func (h *Handler) InvalidKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	login, ok := r.Context().Value(middlewares.UserLoginKey).(string)
	if !ok || login == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}
	err := h.storage.InvalidLicense(login, key)
	if err != nil {
		h.logger.Printf("Error invalidating license: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "License invalided successfully",
		"key":     key,
	})
}

type ExtendRequest struct {
	AdditionalTime int64 `json:"additional_time"`
}

func (h *Handler) ExtendKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	login, ok := r.Context().Value(middlewares.UserLoginKey).(string)
	if !ok || login == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key := r.URL.Query().Get("key")

	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	var req ExtendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AdditionalTime < 5 {
		http.Error(w, "Additional time must be more than 5 minutes", http.StatusBadRequest)
		return
	}

	err := h.storage.ExtendKey(login, key, req.AdditionalTime)
	if err != nil {
		h.logger.Printf("Error extending license: %v", err)
		// Различаем ошибки для более информативного ответа
		errMsg := err.Error()
		if strings.Contains(errMsg, "license not found") || strings.Contains(errMsg, "does not belong to user") {
			http.Error(w, "License not found or access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "License extended successfully",
		"key":     key,
	})
}
