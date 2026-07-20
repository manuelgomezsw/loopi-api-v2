-- Migration 018: Add tienda_id to items table (multi-tenant per P-II)
-- Items deben ser por tienda. Esta migración agrega soporte multi-tenant.
-- Workaround para avanzar feature 009 mientras 007 se ajusta.

ALTER TABLE items ADD COLUMN tienda_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER id;

-- Agregar FK a tiendas
ALTER TABLE items ADD CONSTRAINT fk_items_tienda
  FOREIGN KEY (tienda_id) REFERENCES tiendas (id) ON DELETE RESTRICT;

-- Crear índices para búsquedas por tienda
CREATE INDEX ix_items_tienda_activo ON items (tienda_id, activo);
CREATE INDEX ix_items_tienda_frecuencia ON items (tienda_id, frecuencia_inventario, activo);

-- Eliminar constraints de unicidad a nivel global (ahora son por tienda)
ALTER TABLE items DROP CONSTRAINT uq_items_codigo;
ALTER TABLE items DROP CONSTRAINT uq_items_nombre;

-- Recrear constraints como compound (tienda_id + codigo/nombre)
ALTER TABLE items ADD CONSTRAINT uq_items_tienda_codigo
  UNIQUE (tienda_id, codigo);
ALTER TABLE items ADD CONSTRAINT uq_items_tienda_nombre
  UNIQUE (tienda_id, nombre);
