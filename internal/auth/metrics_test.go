package auth_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

// setNoopProvider instala el provider noop de OTel como proveedor global durante el test.
// Restaura el proveedor anterior al finalizar (t.Cleanup).
func setNoopProvider(t *testing.T) {
	t.Helper()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(prev) })
}

// --- Tests: NewMetrics ---

func TestNewMetrics_ConProviderNoop_NoDevuelveError(t *testing.T) {
	setNoopProvider(t)

	m, err := auth.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics no debería fallar con provider noop: %v", err)
	}
	if m == nil {
		t.Fatal("NewMetrics no debe devolver nil")
	}
}

// --- Tests: RecordBlacklistCheck ---

func TestRecordBlacklistCheck_NoHacePanic(t *testing.T) {
	setNoopProvider(t)

	m, err := auth.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics falló: %v", err)
	}

	// Verificar que llamar RecordBlacklistCheck no hace panic ni error.
	m.RecordBlacklistCheck(context.Background(), 2.5)
}
