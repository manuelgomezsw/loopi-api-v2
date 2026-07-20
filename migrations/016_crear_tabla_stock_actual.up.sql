CREATE TABLE stock_actual (
  id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  tienda_id       BIGINT UNSIGNED  NOT NULL,
  item_id         BIGINT UNSIGNED  NOT NULL,
  inventario_id   BIGINT UNSIGNED  NOT NULL,
  valor_snapshot  DECIMAL(12,4)    NOT NULL
                    COMMENT 'Snapshot de cantidad disponible al iniciar conteo (valor_sugerido)',
  tomado_en       DATETIME         NOT NULL
                    COMMENT 'Timestamp exacto cuando se tomó el snapshot',
  creado_en       DATETIME         NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_stock_actual_tienda_item_inventario
    UNIQUE (tienda_id, item_id, inventario_id),
  CONSTRAINT fk_stock_actual_tienda
    FOREIGN KEY (tienda_id) REFERENCES tiendas (id),
  CONSTRAINT fk_stock_actual_item
    FOREIGN KEY (item_id) REFERENCES items (id),
  CONSTRAINT fk_stock_actual_inventario
    FOREIGN KEY (inventario_id) REFERENCES inventarios (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_stock_actual_tienda_inventario ON stock_actual (tienda_id, inventario_id);
CREATE INDEX ix_stock_actual_inventario ON stock_actual (inventario_id);
