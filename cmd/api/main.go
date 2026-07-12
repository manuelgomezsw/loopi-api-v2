package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	otelsql "github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/manuelgomezsw/loopi-api-v2/config"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
	"github.com/manuelgomezsw/loopi-api-v2/internal/empleados"
	"github.com/manuelgomezsw/loopi-api-v2/internal/items"
	"github.com/manuelgomezsw/loopi-api-v2/internal/jobs"
	"github.com/manuelgomezsw/loopi-api-v2/internal/observability"
	"github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
	"github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

func main() {
	ctx := context.Background()

	// Bootstrap OTel — no-op si OTEL_EXPORTER_OTLP_ENDPOINT está vacía.
	otelShutdown, err := observability.Setup(ctx)
	if err != nil {
		log.Fatalf("error al inicializar observabilidad: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}()

	// Cargar configuración desde variables de entorno.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error al cargar configuración: %v", err)
	}

	// Conectar a Cloud SQL (MySQL) con instrumentación automática de queries OTel.
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN es obligatorio")
	}
	db, err := otelsql.Open("mysql", dsn,
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{Ping: false}),
	)
	if err != nil {
		log.Fatalf("error al abrir conexión a BD: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error al conectar a BD: %v", err)
	}

	// Inicializar métricas OTel del dominio auth (MeterProvider global ya configurado).
	m, err := auth.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de auth: %v", err)
	}

	// Repositorio, servicio y handler de autenticación.
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(cfg, authRepo)
	authHandler := auth.NewHandlerWithMetrics(authSvc, authRepo, m)
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

	const cacheTTL = 24 * time.Hour

	// Módulo de tiendas.
	tiendasMetrics, err := tiendas.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de tiendas: %v", err)
	}
	rawTiendasRepo := tiendas.NewRepository(db)
	tiendasRepo, err := tiendas.NewCachedRepository(rawTiendasRepo, cacheTTL)
	if err != nil {
		log.Fatalf("cache tiendas: %v", err)
	}
	tiendasSvc := tiendas.NewService(tiendasRepo)
	tiendasHandler := tiendas.NewTiendaHandlerWithMetrics(tiendasSvc, tiendasMetrics)
	tiendasHandler.RegisterRoutes(mux, jwtMiddleware)

	// Módulo de empleados.
	empleadosMetrics, err := empleados.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de empleados: %v", err)
	}
	rawEmpleadosRepo := empleados.NewRepository(db)
	empleadosRepo, err := empleados.NewCachedRepository(rawEmpleadosRepo, cacheTTL)
	if err != nil {
		log.Fatalf("cache empleados: %v", err)
	}
	empleadosSvc := empleados.NewService(empleadosRepo)
	empleadosHandler := empleados.NewHandlerWithMetrics(empleadosSvc, empleadosMetrics)
	empleadosHandler.RegisterRoutes(mux, jwtMiddleware)

	// Módulo de unidades de medida.
	umMetrics, err := unidades_medida.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de unidades_medida: %v", err)
	}
	rawUmRepo := unidades_medida.NewRepository(db)
	umRepo, err := unidades_medida.NewCachedRepository(rawUmRepo, cacheTTL)
	if err != nil {
		log.Fatalf("cache unidades_medida: %v", err)
	}
	umSvc := unidades_medida.NewService(umRepo)
	umHandler := unidades_medida.NewHandlerWithMetrics(umSvc, umMetrics)
	umHandler.RegisterRoutes(mux, jwtMiddleware)

	// Módulo de categorías del catálogo.
	catMetrics, err := categorias.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de categorias: %v", err)
	}
	rawCatRepo := categorias.NewRepository(db)
	catRepo, err := categorias.NewCachedRepository(rawCatRepo, cacheTTL)
	if err != nil {
		log.Fatalf("cache categorias: %v", err)
	}
	catSvc := categorias.NewService(catRepo)
	catHandler := categorias.NewHandlerWithMetrics(catSvc, catMetrics)
	catHandler.RegisterRoutes(mux, jwtMiddleware)

	// Módulo de proveedores del catálogo.
	provMetrics, err := proveedores.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de proveedores: %v", err)
	}
	rawProvRepo := proveedores.NewRepository(db)
	provRepo, err := proveedores.NewCachedRepository(rawProvRepo, cacheTTL)
	if err != nil {
		log.Fatalf("cache proveedores: %v", err)
	}
	provSvc := proveedores.NewService(provRepo)
	provHandler := proveedores.NewHandlerWithMetrics(provSvc, provMetrics)
	provHandler.RegisterRoutes(mux, jwtMiddleware)

	// Módulo de items del catálogo.
	itemsMetrics, err := items.NewMetrics()
	if err != nil {
		log.Fatalf("error al inicializar métricas de items: %v", err)
	}
	const itemsCacheTTL = 5 * time.Minute
	rawItemsRepo := items.NewRepository(db)
	itemsRepo, err := items.NewCachedRepository(rawItemsRepo, itemsCacheTTL)
	if err != nil {
		log.Fatalf("cache items: %v", err)
	}
	itemsSvc := items.NewService(itemsRepo)
	itemsHandler := items.NewHandlerWithMetrics(itemsSvc, itemsMetrics)
	itemsHandler.RegisterRoutes(mux, jwtMiddleware)

	// Job de limpieza — sin middleware JWT, con validación de header X-CloudScheduler.
	mux.HandleFunc("POST /internal/jobs/limpiar_tokens_revocados",
		jobs.LimpiarTokensHandler(authRepo))

	// Configurar CORS según ambiente.
	// NUNCA usar wildcard "*" con AllowCredentials: true (requiere cookies).
	allowedOrigin := corsOrigin()
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
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
