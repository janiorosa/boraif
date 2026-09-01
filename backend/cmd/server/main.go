package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"boraif/internal/ai"
	"boraif/internal/applications"
	"boraif/internal/auth"
	"boraif/internal/booklets"
	"boraif/internal/catalogs"
	"boraif/internal/config"
	"boraif/internal/db"
	"boraif/internal/disciplines"
	"boraif/internal/httpserver"
	"boraif/internal/images"
	"boraif/internal/pdf"
	"boraif/internal/questions"
	"boraif/internal/security"
	"boraif/internal/subjects"
	"boraif/internal/users"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "create-admin" {
		runCreateAdmin(os.Args[2:])
		return
	}
	runServe()
}

func runServe() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	log.Println("running database migrations...")
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer pool.Close()

	userRepo := users.NewRepository(pool)
	sessionStore := auth.NewSessionStore(pool)
	disciplineRepo := disciplines.NewRepository(pool)
	subjectRepo := subjects.NewRepository(pool)
	catalogRepo := catalogs.NewRepository(pool)
	questionRepo := questions.NewRepository(pool)
	imageRepo := images.NewRepository(pool)
	imageStorage := images.NewStorage(cfg.UploadsDir)
	applicationRepo := applications.NewRepository(pool)
	bookletRepo := booklets.NewRepository(pool)
	aiClient := ai.NewClient(cfg.OpenAIModel)
	pdfRepo := pdf.NewRepository(pool)
	pdfStorage := pdf.NewStorage(cfg.GeneratedDir)
	pdfService := &pdf.Service{
		Repo:         pdfRepo,
		Booklets:     bookletRepo,
		Applications: applicationRepo,
		Storage:      pdfStorage,
		ChromePath:   cfg.ChromePath,
	}

	authMiddleware := &auth.Middleware{
		Sessions:   sessionStore,
		Users:      userRepo,
		CookieName: cfg.SessionCookie,
	}

	router := httpserver.NewRouter(httpserver.Deps{
		AuthMiddleware: authMiddleware,
		Auth: &auth.Handlers{
			Users:               userRepo,
			Sessions:            sessionStore,
			CookieName:          cfg.SessionCookie,
			CookieSecure:        cfg.CookieSecure,
			APIKeyEncryptionKey: cfg.APIKeyEncryptionKey,
		},
		Users:       &users.Handlers{Repo: userRepo},
		Disciplines: &disciplines.Handlers{Repo: disciplineRepo},
		Subjects:    &subjects.Handlers{Repo: subjectRepo},
		Catalogs:    &catalogs.Handlers{Repo: catalogRepo},
		Questions:   &questions.Handlers{Repo: questionRepo, Statuses: catalogRepo},
		Images: &images.Handlers{
			Repo:         imageRepo,
			Storage:      imageStorage,
			Disciplines:  disciplineRepo,
			MaxSizeBytes: cfg.MaxUploadSizeBytes,
		},
		AI: &ai.Handlers{
			Questions:           questionRepo,
			Users:               userRepo,
			Client:              aiClient,
			APIKeyEncryptionKey: cfg.APIKeyEncryptionKey,
		},
		Applications: &applications.Handlers{Repo: applicationRepo},
		Booklets:     &booklets.Handlers{Repo: bookletRepo},
		PDF:          &pdf.Handlers{Service: pdfService, Booklets: bookletRepo, Storage: pdfStorage},
		UploadsDir:   cfg.UploadsDir,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("boraif backend listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runCreateAdmin é o mecanismo documentado no README para criar o primeiro
// administrador, sem embutir nenhuma credencial padrão nas migrations.
func runCreateAdmin(args []string) {
	fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
	name := fs.String("name", "", "nome do administrador")
	email := fs.String("email", "", "email de login")
	password := fs.String("password", "", "senha inicial")
	_ = fs.Parse(args)

	if *name == "" || *email == "" || *password == "" {
		fmt.Println("uso: server create-admin --name=\"...\" --email=\"...\" --password=\"...\"")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer pool.Close()

	hash, err := security.HashPassword(*password)
	if err != nil {
		log.Fatalf("could not hash password: %v", err)
	}

	repo := users.NewRepository(pool)
	id, err := repo.Create(ctx, users.User{
		Name:         *name,
		Email:        *email,
		PasswordHash: hash,
		Role:         users.RoleAdmin,
		Active:       true,
	})
	if err != nil {
		log.Fatalf("could not create admin: %v", err)
	}

	fmt.Printf("administrador criado com id=%d, email=%s\n", id, *email)
}
