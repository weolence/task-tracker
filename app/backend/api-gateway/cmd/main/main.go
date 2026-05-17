package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcclients "api-gateway/internal/adapters/grpc"
	httpadapter "api-gateway/internal/adapters/http"
	appconfig "api-gateway/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := flag.String("config", appconfig.DefaultPath, "path to YAML config file")
	flag.Parse()

	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		return err
	}
	log.Printf("api-gateway starting on %s | auth-grpc=%s project-grpc=%s",
		cfg.HTTP.Addr, cfg.AuthService.GRPCAddr, cfg.ProjectService.GRPCAddr)

	clients, err := grpcclients.New(cfg.AuthService.GRPCAddr, cfg.ProjectService.GRPCAddr)
	if err != nil {
		return err
	}
	defer clients.Close()

	authH := httpadapter.NewAuthHandler(clients.Auth)
	projH := httpadapter.NewProjectHandler(clients.Project)
	authMW := httpadapter.AuthMiddleware(clients.Auth)
	adminOnly := httpadapter.AdminOnly

	mux := http.NewServeMux()

	// ── Public auth routes ────────────────────────────────────────────────────
	mux.HandleFunc("/api/auth/register", authH.Register)
	mux.HandleFunc("/api/auth/login", authH.Login)

	// ── Protected auth routes ─────────────────────────────────────────────────
	mux.Handle("/api/user-id", authMW(http.HandlerFunc(authH.GetCurrentUserID)))
	mux.Handle("/api/auth/user-info", authMW(http.HandlerFunc(authH.GetUserInfo)))

	// ── Admin: user management ────────────────────────────────────────────────
	mux.Handle("/api/admin/users/get", authMW(adminOnly(http.HandlerFunc(authH.AdminGetUser))))
	mux.Handle("/api/admin/users", authMW(adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			authH.AdminUpdateUser(w, r)
		case http.MethodDelete:
			authH.AdminDeleteUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	// ── Dashboard ─────────────────────────────────────────────────────────────
	mux.Handle("/api/dashboard", authMW(http.HandlerFunc(projH.Dashboard)))

	// ── Projects ──────────────────────────────────────────────────────────────
	mux.Handle("/api/projects", authMW(http.HandlerFunc(projH.CreateProject)))
	mux.Handle("/api/projects/", authMW(http.HandlerFunc(projH.GetProjectTasks)))
	mux.Handle("/api/project-info", authMW(http.HandlerFunc(projH.GetProjectInfo)))
	mux.Handle("/api/project-members", authMW(http.HandlerFunc(projH.GetProjectMembers)))
	mux.Handle("/api/project-members-details", authMW(http.HandlerFunc(projH.GetProjectMembersDetails)))
	mux.Handle("/api/project-members/add", authMW(http.HandlerFunc(projH.AddProjectMember)))
	mux.Handle("/api/project-members/remove", authMW(http.HandlerFunc(projH.RemoveProjectMember)))
	mux.Handle("/api/project/leave", authMW(http.HandlerFunc(projH.LeaveProject)))
	mux.Handle("/api/project/delete", authMW(http.HandlerFunc(projH.DeleteProject)))
	mux.Handle("/api/project-manager/transfer", authMW(http.HandlerFunc(projH.TransferProjectManager)))
	mux.Handle("/api/is-manager", authMW(http.HandlerFunc(projH.IsUserManager)))
	mux.Handle("/api/user-projects", authMW(http.HandlerFunc(projH.GetUserProjects)))

	// ── Tasks & Comments (shared router) ──────────────────────────────────────
	mux.Handle("/api/tasks", authMW(http.HandlerFunc(projH.TasksRouter)))
	mux.Handle("/api/tasks/", authMW(http.HandlerFunc(projH.TasksRouter)))

	// ── Admin: project/task/comment CRUD ─────────────────────────────────────
	mux.Handle("/api/admin/projects/get", authMW(adminOnly(http.HandlerFunc(projH.AdminGetProject))))
	mux.Handle("/api/admin/projects", authMW(adminOnly(http.HandlerFunc(projH.AdminProjectCRUD))))
	mux.Handle("/api/admin/tasks/get", authMW(adminOnly(http.HandlerFunc(projH.AdminGetTask))))
	mux.Handle("/api/admin/tasks", authMW(adminOnly(http.HandlerFunc(projH.AdminTaskCRUD))))
	mux.Handle("/api/admin/comments/get", authMW(adminOnly(http.HandlerFunc(projH.AdminGetComment))))
	mux.Handle("/api/admin/comments", authMW(adminOnly(http.HandlerFunc(projH.AdminCommentCRUD))))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	return nil
}

func init() {
	// make log output consistent with other services
	log.SetFlags(log.LstdFlags)
	if tz := os.Getenv("TZ"); tz == "" {
		os.Setenv("TZ", "UTC")
	}
}
