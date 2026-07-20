package realizar

import (
	"context"
	"net/http"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
)

// mockRepository implementa Repository para tests
type mockRepository struct {
	updateDetalleFn        func(context.Context, int64, int64, float64) (*core.DetalleInventario, error)
	getItemsFn             func(context.Context, int64) ([]ItemDetalle, error)
	getInventarioFn        func(context.Context, int64) (*core.Inventario, error)
	getDetalleItemFn       func(context.Context, int64, int64) (*core.DetalleInventario, error)
	getItemsDetalleFn      func(context.Context, int64) ([]core.DetalleInventario, error)
}

func (m *mockRepository) UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*core.DetalleInventario, error) {
	if m.updateDetalleFn != nil {
		return m.updateDetalleFn(ctx, inventarioID, itemID, valorReal)
	}
	return nil, nil
}

func (m *mockRepository) GetItems(ctx context.Context, inventarioID int64) ([]ItemDetalle, error) {
	if m.getItemsFn != nil {
		return m.getItemsFn(ctx, inventarioID)
	}
	return nil, nil
}

func (m *mockRepository) GetInventario(ctx context.Context, inventarioID int64) (*core.Inventario, error) {
	if m.getInventarioFn != nil {
		return m.getInventarioFn(ctx, inventarioID)
	}
	return nil, nil
}

func (m *mockRepository) GetDetalleItem(ctx context.Context, inventarioID, itemID int64) (*core.DetalleInventario, error) {
	if m.getDetalleItemFn != nil {
		return m.getDetalleItemFn(ctx, inventarioID, itemID)
	}
	return nil, nil
}

func (m *mockRepository) GetItemsDetalle(ctx context.Context, inventarioID int64) ([]core.DetalleInventario, error) {
	if m.getItemsDetalleFn != nil {
		return m.getItemsDetalleFn(ctx, inventarioID)
	}
	return nil, nil
}

// TestRegistrarValor_Success valida flujo completo exitoso
func TestRegistrarValor_Success(t *testing.T) {
	mockRepo := &mockRepository{
		getInventarioFn: func(ctx context.Context, invID int64) (*core.Inventario, error) {
			return &core.Inventario{
				ID:     invID,
				Estado: core.EstadoEnProgreso,
			}, nil
		},
		getDetalleItemFn: func(ctx context.Context, invID, itemID int64) (*core.DetalleInventario, error) {
			return &core.DetalleInventario{
				ID:            1,
				InventarioID:  invID,
				ItemID:        itemID,
				ValorEsperado: 20.0,
			}, nil
		},
		updateDetalleFn: func(ctx context.Context, invID, itemID int64, valorReal float64) (*core.DetalleInventario, error) {
			return &core.DetalleInventario{
				ID:            1,
				InventarioID:  invID,
				ItemID:        itemID,
				ValorReal:     &valorReal,
				ValorEsperado: 20.0,
			}, nil
		},
	}

	svc := NewService(mockRepo)
	req := &RegistrarValorRequest{ValorReal: 18.0}

	resp, err := svc.RegistrarValor(&http.Request{}, 1, 100, req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp.ValorReal != 18.0 {
		t.Errorf("expected valor_real 18.0, got %f", resp.ValorReal)
	}
}

// TestRegistrarValor_ValorNegativo valida validación de valor >= 0
func TestRegistrarValor_ValorNegativo(t *testing.T) {
	mockRepo := &mockRepository{}
	svc := NewService(mockRepo)
	req := &RegistrarValorRequest{ValorReal: -5.0}

	resp, err := svc.RegistrarValor(&http.Request{}, 1, 100, req)

	if err == nil {
		t.Errorf("expected error for negative value, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

// TestRegistrarValor_CalculaDiferencia valida cálculo correcto de diferencia
func TestRegistrarValor_CalculaDiferencia(t *testing.T) {
	valorReal := 18.0
	mockRepo := &mockRepository{
		getInventarioFn: func(ctx context.Context, invID int64) (*core.Inventario, error) {
			return &core.Inventario{
				ID:     invID,
				Estado: core.EstadoEnProgreso,
			}, nil
		},
		getDetalleItemFn: func(ctx context.Context, invID, itemID int64) (*core.DetalleInventario, error) {
			return &core.DetalleInventario{
				ID:            1,
				InventarioID:  invID,
				ItemID:        itemID,
				ValorEsperado: 20.0,
			}, nil
		},
		updateDetalleFn: func(ctx context.Context, invID, itemID int64, vr float64) (*core.DetalleInventario, error) {
			diferencia := vr - 20.0 // valor_esperado = 20.0
			return &core.DetalleInventario{
				ID:            1,
				InventarioID:  invID,
				ItemID:        itemID,
				ValorReal:     &vr,
				ValorEsperado: 20.0,
				Diferencia:    &diferencia,
			}, nil
		},
	}

	svc := NewService(mockRepo)
	req := &RegistrarValorRequest{ValorReal: valorReal}

	resp, err := svc.RegistrarValor(&http.Request{}, 1, 100, req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedDiferencia := -2.0
	if resp.Diferencia != expectedDiferencia {
		t.Errorf("expected diferencia %f, got %f", expectedDiferencia, resp.Diferencia)
	}
}

// TestRegistrarValor_ConteoNoProgreso valida error cuando conteo no está en progreso
func TestRegistrarValor_ConteoNoProgreso(t *testing.T) {
	mockRepo := &mockRepository{
		getInventarioFn: func(ctx context.Context, invID int64) (*core.Inventario, error) {
			return &core.Inventario{
				ID:     invID,
				Estado: core.EstadoCompletado, // NO en progreso
			}, nil
		},
	}

	svc := NewService(mockRepo)
	req := &RegistrarValorRequest{ValorReal: 18.0}

	resp, err := svc.RegistrarValor(&http.Request{}, 1, 100, req)

	if err == nil {
		t.Errorf("expected error for non-in-progress conteo, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}
