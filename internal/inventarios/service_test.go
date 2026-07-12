package inventarios

import (
	"context"
	"testing"
	"time"
)

// MockRepository es un mock simple para testing
type MockRepository struct {
	inventarios map[int64]*Inventario
	nextID      int64
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		inventarios: make(map[int64]*Inventario),
		nextID:      1,
	}
}

func (m *MockRepository) CreateInventario(ctx context.Context, inv *Inventario) (*Inventario, error) {
	inv.ID = m.nextID
	m.inventarios[m.nextID] = inv
	m.nextID++
	return inv, nil
}

func (m *MockRepository) GetInventario(ctx context.Context, id int64) (*Inventario, error) {
	inv, ok := m.inventarios[id]
	if !ok {
		return nil, NewError("not_found", "inventario no encontrado")
	}
	return inv, nil
}

func (m *MockRepository) CreateDetalleInventario(ctx context.Context, detalles []DetalleInventario) error {
	return nil
}

func (m *MockRepository) GetInventarioDetalle(ctx context.Context, id int64) (*Inventario, error) {
	inv, err := m.GetInventario(ctx, id)
	if err != nil {
		return nil, err
	}
	inv.Items = []DetalleInventario{}
	return inv, nil
}

func (m *MockRepository) ListInventarios(ctx context.Context, filtros *FiltrosInventario) ([]*Inventario, int64, error) {
	return []*Inventario{}, 0, nil
}

func (m *MockRepository) UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error) {
	return &DetalleInventario{}, nil
}

func (m *MockRepository) UpdateDetalleCompletado(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error) {
	return &DetalleInventario{}, nil
}

func (m *MockRepository) ConfirmarInventario(ctx context.Context, id int64) (*Inventario, error) {
	inv, err := m.GetInventario(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	inv.Estado = EstadoCompletado
	inv.CompletadoEn = &now
	return inv, nil
}

func (m *MockRepository) DeleteInventario(ctx context.Context, id int64) error {
	delete(m.inventarios, id)
	return nil
}

func (m *MockRepository) GetStockReferenciaByTipo(ctx context.Context, tiendaID int64, tipo Tipo, itemID int64) (*Inventario, float64, error) {
	return nil, 0, nil
}

func (m *MockRepository) SumarComprasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	return 0, nil
}

func (m *MockRepository) SumarVentasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	return 0, nil
}

func (m *MockRepository) SumarMermasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	return 0, nil
}

// Tests
func TestValidarTipo(t *testing.T) {
	svc := NewService(NewMockRepository())

	tests := []struct {
		name    string
		tipo    Tipo
		wantErr bool
	}{
		{"diario válido", TipoDiario, false},
		{"semanal válido", TipoSemanal, false},
		{"mensual válido", TipoMensual, false},
		{"inicial válido", TipoInicial, false},
		{"tipo inválido", Tipo("invalido"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidarTipo(tt.tipo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidarTipo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidarHorario(t *testing.T) {
	svc := NewService(NewMockRepository())

	tests := []struct {
		name    string
		horario *Horario
		tipo    Tipo
		wantErr bool
	}{
		{"diario con apertura", ptrHorario(HorarioApertura), TipoDiario, false},
		{"diario con mediodia", ptrHorario(HorarioMediodia), TipoDiario, false},
		{"diario sin horario", nil, TipoDiario, true},
		{"semanal sin horario", nil, TipoSemanal, false},
		{"semanal con horario", ptrHorario(HorarioApertura), TipoSemanal, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidarHorario(tt.horario, tt.tipo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidarHorario() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSugerir(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	sugerencia, err := svc.Sugerir(ctx)
	if err != nil {
		t.Fatalf("Sugerir() error = %v", err)
	}

	if sugerencia.Tipo == "" {
		t.Errorf("Sugerir() tipo vacío")
	}
}

func TestIniciar(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	req := &CreateInventarioReq{
		TiendaID: 1,
		Tipo:     TipoDiario,
		Horario:  ptrHorario(HorarioApertura),
	}

	resp, err := svc.Iniciar(ctx, req, 123, 1)
	if err != nil {
		t.Fatalf("Iniciar() error = %v", err)
	}

	if resp == nil {
		t.Errorf("Iniciar() retornó nil")
	}

	if resp.TiendaID != 1 {
		t.Errorf("Iniciar() TiendaID = %d, want 1", resp.TiendaID)
	}

	if resp.Estado != EstadoEnProgreso {
		t.Errorf("Iniciar() Estado = %v, want %v", resp.Estado, EstadoEnProgreso)
	}
}

func TestRegistrarValor(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	// Crear inventario base
	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
		Items: []DetalleInventario{
			{ItemID: 1, ValorEsperado: 10},
		},
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)

	tests := []struct {
		name      string
		invID     int64
		itemID    int64
		valorReal float64
		userID    int64
		wantErr   bool
	}{
		{"success", createdInv.ID, 1, 12.5, 123, false},
		{"not found", 999, 1, 12.5, 123, true},
		{"wrong responsable", createdInv.ID, 1, 12.5, 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.RegistrarValor(ctx, tt.invID, tt.itemID, tt.valorReal, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegistrarValor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && resp == nil {
				t.Errorf("RegistrarValor() retornó nil response")
			}
		})
	}
}

func TestConfirmar(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	// Crear inventario con item registrado
	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
		Items: []DetalleInventario{
			{ItemID: 1, ValorReal: ptrFloat64(10)},
		},
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)
	createdInv, _ = mockRepo.GetInventarioDetalle(ctx, createdInv.ID)

	tests := []struct {
		name    string
		invID   int64
		userID  int64
		wantErr bool
	}{
		{"success", createdInv.ID, 123, false},
		{"not found", 999, 123, true},
		{"wrong responsable", createdInv.ID, 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.Confirmar(ctx, tt.invID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Confirmar() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && resp.Estado != EstadoCompletado {
				t.Errorf("Confirmar() Estado = %v, want %v", resp.Estado, EstadoCompletado)
			}
		})
	}
}

func TestListar(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	filtros := &FiltrosInventario{
		Pagina:    1,
		PorPagina: 50,
	}

	resp, err := svc.Listar(ctx, filtros, 123, 1)
	if err != nil {
		t.Fatalf("Listar() error = %v", err)
	}

	if resp == nil {
		t.Errorf("Listar() retornó nil")
	}

	if resp.Total != 0 {
		t.Errorf("Listar() Total = %d, want 0", resp.Total)
	}
}

func TestBuscar(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)

	resp, err := svc.Buscar(ctx, createdInv.ID, 123, 1)
	if err != nil {
		t.Errorf("Buscar() error = %v", err)
	}

	if resp == nil {
		t.Errorf("Buscar() retornó nil")
	}

	if resp.ID != createdInv.ID {
		t.Errorf("Buscar() ID = %d, want %d", resp.ID, createdInv.ID)
	}
}

func TestModificar(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	// Crear inventario completado
	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoCompletado,
		ResponsableID: 123,
		Items: []DetalleInventario{
			{ItemID: 1, ValorEsperado: 10},
		},
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)

	resp, err := svc.Modificar(ctx, createdInv.ID, 1, 15.5, 123, 1)
	if err != nil {
		t.Errorf("Modificar() error = %v", err)
	}

	if resp == nil {
		t.Errorf("Modificar() retornó nil")
	}
}

func TestEliminar(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)

	tests := []struct {
		name    string
		invID   int64
		userID  int64
		roleID  int64
		wantErr bool
	}{
		{"success", createdInv.ID, 123, 1, false},
		{"not found", 999, 123, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Eliminar(ctx, tt.invID, tt.userID, tt.roleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eliminar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions
func ptrHorario(h Horario) *Horario {
	return &h
}

func ptrFloat64(f float64) *float64 {
	return &f
}
