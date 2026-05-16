package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	authsvcv1 "auth-service/api/proto/authsvcv1"
	grpcadapter "auth-service/internal/adapters/grpc"
	httpadapter "auth-service/internal/adapters/http"
	postgresadapter "auth-service/internal/adapters/postgres"
	appconfig "auth-service/internal/config"
	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
)

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

	pool, err := postgresadapter.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer pool.Close()

	userRepo := postgresadapter.NewUserRepository(pool)
	auth := usecase.NewAuthUseCase(userRepo, []byte(cfg.JWT.Secret))

	authHandler := httpadapter.NewAuthHandler(auth)
	adminHandler := httpadapter.NewAdminHandler(auth, cfg.Admin.ProjectServiceURL)

	serveHTML := func(filename string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := os.ReadFile(filepath.Join(cfg.StaticDir, filename))
			if err != nil {
				http.Error(w, "failed to load page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(page)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveHTML("index.html")(w, r)
	})
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" {
			http.NotFound(w, r)
			return
		}
		serveHTML("admin.html")(w, r)
	})
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/validate-token", authHandler.ValidateToken)

	authMW := httpadapter.AuthMiddleware([]byte(cfg.JWT.Secret))
	mux.HandleFunc("/user-info", authHandler.GetUserInfo)

	adminAuth := authMW
	mux.Handle("/admin/api/users/get", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(adminHandler.GetUser))))
	mux.Handle("/admin/api/users", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			adminHandler.UpdateUser(w, r)
		case http.MethodDelete:
			adminHandler.DeleteUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))))
	mux.Handle("/admin/api/comments/get", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyComment(w, r, "/api/admin/comments/get")
	}))))
	mux.Handle("/admin/api/comments", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyComment(w, r, "/api/admin/comments")
	}))))
	mux.Handle("/admin/api/projects/get", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyProject(w, r, "/api/admin/projects/get")
	}))))
	mux.Handle("/admin/api/projects", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyProject(w, r, "/api/admin/projects")
	}))))
	mux.Handle("/admin/api/tasks/get", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyTask(w, r, "/api/admin/tasks/get")
	}))))
	mux.Handle("/admin/api/tasks", adminAuth(httpadapter.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminHandler.ProxyTask(w, r, "/api/admin/tasks")
	}))))

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("starting auth-service HTTP on %s", cfg.HTTP.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	grpcServer := grpc.NewServer()
	authsvcv1.RegisterAuthServiceServer(grpcServer, grpcadapter.NewAuthServer(auth))
	grpcListener, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		return fmt.Errorf("gRPC listen: %w", err)
	}
	go func() {
		log.Printf("starting auth-service gRPC on %s", cfg.GRPC.Addr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	stdinInfo, err := os.Stdin.Stat()
	interactive := err == nil && (stdinInfo.Mode()&os.ModeCharDevice) != 0

	consoleDone := make(chan struct{})
	if interactive {
		go runInteractiveConsole(auth, consoleDone)
	} else {
		log.Println("stdin is not a terminal; interactive console disabled")
	}

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case <-consoleDone:
		log.Println("console requested shutdown")
	}

	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	return nil
}

func resolveConfigPath() string {
	defaultPath := appconfig.DefaultPath
	if v := os.Getenv("AUTH_CONFIG_PATH"); v != "" {
		defaultPath = v
	}
	configPath := flag.String("config", defaultPath, "path to YAML config file")
	flag.Parse()
	return *configPath
}

func runInteractiveConsole(auth *usecase.AuthUseCase, done chan<- struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Interactive console started.")
	fmt.Println("Commands: register | delete | set-role | help | exit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println("\nstdin closed, shutting down")
			close(done)
			return
		}

		line := strings.TrimSpace(scanner.Text())
		switch strings.ToLower(line) {
		case "register":
			runInteractiveRegister(auth)
		case "delete":
			runInteractiveDelete(auth)
		case "set-role":
			runInteractiveSetRole(auth)
		case "help":
			printHelp()
		case "exit", "quit":
			fmt.Println("Shutting down...")
			close(done)
			return
		case "":
			continue
		default:
			fmt.Println("unknown command. type help")
		}
	}
}

func runInteractiveRegister(auth *usecase.AuthUseCase) {
	reader := bufio.NewReader(os.Stdin)
	email := askValue(reader, "Email")
	password := askValue(reader, "Password")
	titler := cases.Title(language.English, cases.Compact)
	name := titler.String(askValue(reader, "Name"))
	surname := titler.String(askValue(reader, "Surname"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := auth.Register(ctx, domain.User{
		Email:    strings.TrimSpace(email),
		Password: password,
		Name:     strings.TrimSpace(name),
		Surname:  strings.TrimSpace(surname),
	}); err != nil {
		fmt.Printf("register failed: %v\n", err)
		return
	}

	fmt.Println("user registered successfully")
}

func runInteractiveDelete(auth *usecase.AuthUseCase) {
	reader := bufio.NewReader(os.Stdin)
	email := askValue(reader, "Email to delete")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := auth.DeleteUser(ctx, strings.TrimSpace(email)); err != nil {
		fmt.Printf("delete failed: %v\n", err)
		return
	}

	fmt.Println("user deleted successfully")
}

func runInteractiveSetRole(auth *usecase.AuthUseCase) {
	reader := bufio.NewReader(os.Stdin)
	email := askValue(reader, "Email")
	role := strings.ToLower(strings.TrimSpace(askValue(reader, "Role (user/admin)")))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := auth.ChangeRole(ctx, strings.TrimSpace(email), role); err != nil {
		fmt.Printf("set role failed: %v\n", err)
		return
	}

	fmt.Println("user role updated successfully")
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  register   - create a user interactively")
	fmt.Println("  delete     - delete user by email")
	fmt.Println("  set-role   - set user role (user/admin) by email")
	fmt.Println("  help       - show this help")
	fmt.Println("  exit, quit - stop server and quit")
}

func askValue(reader *bufio.Reader, prompt string) string {
	fmt.Printf("%s: ", prompt)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}
