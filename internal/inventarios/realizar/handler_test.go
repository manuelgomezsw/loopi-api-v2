package realizar

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"go.opentelemetry.io/otel"
)

// mockService implementa Service para tests
type mockService struct {
	registrarValorFn        func(*http.Request, int64, int64, *RegistrarValorRequest) (*RegistrarValorResponse, error)
	getDetallesInventarioFn func(*http.Request, int64) (*GetDetallesResponse, error)
}

func (m *mockService) RegistrarValor(r *http.Request, inventarioID, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error) {
	if m.registrarValorFn != nil {
		return m.registrarValorFn(r, inventarioID, itemID, req)
	}
	return nil, nil
}

func (m *mockService) GetDetallesInventario(r *http.Request, inventarioID int64) (*GetDetallesResponse, error) {
	if m.getDetallesInventarioFn != nil {
		return m.getDetallesInventarioFn(r, inventarioID)
	}
	return nil, nil
}

// TestPostRegistrarValor_Success valida registrar valor exitoso (200 OK)
func TestPostRegistrarValor_Success(t *testing.T) {
	mockSvc := &mockService{
		registrarValorFn: func(r *http.Request, invID int64, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error) {
			return &RegistrarValorResponse{
				Success:              true,
				ItemID:               itemID,
				ItemCodigo:           "ITEM-001",
				ItemDescripcion:      "Arroz",
				ValorEsperado:        20.0,
				ValorReal:            18.0,
				Diferencia:           -2.0,
				DiferenciaPorcentaje: -10.0,
				Unidad:               "bolsas",
			}, nil
		},
	}

	handler := &Handler{
		service: mockSvc,
		logger:  slog.Default(),
		tracer:  otel.Tracer("test"),
		metrics: noopMetrics(),
	}

	req := RegistrarValorRequest{ValorReal: 18.0}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/api/v1/inventarios/1/items/100/valor", bytes.NewReader(body))
	httpReq.SetPathValue("inventario_id", "1")
	httpReq.SetPathValue("item_id", "100")
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), auth.ContextKeyClaims, &auth.Claims{Rol: "barista"}))

	w := httptest.NewRecorder()
	handler.PostRegistrarValor(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp RegistrarValorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ValorReal != 18.0 {
		t.Errorf("expected valor_real 18.0, got %f", resp.ValorReal)
	}
}

// TestPostRegistrarValor_ValorNegativo valida rechazo de valor < 0 (400 Bad Request)
func TestPostRegistrarValor_ValorNegativo(t *testing.T) {
	mockSvc := &mockService{}
	handler := &Handler{
		service: mockSvc,
		logger:  slog.Default(),
		tracer:  otel.Tracer("test"),
		metrics: noopMetrics(),
	}

	req := RegistrarValorRequest{ValorReal: -5.0}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/api/v1/inventarios/1/items/100/valor", bytes.NewReader(body))
	httpReq.SetPathValue("inventario_id", "1")
	httpReq.SetPathValue("item_id", "100")
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), auth.ContextKeyClaims, &auth.Claims{Rol: "barista"}))

	w := httptest.NewRecorder()
	handler.PostRegistrarValor(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestPostRegistrarValor_ConteoNoProgreso valida error cuando conteo no está en progreso (409 Conflict)
func TestPostRegistrarValor_ConteoNoProgreso(t *testing.T) {
	mockSvc := &mockService{
		registrarValorFn: func(r *http.Request, invID int64, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error) {
			return nil, &Error{Code: "CONTEO_NO_EN_PROGRESO", Message: "El conteo no está en estado 'en_progreso'"}
		},
	}

	handler := &Handler{
		service: mockSvc,
		logger:  slog.Default(),
		tracer:  otel.Tracer("test"),
		metrics: noopMetrics(),
	}

	req := RegistrarValorRequest{ValorReal: 18.0}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/api/v1/inventarios/1/items/100/valor", bytes.NewReader(body))
	httpReq.SetPathValue("inventario_id", "1")
	httpReq.SetPathValue("item_id", "100")
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), auth.ContextKeyClaims, &auth.Claims{Rol: "barista"}))

	w := httptest.NewRecorder()
	handler.PostRegistrarValor(w, httpReq)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

// TestPostRegistrarValor_ItemNoExiste valida error cuando item no existe (404 Not Found)
func TestPostRegistrarValor_ItemNoExiste(t *testing.T) {
	mockSvc := &mockService{
		registrarValorFn: func(r *http.Request, invID int64, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error) {
			return nil, &Error{Code: "ITEM_NO_ENCONTRADO", Message: "Item no encontrado"}
		},
	}

	handler := &Handler{
		service: mockSvc,
		logger:  slog.Default(),
		tracer:  otel.Tracer("test"),
		metrics: noopMetrics(),
	}

	req := RegistrarValorRequest{ValorReal: 18.0}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/api/v1/inventarios/1/items/999/valor", bytes.NewReader(body))
	httpReq.SetPathValue("inventario_id", "1")
	httpReq.SetPathValue("item_id", "999")
	httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), auth.ContextKeyClaims, &auth.Claims{Rol: "barista"}))

	w := httptest.NewRecorder()
	handler.PostRegistrarValor(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
