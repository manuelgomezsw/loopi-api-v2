CREATE TABLE items_costos_tienda (
  id             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  item_id        BIGINT UNSIGNED  NOT NULL
                   COMMENT 'FK a items.id.',
  tienda_id      BIGINT UNSIGNED  NOT NULL
                   COMMENT 'FK a tiendas.id.',
  costo_unitario INT              NOT NULL
                   COMMENT 'Costo en COP sin decimales para esta tienda en este momento.',
  vigente_desde  DATETIME         NOT NULL
                   COMMENT 'Momento desde el que aplica este costo. Se asigna con NOW() al insertar.',
  creado_por     BIGINT UNSIGNED  NOT NULL
                   COMMENT 'FK a empleados.id — admin que registró este costo.',
  creado_en      DATETIME         NOT NULL,

  PRIMARY KEY (id),

  CONSTRAINT fk_ict_item
    FOREIGN KEY (item_id) REFERENCES items (id),
  CONSTRAINT fk_ict_tienda
    FOREIGN KEY (tienda_id) REFERENCES tiendas (id),
  CONSTRAINT fk_ict_creado_por
    FOREIGN KEY (creado_por) REFERENCES empleados (id)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_ict_item_tienda_vigente
  ON items_costos_tienda (item_id, tienda_id, vigente_desde);
