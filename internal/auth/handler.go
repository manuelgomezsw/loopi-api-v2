package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracer para el paquete auth (RF-AUTH-06).
var tracer = otel.Tracer(otelScope)

// Handler agrupa los handlers HTTP de autenticación.
type Handler struct {
	svc     Service
	repo    Repository
	metrics *Metrics
}

// NewHandler crea un nuevo Handler de autenticación con métricas OTel no-op por defecto.
func NewHandler(svc Service, repo Repository) *Handler {
	return &Handler{svc: svc, repo: repo, metrics: noopMetrics()}
}

// NewHandlerWithMetrics crea el Handler con métricas OTel configuradas.
func NewHandlerWithMetrics(svc Service, repo Repository, m *Metrics) *Handler {
	return &Handler{svc: svc, repo: repo, metrics: m}
}

// --- Tipos de request/response ---

type loginRequest struct {
	Usuario   string `json:"usuario"`
	Contrasena string `json:"contrasena"`
}

type loginResponse struct {
	Rol      string `json:"rol"`
	TiendaID *int   `json:"tienda_id"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
}

type cuentaBloqueadaResponse struct {
	Error          string `json:"error"`
	Mensaje        string `json:"mensaje"`
	BloqueadoHasta string `json:"bloqueado_hasta"`
}

type limpiarTokensResponse struct {
	Eliminados   int64  `json:"eliminados"`
	IniciadoEn   string `json:"iniciado_en"`
	CompletadoEn string `json:"completado_en"`
	Resultado    string `json:"resultado"`
}

// --- Handlers ---

// Login maneja POST /api/v1/auth/login (sin middleware JWT).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "auth.login",
		trace.WithAttributes(attribute.String(AttrHTTPRoute, "/api/v1/auth/login")),
	)
	defer span.End()

	inicio := time.Now()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Usuario == "" || req.Contrasena == "" {
		span.SetAttributes(attribute.String(AttrAuthResult, "bad_request"))
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "datos_invalidos",
			Mensaje: "Los campos usuario y contrasena son requeridos",
		})
		return
	}

	result, err := h.svc.Authenticate(req.Usuario, req.Contrasena)
	duracionMs := float64(time.Since(inicio).Milliseconds())

	if err != nil {
		var authResult string
		var status int
		var body interface{}

		switch err {
		case ErrCuentaBloqueada:
			authResult = ResultAccountLocked
			status = http.StatusLocked
			body = cuentaBloqueadaResponse{
				Error:          "cuenta_bloqueada",
				Mensaje:        "Demasiados intentos fallidos. Intente nuevamente en 5 minutos.",
				BloqueadoHasta: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
			}
		default:
			authResult = ResultInvalidCredentials
			status = http.StatusUnauthorized
			body = errorResponse{
				Error:   "credenciales_invalidas",
				Mensaje: "Usuario o contraseña incorrectos",
			}
		}

		span.SetAttributes(attribute.String(AttrAuthResult, authResult))
		h.metrics.RecordLogin(ctx, duracionMs, authResult)
		writeJSON(w, status, body)
		return
	}

	// Login exitoso.
	span.SetAttributes(
		attribute.String(AttrAuthResult, ResultSuccess),
		attribute.String(AttrUserRole, result.Rol),
	)
	h.metrics.RecordLogin(ctx, duracionMs, ResultSuccess)

	maxAge := int(time.Until(result.ExpiresAt).Seconds())

	// Cookie jwt: httpOnly, no accesible desde JS.
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    result.Token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Cookie XSRF-TOKEN: sin httpOnly — Angular la lee para el header X-XSRF-TOKEN.
	xsrfToken := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "XSRF-TOKEN",
		Value:    xsrfToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, loginResponse{
		Rol:      result.Rol,
		TiendaID: result.TiendaID,
	})
}

// Logout maneja POST /api/v1/auth/logout (requiere middleware JWT).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Validar CSRF: el header X-XSRF-TOKEN debe coincidir con la cookie XSRF-TOKEN.
	xsrfHeader := r.Header.Get("X-XSRF-TOKEN")
	xsrfCookie, err := r.Cookie("XSRF-TOKEN")
	if err != nil || xsrfHeader == "" || xsrfHeader != xsrfCookie.Value {
		writeJSON(w, http.StatusForbidden, errorResponse{
			Error:   "csrf_invalido",
			Mensaje: "Token CSRF requerido",
		})
		return
	}

	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Error:   "no_autenticado",
			Mensaje: "Sesión no válida o expirada",
		})
		return
	}

	if err := h.svc.RevocarToken(claims); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error:   "error_interno",
			Mensaje: "Error al revocar sesión",
		})
		return
	}

	// Expirar ambas cookies.
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "XSRF-TOKEN",
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Me maneja GET /api/v1/auth/me (requiere middleware JWT).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Error:   "no_autenticado",
			Mensaje: "Sesión no válida o expirada",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sub":       claims.Subject,
		"rol":       claims.Rol,
		"tienda_id": claims.TiendaID,
	})
}

// LimpiarTokensRevocados maneja POST /internal/jobs/limpiar_tokens_revocados.
// Solo acepta requests con el header X-CloudScheduler: true (Cloud Scheduler).
func (h *Handler) LimpiarTokensRevocados(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-CloudScheduler") != "true" {
		writeJSON(w, http.StatusForbidden, errorResponse{
			Error:   "no_autorizado",
			Mensaje: "Acceso restringido a jobs internos",
		})
		return
	}

	iniciadoEn := time.Now().UTC()
	eliminados, err := h.repo.LimpiarTokensExpirados()
	completadoEn := time.Now().UTC()

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, limpiarTokensResponse{
			Eliminados:   0,
			IniciadoEn:   iniciadoEn.Format(time.RFC3339),
			CompletadoEn: completadoEn.Format(time.RFC3339),
			Resultado:    "error",
		})
		return
	}

	writeJSON(w, http.StatusOK, limpiarTokensResponse{
		Eliminados:   eliminados,
		IniciadoEn:   iniciadoEn.Format(time.RFC3339),
		CompletadoEn: completadoEn.Format(time.RFC3339),
		Resultado:    "ok",
	})
}

// writeJSON escribe una respuesta JSON con el código de estado dado.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
