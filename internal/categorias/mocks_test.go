package categorias_test

import (
	"context"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	cat "github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
)

// --- Mock Repository ---

type mockRepo struct {
	insertarCategoriaFunc                func(nombre string, creadoPor uint64) (*cat.Categoria, error)
	obtenerCategoriaPorIDFunc            func(id uint64) (*cat.Categoria, error)
	listarConItemsFunc                   func(soloActivas *bool) (*cat.CatalogoResponse, error)
	actualizarCategoriaFunc              func(id uint64, nombre string, actualizadoPor uint64) (*cat.Categoria, error)
	contarSubcategoriasActivasFunc       func(categoriaID uint64) (int, error)
	inactivarCategoriaFunc               func(id, actualizadoPor uint64) error
	inactivarSubcategoriasDeCategFunc    func(categoriaID, actualizadoPor uint64) (int, error)
	reactivarCategoriaFunc               func(id, actualizadoPor uint64) (*cat.Categoria, error)
	insertarSubcategoriaFunc             func(nombre string, categoriaID, creadoPor uint64) (*cat.Subcategoria, error)
	obtenerSubcategoriaPorIDFunc         func(id uint64) (*cat.Subcategoria, error)
	actualizarSubcategoriaFunc           func(id uint64, nombre string, actualizadoPor uint64) (*cat.Subcategoria, error)
	inactivarSubcategoriaFunc            func(id, actualizadoPor uint64) error
	reactivarSubcategoriaFunc            func(id, actualizadoPor uint64) (*cat.Subcategoria, error)
	contarItemsPorSubcategoriaFunc       func(subcategoriaID uint64) (int, error)
}

func (m *mockRepo) InsertarCategoria(nombre string, creadoPor uint64) (*cat.Categoria, error) {
	return m.insertarCategoriaFunc(nombre, creadoPor)
}
func (m *mockRepo) ObtenerCategoriaPorID(id uint64) (*cat.Categoria, error) {
	return m.obtenerCategoriaPorIDFunc(id)
}
func (m *mockRepo) ListarConItems(soloActivas *bool) (*cat.CatalogoResponse, error) {
	return m.listarConItemsFunc(soloActivas)
}
func (m *mockRepo) ActualizarCategoria(id uint64, nombre string, actualizadoPor uint64) (*cat.Categoria, error) {
	return m.actualizarCategoriaFunc(id, nombre, actualizadoPor)
}
func (m *mockRepo) ContarSubcategoriasActivas(categoriaID uint64) (int, error) {
	return m.contarSubcategoriasActivasFunc(categoriaID)
}
func (m *mockRepo) InactivarCategoria(id, actualizadoPor uint64) error {
	return m.inactivarCategoriaFunc(id, actualizadoPor)
}
func (m *mockRepo) InactivarSubcategoriasDeCategoria(categoriaID, actualizadoPor uint64) (int, error) {
	return m.inactivarSubcategoriasDeCategFunc(categoriaID, actualizadoPor)
}
func (m *mockRepo) ReactivarCategoria(id, actualizadoPor uint64) (*cat.Categoria, error) {
	return m.reactivarCategoriaFunc(id, actualizadoPor)
}
func (m *mockRepo) InsertarSubcategoria(nombre string, categoriaID, creadoPor uint64) (*cat.Subcategoria, error) {
	return m.insertarSubcategoriaFunc(nombre, categoriaID, creadoPor)
}
func (m *mockRepo) ObtenerSubcategoriaPorID(id uint64) (*cat.Subcategoria, error) {
	return m.obtenerSubcategoriaPorIDFunc(id)
}
func (m *mockRepo) ActualizarSubcategoria(id uint64, nombre string, actualizadoPor uint64) (*cat.Subcategoria, error) {
	return m.actualizarSubcategoriaFunc(id, nombre, actualizadoPor)
}
func (m *mockRepo) InactivarSubcategoria(id, actualizadoPor uint64) error {
	return m.inactivarSubcategoriaFunc(id, actualizadoPor)
}
func (m *mockRepo) ReactivarSubcategoria(id, actualizadoPor uint64) (*cat.Subcategoria, error) {
	return m.reactivarSubcategoriaFunc(id, actualizadoPor)
}
func (m *mockRepo) ContarItemsPorSubcategoria(subcategoriaID uint64) (int, error) {
	return m.contarItemsPorSubcategoriaFunc(subcategoriaID)
}

// --- Mock Service ---

type mockSvc struct {
	obtenerCatalogoFunc         func(estado string) (*cat.CatalogoResponse, error)
	obtenerCategoriaFunc        func(id uint64) (*cat.CategoriaResponse, error)
	crearCategoriaFunc          func(nombre string, userID uint64, rol string) (*cat.CategoriaResponse, error)
	editarCategoriaFunc         func(id uint64, nombre string, userID uint64, rol string) (*cat.Categoria, error)
	obtenerImpactoCatFunc       func(id uint64) (*cat.ImpactoResponse, error)
	inactivarCategoriaFunc      func(id, userID uint64, rol string) (*cat.InactivarCategoriaResponse, error)
	reactivarCategoriaFunc      func(id, userID uint64, rol string) (*cat.Categoria, error)
	crearSubcategoriaFunc       func(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error)
	editarSubcategoriaFunc      func(id uint64, nombre string, userID uint64, rol string) (*cat.Subcategoria, error)
	inactivarSubcategoriaFunc   func(id, userID uint64, rol string) (*cat.InactivarSubcategoriaResponse, error)
	reactivarSubcategoriaFunc   func(id, userID uint64, rol string) (*cat.Subcategoria, error)
}

func (m *mockSvc) ObtenerCatalogo(estado string) (*cat.CatalogoResponse, error) {
	return m.obtenerCatalogoFunc(estado)
}
func (m *mockSvc) ObtenerCategoria(id uint64) (*cat.CategoriaResponse, error) {
	return m.obtenerCategoriaFunc(id)
}
func (m *mockSvc) CrearCategoria(nombre string, userID uint64, rol string) (*cat.CategoriaResponse, error) {
	return m.crearCategoriaFunc(nombre, userID, rol)
}
func (m *mockSvc) EditarCategoria(id uint64, nombre string, userID uint64, rol string) (*cat.Categoria, error) {
	return m.editarCategoriaFunc(id, nombre, userID, rol)
}
func (m *mockSvc) ObtenerImpactoCategoria(id uint64) (*cat.ImpactoResponse, error) {
	return m.obtenerImpactoCatFunc(id)
}
func (m *mockSvc) InactivarCategoria(id, userID uint64, rol string) (*cat.InactivarCategoriaResponse, error) {
	return m.inactivarCategoriaFunc(id, userID, rol)
}
func (m *mockSvc) ReactivarCategoria(id, userID uint64, rol string) (*cat.Categoria, error) {
	return m.reactivarCategoriaFunc(id, userID, rol)
}
func (m *mockSvc) CrearSubcategoria(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error) {
	return m.crearSubcategoriaFunc(nombre, categoriaID, userID, rol)
}
func (m *mockSvc) EditarSubcategoria(id uint64, nombre string, userID uint64, rol string) (*cat.Subcategoria, error) {
	return m.editarSubcategoriaFunc(id, nombre, userID, rol)
}
func (m *mockSvc) InactivarSubcategoria(id, userID uint64, rol string) (*cat.InactivarSubcategoriaResponse, error) {
	return m.inactivarSubcategoriaFunc(id, userID, rol)
}
func (m *mockSvc) ReactivarSubcategoria(id, userID uint64, rol string) (*cat.Subcategoria, error) {
	return m.reactivarSubcategoriaFunc(id, userID, rol)
}

// --- Helpers ---

func ctxAdmin() context.Context {
	claims := &auth.Claims{}
	claims.Subject = "42"
	claims.Rol = "admin"
	return auth.ContextWithClaims(context.Background(), claims)
}

func ctxRol(rol string) context.Context {
	claims := &auth.Claims{}
	claims.Subject = "99"
	claims.Rol = rol
	return auth.ContextWithClaims(context.Background(), claims)
}

func categoriaEjemplo() *cat.Categoria {
	return &cat.Categoria{ID: 1, Nombre: "Lácteo", Activo: true, CreadoPor: 42, ActualizadoPor: 42}
}

func categoriaInactiva() *cat.Categoria {
	c := categoriaEjemplo()
	c.Activo = false
	return c
}

func subcategoriaEjemplo() *cat.Subcategoria {
	return &cat.Subcategoria{ID: 1, Nombre: "Quesos", CategoriaID: 1, Activo: true, CreadoPor: 42, ActualizadoPor: 42}
}

func subcategoriaInactiva() *cat.Subcategoria {
	s := subcategoriaEjemplo()
	s.Activo = false
	return s
}
