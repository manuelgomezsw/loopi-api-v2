package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"

	"github.com/manuelgomezsw/loopi-api-v2/config"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"github.com/manuelgomezsw/loopi-api-v2/internal/jobs"
)

func main() {
	// Cargar configuración desde variables de entorno.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error al cargar configuración: %v", err)
	}

	// Conectar a Cloud SQL (MySQL).
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN es obligatorio")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("error al abrir conexión a BD: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error al conectar a BD: %v", err)
	}

	// Repositorio, servicio y handler de autenticación.
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(cfg, authRepo)
	authHandler := auth.NewHandler(authSvc, authRepo)
	jwtMiddleware := auth.JWTMiddleware(cfg.JWTSecret, authRepo)

	// Router principal.
	mux := http.NewServeMux()

	// Health check (sin autenticación).
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Rutas de autenticación — sin middleware JWT.
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Rutas de autenticación — con middleware JWT.
	// Los claims validados quedan en el contexto del request para RBAC (RF-AUTH-05.2).
	mux.Handle("POST /api/v1/auth/logout", jwtMiddleware(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("GET /api/v1/auth/me", jwtMiddleware(http.HandlerFunc(authHandler.Me)))

	// Job de limpieza — sin middleware JWT, con validación de header X-CloudScheduler.
	mux.HandleFunc("POST /internal/jobs/limpiar_tokens_revocados",
		jobs.LimpiarTokensHandler(authRepo))

	// Configurar CORS según ambiente.
	// NUNCA usar wildcard "*" con AllowCredentials: true (requiere cookies).
	allowedOrigin := corsOrigin()
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-XSRF-TOKEN"},
		AllowCredentials: true,
	}).Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("servidor iniciado en :%s (CORS origin: %s)", port, allowedOrigin)
	if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
		log.Fatalf("error al iniciar servidor: %v", err)
	}
}

// corsOrigin devuelve el origen CORS permitido según la variable de entorno ENV.
// Nunca devuelve wildcard "*" porque se usan cookies con credenciales.
func corsOrigin() string {
	switch os.Getenv("ENV") {
	case "prod":
		return "https://app.loopi.com"
	case "stage":
		return "https://app.stage.loopi.com"
	default: // dev / local
		return "http://localhost:4200"
	}
}
