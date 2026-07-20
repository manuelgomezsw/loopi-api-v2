package realizar

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestUpdateDetalle_GuardaValorReal valida que UPDATE guarda valor_real correctamente
func TestUpdateDetalle_GuardaValorReal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	repo := &RepositoryImpl{db: db}

	valorReal := 18.0
	now := time.Now()

	// Mock de RETURNING (si la BD lo soporta)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE detalle_inventario`)).
		WithArgs(valorReal, 1, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "inventario_id", "item_id", "valor_esperado", "valor_real", "diferencia", "actualizado_en"}).
			AddRow(1, 1, 100, 20.0, valorReal, -2.0, now))

	detalle, err := repo.UpdateDetalle(context.Background(), 1, 100, valorReal)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if detalle == nil || *detalle.ValorReal != valorReal {
		t.Errorf("expected valor_real %f, got %v", valorReal, detalle)
	}
}

// TestUpdateDetalle_CalculaDiferencia valida que cálculo de diferencia es correcto
func TestUpdateDetalle_CalculaDiferencia(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	repo := &RepositoryImpl{db: db}

	valorReal := 18.0
	valorEsperado := 20.0
	diferencia := valorReal - valorEsperado // -2.0
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE detalle_inventario`)).
		WithArgs(valorReal, 1, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "inventario_id", "item_id", "valor_esperado", "valor_real", "diferencia", "actualizado_en"}).
			AddRow(1, 1, 100, valorEsperado, valorReal, diferencia, now))

	detalle, err := repo.UpdateDetalle(context.Background(), 1, 100, valorReal)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if detalle == nil || *detalle.Diferencia != diferencia {
		t.Errorf("expected diferencia %f, got %v", diferencia, detalle)
	}
}

// TestUpdateDetalle_ErrorSiItemNoExiste valida error cuando fila no existe
func TestUpdateDetalle_ErrorSiItemNoExiste(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	repo := &RepositoryImpl{db: db}

	valorReal := 18.0

	// Mock: no hay filas retornadas (ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE detalle_inventario`)).
		WithArgs(valorReal, 1, 999).
		WillReturnError(sql.ErrNoRows)

	detalle, err := repo.UpdateDetalle(context.Background(), 1, 999, valorReal)

	if err != sql.ErrNoRows && err != ErrItemNoEncontrado {
		t.Errorf("expected ErrItemNoEncontrado or sql.ErrNoRows, got %v", err)
	}
	if detalle != nil {
		t.Errorf("expected nil detalle, got %v", detalle)
	}
}
