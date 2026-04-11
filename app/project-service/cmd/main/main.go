package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-service/internal/controller"
	"project-service/internal/handler"
	"project-service/internal/middleware"
	"project-service/internal/repository"
)

//go:embed static/index.html
var staticFiles embed.FS

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectRepo, err := repository.NewProjectRepository(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	projectController := controller.NewProjectController(*projectRepo)
	projectHandler := handler.NewProjectHandler(projectController)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)

	authMiddleware := middleware.AuthMiddleware()
	mux.Handle("/api/dashboard", authMiddleware(http.HandlerFunc(projectHandler.Dashboard)))
	mux.Handle("/api/projects", authMiddleware(http.HandlerFunc(projectHandler.CreateProject)))

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8081"
	}

	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: mux,
	}

	go func() {
		log.Printf("starting project-service on :%s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	<-shutdownCh
	log.Println("shutdown signal received")

	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	page, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "failed to load page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}
