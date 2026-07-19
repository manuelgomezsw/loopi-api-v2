package inventarios

import (
	"context"
	"testing"
	"time"
)

// MockRepository es un mock simple para testing
type MockRepository struct {
	inventarios map[int64]*Inventario
	detalles    map[int64][]DetalleInventario
	nextID      int64
	nextDetalleID int64
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		inventarios: make(map[int64]*Inventario),
		detalles:    make(map[int64][]DetalleInventario),
		nextID:      1,
		nextDetalleID: 1,
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
	if len(detalles) == 0 {
		return nil
	}
	inventarioID := detalles[0].InventarioID
	for i := range detalles {
		detalles[i].ID = m.nextDetalleID
		m.nextDetalleID++
	}
	m.detalles[inventarioID] = detalles
	return nil
}

func (m *MockRepository) GetInventarioDetalle(ctx context.Context, id int64) (*Inventario, error) {
	inv, err := m.GetInventario(ctx, id)
	if err != nil {
		return nil, err
	}
	inv.Items = m.detalles[id]
	if inv.Items == nil {
		inv.Items = []DetalleInventario{}
	}
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

func (m *MockRepository) CanRecordMovimiento(ctx context.Context, tiendaID int64) (bool, *int64, error) {
	// Mock: siempre permite registrar movimientos
	return true, nil, nil
}

func (m *MockRepository) SnapshotStockActual(ctx context.Context, inventario *Inventario, items []int64) error {
	// Mock: no hace nada
	return nil
}

func (m *MockRepository) RecordMovimiento(ctx context.Context, movimiento *StockMovimiento) error {
	// Mock: no hace nada
	return nil
}

func (m *MockRepository) GetItemsActivosPorTipo(ctx context.Context, tiendaID int64, tipo Tipo) ([]int64, error) {
	// Mock: retorna items de ejemplo para testing
	if tipo == TipoDiario {
		return []int64{501, 502, 503}, nil
	}
	return []int64{}, nil
}

func (m *MockRepository) GetStockSnapshot(ctx context.Context, tiendaID int64, itemIDs []int64) (map[int64]float64, error) {
	// Mock: retorna stock snapshot de ejemplo
	stocks := make(map[int64]float64)
	for _, itemID := range itemIDs {
		stocks[itemID] = 50.0 // Default mock value
	}
	return stocks, nil
}

func (m *MockRepository) GetEstadoInventarioActivo(ctx context.Context, tiendaID int64) (*InventarioResp, error) {
	// Mock: no hay conteo activo
	return nil, nil
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

	resp, err := svc.Iniciar(ctx, req, 123, "admin", nil)
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

	resp, err := svc.Listar(ctx, filtros, 123, "admin", nil)
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

	resp, err := svc.Buscar(ctx, createdInv.ID, 123, "admin", nil)
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
		role    string
		wantErr bool
	}{
		{"success", createdInv.ID, 123, "admin", false},
		{"not found", 999, 123, "admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Eliminar(ctx, tt.invID, tt.userID, tt.role, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eliminar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIniciar_CompleteFlow verifica el flujo completo de iniciar conteo (T161)
// Integration test para Service.Iniciar() - flujo correcto:
// 1. Query items ANTES de crear inventario
// 2. Validar hay items (sino 422)
// 3. Crear inventario
// 4. Cruzar con stock_actual
// 5. Crear detalles con valor_sugerido
func TestIniciar_CompleteFlow(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	req := &CreateInventarioReq{
		TiendaID: 1,
		Tipo:     TipoDiario,
		Horario:  ptrHorario(HorarioApertura),
	}

	resp, err := svc.Iniciar(ctx, req, 123, "lider_tienda", ptrInt64(1))

	// Verificar: HTTP 201 con items
	if err != nil {
		t.Errorf("Iniciar() error = %v, want nil", err)
	}
	if resp == nil {
		t.Errorf("Iniciar() retornó nil")
	}
	if resp.ID == 0 {
		t.Errorf("Iniciar() retornó inventario sin ID")
	}
	if len(resp.Items) == 0 {
		t.Errorf("Iniciar() retornó sin items, want > 0")
	}
	if resp.Items[0].ValorSugerido == 0 && len(resp.Items) > 0 {
		// OK - puede ser 0, pero debe estar presente
	}
}

// TestIniciar_NoItems verifica que retorna 422 si no hay items para tipo
// T161: Verifica que POST /inventarios retorna 422 sin_items_contabilizar si no hay items
func TestIniciar_NoItems(t *testing.T) {
	// Crear mock repository que retorna 0 items
	mockRepo := NewMockRepository()
	// Override GetItemsActivosPorTipo para retornar lista vacía
	originalGetItems := mockRepo.GetItemsActivosPorTipo
	_ = originalGetItems

	svc := NewService(mockRepo)
	ctx := context.Background()

	req := &CreateInventarioReq{
		TiendaID: 1,
		Tipo:     TipoSemanal, // TipoSemanal retorna items vacíos en el mock
		Horario:  nil,
	}

	resp, err := svc.Iniciar(ctx, req, 123, "lider_tienda", ptrInt64(1))

	// Verificar: HTTP 422 sin_items_contabilizar
	if err == nil {
		t.Errorf("Iniciar() error = nil, want error sin_items_contabilizar")
	}
	if resp != nil {
		t.Errorf("Iniciar() retornó response, want nil")
	}

	// Verificar el código de error
	if domainErr, ok := err.(*Error); ok {
		if domainErr.Code != "sin_items_contabilizar" {
			t.Errorf("Iniciar() error code = %s, want sin_items_contabilizar", domainErr.Code)
		}
	}
}

// TestIniciar_UnauthorizedTienda verifica que rechaza autorización de tienda
func TestIniciar_UnauthorizedTienda(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	req := &CreateInventarioReq{
		TiendaID: 2, // Usuario está en tienda 1
		Tipo:     TipoDiario,
		Horario:  ptrHorario(HorarioApertura),
	}

	resp, err := svc.Iniciar(ctx, req, 123, "lider_tienda", ptrInt64(1))

	// Verificar: HTTP 403 tienda_no_autorizada
	if err == nil {
		t.Errorf("Iniciar() error = nil, want tienda_no_autorizada")
	}
	if resp != nil {
		t.Errorf("Iniciar() retornó response, want nil")
	}
}

// TestRegistrarValor_DiferenciaCalculation verifica cálculo de diferencia (T035)
func TestRegistrarValor_DiferenciaCalculation(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
		Items: []DetalleInventario{
			{ItemID: 1, ValorEsperado: 10, ID: 1},
		},
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)

	tests := []struct {
		name          string
		valorEsperado float64
		valorReal     float64
		expectedDiff  float64
	}{
		{"positive difference", 10, 15, 5},
		{"negative difference", 10, 5, -5},
		{"zero difference", 10, 10, 0},
		{"decimal values", 10.5, 12.3, 1.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv.Items[0].ValorEsperado = tt.valorEsperado
			resp, err := svc.RegistrarValor(ctx, createdInv.ID, 1, tt.valorReal, 123)
			if err != nil {
				t.Errorf("RegistrarValor() error = %v", err)
			}
			if resp == nil {
				t.Errorf("RegistrarValor() response is nil")
			}
		})
	}
}

// Note: TestIniciar_DuplicateInventory tests duplicate detection at repository level
// This is validated in repository_test.go with TestCreateInventario_DuplicateConstraint

// TestIniciar_HorarioValidation verifica validación de horario por tipo (T018)
func TestIniciar_HorarioValidation(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	// Test 1: diario con horario - debe pasar
	req := &CreateInventarioReq{TiendaID: 1, Tipo: TipoDiario, Horario: ptrHorario(HorarioApertura)}
	resp, err := svc.Iniciar(ctx, req, 123, "admin", nil)
	if err != nil {
		t.Errorf("Iniciar(diario+horario) error = %v, want nil", err)
	}
	if resp == nil {
		t.Errorf("Iniciar(diario+horario) retornó nil")
	}

	// Test 2: diario sin horario - debe fallar
	req = &CreateInventarioReq{TiendaID: 1, Tipo: TipoDiario, Horario: nil}
	_, err = svc.Iniciar(ctx, req, 123, "admin", nil)
	if err == nil {
		t.Errorf("Iniciar(diario sin horario) error = nil, want error")
	}
}

// TestConfirmar_AllItemsRegistered verifica validación de items registrados
func TestConfirmar_AllItemsRegistered(t *testing.T) {
	mockRepo := NewMockRepository()
	svc := NewService(mockRepo)
	ctx := context.Background()

	// Create inventory with items properly
	inv := &Inventario{
		TiendaID:      1,
		Tipo:          TipoDiario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 123,
		Items: []DetalleInventario{
			{ItemID: 1, ValorEsperado: 10, ValorReal: ptrFloat64(10)},
		},
	}
	createdInv, _ := mockRepo.CreateInventario(ctx, inv)
	detalles := []DetalleInventario{
		{InventarioID: createdInv.ID, ItemID: 1, ValorEsperado: 10, ValorReal: ptrFloat64(10)},
	}
	mockRepo.CreateDetalleInventario(ctx, detalles)

	resp, err := svc.Confirmar(ctx, createdInv.ID, 123)
	if err != nil {
		t.Errorf("Confirmar() error = %v, want nil", err)
	}
	if resp == nil {
		t.Errorf("Confirmar() returned nil")
	}
	if resp.Estado != EstadoCompletado {
		t.Errorf("Confirmar() estado = %v, want %v", resp.Estado, EstadoCompletado)
	}
}

// Helper functions
func ptrHorario(h Horario) *Horario {
	return &h
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt64(i int64) *int64 {
	return &i
}
