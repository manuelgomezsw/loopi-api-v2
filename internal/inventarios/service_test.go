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

func (m *MockRepository) UpdateDetalleCompletado(ctx context.Context, id int64, valorReal float64) (*DetalleInventario, error) {
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

// Helper function
func ptrHorario(h Horario) *Horario {
	return &h
}
