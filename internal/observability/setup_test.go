package observability_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/observability"
)

func TestSetup_ModoNoOp_CuandoEndpointVacio(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	shutdown, err := observability.Setup(context.Background())
	if err != nil {
		t.Fatalf("Setup no debe fallar en modo no-op: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown no debe ser nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown no debe fallar en modo no-op: %v", err)
	}
}

func TestSetup_ShutdownExitoso_ConReceptorOTLP(t *testing.T) {
	received := make(chan struct{}, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", srv.URL)
	t.Setenv("OTEL_SERVICE_NAME", "loopi-api-test")
	t.Setenv("APP_VERSION", "0.0.1")
	t.Setenv("ENV", "stage")

	shutdown, err := observability.Setup(context.Background())
	if err != nil {
		t.Fatalf("Setup falló con endpoint válido: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown no debe ser nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown retornó error inesperado: %v", err)
	}
}
