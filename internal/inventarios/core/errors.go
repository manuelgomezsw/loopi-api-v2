package core

import "errors"

var (
	ErrInventarioEnProgreso = errors.New("hay un conteo en progreso en esta tienda")
	ErrInventarioNoEncontrado = errors.New("inventario no encontrado")
	ErrDuplicado = errors.New("conteo duplicado para esta tienda/tipo/horario/fecha")
	ErrSinItems = errors.New("no hay items a contabilizar para este tipo")
	ErrValorInvalido = errors.New("valor debe ser mayor o igual a 0")
	ErrPermisoDenegado = errors.New("permiso denegado")
)
