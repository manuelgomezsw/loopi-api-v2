package inventarios

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

type MockService struct {
	inventarios map[int64]*Inventario
}

func NewMockService() *MockService {
	return &MockService{
		inventarios: make(map[int64]*Inventario),
	}
}

func (m *MockService) Iniciar(ctx context.Context, req *CreateInventarioReq, userID int64, role string, userTiendaID *int64) (*InventarioResp, error) {
	inv := &Inventario{
		ID:            1,
		TiendaID:      req.TiendaID,
		Tipo:          req.Tipo,
		Horario:       req.Horario,
		Estado:        EstadoEnProgreso,
		ResponsableID: userID,
		Items:         []DetalleInventario{},
	}
	m.inventarios[1] = inv

	return &InventarioResp{
		ID:       inv.ID,
		TiendaID: inv.TiendaID,
		Tipo:     inv.Tipo,
		Horario:  inv.Horario,
		Estado:   inv.Estado,
		Items:    []ItemDetailResp{},
	}, nil
}

func (m *MockService) RegistrarValor(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID int64) (*ItemDetailResp, error) {
	// Validar valor_real >= 0 (BUG-020)
	if valorReal < 0 {
		return nil, NewError("valor_invalido", "La cantidad no puede ser negativa. Ingrese un valor mayor o igual a 0.")
	}
	return &ItemDetailResp{
		ID:       1,
		ItemID:   itemID,
		ValorReal: &valorReal,
	}, nil
}

func (m *MockService) Confirmar(ctx context.Context, inventarioID int64, userID int64) (*InventarioResp, error) {
	inv := &InventarioResp{ID: inventarioID, Estado: EstadoCompletado}
	return inv, nil
}

func (m *MockService) Listar(ctx context.Context, filtros *FiltrosInventario, userID int64, role string, userTiendaID *int64) (*HistorialResp, error) {
	return &HistorialResp{Inventarios: []InventarioResp{}, Total: 0}, nil
}

func (m *MockService) Buscar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) (*InventarioResp, error) {
	return &InventarioResp{ID: inventarioID}, nil
}

func (m *MockService) Modificar(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID, roleID int64) (*ItemDetailResp, error) {
	return &ItemDetailResp{}, nil
}

func (m *MockService) Eliminar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) error {
	return nil
}

func (m *MockService) Sugerir(ctx context.Context) (*SugerenciaResp, error) {
	return &SugerenciaResp{Tipo: TipoDiario, Horario: HorarioApertura}, nil
}

func (m *MockService) ValidarTipo(tipo Tipo) error {
	return nil
}

func (m *MockService) ValidarHorario(horario *Horario, tipo Tipo) error {
	return nil
}

func (m *MockService) GetEstadoInventarioActivo(ctx context.Context, tiendaID int64) (*InventarioResp, error) {
	return nil, nil
}

// Helper function to create context with auth claims
func contextWithClaims() context.Context {
	tiendaID := 1
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "123",
		},
		Rol:      "admin",
		TiendaID: &tiendaID,
	}
	return context.WithValue(context.Background(), auth.ContextKeyClaims, claims)
}

// Tests
func TestGetSugerencia(t *testing.T) {
	handler := NewHandler(NewMockService())
	req := httptest.NewRequest("GET", "/api/v1/inventarios/sugerencia", nil)
	req = req.WithContext(contextWithClaims())
	w := httptest.NewRecorder()

	handler.GetSugerencia(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetSugerencia() status = %d, want 200", w.Code)
	}

	var resp SugerenciaResp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("GetSugerencia() decode error: %v", err)
	}

	if resp.Tipo == "" {
		t.Errorf("GetSugerencia() tipo vacío")
	}

	if resp.Tipo != TipoDiario {
		t.Errorf("GetSugerencia() tipo = %v, want %v", resp.Tipo, TipoDiario)
	}
}

func TestPostInventario(t *testing.T) {
	handler := NewHandler(NewMockService())

	reqBody := CreateInventarioReq{
		TiendaID: 1,
		Tipo:     TipoDiario,
		Horario:  ptrHorario(HorarioApertura),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/inventarios", bytes.NewReader(body))
	req = req.WithContext(contextWithClaims())
	w := httptest.NewRecorder()

	handler.PostInventario(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("PostInventario() status = %d, want 201", w.Code)
	}

	var resp InventarioResp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("PostInventario() decode error: %v", err)
	}

	if resp.Estado != EstadoEnProgreso {
		t.Errorf("PostInventario() estado = %v, want %v", resp.Estado, EstadoEnProgreso)
	}

	if resp.TiendaID != 1 {
		t.Errorf("PostInventario() tienda_id = %d, want 1", resp.TiendaID)
	}
}

func TestGetHistorial(t *testing.T) {
	handler := NewHandler(NewMockService())
	req := httptest.NewRequest("GET", "/api/v1/inventarios", nil)
	req = req.WithContext(contextWithClaims())
	w := httptest.NewRecorder()

	handler.GetHistorial(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetHistorial() status = %d, want 200", w.Code)
	}

	var resp HistorialResp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("GetHistorial() decode error: %v", err)
	}

	if resp.Inventarios == nil {
		t.Errorf("GetHistorial() inventarios es nil")
	}
}

// TestPatchItemValor_RejectedNegative verifica que valores negativos retornan 400 (T174 BUG-020)
func TestPatchItemValor_RejectedNegative(t *testing.T) {
	mockSvc := NewMockService()
	handler := NewHandler(mockSvc)

	reqBody := PatchItemValorReq{
		ValorReal: -500.0, // Valor negativo - debe ser rechazado
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/inventarios/1/items/1", bytes.NewReader(body))
	req = req.WithContext(contextWithClaims())

	// Simular ruta con path values
	req.SetPathValue("id", "1")
	req.SetPathValue("item_id", "1")

	w := httptest.NewRecorder()

	handler.PatchItemValor(w, req)

	// Debe retornar 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("PatchItemValor(-500) status = %d, want 400", w.Code)
	}

	var errResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err == nil {
		if errorCode, ok := errResp["error"].(string); ok {
			if errorCode != "valor_invalido" {
				t.Errorf("PatchItemValor(-500) error code = %s, want valor_invalido", errorCode)
			}
		} else {
			t.Errorf("PatchItemValor(-500) error code missing or wrong type")
		}
	}
}

