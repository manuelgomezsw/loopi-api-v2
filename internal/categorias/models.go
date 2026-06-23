package categorias

import "time"

// Categoria es la entidad de dominio que mapea la tabla `categorias`.
type Categoria struct {
	ID             uint64    `json:"id"`
	Nombre         string    `json:"nombre"`
	Activo         bool      `json:"activo"`
	CreadoPor      uint64    `json:"creado_por"`
	CreadoEn       time.Time `json:"creado_en"`
	ActualizadoPor uint64    `json:"actualizado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

// Subcategoria es la entidad de dominio que mapea la tabla `subcategorias`.
type Subcategoria struct {
	ID             uint64    `json:"id"`
	Nombre         string    `json:"nombre"`
	CategoriaID    uint64    `json:"categoria_id"`
	Activo         bool      `json:"activo"`
	CreadoPor      uint64    `json:"creado_por"`
	CreadoEn       time.Time `json:"creado_en"`
	ActualizadoPor uint64    `json:"actualizado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

// SubcategoriaResponse extiende Subcategoria con el conteo de items asignados.
type SubcategoriaResponse struct {
	Subcategoria
	TotalItems int `json:"total_items"`
}

// CategoriaResponse extiende Categoria con sus subcategorías anidadas.
type CategoriaResponse struct {
	Categoria
	Subcategorias []SubcategoriaResponse `json:"subcategorias"`
}

// CatalogoResponse es la respuesta del listado completo de categorías.
type CatalogoResponse struct {
	Categorias []CategoriaResponse `json:"categorias"`
	Total      int                 `json:"total"`
}

// ImpactoResponse informa cuántas subcategorías activas tiene una categoría.
type ImpactoResponse struct {
	SubcategoriasActivas int `json:"subcategorias_activas"`
}

// InactivarCategoriaResponse confirma la inactivación en cascade.
type InactivarCategoriaResponse struct {
	ID                      uint64    `json:"id"`
	Nombre                  string    `json:"nombre"`
	Activo                  bool      `json:"activo"`
	SubcategoriasInactivadas int      `json:"subcategorias_inactivadas"`
	ActualizadoPor          uint64    `json:"actualizado_por"`
	ActualizadoEn           time.Time `json:"actualizado_en"`
}

// InactivarSubcategoriaResponse confirma la inactivación de una subcategoría.
type InactivarSubcategoriaResponse struct {
	ID             uint64    `json:"id"`
	Nombre         string    `json:"nombre"`
	CategoriaID    uint64    `json:"categoria_id"`
	Activo         bool      `json:"activo"`
	ActualizadoPor uint64    `json:"actualizado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

// errorResponse es el esquema estándar de errores de la API.
type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
}
