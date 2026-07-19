CREATE TABLE stock_movimientos (
  id                BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  tienda_id         BIGINT UNSIGNED  NOT NULL,
  item_id           BIGINT UNSIGNED  NOT NULL,
  tipo_movimiento   ENUM(
                      'compra',
                      'merma',
                      'venta_batch',
                      'ajuste_conteo'
                    )                NOT NULL
                      COMMENT 'Tipo de movimiento de stock',
  cantidad_antes    DECIMAL(12,4)    NOT NULL
                      COMMENT 'Stock antes del movimiento',
  cantidad_despues  DECIMAL(12,4)    NOT NULL
                      COMMENT 'Stock después del movimiento',
  cantidad_delta    DECIMAL(12,4)    NOT NULL
                      COMMENT 'Cambio neto (cantidad_despues - cantidad_antes, puede ser negativo)',
  referencia_id     BIGINT UNSIGNED  NULL
                      COMMENT 'ID de tabla origen (compras.id, mermas.id, etc.)',
  referencia_tipo   VARCHAR(50)      NULL
                      COMMENT 'Tipo de referencia (compra, merma, venta_batch, conteo_ajuste)',
  usuario_id        BIGINT UNSIGNED  NOT NULL,
  motivo            TEXT             NULL
                      COMMENT 'Razón del movimiento (especialmente para ajustes)',
  creado_en         DATETIME         NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT fk_stock_movimientos_tienda
    FOREIGN KEY (tienda_id) REFERENCES tiendas (id),
  CONSTRAINT fk_stock_movimientos_item
    FOREIGN KEY (item_id) REFERENCES items (id),
  CONSTRAINT fk_stock_movimientos_usuario
    FOREIGN KEY (usuario_id) REFERENCES empleados (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_stock_movimientos_tienda_creado ON stock_movimientos (tienda_id, creado_en DESC);
CREATE INDEX ix_stock_movimientos_referencia ON stock_movimientos (referencia_id, referencia_tipo);
CREATE INDEX ix_stock_movimientos_item ON stock_movimientos (item_id, creado_en DESC);
