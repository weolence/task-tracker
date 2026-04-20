package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"auth-service/internal/controller"
	"auth-service/internal/handler"
	"auth-service/internal/model"
	"auth-service/internal/repository"
)

//go:embed static/index.html
var staticFiles embed.FS

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userRepo, err := repository.NewUserRepository(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	authController, err := controller.NewAuthController(*userRepo, []byte(jwtSecret))
	if err != nil {
		log.Fatalf("failed to create auth controller: %v", err)
	}

	authHandler := handler.NewAuthHandler(authController)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/validate-token", authHandler.ValidateToken)
	mux.HandleFunc("/user-info", authHandler.GetUserInfo)

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: mux,
	}

	go func() {
		log.Printf("starting auth-service on :%s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	stdinInfo, err := os.Stdin.Stat()
	interactive := err == nil && (stdinInfo.Mode()&os.ModeCharDevice) != 0

	consoleDone := make(chan struct{})
	if interactive {
		go runInteractiveConsole(authController, consoleDone)
	} else {
		log.Println("stdin is not a terminal; interactive console disabled")
	}

	select {
	case <-shutdownCh:
		log.Println("shutdown signal received")
	case <-consoleDone:
		log.Println("console requested shutdown")
	}

	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func runInteractiveConsole(authController *controller.AuthController, done chan<- struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Interactive console started.")
	fmt.Println("Commands: register | delete | help | exit")

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
			runInteractiveRegister(authController)
		case "delete":
			runInteractiveDelete(authController)
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

func runInteractiveRegister(authController *controller.AuthController) {
	reader := bufio.NewReader(os.Stdin)
	email := askValue(reader, "Email")
	password := askValue(reader, "Password")
	name := askValue(reader, "Name")
	surname := askValue(reader, "Surname")

	user := model.User{
		Email:    strings.TrimSpace(email),
		Password: password,
		Name:     strings.TrimSpace(name),
		Surname:  strings.TrimSpace(surname),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := authController.Register(ctx, user); err != nil {
		fmt.Printf("register failed: %v\n", err)
		return
	}

	fmt.Println("user registered successfully")
}

func runInteractiveDelete(authController *controller.AuthController) {
	reader := bufio.NewReader(os.Stdin)
	email := askValue(reader, "Email to delete")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := authController.DeleteUser(ctx, strings.TrimSpace(email)); err != nil {
		fmt.Printf("delete failed: %v\n", err)
		return
	}

	fmt.Println("user deleted successfully")
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  register   - create a user interactively")
	fmt.Println("  delete     - delete user by email")
	fmt.Println("  help       - show this help")
	fmt.Println("  exit, quit - stop server and quit")
}

func askValue(reader *bufio.Reader, prompt string) string {
	fmt.Printf("%s: ", prompt)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
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
