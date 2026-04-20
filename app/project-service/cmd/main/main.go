package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"project-service/internal/controller"
	"project-service/internal/handler"
	"project-service/internal/middleware"
	"project-service/internal/repository"
)

//go:embed static/index.html
//go:embed static/project.html
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

	taskRepo, err := repository.NewTaskRepository(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to task database: %v", err)
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://localhost:8080"
	}

	projectController := controller.NewProjectController(*projectRepo, *taskRepo, authServiceURL)

	taskController := controller.NewTaskController(*taskRepo)

	projectHandler := handler.NewProjectHandler(projectController)
	taskHandler := handler.NewTaskHandler(taskController, projectController)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)

	authMiddleware := middleware.AuthMiddleware()
	mux.Handle("/api/dashboard", authMiddleware(http.HandlerFunc(projectHandler.Dashboard)))
	mux.Handle("/api/projects", authMiddleware(http.HandlerFunc(projectHandler.CreateProject)))
	mux.Handle("/api/projects/", authMiddleware(http.HandlerFunc(projectHandler.ProjectTasks)))

	// Task endpoints
	mux.Handle("/api/my-tasks", authMiddleware(http.HandlerFunc(taskHandler.GetMyTasks)))
	mux.Handle("/api/project-tasks", authMiddleware(http.HandlerFunc(taskHandler.GetAllProjectTasks)))
	mux.Handle("/api/tasks", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskHandler.CreateTask(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/tasks/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/status") {
			taskHandler.UpdateTaskStatus(w, r)
		} else if strings.Contains(path, "/assign") {
			taskHandler.AssignTask(w, r)
		} else if r.Method == http.MethodDelete {
			taskHandler.DeleteTask(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/user-id", authMiddleware(http.HandlerFunc(projectHandler.GetUserID)))
	mux.Handle("/api/project-members", authMiddleware(http.HandlerFunc(projectHandler.GetProjectMembers)))
	mux.Handle("/api/project-members-details", authMiddleware(http.HandlerFunc(projectHandler.GetProjectMembersWithDetails)))
	mux.Handle("/api/user-projects", authMiddleware(http.HandlerFunc(projectHandler.GetUserProjects)))
	mux.Handle("/api/is-manager", authMiddleware(http.HandlerFunc(projectHandler.IsUserManager)))
	mux.Handle("/api/project-info", authMiddleware(http.HandlerFunc(projectHandler.GetProjectInfo)))
	mux.HandleFunc("/project/", serveProjectPage)

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

func serveProjectPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/project/" || !strings.HasPrefix(r.URL.Path, "/project/") {
		http.NotFound(w, r)
		return
	}

	page, err := staticFiles.ReadFile("static/project.html")
	if err != nil {
		http.Error(w, "failed to load page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}
