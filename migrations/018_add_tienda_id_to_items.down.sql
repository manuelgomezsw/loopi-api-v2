-- Rollback Migration 018: Remove tienda_id from items table

-- Restaurar constraints globales
ALTER TABLE items DROP CONSTRAINT uq_items_tienda_codigo;
ALTER TABLE items DROP CONSTRAINT uq_items_tienda_nombre;

ALTER TABLE items ADD CONSTRAINT uq_items_codigo
  UNIQUE (codigo);
ALTER TABLE items ADD CONSTRAINT uq_items_nombre
  UNIQUE (nombre);

-- Eliminar FK
ALTER TABLE items DROP CONSTRAINT fk_items_tienda;

-- Eliminar índices
DROP INDEX ix_items_tienda_frecuencia ON items;
DROP INDEX ix_items_tienda_activo ON items;

-- Eliminar columna
ALTER TABLE items DROP COLUMN tienda_id;
