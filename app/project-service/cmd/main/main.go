package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpadapter "project-service/internal/adapters/http"
	postgresadapter "project-service/internal/adapters/postgres"
	appconfig "project-service/internal/config"
	"project-service/internal/core/usecase"
)

//go:embed static/index.html
//go:embed static/project.html
var staticFiles embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := appconfig.Load(resolveConfigPath())
	if err != nil {
		return err
	}

	log.Printf("loaded config: %s", cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := postgresadapter.NewPool(initCtx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := postgresadapter.InitSchema(initCtx, pool); err != nil {
		return err
	}

	projectRepo := postgresadapter.NewProjectRepository(pool)
	taskRepo := postgresadapter.NewTaskRepository(pool)
	commentRepo := postgresadapter.NewCommentRepository(pool)

	projectUseCase := usecase.NewProjectUseCase(projectRepo, taskRepo, cfg.AuthService.URL)
	taskUseCase := usecase.NewTaskUseCase(taskRepo)
	commentUseCase := usecase.NewCommentUseCase(commentRepo)

	projectHandler := httpadapter.NewProjectHandler(projectUseCase)
	taskHandler := httpadapter.NewTaskHandler(taskUseCase, projectUseCase)
	commentHandler := httpadapter.NewCommentHandler(commentUseCase, taskUseCase, projectUseCase)
	adminHandler := httpadapter.NewAdminHandler(projectUseCase, taskUseCase)

	authMW := httpadapter.AuthMiddleware(cfg.AuthService.URL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.Handle("/api/dashboard", authMW(http.HandlerFunc(projectHandler.Dashboard)))
	mux.Handle("/api/projects", authMW(http.HandlerFunc(projectHandler.CreateProject)))
	mux.Handle("/api/projects/", authMW(http.HandlerFunc(projectHandler.ProjectTasks)))

	mux.Handle("/api/my-tasks", authMW(http.HandlerFunc(taskHandler.GetMyTasks)))
	mux.Handle("/api/project-tasks", authMW(http.HandlerFunc(taskHandler.GetAllProjectTasks)))
	mux.Handle("/api/closed-project-tasks", authMW(http.HandlerFunc(taskHandler.GetClosedProjectTasks)))
	mux.Handle("/api/tasks", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskHandler.CreateTask(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/tasks/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/comments") {
			switch r.Method {
			case http.MethodGet:
				commentHandler.GetTaskComments(w, r)
			case http.MethodPost:
				commentHandler.CreateTaskComment(w, r)
			case http.MethodPut:
				commentHandler.UpdateTaskComment(w, r)
			case http.MethodDelete:
				commentHandler.DeleteTaskComment(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		} else if strings.Contains(path, "/status") {
			taskHandler.UpdateTaskStatus(w, r)
		} else if strings.Contains(path, "/close") {
			taskHandler.CloseTask(w, r)
		} else if strings.Contains(path, "/unassign") {
			taskHandler.UnassignTask(w, r)
		} else if strings.Contains(path, "/assign") {
			taskHandler.AssignTask(w, r)
		} else if r.Method == http.MethodDelete {
			taskHandler.DeleteTask(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/user-id", authMW(http.HandlerFunc(projectHandler.GetUserID)))
	mux.Handle("/api/project-members", authMW(http.HandlerFunc(projectHandler.GetProjectMembers)))
	mux.Handle("/api/project-members-details", authMW(http.HandlerFunc(projectHandler.GetProjectMembersWithDetails)))
	mux.Handle("/api/project-members/add", authMW(http.HandlerFunc(projectHandler.AddProjectMember)))
	mux.Handle("/api/project-manager/transfer", authMW(http.HandlerFunc(projectHandler.TransferProjectManager)))
	mux.Handle("/api/user-projects", authMW(http.HandlerFunc(projectHandler.GetUserProjects)))
	mux.Handle("/api/is-manager", authMW(http.HandlerFunc(projectHandler.IsUserManager)))
	mux.Handle("/api/project-info", authMW(http.HandlerFunc(projectHandler.GetProjectInfo)))

	mux.Handle("/api/admin/projects/get", authMW(httpadapter.AdminOnly(http.HandlerFunc(adminHandler.GetProject))))
	mux.Handle("/api/admin/projects", authMW(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			adminHandler.UpdateProject(w, r)
		case http.MethodDelete:
			adminHandler.DeleteProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))))
	mux.Handle("/api/admin/tasks/get", authMW(httpadapter.AdminOnly(http.HandlerFunc(adminHandler.GetTask))))
	mux.Handle("/api/admin/tasks", authMW(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			adminHandler.UpdateTask(w, r)
		case http.MethodDelete:
			adminHandler.DeleteTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))))
	mux.Handle("/api/admin/comments/get", authMW(httpadapter.AdminOnly(http.HandlerFunc(commentHandler.GetComment))))
	mux.Handle("/api/admin/comments", authMW(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			commentHandler.UpdateComment(w, r)
		case http.MethodDelete:
			commentHandler.DeleteComment(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))))
	mux.HandleFunc("/project/", serveProjectPage)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("starting project-service on %s", cfg.HTTP.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	return nil
}

func resolveConfigPath() string {
	defaultPath := appconfig.DefaultPath
	if v := os.Getenv("PROJECT_CONFIG_PATH"); v != "" {
		defaultPath = v
	}
	configPath := flag.String("config", defaultPath, "path to YAML config file")
	flag.Parse()
	return *configPath
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
