package inventarios

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockService struct {
	inventarios map[int64]*Inventario
}

func NewMockService() *MockService {
	return &MockService{
		inventarios: make(map[int64]*Inventario),
	}
}

func (m *MockService) Iniciar(ctx any, req *CreateInventarioReq, userID, roleID int64) (*InventarioResp, error) {
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

func (m *MockService) RegistrarValor(ctx any, inventarioID, itemID int64, valorReal float64, userID int64) (*ItemDetailResp, error) {
	return &ItemDetailResp{
		ID:       1,
		ItemID:   itemID,
		ValorReal: &valorReal,
	}, nil
}

func (m *MockService) Confirmar(ctx any, inventarioID int64, userID int64) (*InventarioResp, error) {
	inv := &InventarioResp{ID: inventarioID, Estado: EstadoCompletado}
	return inv, nil
}

func (m *MockService) Listar(ctx any, filtros *FiltrosInventario, userID int64, roleID int64) (*HistorialResp, error) {
	return &HistorialResp{Inventarios: []InventarioResp{}, Total: 0}, nil
}

func (m *MockService) Buscar(ctx any, inventarioID int64, userID int64, roleID int64) (*InventarioResp, error) {
	return &InventarioResp{ID: inventarioID}, nil
}

func (m *MockService) Modificar(ctx any, inventarioID, itemID int64, valorReal float64, userID, roleID int64) (*ItemDetailResp, error) {
	return &ItemDetailResp{}, nil
}

func (m *MockService) Eliminar(ctx any, inventarioID int64, userID, roleID int64) error {
	return nil
}

func (m *MockService) Sugerir(ctx any) (*SugerenciaResp, error) {
	return &SugerenciaResp{Tipo: TipoDiario, Horario: HorarioApertura}, nil
}

func (m *MockService) ValidarTipo(tipo Tipo) error {
	return nil
}

func (m *MockService) ValidarHorario(horario *Horario, tipo Tipo) error {
	return nil
}

// Tests
func TestGetSugerencia(t *testing.T) {
	handler := NewHandler(NewMockService())
	req := httptest.NewRequest("GET", "/api/v1/inventarios/sugerencia", nil)
	w := httptest.NewRecorder()

	handler.GetSugerencia(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetSugerencia() status = %d, want 200", w.Code)
	}

	var resp SugerenciaResp
	json.NewDecoder(w.Body).Decode(&resp)

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
	w := httptest.NewRecorder()

	handler.PostInventario(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("PostInventario() status = %d, want 201", w.Code)
	}
}

func TestGetHistorial(t *testing.T) {
	handler := NewHandler(NewMockService())
	req := httptest.NewRequest("GET", "/api/v1/inventarios", nil)
	w := httptest.NewRecorder()

	handler.GetHistorial(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetHistorial() status = %d, want 200", w.Code)
	}
}
