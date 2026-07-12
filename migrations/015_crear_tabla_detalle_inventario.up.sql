CREATE TABLE detalle_inventario (
  id                        BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  inventario_id             BIGINT UNSIGNED  NOT NULL,
  item_id                   BIGINT UNSIGNED  NOT NULL,
  inventario_referencia_id  BIGINT UNSIGNED  NULL,
  valor_sugerido            DECIMAL(12,4)    NOT NULL,
  valor_esperado            DECIMAL(12,4)    NOT NULL,
  valor_real                DECIMAL(12,4)    NULL,
  diferencia                DECIMAL(12,4)    NULL,
  creado_en                 DATETIME         NOT NULL,
  actualizado_en            DATETIME         NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_detalle_inventario_inventario_item UNIQUE (inventario_id, item_id),
  CONSTRAINT fk_detalle_inventario_inventario FOREIGN KEY (inventario_id) REFERENCES inventarios (id),
  CONSTRAINT fk_detalle_inventario_item FOREIGN KEY (item_id) REFERENCES items (id),
  CONSTRAINT fk_detalle_inventario_referencia FOREIGN KEY (inventario_referencia_id) REFERENCES inventarios (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_detalle_inventario_item ON detalle_inventario (item_id);
