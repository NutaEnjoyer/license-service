package main

import (
	"log"
	"license-service/internal/handlers"
	"license-service/internal/storage"
	"license-service/internal/middlewares"
	"net/http"
)

const DBPath = "./storage/storage.db"

func main() {
	db, err := storage.NewSQLiteStorage(DBPath)

	if err != nil {
		log.Fatal("Fatal error while creating storage", err.Error())
	}

	defer db.Close()

	handler := handlers.NewHandler(db)
	
	http.HandleFunc("/", handler.Health)
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/login", handler.Login)

	http.HandleFunc("/check", handler.CheckLicense)
	http.Handle("/add", middlewares.AuthMiddleware(http.HandlerFunc(handler.AddLicense)))
	http.Handle("/invalid", middlewares.AuthMiddleware(http.HandlerFunc(handler.InvalidKey)))
	http.Handle("/extend", middlewares.AuthMiddleware(http.HandlerFunc(handler.ExtendKey)))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Fatal error while creating storage", err.Error())
	}
}